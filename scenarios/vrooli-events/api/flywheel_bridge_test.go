package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/bridge"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/testutil"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fixtureWakeGate struct{}

func (fixtureWakeGate) WithinWake(ctx context.Context, fn func(context.Context, bridge.Policy) error) error {
	return fn(ctx, bridge.Policy{Enabled: true, SourceInstance: "fixture-source", Scope: bridge.Scope{Epoch: "epoch-1", Digest: strings.Repeat("c", 64), Checkpoint: "known"}})
}

type noWakeSigner struct{}

func (noWakeSigner) SignWake(context.Context, []byte) (string, []byte, error) {
	return "", nil, errors.New("network/signing forbidden in owner integration fixture")
}

type fixtureForwarder struct {
	projector  *bridge.Forwarder
	fail       bool
	deliveries []subscription.QueuedDelivery
}

func (f *fixtureForwarder) Project(ctx context.Context, e store.Event) (string, bool, error) {
	return f.projector.Project(ctx, e)
}

func (f *fixtureForwarder) Deliver(_ context.Context, sub subscription.Subscription, q subscription.QueuedDelivery) error {
	if !bridge.ValidSubscription(sub) || !bridge.IsWakeBody(q.PayloadJSON) {
		return errors.New("invalid fixture delivery")
	}
	f.deliveries = append(f.deliveries, q)
	if f.fail {
		return errors.New("fixture lost response")
	}
	return nil
}

func nativeBridgeServer(t *testing.T) (*Server, *fixtureForwarder, int64, store.Event) {
	t.Helper()
	db := testutil.NewTestStore(t)
	subs, err := subscription.NewSQLiteStore(db.DB())
	if err != nil {
		t.Fatal(err)
	}
	projector, err := bridge.NewForwarder("https://fixture.example.com", noWakeSigner{}, fixtureWakeGate{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fixtureForwarder{projector: projector}
	s := &Server{store: db, db: db.DB(), subStore: subs, flywheel: f}
	spec := bridge.SubscriptionSpec()
	spec.Enabled = true
	id, err := subs.Create(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	facts, _ := structpb.NewStruct(map[string]any{"category": "run-completed", "correlation": map[string]any{"run_id": "11111111-1111-4111-8111-111111111111"}, "title": "PRIVATE TITLE", "body": "/private/source"})
	packed, _ := anypb.New(facts)
	e, err := convert.EnvelopeToEvent(&domain.EventEnvelope{EventId: strings.Repeat("a", 64), EventType: bridge.RunEventType, Source: &domain.EventSource{Scenario: "agent-manager"}, OccurredAt: timestamppb.New(time.Now().UTC()), Data: packed})
	if err != nil {
		t.Fatal(err)
	}
	return s, f, id, e
}

func TestFlywheelAtomicAdmissionRollbackDuplicateAndRestart(t *testing.T) {
	s, f, id, e := nativeBridgeServer(t)
	ctx := context.Background()
	queue := s.subStore.(subscription.QueueStore)
	trigger := fmt.Sprintf("CREATE TRIGGER reject_bridge BEFORE INSERT ON subscription_deliveries WHEN NEW.subscription_id=%d BEGIN SELECT RAISE(ABORT,'fixture failure'); END", id)
	if _, err := s.db.ExecContext(ctx, trigger); err != nil {
		t.Fatal(err)
	}
	if _, err := s.admitEvent(ctx, e); err == nil {
		t.Fatal("admitted event despite queue failure")
	}
	rows, err := s.store.Query(ctx, store.QueryFilters{EventID: e.EventID})
	if err != nil || len(rows) != 0 {
		t.Fatal("canonical event escaped rollback")
	}
	due, err := queue.Due(ctx, 10)
	if err != nil || len(due) != 0 {
		t.Fatal("queue escaped rollback")
	}
	if _, err := s.db.ExecContext(ctx, "DROP TRIGGER reject_bridge"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.admitEvent(ctx, e); err != nil {
		t.Fatal(err)
	}
	due, err = queue.Due(ctx, 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("due=%+v err=%v", due, err)
	}
	original := due[0].PayloadJSON
	if !bridge.IsWakeBody(original) || strings.Contains(original, "PRIVATE") || strings.Contains(original, "/private") {
		t.Fatal("raw payload entered bridge queue")
	}
	if _, err := s.admitEvent(ctx, e); !errors.Is(err, store.ErrDuplicateEvent) {
		t.Fatal("duplicate lost stable event identity")
	}
	due, _ = queue.Due(ctx, 10)
	if len(due) != 1 || due[0].PayloadJSON != original {
		t.Fatal("duplicate reprojected or duplicated queue row")
	}
	// Reconstruct the server over the same durable owner stores; no second
	// outbox or projection process participates in recovery.
	resumed := &Server{store: s.store, db: s.db, subStore: s.subStore, flywheel: f}
	resumed.drainSubscriptionQueue(ctx, queue)
	state, err := queue.GetDelivery(ctx, id, e.EventID)
	if err != nil || state.Status != "delivered" || len(f.deliveries) != 1 || f.deliveries[0].PayloadJSON != original {
		t.Fatalf("restart delivery %+v %v", state, err)
	}
}

func TestFlywheelQueueLostResponseRetainsExactProjection(t *testing.T) {
	s, f, id, e := nativeBridgeServer(t)
	ctx := context.Background()
	queue := s.subStore.(subscription.QueueStore)
	if _, err := s.admitEvent(ctx, e); err != nil {
		t.Fatal(err)
	}
	f.fail = true
	s.drainSubscriptionQueue(ctx, queue)
	state, err := queue.GetDelivery(ctx, id, e.EventID)
	if err != nil || state.Status != "pending" || state.Attempts != 1 {
		t.Fatalf("retry state %+v %v", state, err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE subscription_deliveries SET next_attempt_at='2000-01-01' WHERE subscription_id=?", id); err != nil {
		t.Fatal(err)
	}
	f.fail = false
	s.drainSubscriptionQueue(ctx, queue)
	if len(f.deliveries) != 2 || f.deliveries[0].PayloadJSON != f.deliveries[1].PayloadJSON {
		t.Fatal("retry changed stored bytes")
	}
}

func TestFlywheelDisabledAndMutatedSubscriptionNeverUseGenericTransport(t *testing.T) {
	s, f, _, e := nativeBridgeServer(t)
	ctx := context.Background()
	queue := s.subStore.(subscription.QueueStore)
	s.flywheel = nil
	if _, err := s.admitEvent(ctx, e); err != nil {
		t.Fatal(err)
	}
	due, _ := queue.Due(ctx, 10)
	if len(due) != 0 {
		t.Fatal("default-disabled enqueued a bridge wake")
	}
	// Use a fresh valid canonical event rather than rewriting an index.
	s2, f2, id2, e2 := nativeBridgeServer(t)
	queue2 := s2.subStore.(subscription.QueueStore)
	if _, err := s2.admitEvent(ctx, e2); err != nil {
		t.Fatal(err)
	}
	s2.flywheel = nil
	s2.drainSubscriptionQueue(ctx, queue2)
	state, _ := queue2.GetDelivery(ctx, id2, e2.EventID)
	if state.Status != "pending" || state.Attempts != 0 || len(f2.deliveries) != 0 {
		t.Fatal("disabled dispatcher consumed a row")
	}
	sub, _ := s2.subStore.Get(ctx, id2)
	sub.Name = "changed"
	sub.DeliveryTarget = "https://attacker.example/collect"
	s2.subStore.Update(ctx, sub)
	s2.flywheel = f2
	s2.drainSubscriptionQueue(ctx, queue2)
	state, _ = queue2.GetDelivery(ctx, id2, e2.EventID)
	if state.Status != "pending" || state.Attempts != 1 || len(f2.deliveries) != 0 {
		t.Fatal("mutated subscription escaped reserved path")
	}
	if len(f.deliveries) != 0 {
		t.Fatal("unexpected sends")
	}
}

func TestFlywheelDirectDeliverAndReconcileCannotBypassProjection(t *testing.T) {
	s, _, id, _ := nativeBridgeServer(t)
	for _, route := range []string{"deliver", "reconcile"} {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"event_id":"forged","payload":{"title":"private"}}`))
		request.SetPathValue("id", fmt.Sprint(id))
		recorder := httptest.NewRecorder()
		if route == "deliver" {
			s.handleDeliverSubscription(recorder, request)
		} else {
			s.handleReconcileSubscriptionDelivery(recorder, request)
		}
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d", route, recorder.Code)
		}
	}
}

// A disabled bridge may have a backlog after recovery or revocation. Its rows
// must remain inspectable without blocking unrelated owner deliveries.
func TestFlywheelDisabledBacklogDoesNotStarveOrdinaryQueue(t *testing.T) {
	s, _, id, e := nativeBridgeServer(t)
	ctx := context.Background()
	queue := s.subStore.(subscription.QueueStore)
	if _, err := s.admitEvent(ctx, e); err != nil {
		t.Fatal(err)
	}
	due, _ := queue.Due(ctx, 10)
	for i := 1; i < 30; i++ {
		if err := queue.Enqueue(ctx, id, fmt.Sprintf("held-%d", i), due[0].PayloadJSON); err != nil {
			t.Fatal(err)
		}
	}
	ordinaryID, err := s.subStore.Create(ctx, subscription.Subscription{Name: "ordinary", OwnerScenario: "fixture", EventPattern: "ordinary.*", DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "https://unused.example", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	// Malformed payload exercises ordinary dispatch without any transport.
	if err := queue.Enqueue(ctx, ordinaryID, "ordinary-event", "invalid-json"); err != nil {
		t.Fatal(err)
	}
	s.flywheel = nil
	s.drainSubscriptionQueue(ctx, queue)
	ordinary, err := queue.GetDelivery(ctx, ordinaryID, "ordinary-event")
	if err != nil || ordinary.Status != "dead_letter" || ordinary.Attempts != 1 {
		t.Fatalf("ordinary queue starved: %+v %v", ordinary, err)
	}
	held, _ := queue.GetDelivery(ctx, id, e.EventID)
	if held.Status != "pending" || held.Attempts != 0 {
		t.Fatal("disabled row was consumed")
	}
	// Renaming a subscription does not make its old v2 body eligible either.
	sub, _ := s.subStore.Get(ctx, id)
	sub.Name = "renamed"
	sub.DeliveryTarget = "https://unused.example"
	s.subStore.Update(ctx, sub)
	selected, err := s.subStore.(*subscription.SQLiteStore).DueExcluding(ctx, 25, bridge.ReservedTarget, bridge.SubscriptionSpec().Name, bridge.WakeSchema)
	if err != nil || len(selected) != 0 {
		t.Fatalf("renamed queued bodies escaped exclusion: %+v %v", selected, err)
	}
}
