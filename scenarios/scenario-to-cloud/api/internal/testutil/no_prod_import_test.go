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
	module := moduleName(t, filepath.Join("..", "..", "go.mod"))
	prefix := module + "/internal/testutil"
	err := filepath.WalkDir(filepath.Join("..", ".."), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "testutil" {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, importSpec := range file.Imports {
			if strings.HasPrefix(strings.Trim(importSpec.Path.Value, `"`), prefix) {
				t.Errorf("production file %s imports %s", path, importSpec.Path.Value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func moduleName(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if module, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return module
		}
	}
	t.Fatal("module directive not found")
	return ""
}
