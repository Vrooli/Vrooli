package checks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"experience-manager/internal/spec"
	"experience-manager/internal/testutil"
)

func TestRegistryContainsPhaseEightChecks(t *testing.T) {
	got := Registry()
	names := map[string]bool{}
	for _, check := range got {
		names[check.Name()] = true
	}
	for _, name := range []string{"bas.reference_integrity", "state.coverage", "oracle.claim_coverage", "attestation.manual_freshness", "reconcile.structure"} {
		if !names[name] {
			t.Fatalf("registry = %#v, missing %s", got, name)
		}
	}
}

func TestOracleCoverageCheckRequiresClaimsAndMachineEvidence(t *testing.T) {
	tests := []struct {
		name     string
		claims   []spec.Claim
		findings int
	}{
		{name: "uncovered", findings: 2},
		{name: "covered but not machine", claims: []spec.Claim{{ID: "gct-mobile-002-chrome-color-agreement", Tier: "manual"}, {ID: "gct-mobile-003-grouped", Tier: "manual"}}, findings: 1},
		{name: "machine covered", claims: []spec.Claim{{ID: "gct-mobile-002-chrome-color-agreement", Tier: "machine"}, {ID: "gct-mobile-003-grouped", Tier: "manual"}}, findings: 0},
		{name: "non artifact claim needs no machine tier", claims: []spec.Claim{{ID: "gct-mobile-003-grouped", Tier: "manual"}}, findings: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := spec.Report{Spec: &spec.ScenarioSpec{
				Oracles: map[string]spec.OracleDocument{"fixtures/oracle.json": {Cases: []spec.OracleCase{
					{ID: "gct-mobile-002", Evidence: []string{"screenshot_png"}},
					{ID: "gct-mobile-003", Evidence: []string{"dom_snapshot"}},
				}}},
				Pages: map[string]spec.PageDocument{"workspace": {Claims: tt.claims}},
			}}
			got := (OracleCoverageCheck{}).Run(context.Background(), report)
			if len(got) != tt.findings {
				t.Fatalf("findings = %d, want %d: %+v", len(got), tt.findings, got)
			}
		})
	}
}

func TestStateCoverageCheckEmitsMissingDesignState(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "scenarios", "demo")
	writeMinimalPageExperience(t, target, []string{"default"})
	testutil.WriteFile(t, filepath.Join(target, "DESIGN.md"), `# Demo Design

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
	testutil.WriteFile(t, filepath.Join(scenarioDir, "PRD.md"), "## Operational Targets\n- [ ] OT-P0-001 | Demo | Demo target\n")
	stateDocs := make([]string, 0, len(states))
	for _, state := range states {
		stateDocs = append(stateDocs, `{"id":"`+state+`","description":"`+state+` state."}`)
	}
	testutil.WriteFile(t, filepath.Join(scenarioDir, "experience", "index.json"), `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "pages": [{"id":"home","path":"pages/home.json","status":"active"}],
  "journeys": []
}`)
	testutil.WriteFile(t, filepath.Join(scenarioDir, "experience", "pages", "home.json"), `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {"id":"home","title":"Home","routes":["/"],"purpose":"A sufficiently long purpose."},
  "states": [`+strings.Join(stateDocs, ",")+`],
  "elements": [],
  "claims": [],
  "bindings": {"elements": {}}
}`)
}
