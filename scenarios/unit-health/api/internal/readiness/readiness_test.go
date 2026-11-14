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

func TestReportRejectsMissingStatusSourceAndMalformedRequirements(t *testing.T) {
	for _, report := range []Report{{Source: "source"}, {Status: "future", Source: "source"}, {Status: Ready}, {Status: Ready, Source: "source", Requirements: []Requirement{{Kind: "tool", Remediation: "fix"}}}, {Status: Ready, Source: "source", Requirements: []Requirement{{ID: "id", Remediation: "fix"}}}, {Status: Ready, Source: "source", Requirements: []Requirement{{ID: "id", Kind: "tool"}}}} {
		if err := report.Validate(); err == nil {
			t.Fatalf("invalid report accepted: %+v", report)
		}
	}
	if (Report{Status: Ready, Source: "source"}).BlocksExecution() {
		t.Fatal("ready report blocks execution")
	}
}
