package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
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

func (c *capturingStandingGrants) ListForApprover(context.Context, string, int) ([]*domain.StandingGrant, error) {
	return nil, nil
}

// capturingAuthPublisher records raised decision events.
type capturingAuthPublisher struct {
	events []eventbus.DomainEvent
	err    error
}

func (c *capturingAuthPublisher) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	if c.err != nil {
		return c.err
	}
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

func TestRaiseAuthorizationApprovalRetriesFailedPublication(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{err: errors.New("events unavailable")}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)
	req := standingAskRequest()
	if raised, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil || !raised {
		t.Fatalf("pending ask must survive a publication error: raised=%v err=%v", raised, err)
	}
	id := authorizationCorrelationID(req.Subject, req.Operation, req.Target)
	firstID := pending.byID[id].EventID
	if firstID == "" || pending.byID[id].PublishedAt != nil {
		t.Fatalf("failed publication receipt state: %+v", pending.byID[id])
	}
	pub.err = nil
	if raised, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil || !raised {
		t.Fatalf("retry: raised=%v err=%v", raised, err)
	}
	if len(pub.events) != 1 || operatordecision.EventID(pub.events[0]) != firstID || pending.byID[id].PublishedAt == nil {
		t.Fatalf("retry did not publish the same event and persist receipt: events=%d row=%+v", len(pub.events), pending.byID[id])
	}
	if _, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil || len(pub.events) != 1 {
		t.Fatalf("successful publication must suppress repeat: events=%d err=%v", len(pub.events), err)
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

// Interactive consent is a generic FYI that opens the native authenticated
// consent page. Its durable snapshot drives retries; Hub answers cannot issue
// the interactive grant.
func TestNotifyAuthorizationRequestedPersistsAndPublishesFYI(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	repo := database.NewAuthorizationRepository(db)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	pub := &capturingAuthPublisher{}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	g := &domain.AuthorizationGrant{ID: uuid.New(), RunID: uuid.New(), State: domain.AuthorizationPending, Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 16 * 60 * 60, HarnessKind: "codex", HarnessSession: "fixture", CreatedAt: now, RequestExpiresAt: now.Add(24 * time.Hour)}
	if err := repo.Create(context.Background(), g); err != nil {
		t.Fatal(err)
	}
	o.NotifyAuthorizationRequested(context.Background(), g)
	o.NotifyAuthorizationRequested(context.Background(), g)
	if len(pub.events) != 1 {
		t.Fatalf("successful notification receipt must suppress duplicate publication, got %d events", len(pub.events))
	}
	ev := pub.events[0]
	if ev.EventType != InteractiveAuthorizationNotificationEventType || ev.EventType == AuthorizationDecisionEventType {
		t.Fatalf("interactive push must use generic FYI contract, got %q", ev.EventType)
	}
	link, _ := ev.Payload["context_url"].(string)
	if want := "https://agent-manager.example.test/api/v1/authorizations/" + g.ID.String() + "/approve"; link != want {
		t.Fatalf("consent link = %q, want %q", link, want)
	}
	body, _ := ev.Payload["body"].(string)
	for _, must := range []string{"codex", "swarm-bookkeeping", "swarm-manager:backlog/execute/obsolete", "16-hour", "starts only after you approve", "grants no authority"} {
		if !strings.Contains(body, must) {
			t.Fatalf("FYI body does not state %q: %s", must, body)
		}
	}
	if ev.Payload["options"] != nil || ev.Payload["question"] != nil {
		t.Fatalf("FYI notification must not expose Hub decision choices: %#v", ev.Payload)
	}
	stored, err := repo.Get(context.Background(), g.ID)
	if err != nil || stored.NotificationSnapshot == nil || stored.NotificationPublishedAt == nil || stored.NotificationSnapshot.EventID != operatordecision.EventID(ev) {
		t.Fatalf("notification snapshot/receipt not durable: %+v err=%v", stored, err)
	}
}

func TestNotifyAuthorizationRequestedRetriesSameSnapshotAfterFailure(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	repo := database.NewAuthorizationRepository(db)
	now := time.Now().UTC()
	pub := &capturingAuthPublisher{err: errors.New("events unavailable")}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	g := &domain.AuthorizationGrant{ID: uuid.New(), RunID: uuid.New(), State: domain.AuthorizationPending, Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 57600, HarnessKind: "codex", CreatedAt: now, RequestExpiresAt: now.Add(24 * time.Hour)}
	if err := repo.Create(context.Background(), g); err != nil {
		t.Fatal(err)
	}
	o.NotifyAuthorizationRequested(context.Background(), g)
	failed, err := repo.Get(context.Background(), g.ID)
	if err != nil || failed.NotificationSnapshot == nil || failed.NotificationPublishedAt != nil {
		t.Fatalf("failed publication lost its immutable retry snapshot: %+v err=%v", failed, err)
	}
	firstID := failed.NotificationSnapshot.EventID
	firstLink := failed.NotificationSnapshot.Payload["context_url"]
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://changed-origin.example.test")
	pub.err = nil
	o.NotifyAuthorizationRequested(context.Background(), g)
	success, err := repo.Get(context.Background(), g.ID)
	if err != nil || success.NotificationPublishedAt == nil || len(pub.events) != 1 || operatordecision.EventID(pub.events[0]) != firstID || pub.events[0].Payload["context_url"] != firstLink {
		t.Fatalf("retry did not publish the same event and record success: %+v events=%d err=%v", success, len(pub.events), err)
	}
	o.NotifyAuthorizationRequested(context.Background(), g)
	if len(pub.events) != 1 {
		t.Fatalf("success receipt did not suppress retry, got %d events", len(pub.events))
	}
}

func TestAuthorizationPublicBaseRejectsUntrustedOrigins(t *testing.T) {
	for _, origin := range []string{"https://user@agent-manager.example.test", "https://agent-manager.example.test/?redirect=https://attacker.test", "http://agent-manager.example.test", "javascript:alert(1)"} {
		t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", origin)
		if base, err := authorizationPublicBase(context.Background()); err == nil {
			t.Errorf("origin %q unexpectedly accepted as %q", origin, base)
		}
	}
}

func TestNotifyAuthorizationRequestedMissingOriginIsUnreached(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "")
	oldResolver := resolveExternalURLForNotice
	resolveExternalURLForNotice = func(context.Context, string) (string, error) { return "", errors.New("no public route") }
	t.Cleanup(func() { resolveExternalURLForNotice = oldResolver })
	repo := database.NewAuthorizationRepository(db)
	pub := &capturingAuthPublisher{}
	now := time.Now().UTC()
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	g := &domain.AuthorizationGrant{ID: uuid.New(), State: domain.AuthorizationPending, Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600, CreatedAt: now, RequestExpiresAt: now.Add(time.Hour)}
	if err := repo.Create(context.Background(), g); err != nil {
		t.Fatal(err)
	}
	o.NotifyAuthorizationRequested(context.Background(), g)
	stored, err := repo.Get(context.Background(), g.ID)
	if err != nil || stored.NotificationUnreached == "" || stored.NotificationSnapshot != nil || stored.NotificationPublishedAt != nil || len(pub.events) != 0 {
		t.Fatalf("missing origin must remain explicitly unreached without a broken-link publish: %+v events=%d err=%v", stored, len(pub.events), err)
	}
}

// G3 negative: the interactive decision event is PUSH-ONLY. A hub resolution
// carrying the interactive event type must NOT be handled by the standing-grant
// resolver (it would otherwise mint a grant, bypassing the independent-human +
// process-binding gates of ApproveAuthorization). It falls through unhandled.
func TestInteractiveAuthorizationEventIsNotHubResolvable(t *testing.T) {
	now := time.Now()
	grants := &capturingStandingGrants{}
	o := phase4Orchestrator(newCapturingPending(), grants, &capturingAuthPublisher{}, now)

	res := askResolution("some-authorization-id", "approve", "operator-human")
	res.SourceEventType = InteractiveAuthorizationNotificationEventType
	handled, err := o.ResolveAuthorizationFromHub(context.Background(), res)
	if handled || err != nil {
		t.Fatalf("interactive event must fall through unhandled (handled=%v err=%v)", handled, err)
	}
	if len(grants.put) != 0 {
		t.Fatalf("no grant may be minted from an interactive push event, got %d", len(grants.put))
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

func generationResolution(id, eventID, answer, answeredBy string) supervision.DecisionResolution {
	res := askResolution(id, answer, answeredBy)
	res.RequesterRef = eventID
	return res
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

	handled, err := o.ResolveAuthorizationFromHub(context.Background(), generationResolution(id, pending.byID[id].EventID, "approve", "operator-human"))
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
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), generationResolution(id, pending.byID[id].EventID, "approve", "")); !handled || err == nil {
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
	handled, err := o.ResolveAuthorizationFromHub(context.Background(), generationResolution(id, pending.byID[id].EventID, "reject", "operator-human"))
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

func TestResolveAuthorizationRefusesStaleGeneration(t *testing.T) {
	firstAt := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	clock := firstAt
	pending := newCapturingPending()
	grants := &capturingStandingGrants{}
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, grants, pub, clock)
	o.clock = func() time.Time { return clock }
	req := standingAskRequest()
	if _, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	id := authorizationCorrelationID(req.Subject, req.Operation, req.Target)
	oldEventID := pending.byID[id].EventID
	clock = firstAt.Add(pendingAuthorizationWindow + time.Hour)
	if _, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	newEventID := pending.byID[id].EventID
	if oldEventID == newEventID {
		t.Fatalf("replacement request must have a different event identity: %q", oldEventID)
	}
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), generationResolution(id, oldEventID, "approve", "operator-human")); !handled || err != nil {
		t.Fatalf("stale answer should be acknowledged without effect: handled=%v err=%v", handled, err)
	}
	if len(grants.put) != 0 || pending.byID[id].State != domain.PendingAuthorizationPending {
		t.Fatalf("stale answer changed current generation: grants=%d pending=%+v", len(grants.put), pending.byID[id])
	}
	if handled, err := o.ResolveAuthorizationFromHub(context.Background(), generationResolution(id, newEventID, "approve", "operator-human")); !handled || err != nil || len(grants.put) != 1 {
		t.Fatalf("current-generation control failed: handled=%v err=%v grants=%d", handled, err, len(grants.put))
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
