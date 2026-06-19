package main

import (
	"testing"

	intent "intent-go"
)

func TestStandardsViolationsUseIntentCodeForPRDRefIssues(t *testing.T) {
	report := ScenarioQualityReport{
		RequirementsPath: "requirements/core/module.json",
		HasPRD:           true,
		HasRequirements:  true,
		PRDRefIssues: []PRDValidationIssue{{
			RequirementID: "REQ-1",
			PRDRef:        "OT-P0-999",
			IssueType:     "missing_section",
			Message:       "missing target",
		}},
	}

	got := buildStandardsViolationsFromReport(report)
	if len(got) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(got), got)
	}
	if got[0].RuleID != intent.CodePRDRefUnmatched {
		t.Fatalf("RuleID = %q, want %q", got[0].RuleID, intent.CodePRDRefUnmatched)
	}
	if got[0].Severity != "high" {
		t.Fatalf("Severity = %q, want high", got[0].Severity)
	}
}
