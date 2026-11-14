package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

func (c *capturingAuthPublisher) withdrawals() (out []eventbus.DomainEvent) {
	for _, e := range c.events {
		if e.EventType == "agent_manager.authorization.decision_withdrawn.v1" {
			out = append(out, e)
		}
	}
	return out
}

func (c *capturingAuthPublisher) requests() (out []eventbus.DomainEvent) {
	for _, e := range c.events {
		if e.EventType == AuthorizationDecisionEventType {
			out = append(out, e)
		}
	}
	return out
}

func askID() string {
	req := standingAskRequest()
	return authorizationCorrelationID(req.Subject, req.Operation, req.Target)
}

func TestAuthorizationAskIsKeyedPerCaller(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	event := pub.requests()[0]
	if got := event.Payload["subject_key"]; got != "authorization:"+askID() {
		t.Fatalf("subject_key = %v", got)
	}
	if event.Payload["asserted_at"] == nil {
		t.Fatal("a keyed request carries asserted_at so a later withdrawal outranks it")
	}
}

func TestGrantIssuedElsewhereClosesTheOpenAsk(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	grants := &capturingStandingGrants{}
	o := phase4Orchestrator(pending, grants, pub, now)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	if _, err := o.CreateStandingGrant(context.Background(), StandingGrantRequest{Subject: "prompt-manager", Approver: "operator", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	w := pub.withdrawals()
	if len(w) != 1 || w[0].Payload["subject_key"] != "authorization:"+askID() || w[0].Payload["source_event_type"] != AuthorizationDecisionEventType {
		t.Fatalf("withdrawals = %+v", w)
	}
	if pending.byID[askID()].State != domain.PendingAuthorizationWithdrawn {
		t.Fatalf("pending state = %s", pending.byID[askID()].State)
	}
	// Idempotent: another grant for the subject withdraws nothing more.
	if _, err := o.CreateStandingGrant(context.Background(), StandingGrantRequest{Subject: "prompt-manager", Approver: "operator", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if len(pub.withdrawals()) != 1 {
		t.Fatalf("withdrawal repeated: %d", len(pub.withdrawals()))
	}
	// And the closed ask can no longer issue anything.
	res := generationResolution(askID(), pending.byID[askID()].EventID, "approve", "operator-human")
	before := len(grants.put)
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), res); !handled || err != nil || len(grants.put) != before {
		t.Fatalf("a closed ask issued a grant: handled=%v err=%v grants=%d->%d", handled, err, before, len(grants.put))
	}
}

func TestApprovalFromTheHubDoesNotWithdrawItsOwnAsk(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	if _, err := o.ResolveAuthorizationFromHub(context.Background(), generationResolution(askID(), pending.byID[askID()].EventID, "approve", "operator-human")); err != nil {
		t.Fatal(err)
	}
	if len(pub.withdrawals()) != 0 || pending.byID[askID()].State != domain.PendingAuthorizationApproved {
		t.Fatalf("answered ask was withdrawn: %d state=%s", len(pub.withdrawals()), pending.byID[askID()].State)
	}
}

func TestWithdrawnResolutionIssuesNoGrant(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	grants := &capturingStandingGrants{}
	o := phase4Orchestrator(pending, grants, &capturingAuthPublisher{}, now)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	res := generationResolution(askID(), pending.byID[askID()].EventID, "approve", "operator-human")
	res.State = operatordecision.StateWithdrawn
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), res); !handled || err != nil || len(grants.put) != 0 {
		t.Fatalf("withdrawn resolution: handled=%v err=%v grants=%d", handled, err, len(grants.put))
	}
	if pending.byID[askID()].State != domain.PendingAuthorizationPending {
		t.Fatalf("a withdrawn resolution must not decide the ask: %s", pending.byID[askID()].State)
	}
}

// publishedKeylessAsk stores a pending ask as the previous build left it:
// published, with no subject key.
func publishedKeylessAsk(t *testing.T, o *Orchestrator, pending *capturingPending, now time.Time) *domain.PendingAuthorizationRequest {
	t.Helper()
	req := standingAskRequest()
	legacy := &domain.PendingAuthorizationRequest{
		ID: askID(), Subject: req.Subject, Operation: req.Operation, Target: req.Target,
		Kind: domain.PendingAuthorizationKindStanding, GrantTTL: standingGrantIssueTTL, MaxConcurrentRuns: req.MaxConcurrentRuns,
		State: domain.PendingAuthorizationPending, RaisedAt: now, ExpiresAt: now.Add(pendingAuthorizationWindow),
	}
	legacy.DecisionBigPicture, legacy.DecisionTechnicalDetails = authorizationDecisionContext(legacy)
	legacy.EventID = operatordecision.EventID(o.authorizationDecisionEvent(context.Background(), legacy))
	published := now
	legacy.PublishedAt = &published
	if err := pending.Put(context.Background(), legacy); err != nil {
		t.Fatal(err)
	}
	return legacy
}

func TestKeylessAskIsClosedByEventIDWhenAGrantCoversIt(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)
	legacy := publishedKeylessAsk(t, o, pending, now)
	if _, err := o.CreateStandingGrant(context.Background(), StandingGrantRequest{Subject: "prompt-manager", Approver: "operator", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	w := pub.withdrawals()
	if len(w) != 1 || w[0].Payload["requester_ref"] != legacy.EventID || w[0].Payload["subject_key"] != nil {
		t.Fatalf("a keyless ask must be closed by its request event id: %+v", w)
	}
}

func TestKeylessStillNeededAskIsReplacedUnderTheKeyOnce(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)
	legacy := publishedKeylessAsk(t, o, pending, now)
	for range 3 {
		if raised, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil || !raised {
			t.Fatalf("raise: %v %v", raised, err)
		}
	}
	w := pub.withdrawals()
	if len(w) != 1 || w[0].Payload["requester_ref"] != legacy.EventID {
		t.Fatalf("the keyless request must be closed once by its event id: %+v", w)
	}
	requests := pub.requests()
	if len(requests) != 1 || requests[0].Payload["subject_key"] != "authorization:"+askID() {
		t.Fatalf("the ask must be published once under its key: %d", len(requests))
	}
	stored := pending.byID[askID()]
	if stored.SubjectKey == "" || stored.EventID == legacy.EventID || stored.PublishedAt == nil {
		t.Fatalf("stored ask = %+v", stored)
	}
}
