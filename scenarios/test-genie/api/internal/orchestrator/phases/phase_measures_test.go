package phases

import (
	"testing"

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
