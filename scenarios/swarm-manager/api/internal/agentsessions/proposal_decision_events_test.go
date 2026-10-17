package agentsessions

import (
	"context"
	"strings"
	"testing"

	"swarm-manager/internal/attempt"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// capturingPublisher records every decision event the service raises, so a test
// can assert exactly-one-decision and inspect the facts/correlation.
type capturingPublisher struct {
	events []eventbus.DomainEvent
	err    error
}

func (p *capturingPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	return p.err
}

func newProposalDecisionService(t *testing.T) (*Service, *capturingPublisher, *fakeMutationProposalProcessor) {
	t.Helper()
	processor := &fakeMutationProposalProcessor{application: MutationProposalApplication{Outcomes: []MutationOutcome{{MutationID: "m1", Applied: true}}}}
	service := newTestService(t, &fakeSessionSpawner{})
	service.SetMutationProposalProcessor(processor)
	publisher := &capturingPublisher{}
	service.SetDecisionEventPublisher(publisher)
	return service, publisher, processor
}

const readyMutationPayload = `{"form":"mutation_list","rationale":"close the gate gap","mutations":[{"id":"m1","op":"add_item","target":"goal/quality-gates","rationale":"add missing coverage work"}]}`

func recordReadyMutationProposal(t *testing.T, service *Service, sessionID string) Proposal {
	t.Helper()
	proposal, err := service.RecordProposal(context.Background(), sessionID, Proposal{
		Kind:        ProposalMutationList,
		Status:      ProposalStatusReady,
		Summary:     "Mutation proposal for Quality Gates",
		PayloadJSON: readyMutationPayload,
		Target:      &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	})
	if err != nil {
		t.Fatalf("RecordProposal: %v", err)
	}
	return proposal
}

// A ready mutation-list proposal raises exactly ONE decision on the rail, and a
// re-raise of the same ready proposal is idempotent (same event id).
func TestReadyProposalRaisesExactlyOneDecision(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)

	if len(publisher.events) != 1 {
		t.Fatalf("expected exactly one decision event, got %d", len(publisher.events))
	}
	event := publisher.events[0]
	if event.EventType != ProposalDecisionEventType {
		t.Fatalf("event type = %q, want %q", event.EventType, ProposalDecisionEventType)
	}
	if event.Source != ProposalDecisionSource {
		t.Fatalf("event source = %q, want %q", event.Source, ProposalDecisionSource)
	}
	correlation, _ := event.Payload["correlation"].(map[string]any)
	if got := correlation["subject_ref"]; got != session.ID+"/"+proposal.ID {
		t.Fatalf("subject_ref = %v, want %s", got, session.ID+"/"+proposal.ID)
	}
	if got := correlation["subject_kind"]; got != AttemptSubjectProposal {
		t.Fatalf("subject_kind = %v, want %s", got, AttemptSubjectProposal)
	}
	if got, _ := event.Payload["question"].(string); !strings.Contains(got, "m1") || !strings.Contains(got, "Quality Gates") {
		t.Fatalf("question missing mutation id or target: %q", got)
	}

	// Idempotency: re-raising the SAME ready proposal yields the same event id, so
	// notification-hub dedupes to one operator notification.
	service.emitProposalDecisionRequested(context.Background(), session.ID, proposal)
	if len(publisher.events) != 2 {
		t.Fatalf("expected a second raise to be captured, got %d", len(publisher.events))
	}
	if first, second := operatordecision.EventID(publisher.events[0]), operatordecision.EventID(publisher.events[1]); first != second {
		t.Fatalf("re-raise event id differs: %s vs %s (not idempotent)", first, second)
	}
}

// A needs-revision proposal is not an operator decision and raises nothing; a
// no-change recommendation is reviewed on the keep-as-is path, not this rail.
func TestNonReadyMutationProposalRaisesNoDecision(t *testing.T) {
	service, publisher, _ := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")

	if _, err := service.RecordProposal(context.Background(), session.ID, Proposal{
		Kind:          ProposalMutationList,
		Status:        ProposalStatusNeedsRevision,
		Summary:       "Revision requested for Quality Gates",
		NeedsRevision: true,
		PayloadJSON:   readyMutationPayload,
		Target:        &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	}); err != nil {
		t.Fatalf("RecordProposal(needs_revision): %v", err)
	}
	if _, err := service.RecordProposal(context.Background(), session.ID, Proposal{
		Kind:        ProposalNoChangeRecommendation,
		Status:      ProposalStatusReady,
		Summary:     "Keep Quality Gates as-is",
		PayloadJSON: `{"form":"mutation_list","mutations":[]}`,
		Target:      &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"},
	}); err != nil {
		t.Fatalf("RecordProposal(no_change): %v", err)
	}
	if len(publisher.events) != 0 {
		t.Fatalf("expected no decision events for non-ready/no-change proposals, got %d", len(publisher.events))
	}
}

// An APPROVE answer applies the proposal's candidate mutation ids through the
// single authority; the proposal ends applied.
func TestProposalDecisionApproveApplies(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	handled, err := effect.Apply(context.Background(), answeredResolution(session.ID, proposal.ID, "approve"))
	if err != nil {
		t.Fatalf("Apply(approve): %v", err)
	}
	if !handled {
		t.Fatal("approve resolution was not handled")
	}
	if len(processor.accepted) != 1 || processor.accepted[0] != "m1" {
		t.Fatalf("authority received accepted ids %v, want [m1]", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusApplied)
}

// A REJECT answer records a rejection (no mutation ids accepted); the proposal
// ends rejected and the authority applies nothing.
func TestProposalDecisionRejectRecordsRejection(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	handled, err := effect.Apply(context.Background(), answeredResolution(session.ID, proposal.ID, "reject"))
	if err != nil {
		t.Fatalf("Apply(reject): %v", err)
	}
	if !handled {
		t.Fatal("reject resolution was not handled")
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("authority should apply nothing on reject, got accepted %v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusRejected)
}

// A STALE proposal at resolution time (already decided) is a no-op: the authority
// is never called a second time and no error escapes, so vrooli-events does not
// retry a poison message.
func TestProposalDecisionStaleAtResolutionIsNoOp(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	// First answer applies the proposal (now applied — no longer ready).
	if _, err := effect.Apply(context.Background(), answeredResolution(session.ID, proposal.ID, "approve")); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	acceptedAfterFirst := len(processor.accepted)
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusApplied)

	// A redelivered resolution for the now-stale proposal must NOT re-apply.
	handled, err := effect.Apply(context.Background(), answeredResolution(session.ID, proposal.ID, "approve"))
	if err != nil {
		t.Fatalf("stale Apply returned error (would poison-retry): %v", err)
	}
	if !handled {
		t.Fatal("stale resolution should be acknowledged (handled=true)")
	}
	if len(processor.accepted) != acceptedAfterFirst {
		t.Fatalf("authority was called again for a stale proposal: accepted %v", processor.accepted)
	}
}

// An unanswered (expired/defaulted) resolution leaves the proposal untouched.
func TestProposalDecisionUnansweredIsNoOp(t *testing.T) {
	service, _, processor := newProposalDecisionService(t)
	session := createStartedSession(t, service, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal := recordReadyMutationProposal(t, service, session.ID)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)

	res := answeredResolution(session.ID, proposal.ID, "approve")
	res.State = operatordecision.StateExpired
	handled, err := effect.Apply(context.Background(), res)
	if err != nil {
		t.Fatalf("Apply(expired): %v", err)
	}
	if !handled {
		t.Fatal("expired resolution should be acknowledged")
	}
	if len(processor.accepted) != 0 {
		t.Fatalf("no mutation should apply for an unanswered resolution, got %v", processor.accepted)
	}
	assertProposalStatus(t, service, session.ID, proposal.ID, ProposalStatusReady)
}

// A foreign source event type is not this effect's to handle.
func TestProposalDecisionEffectIgnoresForeignEventType(t *testing.T) {
	service, _, _ := newProposalDecisionService(t)
	effect := NewProposalDecisionEffect(attemptRouterFor(t, service), service)
	if effect.Handles("agent_manager.run.decision_requested.v1") {
		t.Fatal("effect must not claim a foreign source event type")
	}
	res := answeredResolution("s", "p", "approve")
	res.SourceEventType = "agent_manager.run.decision_requested.v1"
	handled, err := effect.Apply(context.Background(), res)
	if err != nil {
		t.Fatalf("Apply(foreign): %v", err)
	}
	if handled {
		t.Fatal("effect claimed a foreign resolution")
	}
}

func attemptRouterFor(t *testing.T, service *Service) *attempt.Router {
	t.Helper()
	router := attempt.NewRouter()
	if err := router.Register(AttemptSubjectProposal, NewAttemptDecider(service)); err != nil {
		t.Fatalf("register attempt decider: %v", err)
	}
	return router
}

func answeredResolution(sessionID, proposalID, answer string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		State:           operatordecision.StateAnswered,
		Answer:          answer,
		AnsweredBy:      "operator@example.test",
		Note:            "reviewed on the rail",
		SourceEventType: ProposalDecisionEventType,
		Correlation: map[string]string{
			"subject_kind": AttemptSubjectProposal,
			"subject_ref":  sessionID + "/" + proposalID,
			"session_id":   sessionID,
			"proposal_id":  proposalID,
		},
	}
}

func assertProposalStatus(t *testing.T, service *Service, sessionID, proposalID string, want ProposalStatus) {
	t.Helper()
	store, err := service.storeFor(context.Background())
	if err != nil {
		t.Fatalf("storeFor: %v", err)
	}
	session, err := store.LoadSession(sessionID)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	proposal, ok := findProposal(session, proposalID)
	if !ok {
		t.Fatalf("proposal %s not found", proposalID)
	}
	if proposal.Status != want {
		t.Fatalf("proposal status = %q, want %q", proposal.Status, want)
	}
}
