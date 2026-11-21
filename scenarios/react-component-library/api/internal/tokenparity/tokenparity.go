// Package tokenparity compares authored CSS custom-property fallbacks with the
// canonical base token vocabulary. It is shared by the corpus gate and the
// per-version publish check so both judge a release by the same rule.
package tokenparity

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/themes"
)

// Fallback is one `var(--property, value)` occurrence in a source file.
type Fallback struct {
	Property string
	Value    string
	Offset   int
}

// Mismatch is a fallback whose value disagrees with the canonical token.
type Mismatch struct {
	File      string
	Offset    int
	Property  string
	Value     string
	Canonical string
}

// CanonicalTokensPath is the base vocabulary every fallback is compared to.
func CanonicalTokensPath(repoRoot string) string {
	return filepath.Join(repoRoot, "templates", "design", "_base", "tokens.css")
}

// LoadCanonical reads the base token vocabulary.
func LoadCanonical(repoRoot string) ([]themes.DesignToken, error) {
	return themes.ReadTokenFile(CanonicalTokensPath(repoRoot))
}

// Normalize collapses whitespace so formatting differences never count.
func Normalize(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

var declarationRE = regexp.MustCompile(`(?m)(--[A-Za-z0-9_-]+)\s*:`)

// CheckVersion compares every fallback in one release's sources with the
// canonical vocabulary. Properties the release declares itself and
// host-computed --rcl-* contracts are component-owned and not compared.
func CheckVersion(canonical []themes.DesignToken, sources map[string][]byte) []Mismatch {
	want := make(map[string]string, len(canonical))
	raw := make(map[string]string, len(canonical))
	for _, token := range canonical {
		want[token.Name] = Normalize(token.Value)
		raw[token.Name] = token.Value
	}
	declared := map[string]bool{}
	files := make([]string, 0, len(sources))
	for file, data := range sources {
		files = append(files, file)
		for _, match := range declarationRE.FindAllSubmatch(data, -1) {
			declared[string(match[1])] = true
		}
	}
	sort.Strings(files)
	var mismatches []Mismatch
	for _, file := range files {
		for _, fallback := range ParseFallbacks(string(sources[file])) {
			canonicalValue, known := want[fallback.Property]
			if !known || declared[fallback.Property] || strings.HasPrefix(fallback.Property, "--rcl-") {
				continue
			}
			if Normalize(fallback.Value) == canonicalValue {
				continue
			}
			mismatches = append(mismatches, Mismatch{File: file, Offset: fallback.Offset, Property: fallback.Property, Value: fallback.Value, Canonical: raw[fallback.Property]})
		}
	}
	return mismatches
}

// ParseFallbacks finds every `var(--property, fallback)` in source, honouring
// nested parentheses inside the fallback.
func ParseFallbacks(source string) []Fallback {
	var result []Fallback
	for offset := 0; offset < len(source); {
		relative := strings.Index(source[offset:], "var(")
		if relative < 0 {
			break
		}
		start := offset + relative
		depth, comma, end := 0, -1, -1
		for i := start + len("var("); i < len(source); i++ {
			switch source[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
				} else {
					depth--
				}
			case ',':
				if depth == 0 && comma < 0 {
					comma = i
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			break
		}
		if comma > 0 {
			property := strings.TrimSpace(source[start+len("var(") : comma])
			if strings.HasPrefix(property, "--") && !strings.Contains(property, "${") {
				result = append(result, Fallback{Property: property, Value: strings.TrimSpace(source[comma+1 : end]), Offset: start})
			}
		}
		offset = end + 1
	}
	return result
}
