package subscription

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:SUB-003] Deliver sends a well-formed POST to the target URL.
func TestWebhookDeliverer_Success(t *testing.T) {
	var received WebhookPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}
		if ua := r.Header.Get("User-Agent"); ua != "vrooli-events/1.0" {
			t.Errorf("expected vrooli-events/1.0, got %s", ua)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &received); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	d := NewWebhookDeliverer()
	payload := WebhookPayload{
		EventID:        "evt-1",
		EventType:      "test.webhook",
		SourceScenario: "test-scenario",
		DeliveredAt:    time.Now().UTC().Format(time.RFC3339),
	}

	err := d.Deliver(context.Background(), ts.URL, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.EventID != "evt-1" {
		t.Errorf("expected event_id evt-1, got %s", received.EventID)
	}
	if received.EventType != "test.webhook" {
		t.Errorf("expected event_type test.webhook, got %s", received.EventType)
	}
}

// [REQ:SUB-003] Deliver returns an error when the target returns >= 400.
func TestWebhookDeliverer_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	d := NewWebhookDeliverer()
	err := d.Deliver(context.Background(), ts.URL, WebhookPayload{EventID: "e1"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// [REQ:SUB-003] Deliver returns an error for unreachable targets.
func TestWebhookDeliverer_ConnectionError(t *testing.T) {
	d := NewWebhookDeliverer()
	err := d.Deliver(context.Background(), "http://127.0.0.1:1", WebhookPayload{EventID: "e2"})
	if err == nil {
		t.Fatal("expected error for unreachable target")
	}
}

// [REQ:SUB-003] Deliver includes the payload field when non-nil.
func TestWebhookDeliverer_WithPayload(t *testing.T) {
	var received WebhookPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	d := NewWebhookDeliverer()
	payload := WebhookPayload{
		EventID:        "evt-payload",
		EventType:      "test.payload",
		SourceScenario: "src",
		Payload:        map[string]string{"key": "value"},
		DeliveredAt:    time.Now().UTC().Format(time.RFC3339),
	}

	if err := d.Deliver(context.Background(), ts.URL, payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Payload == nil {
		t.Fatal("expected payload to be present")
	}
}

// [REQ:SUB-003] Deliver respects context cancellation.
func TestWebhookDeliverer_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	d := NewWebhookDeliverer()
	err := d.Deliver(ctx, ts.URL, WebhookPayload{EventID: "e3"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
