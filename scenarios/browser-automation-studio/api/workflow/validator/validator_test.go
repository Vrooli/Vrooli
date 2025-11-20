package validator

import (
	"context"
	"testing"
)

func TestValidatorValidWorkflow(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"metadata": map[string]any{
			"description": "simple happy path",
			"version":     1,
		},
		"nodes": []any{
			map[string]any{
				"id":       "navigate",
				"type":     "navigate",
				"position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{
					"destinationType": "url",
					"url":             "https://example.com",
				},
			},
			map[string]any{
				"id":       "wait",
				"type":     "wait",
				"position": map[string]any{"x": 200, "y": 0},
				"data": map[string]any{
					"waitType":   "duration",
					"durationMs": 500,
				},
			},
		},
		"edges": []any{
			map[string]any{
				"id":     "edge-nav-wait",
				"source": "navigate",
				"target": "wait",
			},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected workflow to be valid, got %+v", res.Errors)
	}

	if res.Stats.NodeCount != 2 || res.Stats.EdgeCount != 1 {
		t.Fatalf("unexpected stats: %+v", res.Stats)
	}
}

func TestValidatorStrictPromotesWarnings(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"metadata": map[string]any{},
		"nodes": []any{
			map[string]any{
				"id":       "wait",
				"type":     "wait",
				"position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{
					"waitType":   "duration",
					"durationMs": 1000,
				},
			},
			map[string]any{
				"id":       "second",
				"type":     "wait",
				"position": map[string]any{"x": 200, "y": 0},
				"data": map[string]any{
					"waitType":   "duration",
					"durationMs": 500,
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{Strict: true})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected strict validation to fail due to warnings")
	}

	foundPromotion := false
	for _, issue := range res.Errors {
		if issue.Code == "WF_STRICT_WARNING" {
			foundPromotion = true
			break
		}
	}
	if !foundPromotion {
		t.Fatalf("expected WF_STRICT_WARNING error, got %+v", res.Errors)
	}
}

func TestValidatorDetectsSchemaErrors(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"metadata": map[string]any{
			"description": "missing nodes",
			"version":     1,
		},
		"nodes": []any{},
		"edges": []any{
			map[string]any{
				"id":     "dangling",
				"source": "nope",
				"target": "missing",
			},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected invalid workflow to fail")
	}

	foundNodeError := false
	for _, issue := range res.Errors {
		if issue.Code == "WF_NODE_EMPTY" {
			foundNodeError = true
			break
		}
	}
	if !foundNodeError {
		t.Fatalf("expected WF_NODE_EMPTY error, got %+v", res.Errors)
	}
}

func TestValidatorEvaluateRequiresExpression(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "eval",
				"type":     "evaluate",
				"position": map[string]any{"x": 0, "y": 0},
				"data":     map[string]any{},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected evaluate node without expression/script to fail")
	}
	found := false
	for _, issue := range res.Errors {
		if issue.Code == "WF_NODE_FIELD_ONE_OF" && issue.NodeID == "eval" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected WF_NODE_FIELD_ONE_OF error for evaluate node, got %+v", res.Errors)
	}
}

func TestValidatorScreenshotRequiresTarget(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "shot",
				"type":     "screenshot",
				"position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{
					"fullPage": false,
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected screenshot without selector/fullPage to fail")
	}
	found := false
	for _, issue := range res.Errors {
		if issue.Code == "WF_SCREENSHOT_TARGET_REQUIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected WF_SCREENSHOT_TARGET_REQUIRED error, got %+v", res.Errors)
	}
}

func TestValidatorAllowsNodesWithoutPosition(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "navigate",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "browser-automation-studio",
					"scenarioPath":    "/",
				},
			},
			map[string]any{
				"id":   "shot",
				"type": "screenshot",
				"data": map[string]any{
					"fullPage": true,
				},
			},
		},
		"edges": []any{
			map[string]any{"id": "edge-1", "source": "navigate", "target": "shot"},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected workflow without explicit positions to validate; errors: %+v", res.Errors)
	}
}

func TestValidatorWorkflowCallRequiresTarget(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "call",
				"type":     "workflowCall",
				"position": map[string]any{"x": 0, "y": 0},
				"data":     map[string]any{},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected workflow call without target to fail")
	}

	found := false
	for _, issue := range res.Errors {
		if issue.Code == "WF_WORKFLOW_CALL_TARGET" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected WF_WORKFLOW_CALL_TARGET error, got %+v", res.Errors)
	}
}

func TestValidatorWorkflowCallInlineDefinition(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "call",
				"type":     "workflowCall",
				"position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{
					"workflowDefinition": map[string]any{
						"nodes": []any{
							map[string]any{
								"id":       "inline-nav",
								"type":     "navigate",
								"position": map[string]any{"x": 0, "y": 0},
								"data": map[string]any{
									"destinationType": "url",
									"url":             "https://example.com",
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

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected inline workflow definition to be valid, got %+v", res.Errors)
	}
}
