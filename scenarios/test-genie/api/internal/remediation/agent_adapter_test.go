package remediation

import (
	"strings"
	"testing"
	"time"
)

func TestTaskPacketIsEvidenceBoundAndContainsNoAgentPolicy(t *testing.T) {
	job := Job{
		ID: "job-1", Scenario: "demo", SelectedFindingIDs: []string{"afid:1"}, SelectedRequirementIDs: []string{"REQ-1"}, AdditionalContext: "Keep the public API stable.",
		Source: Plan{
			SourceExecutionID: "execution-1", SourceRunID: "run-1", CreatedAt: time.Now(),
			Phases:       []Phase{{Name: "unit", DocsPath: "docs/phases/unit.md", Remediation: "Add the missing regression test.", RunnabilityVerdict: "runnable"}},
			Findings:     []Finding{{StableID: "afid:1", Phase: "unit", Severity: "error", Class: "deterministic", Message: "Missing regression", Locations: []string{"api/service.go"}}},
			Requirements: []RequirementEvidence{{ID: "REQ-1", Title: "Regression evidence", LiveStatus: "failed", Validations: []string{"test:unit:failed"}}},
		},
	}

	packet := taskPacket(job)
	for _, expected := range []string{"Source execution: execution-1", "Source run: run-1", "[afid:1] Missing regression", "docs/phases/unit.md", "REQ-1", "Keep the public API stable.", "server-owned verification"} {
		if !strings.Contains(packet, expected) {
			t.Fatalf("packet missing %q:\n%s", expected, packet)
		}
	}
	for _, forbidden := range []string{"networkEnabled", "allowedTools", "sandbox", "preamble", "merge policy"} {
		if strings.Contains(strings.ToLower(packet), strings.ToLower(forbidden)) {
			t.Fatalf("packet must not claim caller policy %q:\n%s", forbidden, packet)
		}
	}
}
