// Package formatter renders a model.Report as either human-readable
// terminal output or machine-readable JSON, with optional color and quiet
// modes suited to both interactive and CI/CD use.
package formatter

import (
	"fmt"
	"io"
	"sort"

	"github.com/rizkiromadon/envcheck/internal/model"
)

// HumanOptions controls the human-readable renderer.
type HumanOptions struct {
	NoColor bool
	Quiet   bool
	Version string
}

const (
	colReset  = "\x1b[0m"
	colRed    = "\x1b[31m"
	colYellow = "\x1b[33m"
	colGreen  = "\x1b[32m"
	colCyan   = "\x1b[36m"
	colBold   = "\x1b[1m"
	colDim    = "\x1b[2m"
)

// c wraps s in the given color code unless NoColor is set.
func (o HumanOptions) c(code, s string) string {
	if o.NoColor {
		return s
	}
	return code + s + colReset
}

// WriteHuman renders the report as readable terminal output.
func WriteHuman(w io.Writer, report model.Report, opts HumanOptions) {
	if opts.Version != "" {
		fmt.Fprintf(w, "%s\n\n", opts.c(colBold, "EnvCheck "+opts.Version))
	}

	fmt.Fprintf(w, "%s: %s\n\n", opts.c(colBold, "File"), report.File)

	byCategory := groupByCategory(report.Findings)

	printCategoryLine(w, opts, byCategory, model.CategorySyntax, "Syntax")
	printCategoryLine(w, opts, byCategory, model.CategoryNaming, "Variable names")
	printCategoryLine(w, opts, byCategory, model.CategoryDup, "Duplicate keys")
	printCategoryLine(w, opts, byCategory, model.CategoryEmpty, "Empty values")
	printCategoryLine(w, opts, byCategory, model.CategoryQuote, "Quotes")
	printCategoryLine(w, opts, byCategory, model.CategoryRef, "Variable references")
	printCategoryLine(w, opts, byCategory, model.CategorySchema, "Schema")
	printCategoryLine(w, opts, byCategory, model.CategorySecurity, "Security")

	if report.CompareResult != nil {
		printCompare(w, opts, *report.CompareResult)
	}

	if report.GitTracked != nil {
		fmt.Fprintln(w)
		if *report.GitTracked {
			fmt.Fprintf(w, "%s .env is tracked by Git — consider removing it and adding it to .gitignore\n",
				opts.c(colYellow, "⚠"))
		} else if !opts.Quiet {
			fmt.Fprintf(w, "%s .env is not tracked by Git\n", opts.c(colGreen, "✓"))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, opts.c(colBold, "Summary"))
	fmt.Fprintf(w, "%d variables\n", report.VariableCount)
	if report.ErrorCount > 0 {
		fmt.Fprintf(w, "%s\n", opts.c(colRed, fmt.Sprintf("%d error(s)", report.ErrorCount)))
	} else {
		fmt.Fprintln(w, "0 errors")
	}
	if report.CriticalCount > 0 {
		fmt.Fprintf(w, "%s\n", opts.c(colRed, fmt.Sprintf("%d critical security finding(s)", report.CriticalCount)))
	}
	if report.WarningCount > 0 {
		fmt.Fprintf(w, "%s\n", opts.c(colYellow, fmt.Sprintf("%d warning(s)", report.WarningCount)))
	} else {
		fmt.Fprintln(w, "0 warnings")
	}
	if report.InfoCount > 0 {
		fmt.Fprintf(w, "%d info\n", report.InfoCount)
	}
}

// groupByCategory groups findings by category, sorting each group by line number.
func groupByCategory(findings []model.Finding) map[model.FindingCategory][]model.Finding {
	m := make(map[model.FindingCategory][]model.Finding)
	for _, f := range findings {
		m[f.Category] = append(m[f.Category], f)
	}
	for k := range m {
		sort.Slice(m[k], func(i, j int) bool { return m[k][i].Line < m[k][j].Line })
	}
	return m
}

// printCategoryLine writes the findings for a single category, or a success
// line if the category has none.
func printCategoryLine(w io.Writer, opts HumanOptions, byCat map[model.FindingCategory][]model.Finding, cat model.FindingCategory, label string) {
	items := byCat[cat]
	if len(items) == 0 {
		if !opts.Quiet {
			fmt.Fprintf(w, "%s %s\n", opts.c(colGreen, "✓"), label)
		}
		return
	}
	for _, f := range items {
		symbol, color := symbolFor(f.Severity)
		loc := ""
		if f.Line > 0 {
			loc = fmt.Sprintf(" (line %d)", f.Line)
		}
		msg := f.Message
		if f.Masked != "" {
			msg = fmt.Sprintf("%s [%s]", msg, f.Masked)
		}
		fmt.Fprintf(w, "%s %s%s\n", opts.c(color, symbol), msg, loc)
	}
}

// symbolFor returns the display symbol and color code for a severity.
func symbolFor(sev model.Severity) (string, string) {
	switch sev {
	case model.SeverityError:
		return "✗", colRed
	case model.SeverityCritical:
		return "✗", colRed
	case model.SeverityWarning:
		return "⚠", colYellow
	default:
		return "ℹ", colCyan
	}
}

// printCompare writes the .env.example comparison section.
func printCompare(w io.Writer, opts HumanOptions, cr model.CompareResult) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, opts.c(colBold, "Comparison with .env.example"))

	if len(cr.MissingRequired) == 0 {
		fmt.Fprintf(w, "%s No missing variables\n", opts.c(colGreen, "✓"))
	} else {
		for _, k := range cr.MissingRequired {
			fmt.Fprintf(w, "%s Missing: %s\n", opts.c(colRed, "✗"), k)
		}
	}
	for _, k := range cr.ExtraInEnv {
		fmt.Fprintf(w, "%s Extra (not in example): %s\n", opts.c(colCyan, "ℹ"), k)
	}
	if !opts.Quiet {
		for _, k := range cr.InBoth {
			fmt.Fprintf(w, "%s In both: %s\n", opts.c(colGreen, "✓"), k)
		}
	}
	for _, k := range cr.DuplicatesInEnv {
		fmt.Fprintf(w, "%s Duplicate in .env: %s\n", opts.c(colRed, "✗"), k)
	}
	for _, k := range cr.DuplicatesInExample {
		fmt.Fprintf(w, "%s Duplicate in example: %s\n", opts.c(colYellow, "⚠"), k)
	}
}
