package testutil_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPersistedShapesUseTypedWriters fails when a test writes a raw string
// literal to a path ending in .json.
//
// A JSON string literal in a test is a second, untyped copy of a persisted
// schema. It keeps compiling after the real struct changes, so the test goes on
// asserting against a shape the production code no longer produces — the drift
// is invisible until something downstream breaks. Encoding through the
// production struct makes a field rename a compile error instead.
//
// The check is deliberately narrow: it looks for os.WriteFile calls whose
// destination is a .json path and whose content is a string literal. Building
// JSON as a value and marshalling it is fine, which is the pattern this pushes
// toward.
func TestPersistedShapesUseTypedWriters(t *testing.T) {
	apiRoot := repoAPIRoot(t)

	// Packages still carrying literal fixtures. Shrinking this list is the
	// point; adding to it needs a reason recorded here.
	allowed := map[string]string{
		// This fixture is malformed on purpose — it writes truncated JSON to
		// exercise the loader's parse-error path. A typed writer cannot
		// produce invalid JSON, which is exactly why it is exempt.
		"internal/store/action_store_test.go": "writes deliberately malformed JSON to test the parse-error path",
		// Same reason: por_manifest_invalid exists for manifests the parser
		// rejects, and a typed writer cannot produce one.
		"internal/memberflow/plan_of_record_rules_behavior_test.go": "writes deliberately truncated JSON to test por_manifest_invalid",
	}

	err := filepath.WalkDir(apiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		if reason, ok := allowed[rel]; ok {
			t.Logf("skipping %s: %s", rel, reason)
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "WriteFile" {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return true
			}
			if !destinationIsJSON(call.Args[0]) {
				return true
			}
			if !contentIsStringLiteral(call.Args[1]) {
				return true
			}
			t.Errorf("%s:%d writes a raw string literal to a .json path; encode through the production struct instead (see api/TESTING_GUIDE.md, Typed writer)",
				rel, fset.Position(call.Pos()).Line)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk api tree: %v", err)
	}
}

// destinationIsJSON reports whether any string literal in the destination
// expression ends in .json, which covers both a bare path and a filepath.Join.
func destinationIsJSON(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if ok && lit.Kind == token.STRING && strings.HasSuffix(strings.Trim(lit.Value, "`\""), ".json") {
			found = true
		}
		return !found
	})
	return found
}

// contentIsStringLiteral reports whether the written bytes come from a literal
// rather than from a marshalled value.
func contentIsStringLiteral(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	// []byte(...) conversion
	arrayType, ok := call.Fun.(*ast.ArrayType)
	if !ok {
		return false
	}
	if ident, ok := arrayType.Elt.(*ast.Ident); !ok || ident.Name != "byte" {
		return false
	}
	switch inner := call.Args[0].(type) {
	case *ast.BasicLit:
		return inner.Kind == token.STRING
	case *ast.BinaryExpr:
		// Concatenated literals are still literals.
		return true
	}
	return false
}

func repoAPIRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	// internal/testutil -> internal -> api
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
