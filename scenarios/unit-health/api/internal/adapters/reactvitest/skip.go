package reactvitest

import (
	"strings"
	"unicode"

	"unit-health/internal/testquality"
)

type skipCall struct {
	start int
	open  int
	close int
}

// projectSkipDeclaration owns the source-required Vitest skip rule. Quality
// Health owns the native ESLint profile for the other syntax rules; its
// reviewed profile intentionally does not claim skip declarations.
func projectSkipDeclaration(source, file string) testquality.Result {
	base := testquality.Result{
		RuleID: "skip-declaration", RuleVersion: "1",
		Target:   testquality.Target{File: file, Scope: "file"},
		TestKind: "unit", SupportProfile: "vitest-syntax-1.6.9",
		Status: testquality.CheckedClean, Reason: testquality.ReasonNone,
		EvidenceKind: testquality.Static, Enforcement: testquality.Advisory,
		Severity: testquality.Warning,
	}
	if strings.TrimSpace(source) == "" {
		base.Status, base.Reason = testquality.Unknown, testquality.MissingInput
		return base
	}
	calls, conditional := vitestSkipCalls(source)
	if conditional {
		base.Status, base.Reason = testquality.Unknown, testquality.MissingAnalysis
		base.Limitations = []string{"Conditional skipIf declarations require execution-aware interpretation."}
		return base
	}
	for _, call := range calls {
		body, ok := skipCallbackBody(source, call)
		if !ok {
			base.Status, base.Reason = testquality.Unknown, testquality.MissingAnalysis
			base.Location = sourceLocation(source, call.start)
			return base
		}
		if strings.TrimSpace(stripJSComments(body)) == "" {
			base.Status, base.Reason = testquality.Violation, testquality.ReasonNone
			base.Location = sourceLocation(source, call.start)
			return base
		}
		base.Status, base.Reason = testquality.Unknown, testquality.MissingAnalysis
		base.Location = sourceLocation(source, call.start)
	}
	return base
}

func vitestSkipCalls(source string) ([]skipCall, bool) {
	var calls []skipCall
	conditional := false
	for i := 0; i < len(source); {
		if next, ok := skipJSTrivia(source, i); ok {
			i = next
			continue
		}
		if !isIdentifierStart(source[i]) {
			i++
			continue
		}
		start := i
		for i < len(source) && isIdentifierPart(source[i]) {
			i++
		}
		name := source[start:i]
		if name != "test" && name != "it" && name != "describe" {
			continue
		}
		i = skipSpaces(source, i)
		if i >= len(source) || source[i] != '.' {
			continue
		}
		i = skipSpaces(source, i+1)
		methodStart := i
		for i < len(source) && isIdentifierPart(source[i]) {
			i++
		}
		method := source[methodStart:i]
		i = skipSpaces(source, i)
		if method != "skip" && method != "skipIf" {
			continue
		}
		if method == "skipIf" {
			conditional = true
			continue
		}
		if i >= len(source) || source[i] != '(' {
			continue
		}
		close, ok := matchingJSDelimiter(source, i, '(', ')')
		if !ok {
			calls = append(calls, skipCall{start: start, open: i})
			continue
		}
		calls = append(calls, skipCall{start: start, open: i, close: close})
		i = close + 1
	}
	return calls, conditional
}

func skipCallbackBody(source string, call skipCall) (string, bool) {
	if call.close <= call.open {
		return "", false
	}
	inside := source[call.open+1 : call.close]
	arrow := findToken(inside, "=>")
	if arrow < 0 {
		return "", false
	}
	bodyStart := skipSpaces(inside, arrow+2)
	if bodyStart >= len(inside) || inside[bodyStart] != '{' {
		return "", false
	}
	bodyEnd, ok := matchingJSDelimiter(inside, bodyStart, '{', '}')
	if !ok {
		return "", false
	}
	return inside[bodyStart+1 : bodyEnd], true
}

func findToken(source, token string) int {
	for i := 0; i <= len(source)-len(token); {
		if next, ok := skipJSTrivia(source, i); ok {
			i = next
			continue
		}
		if source[i:i+len(token)] == token {
			return i
		}
		i++
	}
	return -1
}

func matchingJSDelimiter(source string, start int, open, close byte) (int, bool) {
	depth := 0
	for i := start; i < len(source); {
		if next, ok := skipJSTrivia(source, i); ok {
			i = next
			continue
		}
		if source[i] == '\'' || source[i] == '"' || source[i] == '`' {
			i = skipJSString(source, i)
			continue
		}
		switch source[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i, true
			}
		}
		i++
	}
	return 0, false
}

func skipJSTrivia(source string, i int) (int, bool) {
	if i >= len(source) {
		return i, false
	}
	if source[i] == '\'' || source[i] == '"' || source[i] == '`' {
		return skipJSString(source, i), true
	}
	if source[i] == '/' && i+1 < len(source) && source[i+1] == '/' {
		i += 2
		for i < len(source) && source[i] != '\n' {
			i++
		}
		return i, true
	}
	if source[i] == '/' && i+1 < len(source) && source[i+1] == '*' {
		i += 2
		for i+1 < len(source) && (source[i] != '*' || source[i+1] != '/') {
			i++
		}
		if i+1 < len(source) {
			i += 2
		}
		return i, true
	}
	return i, false
}

func skipJSString(source string, i int) int {
	quote := source[i]
	i++
	for i < len(source) {
		if source[i] == '\\' {
			i += 2
			continue
		}
		if source[i] == quote {
			return i + 1
		}
		i++
	}
	return i
}

func skipSpaces(source string, i int) int {
	for i < len(source) && unicode.IsSpace(rune(source[i])) {
		i++
	}
	return i
}

func isIdentifierStart(b byte) bool { return b == '_' || b == '$' || unicode.IsLetter(rune(b)) }
func isIdentifierPart(b byte) bool  { return isIdentifierStart(b) || unicode.IsDigit(rune(b)) }

func stripJSComments(source string) string {
	var b strings.Builder
	for i := 0; i < len(source); {
		if next, ok := skipJSTrivia(source, i); ok {
			if source[i] == '\'' || source[i] == '"' || source[i] == '`' {
				b.WriteByte('x')
			} else {
				b.WriteByte(' ')
			}
			i = next
			continue
		}
		b.WriteByte(source[i])
		i++
	}
	return b.String()
}

func sourceLocation(source string, offset int) testquality.Location {
	line, column := 1, 1
	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return testquality.Location{Line: line, Column: column}
}
