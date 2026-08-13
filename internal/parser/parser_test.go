package parser

import (
	"testing"

	"github.com/rizkiromadon/envcheck/internal/model"
)

func findEntry(t *testing.T, entries []model.Entry, key string) model.Entry {
	t.Helper()
	for _, e := range entries {
		if e.Key == key {
			return e
		}
	}
	t.Fatalf("entry %q not found", key)
	return model.Entry{}
}

func TestParse_BasicUnquoted(t *testing.T) {
	res := Parse("FOO=bar\nBAZ=123\n")
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", res.Errors)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(res.Entries))
	}
	if findEntry(t, res.Entries, "FOO").Value != "bar" {
		t.Errorf("FOO value mismatch")
	}
}

func TestParse_Comments(t *testing.T) {
	res := Parse("# full line comment\nFOO=bar # inline comment\n")
	if len(res.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(res.Entries))
	}
	e := findEntry(t, res.Entries, "FOO")
	if e.Value != "bar" {
		t.Errorf("expected value 'bar', got %q", e.Value)
	}
	if e.CommentTail != "inline comment" {
		t.Errorf("expected comment tail 'inline comment', got %q", e.CommentTail)
	}
}

func TestParse_UnquotedHashNotPrecededBySpace(t *testing.T) {
	res := Parse("URL=http://example.com/page#section\n")
	e := findEntry(t, res.Entries, "URL")
	if e.Value != "http://example.com/page#section" {
		t.Errorf("expected full URL preserved, got %q", e.Value)
	}
}

func TestParse_SingleQuoted(t *testing.T) {
	res := Parse(`FOO='hello world'` + "\n")
	e := findEntry(t, res.Entries, "FOO")
	if e.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", e.Value)
	}
	if e.Quote != model.SingleQuote {
		t.Errorf("expected SingleQuote style")
	}
}

func TestParse_SingleQuotedNoEscapes(t *testing.T) {
	res := Parse(`FOO='line1\nline2'` + "\n")
	e := findEntry(t, res.Entries, "FOO")
	if e.Value != `line1\nline2` {
		t.Errorf("single quotes must not process escapes, got %q", e.Value)
	}
}

func TestParse_DoubleQuotedWithEscapes(t *testing.T) {
	res := Parse(`FOO="line1\nline2"` + "\n")
	e := findEntry(t, res.Entries, "FOO")
	if e.Value != "line1\nline2" {
		t.Errorf("expected escaped newline processed, got %q", e.Value)
	}
}

func TestParse_DoubleQuotedMultiline(t *testing.T) {
	content := "PRIVATE_KEY=\"-----BEGIN KEY-----\nabc123\ndef456\n-----END KEY-----\"\nNEXT=value\n"
	res := Parse(content)
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", res.Errors)
	}
	pk := findEntry(t, res.Entries, "PRIVATE_KEY")
	expected := "-----BEGIN KEY-----\nabc123\ndef456\n-----END KEY-----"
	if pk.Value != expected {
		t.Errorf("expected multiline value:\n%q\ngot:\n%q", expected, pk.Value)
	}
	next := findEntry(t, res.Entries, "NEXT")
	if next.Value != "value" {
		t.Errorf("expected NEXT to parse correctly after multiline block, got %q", next.Value)
	}
}

func TestParse_UnclosedDoubleQuote(t *testing.T) {
	res := Parse("FOO=\"unterminated\n")
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(res.Errors), res.Errors)
	}
	if res.Errors[0].Kind != model.ErrUnclosedQuote {
		t.Errorf("expected ErrUnclosedQuote, got %s", res.Errors[0].Kind)
	}
}

func TestParse_UnclosedSingleQuote(t *testing.T) {
	res := Parse("FOO='unterminated\n")
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(res.Errors))
	}
	if res.Errors[0].Kind != model.ErrUnclosedQuote {
		t.Errorf("expected ErrUnclosedQuote, got %s", res.Errors[0].Kind)
	}
}

func TestParse_ExportPrefix(t *testing.T) {
	res := Parse("export FOO=bar\n")
	e := findEntry(t, res.Entries, "FOO")
	if !e.Exported {
		t.Errorf("expected Exported=true")
	}
	if e.Value != "bar" {
		t.Errorf("expected value 'bar', got %q", e.Value)
	}
}

func TestParse_DuplicateKeys(t *testing.T) {
	res := Parse("FOO=1\nFOO=2\nBAR=3\n")
	if len(res.DuplicateLines) != 1 {
		t.Fatalf("expected 1 duplicate key, got %d", len(res.DuplicateLines))
	}
	lines, ok := res.DuplicateLines["FOO"]
	if !ok {
		t.Fatalf("expected FOO to be flagged as duplicate")
	}
	if len(lines) != 2 || lines[0] != 1 || lines[1] != 2 {
		t.Errorf("expected duplicate lines [1 2], got %v", lines)
	}
}

func TestParse_EmptyValue(t *testing.T) {
	res := Parse("FOO=\n")
	e := findEntry(t, res.Entries, "FOO")
	if e.Value != "" {
		t.Errorf("expected empty value, got %q", e.Value)
	}
}

func TestParse_InvalidVariableName(t *testing.T) {
	res := Parse("1FOO=bar\n")
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error for invalid name, got %d: %+v", len(res.Errors), res.Errors)
	}
	if res.Errors[0].Kind != model.ErrInvalidName {
		t.Errorf("expected ErrInvalidName, got %s", res.Errors[0].Kind)
	}
}

func TestParse_MissingEquals(t *testing.T) {
	res := Parse("THIS_IS_NOT_VALID\n")
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(res.Errors))
	}
	if res.Errors[0].Kind != model.ErrMissingEquals {
		t.Errorf("expected ErrMissingEquals, got %s", res.Errors[0].Kind)
	}
}

func TestParse_VariableReference(t *testing.T) {
	res := Parse("HOST=localhost\nURL=http://${HOST}:8080\n")
	e := findEntry(t, res.Entries, "URL")
	if !e.HasRefs {
		t.Fatalf("expected HasRefs=true")
	}
	if len(e.Refs) != 1 || e.Refs[0] != "HOST" {
		t.Errorf("expected refs [HOST], got %v", e.Refs)
	}
}

func TestParse_QuotedValueWithComment(t *testing.T) {
	res := Parse(`FOO="bar" # trailing comment` + "\n")
	e := findEntry(t, res.Entries, "FOO")
	if e.Value != "bar" {
		t.Errorf("expected 'bar', got %q", e.Value)
	}
	if e.CommentTail != "trailing comment" {
		t.Errorf("expected 'trailing comment', got %q", e.CommentTail)
	}
}

func TestParse_WhitespaceOnlyAndBlankLinesIgnored(t *testing.T) {
	res := Parse("\n\n   \nFOO=bar\n\n")
	if len(res.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(res.Entries))
	}
}

func TestParse_KeyWithSpaceBeforeEquals(t *testing.T) {
	res := Parse("FOO BAR=baz\n")
	if len(res.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(res.Errors), res.Errors)
	}
}

func TestParse_CRLFLineEndings(t *testing.T) {
	res := Parse("FOO=bar\r\nBAZ=qux\r\n")
	if len(res.Entries) != 2 {
		t.Fatalf("expected 2 entries with CRLF input, got %d", len(res.Entries))
	}
}
