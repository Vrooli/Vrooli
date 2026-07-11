package conformance

import "testing"

func TestBuildAssessmentRepresentsFullLadderAndAdvisoryFindings(t *testing.T) {
	clean := buildAssessment(Report{Scenario: "consumer"})
	if clean.GetLocal().GetCurrentLevel() != "L4" || !clean.GetLocal().GetClean() || clean.GetLocal().GetNextLevel() != "" {
		t.Fatalf("clean assessment = %#v, want clean L4", clean.GetLocal())
	}

	assessment := buildAssessment(Report{Scenario: "consumer", Findings: []Finding{
		{Code: CodeRoleUnresolved, Severity: "SEVERITY_ERROR"},
		{Code: CodeDirectSpawnBypass, Severity: "SEVERITY_WARNING"},
	}})
	if assessment.GetLocal().GetCurrentLevel() != "L2" || assessment.GetLocal().GetNextLevel() != "L3" {
		t.Fatalf("assessment level = %#v, want L2 -> L3", assessment.GetLocal())
	}
	if got := assessment.GetFindingsBySeverity(); got["SEVERITY_ERROR"] != 1 || got["SEVERITY_WARNING"] != 1 {
		t.Fatalf("severity counts = %#v", got)
	}
	if got := assessment.GetLocal().GetBlockingFindingCodes(); len(got) != 1 || got[0] != CodeRoleUnresolved {
		t.Fatalf("blocking findings = %#v", got)
	}
}
