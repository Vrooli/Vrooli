package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRejectLegacyWorkflowNodes(t *testing.T) {
	err := RejectLegacyWorkflowNodes(map[string]any{
		"nodes": []any{map[string]any{
			"id":   "legacy-click",
			"type": "click",
			"data": map[string]any{"selector": "#submit"},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "legacy type/data shape") {
		t.Fatalf("RejectLegacyWorkflowNodes() error = %v, want legacy shape rejection", err)
	}
}

func TestNormalizeWorkflowDefinitionV2BytesRejectsLegacyNodes(t *testing.T) {
	_, err := NormalizeWorkflowDefinitionV2Bytes([]byte(`{"nodes":[{"id":"legacy-click","type":"click","data":{"selector":"#submit"}}]}`))
	if err == nil || !strings.Contains(err.Error(), "legacy type/data shape") {
		t.Fatalf("NormalizeWorkflowDefinitionV2Bytes() error = %v, want legacy shape rejection", err)
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

func TestNormalizeWorkflowDefinitionV2_TypedNodes(t *testing.T) {
	// V2 workflows remain typed while their protojson details are normalized.
	doc := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "node-1",
				"action": map[string]any{
					"type": "ACTION_TYPE_CLICK",
					"click": map[string]any{
						"selector": "#btn1",
					},
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

	// First node remains typed.
	node1 := nodes[0].(map[string]any)
	action1, ok := node1["action"].(map[string]any)
	if !ok {
		t.Fatal("node1 should have typed action")
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
