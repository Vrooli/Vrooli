package phases

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"test-genie/internal/lint/execution"
	"test-genie/internal/orchestrator/workspace"
	"testing"
)

type stubLintRunner struct {
	byName map[string]execution.Result
}

func (s stubLintRunner) Run(_ context.Context, cmd execution.Command) (execution.Result, error) {
	if result, ok := s.byName[filepath.Base(cmd.Name)]; ok {
		return result, nil
	}
	return execution.Result{}, errors.New("unexpected command")
}

func TestRunLintPhaseWithGoComponent(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to seed go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to seed main.go: %v", err)
	}

	restoreLookup := OverrideCommandLookup(func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	defer restoreLookup()

	restoreRunner := OverrideLintCommandRunner(stubLintRunner{
		byName: map[string]execution.Result{
			"golangci-lint": {Stdout: []byte(`{"Issues":[]}`), ExitCode: 0},
		},
	})
	defer restoreRunner()

	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: scenarioDir, TestDir: filepath.Join(scenarioDir, "test")}
	report := runLintPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("lint phase failed unexpectedly: %v", report.Err)
	}
}

func TestRunLintPhaseNoLintableComponents(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{
  "lint": {
    "policy": {
      "unconfigured_common_components": {
        "api": "ignore",
        "ui": "ignore",
        "cli": "ignore"
      },
      "unmatched_code_components": "ignore"
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	restoreLookup := OverrideCommandLookup(func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	defer restoreLookup()
	restoreRunner := OverrideLintCommandRunner(stubLintRunner{})
	defer restoreRunner()

	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: scenarioDir, TestDir: filepath.Join(scenarioDir, "test")}
	report := runLintPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("lint phase should succeed when no lintable components: %v", report.Err)
	}

	found := false
	for _, obs := range report.Observations {
		if strings.Contains(obs.Text, "0 component(s) linted") || strings.Contains(obs.Text, "No lintable top-level components") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected no-components observation, got %+v", report.Observations)
	}
}

func TestRunLintPhaseCommonUnconfiguredUIWarns(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{
  "lint": {
    "policy": {
      "unconfigured_common_components": {
        "api": "ignore",
        "ui": "warning",
        "cli": "ignore"
      },
      "unmatched_code_components": "warning"
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "ui", "main.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed ui file: %v", err)
	}

	restoreRunner := OverrideLintCommandRunner(stubLintRunner{})
	defer restoreRunner()

	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: scenarioDir, TestDir: filepath.Join(scenarioDir, "test")}
	report := runLintPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("lint phase should warn, not fail, for unmatched ui: %v", report.Err)
	}
}

func TestRunLintPhaseWithCancelledContext(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: scenarioDir, TestDir: filepath.Join(scenarioDir, "test")}
	report := runLintPhase(ctx, env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure for cancelled context")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("expected system classification, got %s", report.FailureClassification)
	}
}

func TestRunLintPhaseMultipleComponents(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to seed go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to seed main.go: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "worker"), 0o755); err != nil {
		t.Fatalf("failed to create worker dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "worker", "package.json"), []byte(`{"name":"worker"}`), 0o644); err != nil {
		t.Fatalf("failed to seed package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "worker", "eslint.config.js"), []byte(`export default [];`), 0o644); err != nil {
		t.Fatalf("failed to seed eslint config: %v", err)
	}

	restoreLookup := OverrideCommandLookup(func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	defer restoreLookup()
	restoreRunner := OverrideLintCommandRunner(stubLintRunner{
		byName: map[string]execution.Result{
			"golangci-lint": {Stdout: []byte(`{"Issues":[]}`), ExitCode: 0},
			"eslint":        {Stdout: []byte(`[]`), ExitCode: 0},
		},
	})
	defer restoreRunner()

	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: scenarioDir, TestDir: filepath.Join(scenarioDir, "test")}
	report := runLintPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("unexpected error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatal("expected observations to be recorded")
	}
}

func TestRunLintPhaseUsesComponentOverridesAndRootDiscovery(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{
  "lint": {
    "policy": {
      "unconfigured_common_components": {
        "api": "ignore",
        "ui": "warning",
        "cli": "ignore"
      }
    },
    "components": {
      "worker": {
        "handler": "node_package",
        "strict": true
      }
    },
    "ignore": ["docs"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "pyproject.toml"), []byte("[project]\nname='demo'\nversion='0.1.0'\n"), 0o644); err != nil {
		t.Fatalf("failed to seed pyproject.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "main.py"), []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("failed to seed main.py: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "worker"), 0o755); err != nil {
		t.Fatalf("failed to create worker dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "worker", "package.json"), []byte(`{"name":"worker"}`), 0o644); err != nil {
		t.Fatalf("failed to seed worker package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "worker", "eslint.config.js"), []byte(`export default [];`), 0o644); err != nil {
		t.Fatalf("failed to seed worker eslint config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "ui", "main.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed unmatched ui file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "docs", "helper.go"), []byte("package docs\n"), 0o644); err != nil {
		t.Fatalf("failed to seed ignored docs file: %v", err)
	}

	restoreLookup := OverrideCommandLookup(func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	defer restoreLookup()
	restoreRunner := OverrideLintCommandRunner(stubLintRunner{
		byName: map[string]execution.Result{
			"ruff":   {Stdout: []byte(`[]`), ExitCode: 0},
			"eslint": {Stdout: []byte(`[]`), ExitCode: 0},
		},
	})
	defer restoreRunner()

	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: scenarioDir, TestDir: filepath.Join(scenarioDir, "test")}
	report := runLintPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("unexpected lint phase error: %v", report.Err)
	}

	var sawRootPython, sawWorkerOverride, sawUIWarning bool
	for _, obs := range report.Observations {
		combined := obs.Section + " " + obs.Text
		switch {
		case strings.Contains(combined, "Linting . (python_project)"):
			sawRootPython = true
		case strings.Contains(combined, "Linting worker (node_package)"):
			sawWorkerOverride = true
		case strings.Contains(combined, "ui: common component is present without a supported lint contract"):
			sawUIWarning = true
		case strings.Contains(combined, "docs:"):
			t.Fatalf("ignored docs component should not appear in observations: %+v", report.Observations)
		}
	}

	if !sawRootPython {
		t.Fatalf("expected root python component observation, got %+v", report.Observations)
	}
	if !sawWorkerOverride {
		t.Fatalf("expected worker override observation, got %+v", report.Observations)
	}
	if !sawUIWarning {
		t.Fatalf("expected unmatched ui policy warning, got %+v", report.Observations)
	}
}
