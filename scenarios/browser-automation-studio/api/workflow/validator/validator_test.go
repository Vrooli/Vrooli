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
			"requirement": "BAS-TEST-001",
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
	if !res.Stats.HasRequirement {
		t.Fatalf("expected requirement metadata to be detected")
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
			"requirement": "BAS-TEST-ERR",
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
