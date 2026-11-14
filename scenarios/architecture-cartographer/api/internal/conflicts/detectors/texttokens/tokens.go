// Package texttokens owns the normalization shared by text-based detectors.
package texttokens

import (
	"sort"
	"strings"
	"unicode"
)

// Extract splits paths and CamelCase words into a stable set of lowercase
// tokens. A detector can supply its own ignored vocabulary without
// reimplementing the normalization rules.
func Extract(text string, ignored map[string]struct{}) []string {
	words := make(map[string]struct{})
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		word := strings.ToLower(string(current))
		current = current[:0]
		if len(word) >= 3 {
			if _, skip := ignored[word]; !skip {
				words[word] = struct{}{}
			}
		}
	}
	for i, r := range text {
		switch {
		case r == '_' || r == '-' || r == '/' || r == '.' || unicode.IsSpace(r):
			flush()
		case unicode.IsUpper(r) && i > 0 && len(current) > 0 && !unicode.IsUpper(current[len(current)-1]):
			flush()
			current = append(current, unicode.ToLower(r))
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			current = append(current, unicode.ToLower(r))
		default:
			flush()
		}
	}
	flush()
	return SortedKeys(words)
}

// SortedKeys returns a deterministic projection of a string set.
func SortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
