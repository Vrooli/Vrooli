package testutil_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProductionImports prevents CLI production packages from depending on
// test-only helpers.
func TestNoProductionImports(t *testing.T) {
	const root = "../.."
	var violations []string
	fset := token.NewFileSet()
	var walk func(string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				if entry.Name() != "testutil" {
					walk(path)
				}
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Errorf("parse %s: %v", path, err)
				continue
			}
			for _, imp := range file.Imports {
				if strings.Contains(strings.Trim(imp.Path.Value, `"`), "/internal/testutil") {
					violations = append(violations, path+" imports "+imp.Path.Value)
				}
			}
		}
	}
	walk(root)
	for _, violation := range violations {
		t.Errorf("production code must not import testutil: %s", violation)
	}
}
