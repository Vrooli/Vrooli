package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

// [REQ:REQ-SUB-002] Glob pattern validation on creation
func TestSubscriptionCreate_GlobPattern(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{
		"name": "glob-sub",
		"owner_scenario": "monitor",
		"event_pattern": "swarm-manager.backlog.**",
		"delivery_type": "sse",
		"enabled": true
	}`
	resp, _ := http.Post(ts.URL+"/api/v1/subscriptions", "application/json", strings.NewReader(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-004] Subscription health endpoint
func TestSubscriptionHealth(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"health-sub","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)

	resp, _ := http.Get(fmt.Sprintf("%s/api/v1/subscriptions/%d/health", ts.URL, id))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	health := decodeJSON[subscription.Health](t, resp)
	if health.Status != "active" {
		t.Fatalf("expected status=active, got %s", health.Status)
	}
	if health.TotalDelivered != 0 {
		t.Fatalf("expected total_delivered=0, got %d", health.TotalDelivered)
	}
}

// [REQ:REQ-SUB-005] Subscription test endpoint
func TestSubscriptionTest(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"test-sub","owner_scenario":"o","event_pattern":"app.**","delivery_type":"sse","enabled":true}`)

	resp, _ := http.Post(fmt.Sprintf("%s/api/v1/subscriptions/%d/test", ts.URL, id), "application/json", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	result := decodeJSON[map[string]any](t, resp)
	if result["test_result"] != "ok" {
		t.Fatalf("expected test_result=ok, got %v", result["test_result"])
	}
}

// [REQ:REQ-SUB-004] Subscription health returns 404 for nonexistent
func TestSubscriptionHealth_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions/99999/health")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-004] Subscription health returns 400 for invalid ID
func TestSubscriptionHealth_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions/notanumber/health")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-005] Subscription test returns 404 for nonexistent
func TestSubscriptionTest_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Post(ts.URL+"/api/v1/subscriptions/99999/test", "application/json", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-005] Subscription test returns 400 for invalid ID
func TestSubscriptionTest_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Post(ts.URL+"/api/v1/subscriptions/abc/test", "application/json", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-005] Subscription test response includes subscription details
func TestSubscriptionTest_ResponseDetails(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"detail-sub","owner_scenario":"o","event_pattern":"evt.*","delivery_type":"webhook","delivery_target":"https://example.com/hook","enabled":true}`)

	resp, _ := http.Post(fmt.Sprintf("%s/api/v1/subscriptions/%d/test", ts.URL, id), "application/json", nil)
	defer resp.Body.Close()

	result := decodeJSON[map[string]any](t, resp)
	if result["name"] != "detail-sub" {
		t.Fatalf("expected name=detail-sub, got %v", result["name"])
	}
	if result["event_pattern"] != "evt.*" {
		t.Fatalf("expected event_pattern=evt.*, got %v", result["event_pattern"])
	}
	if result["delivery_type"] != "webhook" {
		t.Fatalf("expected delivery_type=webhook, got %v", result["delivery_type"])
	}
}
