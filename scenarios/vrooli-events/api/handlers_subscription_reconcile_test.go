package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

func TestReconcileSubscriptionDeliveryUsesRetainedEventAndIsIdempotent(t *testing.T) {
	srv, server := newTestServer(t)
	ctx := context.Background()
	_, err := srv.store.Insert(ctx, store.Event{EventID: "missed-decision", EventType: "agent_manager.effort.decision_requested.v1", SourceScenario: "agent-manager", Payload: []byte(`{"marker":"original"}`), CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	id, err := srv.subStore.Create(ctx, subscription.Subscription{Name: "hub", OwnerScenario: "notification-hub", EventPattern: "**.decision_requested.v1", DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "http://127.0.0.1:1/events", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := server.URL + "/api/v1/subscriptions/" + itoa(id) + "/deliveries/reconcile"
	for range 2 {
		response, err := http.Post(endpoint, "application/json", strings.NewReader(`{"event_id":"missed-decision","payload":{"marker":"replacement"}}`))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusAccepted {
			t.Fatalf("reconcile returned %d", response.StatusCode)
		}
	}
	queue := srv.subStore.(subscription.QueueStore)
	due, err := queue.Due(ctx, 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("expected one durable delivery: %+v %v", due, err)
	}
	var payload struct {
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal([]byte(due[0].PayloadJSON), &payload); err != nil || payload.Payload["marker"] != "original" {
		t.Fatalf("stored event changed: %+v %v", payload, err)
	}
	markQueueDeadLetter(t, queue, id, 5)
	response, err := http.Post(endpoint, "application/json", strings.NewReader(`{"event_id":"missed-decision"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	metadata, err := queue.GetDelivery(ctx, id, "missed-decision")
	if err != nil || metadata.Status != "dead_letter" || metadata.Attempts != 5 {
		t.Fatalf("reconciliation must preserve dead-letter history: %+v %v", metadata, err)
	}
	response, err = http.Post(endpoint, "application/json", strings.NewReader(`{"event_id":"absent"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("missing event returned %d", response.StatusCode)
	}
}

func TestReconciledMissingDeliveryReachesReceiver(t *testing.T) {
	received := make(chan []byte, 1)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(http.StatusAccepted)
	}))
	defer receiver.Close()
	srv, server := newTestServer(t)
	ctx := context.Background()
	_, err := srv.store.Insert(ctx, store.Event{EventID: "before-subscription", EventType: "agent_manager.effort.decision_requested.v1", SourceScenario: "agent-manager", Payload: []byte(`{"question":"retained"}`), CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	id, err := srv.subStore.Create(ctx, subscription.Subscription{Name: "hub", OwnerScenario: "notification-hub", EventPattern: "**.decision_requested.v1", DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: receiver.URL, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(server.URL+"/api/v1/subscriptions/"+itoa(id)+"/deliveries/reconcile", "application/json", strings.NewReader(`{"event_id":"before-subscription"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("reconcile returned %d", response.StatusCode)
	}
	queue := srv.subStore.(subscription.QueueStore)
	srv.drainSubscriptionQueue(ctx, queue)
	select {
	case body := <-received:
		var event struct {
			EventID string         `json:"event_id"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.Unmarshal(body, &event); err != nil || event.EventID != "before-subscription" || event.Payload["question"] != "retained" {
			t.Fatalf("wrong delivered event: %s %v", body, err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reconciled event was not delivered")
	}
	metadata, err := queue.GetDelivery(ctx, id, "before-subscription")
	if err != nil || metadata.Status != "delivered" {
		t.Fatalf("missing delivery receipt: %+v %v", metadata, err)
	}
}

func TestReconcileSubscriptionDeliveryRejectsUnmatchedEvent(t *testing.T) {
	srv, server := newTestServer(t)
	ctx := context.Background()
	_, err := srv.store.Insert(ctx, store.Event{EventID: "other", EventType: "unrelated.event.v1", SourceScenario: "other", Payload: []byte(`{}`), CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	id, err := srv.subStore.Create(ctx, subscription.Subscription{Name: "hub", OwnerScenario: "notification-hub", EventPattern: "**.decision_requested.v1", DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "http://127.0.0.1:1/events", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(server.URL+"/api/v1/subscriptions/"+itoa(id)+"/deliveries/reconcile", "application/json", strings.NewReader(`{"event_id":"other"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("unmatched event returned %d", response.StatusCode)
	}
}
