package compiler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestCompileWorkflowSequential(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "test-flow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{
					"id":       "node-1",
					"type":     "navigate",
					"data":     map[string]any{"url": "https://example.com"},
					"position": map[string]any{"x": 0, "y": 0},
				},
				map[string]any{
					"id":       "node-2",
					"type":     "wait",
					"data":     map[string]any{"type": "time", "duration": 1000},
					"position": map[string]any{"x": 200, "y": 0},
				},
				map[string]any{
					"id":       "node-3",
					"type":     "screenshot",
					"data":     map[string]any{"name": "after"},
					"position": map[string]any{"x": 400, "y": 0},
				},
			},
			"edges": []any{
				map[string]any{"id": "edge-1", "source": "node-1", "target": "node-2"},
				map[string]any{"id": "edge-2", "source": "node-2", "target": "node-3"},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	if err != nil {
		t.Fatalf("compiled failed: %v", err)
	}

	if len(plan.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(plan.Steps))
	}

	if plan.Steps[0].NodeID != "node-1" || plan.Steps[1].NodeID != "node-2" || plan.Steps[2].NodeID != "node-3" {
		t.Fatalf("unexpected step order: %+v", plan.Steps)
	}

	if plan.Steps[0].Type != StepNavigate {
		t.Fatalf("expected first step to be navigate, got %s", plan.Steps[0].Type)
	}

	if got := plan.Steps[0].OutgoingEdges; len(got) != 1 || got[0].TargetNode != "node-2" {
		t.Fatalf("unexpected outgoing edges: %+v", got)
	}
}

func TestCompileWorkflowDetectsCycles(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "cycle",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "a", "type": "navigate", "data": map[string]any{"url": "https://example.com"}},
				map[string]any{"id": "b", "type": "wait", "data": map[string]any{"type": "time", "duration": 1000}},
			},
			"edges": []any{
				map[string]any{"id": "ab", "source": "a", "target": "b"},
				map[string]any{"id": "ba", "source": "b", "target": "a"},
			},
		},
	}

	if _, err := CompileWorkflow(workflow); err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestCompileWorkflowUnsupportedType(t *testing.T) {
	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "x", "type": "foo", "data": map[string]any{}},
			},
		},
	}

	if _, err := CompileWorkflow(workflow); err == nil {
		t.Fatal("expected unsupported type error, got nil")
	}
}

func TestCompileWorkflowLoopExtractsBody(t *testing.T) {
	workflow := &database.Workflow{
		ID:   uuid.New(),
		Name: "loop-flow",
		FlowDefinition: database.JSONMap{
			"nodes": []any{
				map[string]any{"id": "loop", "type": "loop", "data": map[string]any{"loopType": "forEach", "arraySource": "rows"}},
				map[string]any{"id": "body-click", "type": "click", "data": map[string]any{"selector": "#save"}},
				map[string]any{"id": "body-type", "type": "type", "data": map[string]any{"selector": "#input", "text": "value"}},
				map[string]any{"id": "after", "type": "screenshot", "data": map[string]any{"name": "after"}},
			},
			"edges": []any{
				map[string]any{"id": "loop-body", "source": "loop", "target": "body-click", "sourceHandle": "loopBody"},
				map[string]any{"id": "body-chain", "source": "body-click", "target": "body-type"},
				map[string]any{"id": "loop-continue", "source": "body-type", "target": "loop", "targetHandle": "loopContinue"},
				map[string]any{"id": "loop-after", "source": "loop", "target": "after", "sourceHandle": "loopAfter"},
			},
		},
	}

	plan, err := CompileWorkflow(workflow)
	if err != nil {
		t.Fatalf("expected loop workflow to compile, got error: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected loop plan to include loop node plus after node, got %d steps", len(plan.Steps))
	}
	loopStep := plan.Steps[0]
	if loopStep.Type != StepLoop {
		t.Fatalf("expected first step to be loop, got %s", loopStep.Type)
	}
	if loopStep.LoopPlan == nil {
		t.Fatalf("expected loop plan to include nested plan")
	}
	if len(loopStep.LoopPlan.Steps) != 2 {
		t.Fatalf("expected nested plan to include 2 body nodes, got %d", len(loopStep.LoopPlan.Steps))
	}
	bodyNodes := map[string]struct{}{}
	for _, step := range loopStep.LoopPlan.Steps {
		bodyNodes[step.NodeID] = struct{}{}
	}
	for _, expected := range []string{"body-click", "body-type"} {
		if _, ok := bodyNodes[expected]; !ok {
			t.Fatalf("expected loop body to contain %s", expected)
		}
	}
	if loopStep.LoopPlan.Steps[1].LoopPlan != nil {
		t.Fatalf("unexpected nested loop plan for non-loop body node")
	}
}
