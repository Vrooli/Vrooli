package testutil_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoProductionImports(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	var violations []string

	walkGoFiles(t, root, func(path string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("rel path for %s: %v", path, err)
		}
		if strings.HasPrefix(filepath.ToSlash(rel), "internal/testutil/") {
			return
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, "git-control-tower/internal/testutil") {
				violations = append(violations, rel+" imports "+importPath)
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

func walkGoFiles(t *testing.T, root string, fn func(path string)) {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read dir %s: %v", root, err)
	}
	for _, entry := range entries {
		full := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				continue
			default:
				walkGoFiles(t, full, fn)
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			fn(full)
		}
	}
}
