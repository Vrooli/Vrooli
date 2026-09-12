package testutil_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProductionImports keeps test-only helpers out of the CLI binary.
func TestNoProductionImports(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	var violations []string
	walkGoFiles(t, root, func(path string) {
		if strings.HasSuffix(path, "_test.go") || filepath.Base(filepath.Dir(path)) == "testutil" {
			return
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return
		}
		for _, imp := range file.Imports {
			if strings.HasPrefix(strings.Trim(imp.Path.Value, `"`), "agent-manager/cli/internal/testutil") {
				violations = append(violations, path+" imports "+imp.Path.Value)
			}
		}
	})
	if len(violations) > 0 {
		t.Errorf("production code must not import internal/testutil/...")
		for _, violation := range violations {
			t.Errorf("  %s", violation)
		}
	}
}

func walkGoFiles(t *testing.T, root string, fn func(string)) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			if entry.Name() != "testutil" && entry.Name() != "vendor" {
				walkGoFiles(t, path, fn)
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			fn(path)
		}
	}
}
