// Package validator inspects a parsed .env file for structural problems:
// invalid syntax, duplicate keys, empty values, bad variable names,
// unclosed quotes, and unresolved variable references.
package validator

import (
	"fmt"
	"regexp"

	"github.com/rizkiromadon/envcheck/internal/model"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Options controls validator behavior.
type Options struct {
	AllowEmptyValues bool
}

// Validate runs all structural checks against a ParseResult and returns
// normalized Findings. It does not consider the schema or .env.example;
// those are handled by their own packages and merged by the caller.
func Validate(pr model.ParseResult, opts Options) []model.Finding {
	var findings []model.Finding

	for _, pe := range pr.Errors {
		findings = append(findings, model.Finding{
			Category: model.CategorySyntax,
			Severity: model.SeverityError,
			Line:     pe.Line,
			Message:  fmt.Sprintf("%s: %s", syntaxLabel(pe.Kind), pe.Message),
			RuleID:   string(pe.Kind),
		})
	}

	for key, lines := range pr.DuplicateLines {
		findings = append(findings, model.Finding{
			Category: model.CategoryDup,
			Severity: model.SeverityError,
			Key:      key,
			Line:     lines[len(lines)-1],
			Message:  fmt.Sprintf("duplicate key '%s' declared on lines %v", key, lines),
			RuleID:   "duplicate_key",
		})
	}

	definedKeys := make(map[string]bool, len(pr.Entries))
	for _, e := range pr.Entries {
		definedKeys[e.Key] = true
	}

	for _, e := range pr.Entries {
		if !keyPattern.MatchString(e.Key) {
			findings = append(findings, model.Finding{
				Category: model.CategoryNaming,
				Severity: model.SeverityError,
				Key:      e.Key,
				Line:     e.Line,
				Message:  fmt.Sprintf("invalid variable name '%s' (must match ^[A-Za-z_][A-Za-z0-9_]*$)", e.Key),
				RuleID:   "invalid_name",
			})
		}

		if !opts.AllowEmptyValues && e.Value == "" && e.Quote != model.SingleQuote {
			findings = append(findings, model.Finding{
				Category: model.CategoryEmpty,
				Severity: model.SeverityWarning,
				Key:      e.Key,
				Line:     e.Line,
				Message:  fmt.Sprintf("'%s' has an empty value", e.Key),
				RuleID:   "empty_value",
			})
		}

		if e.HasRefs {
			for _, ref := range e.Refs {
				if !definedKeys[ref] {
					findings = append(findings, model.Finding{
						Category: model.CategoryRef,
						Severity: model.SeverityWarning,
						Key:      e.Key,
						Line:     e.Line,
						Message:  fmt.Sprintf("'%s' references undefined variable '${%s}'", e.Key, ref),
						RuleID:   "undefined_reference",
					})
				}
			}
		}
	}

	return findings
}

// syntaxLabel returns a short human-readable label for a parse error kind.
func syntaxLabel(kind model.ParseErrorKind) string {
	switch kind {
	case model.ErrUnclosedQuote:
		return "unclosed quote"
	case model.ErrInvalidSyntax:
		return "invalid syntax"
	case model.ErrInvalidName:
		return "invalid variable name"
	case model.ErrMissingEquals:
		return "missing '='"
	case model.ErrUnterminatedMulti:
		return "unterminated multiline value"
	default:
		return "syntax error"
	}
}
