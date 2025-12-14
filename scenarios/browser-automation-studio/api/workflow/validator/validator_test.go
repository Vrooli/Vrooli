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

func TestValidatorStrictModeDoesNotPromoteStylisticWarnings(t *testing.T) {
	// Strict mode is designed for schema/token-resolution problems only,
	// not stylistic lint warnings (e.g. missing labels, edge sparsity).
	// See shouldPromoteWarningInStrictMode() for the allowlist.
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

	// Workflow should have warnings (missing labels, no edges) but no errors
	if len(res.Warnings) == 0 {
		t.Fatal("expected warnings to be present")
	}

	// Strict mode should NOT promote stylistic warnings to errors
	if !res.Valid {
		t.Fatalf("expected workflow to be valid even in strict mode (stylistic warnings are not promoted), got errors: %+v", res.Errors)
	}

	// Verify no WF_STRICT_WARNING was generated
	for _, issue := range res.Errors {
		if issue.Code == "WF_STRICT_WARNING" {
			t.Fatalf("stylistic warnings should not be promoted in strict mode, but got: %+v", issue)
		}
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

// ============================================================================
// Node Type Validation Tests
// ============================================================================

func TestValidatorClickRequiresSelector(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "click-missing",
				"type":     "click",
				"position": map[string]any{"x": 0, "y": 0},
				"data":     map[string]any{"label": "Click something"},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected click without selector to fail")
	}
	assertIssue(t, res.Errors, "WF_NODE_FIELD_REQUIRED")
}

func TestValidatorHoverRequiresSelector(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "hover-missing",
				"type":     "hover",
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
		t.Fatalf("expected hover without selector to fail")
	}
	assertIssue(t, res.Errors, "WF_NODE_FIELD_REQUIRED")
}

func TestValidatorFocusRequiresSelector(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "focus-missing",
				"type":     "focus",
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
		t.Fatalf("expected focus without selector to fail")
	}
	assertIssue(t, res.Errors, "WF_NODE_FIELD_REQUIRED")
}

func TestValidatorBlurRequiresSelector(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "blur-missing",
				"type": "blur",
				"data": map[string]any{},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected blur without selector to fail")
	}
	assertIssue(t, res.Errors, "WF_NODE_FIELD_REQUIRED")
}

func TestValidatorDragDropValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name          string
		data          map[string]any
		expectValid   bool
		expectedCodes []string
	}{
		{
			name:        "missing both source options and target",
			data:        map[string]any{},
			expectValid: false,
			expectedCodes: []string{
				"WF_NODE_FIELD_ONE_OF",
				"WF_NODE_FIELD_REQUIRED",
			},
		},
		{
			name: "has sourceSelector but missing target",
			data: map[string]any{
				"sourceSelector": "#drag-source",
			},
			expectValid:   false,
			expectedCodes: []string{"WF_NODE_FIELD_REQUIRED"},
		},
		{
			name: "has sourceCoordinates but missing target",
			data: map[string]any{
				"sourceCoordinates": map[string]any{"x": 100, "y": 100},
			},
			expectValid:   false,
			expectedCodes: []string{"WF_NODE_FIELD_REQUIRED"},
		},
		{
			name: "valid with sourceSelector and targetSelector",
			data: map[string]any{
				"sourceSelector": "#drag-source",
				"targetSelector": "#drop-target",
			},
			expectValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "drag",
						"type": "dragDrop",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid != tc.expectValid {
				t.Fatalf("expected valid=%v, got valid=%v, errors=%+v", tc.expectValid, res.Valid, res.Errors)
			}
			for _, code := range tc.expectedCodes {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == code {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", code, res.Errors)
				}
			}
		})
	}
}

func TestValidatorTypeNodeValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectValid bool
		errorCode   string
	}{
		{
			name:        "missing selector and text",
			data:        map[string]any{},
			expectValid: false,
			errorCode:   "WF_NODE_FIELD_REQUIRED",
		},
		{
			name: "has selector but no text/value/variable",
			data: map[string]any{
				"selector": "#input",
			},
			expectValid: false,
			errorCode:   "WF_TYPE_INPUT_REQUIRED",
		},
		{
			name: "valid with selector and text",
			data: map[string]any{
				"selector": "#input",
				"text":     "hello world",
			},
			expectValid: true,
		},
		{
			name: "valid with selector and value",
			data: map[string]any{
				"selector": "#input",
				"value":    "some value",
			},
			expectValid: true,
		},
		{
			name: "valid with selector and variable",
			data: map[string]any{
				"selector": "#input",
				"variable": "myVar",
			},
			expectValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "type-node",
						"type": "type",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid != tc.expectValid {
				t.Fatalf("expected valid=%v, got valid=%v, errors=%+v", tc.expectValid, res.Valid, res.Errors)
			}
			if !tc.expectValid && tc.errorCode != "" {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == tc.errorCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", tc.errorCode, res.Errors)
				}
			}
		})
	}
}

func TestValidatorKeyboardNodeValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectValid bool
		errorCode   string
		warningCode string
	}{
		{
			name:        "missing keys and sequence",
			data:        map[string]any{},
			expectValid: false,
			errorCode:   "WF_KEYBOARD_INPUT_REQUIRED",
		},
		{
			name: "deprecated key field",
			data: map[string]any{
				"key": "Enter",
			},
			expectValid: true,
			warningCode: "WF_KEYBOARD_KEY_FIELD",
		},
		{
			name: "valid with keys array",
			data: map[string]any{
				"keys": []any{"Enter"},
			},
			expectValid: true,
		},
		{
			name: "valid with sequence",
			data: map[string]any{
				"sequence": "Hello",
			},
			expectValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "keyboard-node",
						"type": "keyboard",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid != tc.expectValid {
				t.Fatalf("expected valid=%v, got valid=%v, errors=%+v", tc.expectValid, res.Valid, res.Errors)
			}
			if tc.errorCode != "" {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == tc.errorCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", tc.errorCode, res.Errors)
				}
			}
			if tc.warningCode != "" {
				found := false
				for _, warn := range res.Warnings {
					if warn.Code == tc.warningCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning code %s, not found in %+v", tc.warningCode, res.Warnings)
				}
			}
		})
	}
}

func TestValidatorShortcutRequiresKeys(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "shortcut-missing",
				"type": "shortcut",
				"data": map[string]any{},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected shortcut without keys to fail")
	}
	assertIssue(t, res.Errors, "WF_NODE_FIELD_REQUIRED")
}

func TestValidatorSelectNodeValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectValid bool
		errorCodes  []string
	}{
		{
			name:        "missing everything",
			data:        map[string]any{},
			expectValid: false,
			errorCodes:  []string{"WF_NODE_FIELD_REQUIRED", "WF_NODE_FIELD_ONE_OF"},
		},
		{
			name: "has selector but missing option selection",
			data: map[string]any{
				"selector": "#dropdown",
			},
			expectValid: false,
			errorCodes:  []string{"WF_NODE_FIELD_ONE_OF"},
		},
		{
			name: "valid with selector and optionText",
			data: map[string]any{
				"selector":   "#dropdown",
				"optionText": "Option 1",
			},
			expectValid: true,
		},
		{
			name: "valid with selector and optionValue",
			data: map[string]any{
				"selector":    "#dropdown",
				"optionValue": "opt1",
			},
			expectValid: true,
		},
		{
			name: "valid with selector and optionIndex",
			data: map[string]any{
				"selector":    "#dropdown",
				"optionIndex": "0",
			},
			expectValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "select-node",
						"type": "select",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid != tc.expectValid {
				t.Fatalf("expected valid=%v, got valid=%v, errors=%+v", tc.expectValid, res.Valid, res.Errors)
			}
			for _, code := range tc.errorCodes {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == code {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", code, res.Errors)
				}
			}
		})
	}
}

func TestValidatorWaitNodeValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectValid bool
		errorCode   string
		warningCode string
	}{
		{
			name: "element wait missing selector",
			data: map[string]any{
				"waitType": "element",
			},
			expectValid: false,
			errorCode:   "WF_WAIT_SELECTOR_REQUIRED",
		},
		{
			name: "duration wait with zero duration",
			data: map[string]any{
				"waitType":   "duration",
				"durationMs": 0,
			},
			expectValid: false,
			errorCode:   "WF_WAIT_DURATION_REQUIRED",
		},
		{
			name: "duration wait with negative duration",
			data: map[string]any{
				"waitType":   "duration",
				"durationMs": -100,
			},
			expectValid: false,
			errorCode:   "WF_WAIT_DURATION_REQUIRED",
		},
		{
			name: "duration wait with excessive duration",
			data: map[string]any{
				"waitType":   "duration",
				"durationMs": 120000,
			},
			expectValid: true,
			warningCode: "WF_WAIT_DURATION_LONG",
		},
		{
			name: "unknown wait type",
			data: map[string]any{
				"waitType": "unknown-type",
			},
			// Schema validation rejects unknown wait types before lint warning
			expectValid: false,
			errorCode:   "WF_SCHEMA_INVALID",
		},
		{
			name: "valid element wait",
			data: map[string]any{
				"waitType": "element",
				"selector": "#loading-done",
			},
			expectValid: true,
		},
		{
			name: "valid duration wait",
			data: map[string]any{
				"waitType":   "duration",
				"durationMs": 1000,
			},
			expectValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "wait-node",
						"type": "wait",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid != tc.expectValid {
				t.Fatalf("expected valid=%v, got valid=%v, errors=%+v, warnings=%+v",
					tc.expectValid, res.Valid, res.Errors, res.Warnings)
			}
			if tc.errorCode != "" {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == tc.errorCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", tc.errorCode, res.Errors)
				}
			}
			if tc.warningCode != "" {
				found := false
				for _, warn := range res.Warnings {
					if warn.Code == tc.warningCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning code %s, not found in %+v", tc.warningCode, res.Warnings)
				}
			}
		})
	}
}

func TestValidatorExtractRequiresSelector(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "extract-missing",
				"type": "extract",
				"data": map[string]any{},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected extract without selector to fail")
	}
	assertIssue(t, res.Errors, "WF_NODE_FIELD_REQUIRED")
}

func TestValidatorAssertNodeValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		errorCodes []string
	}{
		{
			name:       "missing everything",
			data:       map[string]any{},
			errorCodes: []string{"WF_NODE_FIELD_REQUIRED"},
		},
		{
			name: "text_equals missing expectedValue",
			data: map[string]any{
				"selector":   "#element",
				"assertMode": "text_equals",
			},
			errorCodes: []string{"WF_ASSERT_EXPECTED_VALUE"},
		},
		{
			name: "attribute_equals missing expectedValue and attributeName",
			data: map[string]any{
				"selector":   "#element",
				"assertMode": "attribute_equals",
			},
			errorCodes: []string{"WF_ASSERT_EXPECTED_VALUE", "WF_ASSERT_ATTRIBUTE_NAME"},
		},
		{
			name: "attribute_contains missing attributeName",
			data: map[string]any{
				"selector":      "#element",
				"assertMode":    "attribute_contains",
				"expectedValue": "some-value",
			},
			errorCodes: []string{"WF_ASSERT_ATTRIBUTE_NAME"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "assert-node",
						"type": "assert",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid {
				t.Fatalf("expected validation to fail, got valid=true")
			}
			for _, code := range tc.errorCodes {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == code {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", code, res.Errors)
				}
			}
		})
	}
}

func TestValidatorNavigateNodeValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectValid bool
		errorCode   string
		warningCode string
	}{
		{
			name: "url type missing url",
			data: map[string]any{
				"destinationType": "url",
			},
			expectValid: false,
			errorCode:   "WF_NAVIGATE_URL_REQUIRED",
		},
		{
			name: "scenario type missing scenario",
			data: map[string]any{
				"destinationType": "scenario",
			},
			expectValid: false,
			errorCode:   "WF_NAVIGATE_SCENARIO_REQUIRED",
		},
		{
			name: "scenario type missing scenarioPath",
			data: map[string]any{
				"destinationType": "scenario",
				"scenario":        "browser-automation-studio",
			},
			expectValid: true,
			warningCode: "WF_NAVIGATE_SCENARIO_PATH",
		},
		{
			name: "unknown destination type",
			data: map[string]any{
				"destinationType": "unknown",
			},
			// Schema validation rejects unknown destination types before lint warning
			expectValid: false,
			errorCode:   "WF_SCHEMA_INVALID",
		},
		{
			name: "valid url navigation",
			data: map[string]any{
				"destinationType": "url",
				"url":             "https://example.com",
			},
			expectValid: true,
		},
		{
			name: "valid scenario navigation",
			data: map[string]any{
				"destinationType": "scenario",
				"scenario":        "browser-automation-studio",
				"scenarioPath":    "/workflows",
			},
			expectValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "navigate-node",
						"type": "navigate",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid != tc.expectValid {
				t.Fatalf("expected valid=%v, got valid=%v, errors=%+v", tc.expectValid, res.Valid, res.Errors)
			}
			if tc.errorCode != "" {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == tc.errorCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", tc.errorCode, res.Errors)
				}
			}
			if tc.warningCode != "" {
				found := false
				for _, warn := range res.Warnings {
					if warn.Code == tc.warningCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning code %s, not found in %+v", tc.warningCode, res.Warnings)
				}
			}
		})
	}
}

func TestValidatorLoopRequiresLoopType(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "loop-missing",
				"type": "loop",
				"data": map[string]any{},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected loop without loopType to fail")
	}
	assertIssue(t, res.Errors, "WF_NODE_FIELD_REQUIRED")
}

func TestValidatorSubflowNodeValidation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		errorCodes []string
	}{
		{
			name:       "missing workflowId and workflowDefinition",
			data:       map[string]any{},
			errorCodes: []string{"WF_SUBFLOW_TARGET"},
		},
		{
			name: "inline definition with empty nodes",
			data: map[string]any{
				"workflowDefinition": map[string]any{
					"nodes": []any{},
					"edges": []any{},
				},
			},
			errorCodes: []string{"WF_SUBFLOW_INLINE_NODES"},
		},
		{
			name: "inline definition missing edges",
			data: map[string]any{
				"workflowDefinition": map[string]any{
					"nodes": []any{
						map[string]any{"id": "n1", "type": "wait"},
					},
				},
			},
			errorCodes: []string{"WF_SUBFLOW_INLINE_EDGES"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workflow := map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "subflow-node",
						"type": "subflow",
						"data": tc.data,
					},
				},
				"edges": []any{},
			}

			res, err := v.Validate(context.Background(), workflow, Options{})
			if err != nil {
				t.Fatalf("validation returned error: %v", err)
			}
			if res.Valid {
				t.Fatalf("expected validation to fail, got valid=true")
			}
			for _, code := range tc.errorCodes {
				found := false
				for _, issue := range res.Errors {
					if issue.Code == code {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s, not found in %+v", code, res.Errors)
				}
			}
		})
	}
}

// ============================================================================
// Edge Validation Tests
// ============================================================================

func TestValidatorEdgeSelfReference(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "self-loop",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
		},
		"edges": []any{
			map[string]any{
				"id":     "self-edge",
				"source": "self-loop",
				"target": "self-loop",
			},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected self-referencing edge to fail validation")
	}
	assertIssue(t, res.Errors, "WF_EDGE_CYCLE_SELF")
}

func TestValidatorEdgeUnknownSource(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "valid-node",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
		},
		"edges": []any{
			map[string]any{
				"id":     "bad-edge",
				"source": "nonexistent",
				"target": "valid-node",
			},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected edge with unknown source to fail validation")
	}
	assertIssue(t, res.Errors, "WF_EDGE_SOURCE_UNKNOWN")
}

func TestValidatorEdgeUnknownTarget(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "valid-node",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
		},
		"edges": []any{
			map[string]any{
				"id":     "bad-edge",
				"source": "valid-node",
				"target": "nonexistent",
			},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected edge with unknown target to fail validation")
	}
	assertIssue(t, res.Errors, "WF_EDGE_TARGET_UNKNOWN")
}

func TestValidatorEdgeMissingEndpoints(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "node",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
		},
		"edges": []any{
			map[string]any{
				"id": "incomplete-edge",
			},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected edge without source/target to fail validation")
	}
	assertIssue(t, res.Errors, "WF_EDGE_ENDPOINT_MISSING")
}

// ============================================================================
// Node Structural Tests
// ============================================================================

func TestValidatorDuplicateNodeIDs(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "duplicate-id",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
			map[string]any{
				"id":   "duplicate-id",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 500},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected duplicate node IDs to fail validation")
	}
	assertIssue(t, res.Errors, "WF_NODE_ID_DUPLICATE")
}

func TestValidatorMissingNodeID(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected missing node ID to fail validation")
	}
	assertIssue(t, res.Errors, "WF_NODE_ID_MISSING")
}

func TestValidatorMissingNodeType(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "no-type",
				"data": map[string]any{},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected missing node type to fail validation")
	}
	assertIssue(t, res.Errors, "WF_NODE_TYPE_MISSING")
}

// ============================================================================
// Settings and Stats Tests
// ============================================================================

func TestValidatorViewportSizeWarning(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "node",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
		},
		"edges": []any{},
		"settings": map[string]any{
			"executionViewport": map[string]any{
				"width":  100,
				"height": 100,
			},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !assertWarning(res.Warnings, "WF_VIEWPORT_SMALL") {
		t.Fatalf("expected WF_VIEWPORT_SMALL warning, got %+v", res.Warnings)
	}
	if !res.Stats.HasExecutionViewport {
		t.Fatalf("expected HasExecutionViewport to be true")
	}
}

func TestValidatorEdgeSparsityWarning(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "node1",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
			map[string]any{
				"id":   "node2",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
			map[string]any{
				"id":   "node3",
				"type": "wait",
				"data": map[string]any{"waitType": "duration", "durationMs": 1000},
			},
		},
		"edges": []any{},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !assertWarning(res.Warnings, "WF_EDGE_SPARSITY") {
		t.Fatalf("expected WF_EDGE_SPARSITY warning for 3 nodes with 0 edges, got %+v", res.Warnings)
	}
}

func TestValidatorStatsComputation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"metadata": map[string]any{
			"description": "test workflow",
		},
		"nodes": []any{
			map[string]any{
				"id":   "click1",
				"type": "click",
				"data": map[string]any{"selector": "#button1"},
			},
			map[string]any{
				"id":   "click2",
				"type": "click",
				"data": map[string]any{"selector": "#button2"},
			},
			map[string]any{
				"id":   "click3",
				"type": "click",
				"data": map[string]any{"selector": "#button1"}, // duplicate selector
			},
			map[string]any{
				"id":   "wait1",
				"type": "wait",
				"data": map[string]any{"waitType": "element", "selector": "#loaded"},
			},
		},
		"edges": []any{
			map[string]any{"id": "e1", "source": "click1", "target": "click2"},
			map[string]any{"id": "e2", "source": "click2", "target": "click3"},
			map[string]any{"id": "e3", "source": "click3", "target": "wait1"},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}

	if res.Stats.NodeCount != 4 {
		t.Errorf("expected NodeCount=4, got %d", res.Stats.NodeCount)
	}
	if res.Stats.EdgeCount != 3 {
		t.Errorf("expected EdgeCount=3, got %d", res.Stats.EdgeCount)
	}
	if res.Stats.SelectorCount != 4 {
		t.Errorf("expected SelectorCount=4 (including wait selector), got %d", res.Stats.SelectorCount)
	}
	if res.Stats.UniqueSelectorCount != 3 {
		t.Errorf("expected UniqueSelectorCount=3, got %d", res.Stats.UniqueSelectorCount)
	}
	if res.Stats.ElementWaitCount != 1 {
		t.Errorf("expected ElementWaitCount=1, got %d", res.Stats.ElementWaitCount)
	}
	if !res.Stats.HasMetadata {
		t.Error("expected HasMetadata=true")
	}
}

func TestValidatorSelectorDuplicatesWarning(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "click1",
				"type": "click",
				"data": map[string]any{"selector": "#same-selector"},
			},
			map[string]any{
				"id":   "click2",
				"type": "click",
				"data": map[string]any{"selector": "#same-selector"},
			},
		},
		"edges": []any{
			map[string]any{"id": "e1", "source": "click1", "target": "click2"},
		},
	}

	res, err := v.Validate(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !assertWarning(res.Warnings, "WF_SELECTOR_DUPLICATES") {
		t.Fatalf("expected WF_SELECTOR_DUPLICATES warning, got %+v", res.Warnings)
	}
}

// ============================================================================
// Context Cancellation Test
// ============================================================================

func TestValidatorContextCancellation(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	workflow := map[string]any{
		"nodes": []any{},
		"edges": []any{},
	}

	_, err = v.Validate(ctx, workflow, Options{})
	if err == nil {
		t.Fatalf("expected error from cancelled context")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}

// ============================================================================
// Nil/Empty Input Tests
// ============================================================================

func TestValidatorNilDefinition(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	res, err := v.Validate(context.Background(), nil, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	// nil definition should be treated as empty workflow
	if res.Valid {
		t.Fatalf("expected empty workflow to be invalid")
	}
}

// ============================================================================
// SortIssues Test
// ============================================================================

func TestSortIssues(t *testing.T) {
	issues := []Issue{
		{Severity: SeverityWarning, Code: "WF_B", NodeID: "n2", Message: "msg2"},
		{Severity: SeverityError, Code: "WF_A", NodeID: "n1", Message: "msg1"},
		{Severity: SeverityError, Code: "WF_A", NodeID: "n1", Message: "msg0"},
		{Severity: SeverityWarning, Code: "WF_A", NodeID: "n1", Message: "msg3"},
	}

	SortIssues(issues)

	// After sorting: errors before warnings, then by code, then by nodeID, then by message
	if issues[0].Severity != SeverityError || issues[0].Message != "msg0" {
		t.Errorf("expected first issue to be error with msg0, got %+v", issues[0])
	}
	if issues[1].Severity != SeverityError || issues[1].Message != "msg1" {
		t.Errorf("expected second issue to be error with msg1, got %+v", issues[1])
	}
	if issues[2].Severity != SeverityWarning || issues[2].Code != "WF_A" {
		t.Errorf("expected third issue to be warning WF_A, got %+v", issues[2])
	}
	if issues[3].Severity != SeverityWarning || issues[3].Code != "WF_B" {
		t.Errorf("expected fourth issue to be warning WF_B, got %+v", issues[3])
	}
}

// ============================================================================
// ValidateResolved Tests - Token resolution validation
// ============================================================================

func TestValidateResolvedCleanWorkflow(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	// A fully resolved workflow should pass
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "url",
					"url":             "https://example.com",
				},
			},
			map[string]any{
				"id":   "click",
				"type": "click",
				"data": map[string]any{
					"selector": "#button",
				},
			},
		},
		"edges": []any{
			map[string]any{"id": "e1", "source": "nav", "target": "click"},
		},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected resolved workflow to be valid, got errors: %+v", res.Errors)
	}
}

func TestValidateResolvedUnresolvedFixture(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "subflow",
				"type": "subflow",
				"data": map[string]any{
					"workflowId": "@fixture/login-flow",
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected unresolved fixture to fail validation")
	}
	assertIssue(t, res.Errors, "WF_UNRESOLVED_FIXTURE")
}

// TestValidateResolvedSelectorTokensAllowed verifies that @selector/ tokens are NOT flagged
// as unresolved by ValidateResolved(), since they are resolved at compile time by BAS compiler.
// This allows test-genie to pass @selector/ tokens through to BAS for native resolution.
func TestValidateResolvedSelectorTokensAllowed(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "click",
				"type": "click",
				"data": map[string]any{
					"selector": "@selector/button.submit",
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	// @selector/ tokens should NOT be flagged as errors - they're resolved at compile time
	if !res.Valid {
		t.Fatalf("expected @selector/ tokens to be allowed (resolved at compile time), but got errors: %+v", res.Errors)
	}
}

func TestValidateResolvedUnresolvedSeed(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "type",
				"type": "type",
				"data": map[string]any{
					"selector": "#email",
					"text":     "@seed/user.email",
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected unresolved seed to fail validation")
	}
	assertIssue(t, res.Errors, "WF_UNRESOLVED_SEED")
}

func TestValidateResolvedUnresolvedPlaceholder(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "url",
					"url":             "${BASE_URL}/dashboard",
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected unresolved placeholder to fail validation")
	}
	assertIssue(t, res.Errors, "WF_UNRESOLVED_PLACEHOLDER")
}

func TestValidateResolvedUnresolvedTemplate(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "type",
				"type": "type",
				"data": map[string]any{
					"selector": "#name",
					"text":     "{{userName}}",
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected unresolved template to fail validation")
	}
	assertIssue(t, res.Errors, "WF_UNRESOLVED_TEMPLATE")
}

func TestValidateResolvedScenarioDestination(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	// Navigate with destinationType=scenario should flag unresolved scenario reference
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "test-app",
					"scenarioPath":    "/dashboard",
				},
			},
		},
		"edges": []any{},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected unresolved scenario navigation to fail validation")
	}
	assertIssue(t, res.Errors, "WF_UNRESOLVED_SCENARIO_URL")
}

func TestValidateResolvedMultipleIssues(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}

	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "nav",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "url",
					"url":             "${BASE_URL}/page",
				},
			},
			map[string]any{
				"id":   "click",
				"type": "click",
				"data": map[string]any{
					"selector": "@selector/button.main",
				},
			},
			map[string]any{
				"id":   "type",
				"type": "type",
				"data": map[string]any{
					"selector": "#input",
					"text":     "@seed/test.value",
				},
			},
		},
		"edges": []any{
			map[string]any{"id": "e1", "source": "nav", "target": "click"},
			map[string]any{"id": "e2", "source": "click", "target": "type"},
		},
	}

	res, err := v.ValidateResolved(context.Background(), workflow, Options{})
	if err != nil {
		t.Fatalf("validation returned error: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected multiple unresolved tokens to fail validation")
	}

	// Should have at least 2 errors (placeholder and seed - selector is allowed)
	// @selector/ tokens are NOT flagged as unresolved since they're resolved at compile time
	if len(res.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %+v", len(res.Errors), res.Errors)
	}

	assertIssue(t, res.Errors, "WF_UNRESOLVED_PLACEHOLDER")
	// NOTE: @selector/ is intentionally NOT checked - it's resolved at compile time by BAS compiler
	assertIssue(t, res.Errors, "WF_UNRESOLVED_SEED")
}
