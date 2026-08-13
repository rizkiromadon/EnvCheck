// Package app orchestrates the full EnvCheck pipeline: read file -> parse
// -> validate -> security scan -> schema validate -> compare -> git check
// -> build unified Report. It has no CLI-specific or output-format
// concerns; those live in cmd/envcheck and internal/formatter.
package app

import (
	"fmt"
	"os"

	"github.com/rizkiromadon/envcheck/internal/compare"
	"github.com/rizkiromadon/envcheck/internal/gitcheck"
	"github.com/rizkiromadon/envcheck/internal/model"
	"github.com/rizkiromadon/envcheck/internal/parser"
	"github.com/rizkiromadon/envcheck/internal/schema"
	"github.com/rizkiromadon/envcheck/internal/security"
	"github.com/rizkiromadon/envcheck/internal/validator"
)

// Config describes one run of EnvCheck against a target .env file.
type Config struct {
	Version string

	TargetFile string
	CompareTo  string
	SchemaFile string

	CheckGit     bool
	SkipSecurity bool

	ValidatorOpts validator.Options
}

// Run executes the full pipeline and returns a populated Report. It
// returns an error only for unrecoverable I/O problems; content-level
// problems are always surfaced as Findings, never as Go errors.
func Run(cfg Config) (model.Report, error) {
	report := model.Report{
		ToolVersion: cfg.Version,
		File:        cfg.TargetFile,
	}

	raw, err := os.ReadFile(cfg.TargetFile)
	if err != nil {
		return report, fmt.Errorf("cannot read %s: %w", cfg.TargetFile, err)
	}

	pr := parser.Parse(string(raw))
	report.VariableCount = len(pr.Entries)

	report.Findings = append(report.Findings, validator.Validate(pr, cfg.ValidatorOpts)...)

	if !cfg.SkipSecurity {
		scanner := security.New()
		report.Findings = append(report.Findings, scanner.Scan(pr.Entries)...)
	}

	if cfg.SchemaFile != "" {
		schemaRaw, err := os.ReadFile(cfg.SchemaFile)
		if err != nil {
			return report, fmt.Errorf("cannot read schema file %s: %w", cfg.SchemaFile, err)
		}
		sc, schemaErrs := schema.Parse(string(schemaRaw))
		for _, se := range schemaErrs {
			report.Findings = append(report.Findings, model.Finding{
				Category: model.CategorySchema,
				Severity: model.SeverityError,
				Message:  se.Error(),
				RuleID:   "schema_parse_error",
			})
		}
		report.Findings = append(report.Findings, sc.Validate(pr.Entries)...)
	}

	if cfg.CompareTo != "" {
		exampleRaw, err := os.ReadFile(cfg.CompareTo)
		if err != nil {
			return report, fmt.Errorf("cannot read example file %s: %w", cfg.CompareTo, err)
		}
		examplePR := parser.Parse(string(exampleRaw))
		cr := compare.Compare(pr.Entries, examplePR.Entries, pr.DuplicateLines, examplePR.DuplicateLines)
		report.CompareResult = &cr
		report.Findings = append(report.Findings, compare.ToFindings(cr)...)
	}

	if cfg.CheckGit {
		status, err := gitcheck.Check(cfg.TargetFile)
		if err == nil && status.InRepo {
			tracked := status.Tracked
			report.GitTracked = &tracked
			if tracked {
				report.Findings = append(report.Findings, model.Finding{
					Category: model.CategoryGit,
					Severity: model.SeverityWarning,
					Message:  ".env is tracked by Git",
					RuleID:   "git_tracked",
				})
			}
		}
	}

	report.Recompute()
	return report, nil
}
