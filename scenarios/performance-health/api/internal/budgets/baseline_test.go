package budgets

import (
	"context"
	"path/filepath"
	"testing"

	mga "github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// [REQ:PH-BUDGET-002] Budget violations project to ERROR-severity maturity
// findings; an ERROR finding is what drives the shared scenario-validation
// status to FAILED, which is how a perf regression fails baseline-diff (exit 1)
// like any other health regression.
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
// the exact pipeline the validation provider feeds into baseline-diff.
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
		t.Fatalf("a budget breach must fail validation (exit 1), got %s", got)
	}
	// And the breach surfaces as an ERROR architecture finding for the
	// finding/baseline-diff pipeline test-genie already consumes.
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
// .vrooli/perf-budgets.json and enforces the ratchet on disk.
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

	// Persisted at the documented path with the scenario's record present.
	cfg, err := loadConfigFile(filepath.Join(root, "scenarios", "demo", BudgetsConfigRelPath))
	if err != nil {
		t.Fatalf("load persisted config: %v", err)
	}
	if _, ok := cfg.Budgets["demo"]; !ok {
		t.Fatalf("expected demo budget persisted at %s", BudgetsConfigRelPath)
	}

	// Ratchet enforced against the persisted budget.
	if _, err := store.Set(context.Background(), Budget{Scenario: "demo", GoBuildMaxMs: 150000, Ratchet: true}, false); err == nil {
		t.Fatal("expected ratchet to reject a loosening write on disk")
	}
}
