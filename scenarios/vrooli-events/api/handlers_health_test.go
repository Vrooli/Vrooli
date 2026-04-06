package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/testutil"
)

// [REQ:REQ-API-003] Health endpoint returns schema-compliant response
func TestHealthEndpoint(t *testing.T) {
	_, ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeJSON[map[string]any](t, resp)
	if body["status"] != "healthy" {
		t.Fatalf("expected healthy, got %v", body["status"])
	}
	if body["service"] != "vrooli-events" {
		t.Fatalf("expected service=vrooli-events, got %v", body["service"])
	}
	if body["readiness"] != true {
		t.Fatalf("expected readiness=true, got %v", body["readiness"])
	}
	ts_, ok := body["timestamp"].(string)
	if !ok || ts_ == "" {
		t.Fatal("expected non-empty timestamp")
	}
	if _, err := time.Parse(time.RFC3339, ts_); err != nil {
		t.Fatalf("timestamp not RFC3339: %v", err)
	}
}

// [REQ:REQ-API-003] Health returns unhealthy when store is down
func TestHealthEndpoint_StoreError(t *testing.T) {
	ms := (&testutil.MockStore{}).WithStatsError(fmt.Errorf("db gone"))
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}

	body := decodeJSON[map[string]any](t, resp)
	if body["status"] != "unhealthy" {
		t.Fatalf("expected unhealthy, got %v", body["status"])
	}
	if body["readiness"] != false {
		t.Fatalf("expected readiness=false, got %v", body["readiness"])
	}
}

// [REQ:REQ-API-003] Health reflects subscriber count from broker
func TestHealthEndpoint_SubscriberCount(t *testing.T) {
	ms := (&testutil.MockStore{}).WithStatsResult(store.Stats{TotalEvents: 42, TotalPayloadBytes: 1024}, nil)
	mb := testutil.NewMockBroker().WithSubscriberCount(5)
	ts := newMockedServer(t, ms, mb)

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeJSON[map[string]any](t, resp)
	if body["subscribers"] != float64(5) {
		t.Fatalf("expected subscribers=5, got %v", body["subscribers"])
	}
	storeData := body["store"].(map[string]any)
	if storeData["totalEvents"] != float64(42) {
		t.Fatalf("expected totalEvents=42, got %v", storeData["totalEvents"])
	}
}
