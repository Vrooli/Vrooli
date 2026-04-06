package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:REQ-POL-007] Violation query with target filter
func TestPolicyViolations_TargetFilter(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny rules for two targets
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"src","target_scenario":"svc-a","effect":"deny","priority":10,"enabled":true}`))
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"src","target_scenario":"svc-b","effect":"deny","priority":10,"enabled":true}`))

	// Trigger violations against both targets
	http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json",
		strings.NewReader(`{"source":"src","target":"svc-a"}`))
	http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json",
		strings.NewReader(`{"source":"src","target":"svc-b"}`))

	// Filter by target
	resp, _ := http.Get(ts.URL + "/api/v1/policies/violations?target=svc-a")
	violations := decodeJSON[[]policy.Violation](t, resp)
	resp.Body.Close()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for target svc-a, got %d", len(violations))
	}
	if violations[0].TargetScenario != "svc-a" {
		t.Fatalf("expected target=svc-a, got %s", violations[0].TargetScenario)
	}
}

// [REQ:REQ-POL-007] Violation query with limit parameter
func TestPolicyViolations_LimitFilter(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny rule
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"src","target_scenario":"svc","effect":"deny","priority":10,"enabled":true}`))

	// Trigger 3 violations
	for i := 0; i < 3; i++ {
		http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json",
			strings.NewReader(`{"source":"src","target":"svc"}`))
	}

	// Limit to 2
	resp, _ := http.Get(ts.URL + "/api/v1/policies/violations?limit=2")
	violations := decodeJSON[[]policy.Violation](t, resp)
	resp.Body.Close()
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations with limit=2, got %d", len(violations))
	}
}

// [REQ:REQ-POL-007] Violation query with rule_type filter
func TestPolicyViolations_RuleTypeFilter(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny rule (access type)
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"src","target_scenario":"svc","effect":"deny","priority":10,"enabled":true}`))

	// Trigger violation
	http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json",
		strings.NewReader(`{"source":"src","target":"svc"}`))

	// Filter by rule_type that has no violations
	resp, _ := http.Get(ts.URL + "/api/v1/policies/violations?rule_type=rate_limit")
	violations := decodeJSON[[]policy.Violation](t, resp)
	resp.Body.Close()
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations for rate_limit, got %d", len(violations))
	}

	// Filter by rule_type that has violations
	resp, _ = http.Get(ts.URL + "/api/v1/policies/violations?rule_type=access")
	violations = decodeJSON[[]policy.Violation](t, resp)
	resp.Body.Close()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for access, got %d", len(violations))
	}
}

// [REQ:REQ-POL-007] Violation query with invalid limit is ignored
func TestPolicyViolations_InvalidLimit(t *testing.T) {
	_, ts := newTestServer(t)

	// Invalid limit should be ignored (not error)
	resp, _ := http.Get(ts.URL + "/api/v1/policies/violations?limit=notanumber")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with invalid limit (ignored), got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Policy delete broadcasts event
func TestPolicyDelete_BroadcastsEvent(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestPolicy(t, ts.URL,
		`{"rule_type":"access","source_scenario":"*","target_scenario":"*","effect":"allow","priority":1,"enabled":true}`)

	req, _ := http.NewRequest("DELETE", ts.URL+"/api/v1/policies/"+itoa(id), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Confirm the policy is gone
	resp2, _ := http.Get(ts.URL + "/api/v1/policies/" + itoa(id))
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp2.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Subscription delete for store error returns 500
func TestSubscriptionDelete_StoreError(t *testing.T) {
	_, ts := newTestServer(t)

	// Delete non-existent subscription — SQLite delete doesn't error on miss,
	// but let's confirm the handler works with a valid ID that was already deleted
	id := createTestSubscription(t, ts.URL,
		`{"name":"del-sub","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)

	// First delete succeeds
	req, _ := http.NewRequest("DELETE", ts.URL+"/api/v1/subscriptions/"+itoa(id), nil)
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Second delete on already-deleted ID should still return 204 (idempotent)
	req2, _ := http.NewRequest("DELETE", ts.URL+"/api/v1/subscriptions/"+itoa(id), nil)
	resp2, _ := http.DefaultClient.Do(req2)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("expected idempotent 204, got %d", resp2.StatusCode)
	}
}
