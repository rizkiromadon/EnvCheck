// Package security scans parsed .env entries for likely secrets (API keys,
// tokens, passwords, private keys, database credentials, JWTs) using an
// extensible rule set. It never prints secret values in full; all matches
// are masked before being attached to a Finding.
package security

import (
	"fmt"
	"strings"

	"github.com/rizkiromadon/envcheck/internal/model"
)

// Scanner runs security detection rules against parsed entries.
type Scanner struct {
	rules []Rule
}

// New creates a Scanner with the default built-in rule set.
func New() *Scanner {
	return &Scanner{rules: DefaultRules}
}

// NewWithRules creates a Scanner with a custom rule set, e.g. defaults plus
// user-supplied additions, for callers that want to extend detection.
func NewWithRules(rules []Rule) *Scanner {
	return &Scanner{rules: rules}
}

// Scan evaluates every entry against every rule and returns one Finding per
// match. If multiple rules match the same entry, the highest-severity,
// first-matching rule wins to avoid noisy duplicate findings on one line.
func (s *Scanner) Scan(entries []model.Entry) []model.Finding {
	var findings []model.Finding

	for _, e := range entries {
		if e.Value == "" {
			continue
		}
		if isPlaceholder(e.Value) {
			continue
		}

		var best *Rule
		for i := range s.rules {
			r := &s.rules[i]
			if ruleMatches(r, e) {
				if best == nil || severityRank(r.Severity) > severityRank(best.Severity) {
					best = r
				}
			}
		}

		if best != nil {
			findings = append(findings, model.Finding{
				Category: model.CategorySecurity,
				Severity: best.Severity,
				Key:      e.Key,
				Line:     e.Line,
				Message:  fmt.Sprintf("possible %s detected in '%s'", best.Description, e.Key),
				Masked:   Mask(e.Value),
				RuleID:   best.ID,
			})
		}
	}

	return findings
}

// ruleMatches reports whether rule r matches entry e.
func ruleMatches(r *Rule, e model.Entry) bool {
	if r.KeyPattern != nil && !r.KeyPattern.MatchString(e.Key) {
		return false
	}
	if r.ValuePattern != nil && !r.ValuePattern.MatchString(e.Value) {
		return false
	}
	if r.ValuePattern == nil && len(e.Value) < r.MinValueLen {
		return false
	}
	return true
}

// severityRank returns a numeric rank for a severity, higher meaning more severe.
func severityRank(s model.Severity) int {
	switch s {
	case model.SeverityCritical:
		return 3
	case model.SeverityWarning:
		return 2
	case model.SeverityInfo:
		return 1
	default:
		return 0
	}
}

// Mask redacts a secret value for safe display. It never reveals the full
// value. For short values it returns a fixed-length mask; for longer values
// it may reveal a tiny prefix to aid recognition without leaking the secret.
func Mask(value string) string {
	n := len(value)
	switch {
	case n == 0:
		return ""
	case n <= 4:
		return strings.Repeat("*", 8)
	case n <= 12:
		return value[:1] + strings.Repeat("*", 7)
	default:
		return value[:2] + strings.Repeat("*", 6) + value[n-2:n-1] + "*"
	}
}
