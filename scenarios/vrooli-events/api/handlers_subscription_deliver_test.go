package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// [REQ:SUB-003] Deliver endpoint returns 404 for nonexistent subscription.
func TestDeliverSubscription_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"event_id":"e1","event_type":"test"}`
	resp, err := http.Post(ts.URL+"/api/v1/subscriptions/9999/deliver", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:SUB-003] Deliver endpoint rejects non-webhook subscriptions.
func TestDeliverSubscription_NonWebhookRejected(t *testing.T) {
	_, ts := newTestServer(t)

	// Create an SSE subscription (not webhook).
	id := createTestSubscription(t, ts.URL, `{"name":"sse-sub","owner_scenario":"test","event_pattern":"*","delivery_type":"sse"}`)

	body := `{"event_id":"e1","event_type":"test"}`
	resp, err := http.Post(ts.URL+"/api/v1/subscriptions/"+itoa(id)+"/deliver", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:SUB-003] Deliver endpoint successfully delivers to a webhook target.
func TestDeliverSubscription_Success(t *testing.T) {
	_, ts := newTestServer(t)

	// Set up a mock webhook server.
	var receivedBody []byte
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Create a webhook subscription pointing at the mock server.
	id := createTestSubscription(t, ts.URL, `{"name":"wh-sub","owner_scenario":"test","event_pattern":"*","delivery_type":"webhook","delivery_target":"`+webhookServer.URL+`"}`)

	// Trigger delivery.
	payload := `{"event_id":"evt-42","event_type":"test.deliver","source_scenario":"src"}`
	resp, err := http.Post(ts.URL+"/api/v1/subscriptions/"+itoa(id)+"/deliver", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	// Verify the webhook received the payload.
	var received map[string]any
	if err := json.Unmarshal(receivedBody, &received); err != nil {
		t.Fatalf("unmarshal webhook body: %v", err)
	}
	if received["event_id"] != "evt-42" {
		t.Errorf("expected event_id evt-42, got %v", received["event_id"])
	}

	// Verify API response.
	_ = decodeJSON[map[string]any](t, resp)
}

// [REQ:SUB-003] Deliver endpoint returns 502 when webhook target fails.
func TestDeliverSubscription_WebhookFails(t *testing.T) {
	_, ts := newTestServer(t)

	// Set up a failing webhook server.
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer webhookServer.Close()

	id := createTestSubscription(t, ts.URL, `{"name":"wh-fail","owner_scenario":"test","event_pattern":"*","delivery_type":"webhook","delivery_target":"`+webhookServer.URL+`"}`)

	payload := `{"event_id":"evt-fail","event_type":"test.fail"}`
	resp, err := http.Post(ts.URL+"/api/v1/subscriptions/"+itoa(id)+"/deliver", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
}

// [REQ:SUB-003] Deliver endpoint returns 400 for invalid body.
func TestDeliverSubscription_InvalidBody(t *testing.T) {
	_, ts := newTestServer(t)

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	id := createTestSubscription(t, ts.URL, `{"name":"wh-bad","owner_scenario":"test","event_pattern":"*","delivery_type":"webhook","delivery_target":"`+webhookServer.URL+`"}`)

	resp, err := http.Post(ts.URL+"/api/v1/subscriptions/"+itoa(id)+"/deliver", "application/json", strings.NewReader("not json"))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:SUB-003] Deliver endpoint returns 400 for invalid ID.
func TestDeliverSubscription_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	resp, err := http.Post(ts.URL+"/api/v1/subscriptions/notanumber/deliver", "application/json", strings.NewReader(`{"event_id":"e1"}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:SUB-003] Deliver endpoint response includes subscription ID and target.
func TestDeliverSubscription_ResponseFields(t *testing.T) {
	_, ts := newTestServer(t)

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	id := createTestSubscription(t, ts.URL, `{"name":"wh-resp","owner_scenario":"test","event_pattern":"*","delivery_type":"webhook","delivery_target":"`+webhookServer.URL+`"}`)

	payload := `{"event_id":"evt-resp","event_type":"test.v1"}`
	resp, err := http.Post(ts.URL+"/api/v1/subscriptions/"+itoa(id)+"/deliver", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	result := decodeJSON[map[string]any](t, resp)
	if result["status"] != "delivered" {
		t.Fatalf("expected status=delivered, got %v", result["status"])
	}
	if result["subscription_id"] != float64(id) {
		t.Fatalf("expected subscription_id=%d, got %v", id, result["subscription_id"])
	}
	if result["target"] != webhookServer.URL {
		t.Fatalf("expected target=%s, got %v", webhookServer.URL, result["target"])
	}
}
