package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReportCarriesParserContractFields(t *testing.T) {
	report := Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
		Findings: []Finding{{
			Code:       "experience.schema_invalid",
			Severity:   "SEVERITY_ERROR",
			Message:    "invalid",
			Locations:  []string{"experience/index.json"},
			Suggestion: "repair schema",
		}},
		DegradedReason: "design contract absent",
	}

	if report.Scenario != "demo" || report.TargetPath == "" {
		t.Fatalf("report identity not preserved: %+v", report)
	}
	if got := report.Findings[0].Code; got != "experience.schema_invalid" {
		t.Fatalf("finding code = %q", got)
	}
	if report.DegradedReason == "" {
		t.Fatal("degraded reason should be preserved")
	}
}

func TestParseScenarioFixturesContractGreen(t *testing.T) {
	root := repoRoot(t)
	for _, scenario := range []string{"experience-manager", "business-health", "web-console"} {
		t.Run(scenario, func(t *testing.T) {
			report, err := ParseScenario(filepath.Join(root, "scenarios", scenario))
			if err != nil {
				t.Fatalf("ParseScenario: %v", err)
			}
			if len(report.Findings) > 0 {
				t.Fatalf("expected contract-green fixture, got findings: %+v", report.Findings)
			}
			if report.Spec == nil || report.Spec.Index.Scenario != scenario {
				t.Fatalf("parsed spec identity mismatch: %+v", report.Spec)
			}
		})
	}
}

func TestParseScenarioPreservesSpikeExtensions(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "business-health"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	matrix := report.Spec.Pages["matrix"]
	if _, ok := matrix.Extensions["x-spike"]; !ok {
		t.Fatalf("matrix x-spike extension was not preserved: %#v", matrix.Extensions)
	}
}

func TestParseScenarioPreservesClaimExtensions(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "web-console"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	workspace := report.Spec.Pages["workspace"]
	for _, claim := range workspace.Claims {
		if _, ok := claim.Extensions["x-display-mode"]; !ok {
			continue
		}
		var mode string
		if err := json.Unmarshal(claim.Extensions["x-display-mode"], &mode); err != nil {
			t.Fatalf("x-display-mode is not a string: %v", err)
		}
		if mode != "tabs" {
			t.Fatalf("x-display-mode = %q, want tabs", mode)
		}
		return
	}
	t.Fatalf("workspace claim-level x-display-mode extension was not preserved")
}

func TestParseScenarioComputesDepths(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "experience-manager"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	if got := report.PageDepths["studio"]; got != 4 {
		t.Fatalf("studio depth = L%d, want L4", got)
	}
	if got := report.PageDepths["findings"]; got < 3 {
		t.Fatalf("findings depth = L%d, want at least L3", got)
	}
}

func TestParseScenarioFindsContractViolations(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "demo")
	mustWrite(t, filepath.Join(scenario, "PRD.md"), "## Operational Targets\n- [ ] OT-P0-001 | Demo | Demo target\n")
	mustWrite(t, filepath.Join(scenario, "experience", "index.json"), `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "pages": [{"id":"home","path":"pages/home.json","status":"active"}],
  "journeys": [{"id":"journey","path":"journeys/journey.json","status":"active"}]
}`)
	mustWrite(t, filepath.Join(scenario, "experience", "pages", "home.json"), `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {"id":"home","title":"Home","routes":["/"],"purpose":"A sufficiently long purpose.","prd_refs":["OT-P9-999"]},
  "states": [{"id":"default"}],
  "elements": [{"id":"known","role":"button"}],
  "claims": [
    {"id":"bad-custom","type":"custom","statement":"This custom claim is invalid as machine.","tier":"machine","elements":["known"]},
    {"id":"missing-ref","type":"element-present","statement":"This points at a missing element.","tier":"machine","elements":["ghost"],"states":["ghost-state"]}
  ],
  "bindings": {"elements": {"ghost-binding": {"testid":"ghost"}}}
}`)
	mustWrite(t, filepath.Join(scenario, "experience", "journeys", "journey.json"), `{
  "kind": "experience-journey",
  "schemaVersion": "1.0.0",
  "journey": {"id":"journey","title":"Journey","purpose":"A sufficiently long journey purpose."},
  "steps": [{"page":"home","state":"missing","intent":"A sufficiently long step intent."}]
}`)

	report, err := ParseScenario(scenario)
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	for _, code := range []string{CodePRDRefUnmatched, CodeTierViolation, CodeRefUnresolved, CodeBindingOrphan} {
		if !hasCode(report.Findings, code) {
			t.Fatalf("missing %s in findings: %+v", code, report.Findings)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
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

func hasCode(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
