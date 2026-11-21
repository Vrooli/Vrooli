// Package testguard provides source-tree checks shared by scenario test suites.
package testguard

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ImportViolation identifies production source that depends on a forbidden
// import prefix.
type ImportViolation struct {
	File   string
	Import string
}

// AssertNoProductionTestutilImports fails tb when production files under
// apiRoot import that API module's internal/testutil package.
func AssertNoProductionTestutilImports(tb testing.TB, apiRoot string) {
	tb.Helper()
	violations, err := ProductionTestutilImports(apiRoot)
	if err != nil {
		tb.Fatal(err)
	}
	if len(violations) == 0 {
		return
	}
	tb.Errorf("production code must not import %s/internal/testutil", modulePrefix(apiRoot))
	for _, violation := range violations {
		tb.Errorf("  %s imports %s", violation.File, violation.Import)
	}
}

// ProductionTestutilImports finds non-test Go files that import this API
// module's internal/testutil tree. It scans the entire API root (including
// handlers and commands), not only internal/, and excludes test-support,
// generated and vendored code that is intentionally outside production.
func ProductionTestutilImports(apiRoot string) ([]ImportViolation, error) {
	root, err := filepath.Abs(apiRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve API root: %w", err)
	}
	modulePath, err := modulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	prefix := modulePath + "/internal/testutil"
	files := token.NewFileSet()
	var violations []ImportViolation
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && excludedDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(files, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("make %s relative to API root: %w", path, err)
		}
		for _, imported := range file.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return fmt.Errorf("decode import path in %s: %w", relative, err)
			}
			if name == prefix || strings.HasPrefix(name, prefix+"/") {
				violations = append(violations, ImportViolation{File: filepath.ToSlash(relative), Import: name})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk API root %s: %w", root, err)
	}
	return violations, nil
}

// AssertNoForbiddenReferences scans production files directly inside
// sourceDir. A key without a dot bans an import path; a qualified key such as
// time.Now bans only that member, allowing types such as time.Duration. Member
// checks resolve aliases through import declarations.
func AssertNoForbiddenReferences(tb testing.TB, sourceDir string, forbidden map[string]string) {
	tb.Helper()
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		tb.Fatalf("read source directory %s: %v", sourceDir, err)
	}
	fset := token.NewFileSet()
	var violations []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filename := filepath.Join(sourceDir, entry.Name())
		file, err := parser.ParseFile(fset, filename, nil, 0)
		if err != nil {
			tb.Errorf("parse %s: %v", filename, err)
			continue
		}
		imports := make(map[string]string, len(file.Imports))
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				tb.Errorf("decode import path in %s: %v", filename, err)
				continue
			}
			localName := path.Base(importPath)
			if imported.Name != nil {
				localName = imported.Name.Name
			}
			imports[localName] = importPath
			if reason, blocked := forbidden[importPath]; blocked {
				violations = append(violations, fmt.Sprintf("%s imports %s (%s)", entry.Name(), importPath, reason))
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			packageName, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			importPath, ok := imports[packageName.Name]
			if !ok {
				return true
			}
			member := importPath + "." + selector.Sel.Name
			if reason, blocked := forbidden[member]; blocked {
				violations = append(violations, fmt.Sprintf("%s references %s (%s)", entry.Name(), member, reason))
			}
			return true
		})
	}
	if len(violations) == 0 {
		return
	}
	tb.Errorf("source directory %s imports forbidden packages or references forbidden members:", sourceDir)
	for _, violation := range violations {
		tb.Errorf("  %s", violation)
	}
}

func modulePath(goMod string) (string, error) {
	file, err := os.Open(goMod)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", goMod, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if value, ok := strings.CutPrefix(line, "module "); ok {
			if module := strings.TrimSpace(value); module != "" {
				return module, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan %s: %w", goMod, err)
	}
	return "", fmt.Errorf("module directive not found in %s", goMod)
}

func modulePrefix(apiRoot string) string {
	root, err := filepath.Abs(apiRoot)
	if err != nil {
		return "<unknown-module>"
	}
	module, err := modulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return "<unknown-module>"
	}
	return module
}

func excludedDirectory(name string) bool {
	switch name {
	case ".git", "build", "dist", "generated", "mocks", "node_modules", "testutil", "vendor":
		return true
	default:
		return false
	}
}
