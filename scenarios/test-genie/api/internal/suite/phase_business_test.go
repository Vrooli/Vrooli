package suite

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBusinessPhaseValidatesModules(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	env := PhaseEnvironment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}

	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected business phase to pass, got error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestRunBusinessPhaseFailsWhenModuleIncomplete(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	brokenDir := filepath.Join(scenarioDir, "requirements", "02-broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("failed to create broken module dir: %v", err)
	}
	payload := `{"requirements":[{"id":"","title":"","criticality":"","status":"","validation":[]}]}`
	if err := os.WriteFile(filepath.Join(brokenDir, "module.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("failed to write broken module: %v", err)
	}

	env := PhaseEnvironment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected validation failure for malformed module")
	}
	if !strings.Contains(report.Err.Error(), "02-broken") {
		t.Fatalf("expected error to reference module, got %v", report.Err)
	}
	if report.FailureClassification != failureClassMisconfiguration {
		t.Fatalf("expected misconfiguration classification, got %s", report.FailureClassification)
	}
	if report.Remediation == "" {
		t.Fatalf("expected remediation guidance")
	}
}

func TestRunBusinessPhaseFailsWhenNoModulesFound(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	if err := os.RemoveAll(filepath.Join(scenarioDir, "requirements", "01-internal-orchestrator")); err != nil {
		t.Fatalf("failed to remove seeded module: %v", err)
	}

	env := PhaseEnvironment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure when no modules exist")
	}
	if !strings.Contains(report.Err.Error(), "no requirement modules") {
		t.Fatalf("unexpected error: %v", report.Err)
	}
}
