package planworkshop

import (
	"context"
	"testing"

	"swarm-manager/internal/attempt"
)

func TestAttemptDeciderAppliesCandidateThroughServiceAuthority(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return "v1", "plan-1", "hash-1", nil })
	svc.SetCandidateApplier(func(_ context.Context, _ Session, candidate CandidateReference, acknowledged bool) error {
		if candidate.ID != "candidate-1" || !acknowledged {
			t.Fatalf("candidate=%+v acknowledged=%v", candidate, acknowledged)
		}
		return nil
	})
	session, err := svc.Open(Subject{Kind: SubjectBacklog, Ref: "execute/example"}, ReviewPacket{})
	if err != nil {
		t.Fatal(err)
	}
	resolution := Resolution{ResponseID: "response-1", State: ResolutionCandidateReady, Candidate: &CandidateReference{ID: "candidate-1", PlanID: "plan-1", ExpectedBaseContentHash: "hash-1"}}
	session.Resolutions = []Resolution{resolution}
	if err := svc.store.Save(session); err != nil {
		t.Fatal(err)
	}
	decider := NewAttemptDecider(svc)
	result, err := decider.DecideAttempt(context.Background(), attempt.DecisionRequest{SubjectKind: AttemptSubjectCandidate, SubjectRef: session.ID + "/" + resolution.ResponseID, RoundNum: 1, Decision: "accept", Actor: "operator@example.test", Rationale: "Candidate is safe."})
	if err != nil {
		t.Fatalf("DecideAttempt: %v", err)
	}
	if result.Status != string(ResolutionCandidateApplied) {
		t.Fatalf("status = %q, want %q", result.Status, ResolutionCandidateApplied)
	}
}
