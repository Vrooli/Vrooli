package project

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

func TestStatusAggregatesResourcesAndScenarios(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeProjectService(t, root, `{
  "service": { "name": "project-alpha" },
  "dependencies": {
    "resources": {
      "redis": { "enabled": true }
    }
  }
}`)
	writeScenarioService(t, root, "alpha")
	writeResourceCLI(t, root, "redis", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)
	writeScenarioProcess(t, home, "alpha", 18081)

	controller := New(root, home, io.Discard, io.Discard)
	report, err := controller.Status(StatusOptions{Fast: true})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if got := report.Summary["resources_running"]; got != 1 {
		t.Fatalf("resources_running = %d, want 1", got)
	}
	if got := report.Summary["scenarios_running"]; got != 1 {
		t.Fatalf("scenarios_running = %d, want 1", got)
	}
	if len(report.Resources) != 1 || report.Resources[0].Resource.Name != "redis" {
		t.Fatalf("resources = %#v", report.Resources)
	}
	if len(report.Scenarios) != 1 || report.Scenarios[0].Name != "alpha" {
		t.Fatalf("scenarios = %#v", report.Scenarios)
	}
}

func TestRunProjectPhaseRejectsUndefinedPhase(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeProjectService(t, root, `{
  "service": { "name": "project-alpha" },
  "lifecycle": {
    "develop": {
      "steps": [{ "name": "noop", "run": "true" }]
    }
  }
}`)

	controller := New(root, home, io.Discard, io.Discard)
	if err := controller.RunProjectPhase("deploy", nil); err == nil {
		t.Fatal("expected undefined phase to fail")
	}
}

func writeProjectService(t *testing.T, root, contents string) {
	t.Helper()
	path := filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioService(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "service": { "name": "` + name + `", "displayName": "` + name + `" },
  "ports": {
    "api": { "env_var": "API_PORT", "range": "18080-18090" }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeResourceCLI(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	path := filepath.Join(root, "resources", name, "cli.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"status\" ]]; then\n  printf '%s\\n' '" + statusJSON + "'\n  exit 0\nfi\nprintf '{\"message\":\"ok\"}\\n'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioProcess(t *testing.T, home, name string, port int) {
	t.Helper()
	record := process.Record{
		PID:       os.Getpid(),
		PGID:      os.Getpid(),
		Scenario:  name,
		Step:      "start-api",
		Port:      port,
		StartedAt: time.Now().Add(-time.Minute).UTC(),
		Status:    "running",
	}
	if err := process.WriteScenarioRecord(home, name, "start-api", record); err != nil {
		t.Fatalf("WriteScenarioRecord: %v", err)
	}
}
