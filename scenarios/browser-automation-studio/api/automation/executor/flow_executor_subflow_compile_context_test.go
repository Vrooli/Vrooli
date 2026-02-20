package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func TestSelectorResolutionRequiresProjectRootContext(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	manifestPath := filepath.Join(projectRoot, "ui", "src", "consts", "selectors.manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	manifest := []byte(`{"selectors":{"workspace.newTerminalButton":{"selector":"[data-testid=\"new-terminal-button\"]"}}}`)
	if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	workflow := &basapi.WorkflowSummary{
		Id:             uuid.NewString(),
		Name:           "selector-resolution",
		FlowDefinition: waitSelectorFlow("@selector/workspace.newTerminalButton"),
	}

	_, noRootInstructions, err := BuildContractsPlanWithCompiler(context.Background(), uuid.New(), workflow, DefaultPlanCompiler)
	if err != nil {
		t.Fatalf("compile without project root: %v", err)
	}
	noRootSelector := instructionSelector(noRootInstructions[0])
	if noRootSelector != "@selector/workspace.newTerminalButton" {
		t.Fatalf("expected unresolved selector without project root, got %q", noRootSelector)
	}

	ctxWithRoot := autocompiler.WithProjectRoot(context.Background(), projectRoot)
	_, withRootInstructions, err := BuildContractsPlanWithCompiler(ctxWithRoot, uuid.New(), workflow, DefaultPlanCompiler)
	if err != nil {
		t.Fatalf("compile with project root: %v", err)
	}
	withRootSelector := instructionSelector(withRootInstructions[0])
	if withRootSelector != "[data-testid=\"new-terminal-button\"]" {
		t.Fatalf("expected resolved selector with project root, got %q", withRootSelector)
	}
}

func waitSelectorFlow(selector string) *basworkflows.WorkflowDefinitionV2 {
	state := basactions.WaitState_WAIT_STATE_VISIBLE
	return &basworkflows.WorkflowDefinitionV2{
		Nodes: []*basworkflows.WorkflowNodeV2{
			{
				Id: "wait",
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_WAIT,
					Params: &basactions.ActionDefinition_Wait{
						Wait: &basactions.WaitParams{
							WaitFor: &basactions.WaitParams_Selector{Selector: selector},
							State:   &state,
						},
					},
				},
			},
		},
		Edges: []*basworkflows.WorkflowEdgeV2{},
	}
}

func instructionSelector(instr contracts.CompiledInstruction) string {
	if instr.Action == nil {
		return ""
	}
	wait := instr.Action.GetWait()
	if wait == nil {
		return ""
	}
	return wait.GetSelector()
}
