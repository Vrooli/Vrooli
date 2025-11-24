package validator

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

func TestValidatorSubflowWithWorkflowID(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	childID := uuid.New().String()
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "call",
				"type":     "subflow",
				"position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{
					"label":      "Child flow",
					"workflowId": childID,
					"parameters": map[string]any{"foo": "bar"},
				},
			},
			map[string]any{
				"id":       "done",
				"type":     "navigate",
				"position": map[string]any{"x": 100, "y": 0},
				"data": map[string]any{
					"label":             "Next",
					"destinationType":   "url",
					"url":               "https://example.com",
					"waitForNavigation": true,
				},
			},
		},
		"edges": []any{map[string]any{"id": "edge-1", "source": "call", "target": "done"}},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected subflow with workflowId to validate; errors: %+v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("expected no warnings for subflow, got %+v", res.Warnings)
	}
}

func TestValidatorRejectsWorkflowCallType(t *testing.T) {
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
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected workflowCall to be rejected by schema; warnings=%+v", res.Warnings)
	}
}

func TestValidatorRejectsStoreAsField(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	// Playbook using deprecated "storeAs" instead of "storeResult"
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "evaluate-wrong",
				"type": "evaluate",
				"data": map[string]any{
					"expression": "document.title",
					"storeAs":    "pageTitle", // ❌ Wrong field name
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}

	// Should be invalid due to storeAs field
	if res.Valid {
		t.Fatalf("expected validation to fail for evaluate node with 'storeAs' field")
	}

	// Check that error mentions storeAs
	foundStoreAsError := false
	for _, issue := range res.Errors {
		if issue.Code == "WF_SCHEMA_INVALID" {
			foundStoreAsError = true
			t.Logf("Schema validation error: %s", issue.Message)
		}
	}

	if !foundStoreAsError {
		t.Fatalf("expected schema error for 'storeAs' field, got errors: %+v", res.Errors)
	}
}

func TestValidatorAcceptsStoreResultField(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	// Playbook using correct "storeResult" field
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "evaluate-correct",
				"type": "evaluate",
				"data": map[string]any{
					"expression":  "document.title",
					"storeResult": "pageTitle", // ✅ Correct field name
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
		t.Fatalf("expected workflow with 'storeResult' to be valid, got errors: %+v", res.Errors)
	}
}

func assertIssue(t *testing.T, issues []Issue, code string) bool {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	t.Fatalf("expected issue code %s, got %+v", code, issues)
	return false
}

func assertWarning(warnings []Issue, code string) bool {
	for _, warn := range warnings {
		if warn.Code == code {
			return true
		}
	}
	return false
}
