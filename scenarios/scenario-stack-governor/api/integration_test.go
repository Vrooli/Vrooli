package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postJSON(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(ts.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestIntegration_ListRulesExcludesSupersededRules(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/rules")
	if err != nil {
		t.Fatalf("GET /api/v1/rules: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Rules []struct {
			ID string `json:"id"`
		} `json:"rules"`
	}
	decodeJSON(t, resp, &payload)

	seen := map[string]bool{}
	for _, rule := range payload.Rules {
		seen[rule.ID] = true
	}
	for _, id := range []string{
		"REACT_VITE_UI_INSTALLS_DEPENDENCIES",
		packageGovernanceRuleID,
	} {
		if !seen[id] {
			t.Fatalf("expected %s in rule catalog", id)
		}
	}
	for _, id := range []string{"GO_CLI_WORKSPACE_INDEPENDENCE", "MAKEFILE_STRUCTURE", "MAKEFILE_LIFECYCLE", "MAKEFILE_QUALITY"} {
		if seen[id] {
			t.Fatalf("superseded rule %s should not be in rule catalog", id)
		}
	}
}

func TestIntegration_RunWithAllRulesDisabledReturnsNoResults(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp := postJSON(t, ts, "/api/v1/run", RunRequest{ScenarioNames: []string{"demo"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var runResp RunResponse
	decodeJSON(t, resp, &runResp)
	if len(runResp.Results) != 0 {
		t.Fatalf("expected no results with all test rules disabled, got %d", len(runResp.Results))
	}
}

func TestIntegration_RunRejectsInvalidJSON(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/v1/run", "application/json", strings.NewReader("{invalid"))
	if err != nil {
		t.Fatalf("POST /run: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestIntegration_FixPanicRecovery(t *testing.T) {
	entry := RuleEntry{
		Definition: RuleDefinition{ID: "PANIC_TEST", Fixable: true},
		Fixer: func(ctx context.Context, repoRoot, scenario string, dryRun bool) []FixResult {
			panic("intentional test panic")
		},
	}

	results := callFixerSafe(t.Context(), entry, "/tmp", "test-scenario", false)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Fixed {
		t.Error("expected fixed=false after panic")
	}
	if r.Error == "" || !strings.Contains(r.Error, "fixer panicked") {
		t.Errorf("expected panic error, got: %s", r.Error)
	}
	if r.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got: %s", r.ScenarioName)
	}
}

func TestRegistryConsistency(t *testing.T) {
	seen := map[string]bool{}
	for _, entry := range AllRules() {
		if entry.Runner == nil {
			t.Errorf("rule %s has nil Runner", entry.Definition.ID)
		}
		if entry.Definition.Fixable && entry.Fixer == nil {
			t.Errorf("rule %s is marked Fixable but has nil Fixer", entry.Definition.ID)
		}
		if !entry.Definition.Fixable && entry.Fixer != nil {
			t.Errorf("rule %s is not Fixable but has a Fixer set", entry.Definition.ID)
		}
		if entry.Definition.ID == "" {
			t.Error("found rule with empty ID")
		}
		if seen[entry.Definition.ID] {
			t.Errorf("duplicate rule ID: %s", entry.Definition.ID)
		}
		seen[entry.Definition.ID] = true
	}
}
