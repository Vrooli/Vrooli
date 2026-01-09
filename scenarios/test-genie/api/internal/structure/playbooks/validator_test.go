package playbooks

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidator_NoBasDir(t *testing.T) {
	root := t.TempDir()

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success when no bas dir, got error: %v", result.Error)
	}

	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationInfo && obs.Message == "No bas/ directory found (optional)" {
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
	mustMkdir(t, filepath.Join(root, "bas"))

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

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
	mustMkdir(t, filepath.Join(root, "bas"))

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      true,
	}, io.Discard)

	result := v.Validate()
	if result.Success {
		t.Fatal("expected failure in strict mode for missing registry")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestValidator_InvalidRegistryJSON(t *testing.T) {
	root := t.TempDir()
	basDir := filepath.Join(root, "bas")
	mustMkdir(t, basDir)

	writeFile(t, filepath.Join(basDir, "registry.json"), "not valid json {{{")

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

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

func TestValidator_CasesWithoutPrefix(t *testing.T) {
	root := t.TempDir()
	basDir := filepath.Join(root, "bas")
	mustMkdir(t, basDir)

	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(basDir, "registry.json"), registry)

	mustMkdir(t, filepath.Join(basDir, "cases", "foundation"))
	mustMkdir(t, filepath.Join(basDir, "cases", "01-builder"))
	mustMkdir(t, filepath.Join(basDir, "cases", "invalid-prefix"))

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && obs.Message == "bas/cases/ has directories without NN- prefix: foundation, invalid-prefix" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about directories without NN- prefix")
	}
}

func TestValidator_FlowsWithValidPrefix(t *testing.T) {
	root := t.TempDir()
	basDir := filepath.Join(root, "bas")
	mustMkdir(t, basDir)

	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(basDir, "registry.json"), registry)

	mustMkdir(t, filepath.Join(basDir, "flows", "01-new-user"))
	mustMkdir(t, filepath.Join(basDir, "flows", "02-power-user"))

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestValidator_SeedsMissingEntrypoint(t *testing.T) {
	root := t.TempDir()
	basDir := filepath.Join(root, "bas")
	mustMkdir(t, basDir)

	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(basDir, "registry.json"), registry)

	mustMkdir(t, filepath.Join(basDir, "seeds"))

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

	found := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning && obs.Message == "bas/seeds/ directory exists but missing seed.go or seed.sh" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about missing seed.go or seed.sh")
	}
}

func TestValidator_SeedsWithEntrypoint(t *testing.T) {
	root := t.TempDir()
	basDir := filepath.Join(root, "bas")
	mustMkdir(t, basDir)

	registry := `{"_note": "AUTO-GENERATED", "generated_at": "2025-12-05T10:00:00Z", "playbooks": []}`
	writeFile(t, filepath.Join(basDir, "registry.json"), registry)

	seedsDir := filepath.Join(basDir, "seeds")
	mustMkdir(t, seedsDir)
	writeFile(t, filepath.Join(seedsDir, "seed.go"), "package main\nfunc main() {}")

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
	basDir := filepath.Join(root, "bas")
	mustMkdir(t, basDir)

	registry := `{"_note": "AUTO-GENERATED", "generated_at": null, "playbooks": []}`
	writeFile(t, filepath.Join(basDir, "registry.json"), registry)

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      false,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success in non-strict mode, got error: %v", result.Error)
	}

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
	basDir := filepath.Join(root, "bas")
	mustMkdir(t, basDir)

	registry := `{
		"_note": "AUTO-GENERATED",
		"scenario": "test-scenario",
		"generated_at": "2025-12-05T10:00:00Z",
		"playbooks": [
			{
				"file": "bas/cases/01-foundation/ui/test.json",
				"description": "Test playbook",
				"order": "01.01.01",
				"requirements": ["REQ-001"],
				"fixtures": [],
				"reset": "none"
			}
		]
	}`
	writeFile(t, filepath.Join(basDir, "registry.json"), registry)

	mustMkdir(t, filepath.Join(basDir, "cases", "01-foundation", "ui"))
	mustMkdir(t, filepath.Join(basDir, "flows", "01-happy-path"))

	seedsDir := filepath.Join(basDir, "seeds")
	mustMkdir(t, seedsDir)
	writeFile(t, filepath.Join(seedsDir, "seed.go"), "package main\nfunc main() {}")

	v := New(Config{
		ScenarioDir: root,
		Enabled:     true,
		Strict:      true,
	}, io.Discard)

	result := v.Validate()
	if !result.Success {
		t.Fatalf("expected success for complete valid structure, got error: %v", result.Error)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

