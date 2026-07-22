package main

import (
	"testing"

	"swarm-manager/internal/planworkshop"
)

func TestAutomaticFollowUpPolicyIsExplicitAndBounded(t *testing.T) {
	proposal := planworkshop.FollowUpProposal{
		Route:             "work.correct",
		Target:            "current_work",
		SourceReviewRef:   "review/fix/example/round/1",
		SourceExecutionID: "execution-1",
		Rationale:         "verified regression",
		Confidence:        "high",
	}
	if automaticFollowUpAllowed(proposal) {
		t.Fatal("automatic follow-up must be denied by default")
	}
	t.Setenv("SWARM_MANAGER_AUTO_FOLLOW_UP", "true")
	if !automaticFollowUpAllowed(proposal) {
		t.Fatal("enabled high-confidence current-work correction was denied")
	}
	proposal.Confidence = "medium"
	if automaticFollowUpAllowed(proposal) {
		t.Fatal("medium-confidence follow-up bypassed bounded policy")
	}
}
