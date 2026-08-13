// Package schema implements EnvCheck's lightweight schema format for
// declaring per-variable requirements, e.g.:
//
//	PORT=number|required
//	DATABASE_URL=url|required|secret
//	NODE_ENV=enum:development,production,test|required
//	DEBUG=boolean|optional
package schema

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/rizkiromadon/envcheck/internal/model"
)

// ValueType enumerates the supported type checks.
type ValueType string

// The recognized value types.
const (
	TypeString  ValueType = "string"
	TypeNumber  ValueType = "number"
	TypeBoolean ValueType = "boolean"
	TypeURL     ValueType = "url"
	TypePort    ValueType = "port"
	TypeEnum    ValueType = "enum"
)

// Rule is the parsed schema definition for a single variable.
type Rule struct {
	Key      string
	Type     ValueType
	EnumVals []string
	Required bool
	Secret   bool
	LineNo   int
}

// Schema is an ordered collection of rules, keyed by variable name.
type Schema struct {
	Rules []Rule
	byKey map[string]*Rule
}

var boolValues = map[string]bool{
	"true": true, "false": true, "1": true, "0": true,
	"yes": true, "no": true, "on": true, "off": true,
}

// Parse reads a schema file's raw text and returns a Schema. Malformed lines
// produce an error collected in errs; parsing continues past bad lines.
func Parse(content string) (*Schema, []error) {
	var errs []error
	sc := &Schema{byKey: make(map[string]*Rule)}

	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 {
			errs = append(errs, fmt.Errorf("schema line %d: expected KEY=type|modifiers", lineNo))
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		spec := strings.TrimSpace(line[eqIdx+1:])
		if key == "" || spec == "" {
			errs = append(errs, fmt.Errorf("schema line %d: empty key or spec", lineNo))
			continue
		}

		parts := strings.Split(spec, "|")
		typePart := strings.TrimSpace(parts[0])

		rule := Rule{Key: key, LineNo: lineNo, Required: false}

		if strings.HasPrefix(typePart, "enum:") {
			rule.Type = TypeEnum
			valsRaw := strings.TrimPrefix(typePart, "enum:")
			for _, v := range strings.Split(valsRaw, ",") {
				v = strings.TrimSpace(v)
				if v != "" {
					rule.EnumVals = append(rule.EnumVals, v)
				}
			}
			if len(rule.EnumVals) == 0 {
				errs = append(errs, fmt.Errorf("schema line %d: enum type requires at least one value", lineNo))
				continue
			}
		} else {
			switch ValueType(typePart) {
			case TypeString, TypeNumber, TypeBoolean, TypeURL, TypePort:
				rule.Type = ValueType(typePart)
			default:
				errs = append(errs, fmt.Errorf("schema line %d: unknown type '%s'", lineNo, typePart))
				continue
			}
		}

		for _, mod := range parts[1:] {
			mod = strings.ToLower(strings.TrimSpace(mod))
			switch mod {
			case "required":
				rule.Required = true
			case "optional":
				rule.Required = false
			case "secret":
				rule.Secret = true
			case "":
			default:
				errs = append(errs, fmt.Errorf("schema line %d: unknown modifier '%s'", lineNo, mod))
			}
		}

		sc.Rules = append(sc.Rules, rule)
	}

	for i := range sc.Rules {
		sc.byKey[sc.Rules[i].Key] = &sc.Rules[i]
	}

	return sc, errs
}

var portPattern = regexp.MustCompile(`^[0-9]+$`)

// Validate checks parsed .env entries against the schema and returns
// normalized findings, reporting missing required variables and type
// mismatches.
func (sc *Schema) Validate(entries []model.Entry) []model.Finding {
	var findings []model.Finding

	values := make(map[string]model.Entry, len(entries))
	for _, e := range entries {
		values[e.Key] = e
	}

	for _, rule := range sc.Rules {
		entry, present := values[rule.Key]
		if !present {
			if rule.Required {
				findings = append(findings, model.Finding{
					Category: model.CategorySchema,
					Severity: model.SeverityError,
					Key:      rule.Key,
					Message:  fmt.Sprintf("required variable '%s' is missing (schema line %d)", rule.Key, rule.LineNo),
					RuleID:   "schema_missing_required",
				})
			}
			continue
		}

		if err := checkType(rule, entry.Value); err != "" {
			findings = append(findings, model.Finding{
				Category: model.CategorySchema,
				Severity: model.SeverityError,
				Key:      rule.Key,
				Line:     entry.Line,
				Message:  fmt.Sprintf("'%s' %s", rule.Key, err),
				RuleID:   "schema_type_mismatch",
			})
		}
	}

	return findings
}

// checkType checks value against rule's declared type and returns a
// human-readable error message, or an empty string if it is valid.
func checkType(rule Rule, value string) string {
	if value == "" {
		return ""
	}
	switch rule.Type {
	case TypeString:
		return ""
	case TypeNumber:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Sprintf("must be a number, got '%s'", value)
		}
	case TypeBoolean:
		if !boolValues[strings.ToLower(value)] {
			return fmt.Sprintf("must be a boolean (true/false/1/0/yes/no), got '%s'", value)
		}
	case TypeURL:
		u, err := url.Parse(value)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Sprintf("must be a valid URL, got '%s'", value)
		}
	case TypePort:
		if !portPattern.MatchString(value) {
			return fmt.Sprintf("must be a valid port number, got '%s'", value)
		}
		n, _ := strconv.Atoi(value)
		if n < 1 || n > 65535 {
			return fmt.Sprintf("must be between 1 and 65535, got '%s'", value)
		}
	case TypeEnum:
		for _, v := range rule.EnumVals {
			if v == value {
				return ""
			}
		}
		return fmt.Sprintf("must be one of [%s], got '%s'", strings.Join(rule.EnumVals, ", "), value)
	}
	return ""
}
