package compat

import (
	"encoding/json"
	"testing"
)

func TestNormalizeNodeV1ToV2_ClickNode(t *testing.T) {
	node := map[string]any{
		"id":   "node-1",
		"type": "click",
		"data": map[string]any{
			"selector": "#submit-btn",
			"label":    "Submit Button",
		},
	}

	normalizeNodeV1ToV2(node)

	// Should have action field now
	action, ok := node["action"].(map[string]any)
	if !ok {
		t.Fatal("expected action field after normalization")
	}

	// type field should be removed
	if _, hasType := node["type"]; hasType {
		t.Error("type field should be removed after normalization")
	}

	// data field should be removed
	if _, hasData := node["data"]; hasData {
		t.Error("data field should be removed after normalization")
	}

	// Check action structure
	if action["type"] != "ACTION_TYPE_CLICK" {
		t.Errorf("expected ACTION_TYPE_CLICK, got %v", action["type"])
	}

	click, ok := action["click"].(map[string]any)
	if !ok {
		t.Fatal("expected click params")
	}
	if click["selector"] != "#submit-btn" {
		t.Errorf("expected selector '#submit-btn', got %v", click["selector"])
	}

	// Label should be in metadata
	metadata, ok := action["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata")
	}
	if metadata["label"] != "Submit Button" {
		t.Errorf("expected label 'Submit Button', got %v", metadata["label"])
	}
}

func TestNormalizeNodeV1ToV2_TypeNode(t *testing.T) {
	node := map[string]any{
		"id":   "node-1",
		"type": "type",
		"data": map[string]any{
			"selector": "#email",
			"text":     "test@example.com",
		},
	}

	normalizeNodeV1ToV2(node)

	action, ok := node["action"].(map[string]any)
	if !ok {
		t.Fatal("expected action field after normalization")
	}

	// type step becomes input action
	if action["type"] != "ACTION_TYPE_INPUT" {
		t.Errorf("expected ACTION_TYPE_INPUT, got %v", action["type"])
	}

	// params key should be "input" not "type"
	input, ok := action["input"].(map[string]any)
	if !ok {
		t.Fatal("expected input params")
	}

	// "text" should be renamed to "value"
	if input["value"] != "test@example.com" {
		t.Errorf("expected value 'test@example.com', got %v", input["value"])
	}
	if _, hasText := input["text"]; hasText {
		t.Error("text field should be renamed to value")
	}
}

func TestNormalizeNodeV1ToV2_NavigateNode(t *testing.T) {
	node := map[string]any{
		"id":   "node-1",
		"type": "navigate",
		"data": map[string]any{
			"url":       "http://example.com",
			"waitUntil": "networkidle",
		},
	}

	normalizeNodeV1ToV2(node)

	action, ok := node["action"].(map[string]any)
	if !ok {
		t.Fatal("expected action field after normalization")
	}

	if action["type"] != "ACTION_TYPE_NAVIGATE" {
		t.Errorf("expected ACTION_TYPE_NAVIGATE, got %v", action["type"])
	}

	navigate, ok := action["navigate"].(map[string]any)
	if !ok {
		t.Fatal("expected navigate params")
	}
	if navigate["url"] != "http://example.com" {
		t.Errorf("expected url 'http://example.com', got %v", navigate["url"])
	}
	if navigate["wait_until"] != "NAVIGATE_WAIT_EVENT_NETWORKIDLE" {
		t.Errorf("expected normalized wait event, got %v", navigate["wait_until"])
	}
}

func TestNormalizeNodeV1ToV2_V2NodeUnchanged(t *testing.T) {
	// V2 node already has action field - should not be modified
	node := map[string]any{
		"id": "node-1",
		"action": map[string]any{
			"type": "ACTION_TYPE_CLICK",
			"click": map[string]any{
				"selector": "#btn",
			},
		},
	}

	// Make a copy to compare
	originalJSON, _ := json.Marshal(node)

	normalizeNodeV1ToV2(node)

	newJSON, _ := json.Marshal(node)
	if string(originalJSON) != string(newJSON) {
		t.Error("V2 node should not be modified")
	}
}

func TestNormalizeNodeV1ToV2_AssertNode(t *testing.T) {
	node := map[string]any{
		"id":   "node-1",
		"type": "assert",
		"data": map[string]any{
			"selector":   "[data-testid='dashboard']",
			"assertMode": "exists",
		},
	}

	normalizeNodeV1ToV2(node)

	action, ok := node["action"].(map[string]any)
	if !ok {
		t.Fatal("expected action field after normalization")
	}

	if action["type"] != "ACTION_TYPE_ASSERT" {
		t.Errorf("expected ACTION_TYPE_ASSERT, got %v", action["type"])
	}

	assertParams, ok := action["assert"].(map[string]any)
	if !ok {
		t.Fatal("expected assert params")
	}
	if assertParams["selector"] != "[data-testid='dashboard']" {
		t.Errorf("expected selector, got %v", assertParams["selector"])
	}
	if assertParams["mode"] != "ASSERTION_MODE_EXISTS" {
		t.Errorf("expected normalized assertion mode, got %v", assertParams["mode"])
	}
}

func TestNormalizeWorkflowDefinitionV2_ExecutionModeMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"observer short form", "observer", "EXECUTION_MODE_OBSERVER"},
		{"mutating short form", "mutating", "EXECUTION_MODE_MUTATING"},
		{"destructive short form", "destructive", "EXECUTION_MODE_DESTRUCTIVE"},
		{"already full enum name", "EXECUTION_MODE_OBSERVER", "EXECUTION_MODE_OBSERVER"},
		{"unknown value passes through", "custom", "custom"},
		{"empty string passes through", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := map[string]any{
				"metadata": map[string]any{
					"execution_mode": tc.input,
				},
			}

			NormalizeWorkflowDefinitionV2(doc)

			metadata := doc["metadata"].(map[string]any)
			got := metadata["execution_mode"]
			if got != tc.expected {
				t.Errorf("execution_mode: got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestNormalizeWorkflowDefinitionV2_ExecutionModeCamelCase(t *testing.T) {
	// When the field is already in camelCase (e.g., from a pre-normalized payload)
	doc := map[string]any{
		"metadata": map[string]any{
			"executionMode": "observer",
		},
	}

	NormalizeWorkflowDefinitionV2(doc)

	metadata := doc["metadata"].(map[string]any)
	got := metadata["executionMode"]
	if got != "EXECUTION_MODE_OBSERVER" {
		t.Errorf("executionMode: got %q, want %q", got, "EXECUTION_MODE_OBSERVER")
	}
}

func TestNormalizeExecuteAdhocRequest_WithExecutionMode(t *testing.T) {
	// End-to-end: a realistic workflow with short-form execution_mode
	// should be normalized so protojson accepts it.
	body := []byte(`{
		"flow_definition": {
			"metadata": {
				"name": "test-workflow",
				"description": "Test workflow",
				"execution_mode": "observer"
			},
			"nodes": [],
			"edges": []
		},
		"wait_for_completion": true
	}`)

	result, err := NormalizeExecuteAdhocRequest(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	flowDef := parsed["flowDefinition"].(map[string]any)
	metadata := flowDef["metadata"].(map[string]any)
	got := metadata["executionMode"]
	if got != "EXECUTION_MODE_OBSERVER" {
		t.Errorf("executionMode: got %q, want %q", got, "EXECUTION_MODE_OBSERVER")
	}
}

func TestNormalizeWorkflowDefinitionV2_MixedNodes(t *testing.T) {
	// Workflow with both V1 and V2 nodes
	doc := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "node-1",
				"type": "click",
				"data": map[string]any{
					"selector": "#btn1",
				},
			},
			map[string]any{
				"id": "node-2",
				"action": map[string]any{
					"type": "ACTION_TYPE_CLICK",
					"click": map[string]any{
						"selector": "#btn2",
					},
				},
			},
		},
	}

	NormalizeWorkflowDefinitionV2(doc)

	nodes := doc["nodes"].([]any)

	// First node should be transformed
	node1 := nodes[0].(map[string]any)
	action1, ok := node1["action"].(map[string]any)
	if !ok {
		t.Fatal("node1 should have action after normalization")
	}
	if action1["type"] != "ACTION_TYPE_CLICK" {
		t.Errorf("node1: expected ACTION_TYPE_CLICK, got %v", action1["type"])
	}

	// Second node should remain unchanged
	node2 := nodes[1].(map[string]any)
	action2, ok := node2["action"].(map[string]any)
	if !ok {
		t.Fatal("node2 should still have action")
	}
	click2, ok := action2["click"].(map[string]any)
	if !ok {
		t.Fatal("node2 should still have click params")
	}
	if click2["selector"] != "#btn2" {
		t.Errorf("node2: selector should be unchanged, got %v", click2["selector"])
	}
}

func TestStepTypeToActionType(t *testing.T) {
	tests := []struct {
		stepType   string
		actionType string
	}{
		{"navigate", "ACTION_TYPE_NAVIGATE"},
		{"click", "ACTION_TYPE_CLICK"},
		{"type", "ACTION_TYPE_INPUT"},
		{"input", "ACTION_TYPE_INPUT"},
		{"assert", "ACTION_TYPE_ASSERT"},
		{"wait", "ACTION_TYPE_WAIT"},
		{"screenshot", "ACTION_TYPE_SCREENSHOT"},
		{"evaluate", "ACTION_TYPE_EVALUATE"},
		{"hover", "ACTION_TYPE_HOVER"},
		{"subflow", "ACTION_TYPE_SUBFLOW"},
		{"dragDrop", "ACTION_TYPE_DRAG_DROP"},
		{"drag_drop", "ACTION_TYPE_DRAG_DROP"},
	}

	for _, tc := range tests {
		t.Run(tc.stepType, func(t *testing.T) {
			result := stepTypeToActionType(tc.stepType)
			if result != tc.actionType {
				t.Errorf("stepTypeToActionType(%q) = %q, want %q", tc.stepType, result, tc.actionType)
			}
		})
	}
}

func TestStepTypeToParamsKey(t *testing.T) {
	tests := []struct {
		stepType  string
		paramsKey string
	}{
		{"click", "click"},
		{"navigate", "navigate"},
		{"type", "input"},
		{"assert", "assert"},
		{"dragDrop", "dragDrop"},
		{"drag_drop", "dragDrop"},
	}

	for _, tc := range tests {
		t.Run(tc.stepType, func(t *testing.T) {
			result := stepTypeToParamsKey(tc.stepType)
			if result != tc.paramsKey {
				t.Errorf("stepTypeToParamsKey(%q) = %q, want %q", tc.stepType, result, tc.paramsKey)
			}
		})
	}
}

func TestNormalizeExecutionParameters_UnknownFieldsToInitialParams(t *testing.T) {
	params := map[string]any{
		"username": "test@example.com",
		"password": "secret123",
	}

	NormalizeExecutionParameters(params)

	// Unknown fields should be moved to initial_params
	initialParams, ok := params["initial_params"].(map[string]any)
	if !ok {
		t.Fatal("expected initial_params to be created")
	}
	if initialParams["username"] != "test@example.com" {
		t.Errorf("expected username in initial_params, got %v", initialParams["username"])
	}
	if initialParams["password"] != "secret123" {
		t.Errorf("expected password in initial_params, got %v", initialParams["password"])
	}

	// Original fields should be removed
	if _, hasUsername := params["username"]; hasUsername {
		t.Error("username should be removed from top level")
	}
	if _, hasPassword := params["password"]; hasPassword {
		t.Error("password should be removed from top level")
	}
}

func TestNormalizeExecutionParameters_MergeWithExisting(t *testing.T) {
	params := map[string]any{
		"username": "test@example.com",
		"initial_params": map[string]any{
			"existing_key": "existing_value",
			"username":     "original_user", // Should NOT be overwritten
		},
	}

	NormalizeExecutionParameters(params)

	initialParams, ok := params["initial_params"].(map[string]any)
	if !ok {
		t.Fatal("expected initial_params")
	}

	// Existing value should be preserved
	if initialParams["existing_key"] != "existing_value" {
		t.Errorf("expected existing_key to be preserved, got %v", initialParams["existing_key"])
	}

	// Should NOT overwrite existing username
	if initialParams["username"] != "original_user" {
		t.Errorf("expected original username to be preserved, got %v", initialParams["username"])
	}
}

func TestNormalizeExecutionParameters_KnownFieldsUnchanged(t *testing.T) {
	params := map[string]any{
		"initial_params": map[string]any{"key": "value"},
		"initial_store":  map[string]any{"counter": 0},
		"env":            map[string]any{"debug": true},
		"startUrl":       "http://localhost:3000",
		"projectRoot":    "/path/to/project",
	}

	originalJSON, _ := json.Marshal(params)
	NormalizeExecutionParameters(params)
	newJSON, _ := json.Marshal(params)

	if string(originalJSON) != string(newJSON) {
		t.Error("known fields should not be modified")
	}
}

func TestNormalizeExecutionParameters_EmptyParams(t *testing.T) {
	params := map[string]any{}
	NormalizeExecutionParameters(params)

	if _, hasInitialParams := params["initial_params"]; hasInitialParams {
		t.Error("should not create initial_params for empty params")
	}
}

func TestNormalizeExecutionParameters_NilParams(t *testing.T) {
	// Should not panic
	NormalizeExecutionParameters(nil)
}

func TestNormalizeExecutionParameters_SessionProfileFields(t *testing.T) {
	// Session profile fields should be recognized as known fields
	// and NOT moved to initial_params
	tests := []struct {
		name   string
		params map[string]any
	}{
		{
			name: "session_profile_id snake_case",
			params: map[string]any{
				"session_profile_id": "abc-123",
			},
		},
		{
			name: "sessionProfileId camelCase",
			params: map[string]any{
				"sessionProfileId": "abc-123",
			},
		},
		{
			name: "save_session_profile_id snake_case",
			params: map[string]any{
				"save_session_profile_id": "def-456",
			},
		},
		{
			name: "saveSessionProfileId camelCase",
			params: map[string]any{
				"saveSessionProfileId": "def-456",
			},
		},
		{
			name: "both session fields together",
			params: map[string]any{
				"session_profile_id":      "abc-123",
				"save_session_profile_id": "def-456",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalJSON, _ := json.Marshal(tc.params)
			NormalizeExecutionParameters(tc.params)
			newJSON, _ := json.Marshal(tc.params)

			if string(originalJSON) != string(newJSON) {
				t.Errorf("session profile fields should not be modified\nbefore: %s\nafter:  %s", originalJSON, newJSON)
			}

			// Verify no initial_params was created
			if _, hasInitialParams := tc.params["initial_params"]; hasInitialParams {
				t.Error("should not create initial_params for known session profile fields")
			}
		})
	}
}

func TestNormalizeExecutionParameters_SessionProfileWithOtherFields(t *testing.T) {
	// Test that session profile fields are preserved while unknown fields are moved
	params := map[string]any{
		"session_profile_id":      "abc-123",
		"save_session_profile_id": "def-456",
		"username":                "test@example.com", // unknown field
	}

	NormalizeExecutionParameters(params)

	// Session profile fields should remain at top level
	if params["session_profile_id"] != "abc-123" {
		t.Errorf("session_profile_id should remain at top level, got %v", params["session_profile_id"])
	}
	if params["save_session_profile_id"] != "def-456" {
		t.Errorf("save_session_profile_id should remain at top level, got %v", params["save_session_profile_id"])
	}

	// Unknown field should be moved to initial_params
	initialParams, ok := params["initial_params"].(map[string]any)
	if !ok {
		t.Fatal("expected initial_params to be created for unknown fields")
	}
	if initialParams["username"] != "test@example.com" {
		t.Errorf("expected username in initial_params, got %v", initialParams["username"])
	}

	// Username should be removed from top level
	if _, hasUsername := params["username"]; hasUsername {
		t.Error("username should be removed from top level")
	}
}
