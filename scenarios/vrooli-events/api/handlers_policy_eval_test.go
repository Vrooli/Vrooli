package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:REQ-POL-001] Access control rules: deny rule blocks matching scenarios
func TestPolicyEvaluate_AccessDeny(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny rule
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"blocked-src","target_scenario":"target-svc","effect":"deny","priority":10,"enabled":true}`))

	// Evaluate
	body := `{"source":"blocked-src","target":"target-svc","endpoint":"/api/v1/something"}`
	resp, err := http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	defer resp.Body.Close()

	decision := decodeJSON[policy.Decision](t, resp)
	if decision.Allowed {
		t.Fatal("expected deny, got allow")
	}
	if decision.RuleType != "access" {
		t.Fatalf("expected rule_type=access, got %s", decision.RuleType)
	}
}

// [REQ:REQ-POL-001] Access control rules: allow rule permits matching scenarios
func TestPolicyEvaluate_AccessAllow(t *testing.T) {
	_, ts := newTestServer(t)

	// Create allow rule
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"allowed-src","target_scenario":"target-svc","effect":"allow","priority":10,"enabled":true}`))

	body := `{"source":"allowed-src","target":"target-svc"}`
	resp, err := http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	defer resp.Body.Close()

	decision := decodeJSON[policy.Decision](t, resp)
	if !decision.Allowed {
		t.Fatal("expected allow, got deny")
	}
}

// [REQ:REQ-POL-001] Default allow when no rules match
func TestPolicyEvaluate_DefaultAllow(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"source":"unknown-src","target":"unknown-tgt"}`
	resp, err := http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	defer resp.Body.Close()

	decision := decodeJSON[policy.Decision](t, resp)
	if !decision.Allowed {
		t.Fatal("expected default allow when no rules match")
	}
}

// [REQ:REQ-POL-006] Policy evaluation with priority ordering
func TestPolicyEvaluate_PriorityOrdering(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny with lower priority
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"*","target_scenario":"svc","effect":"deny","priority":1,"enabled":true}`))
	// Create allow with higher priority
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"trusted","target_scenario":"svc","effect":"allow","priority":10,"enabled":true}`))

	body := `{"source":"trusted","target":"svc"}`
	resp, _ := http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json", strings.NewReader(body))
	defer resp.Body.Close()

	decision := decodeJSON[policy.Decision](t, resp)
	if !decision.Allowed {
		t.Fatal("expected allow from higher-priority rule")
	}
}

// [REQ:REQ-POL-006] Evaluation validation
func TestPolicyEvaluate_MissingFields(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"source":"","target":""}`
	resp, _ := http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json", strings.NewReader(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-001] Glob patterns in access control rules
func TestPolicyEvaluate_GlobPatterns(t *testing.T) {
	_, ts := newTestServer(t)

	// Create deny rule with glob source
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"*","target_scenario":"protected","effect":"deny","priority":5,"enabled":true}`))

	body := `{"source":"any-scenario","target":"protected"}`
	resp, _ := http.Post(ts.URL+"/api/v1/policies/evaluate", "application/json", strings.NewReader(body))
	defer resp.Body.Close()

	decision := decodeJSON[policy.Decision](t, resp)
	if decision.Allowed {
		t.Fatal("expected deny from glob pattern rule")
	}
}
