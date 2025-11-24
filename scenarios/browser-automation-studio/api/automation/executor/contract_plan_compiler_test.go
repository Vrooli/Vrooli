package executor

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestContractPlanCompilerProducesContractsPlan(t *testing.T) {
	execID := uuid.New()
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "contract-plan",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "nav", "type": "navigate", "data": map[string]any{"url": "https://example.com"}, "position": map[string]any{"x": 0, "y": 0}},
				map[string]any{"id": "click", "type": "click", "data": map[string]any{"selector": "#btn"}, "position": map[string]any{"x": 200, "y": 0}},
				map[string]any{"id": "assert", "type": "assert", "data": map[string]any{"selector": "#btn", "text": "Submit"}, "position": map[string]any{"x": 400, "y": 0}},
			},
			"edges": []any{
				map[string]any{"id": "nav-click", "source": "nav", "target": "click"},
				map[string]any{"id": "click-assert", "source": "click", "target": "assert"},
			},
		},
	}

	compiler := &ContractPlanCompiler{}
	plan, instructions, err := compiler.Compile(context.Background(), execID, workflow)
	if err != nil {
		t.Fatalf("compile returned error: %v", err)
	}

	if plan.ExecutionID != execID || plan.WorkflowID != workflow.ID {
		t.Fatalf("unexpected plan IDs: exec=%s workflow=%s", plan.ExecutionID, plan.WorkflowID)
	}
	if plan.SchemaVersion != "automation-plan-v1" || plan.PayloadVersion != "1" {
		t.Fatalf("unexpected schema/payload: %s/%s", plan.SchemaVersion, plan.PayloadVersion)
	}
	if len(instructions) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(instructions))
	}
	if instructions[0].NodeID != "nav" || instructions[1].NodeID != "click" || instructions[2].NodeID != "assert" {
		t.Fatalf("unexpected instruction order: %+v", instructions)
	}
	if plan.Graph == nil || len(plan.Graph.Steps) != 3 {
		t.Fatalf("expected graph with 3 steps, got %+v", plan.Graph)
	}
	if plan.CreatedAt.IsZero() || plan.CreatedAt.After(time.Now().UTC().Add(5*time.Second)) {
		t.Fatalf("unexpected CreatedAt: %s", plan.CreatedAt)
	}
}

func TestPlanCompilerEnvOverride(t *testing.T) {
	t.Setenv("BAS_PLAN_COMPILER", "contract")
	if os.Getenv("BAS_PLAN_COMPILER") == "" {
		t.Skip("env override not supported on this platform")
	}
	comp := PlanCompilerForEngine("browserless")
	if _, ok := comp.(*ContractPlanCompiler); !ok {
		t.Fatalf("expected ContractPlanCompiler from env override, got %T", comp)
	}
}
