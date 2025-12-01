package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	env := PhaseEnvironment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	report := runUnitPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success, got %v", report.Err)
	}
	if len(commands) != 4 {
		t.Fatalf("expected four commands, got %d", len(commands))
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

	env := PhaseEnvironment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	report := runUnitPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected error when go missing")
	}
	if report.FailureClassification != failureClassMissingDependency {
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

	env := PhaseEnvironment{
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

func TestRunUnitPhaseFailsWhenShellScriptMissing(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	target := filepath.Join(scenarioDir, "test", "lib", "runtime.sh")
	if err := os.Remove(target); err != nil {
		t.Fatalf("failed to remove runtime script: %v", err)
	}
	stubCommandLookup(t, func(name string) (string, error) {
		return "/tmp/" + name, nil
	})
	stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		return nil
	})

	env := PhaseEnvironment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}
	report := runUnitPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected error when runtime script missing")
	}
	if report.FailureClassification != failureClassMisconfiguration {
		t.Fatalf("expected misconfiguration classification, got %s", report.FailureClassification)
	}
}
