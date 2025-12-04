package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func stubCommandLookup(t *testing.T, stub func(string) (string, error)) {
	t.Helper()
	prev := commandLookup
	commandLookup = stub
	t.Cleanup(func() {
		commandLookup = prev
	})
}

func TestRunDependenciesPhaseDetectsRuntimesAndManagers(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to seed go.mod: %v", err)
	}
	uiDir := filepath.Join(scenarioDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"name":"demo-ui"}`), 0o644); err != nil {
		t.Fatalf("failed to seed package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "pnpm-lock.yaml"), []byte("lockfileVersion: '9'\n"), 0o644); err != nil {
		t.Fatalf("failed to seed pnpm lock file: %v", err)
	}

	lookups := make(map[string]int)
	stubCommandLookup(t, func(name string) (string, error) {
		lookups[name]++
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runDependenciesPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("dependencies phase failed: %v", report.Err)
	}

	expected := []string{"bash", "curl", "jq", "go", "node", "pnpm"}
	for _, cmd := range expected {
		if lookups[cmd] == 0 {
			t.Fatalf("expected lookup for %s", cmd)
		}
	}
}

func TestRunDependenciesPhaseFailsWhenRuntimeMissing(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to seed go.mod: %v", err)
	}

	stubCommandLookup(t, func(name string) (string, error) {
		if name == "go" {
			return "", &commandNotFoundError{name}
		}
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runDependenciesPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure when go runtime is unavailable")
	}
	if !strings.Contains(report.Err.Error(), "go") {
		t.Fatalf("unexpected error: %v", report.Err)
	}
	if report.FailureClassification != FailureClassMissingDependency {
		t.Fatalf("expected missing dependency classification, got %s", report.FailureClassification)
	}
}

func TestRunDependenciesPhaseFailsOnMissingBaselineCommand(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	stubCommandLookup(t, func(name string) (string, error) {
		if name == "jq" {
			return "", &commandNotFoundError{name}
		}
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runDependenciesPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure when baseline command is unavailable")
	}
	if !strings.Contains(report.Err.Error(), "jq") {
		t.Fatalf("expected error to mention jq, got: %v", report.Err)
	}
}

func TestRunDependenciesPhaseGeneratesObservations(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runDependenciesPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("unexpected error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestRunDependenciesPhaseWithCancelledContext(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runDependenciesPhase(ctx, env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure for cancelled context")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("expected system classification, got %s", report.FailureClassification)
	}
}

// commandNotFoundError simulates exec.LookPath error.
type commandNotFoundError struct {
	name string
}

func (e *commandNotFoundError) Error() string {
	return "executable file not found in $PATH: " + e.name
}
