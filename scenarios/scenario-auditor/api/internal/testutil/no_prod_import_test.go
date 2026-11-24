package testutil_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProductionImports keeps test-only helpers out of production imports.
// This policy is declared in .vrooli/testing.json and must remain executable,
// rather than relying on reviewers to notice a misplaced import.
func TestNoProductionImports(t *testing.T) {
	module := readModuleName(t)
	prefix := module + "/internal/testutil"
	root := "../"
	fset := token.NewFileSet()
	var violations []string

	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == "testutil" || name == "vendor" || name == "generated" || name == "mocks" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
				violations = append(violations, path+" imports "+importPath)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk production Go files: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("production code must not import %s or its subpackages", prefix)
		for _, violation := range violations {
			t.Errorf("  %s", violation)
		}
	}
}

func readModuleName(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if module, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(module)
		}
	}
	t.Fatal("module directive not found in go.mod")
	return ""
}
