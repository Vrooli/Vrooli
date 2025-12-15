package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPreflightValidatorBasicStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple workflow with no tokens
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

	// Create workflow with subflow using new workflowPath format
	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "subflow",
				"type": "subflow",
				"data": map[string]any{
					"label":        "Call helper",
					"workflowPath": "actions/helper.json",
					"params": map[string]any{
						"projectName": "${@params/projectName}",
						"projectId":   "${@params/projectId}",
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
				"id":   "subflow",
				"type": "subflow",
				"data": map[string]any{
					"workflowPath": "/absolute/path/to/helper.json",
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
				"id":   "click1",
				"type": "click",
				"data": map[string]any{
					"selector": "@selector/button.submit",
				},
			},
			map[string]any{
				"id":   "click2",
				"type": "click",
				"data": map[string]any{
					"selector": "@selector/button.cancel",
				},
			},
			map[string]any{
				"id":   "type1",
				"type": "type",
				"data": map[string]any{
					"selector": "@selector/input.name",
					"text":     "test",
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
				"id":   "nav",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "browser-automation-studio",
					"scenarioPath":    "/workflows",
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
				"id":   "nav",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenarioPath":    "/workflows",
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

	// Create two workflows with different characteristics
	wf1 := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "sf1",
				"type": "subflow",
				"data": map[string]any{
					"workflowPath": "actions/helper1.json",
					"params": map[string]any{
						"param1": "${@params/value1}",
					},
				},
			},
		},
		"edges": []any{},
	}
	wf2 := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "sf2",
				"type": "subflow",
				"data": map[string]any{
					"workflowPath": "actions/helper2.json",
					"params": map[string]any{
						"param2": "${@params/value2}",
					},
				},
			},
			map[string]any{
				"id":   "nav",
				"type": "navigate",
				"data": map[string]any{
					"destinationType": "scenario",
					"scenario":        "test-scenario",
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
				"id":   "subflow",
				"type": "subflow",
				"data": map[string]any{
					"label": "Inline subflow",
					"workflowDefinition": map[string]any{
						"nodes": []any{
							map[string]any{
								"id":   "nested-click",
								"type": "click",
								"data": map[string]any{
									"selector": "@selector/nested.button",
								},
							},
							map[string]any{
								"id":   "nested-subflow",
								"type": "subflow",
								"data": map[string]any{
									"workflowPath": "actions/deeply-nested.json",
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
	// Note: The scanner recursively scans nested workflowDefinition nodes
	if result.TokenCounts.Selectors < 1 {
		t.Errorf("expected at least 1 selector (from nested def), got %d", result.TokenCounts.Selectors)
	}

	if result.TokenCounts.Subflows != 1 {
		t.Errorf("expected 1 subflow (from nested def), got %d", result.TokenCounts.Subflows)
	}
}
