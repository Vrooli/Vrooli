package main

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleServiceFile = `{
  "service": {
    "name": "demo-scenario",
    "displayName": "Demo Scenario",
    "description": "Test scenario",
    "tags": ["test", "demo"]
  },
  "dependencies": {
    "scenarios": {
      "upstream": {
        "required": true
      }
    },
    "resources": {
      "postgres": {
        "type": "postgres"
      }
    }
  }
}`

func TestScenarioCatalogRefresh(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioDir := filepath.Join(repoRoot, "scenarios", "demo-scenario", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed creating directories: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "service.json"), []byte(sampleServiceFile), 0o644); err != nil {
		t.Fatalf("failed writing service file: %v", err)
	}

	visibilityPath := filepath.Join(repoRoot, "visibility.json")
	manager, err := NewScenarioCatalogManager(repoRoot, visibilityPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	scenarios, edges, hidden, lastSynced := manager.Snapshot()
	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Name != "demo-scenario" {
		t.Fatalf("unexpected scenario name: %s", scenarios[0].Name)
	}
	if len(edges) != 1 || edges[0].To != "upstream" {
		t.Fatalf("expected dependency edge to upstream, got %+v", edges)
	}
	if len(hidden) != 0 {
		t.Fatalf("expected no hidden scenarios, got %v", hidden)
	}
	if lastSynced.IsZero() {
		t.Fatalf("expected last synced timestamp to be recorded")
	}
}

func TestScenarioVisibilityToggle(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioDir := filepath.Join(repoRoot, "scenarios", "demo", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed creating directories: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "service.json"), []byte(sampleServiceFile), 0o644); err != nil {
		t.Fatalf("failed writing service file: %v", err)
	}

	visibilityPath := filepath.Join(repoRoot, "visibility.json")
	manager, err := NewScenarioCatalogManager(repoRoot, visibilityPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if err := manager.UpdateVisibility("demo-scenario", true); err != nil {
		t.Fatalf("failed to hide scenario: %v", err)
	}

	scenarios, _, hidden, _ := manager.Snapshot()
	if len(hidden) != 1 {
		t.Fatalf("expected single hidden scenario, got %d", len(hidden))
	}
	if !scenarios[0].Hidden {
		t.Fatalf("expected catalog entry to be marked hidden")
	}

	if err := manager.UpdateVisibility("demo-scenario", false); err != nil {
		t.Fatalf("failed to unhide scenario: %v", err)
	}

	scenarios, _, hidden, _ = manager.Snapshot()
	if len(hidden) != 0 {
		t.Fatalf("expected hidden list cleared, got %v", hidden)
	}
	if scenarios[0].Hidden {
		t.Fatalf("expected catalog entry to be visible")
	}
}
