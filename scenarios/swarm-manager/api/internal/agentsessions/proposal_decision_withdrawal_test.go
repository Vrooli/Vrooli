package agentsessions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"swarm-manager/internal/attempt"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const withdrawnEventType = "swarm-manager.proposal.decision_withdrawn.v1"

func withdrawalsIn(events []eventbus.DomainEvent) []eventbus.DomainEvent {
	var out []eventbus.DomainEvent
	for _, e := range events {
		if e.EventType == withdrawnEventType {
			out = append(out, e)
		}
	}
	return out
}

func reconcileAll(t *testing.T, service *Service) {
	t.Helper()
	if _, err := service.ReconcileProposalDecisionPage(context.Background(), ProposalDecisionCursor{}, 25); err != nil {
		t.Fatal(err)
	}
}

func publicationOf(t *testing.T, service *Service, sessionID, proposalID string) *ProposalDecisionPublication {
	t.Helper()
	session, err := service.loadSession(context.Background(), sessionID)
	if err != nil {
		t.Fatal(err)
	}
	proposal, ok := findProposal(session, proposalID)
	if !ok || proposal.DecisionPublication == nil {
		t.Fatalf("proposal %q has no publication", proposalID)
	}
	return proposal.DecisionPublication
}

// decideInSwarmManager closes a proposal the way the swarm-manager review page
// does, without any answer through the hub.
func decideInSwarmManager(t *testing.T, service *Service, sessionID, proposalID, decision string) {
	t.Helper()
	if _, err := attemptRouterFor(t, service).DecideAttempt(context.Background(), attempt.DecisionRequest{
		SubjectKind: AttemptSubjectProposal, SubjectRef: sessionID + "/" + proposalID, RoundNum: 1,
		Decision: decision, Actor: "operator", AcceptedProposalIDs: nil,
	}); err != nil {
		t.Fatalf("decide in swarm-manager: %v", err)
	}
}

// A ready proposal's decision is keyed by session and proposal, so the hub keeps
// one open decision for it.
func TestProposalDecisionIsKeyedBySessionAndProposal(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	key, _ := publisher.events[0].Payload["subject_key"].(string)
	if key != "proposal:"+session.ID+"/"+proposal.ID || !operatordecision.SubjectKeyRule.MatchString(key) {
		t.Fatalf("subject_key = %q", key)
	}
	if long := proposalDecisionSubjectKey(strings.Repeat("s", 100), strings.Repeat("p", 100)); !operatordecision.SubjectKeyRule.MatchString(long) {
		t.Fatalf("a long id still gives a valid key: %q", long)
	}
}

// A proposal decided in swarm-manager (not through the hub) closes the operator's
// open decision once, with a reason; one still in review is left alone.
func TestProposalLeavingReviewWithdrawsItsDecisionOnce(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	decided := recordReadyMutationProposal(t, service, session.ID)
	waiting, err := service.RecordProposal(context.Background(), session.ID, Proposal{
		Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: "Another", PayloadJSON: readyMutationPayload,
		Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	})
	if err != nil {
		t.Fatal(err)
	}
	reconcileAll(t, service)
	if got := withdrawalsIn(publisher.events); len(got) != 0 {
		t.Fatalf("a proposal in review is not withdrawn: %+v", got)
	}

	decideInSwarmManager(t, service, session.ID, decided.ID, "drop")
	reconcileAll(t, service)
	got := withdrawalsIn(publisher.events)
	if len(got) != 1 || got[0].Payload["subject_key"] != "proposal:"+session.ID+"/"+decided.ID || got[0].Payload["reason"] != "This proposal has been rejected." {
		t.Fatalf("the rejected proposal's decision is withdrawn with its reason: %+v", got)
	}
	if publicationOf(t, service, session.ID, decided.ID).WithdrawnAt == "" {
		t.Fatal("the withdrawal is recorded on the publication")
	}
	if publicationOf(t, service, session.ID, waiting.ID).WithdrawnAt != "" {
		t.Fatal("the proposal still in review keeps its decision")
	}
	reconcileAll(t, service)
	reconcileAll(t, service)
	if len(withdrawalsIn(publisher.events)) != 1 {
		t.Fatal("a recorded withdrawal is not sent again")
	}
}

// A withdrawal that could not be published is retried by the next pass.
func TestFailedProposalWithdrawalIsRetried(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	decideInSwarmManager(t, service, session.ID, proposal.ID, "drop")

	publisher.err = errors.New("vrooli-events is down")
	reconcileAll(t, service)
	if got := publicationOf(t, service, session.ID, proposal.ID); got.WithdrawnAt != "" || got.WithdrawalAttempts != 1 || got.WithdrawalError == "" {
		t.Fatalf("the failure is recorded: %+v", got)
	}
	publisher.err = nil
	reconcileAll(t, service)
	if got := publicationOf(t, service, session.ID, proposal.ID); got.WithdrawnAt == "" || got.WithdrawalError != "" || got.WithdrawalAttempts != 2 {
		t.Fatalf("the retry succeeds and clears the error: %+v", got)
	}
}

// A decision raised before subject keys has none to name: its withdrawal names the
// request event id its frozen snapshot holds.
func TestPreSubjectKeyProposalDecisionIsWithdrawnByRequestEventID(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// A proposal whose decision the earlier code froze and published: the same
	// snapshot, without a subject key.
	proposal := Proposal{
		ID: "prop_before_subject_keys", Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: "Earlier",
		PayloadJSON: readyMutationPayload, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339(),
		Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	}
	publication, err := service.newProposalDecisionPublication(context.Background(), session.ID, proposal)
	if err != nil {
		t.Fatal(err)
	}
	delete(publication.Event.Payload, "subject_key")
	publication.EventID = operatordecision.EventID(publication.Event)
	publication.State, publication.PublishedAt = ProposalDecisionPublicationPublished, nowRFC3339()
	proposal.DecisionPublication = publication
	if err := store.SaveProposal(session.ID, proposal); err != nil {
		t.Fatal(err)
	}
	decideInSwarmManager(t, service, session.ID, proposal.ID, "drop")
	reconcileAll(t, service)

	got := withdrawalsIn(publisher.events)
	if len(got) != 1 || got[0].Payload["requester_ref"] != publication.EventID {
		t.Fatalf("withdrawn by the request event id %q: %+v", publication.EventID, got)
	}
	if _, keyed := got[0].Payload["subject_key"]; keyed {
		t.Fatalf("no subject key to name: %+v", got[0].Payload)
	}
}

// A withdrawn resolution is no answer: it applies nothing and the proposal stays in review.
func TestWithdrawnProposalResolutionAppliesNothing(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	res := currentAnsweredResolution(t, service, session.ID, proposal.ID, "approve")
	res.State = operatordecision.StateWithdrawn
	handled, err := effect.Apply(context.Background(), res)
	if err != nil || !handled {
		t.Fatalf("a withdrawn resolution is acknowledged: handled=%v err=%v", handled, err)
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("a withdrawn decision never approves: %v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusReady)
}
