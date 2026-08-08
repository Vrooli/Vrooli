package phases

import (
	"os"
	"path/filepath"
	"testing"
)

func createScenarioLayout(t *testing.T, root, name string) string {
	t.Helper()
	scenarioDir := filepath.Join(root, name)
	requiredDirs := []string{
		"api",
		"cli",
		filepath.Join("cli", "test"),
		"requirements",
		"ui",
		"docs",
		"test",
		".vrooli",
	}
	for _, rel := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", rel, err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"), []byte("module "+name+"/cli\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("failed to seed scenario CLI module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed scenario CLI source: %v", err)
	}
	serviceJSON := `{
  "service": {"name":"` + name + `"},
  "cli": {
    "enabled": true,
    "command": "` + name + `",
    "adapter": {
      "kind": "go_module",
      "module_dir": "cli"
    },
    "source_build": {"kind":"go_module"},
    "invoke": {"kind":"installed_command","command":"` + name + `"},
    "freshness": {"inputs":["cli/**", ".vrooli/service.json"]}
  }
}`
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatalf("failed to seed service.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{"descriptor-key":{"ui_smoke":{"enabled":false}}}`), 0o644); err != nil {
		t.Fatalf("failed to seed testing.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(`{"imports":["01-internal-orchestrator/module.json"]}`), 0o644); err != nil {
		t.Fatalf("failed to seed requirements index: %v", err)
	}
	moduleDir := filepath.Join(scenarioDir, "requirements", "01-internal-orchestrator")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}
	modulePayload := `{"requirements":[{"id":"REQ-1","title":"Seed","criticality":"p1","status":"draft","validation":[{"type":"manual","ref":"docs"}]}]}`
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), []byte(modulePayload), 0o644); err != nil {
		t.Fatalf("failed to seed module.json: %v", err)
	}
	return scenarioDir
}
