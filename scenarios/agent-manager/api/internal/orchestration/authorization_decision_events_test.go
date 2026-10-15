package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/supervision"

	"github.com/vrooli/api-core/eventbus"
)

// capturingPending is an in-memory PendingAuthorizationRepository that records
// what was stored so a test can inspect dedup/anti-flood/resolution.
type capturingPending struct {
	byID map[string]*domain.PendingAuthorizationRequest
}

func newCapturingPending() *capturingPending {
	return &capturingPending{byID: map[string]*domain.PendingAuthorizationRequest{}}
}

func (c *capturingPending) Get(_ context.Context, id string) (*domain.PendingAuthorizationRequest, error) {
	if r, ok := c.byID[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, nil
}

func (c *capturingPending) Put(_ context.Context, req *domain.PendingAuthorizationRequest) error {
	cp := *req
	c.byID[req.ID] = &cp
	return nil
}

func (c *capturingPending) CountOutstandingForSubject(_ context.Context, subject string, now time.Time) (int, error) {
	n := 0
	for _, r := range c.byID {
		if r.Subject == subject && r.Outstanding(now) {
			n++
		}
	}
	return n, nil
}

// capturingStandingGrants records grants a resolution issues.
type capturingStandingGrants struct{ put []*domain.StandingGrant }

func (c *capturingStandingGrants) Put(_ context.Context, g *domain.StandingGrant) error {
	c.put = append(c.put, g)
	return nil
}

func (c *capturingStandingGrants) ActiveForSubject(context.Context, string, time.Time) ([]*domain.StandingGrant, error) {
	return nil, nil
}

func (c *capturingStandingGrants) Revoke(context.Context, string, time.Time) (bool, error) {
	return false, nil
}

func (c *capturingStandingGrants) List(context.Context, int) ([]*domain.StandingGrant, error) {
	return nil, nil
}

// capturingAuthPublisher records raised decision events.
type capturingAuthPublisher struct{ events []eventbus.DomainEvent }

func (c *capturingAuthPublisher) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	c.events = append(c.events, e)
	return nil
}

func phase4Orchestrator(pending domain.PendingAuthorizationRepository, grants domain.StandingGrantRepository, pub ApprovalEventPublisher, now time.Time) *Orchestrator {
	return &Orchestrator{
		clock:                 func() time.Time { return now },
		standingGrants:        grants,
		pendingAuthorizations: pending,
		approvalPublisher:     pub,
	}
}

func standingAskRequest() AuthorizationApprovalRequest {
	return AuthorizationApprovalRequest{
		Subject:           "prompt-manager",
		Operation:         "run.create",
		Target:            "agent-manager:*",
		Kind:              domain.PendingAuthorizationKindStanding,
		MaxConcurrentRuns: 3,
	}
}

// Idempotent raise: repeating the same ask raises exactly one operator event and
// stores one pending row.
func TestRaiseAuthorizationApprovalIsIdempotent(t *testing.T) {
	now := time.Now()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)

	for i := 0; i < 3; i++ {
		raised, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest())
		if err != nil || !raised {
			t.Fatalf("raise %d: raised=%v err=%v", i, raised, err)
		}
	}
	if len(pub.events) != 1 {
		t.Fatalf("expected exactly one operator notification for a repeated ask, got %d", len(pub.events))
	}
	if len(pending.byID) != 1 {
		t.Fatalf("expected one pending row for a repeated ask, got %d", len(pending.byID))
	}
}

// Scope-in-question: the operator notification states the grant KIND, subject,
// bounds, and expiry so a one-tap approve is informed.
func TestAuthorizationAskStatesItsScope(t *testing.T) {
	now := time.Now()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(newCapturingPending(), &capturingStandingGrants{}, pub, now)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	if len(pub.events) != 1 {
		t.Fatalf("expected one event, got %d", len(pub.events))
	}
	q, _ := pub.events[0].Payload["question"].(string)
	for _, must := range []string{"prompt-manager", "run-creation", "max 3 concurrent runs", "30-day", "revocable"} {
		if !strings.Contains(q, must) {
			t.Fatalf("ask question does not state %q: %s", must, q)
		}
	}
}

// Anti-flood: beyond the per-subject outstanding cap, a NEW distinct ask is
// refused without raising another notification.
func TestRaiseAuthorizationApprovalAntiFloodCap(t *testing.T) {
	now := time.Now()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)

	// Fill the cap with distinct asks (distinct targets → distinct correlation).
	for i := 0; i < MaxOutstandingAuthorizationRequestsPerSubject; i++ {
		req := standingAskRequest()
		req.Target = "agent-manager:run/" + string(rune('a'+i))
		if raised, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil || !raised {
			t.Fatalf("fill %d: raised=%v err=%v", i, raised, err)
		}
	}
	eventsAtCap := len(pub.events)
	// One more DISTINCT ask → refused, no new notification.
	over := standingAskRequest()
	over.Target = "agent-manager:run/over"
	raised, err := o.RaiseAuthorizationApproval(context.Background(), over)
	if err == nil || raised {
		t.Fatalf("over the cap should refuse (raised=%v err=%v)", raised, err)
	}
	if len(pub.events) != eventsAtCap {
		t.Fatalf("over the cap must NOT raise another notification: was %d now %d", eventsAtCap, len(pub.events))
	}
}

func askResolution(id, answer, answeredBy string) supervision.DecisionResolution {
	return supervision.DecisionResolution{
		AskID:           "ask-1",
		State:           "answered",
		Answer:          answer,
		AnsweredBy:      answeredBy,
		SourceEventType: AuthorizationDecisionEventType,
		Correlation:     map[string]string{"authorization_request_id": id},
	}
}

// Approve → a bounded grant is issued, attributed to the approving human, and
// the pending request is marked approved.
func TestResolveAuthorizationApproveIssuesBoundedGrant(t *testing.T) {
	now := time.Now()
	pending := newCapturingPending()
	grants := &capturingStandingGrants{}
	o := phase4Orchestrator(pending, grants, &capturingAuthPublisher{}, now)

	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	var id string
	for k := range pending.byID {
		id = k
	}

	handled, err := o.ResolveAuthorizationFromHub(context.Background(), askResolution(id, "approve", "operator-human"))
	if !handled || err != nil {
		t.Fatalf("approve resolution: handled=%v err=%v", handled, err)
	}
	if len(grants.put) != 1 {
		t.Fatalf("approve should issue exactly one grant, got %d", len(grants.put))
	}
	g := grants.put[0]
	if g.Grant == nil || g.Grant.Approver != "operator-human" || g.ApprovedBy != "operator-human" {
		t.Fatalf("issued grant not attributed to the approving human: %+v", g)
	}
	if g.Subject != "prompt-manager" || g.Limits.MaxConcurrentRuns != 3 {
		t.Fatalf("issued grant lost subject/bounds: %+v", g)
	}
	if !g.ExpiresAt.After(now) || g.ExpiresAt.After(now.Add(MaxStandingGrantLifetime+time.Minute)) {
		t.Fatalf("issued grant expiry not bounded: %v", g.ExpiresAt)
	}
	if pending.byID[id].State != domain.PendingAuthorizationApproved {
		t.Fatalf("pending request not marked approved: %s", pending.byID[id].State)
	}
}

// Approve with NO approver identity → refuse to issue (human-only issuance).
func TestResolveAuthorizationApproveRequiresApprover(t *testing.T) {
	now := time.Now()
	pending := newCapturingPending()
	grants := &capturingStandingGrants{}
	o := phase4Orchestrator(pending, grants, &capturingAuthPublisher{}, now)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	var id string
	for k := range pending.byID {
		id = k
	}
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), askResolution(id, "approve", "")); !handled || err == nil {
		t.Fatalf("approve without approver must error (handled=%v err=%v)", handled, err)
	}
	if len(grants.put) != 0 {
		t.Fatalf("no grant may be issued without an approver, got %d", len(grants.put))
	}
}

// Reject → no grant; pending marked rejected.
func TestResolveAuthorizationRejectIssuesNothing(t *testing.T) {
	now := time.Now()
	pending := newCapturingPending()
	grants := &capturingStandingGrants{}
	o := phase4Orchestrator(pending, grants, &capturingAuthPublisher{}, now)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), standingAskRequest()); err != nil {
		t.Fatal(err)
	}
	var id string
	for k := range pending.byID {
		id = k
	}
	handled, err := o.ResolveAuthorizationFromHub(context.Background(), askResolution(id, "reject", "operator-human"))
	if !handled || err != nil {
		t.Fatalf("reject resolution: handled=%v err=%v", handled, err)
	}
	if len(grants.put) != 0 {
		t.Fatalf("reject must issue no grant, got %d", len(grants.put))
	}
	if pending.byID[id].State != domain.PendingAuthorizationRejected {
		t.Fatalf("pending not marked rejected: %s", pending.byID[id].State)
	}
}

// Stale / unknown correlation or a non-authorization event → benign no-op that
// falls through (handled=false for a foreign event; handled=true no-op for an
// unknown ask of our type).
func TestResolveAuthorizationStaleIsBenign(t *testing.T) {
	o := phase4Orchestrator(newCapturingPending(), &capturingStandingGrants{}, &capturingAuthPublisher{}, time.Now())

	// Foreign event type → not ours, fall through.
	foreign := askResolution("whatever", "approve", "operator")
	foreign.SourceEventType = "agent_manager.run.decision_requested.v1"
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), foreign); handled || err != nil {
		t.Fatalf("foreign event should fall through (handled=false), got handled=%v err=%v", handled, err)
	}

	// Our event type but unknown ask id → handled no-op (acknowledge + drop).
	unknown := askResolution("auth-does-not-exist", "approve", "operator")
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), unknown); !handled || err != nil {
		t.Fatalf("unknown ask should be a benign handled no-op, got handled=%v err=%v", handled, err)
	}
}
