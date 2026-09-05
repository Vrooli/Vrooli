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
	root := "../../"
	fset := token.NewFileSet()
	violations := []string{}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			if rel == filepath.Join("internal", "testutil") {
				return filepath.SkipDir
			}
			if entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Errorf("parse %s: %v", path, parseErr)
			return nil
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, "prompt-manager/internal/testutil") {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				violations = append(violations, rel+" imports "+importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production files: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("production code must not import internal/testutil/...")
		for _, violation := range violations {
			t.Errorf("  %s", violation)
		}
	}
}
