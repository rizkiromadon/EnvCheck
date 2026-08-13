package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rizkiromadon/envcheck/internal/exitcode"
	"github.com/rizkiromadon/envcheck/internal/validator"
)

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return p
}

func TestRun_ValidFile(t *testing.T) {
	dir := t.TempDir()
	envPath := writeTemp(t, dir, ".env", "PORT=3000\nNODE_ENV=production\n")

	report, err := Run(Config{
		Version:       "test",
		TargetFile:    envPath,
		ValidatorOpts: validator.Options{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d: %+v", report.ErrorCount, report.Findings)
	}
	if report.VariableCount != 2 {
		t.Errorf("expected 2 variables, got %d", report.VariableCount)
	}
	if exitcode.Resolve(report, exitcode.Options{}) != exitcode.OK {
		t.Errorf("expected exit code OK")
	}
}

func TestRun_FileNotFound(t *testing.T) {
	_, err := Run(Config{
		Version:    "test",
		TargetFile: "/nonexistent/path/.env",
	})
	if err == nil {
		t.Fatalf("expected error for nonexistent file")
	}
}

func TestRun_SecurityDetection(t *testing.T) {
	dir := t.TempDir()
	envPath := writeTemp(t, dir, ".env", "DATABASE_PASSWORD=SuperSecret123\n")

	report, err := Run(Config{
		Version:    "test",
		TargetFile: envPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.WarningCount == 0 && report.CriticalCount == 0 {
		t.Errorf("expected a security finding, got %+v", report.Findings)
	}
	for _, f := range report.Findings {
		if f.Message == "" {
			continue
		}
		if containsSecret(f.Message, "SuperSecret123") {
			t.Errorf("finding message leaked the raw secret: %s", f.Message)
		}
	}
}

func containsSecret(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

func TestRun_SchemaValidation(t *testing.T) {
	dir := t.TempDir()
	envPath := writeTemp(t, dir, ".env", "PORT=abc\n")
	schemaPath := writeTemp(t, dir, ".env.schema", "PORT=number|required\n")

	report, err := Run(Config{
		Version:    "test",
		TargetFile: envPath,
		SchemaFile: schemaPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.ErrorCount == 0 {
		t.Errorf("expected schema type mismatch error, got %+v", report.Findings)
	}
}

func TestRun_CompareMode(t *testing.T) {
	dir := t.TempDir()
	envPath := writeTemp(t, dir, ".env", "FOO=1\n")
	examplePath := writeTemp(t, dir, ".env.example", "FOO=\nBAR=\n")

	report, err := Run(Config{
		Version:    "test",
		TargetFile: envPath,
		CompareTo:  examplePath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.CompareResult == nil {
		t.Fatalf("expected CompareResult to be populated")
	}
	if len(report.CompareResult.MissingRequired) != 1 || report.CompareResult.MissingRequired[0] != "BAR" {
		t.Errorf("expected BAR missing, got %v", report.CompareResult.MissingRequired)
	}
}

func TestRun_DuplicateKeyProducesValidationError(t *testing.T) {
	dir := t.TempDir()
	envPath := writeTemp(t, dir, ".env", "FOO=1\nFOO=2\n")

	report, err := Run(Config{Version: "test", TargetFile: envPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitcode.Resolve(report, exitcode.Options{}) != exitcode.ValidationError {
		t.Errorf("expected ValidationError exit code for duplicate key")
	}
}
