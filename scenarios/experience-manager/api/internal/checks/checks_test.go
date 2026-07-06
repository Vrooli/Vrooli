package checks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"experience-manager/internal/spec"
)

func TestRegistryContainsPhaseEightChecks(t *testing.T) {
	got := Registry()
	names := map[string]bool{}
	for _, check := range got {
		names[check.Name()] = true
	}
	for _, name := range []string{"bas.reference_integrity", "state.coverage", "attestation.manual_freshness", "reconcile.structure"} {
		if !names[name] {
			t.Fatalf("registry = %#v, missing %s", got, name)
		}
	}
}

func TestStateCoverageCheckEmitsMissingDesignState(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "scenarios", "demo")
	writeMinimalPageExperience(t, target, []string{"default"})
	mustWrite(t, filepath.Join(target, "DESIGN.md"), `# Demo Design

## UX-State Contract

- loading
- empty
`)

	report, err := New(root, StateCoverageCheck{}).ValidateScenario(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("findings = %d, want missing loading + empty: %+v", len(report.Findings), report.Findings)
	}
	for _, finding := range report.Findings {
		if finding.Code != spec.CodeStateMissing || finding.Severity != spec.SeverityWarning {
			t.Fatalf("finding = %+v, want state_missing warning", finding)
		}
	}
}

func TestStateCoverageCheckDegradesWhenDesignAbsent(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "scenarios", "demo")
	writeMinimalPageExperience(t, target, []string{"default"})

	report, err := New(root, StateCoverageCheck{}).ValidateScenario(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if report.DegradedReason == "" {
		t.Fatal("expected DESIGN.md absence to set degraded reason")
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %+v, want advisory degradation without findings", report.Findings)
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

func writeMinimalPageExperience(t *testing.T, scenarioDir string, states []string) {
	t.Helper()
	mustWrite(t, filepath.Join(scenarioDir, "PRD.md"), "## Operational Targets\n- [ ] OT-P0-001 | Demo | Demo target\n")
	stateDocs := make([]string, 0, len(states))
	for _, state := range states {
		stateDocs = append(stateDocs, `{"id":"`+state+`","description":"`+state+` state."}`)
	}
	mustWrite(t, filepath.Join(scenarioDir, "experience", "index.json"), `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "pages": [{"id":"home","path":"pages/home.json","status":"active"}],
  "journeys": []
}`)
	mustWrite(t, filepath.Join(scenarioDir, "experience", "pages", "home.json"), `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {"id":"home","title":"Home","routes":["/"],"purpose":"A sufficiently long purpose."},
  "states": [`+strings.Join(stateDocs, ",")+`],
  "elements": [],
  "claims": [],
  "bindings": {"elements": {}}
}`)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
