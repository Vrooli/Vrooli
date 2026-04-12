package orchestrator

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-11

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

func TestDetailReturnsTypedNotFoundError(t *testing.T) {
	service := New(t.TempDir(), t.TempDir(), io.Discard, io.Discard)

	_, err := service.Detail("missing")
	if err == nil {
		t.Fatal("expected missing scenario error")
	}
	if got := vroolierr.Code(err, ""); got != "scenario_not_found" {
		t.Fatalf("error code = %q, want scenario_not_found", got)
	}
}

func TestResolvePortFallsBackFromUIToAPI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioService(t, root, "alpha", "Alpha", "running")
	writeProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	service := New(root, home, io.Discard, io.Discard)
	resolved, err := service.ResolvePort("alpha", "UI_PORT")
	if err != nil {
		t.Fatalf("ResolvePort: %v", err)
	}
	if resolved.Name != "API_PORT" || resolved.Port != 18080 || resolved.URL != "http://localhost:18080" {
		t.Fatalf("resolved = %+v", resolved)
	}
}

func TestResolvePortRejectsStoppedScenario(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioService(t, root, "alpha", "Alpha", "stopped")

	service := New(root, home, io.Discard, io.Discard)
	_, err := service.ResolvePort("alpha", "API_PORT")
	if err == nil {
		t.Fatal("expected stopped scenario error")
	}
	if got := vroolierr.Code(err, ""); got != "scenario_not_running" {
		t.Fatalf("error code = %q, want scenario_not_running", got)
	}
}

func TestStartDetailedUsesInjectedRunnerFactory(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioService(t, root, "alpha", "Alpha", "running")
	writeProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	service := New(root, home, io.Discard, io.Discard)
	service.newRunner = func(root, home string, stdout, stderr io.Writer, logger ...*slog.Logger) (lifecycleRunner, error) {
		return fakeLifecycleRunner{
			startFn: func(name string, opts lifecycle.StartOptions) (lifecycle.Result, error) {
				return lifecycle.Result{
					Scenario: scenarioFixture(name, filepath.Join(root, "scenarios", name)),
					AllocatedPorts: map[string]int{
						"API_PORT": 18080,
					},
					Health: "healthy",
				}, nil
			},
		}, nil
	}

	result, err := service.StartDetailed("alpha", lifecycle.StartOptions{})
	if err != nil {
		t.Fatalf("StartDetailed: %v", err)
	}
	if result.View.Name != "alpha" {
		t.Fatalf("view name = %q", result.View.Name)
	}
}

type fakeLifecycleRunner struct {
	startFn   func(name string, opts lifecycle.StartOptions) (lifecycle.Result, error)
	restartFn func(name string, opts lifecycle.StartOptions) (lifecycle.Result, error)
	stopFn    func(name string, opts lifecycle.StopOptions) error
}

func (f fakeLifecycleRunner) Start(name string, opts lifecycle.StartOptions) (lifecycle.Result, error) {
	if f.startFn == nil {
		return lifecycle.Result{}, nil
	}
	return f.startFn(name, opts)
}

func (f fakeLifecycleRunner) Restart(name string, opts lifecycle.StartOptions) (lifecycle.Result, error) {
	if f.restartFn == nil {
		return lifecycle.Result{}, nil
	}
	return f.restartFn(name, opts)
}

func (f fakeLifecycleRunner) Stop(name string, opts lifecycle.StopOptions) error {
	if f.stopFn == nil {
		return nil
	}
	return f.stopFn(name, opts)
}

func scenarioFixture(name, path string) scenario.Scenario {
	return scenario.Scenario{
		Slug: name,
		Path: path,
		Manifest: scenario.ServiceManifest{
			Service: scenario.ServiceMetadata{
				Name: name,
			},
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT"},
			},
		},
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
