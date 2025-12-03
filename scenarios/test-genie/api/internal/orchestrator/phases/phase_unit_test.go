package phases

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestRunUnitPhaseSuccess(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	var commands []string
	stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	report := runUnitPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success, got %v", report.Err)
	}
	// Expect 2 commands: "go test ./..." and "bash -n <cli-binary>"
	if len(commands) != 2 {
		t.Fatalf("expected two commands, got %d: %v", len(commands), commands)
	}
	if commands[0] != "go test ./..." {
		t.Fatalf("unexpected first command: %s", commands[0])
	}
}

func TestRunUnitPhaseFailsWhenGoMissing(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	stubCommandLookup(t, func(name string) (string, error) {
		if name == "go" {
			return "", errors.New("not found")
		}
		return "/tmp/" + name, nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	report := runUnitPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected error when go missing")
	}
	if report.FailureClassification != FailureClassMissingDependency {
		t.Fatalf("expected missing dependency classification, got %s", report.FailureClassification)
	}
}

func TestRunUnitPhaseFailsWhenGoTestFails(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		if name == "go" {
			return errors.New("go test failed")
		}
		return nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	report := runUnitPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected go test failure to propagate")
	}
	if !strings.Contains(report.Err.Error(), "go test") {
		t.Fatalf("unexpected error: %v", report.Err)
	}
}

func TestRunUnitPhaseSkipsMissingShellTargets(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	// Remove CLI binary to test graceful handling of missing shell targets
	if err := os.Remove(filepath.Join(scenarioDir, "cli", "demo")); err != nil {
		t.Fatalf("failed to remove cli/demo: %v", err)
	}
	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		return nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	report := runUnitPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success despite missing shell targets: %v", report.Err)
	}
	found := false
	for _, obs := range report.Observations {
		if strings.Contains(obs.Text, "no shell entrypoints") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected observation about missing shell entrypoints, got %v", report.Observations)
	}
}
