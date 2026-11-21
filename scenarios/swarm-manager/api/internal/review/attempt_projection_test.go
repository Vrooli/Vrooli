package review

import "testing"

func TestRoundAsAttemptPreservesReviewFactsWithoutLifecycle(t *testing.T) {
	round := Round{RoundNum: 2, GeneratedAt: "2026-07-29T00:00:00Z", Status: RoundStatusComplete, AgentAssessment: "ready", Classification: "accepted", Evidence: []EvidenceItem{{ID: "e1", CriterionID: "criterion-1", Settlement: "settled", Producer: "command", Trust: "observed", Title: "check"}}}
	got := round.AsAttempt("execute", "item")
	if got.SubjectRef != "execute/item" || got.TransitionKey != "work.review" || len(got.Evidence) != 1 || got.Evidence[0].CriterionID != "criterion-1" {
		t.Fatalf("attempt = %#v", got)
	}
}
