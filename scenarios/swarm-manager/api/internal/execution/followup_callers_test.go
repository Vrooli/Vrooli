package execution

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestFollowUp_OnlyCalledFromHandlerLifecycle pins that Service.FollowUp has
// exactly one caller: the HTTP handler in handler_lifecycle.go. If the
// execution system ever gains an auto-FollowUp path (e.g. from finalization
// or polling), this test fails and forces a deliberate review — the W1
// plan routes post-run state through in_review / review_pending so the
// user decides whether a follow-up is warranted, not the agent.
//
// See the FollowUp doc comment in followup.go for the semantic distinction
// from backlog.StatusNeedsFollowup (item-level terminal).
func TestFollowUp_OnlyCalledFromHandlerLifecycle(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	fset := token.NewFileSet()
	callers := make(map[string]int) // file → count
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		file, err := parser.ParseFile(fset, e.Name(), src, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "FollowUp" {
				return true
			}
			// Skip calls inside the FollowUp method itself (e.g., recursive
			// references via types — none today, but don't let future
			// refactors confuse the counter).
			// We treat any reference to `.FollowUp(` outside its own
			// declaration as a call.
			callers[e.Name()]++
			return true
		})
	}

	// Drop the declaration site. FollowUp is declared in followup.go; the
	// AST walk doesn't count declarations as call expressions, but any
	// test fixture or doc example would. Whitelist explicitly.
	allowed := map[string]bool{
		"handler_lifecycle.go": true,
	}
	for file := range callers {
		if !allowed[file] {
			t.Errorf("unexpected caller of Service.FollowUp in %s (count=%d). "+
				"The W1 plan forbids auto-FollowUp — only the user-invoked "+
				"handler_lifecycle.FollowUp should call it. If you're adding "+
				"a sanctioned caller, update the allowlist and the plan.",
				file, callers[file])
		}
	}
	if callers["handler_lifecycle.go"] == 0 {
		t.Fatalf("expected handler_lifecycle.go to call Service.FollowUp; found none")
	}
}
