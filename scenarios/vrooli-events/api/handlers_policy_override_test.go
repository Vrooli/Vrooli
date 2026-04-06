package main

import (
	"net/http"
	"testing"
)

const testCircuitBreakerBody = `{"rule_type":"circuit_breaker","source_scenario":"s","target_scenario":"t","failure_threshold":5,"cooldown_seconds":30,"enabled":true}`

// [REQ:REQ-POL-008] Circuit breaker manual override - set state to open
func TestCircuitBreakerOverride_Open(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, testCircuitBreakerBody)

	resp := postOverride(t, ts.URL, id, `{"state":"open"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	result := decodeJSON[map[string]any](t, resp)
	if result["status"] != "overridden" {
		t.Fatalf("expected status=overridden, got %v", result["status"])
	}
	if result["state"] != "open" {
		t.Fatalf("expected state=open, got %v", result["state"])
	}
}

// [REQ:REQ-POL-008] Circuit breaker manual override - set state to closed
func TestCircuitBreakerOverride_Closed(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, testCircuitBreakerBody)

	resp := postOverride(t, ts.URL, id, `{"state":"closed"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-008] Circuit breaker manual override - set state to half_open
func TestCircuitBreakerOverride_HalfOpen(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, testCircuitBreakerBody)

	resp := postOverride(t, ts.URL, id, `{"state":"half_open"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-008] Override rejects invalid state
func TestCircuitBreakerOverride_InvalidState(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, testCircuitBreakerBody)

	resp := postOverride(t, ts.URL, id, `{"state":"invalid"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-008] Override rejects non-circuit-breaker rules
func TestCircuitBreakerOverride_WrongRuleType(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow","enabled":true}`)

	resp := postOverride(t, ts.URL, id, `{"state":"open"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-008] Override with custom TTL
func TestCircuitBreakerOverride_CustomTTL(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, testCircuitBreakerBody)

	resp := postOverride(t, ts.URL, id, `{"state":"open","ttl_seconds":1800}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	result := decodeJSON[map[string]any](t, resp)
	if result["ttl_seconds"] != float64(1800) {
		t.Fatalf("expected ttl_seconds=1800, got %v", result["ttl_seconds"])
	}
}

// [REQ:REQ-POL-008] Override for nonexistent rule returns 404
func TestCircuitBreakerOverride_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	resp := postOverride(t, ts.URL, 99999, `{"state":"open"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-008] Override with invalid body returns 400
func TestCircuitBreakerOverride_InvalidBody(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, testCircuitBreakerBody)

	resp := postOverride(t, ts.URL, id, "not json")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-008] Override with zero TTL defaults to 3600
func TestCircuitBreakerOverride_DefaultTTL(t *testing.T) {
	_, ts := newTestServer(t)
	id := createTestPolicy(t, ts.URL, testCircuitBreakerBody)

	resp := postOverride(t, ts.URL, id, `{"state":"open","ttl_seconds":0}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	result := decodeJSON[map[string]any](t, resp)
	if result["ttl_seconds"] != float64(3600) {
		t.Fatalf("expected default ttl_seconds=3600, got %v", result["ttl_seconds"])
	}
}
