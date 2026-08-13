// Package exitcode centralizes EnvCheck's exit code policy so that CLI,
// CI/CD, and scripting consumers get a single, documented, stable contract.
package exitcode

import "github.com/rizkiromadon/envcheck/internal/model"

// The exit codes returned by Resolve.
const (
	// OK indicates no errors and no security findings.
	OK = 0
	// ValidationError indicates one or more validation errors.
	ValidationError = 1
	// SecurityIssue indicates one or more security findings with no
	// validation errors taking precedence.
	SecurityIssue = 2
	// UsageError indicates a bad CLI invocation.
	UsageError = 3
	// InternalError indicates an unexpected internal failure.
	InternalError = 4
)

// Options configures how Resolve maps a Report to a process exit code.
type Options struct {
	Strict bool
}

// Resolve applies EnvCheck's exit code precedence rules to report and
// returns the corresponding process exit code.
func Resolve(report model.Report, opts Options) int {
	hasValidationError := report.ErrorCount > 0
	hasCriticalSecurity := false
	hasAnySecurity := false

	for _, f := range report.Findings {
		if f.Category == model.CategorySecurity {
			hasAnySecurity = true
			if f.Severity == model.SeverityCritical {
				hasCriticalSecurity = true
			}
		}
	}

	if opts.Strict {
		if hasValidationError || report.WarningCount > 0 {
			return ValidationError
		}
		if hasAnySecurity {
			return SecurityIssue
		}
		return OK
	}

	if hasValidationError {
		return ValidationError
	}
	if hasCriticalSecurity {
		return SecurityIssue
	}
	return OK
}
