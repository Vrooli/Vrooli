package cliapptest

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AssertNoProductionImports scans root for non-test Go files that import the
// current module's forbiddenSuffix. The module name is read from root/go.mod.
func AssertNoProductionImports(t testing.TB, root, forbiddenSuffix string) {
	t.Helper()
	module, err := moduleName(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read module name: %v", err)
	}
	violations, err := productionImportViolations(root, module+forbiddenSuffix)
	if err != nil {
		t.Fatalf("scan production imports: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("production code must not import %s/...", module+forbiddenSuffix)
		for _, violation := range violations {
			t.Errorf("  %s", violation)
		}
	}
}

func moduleName(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if name, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(name), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", os.ErrNotExist
}

func productionImportViolations(root, forbiddenPrefix string) ([]string, error) {
	fset := token.NewFileSet()
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == "testutil" || entry.Name() == "vendor") {
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
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			if strings.HasPrefix(strings.Trim(imp.Path.Value, `"`), forbiddenPrefix) {
				violations = append(violations, rel)
			}
		}
		return nil
	})
	return violations, err
}
