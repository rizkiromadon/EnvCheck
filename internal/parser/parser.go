// Package parser converts raw .env file bytes into structured entries
// without ever writing back to disk.
package parser

import (
	"regexp"
	"strings"

	"github.com/rizkiromadon/envcheck/internal/model"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var looseKeyToken = regexp.MustCompile(`^[^\s=]+`)

var refPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-[^}]*)?\}`)

// Parse reads raw .env content and returns a ParseResult. It never mutates
// or writes the source; callers are responsible for reading the file.
func Parse(content string) model.ParseResult {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")

	result := model.ParseResult{
		DuplicateLines: make(map[string][]int),
	}

	seen := make(map[string][]int)

	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		working := trimmed
		exported := false
		if strings.HasPrefix(working, "export ") || strings.HasPrefix(working, "export\t") {
			exported = true
			working = strings.TrimSpace(working[len("export"):])
		}

		eqIdx := strings.Index(working, "=")
		if eqIdx == -1 {
			result.Errors = append(result.Errors, model.ParseError{
				Kind:    model.ErrMissingEquals,
				Line:    lineNo,
				Message: "expected KEY=VALUE syntax, no '=' found",
				Snippet: sanitizeSnippet(working),
			})
			continue
		}

		keyPart := strings.TrimSpace(working[:eqIdx])
		valuePart := working[eqIdx+1:]

		if keyPart == "" {
			result.Errors = append(result.Errors, model.ParseError{
				Kind:    model.ErrInvalidSyntax,
				Line:    lineNo,
				Message: "missing variable name before '='",
				Snippet: sanitizeSnippet(working),
			})
			continue
		}

		keyToken := looseKeyToken.FindString(keyPart)
		if keyToken != keyPart {
			result.Errors = append(result.Errors, model.ParseError{
				Kind:    model.ErrInvalidSyntax,
				Line:    lineNo,
				Message: "unexpected characters in variable name: '" + keyPart + "'",
				Snippet: sanitizeSnippet(working),
			})
			continue
		}

		if !keyPattern.MatchString(keyToken) {
			result.Errors = append(result.Errors, model.ParseError{
				Kind:    model.ErrInvalidName,
				Line:    lineNo,
				Message: "invalid environment variable name: '" + keyToken + "'",
				Snippet: sanitizeSnippet(keyToken + "=..."),
			})
		}

		value, quote, endLine, consumedErr := extractValue(lines, i, valuePart)
		if consumedErr != nil {
			result.Errors = append(result.Errors, *consumedErr)
			continue
		}

		entry := model.Entry{
			Key:         keyToken,
			Value:       value.resolved,
			RawValue:    value.raw,
			Quote:       quote,
			Exported:    exported,
			Line:        lineNo,
			EndLine:     endLine,
			CommentTail: value.commentTail,
		}

		refs := refPattern.FindAllStringSubmatch(value.raw, -1)
		if len(refs) > 0 {
			entry.HasRefs = true
			for _, m := range refs {
				entry.Refs = append(entry.Refs, m[1])
			}
		}

		result.Entries = append(result.Entries, entry)
		seen[keyToken] = append(seen[keyToken], lineNo)

		if endLine > lineNo {
			i = endLine - 1
		}
	}

	for k, ls := range seen {
		if len(ls) > 1 {
			result.DuplicateLines[k] = ls
		}
	}

	return result
}

// extractedValue holds the raw and resolved forms of a parsed value,
// along with any trailing inline comment.
type extractedValue struct {
	raw         string
	resolved    string
	commentTail string
}

// extractValue interprets the right-hand side of KEY=VALUE and returns the
// parsed value, its quote style, the line number on which it ends, and a
// parse error if a quote was opened but never closed.
func extractValue(lines []string, startIdx int, firstLineRHS string) (extractedValue, model.QuoteStyle, int, *model.ParseError) {
	s := firstLineRHS
	lineNo := startIdx + 1

	trimmedLeading := strings.TrimLeft(s, " \t")
	if trimmedLeading == "" {
		return extractedValue{raw: "", resolved: ""}, model.NoQuote, lineNo, nil
	}

	switch trimmedLeading[0] {
	case '\'':
		body := trimmedLeading[1:]
		if idx := strings.IndexByte(body, '\''); idx != -1 {
			val := body[:idx]
			tail := strings.TrimSpace(body[idx+1:])
			comment := extractInlineComment(tail)
			return extractedValue{raw: val, resolved: val, commentTail: comment}, model.SingleQuote, lineNo, nil
		}
		return extractedValue{}, model.SingleQuote, lineNo, &model.ParseError{
			Kind:    model.ErrUnclosedQuote,
			Line:    lineNo,
			Message: "unclosed single quote",
			Snippet: sanitizeSnippet(s),
		}

	case '"':
		body := trimmedLeading[1:]
		full := body
		curLine := startIdx
		closed := false
		var closeIdx int
		for {
			closeIdx = findUnescapedDoubleQuote(full)
			if closeIdx != -1 {
				closed = true
				break
			}
			curLine++
			if curLine >= len(lines) {
				break
			}
			full = full + "\n" + lines[curLine]
		}
		if !closed {
			return extractedValue{}, model.DoubleQuote, lineNo, &model.ParseError{
				Kind:    model.ErrUnclosedQuote,
				Line:    lineNo,
				Message: "unclosed double quote (scanned to end of file)",
				Snippet: sanitizeSnippet(s),
			}
		}
		raw := full[:closeIdx]
		tail := strings.TrimSpace(full[closeIdx+1:])
		comment := extractInlineComment(tail)
		resolved := unescapeDouble(raw)
		endLine := curLine + 1
		return extractedValue{raw: raw, resolved: resolved, commentTail: comment}, model.DoubleQuote, endLine, nil

	default:
		val, comment := splitUnquotedValueAndComment(s)
		val = strings.TrimSpace(val)
		return extractedValue{raw: val, resolved: val, commentTail: comment}, model.NoQuote, lineNo, nil
	}
}

// findUnescapedDoubleQuote returns the index of the first double quote in s
// that is not preceded by an odd number of backslashes.
func findUnescapedDoubleQuote(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			backslashes := 0
			for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
				backslashes++
			}
			if backslashes%2 == 0 {
				return i
			}
		}
	}
	return -1
}

// splitUnquotedValueAndComment splits an unquoted RHS at the first
// unescaped, whitespace-preceded '#' into (value, commentText).
func splitUnquotedValueAndComment(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' {
			if i == 0 || s[i-1] == ' ' || s[i-1] == '\t' {
				return s[:i], strings.TrimSpace(s[i+1:])
			}
		}
	}
	return s, ""
}

// extractInlineComment strips a leading '#' from tail and returns the
// remaining trimmed comment text, or an empty string if tail is not a comment.
func extractInlineComment(tail string) string {
	tail = strings.TrimSpace(tail)
	if strings.HasPrefix(tail, "#") {
		return strings.TrimSpace(strings.TrimPrefix(tail, "#"))
	}
	return ""
}

// unescapeDouble processes common escape sequences inside a double-quoted
// value: \n \t \r \\ \" and leaves unrecognized escapes as-is.
func unescapeDouble(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case 't':
				b.WriteByte('\t')
				i++
				continue
			case 'r':
				b.WriteByte('\r')
				i++
				continue
			case '"':
				b.WriteByte('"')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// sanitizeSnippet produces a short, secret-free representation of a line for
// error messages, replacing any value content with a fixed placeholder.
func sanitizeSnippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 40 {
		s = s[:40] + "..."
	}
	if idx := strings.Index(s, "="); idx != -1 {
		return s[:idx+1] + "<redacted>"
	}
	return s
}
