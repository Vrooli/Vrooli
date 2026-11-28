package incidents

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestResolverEventPublisherUsesCurrentEventsAddress(t *testing.T) {
	var firstCalls, secondCalls atomic.Int32
	makeServer := func(calls *atomic.Int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/events" || r.Method != http.MethodPost {
				t.Errorf("event request = %s %s, want POST /api/v1/events", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode event: %v", err)
			}
			if body["eventType"] != "incident.opened.v1" {
				t.Errorf("event type = %v, want incident.opened.v1", body["eventType"])
			}
			calls.Add(1)
			w.WriteHeader(http.StatusAccepted)
		}))
	}
	first, second := makeServer(&firstCalls), makeServer(&secondCalls)
	defer first.Close()
	defer second.Close()

	addresses := []string{first.URL, second.URL}
	var resolves atomic.Int32
	publisher := NewEventBusPublisherResolver(func(context.Context) (string, error) {
		index := int(resolves.Add(1)) - 1
		return addresses[index], nil
	}).(EventBusPublisher)
	publisher.Client.HTTPClient = first.Client()
	if !publisher.Enabled() {
		t.Fatal("a configured resolver must enable the optional event publisher")
	}
	for range 2 {
		if err := publisher.Publish(context.Background(), "incident.opened.v1", map[string]any{"incident_id": "inc-1"}); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}
	if resolves.Load() != 2 || firstCalls.Load() != 1 || secondCalls.Load() != 1 {
		t.Fatalf("resolutions/first requests/second requests = %d/%d/%d, want 2/1/1", resolves.Load(), firstCalls.Load(), secondCalls.Load())
	}
}

func TestResolverEventPublisherReportsUnavailablePeer(t *testing.T) {
	publisher := NewEventBusPublisherResolver(func(context.Context) (string, error) {
		return "", errors.New("events unavailable")
	}).(EventBusPublisher)
	if !publisher.Enabled() {
		t.Fatal("configured resolver must leave delivery enabled so the durable queue can observe failure")
	}
	if err := publisher.Publish(context.Background(), "incident.opened.v1", nil); err == nil {
		t.Fatal("Publish() error = nil, want unavailable peer error")
	}
}

func TestFixedEventPublisherUsesConfiguredBaseURL(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	publisher := NewEventBusPublisher(server.URL)
	if err := publisher.Publish(context.Background(), "incident.opened.v1", map[string]any{"incident_id": "inc-1"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("configured endpoint calls = %d, want 1", calls.Load())
	}
}
