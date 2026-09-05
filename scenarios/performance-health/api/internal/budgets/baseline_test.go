package budgets

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mga "github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// [REQ:PH-BUDGET-002] Budget violations project to ERROR-severity maturity
// findings; an ERROR finding is what drives the shared scenario-validation
// status to FAILED, which is how a perf regression fails the test-genie
// Performance phase — and therefore the suite run (`vrooli scenario test`
// exit 1) — like any other health regression.
func TestFindingsAreErrorSeverity(t *testing.T) {
	violations := []Violation{
		{Axis: "go_build", Measured: 130000, Budget: 90000, Unit: "ms"},
		{Axis: "bundle", Measured: 700000, Budget: 500000, Unit: "bytes"},
	}
	findings := Findings("demo", violations)
	if len(findings) != 2 {
		t.Fatalf("expected one finding per violation, got %d", len(findings))
	}
	for _, f := range findings {
		if f.Severity != "error" {
			t.Fatalf("budget findings must be error severity, got %q", f.Severity)
		}
		if f.Code == "" {
			t.Fatal("finding must carry a stable code")
		}
	}
	if findings[0].Code != "PERF_BUDGET_BREACH_GO_BUILD" {
		t.Fatalf("unexpected code %q", findings[0].Code)
	}
}

// budgetMaturitySpec is a minimal valid maturity spec that classifies the
// budget-breach finding codes at ERROR severity, so the test is deterministic
// regardless of which findings the live scenario spec declares.
const budgetMaturitySpec = `{
  "provider": "performance-health",
  "phase": "performance",
  "version": "1.0.0",
  "levels": [
    {"id": "L0", "name": "exists", "description": "scenario resolvable",
     "entry_criteria": ["a scenario is provided"], "exit_criteria": ["scenario resolvable"]},
    {"id": "L1", "name": "within budget", "description": "no perf budget breach",
     "entry_criteria": ["scenario resolvable"], "exit_criteria": ["within performance budget"]}
  ],
  "findings": {
    "PERF_BUDGET_BREACH_GO_BUILD": {
      "local_level_impact": "L1", "global_impact": "evolvability_gap",
      "dimension": "performance", "severity_default": "SEVERITY_ERROR", "clean_requirement": "required"
    }
  },
  "fallback": {
    "local_level_impact": "L1", "global_impact": "evolvability_gap",
    "dimension": "performance", "severity_default": "SEVERITY_ERROR", "clean_requirement": "required"
  }
}`

// [REQ:PH-BUDGET-002] An ERROR finding makes the assessment status FAILED; a
// clean (within-budget) run yields no findings and a non-failing status. This is
// the exact pipeline the validation provider feeds into the test-genie
// Performance phase (and therefore the suite run).
func TestFindingsDriveFailedStatus(t *testing.T) {
	spec, err := mga.ParseSpec([]byte(budgetMaturitySpec))
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}

	clean, err := mga.BuildProtoAssessment(mga.BuildInput{Scenario: "demo", Spec: *spec})
	if err != nil {
		t.Fatalf("clean assessment: %v", err)
	}
	if mga.DeriveValidationStatus(clean).String() == "VALIDATION_STATUS_FAILED" {
		t.Fatal("a within-budget run must not be FAILED")
	}

	breached, err := mga.BuildProtoAssessment(mga.BuildInput{
		Scenario: "demo",
		Spec:     *spec,
		Findings: Findings("demo", []Violation{{Axis: "go_build", Measured: 130000, Budget: 90000, Unit: "ms"}}),
	})
	if err != nil {
		t.Fatalf("breached assessment: %v", err)
	}
	if got := mga.DeriveValidationStatus(breached); got.String() != "VALIDATION_STATUS_FAILED" {
		t.Fatalf("a budget breach must fail validation (suite run exit 1), got %s", got)
	}
	// And the breach surfaces as an ERROR architecture finding for the
	// finding pipeline test-genie's Performance phase already consumes.
	arch := mga.AssessmentToArchitectureFindings("demo", breached, architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED)
	var sawError bool
	for _, f := range arch {
		if f.GetSeverity() == architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
			sawError = true
		}
	}
	if !sawError {
		t.Fatal("expected at least one ERROR architecture finding from the breach")
	}
}

// [REQ:PH-BUDGET-001] The config-backed store round-trips a budget through
// .vrooli/testing.json performance.budgets and enforces the ratchet on disk.
func TestConfigStoreRoundTripAndRatchet(t *testing.T) {
	root := t.TempDir()
	store := NewConfigStore(root, func(scenario string) (string, error) {
		return filepath.Join(root, "scenarios", scenario), nil
	})

	if _, declared, err := store.Get(context.Background(), "demo"); err != nil || declared {
		t.Fatalf("expected no budget initially, got declared=%v err=%v", declared, err)
	}

	want := Budget{Scenario: "demo", GoBuildMaxMs: 90000, BundleMaxBytes: 500000, Ratchet: true}
	if _, err := store.Set(context.Background(), want, false); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, declared, err := store.Get(context.Background(), "demo")
	if err != nil || !declared {
		t.Fatalf("expected declared budget after write, got declared=%v err=%v", declared, err)
	}
	if got.GoBuildMaxMs != 90000 || got.BundleMaxBytes != 500000 || !got.Ratchet {
		t.Fatalf("round-trip mismatch: %#v", got)
	}

	// Persisted under performance.budgets of the scenario's testing.json.
	rec, ok, err := loadBudgetRecord(filepath.Join(root, "scenarios", "demo", TestingConfigRelPath))
	if err != nil || !ok {
		t.Fatalf("load persisted config: ok=%v err=%v", ok, err)
	}
	if rec.GoBuildMaxMs != 90000 {
		t.Fatalf("expected demo budget persisted at %s, got %#v", TestingConfigRelPath, rec)
	}

	// Ratchet enforced against the persisted budget.
	if _, err := store.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 150000, Ratchet: true}, false); err == nil {
		t.Fatal("expected ratchet to reject a loosening write on disk")
	}
}

// [REQ:PH-BUDGET-001] A budget write is a structured read-modify-write that
// preserves every other testing.json key (and their order) — only
// performance.budgets is touched.
func TestConfigStoreSetPreservesSiblingKeys(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const original = `{
  "version": "1.0.0",
  "lint": {"policy": {"unmatched_code_components": "warning"}},
  "performance": {
    "checks": {"lighthouse": {"enabled": true}}
  },
  "requirements": {"enforce": false}
}`
	testingPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	if err := os.WriteFile(testingPath, []byte(original), 0o644); err != nil {
		t.Fatalf("seed testing.json: %v", err)
	}

	store := NewConfigStore(root, func(scenario string) (string, error) {
		return filepath.Join(root, "scenarios", scenario), nil
	})
	if _, err := store.Set(context.Background(), Budget{Scenario: "demo", UIBuildMaxMs: 12000}, false); err != nil {
		t.Fatalf("Set: %v", err)
	}

	raw, err := os.ReadFile(testingPath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse written testing.json: %v", err)
	}
	for _, key := range []string{"version", "lint", "performance", "requirements"} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("sibling key %q was dropped by budget write; got %s", key, raw)
		}
	}
	var perf map[string]json.RawMessage
	if err := json.Unmarshal(doc["performance"], &perf); err != nil {
		t.Fatalf("parse performance block: %v", err)
	}
	if _, ok := perf["checks"]; !ok {
		t.Fatalf("performance.checks was dropped; got %s", doc["performance"])
	}
	if _, ok := perf["budgets"]; !ok {
		t.Fatalf("performance.budgets was not written; got %s", doc["performance"])
	}
}
