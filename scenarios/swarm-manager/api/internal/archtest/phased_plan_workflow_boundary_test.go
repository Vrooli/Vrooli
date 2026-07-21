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

// TestPlanExecutionStartsOnlyThroughDeclaredWorkflow guards the programmatic
// boundary: execution may construct its immutable domain snapshot, but starts
// it only through Agent Manager's generic declared-workflow seam.
func TestPlanExecutionStartsOnlyThroughPhasedPlanWorkflow(t *testing.T) {
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
	if !strings.Contains(body, `resolveWorkflow("plan.execute")`) || !strings.Contains(body, ".StartWorkflow(") || !strings.Contains(body, "WorkflowKey: workflow.Key") {
		t.Fatal("plan execution no longer resolves and starts the declared phased-plan workflow through the generic seam")
	}
	for _, forbidden := range []string{".StartOperation(", ".CreateRun(", ".ContinueRun(", "SpawnBacklog(", "SpawnResearch("} {
		if strings.Contains(body, forbidden) {
			t.Errorf("plan execution contains forbidden launch call %q", forbidden)
		}
	}
}
