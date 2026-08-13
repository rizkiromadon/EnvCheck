package schema

import (
	"testing"

	"github.com/rizkiromadon/envcheck/internal/parser"
)

func TestSchemaParse_Basic(t *testing.T) {
	content := "PORT=number|required\nDATABASE_URL=url|required|secret\nNODE_ENV=enum:development,production,test|required\nDEBUG=boolean|optional\n"
	sc, errs := Parse(content)
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(sc.Rules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(sc.Rules))
	}
}

func TestSchemaValidate_RequiredMissing(t *testing.T) {
	sc, _ := Parse("PORT=number|required\n")
	pr := parser.Parse("OTHER=1\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for missing required var, got %d: %+v", len(findings), findings)
	}
	if findings[0].RuleID != "schema_missing_required" {
		t.Errorf("expected schema_missing_required, got %s", findings[0].RuleID)
	}
}

func TestSchemaValidate_NumberType(t *testing.T) {
	sc, _ := Parse("PORT=number|required\n")
	pr := parser.Parse("PORT=not_a_number\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 1 || findings[0].RuleID != "schema_type_mismatch" {
		t.Fatalf("expected schema_type_mismatch, got %+v", findings)
	}
}

func TestSchemaValidate_NumberTypeValid(t *testing.T) {
	sc, _ := Parse("PORT=number|required\n")
	pr := parser.Parse("PORT=3000\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %+v", findings)
	}
}

func TestSchemaValidate_PortRange(t *testing.T) {
	sc, _ := Parse("PORT=port|required\n")
	pr := parser.Parse("PORT=99999\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for out-of-range port, got %+v", findings)
	}
}

func TestSchemaValidate_BooleanType(t *testing.T) {
	sc, _ := Parse("DEBUG=boolean|optional\n")
	pr := parser.Parse("DEBUG=maybe\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for invalid boolean, got %+v", findings)
	}
}

func TestSchemaValidate_BooleanValidValues(t *testing.T) {
	for _, v := range []string{"true", "false", "1", "0", "yes", "no"} {
		sc, _ := Parse("DEBUG=boolean|optional\n")
		pr := parser.Parse("DEBUG=" + v + "\n")
		findings := sc.Validate(pr.Entries)
		if len(findings) != 0 {
			t.Errorf("value %q should be a valid boolean, got findings %+v", v, findings)
		}
	}
}

func TestSchemaValidate_EnumType(t *testing.T) {
	sc, _ := Parse("NODE_ENV=enum:development,production,test|required\n")
	pr := parser.Parse("NODE_ENV=staging\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for invalid enum value, got %+v", findings)
	}
}

func TestSchemaValidate_EnumTypeValid(t *testing.T) {
	sc, _ := Parse("NODE_ENV=enum:development,production,test|required\n")
	pr := parser.Parse("NODE_ENV=production\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %+v", findings)
	}
}

func TestSchemaValidate_URLType(t *testing.T) {
	sc, _ := Parse("DATABASE_URL=url|required\n")
	pr := parser.Parse("DATABASE_URL=not-a-url\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for invalid URL, got %+v", findings)
	}
}

func TestSchemaValidate_URLTypeValid(t *testing.T) {
	sc, _ := Parse("DATABASE_URL=url|required\n")
	pr := parser.Parse("DATABASE_URL=postgres://localhost:5432/db\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %+v", findings)
	}
}

func TestSchemaParse_UnknownType(t *testing.T) {
	_, errs := Parse("FOO=notatype|required\n")
	if len(errs) != 1 {
		t.Fatalf("expected 1 parse error for unknown type, got %d", len(errs))
	}
}

func TestSchemaValidate_OptionalMissingNoFinding(t *testing.T) {
	sc, _ := Parse("DEBUG=boolean|optional\n")
	pr := parser.Parse("OTHER=1\n")
	findings := sc.Validate(pr.Entries)
	if len(findings) != 0 {
		t.Errorf("expected no findings for missing optional var, got %+v", findings)
	}
}
