package testutil_test

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProductionImports keeps CLI test helpers out of the production binary.
// The module path is read from go.mod so the guard remains valid after a
// scenario is rendered from the same template.
func TestNoProductionImports(t *testing.T) {
	module := readModuleName(t)
	prefix := module + "/internal/testutil"
	root := "../.."
	fset := token.NewFileSet()
	var violations []string

	walkGoFiles(t, root, func(path string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("relative path for %s: %v", path, err)
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
			if strings.HasPrefix(importPath, prefix) {
				violations = append(violations, filepath.ToSlash(rel)+" imports "+importPath)
			}
		}
	})

	if len(violations) > 0 {
		t.Errorf("production code must not import %s/...", prefix)
		for _, violation := range violations {
			t.Errorf("  %s", violation)
		}
	}
}

func readModuleName(t *testing.T) string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("open go.mod: %v", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if module, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(module)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan go.mod: %v", err)
	}
	t.Fatal("module directive not found in go.mod")
	return ""
}

func walkGoFiles(t *testing.T, root string, visit func(string)) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read directory %s: %v", root, err)
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				continue
			}
			walkGoFiles(t, path, visit)
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			visit(path)
		}
	}
}
