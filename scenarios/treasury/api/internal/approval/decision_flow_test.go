package approval

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"treasury/internal/authorization"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
)

type decisionPublisher struct {
	events []eventbus.DomainEvent
	err    error
}

func (p *decisionPublisher) PublishDomainEvent(_ context.Context, value eventbus.DomainEvent) error {
	p.events = append(p.events, value)
	return p.err
}

type decisionController struct {
	value      authorization.Record
	approved   int
	declined   int
	getErr     error
	approveErr error
	releaseErr error
}

func (c *decisionController) Get(context.Context, string) (authorization.Record, error) {
	return c.value, c.getErr
}

func (c *decisionController) Approve(context.Context, string) (authorization.Record, error) {
	c.approved++
	return c.value, c.approveErr
}

func (c *decisionController) Release(context.Context, string) (authorization.Record, error) {
	c.declined++
	return c.value, c.releaseErr
}

func TestDecisionRaiseRetriesWithStableHubEventAndKeepsLocalQueue(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(ctx, `CREATE TABLE authorizations(id TEXT PRIMARY KEY)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE authorization_book_bindings(authorization_id TEXT PRIMARY KEY, book_id TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO authorizations(id) VALUES('auth-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO authorization_book_bindings(authorization_id,book_id) VALUES('auth-1','book-1')`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	publisher := &decisionPublisher{err: errors.New("event bus unavailable")}
	flow := NewDecisionFlow(NewSQLiteDecisionAskStore(db), publisher, clock)
	controller := &decisionController{value: authorization.Record{ID: "auth-1", BookID: "book-1"}}
	service := NewService(NewSQLiteRepository(db), controller, nil, clock).WithDecisionAsker(flow)
	flow.WithApprovalService(service)
	request := authorization.ApprovalAdmission{ID: "approval-1", AuthorizationID: "auth-1", BookID: "book-1", MandateID: "mandate-1", RequestingAgent: "agent:run-1", AmountMinor: 125, Currency: "USD", Counterparty: "api.example", ExpiresAt: now.Add(time.Hour)}
	if err := service.Admit(ctx, request); err != nil {
		t.Fatalf("admit despite unavailable decision publisher: %v", err)
	}
	queue, err := service.List(ctx, StatusQueued)
	if err != nil || len(queue) != 1 {
		t.Fatalf("local approval queue was not durable: rows=%d err=%v", len(queue), err)
	}
	attempts, err := service.RelayAttempts(ctx, request.ID)
	if err != nil || len(attempts) != 1 || attempts[0].Outcome != "failed" {
		t.Fatalf("decision publication failure was not recorded: attempts=%+v err=%v", attempts, err)
	}
	publisher.err = nil
	if err := flow.RetryQueued(ctx, service); err != nil {
		t.Fatalf("retry durable approval decision: %v", err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("expected failed publish and one retry, got %d events", len(publisher.events))
	}
	if got, want := operatordecision.EventID(publisher.events[0]), operatordecision.EventID(publisher.events[1]); got != want {
		t.Fatalf("retry changed Hub event identity: %s != %s", got, want)
	}
	if publisher.events[1].EventType != "treasury.approval.decision_requested.v1" {
		t.Fatalf("unexpected decision event type %q", publisher.events[1].EventType)
	}
	correlation, ok := publisher.events[1].Payload["correlation"].(map[string]any)
	if !ok || correlation["approval_id"] != request.ID || correlation["authorization_id"] != request.AuthorizationID || correlation["amount_minor"] != "125" {
		t.Fatalf("decision did not carry exact approval correlation: %#v", publisher.events[1].Payload["correlation"])
	}
	var options []map[string]any
	raw, err := json.Marshal(publisher.events[1].Payload["options"])
	if err != nil || json.Unmarshal(raw, &options) != nil || len(options) != 2 || options[0]["key"] != "approve" || options[1]["key"] != "decline" {
		t.Fatalf("decision must offer only explicit approve/decline: %s err=%v", raw, err)
	}
}

func TestDecisionEffectRequiresExactVerifiedHumanAnswerAndAppliesOnce(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(ctx, `CREATE TABLE authorizations(id TEXT PRIMARY KEY)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE authorization_book_bindings(authorization_id TEXT PRIMARY KEY, book_id TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO authorizations(id) VALUES('auth-2')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO authorization_book_bindings(authorization_id,book_id) VALUES('auth-2','book-2')`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	store := NewSQLiteDecisionAskStore(db)
	publisher := &decisionPublisher{}
	flow := NewDecisionFlow(store, publisher, clock)
	controller := &decisionController{value: authorization.Record{ID: "auth-2", BookID: "book-2"}}
	service := NewService(NewSQLiteRepository(db), controller, nil, clock).WithDecisionAsker(flow)
	flow.WithApprovalService(service)
	request := authorization.ApprovalAdmission{ID: "approval-2", AuthorizationID: "auth-2", BookID: "book-2", MandateID: "mandate-2", RequestingAgent: "agent:run-2", AmountMinor: 225, Currency: "USD", Counterparty: "vendor.example", ExpiresAt: now.Add(time.Hour)}
	if err := service.Admit(ctx, request); err != nil {
		t.Fatal(err)
	}
	raisedAsk, err := store.Get(ctx, decisionAskID(request.ID))
	if err != nil || raisedAsk == nil || raisedAsk.EventID == "" {
		t.Fatalf("raised decision event identity missing: ask=%+v err=%v", raisedAsk, err)
	}
	resolution := operatordecision.DecisionResolution{
		AskID: "notification-hub-uuid", RequesterRef: raisedAsk.EventID, State: operatordecision.StateAnswered, Answer: "approve",
		AnsweredBy: "operator:owner", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "operator-session",
		ResolvedAt: now.Add(time.Minute).Format(time.RFC3339Nano), Correlation: map[string]string{
			"approval_id": request.ID, "authorization_id": request.AuthorizationID, "book_id": request.BookID,
			"mandate_id": request.MandateID, "requesting_agent": request.RequestingAgent, "amount_minor": "225",
			"currency": request.Currency, "counterparty": request.Counterparty,
		},
	}
	wrong := resolution
	wrong.Correlation = map[string]string{}
	if _, err := flow.Apply(ctx, wrong); err != nil || controller.approved != 0 {
		t.Fatalf("mismatched decision changed the hold: calls=%d err=%v", controller.approved, err)
	}
	foreignAsk := resolution
	foreignAsk.RequesterRef = "another-source-event"
	if _, err := flow.Apply(ctx, foreignAsk); err != nil || controller.approved != 0 {
		t.Fatalf("foreign Hub ask event changed the hold: calls=%d err=%v", controller.approved, err)
	}
	nonHuman := resolution
	nonHuman.AnsweredByIdentityKind = "agent"
	if _, err := flow.Apply(ctx, nonHuman); err != nil || controller.approved != 0 {
		t.Fatalf("non-human decision changed the hold: calls=%d err=%v", controller.approved, err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handled, applyErr := flow.Apply(ctx, resolution)
			if applyErr != nil || !handled {
				results <- fmt.Errorf("concurrent verified human decision: handled=%v err=%w", handled, applyErr)
			}
		}()
	}
	wg.Wait()
	close(results)
	for applyErr := range results {
		t.Fatal(applyErr)
	}
	if controller.approved != 1 {
		t.Fatalf("concurrent Hub deliveries reapplied approval %d times", controller.approved)
	}
	if handled, err := flow.Apply(ctx, resolution); err != nil || !handled || controller.approved != 1 {
		t.Fatalf("duplicate Hub delivery reapplied approval: handled=%v calls=%d err=%v", handled, controller.approved, err)
	}
	resolved, err := service.Get(ctx, request.ID)
	if err != nil || resolved.Status != StatusApproved || resolved.ResolverIdentity != "operator:owner" {
		t.Fatalf("approved local queue state missing: request=%+v err=%v", resolved, err)
	}
}

type failResolvedDecisionStore struct {
	DecisionAskStore
	failOnce bool
}

func (s *failResolvedDecisionStore) Put(ctx context.Context, value *operatordecision.PendingAsk) error {
	if s.failOnce && value != nil && value.State != operatordecision.PendingStatePending {
		s.failOnce = false
		return errors.New("simulated crash before Hub readback persistence")
	}
	return s.DecisionAskStore.Put(ctx, value)
}

// [REQ:TRS-P0-006] A repeated Hub answer after a process restart repairs the
// decision readback without applying the local approval effect a second time.
func TestDecisionReplayRepairsHubReadbackAfterApprovalCommit(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "approval-restart.sqlite")
	db := openDecisionFlowDB(t, databasePath)
	for _, statement := range []string{
		`CREATE TABLE authorizations(id TEXT PRIMARY KEY)`,
		`CREATE TABLE authorization_book_bindings(authorization_id TEXT PRIMARY KEY, book_id TEXT NOT NULL)`,
		`INSERT INTO authorizations(id) VALUES('auth-restart')`,
		`INSERT INTO authorization_book_bindings(authorization_id,book_id) VALUES('auth-restart','book-restart')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	innerStore := NewSQLiteDecisionAskStore(db)
	store := &failResolvedDecisionStore{DecisionAskStore: innerStore, failOnce: true}
	controller := &decisionController{value: authorization.Record{ID: "auth-restart", BookID: "book-restart"}}
	flow := NewDecisionFlow(store, &decisionPublisher{}, clock)
	service := NewService(NewSQLiteRepository(db), controller, nil, clock).WithDecisionAsker(flow)
	flow.WithApprovalService(service)
	admission := authorization.ApprovalAdmission{
		ID: "approval-restart", AuthorizationID: "auth-restart", BookID: "book-restart", MandateID: "mandate-restart",
		RequestingAgent: "agent:run-restart", AmountMinor: 375, Currency: "USD", Counterparty: "api.example", ExpiresAt: now.Add(time.Hour),
	}
	if err := service.Admit(ctx, admission); err != nil {
		t.Fatal(err)
	}
	raisedAsk, err := store.Get(ctx, decisionAskID(admission.ID))
	if err != nil || raisedAsk == nil || raisedAsk.EventID == "" {
		t.Fatalf("raised decision event identity missing: ask=%+v err=%v", raisedAsk, err)
	}
	resolution := operatordecision.DecisionResolution{
		AskID: "notification-hub-uuid", RequesterRef: raisedAsk.EventID, State: operatordecision.StateAnswered, Answer: "approve",
		AnsweredBy: "operator:owner", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "operator-session",
		ResolvedAt: now.Add(time.Minute).Format(time.RFC3339Nano), Correlation: map[string]string{
			"approval_id": admission.ID, "authorization_id": admission.AuthorizationID, "book_id": admission.BookID,
			"mandate_id": admission.MandateID, "requesting_agent": admission.RequestingAgent, "amount_minor": "375",
			"currency": admission.Currency, "counterparty": admission.Counterparty,
		},
	}
	if handled, err := flow.Apply(ctx, resolution); !handled || err == nil {
		t.Fatalf("expected local approval commit followed by a readback persistence failure: handled=%v err=%v", handled, err)
	}
	if controller.approved != 1 {
		t.Fatalf("approval effect calls after first delivery=%d, want 1", controller.approved)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = openDecisionFlowDB(t, databasePath)
	store = &failResolvedDecisionStore{DecisionAskStore: NewSQLiteDecisionAskStore(db)}
	flow = NewDecisionFlow(store, &decisionPublisher{}, clock)
	service = NewService(NewSQLiteRepository(db), controller, nil, clock).WithDecisionAsker(flow)
	flow.WithApprovalService(service)
	if handled, err := flow.Apply(ctx, resolution); !handled || err != nil {
		t.Fatalf("exact replay did not repair durable Hub readback: handled=%v err=%v", handled, err)
	}
	if controller.approved != 1 {
		t.Fatalf("exact replay repeated approval effect: calls=%d", controller.approved)
	}
	ask, err := store.Get(ctx, decisionAskID(admission.ID))
	if err != nil || ask == nil || ask.State != resolution.State || ask.ResolvedBy != resolution.AnsweredBy {
		t.Fatalf("Hub ask readback was not repaired: ask=%+v err=%v", ask, err)
	}
	if handled, err := flow.Apply(ctx, resolution); !handled || err != nil || controller.approved != 1 {
		t.Fatalf("subsequent duplicate was not idempotent: handled=%v calls=%d err=%v", handled, controller.approved, err)
	}
}

func openDecisionFlowDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
	if err != nil {
		t.Fatal(err)
	}
	value, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	value.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = value.Close() })
	return value
}

type unavailableAskReader struct{}

func (unavailableAskReader) ReadAsk(context.Context, string) (askreader.Ask, error) {
	return askreader.Ask{}, askreader.ErrNotIssued
}

func TestApprovalWebhookNeverFallsBackToUnverifiedEventBody(t *testing.T) {
	reader := requireIssuedAskReader{inner: unavailableAskReader{}}
	_, err := reader.ReadAsk(context.Background(), "ask-1")
	if !errors.Is(err, askreader.ErrHubUnavailable) || errors.Is(err, askreader.ErrNotIssued) {
		t.Fatalf("missing ask-reader credential must fail closed as unavailable: %v", err)
	}
}
