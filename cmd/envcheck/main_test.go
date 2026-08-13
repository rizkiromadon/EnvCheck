package main

import "testing"

func TestParseArgs_Defaults(t *testing.T) {
	o, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.targetFile != "" {
		t.Errorf("expected empty target file by default, got %q", o.targetFile)
	}
}

func TestParseArgs_PositionalFile(t *testing.T) {
	o, err := parseArgs([]string{".env.production"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.targetFile != ".env.production" {
		t.Errorf("expected target file '.env.production', got %q", o.targetFile)
	}
}

func TestParseArgs_SubcommandAliases(t *testing.T) {
	for _, sub := range []string{"check", "doctor"} {
		o, err := parseArgs([]string{sub, ".env"})
		if err != nil {
			t.Fatalf("unexpected error for subcommand %q: %v", sub, err)
		}
		if o.targetFile != ".env" {
			t.Errorf("subcommand %q: expected target file '.env', got %q", sub, o.targetFile)
		}
	}
}

func TestParseArgs_CompareFlag(t *testing.T) {
	o, err := parseArgs([]string{".env", "--compare", ".env.example"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.compareTo != ".env.example" {
		t.Errorf("expected compareTo '.env.example', got %q", o.compareTo)
	}
}

func TestParseArgs_SchemaFlag(t *testing.T) {
	o, err := parseArgs([]string{".env", "--schema", ".env.schema"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.schemaFile != ".env.schema" {
		t.Errorf("expected schemaFile '.env.schema', got %q", o.schemaFile)
	}
}

func TestParseArgs_BooleanFlags(t *testing.T) {
	o, err := parseArgs([]string{"--json", "--strict", "--quiet", "--no-color", "--no-git", "--no-security", "--allow-empty"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !o.jsonOut || !o.strict || !o.quiet || !o.noColor || !o.noGit || !o.noSecurity || !o.allowEmpty {
		t.Errorf("expected all boolean flags true, got %+v", o)
	}
}

func TestParseArgs_UnknownFlag(t *testing.T) {
	_, err := parseArgs([]string{"--bogus"})
	if err == nil {
		t.Fatalf("expected error for unknown flag")
	}
}

func TestParseArgs_CompareMissingValue(t *testing.T) {
	_, err := parseArgs([]string{"--compare"})
	if err == nil {
		t.Fatalf("expected error when --compare has no value")
	}
}

func TestParseArgs_ExtraPositionalArg(t *testing.T) {
	_, err := parseArgs([]string{".env", "extra.env"})
	if err == nil {
		t.Fatalf("expected error for unexpected extra positional argument")
	}
}

func TestParseArgs_HelpAndVersion(t *testing.T) {
	o, err := parseArgs([]string{"--help"})
	if err != nil || !o.showHelp {
		t.Errorf("expected showHelp=true, err=nil, got showHelp=%v err=%v", o.showHelp, err)
	}
	o, err = parseArgs([]string{"-v"})
	if err != nil || !o.showVer {
		t.Errorf("expected showVer=true, err=nil, got showVer=%v err=%v", o.showVer, err)
	}
}
