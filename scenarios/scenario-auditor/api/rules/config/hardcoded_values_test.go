//go:build ruletests
// +build ruletests

package config

import "testing"

func TestHardcodedValuesDocCases(t *testing.T) {
	runDocTestsViolations(t, "hardcoded_values.go", "api/main.go", func(content []byte, path string) []Violation {
		return CheckHardcodedValues(content, path)
	})
}

func TestHardcodedValuesSkipsLockfiles(t *testing.T) {
	t.Helper()
	sample := []byte(`        "url": "https://example.com"`)
	paths := []string{
		"package-lock.json",
		"ui/pnpm-lock.yaml",
		"cli/vendor/poetry.lock",
	}

	for _, path := range paths {
		if violations := CheckHardcodedValues(sample, path); len(violations) != 0 {
			t.Fatalf("expected no violations for %s, got %d", path, len(violations))
		}
	}
}
