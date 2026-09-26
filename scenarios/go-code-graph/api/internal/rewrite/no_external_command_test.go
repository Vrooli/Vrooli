package rewrite

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoExternalCommandInvocation enforces OT-P0-003's "never invoke
// git, never invoke go build" invariant at the source level. It walks
// every non-test .go file in this package and asserts:
//
//  1. No file imports "os/exec".
//  2. No file references "exec.Command" or "exec.CommandContext".
//
// If a future change needs to shell out for any reason, the substrate
// is wrong — fix the substrate, not the test. See
// feedback_design_to_ideal_fix_substrate.
func TestNoExternalCommandInvocation(t *testing.T) {
	root := "."
	fset := token.NewFileSet()
	violations := []string{}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(root, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			continue
		}
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if ip == "os/exec" {
				violations = append(violations, path+" imports os/exec")
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if ident.Name == "exec" && (sel.Sel.Name == "Command" || sel.Sel.Name == "CommandContext") {
				violations = append(violations, path+" references exec."+sel.Sel.Name)
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Errorf("rewrite package must never invoke git/go subprocess:")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}
