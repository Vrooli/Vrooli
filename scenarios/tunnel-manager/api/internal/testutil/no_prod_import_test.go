package testutil_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoProductionImports(t *testing.T) {
	root := "../"
	prefix := readModuleName(t) + "/internal/testutil"
	fset := token.NewFileSet()
	violations := []string{}

	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filepath.Base(path) == "testutil" {
				return filepath.SkipDir
			}
			return nil
		}
		rel := strings.TrimPrefix(path, root)
		if shouldSkipProductionImportFile(rel) {
			return nil
		}
		violations = append(violations, productionImportViolations(t, fset, path, rel, prefix)...)
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	if len(violations) == 0 {
		return
	}
	t.Fatalf("production code must not import %s/...\n%s", prefix, strings.Join(violations, "\n"))
}

func readModuleName(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	t.Fatal("module directive not found in go.mod")
	return ""
}

func shouldSkipProductionImportFile(rel string) bool {
	return !strings.HasSuffix(rel, ".go") ||
		strings.HasSuffix(rel, "_test.go") ||
		strings.Contains(rel, "/vendor/") ||
		pathContainsDir(rel, "mocks") ||
		pathContainsDir(rel, "generated")
}

func productionImportViolations(t *testing.T, fset *token.FileSet, path, rel, prefix string) []string {
	t.Helper()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Errorf("parse %s: %v", path, err)
		return nil
	}
	violations := []string{}
	for _, imp := range file.Imports {
		ip := strings.Trim(imp.Path.Value, `"`)
		if strings.HasPrefix(ip, prefix) {
			violations = append(violations, rel+" imports "+ip)
		}
	}
	return violations
}

func pathContainsDir(rel, name string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if p == name {
			return true
		}
	}
	return false
}
