package checks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"experience-manager/internal/spec"
)

func TestRegistryContainsReconciliationCheck(t *testing.T) {
	if got := Registry(); len(got) != 1 || got[0].Name() != "reconcile.structure" {
		t.Fatalf("registry = %#v, want structure reconciliation check", got)
	}
}

func TestValidateScenarioResolvesSlugUnderRepoRoot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "scenarios", "demo")
	writeMinimalExperience(t, target)
	report, err := New(root).ValidateScenario(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if report.Scenario != "demo" || report.TargetPath != target {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestCapSeverityDowngradesBlockerToError(t *testing.T) {
	findings := CapSeverity([]spec.Finding{
		{Code: "experience.claim_failed", Severity: "SEVERITY_BLOCKER"},
	})
	if got := findings[0].Severity; got != "SEVERITY_ERROR" {
		t.Fatalf("severity = %q", got)
	}
}

func writeMinimalExperience(t *testing.T, scenarioDir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(scenarioDir, "experience"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "pages": [],
  "journeys": []
}`)
	if err := os.WriteFile(filepath.Join(scenarioDir, "experience", "index.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
