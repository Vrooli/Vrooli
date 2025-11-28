package compiler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestCompileWorkflowWithInlineSubflow(t *testing.T) {
	// Minimal workflow with an inline subflow definition
	workflowDef := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "test-subflow",
				"type": "subflow",
				"data": map[string]any{
					"label": "Test subflow",
					"workflowDefinition": map[string]any{
						"nodes": []any{
							map[string]any{
								"id":   "nav",
								"type": "navigate",
								"data": map[string]any{
									"label": "Navigate to example.com",
									"url":   "https://example.com",
								},
							},
						},
						"edges": []any{},
					},
				},
			},
		},
		"edges": []any{},
	}

	workflow := &database.Workflow{
		ID:             uuid.New(),
		Name:           "Test Subflow Workflow",
		FlowDefinition: database.JSONMap(workflowDef),
	}

	plan, err := CompileWorkflow(workflow)
	if err != nil{
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Expected plan to be non-nil")
	}

	// The subflow should be inlined, so we expect 1 step (the navigate node)
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step after inlining subflow, got %d", len(plan.Steps))
		t.Logf("Steps: %+v", plan.Steps)
	}

	if len(plan.Steps) > 0 {
		if plan.Steps[0].Type != StepNavigate {
			t.Errorf("Expected first step to be navigate, got %v", plan.Steps[0].Type)
		}
		if url, ok := plan.Steps[0].Params["url"].(string); !ok || url != "https://example.com" {
			t.Errorf("Expected navigate URL to be https://example.com, got %v", plan.Steps[0].Params["url"])
		}
	}
}
