package security

import (
	"regexp"

	"github.com/rizkiromadon/envcheck/internal/model"
)

// Rule defines one detection heuristic for the security scanner.
type Rule struct {
	ID           string
	Description  string
	Severity     model.Severity
	KeyPattern   *regexp.Regexp
	ValuePattern *regexp.Regexp
	MinValueLen  int
}

// ci compiles a case-insensitive regexp.
func ci(pattern string) *regexp.Regexp {
	return regexp.MustCompile("(?i)" + pattern)
}

// DefaultRules is the built-in rule set.
var DefaultRules = []Rule{
	{
		ID:           "private_key_block",
		Description:  "PEM-formatted private key",
		Severity:     model.SeverityCritical,
		ValuePattern: ci(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
	},
	{
		ID:           "jwt_token",
		Description:  "JSON Web Token (JWT)",
		Severity:     model.SeverityCritical,
		ValuePattern: regexp.MustCompile(`^eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*$`),
	},
	{
		ID:           "aws_access_key_id",
		Description:  "AWS Access Key ID",
		Severity:     model.SeverityCritical,
		ValuePattern: regexp.MustCompile(`^(AKIA|ASIA)[0-9A-Z]{16}$`),
	},
	{
		ID:           "aws_secret_access_key",
		Description:  "possible AWS Secret Access Key",
		Severity:     model.SeverityCritical,
		KeyPattern:   ci(`aws.*secret`),
		ValuePattern: regexp.MustCompile(`^[A-Za-z0-9/+=]{40}$`),
	},
	{
		ID:           "github_token",
		Description:  "GitHub personal access / app token",
		Severity:     model.SeverityCritical,
		ValuePattern: regexp.MustCompile(`^(ghp|gho|ghu|ghs|ghr|github_pat)_[A-Za-z0-9_]{20,}$`),
	},
	{
		ID:           "slack_token",
		Description:  "Slack API token",
		Severity:     model.SeverityCritical,
		ValuePattern: regexp.MustCompile(`^xox[baprs]-[A-Za-z0-9-]{10,}$`),
	},
	{
		ID:           "stripe_key",
		Description:  "Stripe API key",
		Severity:     model.SeverityCritical,
		ValuePattern: regexp.MustCompile(`^(sk|pk|rk)_(live|test)_[A-Za-z0-9]{16,}$`),
	},
	{
		ID:           "google_api_key",
		Description:  "Google API key",
		Severity:     model.SeverityCritical,
		ValuePattern: regexp.MustCompile(`^AIza[0-9A-Za-z_-]{35}$`),
	},
	{
		ID:           "database_url_with_credentials",
		Description:  "database connection string containing embedded credentials",
		Severity:     model.SeverityCritical,
		ValuePattern: regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://[^:/\s]+:[^@/\s]+@`),
	},
	{
		ID:          "generic_password",
		Description: "possible password",
		Severity:    model.SeverityWarning,
		KeyPattern:  ci(`(password|passwd|pwd)`),
		MinValueLen: 1,
	},
	{
		ID:          "generic_secret",
		Description: "possible secret",
		Severity:    model.SeverityWarning,
		KeyPattern:  ci(`secret`),
		MinValueLen: 4,
	},
	{
		ID:          "generic_token",
		Description: "possible token",
		Severity:    model.SeverityWarning,
		KeyPattern:  ci(`(^|_)token(_|$)`),
		MinValueLen: 8,
	},
	{
		ID:          "generic_api_key",
		Description: "possible API key",
		Severity:    model.SeverityWarning,
		KeyPattern:  ci(`api[_-]?key`),
		MinValueLen: 8,
	},
	{
		ID:          "private_key_named",
		Description: "variable name suggests a private key",
		Severity:    model.SeverityWarning,
		KeyPattern:  ci(`private[_-]?key`),
		MinValueLen: 1,
	},
	{
		ID:          "credential_generic",
		Description: "variable name suggests a credential",
		Severity:    model.SeverityInfo,
		KeyPattern:  ci(`(credential|auth[_-]?token|access[_-]?key)`),
		MinValueLen: 1,
	},
}

var placeholderPattern = ci(`^(changeme|change_me|your[_-]?.*here|xxx+|placeholder|example|todo|<.*>|\*+|password123|test|dummy|fake|none|null|n/?a)$`)

// isPlaceholder reports whether a value looks like an obvious non-secret
// placeholder, to reduce false positives from generic rules.
func isPlaceholder(value string) bool {
	return placeholderPattern.MatchString(value)
}
