package compare

import (
	"testing"

	"github.com/rizkiromadon/envcheck/internal/parser"
)

func TestCompare_MissingAndExtra(t *testing.T) {
	env := parser.Parse("FOO=1\nBAZ=3\n")
	example := parser.Parse("FOO=\nBAR=\n")

	result := Compare(env.Entries, example.Entries, env.DuplicateLines, example.DuplicateLines)

	if len(result.MissingRequired) != 1 || result.MissingRequired[0] != "BAR" {
		t.Errorf("expected missing=[BAR], got %v", result.MissingRequired)
	}
	if len(result.ExtraInEnv) != 1 || result.ExtraInEnv[0] != "BAZ" {
		t.Errorf("expected extra=[BAZ], got %v", result.ExtraInEnv)
	}
	if len(result.InBoth) != 1 || result.InBoth[0] != "FOO" {
		t.Errorf("expected inBoth=[FOO], got %v", result.InBoth)
	}
}

func TestCompare_DuplicatesSurfaced(t *testing.T) {
	env := parser.Parse("FOO=1\nFOO=2\n")
	example := parser.Parse("FOO=\n")

	result := Compare(env.Entries, example.Entries, env.DuplicateLines, example.DuplicateLines)

	if len(result.DuplicatesInEnv) != 1 || result.DuplicatesInEnv[0] != "FOO" {
		t.Errorf("expected duplicatesInEnv=[FOO], got %v", result.DuplicatesInEnv)
	}
}

func TestCompare_IdenticalFilesNoDiff(t *testing.T) {
	env := parser.Parse("FOO=1\nBAR=2\n")
	example := parser.Parse("FOO=\nBAR=\n")

	result := Compare(env.Entries, example.Entries, env.DuplicateLines, example.DuplicateLines)

	if len(result.MissingRequired) != 0 {
		t.Errorf("expected no missing, got %v", result.MissingRequired)
	}
	if len(result.ExtraInEnv) != 0 {
		t.Errorf("expected no extra, got %v", result.ExtraInEnv)
	}
	if len(result.InBoth) != 2 {
		t.Errorf("expected 2 shared keys, got %v", result.InBoth)
	}
}
