package reactvitest

import "strings"

// Config edits address the test block the way Parse finds it (the first
// `test:` key) and then walk real object members, so a property of the same
// name elsewhere (optimizeDeps.include, test.include) is never touched.
var (
	testPath     = []string{"test"}
	coveragePath = []string{"test", "coverage"}
)

// RemoveCoverageThresholds deletes `test.coverage.thresholds` from a Vite or
// Vitest config, together with the comment block written directly above it,
// so the coverage gate has one owner: unit.policy_profile coverage.committed.
// It edits only that property and reports false when the config has none.
func RemoveCoverageThresholds(config string) (string, bool) {
	masked := maskComments(config)
	covOpen, covClose, ok := objectAtPath(masked, coveragePath)
	if !ok {
		return config, false
	}
	keyStart, valueStart, ok := propertySpan(masked, covOpen+1, covClose, "thresholds")
	if !ok {
		return config, false
	}
	end := valueEnd(masked, valueStart, covClose)
	if next := skipInlineSpace(masked, end); next < covClose && masked[next] == ',' {
		end = next + 1
	}

	start := keyStart
	lineStart := strings.LastIndexByte(config[:keyStart], '\n') + 1
	if strings.TrimSpace(config[lineStart:keyStart]) == "" {
		start = lineStart
		if next := skipInlineSpace(config, end); next >= len(config) || config[next] == '\n' {
			end = min(next+1, len(config))
		}
		// The comment block directly above the property documents it.
		for start > 0 {
			prevStart := strings.LastIndexByte(config[:start-1], '\n') + 1
			if !strings.HasPrefix(strings.TrimSpace(config[prevStart:start-1]), "//") {
				break
			}
			start = prevStart
		}
	} else {
		end = skipInlineSpace(config, end)
	}
	return config[:start] + config[end:], true
}

// SetTestProperty sets `test.<key>` to the literal value, replacing an
// existing value or inserting the member. It reports false when the config
// has no test block; the fixer never invents one.
func SetTestProperty(config, key, value string) (string, bool) {
	return setProperty(config, testPath, key, value)
}

// SetCoverageProperty sets `test.coverage.<key>` like SetTestProperty.
func SetCoverageProperty(config, key, value string) (string, bool) {
	return setProperty(config, coveragePath, key, value)
}

// EnsureCoverageStrings makes the string array `test.coverage.<key>` contain
// every value, appending only the missing ones so scenario-specific entries
// survive. A missing property is inserted.
func EnsureCoverageStrings(config, key string, values []string) (string, bool) {
	return ensureStrings(config, coveragePath, key, values)
}

// EnsureTestStrings is EnsureCoverageStrings for `test.<key>`.
func EnsureTestStrings(config, key string, values []string) (string, bool) {
	return ensureStrings(config, testPath, key, values)
}

func ensureStrings(config string, path []string, key string, values []string) (string, bool) {
	masked := maskComments(config)
	open, closeAt, ok := objectAtPath(masked, path)
	if !ok {
		return config, false
	}
	_, valueStart, found := propertySpan(masked, open+1, closeAt, key)
	if !found {
		return setProperty(config, path, key, FormatStringArray(values))
	}
	arrayOpen := skipSpace(masked, valueStart)
	if arrayOpen >= closeAt || masked[arrayOpen] != '[' {
		return setProperty(config, path, key, FormatStringArray(values))
	}
	arrayClose := matchingDelimiter(masked, arrayOpen, '[', ']')
	if arrayClose <= arrayOpen {
		return config, false
	}
	present := stringLiterals(masked[arrayOpen+1 : arrayClose])
	var missing []string
	for _, value := range values {
		if !contains(present, value) {
			missing = append(missing, quoteJS(value))
		}
	}
	if len(missing) == 0 {
		return config, true
	}
	// Append after the last element, keeping a trailing comma style intact.
	last := arrayClose - 1
	for last > arrayOpen && strings.ContainsRune(" \t\r\n", rune(masked[last])) {
		last--
	}
	insert := strings.Join(missing, ", ")
	switch {
	case last == arrayOpen:
		return config[:arrayOpen+1] + insert + config[arrayClose:], true
	case masked[last] == ',':
		return config[:last+1] + " " + insert + "," + config[last+1:], true
	default:
		return config[:last+1] + ", " + insert + config[last+1:], true
	}
}

func setProperty(config string, path []string, key, value string) (string, bool) {
	masked := maskComments(config)
	open, closeAt, ok := objectAtPath(masked, path)
	if !ok {
		return config, false
	}
	if _, valueStart, found := propertySpan(masked, open+1, closeAt, key); found {
		start := skipSpace(masked, valueStart)
		end := valueEnd(masked, start, closeAt)
		return config[:start] + value + config[end:], true
	}
	first := skipSpaceAndCommas(masked, open+1)
	if first >= closeAt {
		return config[:open+1] + " " + key + ": " + value + " " + config[closeAt:], true
	}
	if lineStart := strings.LastIndexByte(config[:first], '\n') + 1; lineStart > open {
		return config[:lineStart] + config[lineStart:first] + key + ": " + value + ",\n" + config[lineStart:], true
	}
	return config[:first] + key + ": " + value + ", " + config[first:], true
}

// FormatStringArray renders values as a single-quoted JS array literal.
func FormatStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteJS(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func quoteJS(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `\'`) + "'"
}

// objectAtPath resolves a member path to the offsets of its object braces.
func objectAtPath(masked string, path []string) (int, int, bool) {
	valueStart, ok := propertyValueIndex(masked, path[0])
	if !ok {
		return 0, 0, false
	}
	open, closeAt, ok := braceSpan(masked, valueStart, len(masked))
	for _, key := range path[1:] {
		if !ok {
			return 0, 0, false
		}
		open, closeAt, ok = objectSpan(masked, open+1, closeAt, key)
	}
	return open, closeAt, ok
}

// objectSpan finds `key: { ... }` inside masked[from:to] and returns the
// offsets of its braces.
func objectSpan(masked string, from, to int, key string) (int, int, bool) {
	_, valueStart, ok := propertySpan(masked, from, to, key)
	if !ok {
		return 0, 0, false
	}
	return braceSpan(masked, valueStart, to)
}

// braceSpan returns the offsets of the object literal starting at the first
// non-space character at or after valueStart.
func braceSpan(masked string, valueStart, to int) (int, int, bool) {
	open := skipSpace(masked, valueStart)
	if open >= to || masked[open] != '{' {
		return 0, 0, false
	}
	closeAt := matchingDelimiter(masked, open, '{', '}')
	if closeAt <= open || closeAt > to {
		return 0, 0, false
	}
	return open, closeAt, true
}

// propertySpan returns the offset of a property name and of its value
// within masked[from:to], visiting only that object's own members.
func propertySpan(masked string, from, to int, key string) (int, int, bool) {
	for i := from; i < to; {
		i = skipSpaceAndCommas(masked, i)
		if i >= to {
			break
		}
		if strings.HasPrefix(masked[i:], "...") {
			i = valueEnd(masked, i+3, to)
			continue
		}
		name, next, ok := readPropertyName(masked, i)
		if !ok {
			return 0, 0, false
		}
		next = skipSpace(masked, next)
		if next >= to || masked[next] != ':' {
			return 0, 0, false
		}
		if name == key {
			return i, next + 1, true
		}
		i = valueEnd(masked, next+1, to)
	}
	return 0, 0, false
}

// valueEnd returns the offset just past the value starting at start: the
// first top-level comma or closing brace ends it.
func valueEnd(masked string, start, limit int) int {
	depth := 0
	var quote byte
	escaped := false
	last := start
	for i := start; i < limit; i++ {
		c := masked[i]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			last = i + 1
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			if depth == 0 {
				return last
			}
			depth--
		case ',':
			if depth == 0 {
				return last
			}
		}
		if !strings.ContainsRune(" \t\n\r", rune(c)) {
			last = i + 1
		}
	}
	return last
}

func skipInlineSpace(src string, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r') {
		i++
	}
	return i
}

// maskComments blanks comments with spaces, keeping newlines and every
// offset, so positions found in the masked text address the original. String
// and regular-expression literals are kept verbatim: a regex such as
// /@vrooli\// must not be read as the start of a line comment.
func maskComments(src string) string {
	out := []byte(src)
	for i := 0; i < len(out); i++ {
		c := out[i]
		switch {
		case c == '\'' || c == '"' || c == '`':
			i = skipQuoted(out, i, c)
		case c == '/' && i+1 < len(out) && out[i+1] == '/':
			for ; i < len(out) && out[i] != '\n'; i++ {
				out[i] = ' '
			}
		case c == '/' && i+1 < len(out) && out[i+1] == '*':
			end := strings.Index(string(out[i+2:]), "*/")
			stop := len(out)
			if end >= 0 {
				stop = i + 2 + end + 2
			}
			for ; i < stop; i++ {
				if out[i] != '\n' {
					out[i] = ' '
				}
			}
			i--
		case c == '/' && regexCanStart(out, i):
			i = skipRegex(out, i)
		}
	}
	return string(out)
}

// skipQuoted returns the offset of the closing quote of the literal at i.
func skipQuoted(src []byte, i int, quote byte) int {
	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case quote:
			return j
		}
	}
	return len(src)
}

// regexCanStart applies the usual JavaScript heuristic: a slash begins a
// regular expression where an expression is expected, i.e. after an operator
// or opening punctuation rather than after a value.
func regexCanStart(src []byte, i int) bool {
	j := i - 1
	for j >= 0 && strings.ContainsRune(" \t\r\n", rune(src[j])) {
		j--
	}
	return j < 0 || strings.ContainsRune("(,=:[!&|?{};+-*%<>~^", rune(src[j]))
}

// skipRegex returns the offset of the closing slash of the regex at i,
// honoring escapes and character classes. An unterminated candidate on the
// line is treated as division.
func skipRegex(src []byte, i int) int {
	inClass := false
	for j := i + 1; j < len(src) && src[j] != '\n'; j++ {
		switch src[j] {
		case '\\':
			j++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				return j
			}
		}
	}
	return i
}
