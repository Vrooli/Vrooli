package playbooks

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidator_NoPlaybooksDir(t *testing.T) {
	root := t.TempDir()

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success when no playbooks dir, got error: %v", result.Error)
	}

	// Should have informational observation
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationInfo && obs.Message == "No test/playbooks/ directory found (optional)" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected informational observation about missing directory")
	}
}

func TestValidator_Disabled(t *testing.T) {
	root := t.TempDir()

	v := New(Config{
		ScenarioDir: root,
		Enabled:     false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success when disabled, got error: %v", result.Error)
	}

	// Should have disabled observation
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationInfo && obs.Message == "Playbooks validation disabled" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected disabled observation")
	}
}

func TestValidator_MissingRegistry(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	// Non-strict mode: should succeed but have warnings
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	// Should have warning about missing registry
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && obs.Message == "Missing registry.json - run 'test-genie registry build'" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about missing registry.json")
	}
}

func TestValidator_MissingRegistry_Strict(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      true,
	}, io.Discard)

	result := v.Validate()
	// Strict mode: should fail
	if result.Success {
		t.Fatal("expected failure in strict mode for missing registry")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestValidator_InvalidRegistryJSON(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create invalid JSON
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), "not valid json {{{")

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	// Should have warning about invalid JSON
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about invalid JSON")
	}
}

func TestValidator_ValidRegistry(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{
		"_note": "AUTO-GENERATED",
		"scenario": "test-scenario",
		"generated_at": "2025-12-05T10:00:00Z",
		"playbooks": []
	}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestValidator_CapabilitiesWithoutPrefix(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	// Create capabilities with invalid directory names
	mustMkdir(t, filepath.Join(playbooksDir, "capabilities", "foundation"))     // Missing NN- prefix
	mustMkdir(t, filepath.Join(playbooksDir, "capabilities", "01-builder"))     // Valid
	mustMkdir(t, filepath.Join(playbooksDir, "capabilities", "invalid-prefix")) // Missing NN- prefix

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	// Should have warning about invalid prefixes
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning {
			t.Logf("Warning: %s", obs.Message)
			if obs.Message == "capabilities/ has directories without NN- prefix: foundation, invalid-prefix" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected warning about directories without NN- prefix")
	}
}

func TestValidator_JourneysWithValidPrefix(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	// Create journeys with valid directory names
	mustMkdir(t, filepath.Join(playbooksDir, "journeys", "01-new-user"))
	mustMkdir(t, filepath.Join(playbooksDir, "journeys", "02-power-user"))

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestValidator_SubflowsMissingFixtureID(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	// Create __subflows directory with fixtures
	subflowsDir := filepath.Join(playbooksDir, "__subflows")
	mustMkdir(t, subflowsDir)

	// Fixture WITHOUT fixture_id
	invalidFixture := `{"metadata": {"description": "Missing fixture_id"}, "nodes": [], "edges": []}`
	writeFile(t, filepath.Join(subflowsDir, "invalid-fixture.json"), invalidFixture)

	// Fixture WITH fixture_id
	validFixture := `{"metadata": {"fixture_id": "valid-fixture", "description": "Has fixture_id"}, "nodes": [], "edges": []}`
	writeFile(t, filepath.Join(subflowsDir, "valid-fixture.json"), validFixture)

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	// Should have warning about missing fixture_id
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && obs.Message == "__subflows/ fixtures missing fixture_id: invalid-fixture.json" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about missing fixture_id")
	}
}

func TestValidator_SubflowsAllValid(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	// Create __subflows directory with valid fixtures
	subflowsDir := filepath.Join(playbooksDir, "__subflows")
	mustMkdir(t, subflowsDir)

	fixture := `{"metadata": {"fixture_id": "my-fixture", "description": "Valid"}, "nodes": [], "edges": []}`
	writeFile(t, filepath.Join(subflowsDir, "my-fixture.json"), fixture)

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	// Should have success observation
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationSuccess && obs.Message == "Playbooks structure valid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected success observation")
	}
}

func TestValidator_SeedsMissingApply(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	// Create __seeds directory without apply.sh
	seedsDir := filepath.Join(playbooksDir, "__seeds")
	mustMkdir(t, seedsDir)
	writeFile(t, filepath.Join(seedsDir, "cleanup.sh"), "#!/bin/bash\necho cleanup")

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	// Should have warning about missing apply.sh
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && obs.Message == "__seeds/ directory exists but missing apply.sh" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about missing apply.sh")
	}
}

func TestValidator_SeedsWithApply(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	// Create __seeds directory with apply.sh
	seedsDir := filepath.Join(playbooksDir, "__seeds")
	mustMkdir(t, seedsDir)
	writeFile(t, filepath.Join(seedsDir, "apply.sh"), "#!/bin/bash\necho apply")
	writeFile(t, filepath.Join(seedsDir, "cleanup.sh"), "#!/bin/bash\necho cleanup")

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestValidator_StaleRegistry(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create registry with null generated_at
	registry := `{"_note": "AUTO-GENERATED", "generated_at": null, "playbooks": []}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	// Should have warning about stale registry
	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && obs.Message == "Registry may be stale (no generated_at) - run 'test-genie registry build'" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about stale registry")
	}
}

func TestValidator_CompleteValidStructure(t *testing.T) {
	root := t.TempDir()
	playbooksDir := filepath.Join(root, "test", "playbooks")
	mustMkdir(t, playbooksDir)

	// Create valid registry
	registry := `{
		"_note": "AUTO-GENERATED",
		"scenario": "test-scenario",
		"generated_at": "2025-12-05T10:00:00Z",
		"playbooks": [
			{
				"file": "test/playbooks/capabilities/01-foundation/01-core/test.json",
				"description": "Test playbook",
				"order": "01.01.01",
				"requirements": ["REQ-001"],
				"fixtures": [],
				"reset": "none"
			}
		]
	}`
	writeFile(t, filepath.Join(playbooksDir, "registry.json"), registry)

	// Create valid capabilities structure
	mustMkdir(t, filepath.Join(playbooksDir, "capabilities", "01-foundation", "01-core"))

	// Create valid journeys structure
	mustMkdir(t, filepath.Join(playbooksDir, "journeys", "01-happy-path"))

	// Create valid __subflows
	subflowsDir := filepath.Join(playbooksDir, "__subflows")
	mustMkdir(t, subflowsDir)
	fixture := `{"metadata": {"fixture_id": "setup", "description": "Setup fixture"}, "nodes": [], "edges": []}`
	writeFile(t, filepath.Join(subflowsDir, "setup.json"), fixture)

	// Create valid __seeds
	seedsDir := filepath.Join(playbooksDir, "__seeds")
	mustMkdir(t, seedsDir)
	writeFile(t, filepath.Join(seedsDir, "apply.sh"), "#!/bin/bash\necho apply")
	writeFile(t, filepath.Join(seedsDir, "cleanup.sh"), "#!/bin/bash\necho cleanup")

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      true, // Even in strict mode, this should pass
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success for complete valid structure, got error: %v", result.Error)
	}
}

// Helper functions

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
