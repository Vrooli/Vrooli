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

// TestPlanExecutionStartsOnlyThroughTransitionRunner guards the programmatic
// boundary: execution may construct its immutable domain snapshot, but the
// shared runner alone owns workflow transport and correlation lifecycle.
func TestPlanExecutionStartsOnlyThroughTransitionRunner(t *testing.T) {
	path := filepath.Join("..", "execution", "service_control.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var body string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "startPlanOperationLocked" {
			continue
		}
		body = string(source[fset.Position(fn.Pos()).Offset:fset.Position(fn.End()).Offset])
	}
	if body == "" {
		t.Fatal("startPlanOperationLocked not found")
	}
	// StartWith, not StartPrepared: plan execution must build its snapshot through
	// the registered input builder so the same projection is used at start and at
	// the apply-time rebuild that detects mid-run plan edits.
	if !strings.Contains(body, `StartWith(ctx, "plan.execute"`) || strings.Contains(body, ".StartWorkflow(") {
		t.Fatal("plan execution must start through the shared transition runner's registered input builder")
	}
	if strings.Contains(body, "StartPrepared(") {
		t.Fatal("plan execution must not pass a pre-built input; that path bypasses the registered builder")
	}
	for _, forbidden := range []string{".StartOperation(", ".CreateRun(", ".ContinueRun(", "SpawnBacklog(", "SpawnResearch("} {
		if strings.Contains(body, forbidden) {
			t.Errorf("plan execution contains forbidden launch call %q", forbidden)
		}
	}
}
