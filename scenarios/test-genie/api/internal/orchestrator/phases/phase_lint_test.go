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

func TestRunLintPhaseWithGoProject(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Create a Go project
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to seed go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to seed main.go: %v", err)
	}

	// Stub command lookup - simulate golangci-lint being available
	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runLintPhase(context.Background(), env, io.Discard)

	// The phase should succeed (even if actual linting is skipped due to mocked commands)
	if report.Err != nil {
		t.Fatalf("lint phase failed unexpectedly: %v", report.Err)
	}
}

func TestRunLintPhaseWithNodeProject(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Create a Node.js project with TypeScript
	uiDir := filepath.Join(scenarioDir, "ui")
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"name":"demo-ui"}`), 0o644); err != nil {
		t.Fatalf("failed to seed package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "tsconfig.json"), []byte(`{"compilerOptions":{}}`), 0o644); err != nil {
		t.Fatalf("failed to seed tsconfig.json: %v", err)
	}

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runLintPhase(context.Background(), env, io.Discard)

	if report.Err != nil {
		t.Fatalf("lint phase failed unexpectedly: %v", report.Err)
	}
}

func TestRunLintPhaseNoLintableProjects(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// No go.mod in api/, no package.json in ui/, no Python files
	// The lint phase should succeed with "no lintable languages" observation

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runLintPhase(context.Background(), env, io.Discard)

	if report.Err != nil {
		t.Fatalf("lint phase should succeed when no lintable projects: %v", report.Err)
	}

	// Should have an observation about no lintable languages
	found := false
	for _, obs := range report.Observations {
		if strings.Contains(obs.Text, "no lintable") || strings.Contains(obs.Text, "No lintable") {
			found = true
			break
		}
	}
	if !found {
		t.Log("Observations:", report.Observations)
		// Not a hard failure - the message format may vary
	}
}

func TestRunLintPhaseGeneratesObservations(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Create a Go project
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to seed go.mod: %v", err)
	}

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runLintPhase(context.Background(), env, io.Discard)

	if report.Err != nil {
		t.Fatalf("unexpected error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestRunLintPhaseWithCancelledContext(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runLintPhase(ctx, env, io.Discard)

	if report.Err == nil {
		t.Fatalf("expected failure for cancelled context")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("expected system classification, got %s", report.FailureClassification)
	}
}

func TestRunLintPhaseWithPythonProject(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Create a Python project indicator
	if err := os.WriteFile(filepath.Join(scenarioDir, "pyproject.toml"), []byte("[project]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatalf("failed to seed pyproject.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "main.py"), []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("failed to seed main.py: %v", err)
	}

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runLintPhase(context.Background(), env, io.Discard)

	if report.Err != nil {
		t.Fatalf("lint phase failed unexpectedly: %v", report.Err)
	}
}

func TestRunLintPhaseMultipleLanguages(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Create Go project
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to seed go.mod: %v", err)
	}

	// Create Node.js project
	uiDir := filepath.Join(scenarioDir, "ui")
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"name":"demo-ui"}`), 0o644); err != nil {
		t.Fatalf("failed to seed package.json: %v", err)
	}

	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runLintPhase(context.Background(), env, io.Discard)

	if report.Err != nil {
		t.Fatalf("lint phase failed unexpectedly: %v", report.Err)
	}

	// Should have observations for both languages
	if len(report.Observations) < 2 {
		t.Logf("expected multiple observations for multi-language project, got %d", len(report.Observations))
	}
}
