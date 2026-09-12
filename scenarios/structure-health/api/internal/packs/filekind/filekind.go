// Package filekind centralizes source-file classification for conformance packs.
package filekind

import "strings"

// IsTestSupportFile reports whether a file is test, fixture, or documentation
// support rather than production runtime code. The predicate is language-neutral
// so TypeScript and Go receive the same validation treatment.
func IsTestSupportFile(path string) bool {
	lower := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.HasPrefix(lower, "test/") || strings.HasPrefix(lower, "tests/") ||
		strings.HasPrefix(lower, "docs/") || strings.Contains(lower, "/testdata/") ||
		strings.Contains(lower, "/__tests__/") || strings.Contains(lower, "/test-utils/") ||
		strings.Contains(lower, "/testutil/") || strings.Contains(lower, "/fixtures/") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") || strings.HasSuffix(lower, "_test.go")
}
