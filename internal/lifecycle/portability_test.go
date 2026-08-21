package lifecycle

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLifecycleExecutorHasNoShellOrProcessTablePath(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bBashCommand\b`),
		regexp.MustCompile(`(?i)exec\.Command(?:Context)?\([^\n]*(?:"bash"|"sh"\s*,\s*"-c")`),
		regexp.MustCompile(`(?i)exec\.Command(?:Context)?\([^\n]*"pkill"`),
		regexp.MustCompile(`(?i)exec\.Command(?:Context)?\([^\n]*"ps"`),
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Clean(name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, pattern := range patterns {
			if pattern.Match(raw) {
				t.Errorf("%s reintroduced forbidden executor path %s", name, pattern)
			}
		}
	}
}
