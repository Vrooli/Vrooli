package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:REQ-POL-004] Policy CRUD - create access rule
func TestPolicyCreate_AccessRule(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{
		"rule_type": "access",
		"source_scenario": "scenario-a",
		"target_scenario": "scenario-b",
		"effect": "deny",
		"priority": 10,
		"enabled": true
	}`
	resp, err := http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	result := decodeJSON[map[string]any](t, resp)
	if result["id"] == nil {
		t.Fatal("expected id in response")
	}
}

// [REQ:REQ-POL-004] Policy CRUD - create rate limit rule
func TestPolicyCreate_RateLimitRule(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{
		"rule_type": "rate_limit",
		"source_scenario": "scenario-a",
		"target_scenario": "scenario-b",
		"max_requests": 100,
		"window_seconds": 60,
		"enabled": true
	}`
	resp, err := http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-003] Policy CRUD - create circuit breaker rule
func TestPolicyCreate_CircuitBreakerRule(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{
		"rule_type": "circuit_breaker",
		"source_scenario": "scenario-a",
		"target_scenario": "scenario-b",
		"failure_threshold": 5,
		"cooldown_seconds": 30,
		"success_threshold": 2,
		"enabled": true
	}`
	resp, err := http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Validation rejects invalid rule types
func TestPolicyCreate_Validation(t *testing.T) {
	_, ts := newTestServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing rule_type", `{"source_scenario":"s","target_scenario":"t","effect":"allow"}`},
		{"invalid rule_type", `{"rule_type":"invalid","source_scenario":"s","target_scenario":"t"}`},
		{"missing source", `{"rule_type":"access","target_scenario":"t","effect":"allow"}`},
		{"missing target", `{"rule_type":"access","source_scenario":"s","effect":"allow"}`},
		{"missing effect for access", `{"rule_type":"access","source_scenario":"s","target_scenario":"t"}`},
		{"rate_limit no max_requests", `{"rule_type":"rate_limit","source_scenario":"s","target_scenario":"t","window_seconds":60}`},
		{"rate_limit no window", `{"rule_type":"rate_limit","source_scenario":"s","target_scenario":"t","max_requests":100}`},
		{"circuit_breaker no threshold", `{"rule_type":"circuit_breaker","source_scenario":"s","target_scenario":"t"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

// [REQ:REQ-POL-004] Policy CRUD - list rules with filters
func TestPolicyList(t *testing.T) {
	_, ts := newTestServer(t)

	// Create two rules of different types
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"access","source_scenario":"s1","target_scenario":"t1","effect":"allow","enabled":true}`))
	http.Post(ts.URL+"/api/v1/policies", "application/json",
		strings.NewReader(`{"rule_type":"rate_limit","source_scenario":"s2","target_scenario":"t2","max_requests":10,"window_seconds":60,"enabled":true}`))

	// List all
	resp, _ := http.Get(ts.URL + "/api/v1/policies")
	rules := decodeJSON[[]policy.Rule](t, resp)
	resp.Body.Close()
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	// List by type
	resp, _ = http.Get(ts.URL + "/api/v1/policies?rule_type=access")
	rules = decodeJSON[[]policy.Rule](t, resp)
	resp.Body.Close()
	if len(rules) != 1 {
		t.Fatalf("expected 1 access rule, got %d", len(rules))
	}
	if rules[0].RuleType != "access" {
		t.Fatalf("expected access type, got %s", rules[0].RuleType)
	}

	// List by source
	resp, _ = http.Get(ts.URL + "/api/v1/policies?source=s1")
	rules = decodeJSON[[]policy.Rule](t, resp)
	resp.Body.Close()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule for source s1, got %d", len(rules))
	}
}

// [REQ:REQ-POL-004] Policy CRUD - get single rule
func TestPolicyGet(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow","enabled":true,"priority":5}`)

	resp, _ := http.Get(fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, id))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	rule := decodeJSON[policy.Rule](t, resp)
	if rule.SourceScenario != "s" {
		t.Fatalf("expected source=s, got %s", rule.SourceScenario)
	}
	if rule.Priority != 5 {
		t.Fatalf("expected priority=5, got %d", rule.Priority)
	}
}

// [REQ:REQ-POL-004] Policy CRUD - get nonexistent returns 404
func TestPolicyGet_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/policies/99999")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Policy CRUD - update rule
func TestPolicyUpdate(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow","enabled":true}`)

	// Update
	updateBody := `{"rule_type":"access","source_scenario":"s-updated","target_scenario":"t","effect":"deny","enabled":true}`
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, id), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify
	resp2, _ := http.Get(fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, id))
	rule := decodeJSON[policy.Rule](t, resp2)
	resp2.Body.Close()
	if rule.SourceScenario != "s-updated" {
		t.Fatalf("expected source=s-updated, got %s", rule.SourceScenario)
	}
	if rule.Effect != "deny" {
		t.Fatalf("expected effect=deny, got %s", rule.Effect)
	}
}

// [REQ:REQ-POL-004] Policy CRUD - delete rule
func TestPolicyDelete(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow","enabled":true}`)

	// Delete
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, id), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify gone
	resp, _ = http.Get(fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, id))
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Empty policy list returns empty array
func TestPolicyList_Empty(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/policies")
	defer resp.Body.Close()

	body := decodeJSON[json.RawMessage](t, resp)
	if string(body) != "[]" {
		t.Fatalf("expected [], got %s", body)
	}
}

// [REQ:REQ-POL-004] Policy CRUD - get with invalid (non-numeric) ID returns 400
func TestPolicyGet_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/policies/notanumber")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeInvalidParam {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidParam, errBody["code"])
	}
}

// [REQ:REQ-POL-004] Policy CRUD - update nonexistent returns 404
func TestPolicyUpdate_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow"}`
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/policies/99999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Policy CRUD - update with invalid body returns 400
func TestPolicyUpdate_InvalidBody(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow","enabled":true}`)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, id), strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Policy CRUD - update with validation errors returns 400
func TestPolicyUpdate_ValidationError(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow","enabled":true}`)

	// Missing source_scenario
	body := `{"rule_type":"access","target_scenario":"t","effect":"allow"}`
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policies/%d", ts.URL, id), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeValidation {
		t.Fatalf("expected %s, got %s", ErrCodeValidation, errBody["code"])
	}
}

// [REQ:REQ-POL-004] Policy CRUD - delete with invalid ID returns 400
func TestPolicyDelete_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	req, _ := http.NewRequest("DELETE", ts.URL+"/api/v1/policies/notanumber", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Policy list filters by target scenario
func TestPolicyList_TargetFilter(t *testing.T) {
	_, ts := newTestServer(t)

	createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s1","target_scenario":"target-a","effect":"allow","enabled":true}`)
	createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s2","target_scenario":"target-b","effect":"deny","enabled":true}`)

	resp, _ := http.Get(ts.URL + "/api/v1/policies?target=target-a")
	rules := decodeJSON[[]policy.Rule](t, resp)
	resp.Body.Close()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule for target-a, got %d", len(rules))
	}
	if rules[0].TargetScenario != "target-a" {
		t.Fatalf("expected target=target-a, got %s", rules[0].TargetScenario)
	}
}

// [REQ:REQ-POL-004] Policy list filters by enabled status
func TestPolicyList_EnabledFilter(t *testing.T) {
	_, ts := newTestServer(t)

	createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s1","target_scenario":"t1","effect":"allow","enabled":true}`)
	createTestPolicy(t, ts.URL, `{"rule_type":"access","source_scenario":"s2","target_scenario":"t2","effect":"deny","enabled":false}`)

	resp, _ := http.Get(ts.URL + "/api/v1/policies?enabled=true")
	rules := decodeJSON[[]policy.Rule](t, resp)
	resp.Body.Close()
	if len(rules) != 1 {
		t.Fatalf("expected 1 enabled rule, got %d", len(rules))
	}

	resp, _ = http.Get(ts.URL + "/api/v1/policies?enabled=false")
	rules = decodeJSON[[]policy.Rule](t, resp)
	resp.Body.Close()
	if len(rules) != 1 {
		t.Fatalf("expected 1 disabled rule, got %d", len(rules))
	}
}

// [REQ:REQ-POL-004] Policy CRUD - create with invalid JSON returns 400
func TestPolicyCreate_InvalidJSON(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader("not json"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeInvalidBody {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidBody, errBody["code"])
	}
}

// [REQ:REQ-POL-004] Policy CRUD - update with invalid (non-numeric) ID returns 400
func TestPolicyUpdate_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"allow"}`
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/policies/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-POL-004] Invalid effect value for access rule returns 400
func TestPolicyCreate_InvalidEffect(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"rule_type":"access","source_scenario":"s","target_scenario":"t","effect":"maybe"}`
	resp, _ := http.Post(ts.URL+"/api/v1/policies", "application/json", strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
