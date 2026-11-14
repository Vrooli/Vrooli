package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/testutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestIngestRollsBackEventAndFanoutTogetherAndKeepsLostResponseReplay(t *testing.T) {
	srv, ts := newTestServer(t)
	firstID, err := srv.subStore.Create(context.Background(), subscription.Subscription{
		Name: "atomic-first", OwnerScenario: "notification-hub", EventPattern: "example.atomic.*",
		DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "http://127.0.0.1/unused", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := srv.subStore.Create(context.Background(), subscription.Subscription{
		Name: "atomic-second", OwnerScenario: "notification-hub", EventPattern: "example.atomic.*",
		DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "http://127.0.0.1/unused", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, sub := range []subscription.Subscription{
		{
			Name: "atomic-source-mismatch", OwnerScenario: "notification-hub", EventPattern: "example.atomic.*",
			SourceFilter: "different-source", DeliveryType: subscription.DeliveryWebhook,
			DeliveryTarget: "http://127.0.0.1/unused", Enabled: true,
		},
		{
			Name: "atomic-disabled", OwnerScenario: "notification-hub", EventPattern: "example.atomic.*",
			DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "http://127.0.0.1/unused", Enabled: false,
		},
		{
			Name: "atomic-sse", OwnerScenario: "notification-hub", EventPattern: "example.atomic.*",
			DeliveryType: subscription.DeliverySSE, Enabled: true,
		},
	} {
		if _, err := srv.subStore.Create(context.Background(), sub); err != nil {
			t.Fatal(err)
		}
	}

	data, err := structpb.NewStruct(map[string]any{"kind": "atomic-fanout"})
	if err != nil {
		t.Fatal(err)
	}
	packed, err := anypb.New(data)
	if err != nil {
		t.Fatal(err)
	}
	env := &domain.EventEnvelope{
		EventId: "evt-atomic-admission", EventType: "example.atomic.v1", OccurredAt: timestamppb.New(time.Now().UTC()),
		Source: &domain.EventSource{Scenario: "atomic-test"}, Data: packed,
	}
	body, err := protojson.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}

	ch, _, cleanup := srv.broker.Subscribe(context.Background(), broker.SubscribeOpts{EventTypePattern: "example.atomic.*"})
	defer cleanup()

	// Fail on the second matching delivery after the first insert has succeeded
	// inside the transaction. This proves partial fan-out and event admission
	// are rolled back as one unit.
	trigger := fmt.Sprintf(`CREATE TRIGGER reject_second_atomic_delivery BEFORE INSERT ON subscription_deliveries WHEN NEW.subscription_id = %d BEGIN SELECT RAISE(ABORT, 'injected queue failure'); END`, secondID)
	if _, err := srv.db.ExecContext(context.Background(), trigger); err != nil {
		t.Fatalf("install failure trigger: %v", err)
	}
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("failed fan-out status=%d want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	events, err := srv.store.Query(context.Background(), store.QueryFilters{EventID: env.EventId})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("failed admission left event rows: %#v", events)
	}
	stats, err := srv.store.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalPayloadBytes != 0 {
		t.Fatalf("failed admission changed payload accounting: %#v", stats)
	}
	queue := srv.subStore.(subscription.QueueStore)
	due, err := queue.Due(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("failed admission left partial delivery rows: %#v", due)
	}
	select {
	case msg := <-ch:
		t.Fatalf("failed admission was broadcast before commit: %#v", msg)
	default:
	}

	if _, err := srv.db.ExecContext(context.Background(), `DROP TRIGGER reject_second_atomic_delivery`); err != nil {
		t.Fatalf("remove failure trigger: %v", err)
	}
	resp, err = http.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("retry status=%d want %d", resp.StatusCode, http.StatusAccepted)
	}
	due, err = queue.Due(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 2 || due[0].SubscriptionID != firstID || due[1].SubscriptionID != secondID {
		t.Fatalf("successful admission did not queue both matching deliveries: %#v", due)
	}
	select {
	case msg := <-ch:
		if msg.Event != env.EventType {
			t.Fatalf("post-commit broadcast event=%q want %q", msg.Event, env.EventType)
		}
	case <-time.After(time.Second):
		t.Fatal("successful admission was not broadcast after commit")
	}

	// Treat the successful first response as lost. The stable event identity
	// preserves the public duplicate conflict while already committed delivery
	// rows remain unique and the replay does not broadcast again.
	resp, err = http.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("same-ID replay status=%d want %d", resp.StatusCode, http.StatusConflict)
	}
	due, err = queue.Due(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 2 {
		t.Fatalf("same-ID replay changed delivery rows: %#v", due)
	}
	select {
	case msg := <-ch:
		t.Fatalf("same-ID conflict replay broadcast duplicate: %#v", msg)
	default:
	}
}

func TestIngestConfiguredDatabaseFailsClosedWithoutAtomicRepositories(t *testing.T) {
	server, _ := newTestServer(t)
	eventStore := &testutil.MockStore{}
	server.store = eventStore
	_, err := server.admitEvent(context.Background(), store.Event{EventID: "must-not-write"})
	if err == nil {
		t.Fatal("configured database without transaction-aware repositories must fail closed")
	}
	if got := len(eventStore.InsertCalls); got != 0 {
		t.Fatalf("atomic capability failure wrote through legacy Insert %d times", got)
	}
}

func TestIngestConfiguredDatabaseFailsClosedWithoutAtomicQueueStore(t *testing.T) {
	server, _ := newTestServer(t)
	eventStore := &insertCountingSQLiteStore{SQLiteStore: server.store.(*store.SQLiteStore)}
	server.store = eventStore
	server.subStore = legacyQueueStore{QueueStore: server.subStore.(subscription.QueueStore)}
	if _, ok := server.subStore.(subscription.AtomicQueueStore); ok {
		t.Fatal("legacy queue fixture unexpectedly supports transaction-aware admission")
	}

	_, err := server.admitEvent(context.Background(), store.Event{EventID: "must-not-write"})
	if err == nil {
		t.Fatal("configured database without AtomicQueueStore must fail closed")
	}
	if eventStore.legacyInsertCalls != 0 || eventStore.insertTxCalls != 0 {
		t.Fatalf("missing queue capability reached event writes: legacy=%d tx=%d", eventStore.legacyInsertCalls, eventStore.insertTxCalls)
	}
	events, err := eventStore.Query(context.Background(), store.QueryFilters{EventID: "must-not-write"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("missing queue capability left event rows: %#v", events)
	}
}

func TestAdmissionRollbackErrorPreservesPrimaryAndIgnoresCompletedTransaction(t *testing.T) {
	primary := errors.New("fan-out failed")
	rollback := errors.New("rollback failed")
	joined := joinAdmissionRollbackError(primary, rollback)
	if !errors.Is(joined, primary) || !errors.Is(joined, rollback) {
		t.Fatalf("joined error does not preserve both causes: %v", joined)
	}
	if got := joinAdmissionRollbackError(primary, sql.ErrTxDone); got != primary {
		t.Fatalf("completed-transaction rollback changed primary error: %v", got)
	}
	if got := joinAdmissionRollbackError(nil, sql.ErrTxDone); got != nil {
		t.Fatalf("completed-transaction rollback after commit reported error: %v", got)
	}
	if got := joinAdmissionRollbackError(nil, rollback); !errors.Is(got, rollback) {
		t.Fatalf("standalone rollback failure was hidden: %v", got)
	}
}

type legacyQueueStore struct {
	subscription.QueueStore
}

type insertCountingSQLiteStore struct {
	*store.SQLiteStore
	legacyInsertCalls int
	insertTxCalls     int
}

func (s *insertCountingSQLiteStore) Insert(ctx context.Context, event store.Event) (int64, error) {
	s.legacyInsertCalls++
	return s.SQLiteStore.Insert(ctx, event)
}

func (s *insertCountingSQLiteStore) InsertTx(ctx context.Context, tx *sql.Tx, event store.Event) (int64, error) {
	s.insertTxCalls++
	return s.SQLiteStore.InsertTx(ctx, tx, event)
}
