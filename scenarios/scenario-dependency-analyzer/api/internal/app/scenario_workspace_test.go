package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	types "scenario-dependency-analyzer/internal/types"
)

func TestScenarioWorkspaceListingAndConfig(t *testing.T) {
	tempDir := t.TempDir()
	cfg := types.ServiceConfig{}
	cfg.Service.Name = "alpha"
	cfg.Service.DisplayName = "Alpha"
	cfg.Service.Description = "Alpha scenario"
	cfg.Service.Version = "1.0.0"
	cfg.Service.Tags = []string{"test"}

	writeServiceConfig(t, tempDir, "alpha", cfg)
	ws := &scenarioWorkspace{root: tempDir}

	names, err := ws.listScenarioNames()
	if err != nil {
		t.Fatalf("listScenarioNames returned error: %v", err)
	}
	if len(names) != 1 || names[0] != "alpha" {
		t.Fatalf("listScenarioNames = %v, want [alpha]", names)
	}

	loaded, err := ws.loadConfig("alpha")
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if loaded.Service.Name != "alpha" {
		t.Fatalf("loadConfig Service.Name = %s, want alpha", loaded.Service.Name)
	}
}

func TestScenarioWorkspaceMissingConfig(t *testing.T) {
	tempDir := t.TempDir()
	ws := &scenarioWorkspace{root: tempDir}

	if ws.hasServiceConfig("missing") {
		t.Fatalf("hasServiceConfig returned true for missing scenario")
	}
	if _, err := ws.loadConfig("missing"); err == nil {
		t.Fatalf("expected error loading missing service.json")
	}
}

func writeServiceConfig(t *testing.T, root, name string, cfg types.ServiceConfig) {
	t.Helper()
	serviceDir := filepath.Join(root, name, ".vrooli")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("failed to create service dir: %v", err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal service config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}
}
