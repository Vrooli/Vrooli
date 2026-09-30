// Meta-test enforcing Round 4 Phase 3's "live-HTTP harness as the
// default" rule for handler tests.
//
// Why this exists:
//
// The 2026-04-28 SSE flusher bug shipped because handler tests used
// httptest.ResponseRecorder, which natively implements http.Flusher,
// so the missing Flusher pass-through in the responseWriter wrapper
// was undetectable from `go test ./...`. Every handler test that
// asserts on middleware-relevant behavior (status codes, headers,
// body, streaming) must run through the production middleware stack
// via testutil/httpx.NewLiveServer.
//
// What is allowed:
//
// Pure helper-function unit tests that don't touch the router or
// middleware (e.g., `runtime.EvaluateProtectedGitAllowlist`, file-size
// bounds) may continue to use raw `*testing.T` and direct calls. They
// don't construct httptest.NewRecorder, so this lint never fires for
// them.
//
// What is denied:
//
// Use of `httptest.NewRecorder` (or its alias `httptest.ResponseRecorder`
// literals) in `internal/handlers/*_test.go`. Tests that need an
// http.Handler invocation must go through the live harness; helper
// tests should not depend on httptest at all.

package handlers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// allowlistedRecorderUsages enumerates files that may legitimately
// use httptest.NewRecorder. The list is intentionally empty: every
// handler test must either go through the live-HTTP harness or live
// in a non-handlers package. If a future helper genuinely needs
// recorder-based testing for a pure-function reason, add it here with
// a one-line justification.
var allowlistedRecorderUsages = map[string]string{
	// (intentionally empty — see file header)
}

func TestNoResponseRecorderInHandlerTests(t *testing.T) {
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read handlers dir: %v", err)
	}
	fset := token.NewFileSet()
	var violations []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, "_test.go") {
			continue
		}
		// Skip the meta-test itself; the constants above mention
		// httptest.NewRecorder for documentation purposes.
		if name == filepath.Base(thisFile) {
			continue
		}

		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if ident.Name != "httptest" {
				return true
			}
			// httptest.NewRecorder() is the surrogate that makes
			// recorder-only tests possible. NewRequest is fine —
			// the harness uses it under the hood.
			if sel.Sel.Name == "NewRecorder" {
				if _, allowed := allowlistedRecorderUsages[name]; !allowed {
					pos := fset.Position(call.Pos())
					violations = append(violations,
						pos.String()+": httptest.NewRecorder() is forbidden in handlers tests; use testutil/httpx.NewLiveServer instead")
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Errorf("Round 4 Phase 3 invariant violated — middleware-relevant tests must run through the live HTTP harness:")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

// thisFile is the on-disk name of the meta-test file. Used to skip
// the file itself in the AST walk so a docstring mentioning
// httptest.NewRecorder doesn't trip the lint.
const thisFile = "handler_test_pattern_test.go"
