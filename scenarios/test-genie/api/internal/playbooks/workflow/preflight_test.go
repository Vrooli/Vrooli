package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPreflightValidatorBasicStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple V2 proto format workflow with no tokens
	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "nav",
				"action": map[string]any{
					"type": "ACTION_TYPE_NAVIGATE",
					"navigate": map[string]any{
						"destination_type": "NAVIGATE_DESTINATION_TYPE_URL",
						"url":              "https://example.com",
					},
				},
			},
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %+v", result.Errors)
	}

	if result.TokenCounts.Selectors != 0 {
		t.Errorf("expected 0 selectors, got %d", result.TokenCounts.Selectors)
	}
	if result.TokenCounts.Subflows != 0 {
		t.Errorf("expected 0 subflows, got %d", result.TokenCounts.Subflows)
	}
	if result.TokenCounts.ParamsTokens != 0 {
		t.Errorf("expected 0 params tokens, got %d", result.TokenCounts.ParamsTokens)
	}
}

func TestPreflightValidatorLegacyV1Format(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a workflow using deprecated V1 format (type+data)
	workflowPath := filepath.Join(tmpDir, "test.json")
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
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if result.Valid {
		t.Fatalf("expected invalid result for V1 format, got valid")
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "PF_LEGACY_V1_FORMAT" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_LEGACY_V1_FORMAT error, got: %+v", result.Errors)
	}
}

func TestPreflightValidatorLegacyStepsFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a workflow using deprecated steps format
	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"steps": []any{
			map[string]any{
				"action":   "navigate",
				"url":      "https://example.com",
				"waitFor":  "networkidle",
				"timeout":  30000,
			},
			map[string]any{
				"action":   "click",
				"selector": "button.submit",
			},
		},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if result.Valid {
		t.Fatalf("expected invalid result for steps format, got valid")
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "PF_LEGACY_STEPS_FORMAT" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_LEGACY_STEPS_FORMAT error, got: %+v", result.Errors)
	}
}

func TestPreflightValidatorMissingNodes(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	os.WriteFile(workflowPath, data, 0644)

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if result.Valid {
		t.Fatalf("expected invalid result for missing nodes")
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "PF_MISSING_NODES" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_MISSING_NODES error, got: %+v", result.Errors)
	}
}

func TestPreflightValidatorSubflowWithWorkflowPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create V2 proto format workflow with subflow
	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "subflow",
				"action": map[string]any{
					"type": "ACTION_TYPE_SUBFLOW",
					"subflow": map[string]any{
						"workflow_path": "actions/helper.json",
						"params": map[string]any{
							"projectName": "${@params/projectName}",
							"projectId":   "${@params/projectId}",
						},
					},
					"metadata": map[string]any{
						"label": "Call helper",
					},
				},
			},
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %+v", result.Errors)
	}

	if result.TokenCounts.Subflows != 1 {
		t.Errorf("expected 1 subflow, got %d", result.TokenCounts.Subflows)
	}

	if result.TokenCounts.ParamsTokens != 2 {
		t.Errorf("expected 2 params tokens, got %d", result.TokenCounts.ParamsTokens)
	}
}

func TestPreflightValidatorAbsoluteWorkflowPathWarning(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "subflow",
				"action": map[string]any{
					"type": "ACTION_TYPE_SUBFLOW",
					"subflow": map[string]any{
						"workflow_path": "/absolute/path/to/helper.json",
					},
				},
			},
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	os.WriteFile(workflowPath, data, 0644)

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Should be valid but with warning
	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %+v", result.Errors)
	}

	found := false
	for _, issue := range result.Warnings {
		if issue.Code == "PF_ABSOLUTE_WORKFLOW_PATH" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_ABSOLUTE_WORKFLOW_PATH warning, got: %+v", result.Warnings)
	}
}

// TestPreflightValidatorSelectorCounting verifies that preflight counts @selector/ tokens
// without validating them. Selector validation is delegated to BAS compiler.
func TestPreflightValidatorSelectorCounting(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "click1",
				"action": map[string]any{
					"type": "ACTION_TYPE_CLICK",
					"click": map[string]any{
						"selector": "@selector/button.submit",
					},
				},
			},
			map[string]any{
				"id": "click2",
				"action": map[string]any{
					"type": "ACTION_TYPE_CLICK",
					"click": map[string]any{
						"selector": "@selector/button.cancel",
					},
				},
			},
			map[string]any{
				"id": "input1",
				"action": map[string]any{
					"type": "ACTION_TYPE_INPUT",
					"input": map[string]any{
						"selector": "@selector/input.name",
						"text":     "test",
					},
				},
			},
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %+v", result.Errors)
	}

	if result.TokenCounts.Selectors != 3 {
		t.Errorf("expected 3 selectors counted, got %d", result.TokenCounts.Selectors)
	}
}

func TestPreflightValidatorScenarioNavigation(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "nav",
				"action": map[string]any{
					"type": "ACTION_TYPE_NAVIGATE",
					"navigate": map[string]any{
						"destination_type": "NAVIGATE_DESTINATION_TYPE_SCENARIO",
						"scenario":         "browser-automation-studio",
						"scenario_path":    "/workflows",
					},
				},
			},
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %+v", result.Errors)
	}

	if len(result.RequiredScenarios) != 1 {
		t.Errorf("expected 1 required scenario, got %d", len(result.RequiredScenarios))
	}

	if len(result.RequiredScenarios) > 0 && result.RequiredScenarios[0] != "browser-automation-studio" {
		t.Errorf("expected browser-automation-studio, got %s", result.RequiredScenarios[0])
	}
}

func TestPreflightValidatorMissingScenarioName(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "nav",
				"action": map[string]any{
					"type": "ACTION_TYPE_NAVIGATE",
					"navigate": map[string]any{
						"destination_type": "NAVIGATE_DESTINATION_TYPE_SCENARIO",
						"scenario_path":    "/workflows",
					},
				},
			},
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if result.Valid {
		t.Fatalf("expected invalid result for missing scenario name")
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "PF_SCENARIO_NAME_MISSING" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_SCENARIO_NAME_MISSING error, got: %+v", result.Errors)
	}
}

func TestPreflightValidatorValidateAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two V2 proto format workflows with different characteristics
	wf1 := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "sf1",
				"action": map[string]any{
					"type": "ACTION_TYPE_SUBFLOW",
					"subflow": map[string]any{
						"workflow_path": "actions/helper1.json",
						"params": map[string]any{
							"param1": "${@params/value1}",
						},
					},
				},
			},
		},
		"edges": []any{},
	}
	wf2 := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "sf2",
				"action": map[string]any{
					"type": "ACTION_TYPE_SUBFLOW",
					"subflow": map[string]any{
						"workflow_path": "actions/helper2.json",
						"params": map[string]any{
							"param2": "${@params/value2}",
						},
					},
				},
			},
			map[string]any{
				"id": "nav",
				"action": map[string]any{
					"type": "ACTION_TYPE_NAVIGATE",
					"navigate": map[string]any{
						"destination_type": "NAVIGATE_DESTINATION_TYPE_SCENARIO",
						"scenario":         "test-scenario",
					},
				},
			},
		},
		"edges": []any{},
	}

	wf1Path := filepath.Join(tmpDir, "wf1.json")
	wf2Path := filepath.Join(tmpDir, "wf2.json")

	data1, _ := json.Marshal(wf1)
	data2, _ := json.Marshal(wf2)
	os.WriteFile(wf1Path, data1, 0644)
	os.WriteFile(wf2Path, data2, 0644)

	v := NewPreflightValidator(tmpDir)
	result, err := v.ValidateAll([]string{wf1Path, wf2Path})
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %+v", result.Errors)
	}

	if result.TokenCounts.Subflows != 2 {
		t.Errorf("expected 2 total subflows, got %d", result.TokenCounts.Subflows)
	}

	if result.TokenCounts.ParamsTokens != 2 {
		t.Errorf("expected 2 total params tokens, got %d", result.TokenCounts.ParamsTokens)
	}

	if len(result.RequiredScenarios) != 1 {
		t.Errorf("expected 1 required scenario, got %d", len(result.RequiredScenarios))
	}
}

func TestPreflightValidatorNestedWorkflowDefinition(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "subflow",
				"action": map[string]any{
					"type": "ACTION_TYPE_SUBFLOW",
					"subflow": map[string]any{
						"workflow_definition": map[string]any{
							"nodes": []any{
								map[string]any{
									"id": "nested-click",
									"action": map[string]any{
										"type": "ACTION_TYPE_CLICK",
										"click": map[string]any{
											"selector": "@selector/nested.button",
										},
									},
								},
								map[string]any{
									"id": "nested-subflow",
									"action": map[string]any{
										"type": "ACTION_TYPE_SUBFLOW",
										"subflow": map[string]any{
											"workflow_path": "actions/deeply-nested.json",
										},
									},
								},
							},
							"edges": []any{},
						},
					},
					"metadata": map[string]any{
						"label": "Inline subflow",
					},
				},
			},
		},
		"edges": []any{},
	}

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %+v", result.Errors)
	}

	// Should count nested selectors and subflows
	// Note: The scanner recursively scans nested workflow_definition nodes
	if result.TokenCounts.Selectors < 1 {
		t.Errorf("expected at least 1 selector (from nested def), got %d", result.TokenCounts.Selectors)
	}

	if result.TokenCounts.Subflows != 1 {
		t.Errorf("expected 1 subflow (from nested def), got %d", result.TokenCounts.Subflows)
	}
}
