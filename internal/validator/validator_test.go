package validator

import (
	"testing"

	"github.com/rizkiromadon/envcheck/internal/model"
	"github.com/rizkiromadon/envcheck/internal/parser"
)

func hasFinding(findings []model.Finding, ruleID string) bool {
	for _, f := range findings {
		if f.RuleID == ruleID {
			return true
		}
	}
	return false
}

func TestValidate_DuplicateKey(t *testing.T) {
	pr := parser.Parse("FOO=1\nFOO=2\n")
	findings := Validate(pr, Options{})
	if !hasFinding(findings, "duplicate_key") {
		t.Errorf("expected duplicate_key finding")
	}
}

func TestValidate_EmptyValue(t *testing.T) {
	pr := parser.Parse("FOO=\n")
	findings := Validate(pr, Options{})
	if !hasFinding(findings, "empty_value") {
		t.Errorf("expected empty_value finding")
	}
}

func TestValidate_AllowEmptyOption(t *testing.T) {
	pr := parser.Parse("FOO=\n")
	findings := Validate(pr, Options{AllowEmptyValues: true})
	if hasFinding(findings, "empty_value") {
		t.Errorf("expected no empty_value finding when AllowEmptyValues=true")
	}
}

func TestValidate_SingleQuotedEmptyIsIntentional(t *testing.T) {
	pr := parser.Parse("FOO=''\n")
	findings := Validate(pr, Options{})
	if hasFinding(findings, "empty_value") {
		t.Errorf("single-quoted empty string should not be flagged as accidental empty value")
	}
}

func TestValidate_InvalidName(t *testing.T) {
	pr := parser.Parse("1FOO=bar\n")
	findings := Validate(pr, Options{})
	found := false
	for _, f := range findings {
		if f.Category == model.CategorySyntax && f.RuleID == string(model.ErrInvalidName) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected invalid name to surface as syntax finding")
	}
}

func TestValidate_UndefinedReference(t *testing.T) {
	pr := parser.Parse("URL=http://${MISSING_HOST}\n")
	findings := Validate(pr, Options{})
	if !hasFinding(findings, "undefined_reference") {
		t.Errorf("expected undefined_reference finding")
	}
}

func TestValidate_DefinedReferenceNoFinding(t *testing.T) {
	pr := parser.Parse("HOST=localhost\nURL=http://${HOST}\n")
	findings := Validate(pr, Options{})
	if hasFinding(findings, "undefined_reference") {
		t.Errorf("did not expect undefined_reference finding when variable is defined")
	}
}

func TestValidate_ValidFileNoFindings(t *testing.T) {
	pr := parser.Parse("PORT=3000\nNODE_ENV=production\n")
	findings := Validate(pr, Options{})
	if len(findings) != 0 {
		t.Errorf("expected no findings for valid file, got %+v", findings)
	}
}
