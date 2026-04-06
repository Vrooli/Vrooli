package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:REQ-POL-007] Policy violations logged on deny
func TestPolicyViolations_LoggedOnDeny(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny rule
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"bad-src","target_scenario":"protected-svc","effect":"deny","priority":10,"enabled":true}`))

	// Trigger evaluation that will be denied
	http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json",
		strings.NewReader(`{"source":"bad-src","target":"protected-svc","endpoint":"/api/secret"}`))

	// Check violations
	resp, _ := http.Get(ts.URL + "/api/v1/policies/violations")
	defer resp.Body.Close()

	violations := decodeJSON[[]policy.Violation](t, resp)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].SourceScenario != "bad-src" {
		t.Fatalf("expected source=bad-src, got %s", violations[0].SourceScenario)
	}
	if violations[0].Endpoint != "/api/secret" {
		t.Fatalf("expected endpoint=/api/secret, got %s", violations[0].Endpoint)
	}
}

// [REQ:REQ-POL-007] Violation query with filters
func TestPolicyViolations_Filters(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny rules for two different sources
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"src-a","target_scenario":"svc","effect":"deny","priority":10,"enabled":true}`))
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"src-b","target_scenario":"svc","effect":"deny","priority":10,"enabled":true}`))

	// Trigger violations
	http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json",
		strings.NewReader(`{"source":"src-a","target":"svc"}`))
	http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json",
		strings.NewReader(`{"source":"src-b","target":"svc"}`))

	// Filter by source
	resp, _ := http.Get(ts.URL + "/api/v1/policies/violations?source=src-a")
	violations := decodeJSON[[]policy.Violation](t, resp)
	resp.Body.Close()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for src-a, got %d", len(violations))
	}
}

// [REQ:REQ-POL-007] Empty violations list returns empty array
func TestPolicyViolations_Empty(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/policies/violations")
	defer resp.Body.Close()

	body := decodeJSON[json.RawMessage](t, resp)
	if string(body) != "[]" {
		t.Fatalf("expected [], got %s", body)
	}
}
