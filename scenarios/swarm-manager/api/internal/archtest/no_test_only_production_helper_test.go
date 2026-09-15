package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReviewHasNoTestOnlyProductionHelper prevents the review domain from
// retaining a production helper after its real workflow caller disappears.
// The prior buildReviewAttachments helper was only called by tests, which
// silently dropped plan content from the actual review input.
func TestReviewHasNoTestOnlyProductionHelper(t *testing.T) {
	const reviewDir = "../review"
	entries, err := os.ReadDir(reviewDir)
	if err != nil {
		t.Fatalf("read review package: %v", err)
	}

	productionCalls := map[string]int{}
	testCalls := map[string]int{}
	privateFunctions := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(reviewDir, entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		isTest := strings.HasSuffix(entry.Name(), "_test.go")
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.IsExported() {
				continue
			}
			if !isTest {
				privateFunctions[fn.Name.Name] = struct{}{}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if isTest {
				testCalls[ident.Name]++
			} else {
				productionCalls[ident.Name]++
			}
			return true
		})
	}

	for name := range privateFunctions {
		if testCalls[name] > 0 && productionCalls[name] == 0 {
			t.Errorf("review helper %q has callers only in test files; delete it or restore a production caller", name)
		}
	}
}
