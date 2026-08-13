// Package compare implements comparison between a .env file and a
// reference file (typically .env.example), reporting missing, extra, and
// shared variables.
package compare

import (
	"fmt"
	"sort"

	"github.com/rizkiromadon/envcheck/internal/model"
)

// Compare computes the difference between the actual entries (.env) and
// the example/reference entries (.env.example). Duplicate keys in either
// file are reported separately via dup slices, which the caller should
// already have from the parser's DuplicateLines.
func Compare(envEntries, exampleEntries []model.Entry, envDups, exampleDups map[string][]int) model.CompareResult {
	envKeys := keySet(envEntries)
	exampleKeys := keySet(exampleEntries)

	var missing, extra, both []string

	for k := range exampleKeys {
		if _, ok := envKeys[k]; !ok {
			missing = append(missing, k)
		} else {
			both = append(both, k)
		}
	}
	for k := range envKeys {
		if _, ok := exampleKeys[k]; !ok {
			extra = append(extra, k)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(both)

	var envDupKeys, exDupKeys []string
	for k := range envDups {
		envDupKeys = append(envDupKeys, k)
	}
	for k := range exampleDups {
		exDupKeys = append(exDupKeys, k)
	}
	sort.Strings(envDupKeys)
	sort.Strings(exDupKeys)

	return model.CompareResult{
		MissingRequired:     missing,
		ExtraInEnv:          extra,
		InBoth:              both,
		DuplicatesInEnv:     envDupKeys,
		DuplicatesInExample: exDupKeys,
	}
}

// keySet returns the set of variable keys present in entries.
func keySet(entries []model.Entry) map[string]bool {
	m := make(map[string]bool, len(entries))
	for _, e := range entries {
		m[e.Key] = true
	}
	return m
}

// ToFindings converts a CompareResult into normalized Findings for
// unified reporting alongside validation and security results.
func ToFindings(cr model.CompareResult) []model.Finding {
	var findings []model.Finding

	for _, k := range cr.MissingRequired {
		findings = append(findings, model.Finding{
			Category: model.CategoryCompare,
			Severity: model.SeverityError,
			Key:      k,
			Message:  fmt.Sprintf("'%s' is present in the example file but missing from .env", k),
			RuleID:   "compare_missing",
		})
	}
	for _, k := range cr.ExtraInEnv {
		findings = append(findings, model.Finding{
			Category: model.CategoryCompare,
			Severity: model.SeverityInfo,
			Key:      k,
			Message:  fmt.Sprintf("'%s' is present in .env but not documented in the example file", k),
			RuleID:   "compare_extra",
		})
	}
	for _, k := range cr.DuplicatesInEnv {
		findings = append(findings, model.Finding{
			Category: model.CategoryCompare,
			Severity: model.SeverityError,
			Key:      k,
			Message:  fmt.Sprintf("'%s' is duplicated in .env", k),
			RuleID:   "compare_duplicate_env",
		})
	}
	for _, k := range cr.DuplicatesInExample {
		findings = append(findings, model.Finding{
			Category: model.CategoryCompare,
			Severity: model.SeverityWarning,
			Key:      k,
			Message:  fmt.Sprintf("'%s' is duplicated in the example file", k),
			RuleID:   "compare_duplicate_example",
		})
	}

	return findings
}
