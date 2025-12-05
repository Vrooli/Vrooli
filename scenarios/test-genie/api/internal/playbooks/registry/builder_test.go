package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuilderBuild(t *testing.T) {
	// Create temporary scenario directory structure
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "test-scenario")

	// Create required directories
	dirs := []string{
		filepath.Join(scenarioDir, "test", "playbooks", "capabilities", "01-feature"),
		filepath.Join(scenarioDir, "test", "playbooks", "__subflows"),
		filepath.Join(scenarioDir, "requirements"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create requirements/index.json
	indexContent := `{"imports": ["01-feature/module.json"]}`
	if err := os.MkdirAll(filepath.Join(scenarioDir, "requirements", "01-feature"), 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(indexContent), 0o644); err != nil {
		t.Fatalf("failed to write index.json: %v", err)
	}

	// Create requirements module with validation reference
	moduleContent := `{
		"requirements": [
			{
				"id": "TEST-REQ-001",
				"validation": [{"ref": "test/playbooks/capabilities/01-feature/test-workflow.json"}]
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "01-feature", "module.json"), []byte(moduleContent), 0o644); err != nil {
		t.Fatalf("failed to write module.json: %v", err)
	}

	// Create playbook file
	playbookContent := `{
		"metadata": {
			"description": "Test workflow description",
			"reset": "full"
		},
		"nodes": [
			{"data": {"workflowId": "@fixture/load-state"}},
			{"data": {"selector": "#button"}}
		],
		"edges": []
	}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "playbooks", "capabilities", "01-feature", "test-workflow.json"), []byte(playbookContent), 0o644); err != nil {
		t.Fatalf("failed to write playbook: %v", err)
	}

	// Create subflow (should be ignored)
	subflowContent := `{"metadata": {"fixture_id": "load-state"}, "nodes": []}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "playbooks", "__subflows", "load-state.json"), []byte(subflowContent), 0o644); err != nil {
		t.Fatalf("failed to write subflow: %v", err)
	}

	// Build registry
	builder := NewBuilder(scenarioDir)
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify result
	if result.PlaybookCount != 1 {
		t.Errorf("expected 1 playbook, got %d", result.PlaybookCount)
	}

	if len(result.Registry.Playbooks) != 1 {
		t.Fatalf("expected 1 registry entry, got %d", len(result.Registry.Playbooks))
	}

	entry := result.Registry.Playbooks[0]
	if entry.File != "test/playbooks/capabilities/01-feature/test-workflow.json" {
		t.Errorf("unexpected file path: %s", entry.File)
	}
	if entry.Description != "Test workflow description" {
		t.Errorf("unexpected description: %s", entry.Description)
	}
	if entry.Reset != "full" {
		t.Errorf("unexpected reset: %s", entry.Reset)
	}
	if len(entry.Requirements) != 1 || entry.Requirements[0] != "TEST-REQ-001" {
		t.Errorf("unexpected requirements: %v", entry.Requirements)
	}
	if len(entry.Fixtures) != 1 || entry.Fixtures[0] != "load-state" {
		t.Errorf("unexpected fixtures: %v", entry.Fixtures)
	}

	// Verify registry file was written
	registryPath := filepath.Join(scenarioDir, "test", "playbooks", "registry.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Error("registry.json was not created")
	}

	// Verify registry file content is valid JSON
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("failed to read registry.json: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("registry.json is not valid JSON: %v", err)
	}
}

func TestBuilderBuildMissingPlaybooksDir(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "test-scenario")

	// Only create requirements dir
	if err := os.MkdirAll(filepath.Join(scenarioDir, "requirements"), 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	builder := NewBuilder(scenarioDir)
	_, err := builder.Build()
	if err == nil {
		t.Error("expected error when test/playbooks is missing")
	}
}

func TestBuilderBuildMissingRequirementsDir(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "test-scenario")

	// Only create playbooks dir
	if err := os.MkdirAll(filepath.Join(scenarioDir, "test", "playbooks"), 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}

	builder := NewBuilder(scenarioDir)
	_, err := builder.Build()
	if err == nil {
		t.Error("expected error when requirements is missing")
	}
}

func TestBuilderBuildInvalidResetValue(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "test-scenario")

	// Create required directories
	if err := os.MkdirAll(filepath.Join(scenarioDir, "test", "playbooks"), 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "requirements"), 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	// Create playbook with invalid reset value
	playbookContent := `{"metadata": {"reset": "partial"}, "nodes": []}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "playbooks", "invalid.json"), []byte(playbookContent), 0o644); err != nil {
		t.Fatalf("failed to write playbook: %v", err)
	}

	builder := NewBuilder(scenarioDir)
	_, err := builder.Build()
	if err == nil {
		t.Error("expected error for invalid reset value")
	}
}

func TestBuilderBuildDefaultResetValue(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "test-scenario")

	// Create required directories
	if err := os.MkdirAll(filepath.Join(scenarioDir, "test", "playbooks"), 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "requirements"), 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	// Create playbook without reset value
	playbookContent := `{"metadata": {"description": "No reset"}, "nodes": []}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "playbooks", "default.json"), []byte(playbookContent), 0o644); err != nil {
		t.Fatalf("failed to write playbook: %v", err)
	}

	builder := NewBuilder(scenarioDir)
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result.Registry.Playbooks) != 1 {
		t.Fatalf("expected 1 playbook, got %d", len(result.Registry.Playbooks))
	}

	if result.Registry.Playbooks[0].Reset != "none" {
		t.Errorf("expected default reset 'none', got %s", result.Registry.Playbooks[0].Reset)
	}
}

func TestBuilderBuildMultiplePlaybooks(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "test-scenario")

	// Create directories
	dirs := []string{
		filepath.Join(scenarioDir, "test", "playbooks", "capabilities", "01-first"),
		filepath.Join(scenarioDir, "test", "playbooks", "capabilities", "02-second"),
		filepath.Join(scenarioDir, "test", "playbooks", "journeys"),
		filepath.Join(scenarioDir, "requirements"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// Create playbooks
	playbooks := map[string]string{
		filepath.Join(scenarioDir, "test", "playbooks", "capabilities", "01-first", "a-test.json"):  `{"metadata": {"description": "A"}, "nodes": []}`,
		filepath.Join(scenarioDir, "test", "playbooks", "capabilities", "01-first", "b-test.json"):  `{"metadata": {"description": "B"}, "nodes": []}`,
		filepath.Join(scenarioDir, "test", "playbooks", "capabilities", "02-second", "c-test.json"): `{"metadata": {"description": "C"}, "nodes": []}`,
		filepath.Join(scenarioDir, "test", "playbooks", "journeys", "happy-path.json"):              `{"metadata": {"description": "Journey"}, "nodes": []}`,
	}
	for path, content := range playbooks {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write playbook: %v", err)
		}
	}

	builder := NewBuilder(scenarioDir)
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.PlaybookCount != 4 {
		t.Errorf("expected 4 playbooks, got %d", result.PlaybookCount)
	}

	// Verify ordering is based on file system order (alphabetical)
	if len(result.Registry.Playbooks) < 2 {
		t.Fatal("not enough playbooks")
	}

	// First playbook should be in 01-first directory
	first := result.Registry.Playbooks[0]
	if first.Description != "A" {
		t.Errorf("expected first playbook to be A, got %s", first.Description)
	}

	// Order should follow directory structure
	if first.Order != "01.01.01" {
		t.Errorf("expected order 01.01.01, got %s", first.Order)
	}
}

func TestBuilderExtractFixtures(t *testing.T) {
	builder := NewBuilder("")

	tests := []struct {
		name     string
		nodes    []playbookNode
		expected []string
	}{
		{
			name:     "no fixtures",
			nodes:    []playbookNode{{Data: nodeData{WorkflowID: ""}}},
			expected: []string{},
		},
		{
			name:     "single fixture",
			nodes:    []playbookNode{{Data: nodeData{WorkflowID: "@fixture/load-state"}}},
			expected: []string{"load-state"},
		},
		{
			name: "fixture with arguments",
			nodes: []playbookNode{
				{Data: nodeData{WorkflowID: "@fixture/setup(param=value)"}},
			},
			expected: []string{"setup"},
		},
		{
			name: "multiple fixtures",
			nodes: []playbookNode{
				{Data: nodeData{WorkflowID: "@fixture/first"}},
				{Data: nodeData{WorkflowID: "@fixture/second"}},
			},
			expected: []string{"first", "second"},
		},
		{
			name: "duplicate fixtures",
			nodes: []playbookNode{
				{Data: nodeData{WorkflowID: "@fixture/same"}},
				{Data: nodeData{WorkflowID: "@fixture/same"}},
			},
			expected: []string{"same"},
		},
		{
			name: "mixed nodes",
			nodes: []playbookNode{
				{Data: nodeData{WorkflowID: ""}},
				{Data: nodeData{WorkflowID: "@fixture/real"}},
				{Data: nodeData{WorkflowID: "not-a-fixture"}},
			},
			expected: []string{"real"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.extractFixtures(tc.nodes)
			if len(result) == 0 && len(tc.expected) == 0 {
				return
			}
			if len(result) != len(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
				return
			}
			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("at index %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestBuilderCollectRequirementValidations(t *testing.T) {
	tempDir := t.TempDir()
	requirementsDir := filepath.Join(tempDir, "requirements")

	// Create requirements structure
	if err := os.MkdirAll(filepath.Join(requirementsDir, "01-feature"), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	indexContent := `{"imports": ["01-feature/module.json"]}`
	if err := os.WriteFile(filepath.Join(requirementsDir, "index.json"), []byte(indexContent), 0o644); err != nil {
		t.Fatalf("failed to write index: %v", err)
	}

	moduleContent := `{
		"requirements": [
			{
				"id": "REQ-001",
				"validation": [
					{"ref": "test/playbooks/a.json"},
					{"ref": "test/unit/test.go"}
				]
			},
			{
				"id": "REQ-002",
				"validation": [
					{"ref": "test/playbooks/a.json"},
					{"ref": "test/playbooks/b.json"}
				]
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(requirementsDir, "01-feature", "module.json"), []byte(moduleContent), 0o644); err != nil {
		t.Fatalf("failed to write module: %v", err)
	}

	builder := NewBuilder(tempDir)
	result, err := builder.collectRequirementValidations(requirementsDir)
	if err != nil {
		t.Fatalf("collectRequirementValidations failed: %v", err)
	}

	// Check a.json has both requirements
	aReqs := result["test/playbooks/a.json"]
	if len(aReqs) != 2 {
		t.Errorf("expected 2 requirements for a.json, got %d", len(aReqs))
	}

	// Check b.json has one requirement
	bReqs := result["test/playbooks/b.json"]
	if len(bReqs) != 1 || bReqs[0] != "REQ-002" {
		t.Errorf("expected [REQ-002] for b.json, got %v", bReqs)
	}

	// Check non-playbook refs are ignored
	if _, exists := result["test/unit/test.go"]; exists {
		t.Error("non-playbook ref should be ignored")
	}
}

func BenchmarkBuilderBuild(b *testing.B) {
	tempDir := b.TempDir()
	scenarioDir := filepath.Join(tempDir, "bench-scenario")

	// Create structure with many playbooks
	if err := os.MkdirAll(filepath.Join(scenarioDir, "test", "playbooks", "capabilities"), 0o755); err != nil {
		b.Fatalf("failed to create dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "requirements"), 0o755); err != nil {
		b.Fatalf("failed to create dir: %v", err)
	}

	// Create 100 playbook files
	for i := 0; i < 100; i++ {
		content := `{"metadata": {"description": "Playbook"}, "nodes": []}`
		path := filepath.Join(scenarioDir, "test", "playbooks", "capabilities", filepath.Base(tempDir)+"-"+string(rune('a'+i%26))+".json")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatalf("failed to write playbook: %v", err)
		}
	}

	builder := NewBuilder(scenarioDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = builder.Build()
	}
}
