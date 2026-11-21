package agentsessions

import (
	"context"
	"testing"

	"swarm-manager/internal/attempt"
)

func TestAttemptDeciderAppliesAgentSessionProposalWithActorAttribution(t *testing.T) {
	processor := &fakeMutationProposalProcessor{application: MutationProposalApplication{Outcomes: []MutationOutcome{{MutationID: "m1", Applied: true}}}}
	service := newTestService(t, &fakeSessionSpawner{})
	service.SetMutationProposalProcessor(processor)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal, err := service.RecordProposal(context.Background(), session.ID, Proposal{Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: "Proposal", PayloadJSON: `{"form":"mutation_list","mutations":[]}`, Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}})
	if err != nil {
		t.Fatal(err)
	}
	decider := NewAttemptDecider(service)

	result, err := decider.DecideAttempt(context.Background(), attempt.DecisionRequest{
		SubjectKind:         AttemptSubjectProposal,
		SubjectRef:          session.ID + "/" + proposal.ID,
		RoundNum:            1,
		Decision:            "accept",
		Actor:               "operator@example.test",
		Rationale:           "The scoped mutation is appropriate.",
		AcceptedProposalIDs: []string{"m1"},
	})
	if err != nil {
		t.Fatalf("DecideAttempt: %v", err)
	}
	if result.Status != string(ProposalStatusApplied) {
		t.Fatalf("status = %q, want %q", result.Status, ProposalStatusApplied)
	}
}
