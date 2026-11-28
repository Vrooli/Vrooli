package supervision

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/goalhome"

	"github.com/vrooli/api-core/eventbus"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func (p *recordingPublisher) withdrawals() (out []eventbus.DomainEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.events {
		if e.EventType == "agent_manager.effort.decision_withdrawn.v1" {
			out = append(out, e)
		}
	}
	return out
}

func TestDesktopAnswerClosesThePhoneAskOnce(t *testing.T) {
	s, publisher, _, _ := decisionFixture(t)
	ctx := context.Background()
	out, err := s.RaiseDecision(ctx, decisionRequest(), "")
	if err != nil {
		t.Fatal(err)
	}
	answer := &pb.AnswerEffortDecisionRequest{GoalHome: decisionHome, DecisionId: "BAS-D-070", Answer: "close"}
	actor := EffortActor{ID: "owner", Operator: true}
	for range 2 {
		if _, err := s.AnswerDecision(ctx, answer, actor); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	w := publisher.withdrawals()
	if len(w) != 1 {
		t.Fatalf("withdrawals = %d, want exactly one for the one answer", len(w))
	}
	if w[0].Payload["requester_ref"] != out.Decision.EventId || w[0].Payload["source_event_type"] != goalhome.DecisionRequestedEventType {
		t.Fatalf("withdrawal must close the request by its event id: %+v", w[0].Payload)
	}
	if reason, _ := w[0].Payload["reason"].(string); reason != decisionAnsweredHereReason {
		t.Fatalf("reason = %q", reason)
	}
	if _, ok := w[0].Payload["subject_key"]; ok {
		t.Fatal("effort decisions are closed by event id, not by a subject key")
	}
}

func TestFailedDecisionWithdrawalRetriesOnTheNextTick(t *testing.T) {
	s, publisher, _, _ := decisionFixture(t)
	ctx := context.Background()
	if _, err := s.RaiseDecision(ctx, decisionRequest(), ""); err != nil {
		t.Fatal(err)
	}
	publisher.fail = errors.New("vrooli-events unavailable")
	if _, err := s.AnswerDecision(ctx, &pb.AnswerEffortDecisionRequest{GoalHome: decisionHome, DecisionId: "BAS-D-070", Answer: "close"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	if len(publisher.withdrawals()) != 0 {
		t.Fatal("nothing was accepted yet")
	}
	publisher.fail = nil
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	if got := len(publisher.withdrawals()); got != 1 {
		t.Fatalf("withdrawals after retry = %d, want 1", got)
	}
}

func TestEndedEffortClosesItsPendingDecision(t *testing.T) {
	s, publisher, effortRef, _ := decisionFixture(t)
	ctx := context.Background()
	out, err := s.RaiseDecision(ctx, decisionRequest(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	if len(publisher.withdrawals()) != 0 {
		t.Fatal("a decision of a live effort must stay open")
	}
	e, _, err := s.repo.GetEffort(ctx, effortRef)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: effortRef, ExpectedRevision: e.Revision, IdempotencyKey: "end", Reason: "owner decision"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.publishPendingDecisions(ctx); err != nil {
			t.Fatal(err)
		}
	}
	w := publisher.withdrawals()
	if len(w) != 1 || w[0].Payload["requester_ref"] != out.Decision.EventId || w[0].Payload["reason"] != decisionEffortEndedReason {
		t.Fatalf("withdrawals = %+v", w)
	}
}

func TestWithdrawnResolutionLeavesTheDecisionUntouched(t *testing.T) {
	s, publisher, _, _ := decisionFixture(t)
	ctx := context.Background()
	out, err := s.RaiseDecision(ctx, decisionRequest(), "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.ResolveDecisionFromHub(ctx, DecisionResolution{AskID: "ask-1", State: "withdrawn", Answer: "close", RequesterRef: out.Decision.EventId, SourceEventType: goalhome.DecisionRequestedEventType})
	if err != nil {
		t.Fatal(err)
	}
	if d != nil && (d.State != DecisionPending || d.Answer != "") {
		t.Fatalf("a withdrawn resolution changed the decision: %+v", d)
	}
	if len(publisher.withdrawals()) != 0 {
		t.Fatal("a hub-side close must not trigger another withdrawal")
	}
}

func TestDecisionNeverPublishedHasNoAskToClose(t *testing.T) {
	s, publisher, _, _ := decisionFixture(t)
	ctx := context.Background()
	publisher.fail = errors.New("vrooli-events unavailable")
	if _, err := s.RaiseDecision(ctx, decisionRequest(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AnswerDecision(ctx, &pb.AnswerEffortDecisionRequest{GoalHome: decisionHome, DecisionId: "BAS-D-070", Answer: "close"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	publisher.fail = nil
	_ = s.publishPendingDecisions(ctx)
	if len(publisher.withdrawals()) != 0 {
		t.Fatal("a request that never reached the hub needs no withdrawal")
	}
}
