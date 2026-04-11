package orchestrator

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

func TestListAndStatusReflectRuntimeRecords(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeScenarioService(t, root, "alpha", "Alpha", "running")
	writeScenarioService(t, root, "beta", "Beta", "stopped")
	writeProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	service := New(root, home, io.Discard, io.Discard)

	items, err := service.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(items))
	}

	alpha, exists, err := service.Status("alpha")
	if err != nil {
		t.Fatalf("Status(alpha): %v", err)
	}
	if !exists {
		t.Fatal("expected alpha to exist")
	}
	if alpha.Status != "running" || alpha.Processes != 1 {
		t.Fatalf("alpha status = %+v", alpha)
	}
	if alpha.Ports["API_PORT"] != 18080 {
		t.Fatalf("alpha ports = %#v", alpha.Ports)
	}

	running, err := service.Running()
	if err != nil {
		t.Fatalf("Running: %v", err)
	}
	if len(running) != 1 || running[0].Name != "alpha" {
		t.Fatalf("running = %#v", running)
	}
}

func writeScenarioService(t *testing.T, root, name, displayName, description string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "service": {
    "name": "` + name + `",
    "displayName": "` + displayName + `",
    "description": "` + description + `"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "18080-18090"
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeProcessRecord(t *testing.T, home, scenarioName, step string, pid, port int, startedAt time.Time) {
	t.Helper()
	record := process.Record{
		PID:       pid,
		PGID:      pid,
		Scenario:  scenarioName,
		Step:      step,
		Port:      port,
		StartedAt: startedAt.UTC(),
		Status:    "running",
	}
	if err := process.WriteScenarioRecord(home, scenarioName, step, record); err != nil {
		t.Fatalf("WriteScenarioRecord: %v", err)
	}
}
