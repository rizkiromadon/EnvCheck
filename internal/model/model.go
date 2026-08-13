// Package model defines the core data structures shared across EnvCheck's
// parser, validator, security scanner, schema engine, and formatter.
package model

// QuoteStyle records how a value was quoted in the source file.
type QuoteStyle int

// The recognized quote styles for a parsed value.
const (
	NoQuote QuoteStyle = iota
	SingleQuote
	DoubleQuote
)

// String returns a human-readable description of the quote style.
func (q QuoteStyle) String() string {
	switch q {
	case SingleQuote:
		return "single-quoted"
	case DoubleQuote:
		return "double-quoted"
	default:
		return "unquoted"
	}
}

// Entry represents a single KEY=VALUE line parsed from a .env file.
type Entry struct {
	Key         string
	Value       string
	RawValue    string
	Quote       QuoteStyle
	Exported    bool
	Line        int
	EndLine     int
	HasRefs     bool
	Refs        []string
	CommentTail string
}

// ParseErrorKind enumerates the categories of syntax errors the parser can emit.
type ParseErrorKind string

// The recognized parse error kinds.
const (
	ErrUnclosedQuote     ParseErrorKind = "unclosed_quote"
	ErrInvalidSyntax     ParseErrorKind = "invalid_syntax"
	ErrInvalidName       ParseErrorKind = "invalid_variable_name"
	ErrMissingEquals     ParseErrorKind = "missing_equals"
	ErrUnterminatedMulti ParseErrorKind = "unterminated_multiline"
)

// ParseError is a syntax-level problem found while parsing raw text into entries.
type ParseError struct {
	Kind    ParseErrorKind
	Line    int
	Column  int
	Message string
	Snippet string
}

// ParseResult is the output of parsing a .env file: zero or more valid
// entries, plus zero or more syntax errors.
type ParseResult struct {
	Entries        []Entry
	Errors         []ParseError
	DuplicateLines map[string][]int
}

// Severity indicates how serious a finding is.
type Severity string

// The recognized severity levels.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityError    Severity = "error"
)

// FindingCategory groups findings by which subsystem produced them.
type FindingCategory string

// The recognized finding categories.
const (
	CategorySyntax   FindingCategory = "syntax"
	CategoryNaming   FindingCategory = "naming"
	CategoryDup      FindingCategory = "duplicate"
	CategoryEmpty    FindingCategory = "empty_value"
	CategoryQuote    FindingCategory = "quote"
	CategoryRef      FindingCategory = "reference"
	CategoryType     FindingCategory = "type"
	CategorySecurity FindingCategory = "security"
	CategoryCompare  FindingCategory = "compare"
	CategoryGit      FindingCategory = "git"
	CategorySchema   FindingCategory = "schema"
)

// Finding is a single, unified diagnostic emitted by any subsystem.
type Finding struct {
	Category FindingCategory `json:"category"`
	Severity Severity        `json:"severity"`
	Key      string          `json:"key,omitempty"`
	Line     int             `json:"line,omitempty"`
	Message  string          `json:"message"`
	Masked   string          `json:"masked,omitempty"`
	RuleID   string          `json:"rule_id,omitempty"`
}

// Report is the complete, aggregated result of running EnvCheck against a file.
type Report struct {
	ToolVersion   string    `json:"tool_version"`
	File          string    `json:"file"`
	VariableCount int       `json:"variable_count"`
	Findings      []Finding `json:"findings"`

	ErrorCount    int `json:"error_count"`
	WarningCount  int `json:"warning_count"`
	CriticalCount int `json:"critical_count"`
	InfoCount     int `json:"info_count"`

	GitTracked *bool `json:"git_tracked,omitempty"`

	CompareResult *CompareResult `json:"compare_result,omitempty"`
}

// CompareResult holds the outcome of comparing a .env file against an
// .env.example (or similar reference) file.
type CompareResult struct {
	MissingRequired     []string `json:"missing_required"`
	ExtraInEnv          []string `json:"extra_in_env"`
	InBoth              []string `json:"in_both"`
	DuplicatesInEnv     []string `json:"duplicates_in_env"`
	DuplicatesInExample []string `json:"duplicates_in_example"`
}

// Recompute updates the derived severity counters on the report based on
// its current Findings slice. Call after mutating Findings.
func (r *Report) Recompute() {
	r.ErrorCount, r.WarningCount, r.CriticalCount, r.InfoCount = 0, 0, 0, 0
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			r.ErrorCount++
		case SeverityWarning:
			r.WarningCount++
		case SeverityCritical:
			r.CriticalCount++
		case SeverityInfo:
			r.InfoCount++
		}
	}
}
