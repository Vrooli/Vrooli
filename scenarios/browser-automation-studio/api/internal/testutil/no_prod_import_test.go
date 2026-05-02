package testutil_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testutilImportPath       = "github.com/vrooli/browser-automation-studio/internal/testutil"
	legacyTestutilImportPath = "github.com/vrooli/browser-automation-studio/testutil"
)

// TestNoProductionImports walks every non-test .go file under the API module
// and asserts that none imports internal/testutil/... .
//
// This is the guardrail that keeps shared test infrastructure test-only while
// still allowing it to depend on testing ergonomics, mutable fakes, temp files,
// and process-wide setup that production code must not see.
func TestNoProductionImports(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	violations := []string{}

	walk(t, root, func(path string) {
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return
		}

		rel := strings.TrimPrefix(path, root+string(filepath.Separator))
		if shouldSkip(rel) {
			return
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return
		}

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, testutilImportPath) {
				violations = append(violations, rel+" imports "+importPath)
			}
		}
	})

	if len(violations) == 0 {
		return
	}

	t.Errorf("production code must not import internal/testutil/...")
	for _, violation := range violations {
		t.Errorf("  %s", violation)
	}
}

// TestNoLegacyTestutilImports keeps new tests from depending on the old
// top-level testutil package while its useful helpers move under
// internal/testutil.
func TestNoLegacyTestutilImports(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	violations := []string{}

	walk(t, root, func(path string) {
		if !strings.HasSuffix(path, ".go") {
			return
		}

		rel := strings.TrimPrefix(path, root+string(filepath.Separator))
		if shouldSkipLegacyTestutilScan(rel) {
			return
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return
		}

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == legacyTestutilImportPath {
				violations = append(violations, rel+" imports "+importPath)
			}
		}
	})

	if len(violations) == 0 {
		return
	}

	t.Errorf("new tests should use internal/testutil/... instead of the legacy top-level testutil package")
	for _, violation := range violations {
		t.Errorf("  %s", violation)
	}
}

func shouldSkip(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		switch part {
		case "testutil", "vendor", ".git":
			return true
		}
	}
	return false
}

func shouldSkipLegacyTestutilScan(rel string) bool {
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "testutil/") ||
		strings.HasPrefix(rel, "internal/testutil/") ||
		strings.Contains(rel, "/vendor/") ||
		strings.Contains(rel, "/.git/")
}

func walk(t *testing.T, root string, fn func(path string)) {
	t.Helper()

	for _, entry := range readDir(t, root) {
		full := filepath.Join(root, entry.name)
		if entry.isDir {
			walk(t, full, fn)
			continue
		}
		fn(full)
	}
}
