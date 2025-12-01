package phases

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"test-genie/internal/orchestrator/workspace"
)

func TestRunPerformancePhaseBuildsGoBinary(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	stubCommandLookup(t, func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	})

	var invoked bool
	stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		if name == "go" && len(args) >= 1 && args[0] == "build" {
			invoked = true
		}
		return nil
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runPerformancePhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("performance phase failed: %v", report.Err)
	}
	if !invoked {
		t.Fatalf("expected go build invocation")
	}
}

func TestRunPerformancePhaseFailsWhenDurationExceedsBudget(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	stubCommandLookup(t, func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	})

	stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	prev := performanceMaxDuration
	performanceMaxDuration = 1 * time.Millisecond
	t.Cleanup(func() {
		performanceMaxDuration = prev
	})

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runPerformancePhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure when go build exceeds budget")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("expected system failure classification, got %s", report.FailureClassification)
	}
}
