package phases

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func TestTranslateMeasuresReport_Passing(t *testing.T) {
	rep := &measuresReport{Scenario: "demo", Passed: true}
	out := translateMeasuresReport(rep, 0)
	if !out.Success {
		t.Fatal("expected Success=true")
	}
	if out.Summary.Scenario != "demo" {
		t.Fatalf("summary not translated: %+v", out.Summary)
	}
}

func TestTranslateMeasuresReport_UncoveredDomainFailsPhase(t *testing.T) {
	rep := &measuresReport{
		Scenario: "swarm-manager",
		Passed:   false,
		Findings: []measuresFinding{
			{Severity: "SEVERITY_ERROR", RuleID: "measures.uncovered-domain", Title: "captures uncovered", FilePath: "cli/manifest.json", Scanner: "coverage"},
		},
	}
	rep.Summary.Errors = 1
	out := translateMeasuresReport(rep, 1)
	if out.Success {
		t.Fatal("expected Success=false on ERROR finding")
	}
	if out.FailureClass == "" {
		t.Fatal("expected failure class set")
	}
	if out.Summary.Errors != 1 {
		t.Fatalf("summary error count not propagated: %+v", out.Summary)
	}
}

func TestTranslateMeasuresReport_WarningOnlySucceeds(t *testing.T) {
	rep := &measuresReport{
		Scenario: "demo",
		Passed:   true,
		Findings: []measuresFinding{
			{Severity: "SEVERITY_WARNING", RuleID: "measures.tier-fallback", Title: "no canonical params", Scanner: "tier"},
		},
	}
	rep.Summary.Warnings = 1
	out := translateMeasuresReport(rep, 0)
	if !out.Success {
		t.Fatal("expected Success=true on warnings-only")
	}
}

func TestMeasuresArchFindings_MapsSourceAndStableID(t *testing.T) {
	rep := &measuresReport{
		Scenario: "swarm-manager",
		Findings: []measuresFinding{
			{Severity: "SEVERITY_ERROR", RuleID: "measures.uncovered-domain", Title: "captures", FilePath: "cli/manifest.json"},
		},
	}
	got := measuresArchFindings("swarm-manager", rep)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}
	if got[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_MEASURES {
		t.Errorf("source = %v, want FINDING_SOURCE_MEASURES", got[0].GetSource())
	}
	if got[0].GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
		t.Errorf("severity = %v, want ERROR", got[0].GetSeverity())
	}
	if got[0].GetStableId() == "" {
		t.Error("stable id must be stamped")
	}
}

func TestParseMeasuresOutput_Empty(t *testing.T) {
	if _, err := parseMeasuresOutput([]byte("  ")); err == nil {
		t.Fatal("expected error on empty output")
	}
}

// --- Phase 4: producer probe-if-reachable ---------------------------------

func measuresEnv(t *testing.T) workspace.Environment {
	t.Helper()
	dir := t.TempDir()
	env := workspace.Environment{ScenarioName: "demo", ScenarioDir: dir, TestDir: filepath.Join(dir, "test")}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return env
}

// swapMeasuresSeams overrides the reachability + validate seams for the duration
// of a test, restoring them on cleanup.
func swapMeasuresSeams(t *testing.T, reachable bool, validate func(probe bool) ([]byte, int, error)) {
	t.Helper()
	origReach := measuresTargetReachable
	origRun := runMeasuresValidate
	measuresTargetReachable = func(_ context.Context, _ string) bool { return reachable }
	runMeasuresValidate = func(_ context.Context, _ string, probe bool) ([]byte, int, error) {
		return validate(probe)
	}
	t.Cleanup(func() {
		measuresTargetReachable = origReach
		runMeasuresValidate = origRun
	})
}

func TestRunMeasuresPhase_ProbesWhenReachable(t *testing.T) {
	const hollow = `{"scenario":"demo","passed":false,` +
		`"findings":[{"rule_id":"measures.hollow-declaration","severity":"SEVERITY_ERROR",` +
		`"title":"Hollow measure declaration: notes.count","scanner":"probe","file_path":"cli/manifest.json"}],` +
		`"summary":{"errors":1}}`
	var gotProbe bool
	swapMeasuresSeams(t, true, func(probe bool) ([]byte, int, error) {
		gotProbe = probe
		return []byte(hollow), 1, nil
	})

	var buf bytes.Buffer
	report := runMeasuresPhase(context.Background(), measuresEnv(t), io.MultiWriter(&buf, io.Discard))
	if !gotProbe {
		t.Fatal("reachable target must take the --probe branch")
	}
	if report.Err == nil {
		t.Fatal("a hollow-declaration ERROR must fail the phase")
	}
	var foundHollow bool
	for _, f := range report.Findings {
		if f.GetSource() == architecturev1.FindingSource_FINDING_SOURCE_MEASURES &&
			f.GetCode() == "measures.hollow-declaration" {
			foundHollow = true
		}
	}
	if !foundHollow {
		t.Fatalf("want a FINDING_SOURCE_MEASURES hollow-declaration, got %+v", report.Findings)
	}
}

func TestRunMeasuresPhase_StaticWhenUnreachable(t *testing.T) {
	const passing = `{"scenario":"demo","passed":true,"findings":[],"summary":{}}`
	var gotProbe bool
	swapMeasuresSeams(t, false, func(probe bool) ([]byte, int, error) {
		gotProbe = probe
		return []byte(passing), 0, nil
	})

	var buf bytes.Buffer
	report := runMeasuresPhase(context.Background(), measuresEnv(t), io.MultiWriter(&buf, io.Discard))
	if gotProbe {
		t.Fatal("unreachable target must take the static (no --probe) branch")
	}
	if report.Err != nil {
		t.Fatalf("unreachable target must not fail the phase; got %v", report.Err)
	}
}
