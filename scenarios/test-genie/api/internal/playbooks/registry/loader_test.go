package registry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoadSuccess(t *testing.T) {
	scenarioDir := t.TempDir()
	setupValidRegistry(t, scenarioDir)

	loader := NewLoader(scenarioDir)
	registry, err := loader.Load()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(registry.Playbooks) != 2 {
		t.Fatalf("expected 2 playbooks, got %d", len(registry.Playbooks))
	}

	// Verify sorting by order
	if registry.Playbooks[0].File != "bas/cases/first.json" {
		t.Errorf("expected first.json first, got %s", registry.Playbooks[0].File)
	}
	if registry.Playbooks[1].File != "bas/cases/second.json" {
		t.Errorf("expected second.json second, got %s", registry.Playbooks[1].File)
	}
}

func TestLoaderLoadNotFound(t *testing.T) {
	scenarioDir := t.TempDir()
	// Don't create the registry file

	loader := NewLoader(scenarioDir)
	_, err := loader.Load()
	if err == nil {
		t.Fatal("expected error for missing registry")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestLoaderLoadInvalidJSON(t *testing.T) {
	scenarioDir := t.TempDir()
	playbooksDir := filepath.Join(scenarioDir, FolderName)
	if err := os.MkdirAll(playbooksDir, 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playbooksDir, RegistryFileName), []byte(`{invalid`), 0o644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}

	loader := NewLoader(scenarioDir)
	_, err := loader.Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !errors.Is(err, ErrParse) {
		t.Errorf("expected ErrParse, got: %v", err)
	}
}

func TestLoaderLoadDeprecatedFallback(t *testing.T) {
	scenarioDir := t.TempDir()
	playbooksDir := filepath.Join(scenarioDir, FolderName)
	if err := os.MkdirAll(playbooksDir, 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}

	content := `{
		"scenario": "test",
		"generated_at": "2024-01-01",
		"playbooks": [],
		"deprecated_playbooks": [
			{"file": "bas/cases/legacy.json", "description": "Legacy", "order": "1"}
		]
	}`
	if err := os.WriteFile(filepath.Join(playbooksDir, RegistryFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}

	loader := NewLoader(scenarioDir)
	registry, err := loader.Load()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(registry.Playbooks) != 1 {
		t.Fatalf("expected 1 playbook from deprecated fallback, got %d", len(registry.Playbooks))
	}
	if registry.Playbooks[0].File != "bas/cases/legacy.json" {
		t.Errorf("expected legacy.json, got %s", registry.Playbooks[0].File)
	}
}

func TestLoaderLoadSortsByOrderThenFile(t *testing.T) {
	scenarioDir := t.TempDir()
	playbooksDir := filepath.Join(scenarioDir, FolderName)
	if err := os.MkdirAll(playbooksDir, 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}

	// Create registry with unsorted entries
	content := `{
		"scenario": "test",
		"generated_at": "2024-01-01",
		"playbooks": [
			{"file": "c.json", "description": "C", "order": "2"},
			{"file": "a.json", "description": "A", "order": "1"},
			{"file": "b.json", "description": "B", "order": "1"},
			{"file": "d.json", "description": "D", "order": "2"}
		]
	}`
	if err := os.WriteFile(filepath.Join(playbooksDir, RegistryFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}

	loader := NewLoader(scenarioDir)
	registry, err := loader.Load()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	expected := []string{"a.json", "b.json", "c.json", "d.json"}
	for i, want := range expected {
		if registry.Playbooks[i].File != want {
			t.Errorf("position %d: expected %s, got %s", i, want, registry.Playbooks[i].File)
		}
	}
}

func TestLoaderRegistryPath(t *testing.T) {
	loader := NewLoader("/test/dir")
	expected := "/test/dir/bas/registry.json"
	if got := loader.RegistryPath(); got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestLoaderLoadWithRequirements(t *testing.T) {
	scenarioDir := t.TempDir()
	playbooksDir := filepath.Join(scenarioDir, FolderName)
	if err := os.MkdirAll(playbooksDir, 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}

	content := `{
		"scenario": "test",
		"playbooks": [
			{
				"file": "login.json",
				"description": "Login flow",
				"order": "1",
				"requirements": ["REQ-001", "REQ-002"],
				"fixtures": ["user-data"],
				"reset": "soft"
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(playbooksDir, RegistryFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}

	loader := NewLoader(scenarioDir)
	registry, err := loader.Load()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	entry := registry.Playbooks[0]
	if len(entry.Requirements) != 2 {
		t.Errorf("expected 2 requirements, got %d", len(entry.Requirements))
	}
	if entry.Requirements[0] != "REQ-001" {
		t.Errorf("expected REQ-001, got %s", entry.Requirements[0])
	}
	if len(entry.Fixtures) != 1 || entry.Fixtures[0] != "user-data" {
		t.Errorf("unexpected fixtures: %v", entry.Fixtures)
	}
	if entry.Reset != "soft" {
		t.Errorf("expected reset=soft, got %s", entry.Reset)
	}
}

func setupValidRegistry(t *testing.T, testDir string) {
	t.Helper()
	playbooksDir := filepath.Join(testDir, FolderName)
	if err := os.MkdirAll(playbooksDir, 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}

	content := `{
		"scenario": "demo",
		"generated_at": "2024-01-01T00:00:00Z",
		"playbooks": [
			{"file": "bas/cases/second.json", "description": "Second test", "order": "2"},
			{"file": "bas/cases/first.json", "description": "First test", "order": "1"}
		]
	}`
	if err := os.WriteFile(filepath.Join(playbooksDir, RegistryFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}
}
