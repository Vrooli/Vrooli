package readiness

import "testing"

func TestReportRequiresGovernedRemediation(t *testing.T) {
	report := Report{Status: Missing, Source: "scenario-dependency-analyzer", Requirements: []Requirement{{ID: "vitest", Kind: "tool"}}}
	if err := report.Validate(); err == nil {
		t.Fatal("missing remediation unexpectedly accepted")
	}
	report.Requirements[0].Remediation = "Provision vitest through Scenario Dependency Analyzer"
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
	if !report.BlocksExecution() {
		t.Fatal("missing dependency did not block execution")
	}
}
