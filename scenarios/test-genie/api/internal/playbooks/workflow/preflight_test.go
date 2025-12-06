package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPreflightValidatorNoTokens(t *testing.T) {
	// Create temp scenario directory
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

	if result.TokenCounts.Fixtures != 0 {
		t.Errorf("expected 0 fixtures, got %d", result.TokenCounts.Fixtures)
	}
	if result.TokenCounts.Selectors != 0 {
		t.Errorf("expected 0 selectors, got %d", result.TokenCounts.Selectors)
	}
}

func TestPreflightValidatorMissingFixture(t *testing.T) {
	tmpDir := t.TempDir()

	// Create __subflows directory (empty)
	if err := os.MkdirAll(filepath.Join(tmpDir, "test", "playbooks", "__subflows"), 0755); err != nil {
		t.Fatalf("failed to create fixtures dir: %v", err)
	}

	// Create workflow referencing a fixture that doesn't exist
	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "subflow",
				"type": "subflow",
				"data": map[string]any{
					"workflowId": "@fixture/missing-fixture",
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
		t.Fatalf("expected invalid result for missing fixture")
	}

	if result.TokenCounts.Fixtures != 1 {
		t.Errorf("expected 1 fixture, got %d", result.TokenCounts.Fixtures)
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "PF_FIXTURE_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_FIXTURE_NOT_FOUND error, got: %+v", result.Errors)
	}
}

func TestPreflightValidatorExistingFixture(t *testing.T) {
	tmpDir := t.TempDir()

	// Create __subflows directory with a fixture
	subflowsDir := filepath.Join(tmpDir, "test", "playbooks", "__subflows")
	if err := os.MkdirAll(subflowsDir, 0755); err != nil {
		t.Fatalf("failed to create fixtures dir: %v", err)
	}

	// Create fixture file
	fixture := map[string]any{
		"metadata": map[string]any{
			"fixture_id": "login-flow",
		},
		"nodes": []any{},
		"edges": []any{},
	}
	fixtureData, _ := json.Marshal(fixture)
	if err := os.WriteFile(filepath.Join(subflowsDir, "login-flow.json"), fixtureData, 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Create workflow referencing the fixture
	workflowPath := filepath.Join(tmpDir, "test.json")
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

	if result.TokenCounts.Fixtures != 1 {
		t.Errorf("expected 1 fixture, got %d", result.TokenCounts.Fixtures)
	}
}

func TestPreflightValidatorMissingSelector(t *testing.T) {
	tmpDir := t.TempDir()

	// Create selector manifest with one selector
	manifestDir := filepath.Join(tmpDir, "ui", "src", "constants")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatalf("failed to create manifest dir: %v", err)
	}

	manifest := SelectorManifest{
		Selectors: map[string]SelectorEntry{
			"button.submit": {Selector: "#submit-btn"},
		},
	}
	manifestData, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(manifestDir, "selectors.manifest.json"), manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create workflow referencing a selector that doesn't exist
	workflowPath := filepath.Join(tmpDir, "test.json")
	workflow := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "click",
				"type": "click",
				"data": map[string]any{
					"selector": "@selector/button.nonexistent",
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
		t.Fatalf("expected invalid result for missing selector")
	}

	if result.TokenCounts.Selectors != 1 {
		t.Errorf("expected 1 selector, got %d", result.TokenCounts.Selectors)
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "PF_SELECTOR_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_SELECTOR_NOT_FOUND error, got: %+v", result.Errors)
	}
}

func TestPreflightValidatorExistingSelector(t *testing.T) {
	tmpDir := t.TempDir()

	// Create selector manifest
	manifestDir := filepath.Join(tmpDir, "ui", "src", "constants")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatalf("failed to create manifest dir: %v", err)
	}

	manifest := SelectorManifest{
		Selectors: map[string]SelectorEntry{
			"button.submit": {Selector: "#submit-btn"},
		},
	}
	manifestData, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(manifestDir, "selectors.manifest.json"), manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create workflow referencing an existing selector
	workflowPath := filepath.Join(tmpDir, "test.json")
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

	if result.TokenCounts.Selectors != 1 {
		t.Errorf("expected 1 selector, got %d", result.TokenCounts.Selectors)
	}
}

func TestPreflightValidatorSeedWarning(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workflow with seed reference
	workflowPath := filepath.Join(tmpDir, "test.json")
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

	data, _ := json.Marshal(workflow)
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	v := NewPreflightValidator(tmpDir)
	result, err := v.Validate(workflowPath)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Seeds generate warnings, not errors
	if !result.Valid {
		t.Fatalf("expected valid result (seeds only produce warnings), got errors: %+v", result.Errors)
	}

	if result.TokenCounts.Seeds != 1 {
		t.Errorf("expected 1 seed, got %d", result.TokenCounts.Seeds)
	}

	found := false
	for _, issue := range result.Warnings {
		if issue.Code == "PF_SEED_RUNTIME_DEPENDENCY" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_SEED_RUNTIME_DEPENDENCY warning, got: %+v", result.Warnings)
	}
}

func TestPreflightValidatorScenarioNavigation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workflow with scenario navigation
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

	// Should extract required scenarios
	if len(result.RequiredScenarios) != 1 {
		t.Errorf("expected 1 required scenario, got %d", len(result.RequiredScenarios))
	}

	if len(result.RequiredScenarios) > 0 && result.RequiredScenarios[0] != "browser-automation-studio" {
		t.Errorf("expected browser-automation-studio, got %s", result.RequiredScenarios[0])
	}
}

func TestPreflightValidatorMissingScenarioName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workflow with scenario navigation but missing scenario name
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

	// Create fixtures dir with one fixture
	subflowsDir := filepath.Join(tmpDir, "test", "playbooks", "__subflows")
	if err := os.MkdirAll(subflowsDir, 0755); err != nil {
		t.Fatalf("failed to create fixtures dir: %v", err)
	}

	fixture := map[string]any{
		"metadata": map[string]any{"fixture_id": "existing"},
		"nodes":    []any{},
		"edges":    []any{},
	}
	fixtureData, _ := json.Marshal(fixture)
	os.WriteFile(filepath.Join(subflowsDir, "existing.json"), fixtureData, 0644)

	// Create two workflows
	wf1 := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "sf1",
				"type": "subflow",
				"data": map[string]any{"workflowId": "@fixture/existing"},
			},
		},
		"edges": []any{},
	}
	wf2 := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "sf2",
				"type": "subflow",
				"data": map[string]any{"workflowId": "@fixture/missing"},
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

	if result.Valid {
		t.Fatalf("expected invalid result due to wf2 missing fixture")
	}

	if result.TokenCounts.Fixtures != 2 {
		t.Errorf("expected 2 total fixtures, got %d", result.TokenCounts.Fixtures)
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "PF_FIXTURE_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PF_FIXTURE_NOT_FOUND error from wf2")
	}
}
