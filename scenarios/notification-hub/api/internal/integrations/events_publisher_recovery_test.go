package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

func TestAgentManagerResolutionStaysInOutboxUntilExactQueueReconciles(t *testing.T) {
	ready := false
	publishedID := ""
	reconciled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/events":
			var event map[string]any
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Error(err)
			}
			publishedID, _ = event["eventId"].(string)
			if publishedID == "" {
				t.Error("event publisher did not send an identity")
			}
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet:
			if ready {
				_, _ = w.Write([]byte(`[{"id":9,"name":"agent-manager-decision-answers","enabled":true,"event_pattern":"notification_hub.ask.resolved.v1"}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/subscriptions/9/deliveries/reconcile":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["event_id"] != publishedID {
				t.Errorf("event identity mismatch: %+v %v", body, err)
			}
			reconciled = true
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	publisher := EventsPublisher{BaseURL: func() string { return server.URL }}
	event := eventbus.DomainEvent{Source: "notification-hub", EventType: "notification_hub.ask.resolved.v1", Occurred: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Payload: map[string]any{"source": "agent-manager"}}
	if err := publisher.PublishDomainEvent(context.Background(), event); err == nil {
		t.Fatal("missing subscriber must retain the resolution in the outbox")
	}
	ready = true
	if err := publisher.PublishDomainEvent(context.Background(), event); err != nil || !reconciled {
		t.Fatalf("recovery failed: %v", err)
	}
}
