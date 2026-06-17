package phases

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// overrideUnitValidate swaps the unit-health invocation seam for a fake and
// returns a restore func. After the hard cutover the unit phase is a thin
// delegate over `unit-health validate scenario <name> --execution --json`, so
// every behavioral test drives the phase through canned provider JSON.
func overrideUnitValidate(fn func(context.Context, string) ([]byte, int, error)) func() {
	prev := runUnitValidate
	runUnitValidate = fn
	return func() { runUnitValidate = prev }
}

func TestRunUnitPhasePassesAndMapsCoverageFindings(t *testing.T) {
	restore := overrideUnitValidate(func(context.Context, string) ([]byte, int, error) {
		return []byte(`{
			"scenario":"demo",
			"status":"passed",
			"findings":[
				{"code":"LOW_COVERAGE","category":"coverage","severity":"warning","message":"Coverage below threshold","file_path":"api","remediation":"Add tests."},
				{"code":"TEST_NOT_COLOCATED","category":"architecture","severity":"warning","message":"Test not colocated","file_path":"api/foo.go"}
			],
			"counts":{"warnings":2,"surfaces":3,"workspaces":3,"coverage_targets":5},
			"assessment":{
				"scenario":"demo",
				"provider":"unit-health",
				"phase":"unit",
				"version":"1.0.0",
				"local":{"current_level":"L4","next_level":"L5"}
			},
			"next_steps":["Raise api coverage above threshold."]
		}`), 0, nil
	})
	defer restore()

	report := runUnitPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if report.Err != nil {
		t.Fatalf("unit phase should pass on warning-only report: %v", report.Err)
	}
	// Only the coverage-category finding maps into the architecture-finding
	// channel; the architecture-category finding stays an observation.
	if len(report.Findings) != 1 {
		t.Fatalf("expected one mapped coverage finding, got %d", len(report.Findings))
	}
	f := report.Findings[0]
	if f.GetSource() != architecturev1.FindingSource_FINDING_SOURCE_COVERAGE {
		t.Fatalf("finding source = %v, want COVERAGE", f.GetSource())
	}
	if f.GetCode() != "LOW_COVERAGE" {
		t.Fatalf("finding code = %q, want LOW_COVERAGE", f.GetCode())
	}
}

func TestParseUnitOutputPreservesMaturityAssessment(t *testing.T) {
	rep, err := parseUnitOutput([]byte(`{
		"scenario":"demo",
		"status":"passed",
		"assessment":{
			"scenario":"demo",
			"provider":"unit-health",
			"phase":"unit",
			"version":"1.0.0",
			"local":{"current_level":"L3","next_level":"L4","blocking_finding_codes":["LOW_COVERAGE"]}
		}
	}`))
	if err != nil {
		t.Fatalf("parseUnitOutput() error = %v", err)
	}
	summary := translateUnitReport(rep, 0).Summary
	if summary.LocalCurrentLevel != "L3" || summary.LocalNextLevel != "L4" {
		t.Fatalf("summary local levels = %q/%q, want L3/L4", summary.LocalCurrentLevel, summary.LocalNextLevel)
	}
}

func TestParseUnitOutputRejectsMissingAssessment(t *testing.T) {
	_, err := parseUnitOutput([]byte(`{"scenario":"demo","status":"passed"}`))
	if err == nil {
		t.Fatalf("expected missing assessment to fail")
	}
	if !errors.Is(err, errProviderMaturityContract) {
		t.Fatalf("error = %v, want provider maturity contract violation", err)
	}
}

func TestParseUnitOutputRejectsWrongProvider(t *testing.T) {
	// The unit phase must reject an assessment that names a different provider/
	// phase — a stale or misrouted provider output is a contract violation, not
	// a silently-accepted pass.
	_, err := parseUnitOutput([]byte(`{
		"scenario":"demo",
		"status":"passed",
		"assessment":{"scenario":"demo","provider":"quality-health","phase":"quality","version":"1.0.0","local":{"current_level":"L1"}}
	}`))
	if err == nil || !errors.Is(err, errProviderMaturityContract) {
		t.Fatalf("error = %v, want provider maturity contract violation", err)
	}
}

func TestRunUnitPhaseFailsOnErrorFindings(t *testing.T) {
	restore := overrideUnitValidate(func(context.Context, string) ([]byte, int, error) {
		return []byte(`{
			"scenario":"demo",
			"status":"failed",
			"findings":[{"code":"TEST_EXECUTION_FAILURE","category":"execution","severity":"error","message":"go test failed","file_path":"api"}],
			"counts":{"errors":1,"surfaces":1,"workspaces":1},
			"assessment":{"scenario":"demo","provider":"unit-health","phase":"unit","version":"1.0.0","local":{"current_level":"L1","next_level":"L2"}}
		}`), 1, nil
	})
	defer restore()

	report := runUnitPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if report.Err == nil {
		t.Fatalf("unit phase should fail on error findings")
	}
	// translateUnitReport classifies test_failure; the shared standardizer maps
	// test_failure → system.
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("classification = %q, want system", report.FailureClassification)
	}
	if !strings.Contains(report.Remediation, "unit-health validate scenario demo --execution --json") {
		t.Fatalf("remediation should point at unit-health validate, got %q", report.Remediation)
	}
}

func TestRunUnitPhaseFailsWhenProviderMissing(t *testing.T) {
	restore := overrideUnitValidate(func(context.Context, string) ([]byte, int, error) {
		return nil, 0, errors.New("not found")
	})
	defer restore()

	report := runUnitPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if report.Err == nil {
		t.Fatalf("unit phase should fail when unit-health is unavailable")
	}
	if report.FailureClassification != FailureClassMissingDependency {
		t.Fatalf("classification = %q, want missing_dependency", report.FailureClassification)
	}
}

func TestRunUnitPhaseSkipViaEnv(t *testing.T) {
	t.Setenv("TEST_GENIE_SKIP_UNIT", "1")
	report := runUnitPhase(context.Background(), workspace.Environment{ScenarioName: "demo"}, io.Discard)
	if report.Err != nil {
		t.Fatalf("skip should not error: %v", report.Err)
	}
	found := false
	for _, obs := range report.Observations {
		if obs.Prefix == "SKIP" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a SKIP observation when TEST_GENIE_SKIP_UNIT=1")
	}
}

func TestParseUnitOutputRejectsEmpty(t *testing.T) {
	if _, err := parseUnitOutput([]byte("   ")); err == nil {
		t.Fatalf("expected empty unit-health output to fail parsing")
	}
}
