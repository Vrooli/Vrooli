package orchestration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/cli-core/cliutil"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
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

type contextBlockingAuthPublisher struct{ entered chan struct{} }

func (p *contextBlockingAuthPublisher) PublishDomainEvent(ctx context.Context, _ eventbus.DomainEvent) error {
	select {
	case p.entered <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return ctx.Err()
}

type advancingAuthPublisher struct {
	events  []eventbus.DomainEvent
	advance func()
}

func (p *advancingAuthPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	if p.advance != nil {
		p.advance()
	}
	return nil
}

type authorizationRepoWithoutRetryScan struct{ domain.AuthorizationRepository }

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

func TestLegacyPendingAuthorizationRetryKeepsOriginalEventBytes(t *testing.T) {
	now := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)
	req := standingAskRequest()
	legacy := &domain.PendingAuthorizationRequest{
		ID:      authorizationCorrelationID(req.Subject, req.Operation, req.Target),
		Subject: req.Subject, Operation: req.Operation, Target: req.Target,
		Kind: domain.PendingAuthorizationKindStanding, GrantTTL: standingGrantIssueTTL,
		MaxConcurrentRuns: req.MaxConcurrentRuns, State: domain.PendingAuthorizationPending,
		RaisedAt: now, ExpiresAt: now.Add(pendingAuthorizationWindow),
	}
	legacy.EventID = operatordecision.EventID(o.authorizationDecisionEvent(context.Background(), legacy))
	if err := pending.Put(context.Background(), legacy); err != nil {
		t.Fatal(err)
	}
	if raised, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil || !raised {
		t.Fatalf("retry legacy pending request: raised=%v err=%v", raised, err)
	}
	if len(pub.events) != 1 || operatordecision.EventID(pub.events[0]) != legacy.EventID {
		t.Fatalf("legacy retry changed the pending EventID: events=%d stored=%q", len(pub.events), legacy.EventID)
	}
	facts := pub.events[0].Payload
	if facts["big_picture"] != nil || facts["technical_details"] != nil {
		t.Fatalf("legacy retry must preserve the original context-free payload: %#v", pub.events[0].Payload)
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
	facts := pub.events[0].Payload
	bigPicture, _ := facts["big_picture"].(string)
	technical, _ := facts["technical_details"].(string)
	if facts == nil || bigPicture != pending.byID[id].DecisionBigPicture || technical != pending.byID[id].DecisionTechnicalDetails {
		t.Fatalf("new context was not persisted and published from the same pending generation: payload=%#v row=%+v", pub.events[0].Payload, pending.byID[id])
	}
	for _, required := range []string{
		"verified caller", "does not execute the requested operation", "Rejection issues no grant", "existing grants are unaffected",
	} {
		if !strings.Contains(bigPicture, required) {
			t.Fatalf("big picture omitted effect boundary %q: %s", required, bigPicture)
		}
	}
	for _, required := range []string{
		"Operation: run.create", "Target: agent-manager:*", "Grant kind: standing",
		"Grant lifetime: 30-day", "Concurrency: at most 3 concurrent runs", "revocable",
	} {
		if !strings.Contains(technical, required) {
			t.Fatalf("technical details omitted request bound %q: %s", required, technical)
		}
	}
	if _, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil || len(pub.events) != 1 {
		t.Fatalf("successful publication must suppress repeat: events=%d err=%v", len(pub.events), err)
	}
}

func TestAuthorizationDecisionContextStatesOneTimeSessionBounds(t *testing.T) {
	now := time.Now().UTC()
	pending := newCapturingPending()
	pub := &capturingAuthPublisher{}
	o := phase4Orchestrator(pending, &capturingStandingGrants{}, pub, now)
	req := AuthorizationApprovalRequest{
		Subject: "web-console-session", Operation: "run.create", Target: "agent-manager:task-123",
		Kind: domain.PendingAuthorizationKindOneTime, SessionID: "opaque-session-id-must-not-be-published",
		MaxConcurrentRuns: 1,
	}
	if raised, err := o.RaiseAuthorizationApproval(context.Background(), req); err != nil || !raised {
		t.Fatalf("raise one-time ask: raised=%v err=%v", raised, err)
	}
	if len(pub.events) != 1 {
		t.Fatalf("expected one event, got %d", len(pub.events))
	}
	facts := pub.events[0].Payload
	if facts == nil {
		t.Fatalf("unexpected event payload: %#v", pub.events[0].Payload)
	}
	bigPicture, _ := facts["big_picture"].(string)
	technical, _ := facts["technical_details"].(string)
	for _, required := range []string{
		"verified caller", "does not execute the requested operation", "Rejection issues no grant",
	} {
		if !strings.Contains(bigPicture, required) {
			t.Fatalf("big picture omitted effect boundary %q: %s", required, bigPicture)
		}
	}
	for _, required := range []string{
		"Operation: run.create", "Target: agent-manager:task-123", "Grant kind: one-time",
		"Grant lifetime: 1-hour", "Concurrency: at most 1 concurrent runs", "Grant scope: session-scoped",
	} {
		if !strings.Contains(technical, required) {
			t.Fatalf("technical details omitted request bound %q: %s", required, technical)
		}
	}
	if strings.Contains(technical, req.SessionID) {
		t.Fatalf("technical details leaked the opaque session identifier: %s", technical)
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

func TestSetReconcilerSurfacesMissingAuthorizationRetryScan(t *testing.T) {
	o := &Orchestrator{
		authorizations:    authorizationRepoWithoutRetryScan{},
		approvalPublisher: &capturingAuthPublisher{},
	}
	reconciler := &Reconciler{}
	o.SetReconciler(reconciler)
	if reconciler.authorizationNotificationRetry == nil {
		t.Fatal("configured publisher must install the retry callback so missing repository support is visible")
	}
	if _, err := reconciler.authorizationNotificationRetry(context.Background(), "", 100, 10); err == nil {
		t.Fatal("repository without bounded retry scan must report a visible error")
	}
	withoutRepository := &Orchestrator{approvalPublisher: &capturingAuthPublisher{}}
	if _, err := withoutRepository.retryPendingAuthorizationNotifications(context.Background(), "cursor", 100, 10); err == nil {
		t.Fatal("configured notification retry without an authorization repository must report a visible error")
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

func TestNotifyAuthorizationRequestedPublishesFreshSnapshotThroughEventBusClient(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	repo := database.NewAuthorizationRepository(db)
	now := time.Date(2026, 10, 18, 7, 0, 0, 0, time.UTC)
	g := &domain.AuthorizationGrant{
		ID: uuid.New(), RunID: uuid.New(), State: domain.AuthorizationPending,
		Policy: "run-creation", Targets: []string{"agent-manager:*"}, DurationSeconds: 3600,
		HarnessKind: "codex", CreatedAt: now, RequestExpiresAt: now.Add(24 * time.Hour),
	}
	if err := repo.Create(context.Background(), g); err != nil {
		t.Fatal(err)
	}

	var requests atomic.Int32
	receivedEvents := make(chan eventdomain.EventEnvelope, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/events" {
			t.Errorf("event request = %s %s, want POST /api/v1/events", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read event body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var received eventdomain.EventEnvelope
		if err := protojson.Unmarshal(body, &received); err != nil {
			t.Errorf("decode event envelope: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		select {
		case receivedEvents <- received:
		default:
			t.Errorf("received more decoded event envelopes than the test can inspect")
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	o := &Orchestrator{
		authorizations:    repo,
		approvalPublisher: eventbus.Client{BaseURL: server.URL, HTTPClient: server.Client()},
		clock:             func() time.Time { return now },
	}
	o.NotifyAuthorizationRequested(context.Background(), g)
	if got := requests.Load(); got != 1 {
		t.Fatalf("first notification should publish one event request, got %d", got)
	}
	var received eventdomain.EventEnvelope
	select {
	case received = <-receivedEvents:
	default:
		t.Fatal("the event server did not receive a decodable event envelope")
	}
	if received.EventType != InteractiveAuthorizationNotificationEventType {
		t.Fatalf("wire event type = %q, want %q", received.EventType, InteractiveAuthorizationNotificationEventType)
	}
	var payload structpb.Struct
	if received.Data == nil {
		t.Fatal("wire event omitted data")
	}
	if err := received.Data.UnmarshalTo(&payload); err != nil {
		t.Fatalf("decode event data: %v", err)
	}
	fields := payload.AsMap()
	if fields["title"] != "Agent needs approval: run-creation" {
		t.Fatalf("wire title = %#v", fields["title"])
	}
	body, ok := fields["body"].(string)
	if !ok || !strings.Contains(body, "run-creation") || !strings.Contains(body, "agent-manager:*") || !strings.Contains(body, "1-hour permission lifetime starts only after you approve") || !strings.Contains(body, "grants no authority") {
		t.Fatalf("wire body omitted the approved operation bounds or explicit non-authority statement: %#v", fields["body"])
	}
	wantURL := "https://agent-manager.example.test/api/v1/authorizations/" + g.ID.String() + "/approve"
	if fields["context_url"] != wantURL {
		t.Fatalf("wire context URL = %#v, want %q", fields["context_url"], wantURL)
	}
	correlation, ok := fields["correlation"].(map[string]any)
	if !ok {
		t.Fatalf("wire correlation has type %T, want object", fields["correlation"])
	}
	if correlation["authorization_id"] != g.ID.String() || correlation["run_id"] != g.RunID.String() || correlation["policy"] != g.Policy {
		t.Fatalf("wire correlation = %#v", correlation)
	}
	stored, err := repo.Get(context.Background(), g.ID)
	if err != nil || stored == nil || stored.NotificationSnapshot == nil || stored.NotificationPublishedAt == nil {
		t.Fatalf("first publication receipt was not persisted: stored=%+v err=%v", stored, err)
	}
	if received.EventId != stored.NotificationSnapshot.EventID || received.EventId != operatordecision.EventID(eventFromAuthorizationNotification(stored.NotificationSnapshot)) {
		t.Fatalf("wire event ID %q does not match durable snapshot %q", received.EventId, stored.NotificationSnapshot.EventID)
	}

	// The durable success receipt suppresses another HTTP publication.
	o.NotifyAuthorizationRequested(context.Background(), g)
	if got := requests.Load(); got != 1 {
		t.Fatalf("duplicate notify published another event, request count=%d", got)
	}
}

func TestReconcilerAutomaticallyRetriesAuthorizationNoticeWithSameEvent(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	repo := database.NewAuthorizationRepository(db)
	now := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	var attempts atomic.Int32
	events := make(chan eventdomain.EventEnvelope, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/events" {
			t.Errorf("event request = %s %s, want POST /api/v1/events", r.Method, r.URL.Path)
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read event body: %v", err)
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		var event eventdomain.EventEnvelope
		if err := protojson.Unmarshal(body, &event); err != nil {
			t.Errorf("decode event envelope: %v", err)
			http.Error(w, "bad envelope", http.StatusBadRequest)
			return
		}
		select {
		case events <- event:
		default:
			t.Errorf("more than two event attempts reached the fixture")
		}
		if attempts.Add(1) == 1 {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	pub := eventbus.Client{BaseURL: server.URL, HTTPClient: server.Client()}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, identitySecret: []byte("retry-test"), clock: func() time.Time { return now }}
	request := AuthorizationRequest{
		RunID: uuid.New(), PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "a221-session",
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
	}
	grant, err := o.RequestAuthorization(context.Background(), request)
	if err != nil || grant == nil {
		t.Fatalf("persist first pending request: grant=%+v err=%v", grant, err)
	}
	failed, err := repo.Get(context.Background(), grant.ID)
	if err != nil || failed.NotificationSnapshot == nil || failed.NotificationPublishedAt != nil || failed.NotificationUnreached == "" {
		t.Fatalf("first failed notice must keep its immutable snapshot and bounded failure: %+v err=%v", failed, err)
	}
	firstID := failed.NotificationSnapshot.EventID
	firstLink := failed.NotificationSnapshot.Payload["context_url"]
	var firstEnvelope eventdomain.EventEnvelope
	select {
	case firstEnvelope = <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("initial EventBus publication did not reach the HTTP fixture")
	}
	if firstEnvelope.EventId != firstID {
		t.Fatalf("first attempted EventID=%q, persisted snapshot=%q", firstEnvelope.EventId, firstID)
	}

	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	reconciler := NewReconciler(repos.Runs, nil)
	reconciler.clock = func() time.Time { return now }
	o.SetReconciler(reconciler)
	stats := reconciler.RunOnce(context.Background())
	if len(stats.Errors) != 0 {
		t.Fatalf("reconciler retry errors = %v", stats.Errors)
	}
	var secondEnvelope eventdomain.EventEnvelope
	select {
	case secondEnvelope = <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("reconciler did not retry the pending authorization notice")
	}
	if attempts.Load() != 2 || secondEnvelope.EventId != firstID {
		t.Fatalf("automatic retry attempts=%d EventID=%q, want same durable EventID %q", attempts.Load(), secondEnvelope.EventId, firstID)
	}
	stored, err := repo.Get(context.Background(), grant.ID)
	if err != nil || stored.NotificationPublishedAt == nil || stored.NotificationSnapshot.EventID != firstID || stored.NotificationSnapshot.Payload["context_url"] != firstLink || stored.NotificationUnreached != "" {
		t.Fatalf("reconciler did not record success for the unchanged snapshot: %+v err=%v", stored, err)
	}
	reconciler.RunOnce(context.Background())
	if attempts.Load() != 2 {
		t.Fatalf("published receipt did not suppress later reconciliation, attempts=%d", attempts.Load())
	}
}

func TestAuthorizationRetryRechecksExpiryBeforePublishing(t *testing.T) {
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repo := database.NewAuthorizationRepository(db)
	now := time.Date(2026, 10, 18, 7, 0, 0, 0, time.UTC)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	for i, expiry := range []time.Time{now.Add(time.Hour), now.Add(time.Minute)} {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1))
		grant := &domain.AuthorizationGrant{
			ID: id, State: domain.AuthorizationPending, Policy: "swarm-bookkeeping",
			Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
			Binding: binding, CreatedAt: now, RequestExpiresAt: expiry,
		}
		if err := repo.Create(context.Background(), grant); err != nil {
			t.Fatal(err)
		}
	}
	pub := &advancingAuthPublisher{advance: func() { now = now.Add(2 * time.Minute) }}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	if _, err := o.retryPendingAuthorizationNotifications(context.Background(), "", authorizationNotificationScanLimit, authorizationNotificationAttemptLimit); err != nil {
		t.Fatal(err)
	}
	if len(pub.events) != 1 {
		t.Fatalf("retry published a request that expired during the scan: events=%d", len(pub.events))
	}
	second, err := repo.Get(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000002"))
	if err != nil || second == nil || second.NotificationSnapshot != nil || second.NotificationPublishedAt != nil {
		t.Fatalf("request that expired before its publication was mutated: row=%+v err=%v", second, err)
	}
}

func TestAuthorizationRetryContextCancellationDoesNotAdvancePastInterruptedRow(t *testing.T) {
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repo := database.NewAuthorizationRepository(db)
	now := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= authorizationNotificationScanLimit; i++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i))
		grant := &domain.AuthorizationGrant{
			ID: id, State: domain.AuthorizationPending, Policy: "swarm-bookkeeping",
			Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
			Binding: binding, CreatedAt: now, RequestExpiresAt: now.Add(time.Hour),
		}
		if err := repo.Create(context.Background(), grant); err != nil {
			t.Fatal(err)
		}
	}
	blocking := &contextBlockingAuthPublisher{entered: make(chan struct{}, 1)}
	o := &Orchestrator{authorizations: repo, approvalPublisher: blocking, clock: func() time.Time { return now }}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-blocking.entered
		cancel()
	}()
	next, err := o.retryPendingAuthorizationNotifications(ctx, "", authorizationNotificationScanLimit, authorizationNotificationAttemptLimit)
	if !errors.Is(err, context.Canceled) || next != "" {
		t.Fatalf("cancelled publication advanced the retry keyset: next=%q err=%v", next, err)
	}
	firstID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	interrupted, err := repo.Get(context.Background(), firstID)
	if err != nil || interrupted == nil || interrupted.NotificationSnapshot == nil || interrupted.NotificationPublishedAt != nil || interrupted.NotificationUnreached == "" || len(interrupted.NotificationUnreached) > authorizationNotificationFailureLimit {
		t.Fatalf("interrupted row did not retain its retryable snapshot and bounded failure diagnostic: row=%+v err=%v", interrupted, err)
	}
	interruptedEventID := interrupted.NotificationSnapshot.EventID

	success := &capturingAuthPublisher{}
	o.approvalPublisher = success
	if _, err := o.retryPendingAuthorizationNotifications(context.Background(), next, authorizationNotificationScanLimit, authorizationNotificationAttemptLimit); err != nil {
		t.Fatal(err)
	}
	if len(success.events) != authorizationNotificationAttemptLimit {
		t.Fatalf("fresh cycle did not resume from the interrupted first row: published=%d", len(success.events))
	}
	first, err := repo.Get(context.Background(), firstID)
	if err != nil || first == nil || first.NotificationPublishedAt == nil || operatordecision.EventID(success.events[0]) != interruptedEventID || first.NotificationSnapshot.EventID != interruptedEventID {
		t.Fatalf("interrupted first row was skipped or its event identity changed: row=%+v err=%v", first, err)
	}
}

func TestReconcilerKeepsCompletedAuthorizationRetryCheckpointOnCallbackError(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	reconciler := NewReconciler(repos.Runs, nil)
	reconciler.authorizationNotificationCursor = "before"
	reconciler.authorizationNotificationRetry = func(context.Context, string, int, int) (string, error) {
		return "checkpoint", context.DeadlineExceeded
	}
	stats := reconciler.RunOnce(context.Background())
	if reconciler.authorizationNotificationCursor != "checkpoint" {
		t.Fatalf("reconciler discarded fully inspected retry checkpoint: %q", reconciler.authorizationNotificationCursor)
	}
	if len(stats.Errors) == 0 {
		t.Fatal("retry callback error was not recorded")
	}
}

func TestReconcilerAuthorizationNotificationRetryIsBoundedFairAndEligibilityChecked(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repo := database.NewAuthorizationRepository(db)
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	pub := &capturingAuthPublisher{}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	const retryScanLimit = 100
	for i := 1; i <= retryScanLimit+1; i++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i))
		grant := &domain.AuthorizationGrant{
			ID: id, State: domain.AuthorizationPending, Policy: "swarm-bookkeeping",
			Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
			Binding: binding, CreatedAt: now, RequestExpiresAt: now.Add(time.Hour),
		}
		if i <= retryScanLimit {
			grant.Binding.Start = "different-process-incarnation"
		}
		if err := repo.Create(context.Background(), grant); err != nil {
			t.Fatal(err)
		}
	}
	// The expired request remains stored as pending, but must not be notified.
	expired := &domain.AuthorizationGrant{
		ID: uuid.MustParse("00000000-0000-4000-8000-000000000102"), State: domain.AuthorizationPending,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"},
		DurationSeconds: 3600, Binding: binding, CreatedAt: now.Add(-2 * time.Hour), RequestExpiresAt: now.Add(-time.Second),
	}
	if err := repo.Create(context.Background(), expired); err != nil {
		t.Fatal(err)
	}
	terminal := *expired
	terminal.ID = uuid.MustParse("00000000-0000-4000-8000-000000000103")
	terminal.State = domain.AuthorizationSuperseded
	terminal.RequestExpiresAt = now.Add(time.Hour)
	if err := repo.Create(context.Background(), &terminal); err != nil {
		t.Fatal(err)
	}
	published := *expired
	published.ID = uuid.MustParse("00000000-0000-4000-8000-000000000104")
	published.RequestExpiresAt = now.Add(time.Hour)
	publishedAt := now.Add(-time.Minute)
	published.NotificationPublishedAt = &publishedAt
	if err := repo.Create(context.Background(), &published); err != nil {
		t.Fatal(err)
	}
	deadProcess := published
	deadProcess.ID = uuid.MustParse("00000000-0000-4000-8000-000000000105")
	deadProcess.NotificationPublishedAt = nil
	deadProcess.Binding = cliutil.ProcessBinding{PID: 2147483647, Start: "not-running"}
	if err := repo.Create(context.Background(), &deadProcess); err != nil {
		t.Fatal(err)
	}

	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	reconciler := NewReconciler(repos.Runs, nil)
	reconciler.clock = func() time.Time { return now }
	o.SetReconciler(reconciler)
	first := reconciler.RunOnce(context.Background())
	if len(first.Errors) != 0 || len(pub.events) != 0 {
		t.Fatalf("first bounded page should inspect only mismatched process incarnations: events=%d errors=%v", len(pub.events), first.Errors)
	}
	second := reconciler.RunOnce(context.Background())
	if len(second.Errors) != 0 || len(pub.events) != 1 {
		t.Fatalf("cursor did not reach the later live pending request: events=%d errors=%v", len(pub.events), second.Errors)
	}
	liveID := uuid.MustParse("00000000-0000-4000-8000-000000000065")
	liveStored, err := repo.Get(context.Background(), liveID)
	if err != nil || liveStored == nil || liveStored.NotificationSnapshot == nil || liveStored.NotificationPublishedAt == nil {
		t.Fatalf("later live request was not the retried row: stored=%+v err=%v", liveStored, err)
	}
	if got := operatordecision.EventID(pub.events[0]); got != liveStored.NotificationSnapshot.EventID {
		t.Fatalf("reconciler published event %q, want later live request %q", got, liveStored.NotificationSnapshot.EventID)
	}
	for _, id := range []uuid.UUID{expired.ID, terminal.ID, published.ID, deadProcess.ID} {
		stored, err := repo.Get(context.Background(), id)
		if err != nil || stored == nil {
			t.Fatalf("read eligibility control %s: stored=%+v err=%v", id, stored, err)
		}
		if id == published.ID && stored.NotificationPublishedAt == nil {
			t.Fatalf("published control lost its receipt: %+v", stored)
		}
		if id != published.ID && (stored.NotificationPublishedAt != nil || stored.NotificationSnapshot != nil) {
			t.Fatalf("ineligible request %s was notified: %+v", id, stored)
		}
	}
}

func TestReconcilerAuthorizationNotificationRetryAdvancesAfterAttemptCap(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	repo := database.NewAuthorizationRepository(db)
	now := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	pub := &capturingAuthPublisher{}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	ids := make([]uuid.UUID, 21)
	for i := range ids {
		ids[i] = uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1))
		grant := &domain.AuthorizationGrant{
			ID: ids[i], State: domain.AuthorizationPending, Policy: "swarm-bookkeeping",
			Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
			Binding: binding, CreatedAt: now, RequestExpiresAt: now.Add(time.Hour),
		}
		if err := repo.Create(context.Background(), grant); err != nil {
			t.Fatal(err)
		}
	}
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	reconciler := NewReconciler(repos.Runs, nil)
	reconciler.clock = func() time.Time { return now }
	o.SetReconciler(reconciler)
	if stats := reconciler.RunOnce(context.Background()); len(stats.Errors) != 0 {
		t.Fatalf("first retry cycle errors = %v", stats.Errors)
	}
	if len(pub.events) != authorizationNotificationAttemptLimit {
		t.Fatalf("first cycle published %d events, want attempt cap %d", len(pub.events), authorizationNotificationAttemptLimit)
	}
	for i := 0; i < authorizationNotificationAttemptLimit; i++ {
		stored, err := repo.Get(context.Background(), ids[i])
		if err != nil || stored.NotificationPublishedAt == nil {
			t.Fatalf("first page row %s was not published: stored=%+v err=%v", ids[i], stored, err)
		}
	}
	for i := authorizationNotificationAttemptLimit; i < len(ids); i++ {
		stored, err := repo.Get(context.Background(), ids[i])
		if err != nil || stored.NotificationPublishedAt != nil {
			t.Fatalf("later page row %s was touched before its turn: stored=%+v err=%v", ids[i], stored, err)
		}
	}
	if stats := reconciler.RunOnce(context.Background()); len(stats.Errors) != 0 {
		t.Fatalf("second retry cycle errors = %v", stats.Errors)
	}
	if len(pub.events) != 2*authorizationNotificationAttemptLimit {
		t.Fatalf("second cycle did not advance past first attempt cap: total publications=%d", len(pub.events))
	}
	for i := authorizationNotificationAttemptLimit; i < 2*authorizationNotificationAttemptLimit; i++ {
		stored, err := repo.Get(context.Background(), ids[i])
		if err != nil || stored.NotificationPublishedAt == nil {
			t.Fatalf("second page row %s was not published: stored=%+v err=%v", ids[i], stored, err)
		}
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
	pub := &capturingAuthPublisher{err: errors.New(strings.Repeat("events unavailable ", 128))}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	g := &domain.AuthorizationGrant{ID: uuid.New(), RunID: uuid.New(), State: domain.AuthorizationPending, Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 57600, HarnessKind: "codex", CreatedAt: now, RequestExpiresAt: now.Add(24 * time.Hour)}
	if err := repo.Create(context.Background(), g); err != nil {
		t.Fatal(err)
	}
	o.NotifyAuthorizationRequested(context.Background(), g)
	failed, err := repo.Get(context.Background(), g.ID)
	if err != nil || failed.NotificationSnapshot == nil || failed.NotificationPublishedAt != nil || len(failed.NotificationUnreached) > authorizationNotificationFailureLimit {
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

func TestNotifyAuthorizationRequestedDoesNotPublishExpiredGrant(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	repo := database.NewAuthorizationRepository(db)
	now := time.Now().UTC()
	pub := &capturingAuthPublisher{}
	o := &Orchestrator{authorizations: repo, approvalPublisher: pub, clock: func() time.Time { return now }}
	g := &domain.AuthorizationGrant{
		ID: uuid.New(), State: domain.AuthorizationPending, Policy: "swarm-bookkeeping",
		Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
		CreatedAt: now.Add(-2 * time.Hour), RequestExpiresAt: now.Add(-time.Second),
	}
	if err := repo.Create(context.Background(), g); err != nil {
		t.Fatal(err)
	}
	o.NotifyAuthorizationRequested(context.Background(), g)
	stored, err := repo.Get(context.Background(), g.ID)
	if err != nil || stored == nil || stored.NotificationSnapshot != nil || stored.NotificationPublishedAt != nil || len(pub.events) != 0 {
		t.Fatalf("expired pending row must not create or publish an FYI: stored=%+v events=%d err=%v", stored, len(pub.events), err)
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
