package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// [REQ:SUB-003] Ingest persists a matching webhook delivery before returning;
// the queue is idempotent per subscription/event and successful draining
// updates subscription health.
func TestIngest_FansOutDurablyWithSignatureAndHealth(t *testing.T) {
	type deliveryObservation struct {
		signature      string
		eventID        string
		idempotencyKey string
		body           string
	}
	observations := make(chan deliveryObservation, 1)
	var observationOnce sync.Once
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observationOnce.Do(func() {
			body, _ := io.ReadAll(r.Body)
			observations <- deliveryObservation{
				signature:      r.Header.Get("X-Vrooli-Events-Signature"),
				eventID:        r.Header.Get("X-Vrooli-Events-Event-ID"),
				idempotencyKey: r.Header.Get("X-Vrooli-Events-Idempotency-Key"),
				body:           string(body),
			}
		})
		w.WriteHeader(http.StatusAccepted)
	}))
	defer webhook.Close()

	srv, ts := newTestServer(t)
	srv.webhookDeliverer = subscription.NewWebhookDelivererWithSecret("test-secret")
	id, err := srv.subStore.Create(context.Background(), subscription.Subscription{
		Name: "audit-webhook", OwnerScenario: "notification-hub", EventPattern: "example.audit.*",
		DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: webhook.URL, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.subStore.Create(context.Background(), subscription.Subscription{
		Name: "unmatched", OwnerScenario: "notification-hub", EventPattern: "other.event.*",
		DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: webhook.URL, Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}

	data, err := structpb.NewStruct(map[string]any{"kind": "fanout"})
	if err != nil {
		t.Fatal(err)
	}
	packed, err := anypb.New(data)
	if err != nil {
		t.Fatal(err)
	}
	env := &domain.EventEnvelope{
		EventId: "evt-fanout-1", EventType: "example.audit.v1", OccurredAt: timestamppb.New(time.Now().UTC()),
		Source: &domain.EventSource{Scenario: "notification-hub"}, Data: packed,
	}
	body, err := protojson.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("ingest status=%d", resp.StatusCode)
	}

	queue, ok := srv.subStore.(subscription.QueueStore)
	if !ok {
		t.Fatal("subscription store does not expose durable queue")
	}
	due, err := queue.Due(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].SubscriptionID != id || due[0].EventID != env.EventId {
		t.Fatalf("unexpected durable queue: %#v", due)
	}
	dispatchCtx, cancelDispatch := context.WithCancel(context.Background())
	defer cancelDispatch()
	go srv.runSubscriptionDispatcher(dispatchCtx)
	select {
	case observation := <-observations:
		if observation.eventID != env.EventId || observation.idempotencyKey != env.EventId || len(observation.signature) < len("sha256=")+64 {
			t.Fatalf("missing delivery headers: event=%q idempotency=%q signature=%q", observation.eventID, observation.idempotencyKey, observation.signature)
		}
		if !strings.Contains(observation.body, `"kind":"fanout"`) {
			t.Fatalf("webhook payload did not preserve structured event facts: %s", observation.body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runtime subscription dispatcher did not deliver the queued webhook")
	}
	var health subscription.Health
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		health, err = srv.subStore.GetHealth(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if health.TotalDelivered == 1 {
			break
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatalf("runtime dispatcher delivered webhook but did not update health: %#v", health)
		}
	}
	if health.TotalDelivered != 1 || health.ConsecutiveFailures != 0 {
		t.Fatalf("unexpected health: %#v", health)
	}

	// A second enqueue for the same event cannot duplicate delivery.
	event := srv.storeQueryEvent(t, env.EventId)
	if err := srv.enqueueSubscriptions(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	allDue, err := queue.Due(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(allDue) != 0 {
		t.Fatalf("duplicate event should not create a new pending delivery: %#v", allDue)
	}
}

func (s *Server) storeQueryEvent(t *testing.T, eventID string) (event store.Event) {
	t.Helper()
	events, err := s.store.Query(context.Background(), store.QueryFilters{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range events {
		if candidate.EventID == eventID {
			return candidate
		}
	}
	t.Fatalf("event %q not found", eventID)
	return store.Event{}
}

// [REQ:SUB-003] Failed webhook deliveries are retried and become dead-lettered
// after the bounded attempt budget is exhausted.
func TestSubscriptionDispatcher_RetriesAndDeadLetters(t *testing.T) {
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer webhook.Close()

	queue := &retryQueueStore{
		sub: subscription.Subscription{
			ID: 1, Name: "failing-webhook", OwnerScenario: "notification-hub",
			EventPattern: "example.audit.*", DeliveryType: subscription.DeliveryWebhook,
			DeliveryTarget: webhook.URL, Enabled: true,
		},
		delivery: subscription.QueuedDelivery{
			ID: 1, SubscriptionID: 1, EventID: "evt-retry-1",
			PayloadJSON: `{"event_id":"evt-retry-1","event_type":"example.audit.v1"}`,
		},
	}
	srv := &Server{
		subStore:         queue,
		webhookDeliverer: subscription.NewWebhookDeliverer(),
	}

	for attempt := 0; attempt < 5; attempt++ {
		srv.drainSubscriptionQueue(context.Background(), queue)
	}

	if len(queue.failed) != 5 {
		t.Fatalf("failure count = %d, want 5", len(queue.failed))
	}
	if !queue.failed[len(queue.failed)-1].deadLetter {
		t.Fatal("fifth failure should dead-letter the delivery")
	}
	if queue.delivery.Attempts != 5 {
		t.Fatalf("attempt count = %d, want 5", queue.delivery.Attempts)
	}
}

type retryFailure struct {
	deadLetter bool
}

// retryQueueStore keeps this test focused on dispatcher policy. Due returns
// the same durable row until MarkFailed marks it dead-lettered.
type retryQueueStore struct {
	sub      subscription.Subscription
	delivery subscription.QueuedDelivery
	failed   []retryFailure
}

func (q *retryQueueStore) Create(context.Context, subscription.Subscription) (int64, error) {
	return q.sub.ID, nil
}

func (q *retryQueueStore) Get(_ context.Context, id int64) (subscription.Subscription, error) {
	if id != q.sub.ID {
		return subscription.Subscription{}, context.Canceled
	}
	return q.sub, nil
}

func (q *retryQueueStore) List(context.Context, subscription.ListFilters) ([]subscription.Subscription, error) {
	return []subscription.Subscription{q.sub}, nil
}
func (q *retryQueueStore) Update(context.Context, subscription.Subscription) error { return nil }
func (q *retryQueueStore) Delete(context.Context, int64) error                     { return nil }
func (q *retryQueueStore) GetHealth(context.Context, int64) (subscription.Health, error) {
	return subscription.Health{SubscriptionID: q.sub.ID}, nil
}
func (q *retryQueueStore) Close() error                                                 { return nil }
func (q *retryQueueStore) Enqueue(_ context.Context, _ int64, _ string, _ string) error { return nil }
func (q *retryQueueStore) Due(context.Context, int) ([]subscription.QueuedDelivery, error) {
	if len(q.failed) > 0 && q.failed[len(q.failed)-1].deadLetter {
		return nil, nil
	}
	return []subscription.QueuedDelivery{q.delivery}, nil
}
func (q *retryQueueStore) MarkDelivered(context.Context, int64, int64, time.Time) error { return nil }
func (q *retryQueueStore) MarkFailed(_ context.Context, _, _ int64, _ string, _ time.Time, deadLetter bool) error {
	q.delivery.Attempts++
	q.failed = append(q.failed, retryFailure{deadLetter: deadLetter})
	return nil
}
