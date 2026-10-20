package releasedecision

import (
	"context"
	"testing"
	"time"

	"deployment-manager/internal/testutil"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

func newStore(t *testing.T) *SQLStore {
	t.Helper()
	db := testutil.OpenSQLite(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return NewSQLStore(db)
}

func TestStoreRoundTripAndOutstandingCount(t *testing.T) {
	store := newStore(t)
	now := time.Now().UTC()
	ask := &operatordecision.PendingAsk{
		ID: "release-abc", Subject: "rr-1", State: operatordecision.PendingStatePending,
		RaisedAt: now, ExpiresAt: now.Add(time.Hour),
		EventID: "event-generation-1", Published: true,
		Extra: map[string]string{"review_key": "rr-1", "requesting_agent": "run-7"},
	}
	if err := store.Put(context.Background(), ask); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(context.Background(), "release-abc")
	if err != nil || got == nil {
		t.Fatalf("get: %v (nil=%v)", err, got == nil)
	}
	if got.Subject != "rr-1" || got.State != operatordecision.PendingStatePending || got.Extra["requesting_agent"] != "run-7" || got.EventID != ask.EventID || !got.Published {
		t.Fatalf("round trip lost fields: %+v", got)
	}
	if !got.Outstanding(now) {
		t.Fatal("pending unexpired ask should be outstanding")
	}

	count, err := store.CountOutstandingForSubject(context.Background(), "rr-1", now)
	if err != nil || count != 1 {
		t.Fatalf("outstanding count=%d err=%v", count, err)
	}

	// Resolve it: no longer outstanding, count drops to zero.
	resolvedAt := now.Add(time.Minute)
	got.State = operatordecision.PendingStateApproved
	got.ResolvedAt = &resolvedAt
	got.ResolvedBy = "operator-human"
	if err := store.Put(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	count, _ = store.CountOutstandingForSubject(context.Background(), "rr-1", now)
	if count != 0 {
		t.Fatalf("resolved ask must not count as outstanding, got %d", count)
	}
	reloaded, _ := store.Get(context.Background(), "release-abc")
	if reloaded.ResolvedBy != "operator-human" || reloaded.ResolvedAt == nil {
		t.Fatalf("resolution not persisted: %+v", reloaded)
	}
}

func TestLegacySchemaAddsPublicationFieldsAndPreservesRows(t *testing.T) {
	db := testutil.OpenSQLite(t)
	legacy := `CREATE TABLE release_decision_asks (
		id TEXT PRIMARY KEY, subject TEXT NOT NULL, state TEXT NOT NULL,
		raised_at TIMESTAMP NOT NULL, expires_at TIMESTAMP NOT NULL,
		resolved_at TIMESTAMP, resolved_by TEXT, extra TEXT NOT NULL DEFAULT '{}'
	);
	INSERT INTO release_decision_asks (id,subject,state,raised_at,expires_at,extra)
	VALUES ('legacy','rr-legacy','pending','2026-10-16T12:00:00Z','2026-10-17T12:00:00Z','{}');`
	if _, err := db.Exec(legacy); err != nil {
		t.Fatal(err)
	}
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("reconcile legacy schema: %v", err)
	}
	store := NewSQLStore(db)
	got, err := store.Get(context.Background(), "legacy")
	if err != nil || got == nil || got.EventID != "" || got.Published {
		t.Fatalf("legacy row should retain its old unpublished meaning: %+v err=%v", got, err)
	}
	got.EventID = "new-generation"
	got.Published = true
	if err := store.Put(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Get(context.Background(), "legacy")
	if err != nil || reloaded == nil || reloaded.EventID != got.EventID || !reloaded.Published {
		t.Fatalf("reconciled row did not persist publication fields: %+v err=%v", reloaded, err)
	}
}

type raiserTestClock struct{ now time.Time }

func (c raiserTestClock) Now() time.Time { return c.now }

type retryPublisher struct {
	err      error
	attempts []eventbus.DomainEvent
}

func (p *retryPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.attempts = append(p.attempts, event)
	return p.err
}

func TestRaiserWithSQLStoreRetriesAndSuppressesPublishedEvent(t *testing.T) {
	store := newStore(t)
	pub := &retryPublisher{err: context.DeadlineExceeded}
	clock := raiserTestClock{now: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	raiser := operatordecision.Raiser{Source: "deployment-manager", Kind: "release", Store: store, Publisher: pub, Clock: clock}
	ask := operatordecision.PendingAsk{ID: "release-retry", Subject: "agent", Extra: map[string]string{"review_key": "review-1"}}
	facts := operatordecision.Facts{Question: "Approve release?", Recommended: "reject", Options: []operatordecision.Option{{Key: "approve"}, {Key: "reject"}}}
	if raised, err := raiser.Raise(context.Background(), ask, facts); !raised || err == nil {
		t.Fatalf("failed publication should leave a pending ask: raised=%v err=%v", raised, err)
	}
	failed, err := store.Get(context.Background(), ask.ID)
	if err != nil || failed == nil || failed.EventID == "" || failed.Published {
		t.Fatalf("failed publication receipt state: %+v err=%v", failed, err)
	}
	firstID := operatordecision.EventID(pub.attempts[0])
	pub.err = nil
	clock.now = clock.now.Add(time.Minute)
	if raised, err := raiser.Raise(context.Background(), ask, facts); !raised || err != nil {
		t.Fatalf("retry: raised=%v err=%v", raised, err)
	}
	if len(pub.attempts) != 2 || operatordecision.EventID(pub.attempts[1]) != firstID {
		t.Fatalf("retry changed event identity or count: attempts=%d", len(pub.attempts))
	}
	retried, err := store.Get(context.Background(), ask.ID)
	if err != nil || retried == nil || !retried.Published || retried.EventID != firstID {
		t.Fatalf("successful publication receipt was not durable: %+v err=%v", retried, err)
	}
	if _, err := raiser.Raise(context.Background(), ask, facts); err != nil || len(pub.attempts) != 2 {
		t.Fatalf("successful receipt must suppress later publish: attempts=%d err=%v", len(pub.attempts), err)
	}
}

func TestStoreGetMissingReturnsNil(t *testing.T) {
	store := newStore(t)
	got, err := store.Get(context.Background(), "nope")
	if err != nil || got != nil {
		t.Fatalf("missing ask should be (nil,nil), got %+v err=%v", got, err)
	}
}

func TestStoreExpiredAskNotOutstanding(t *testing.T) {
	store := newStore(t)
	now := time.Now().UTC()
	if err := store.Put(context.Background(), &operatordecision.PendingAsk{
		ID: "release-old", Subject: "rr-2", State: operatordecision.PendingStatePending,
		RaisedAt: now.Add(-48 * time.Hour), ExpiresAt: now.Add(-24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	count, err := store.CountOutstandingForSubject(context.Background(), "rr-2", now)
	if err != nil || count != 0 {
		t.Fatalf("expired ask must not count, got %d err=%v", count, err)
	}
}
