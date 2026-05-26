package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/smoke"
)

func stubSmokeRunForPhase(t *testing.T, fn func(ctx context.Context, scenarioName, scenarioDir, uiURL, runID string, logWriter io.Writer) (*smoke.PhaseResult, error)) {
	t.Helper()
	prev := smokeRunForPhase
	smokeRunForPhase = fn
	t.Cleanup(func() {
		smokeRunForPhase = prev
	})
}

func TestRunSmokePhasePassesExplicitUIURL(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{"structure":{"ui_smoke":{"enabled":true}}}`), 0o644); err != nil {
		t.Fatalf("failed to enable ui smoke testing: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "coverage"),
		UIURL:        "http://localhost:35771",
	}

	var gotUIURL string
	stubSmokeRunForPhase(t, func(ctx context.Context, scenarioName, scenarioDir, uiURL, runID string, logWriter io.Writer) (*smoke.PhaseResult, error) {
		gotUIURL = uiURL
		return &smoke.PhaseResult{
			Success: true,
			Message: "ui smoke passed",
			Result: &smoke.Result{
				Status:     smoke.StatusPassed,
				Message:    "ui smoke passed",
				UIURL:      uiURL,
				DurationMs: 123,
			},
		}, nil
	})

	report := runSmokePhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected smoke phase to succeed, got %v", report.Err)
	}
	if gotUIURL != env.UIURL {
		t.Fatalf("expected uiURL %q to be forwarded, got %q", env.UIURL, gotUIURL)
	}
	if len(report.Observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(report.Observations))
	}
}

func TestRunSmokePhaseSurfacesConsoleWarnings(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "testing.json"), []byte(`{"structure":{"ui_smoke":{"enabled":true}}}`), 0o644); err != nil {
		t.Fatalf("failed to enable ui smoke testing: %v", err)
	}

	stubSmokeRunForPhase(t, func(ctx context.Context, scenarioName, scenarioDir, uiURL, runID string, logWriter io.Writer) (*smoke.PhaseResult, error) {
		return &smoke.PhaseResult{
			Success: true,
			Message: "ui smoke passed",
			Result: &smoke.Result{
				Status:              smoke.StatusPassed,
				Message:             "ui smoke passed",
				ConsoleWarningCount: 1,
				Artifacts: smoke.ArtifactPaths{
					Console: "coverage/ui-smoke/demo/console.json",
				},
			},
		}, nil
	})

	report := runSmokePhase(context.Background(), workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "coverage"),
	}, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected smoke phase to succeed, got %v", report.Err)
	}
	for _, observation := range report.Observations {
		if observation.Prefix == "WARNING" && observation.Text == "UI smoke captured 1 browser console warning(s); see coverage/ui-smoke/demo/console.json" {
			return
		}
	}
	t.Fatalf("expected console warning observation, got %#v", report.Observations)
}
