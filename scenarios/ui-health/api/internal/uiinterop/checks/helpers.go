package checks

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// isTestFile returns true for test files (*.test.ts, *.spec.tsx, etc.) and
// test-harness setup files (test-setup.ts, *.setup.ts, setup-tests.ts) that
// should be excluded from production-code scans.
func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	ext := filepath.Ext(lower)
	base := strings.TrimSuffix(lower, ext)
	if strings.HasSuffix(base, ".test") || strings.HasSuffix(base, ".spec") ||
		strings.HasSuffix(base, "_test") || strings.HasSuffix(base, "_spec") {
		return true
	}
	// Test-harness setup files (Vitest/Jest/CRA conventions) are test
	// infrastructure, not production code: vitest.setup.ts, jest.setup.ts,
	// test-setup.ts, setup-tests.ts, setupTests.ts.
	if strings.HasSuffix(base, ".setup") || strings.HasSuffix(base, "-setup") {
		return true
	}
	switch base {
	case "test-setup", "setup-tests", "setuptests", "test-utils", "testutils":
		return true
	}
	return false
}

// skipDirectories are directories to skip during file scanning.
var skipDirectories = map[string]struct{}{
	".git":          {},
	".hg":           {},
	".svn":          {},
	".cache":        {},
	".next":         {},
	".nuxt":         {},
	"dist":          {},
	"build":         {},
	"node_modules":  {},
	"vendor":        {},
	".venv":         {},
	".idea":         {},
	".vscode":       {},
	"coverage":      {},
	"tmp":           {},
	"__tests__":     {},
	"__mocks__":     {},
	"__fixtures__":  {},
	"__snapshots__": {},
}

// scanExtensions are file extensions to scan in source directories.
var scanExtensions = map[string]struct{}{
	".cjs":    {},
	".css":    {},
	".go":     {},
	".htm":    {},
	".html":   {},
	".js":     {},
	".jsx":    {},
	".less":   {},
	".mjs":    {},
	".sass":   {},
	".scss":   {},
	".svelte": {},
	".ts":     {},
	".tsx":    {},
	".vue":    {},
}

// mainEntryNames lists possible UI entry file names in priority order.
var mainEntryNames = []string{"main.tsx", "main.ts", "main.jsx", "main.js", "index.tsx", "index.ts"}

// serverFileNames lists possible UI server file names.
var serverFileNames = []string{"server.js", "server.ts", "server.mjs", "server.cjs"}

// findMainEntry locates the UI main entry file and returns its content,
// relative path from scenarioRoot, and the absolute path.
// Returns ("", "", nil, error) if not found.
func findMainEntry(scenarioRoot string) (content string, relPath string, data []byte, err error) {
	for _, name := range mainEntryNames {
		p := filepath.Join(scenarioRoot, "ui", "src", name)
		d, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		return string(d), "ui/src/" + name, d, nil
	}
	return "", "", nil, os.ErrNotExist
}

// lineOf returns the 1-based line number of needle in content, or 0 if not found.
func lineOf(content, needle string) int {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if strings.Contains(scanner.Text(), needle) {
			return lineNum
		}
	}
	return 0
}
