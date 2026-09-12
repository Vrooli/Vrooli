// Package reactvitest owns React/Vitest configuration projection parsing.
// Validation consumes the resulting observations but does not interpret Vite
// object syntax itself.
package reactvitest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Projection is the observed native Vitest/Vite projection.
type Projection struct {
	HasVitestConfig    bool
	Environment        string
	SetupFiles         []string
	CoverageProvider   string
	CoverageReporters  []string
	CoverageInclude    []string
	CoverageExclude    []string
	ReportOnFailure    bool
	HasReportOnFailure bool
	Thresholds         map[string]float64
	HasImportBanRule   bool
}

// Parse reads the bounded subset of Vite and ESLint configuration needed by
// the React/Vitest adapter. It is syntax-tolerant and never executes config.
func Parse(config, eslint string) Projection {
	p := Projection{Thresholds: map[string]float64{}}
	clean := stripComments(config)
	testBlock, ok := objectValueBlock(clean, "test")
	p.HasVitestConfig = ok
	coverageBlock, _ := objectValueBlock(testBlock, "coverage")
	if values := stringArrayPropertyValues(testBlock, "environment"); len(values) == 1 {
		p.Environment = values[0]
	}
	p.SetupFiles = stringArrayPropertyValues(testBlock, "setupFiles")
	if values := stringArrayPropertyValues(coverageBlock, "provider"); len(values) == 1 {
		p.CoverageProvider = values[0]
	}
	p.CoverageReporters = stringArrayPropertyValues(coverageBlock, "reporter")
	p.CoverageInclude = stringArrayPropertyValues(coverageBlock, "include")
	p.CoverageExclude = stringArrayPropertyValues(coverageBlock, "exclude")
	p.ReportOnFailure, p.HasReportOnFailure = booleanProperty(coverageBlock, "reportOnFailure")
	thresholdBlock, _ := objectValueBlock(coverageBlock, "thresholds")
	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		if value, ok := numericProperty(thresholdBlock, key); ok {
			p.Thresholds[key] = value
		}
	}
	p.HasImportBanRule = hasESLintImportBanProjection(eslint)
	return p
}

func ContainsAllStrings(have, want []string) bool {
	for _, value := range want {
		if !contains(have, value) {
			return false
		}
	}
	return true
}

func ContainsAllSetupFiles(have, want []string) bool {
	for _, value := range want {
		found := false
		for _, candidate := range have {
			if normalizeSetupPath(candidate) == normalizeSetupPath(value) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func normalizeSetupPath(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(path)), "./")
}

func hasESLintImportBanProjection(src string) bool {
	hasRule, hasTestUtils, hasFeatureMocks := false, false, false
	for _, literal := range stringLiterals(stripComments(src)) {
		switch {
		case literal == "no-restricted-imports":
			hasRule = true
		case strings.Contains(literal, "test-utils"):
			hasTestUtils = true
		case strings.Contains(literal, "features/*/mocks"):
			hasFeatureMocks = true
		}
	}
	return hasRule && hasTestUtils && hasFeatureMocks
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stripComments(src string) string {
	var b strings.Builder
	line, block := false, false
	var quote byte
	escaped := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if line {
			if c == '\n' {
				line = false
				b.WriteByte(c)
			}
			continue
		}
		if block {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				block = false
				i++
			}
			continue
		}
		if quote != 0 {
			b.WriteByte(c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '/' && i+1 < len(src) {
			switch src[i+1] {
			case '/':
				line = true
				i++
				continue
			case '*':
				block = true
				i++
				continue
			}
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
		}
		b.WriteByte(c)
	}
	return b.String()
}

func objectValueBlock(src, key string) (string, bool) {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return "", false
	}
	i = skipSpace(src, i)
	if i >= len(src) || src[i] != '{' {
		return "", false
	}
	end := matchingDelimiter(src, i, '{', '}')
	if end <= i {
		return "", false
	}
	return src[i+1 : end], true
}

func stringArrayPropertyValues(src, key string) []string {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return nil
	}
	i = skipSpace(src, i)
	if i >= len(src) {
		return nil
	}
	if src[i] == '[' {
		end := matchingDelimiter(src, i, '[', ']')
		if end > i {
			return stringLiterals(src[i+1 : end])
		}
	}
	if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
		value, _, ok := readStringLiteral(src, i)
		if ok {
			return []string{value}
		}
	}
	return nil
}

func numericProperty(src, key string) (float64, bool) {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return 0, false
	}
	i = skipSpace(src, i)
	start := i
	for i < len(src) && ((src[i] >= '0' && src[i] <= '9') || src[i] == '.') {
		i++
	}
	if i == start {
		return 0, false
	}
	var value float64
	if _, err := fmt.Sscanf(src[start:i], "%f", &value); err != nil {
		return 0, false
	}
	return value, true
}

func booleanProperty(src, key string) (bool, bool) {
	i, ok := propertyValueIndex(src, key)
	if !ok {
		return false, false
	}
	i = skipSpace(src, i)
	if strings.HasPrefix(src[i:], "true") {
		return true, true
	}
	if strings.HasPrefix(src[i:], "false") {
		return false, true
	}
	return false, false
}

func propertyValueIndex(src, key string) (int, bool) {
	for i := 0; i < len(src); {
		i = skipSpaceAndCommas(src, i)
		if i >= len(src) {
			return 0, false
		}
		name, next, ok := readPropertyName(src, i)
		if !ok {
			i++
			continue
		}
		next = skipSpace(src, next)
		if next >= len(src) || src[next] != ':' {
			i = next
			continue
		}
		if name == key {
			return next + 1, true
		}
		i = next + 1
	}
	return 0, false
}

func readPropertyName(src string, i int) (string, int, bool) {
	if i >= len(src) {
		return "", i, false
	}
	if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
		return readStringLiteral(src, i)
	}
	if !isIdentStart(src[i]) {
		return "", i, false
	}
	start := i
	for i++; i < len(src) && isIdentPart(src[i]); i++ {
	}
	return src[start:i], i, true
}

func stringLiterals(src string) []string {
	var out []string
	for i := 0; i < len(src); i++ {
		if src[i] != '\'' && src[i] != '"' && src[i] != '`' {
			continue
		}
		value, next, ok := readStringLiteral(src, i)
		if ok {
			out = append(out, value)
			i = next - 1
		}
	}
	return out
}

func readStringLiteral(src string, i int) (string, int, bool) {
	if i >= len(src) || (src[i] != '\'' && src[i] != '"' && src[i] != '`') {
		return "", i, false
	}
	quote := src[i]
	var b strings.Builder
	escaped := false
	for j := i + 1; j < len(src); j++ {
		c := src[j]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == quote {
			return b.String(), j + 1, true
		}
		b.WriteByte(c)
	}
	return "", i, false
}

func matchingDelimiter(src string, start int, open, close byte) int {
	if start >= len(src) || src[start] != open {
		return -1
	}
	depth := 0
	var quote byte
	escaped := false
	for i := start; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
		} else if c == open {
			depth++
		} else if c == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func skipSpace(src string, i int) int {
	for i < len(src) && strings.ContainsRune(" \t\n\r", rune(src[i])) {
		i++
	}
	return i
}

func skipSpaceAndCommas(src string, i int) int {
	for i < len(src) && strings.ContainsRune(" \t\n\r,", rune(src[i])) {
		i++
	}
	return i
}

func isIdentStart(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' || c == '$'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9' || c == '-'
}
