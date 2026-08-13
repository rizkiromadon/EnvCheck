package exitcode

import (
	"testing"

	"github.com/rizkiromadon/envcheck/internal/model"
)

func TestResolve_OK(t *testing.T) {
	report := model.Report{}
	if got := Resolve(report, Options{}); got != OK {
		t.Errorf("expected OK, got %d", got)
	}
}

func TestResolve_ValidationError(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{Category: model.CategorySyntax, Severity: model.SeverityError},
	}}
	report.Recompute()
	if got := Resolve(report, Options{}); got != ValidationError {
		t.Errorf("expected ValidationError, got %d", got)
	}
}

func TestResolve_CriticalSecurityIssue(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{Category: model.CategorySecurity, Severity: model.SeverityCritical},
	}}
	report.Recompute()
	if got := Resolve(report, Options{}); got != SecurityIssue {
		t.Errorf("expected SecurityIssue, got %d", got)
	}
}

func TestResolve_WarningOnlyIsOKWithoutStrict(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{Category: model.CategoryEmpty, Severity: model.SeverityWarning},
	}}
	report.Recompute()
	if got := Resolve(report, Options{}); got != OK {
		t.Errorf("expected OK for warnings without strict mode, got %d", got)
	}
}

func TestResolve_WarningFailsInStrictMode(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{Category: model.CategoryEmpty, Severity: model.SeverityWarning},
	}}
	report.Recompute()
	if got := Resolve(report, Options{Strict: true}); got != ValidationError {
		t.Errorf("expected ValidationError in strict mode with warnings, got %d", got)
	}
}

func TestResolve_ValidationTakesPrecedenceOverSecurity(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{Category: model.CategorySyntax, Severity: model.SeverityError},
		{Category: model.CategorySecurity, Severity: model.SeverityCritical},
	}}
	report.Recompute()
	if got := Resolve(report, Options{}); got != ValidationError {
		t.Errorf("expected ValidationError to take precedence, got %d", got)
	}
}

func TestResolve_WarningSecurityNotStrict(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{Category: model.CategorySecurity, Severity: model.SeverityWarning},
	}}
	report.Recompute()
	if got := Resolve(report, Options{}); got != OK {
		t.Errorf("expected OK for warning-level security without strict, got %d", got)
	}
}

func TestResolve_WarningSecurityStrict(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{Category: model.CategorySecurity, Severity: model.SeverityWarning},
	}}
	report.Recompute()
	if got := Resolve(report, Options{Strict: true}); got != ValidationError {
		t.Errorf("expected ValidationError in strict mode, got %d", got)
	}
}
