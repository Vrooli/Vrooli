package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

type stubPlanCompiler struct {
	called       bool
	plan         contracts.ExecutionPlan
	instructions []contracts.CompiledInstruction
	err          error
	execID       uuid.UUID
	workflowID   uuid.UUID
}

func (s *stubPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	s.called = true
	s.execID = executionID
	if workflow != nil {
		s.workflowID = workflow.ID
	}
	return s.plan, s.instructions, s.err
}

func TestBuildContractsPlanWithCustomCompiler(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	workflow := &database.Workflow{ID: wfID}

	expectedPlan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		WorkflowID:     wfID,
		CreatedAt:      time.Now().UTC(),
	}
	expectedInstructions := []contracts.CompiledInstruction{
		{Index: 0, NodeID: "one", Type: "noop"},
	}

	compiler := &stubPlanCompiler{
		plan:         expectedPlan,
		instructions: expectedInstructions,
	}

	plan, instructions, err := BuildContractsPlanWithCompiler(context.Background(), execID, workflow, compiler)
	if err != nil {
		t.Fatalf("BuildContractsPlanWithCompiler returned error: %v", err)
	}

	if !compiler.called {
		t.Fatalf("expected custom compiler to be invoked")
	}
	if compiler.execID != execID {
		t.Fatalf("expected compiler to receive execID %s, got %s", execID, compiler.execID)
	}
	if compiler.workflowID != wfID {
		t.Fatalf("expected compiler to receive workflowID %s, got %s", wfID, compiler.workflowID)
	}

	if plan.ExecutionID != expectedPlan.ExecutionID || plan.WorkflowID != expectedPlan.WorkflowID {
		t.Fatalf("unexpected plan IDs: %+v", plan)
	}
	if len(instructions) != len(expectedInstructions) {
		t.Fatalf("expected %d instructions, got %d", len(expectedInstructions), len(instructions))
	}
	if instructions[0].NodeID != expectedInstructions[0].NodeID || instructions[0].Type != expectedInstructions[0].Type {
		t.Fatalf("unexpected instruction: %+v", instructions[0])
	}
}

func TestPlanCompilerDefaultsToContractCompiler(t *testing.T) {
	comp := PlanCompilerForEngine("browserless")
	if _, ok := comp.(*ContractPlanCompiler); !ok {
		t.Fatalf("expected default contract compiler, got %T", comp)
	}
}

func TestPlanCompilerEnvOverrideRuntime(t *testing.T) {
	t.Setenv("BAS_PLAN_COMPILER", "legacy")
	comp := PlanCompilerForEngine("browserless")
	if _, ok := comp.(*BrowserlessPlanCompiler); !ok {
		t.Fatalf("expected legacy runtime compiler from env override, got %T", comp)
	}
}
