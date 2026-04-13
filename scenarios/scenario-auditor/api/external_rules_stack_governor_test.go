package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExternalRules_StackGovernorRegistered(t *testing.T) {
	if !isExternalRule("GO_CLI_WORKSPACE_INDEPENDENCE") {
		t.Fatalf("expected GO_CLI_WORKSPACE_INDEPENDENCE to be registered as an external rule")
	}
	if !isExternalRule("REACT_VITE_UI_INSTALLS_DEPENDENCIES") {
		t.Fatalf("expected REACT_VITE_UI_INSTALLS_DEPENDENCIES to be registered as an external rule")
	}
	if !isExternalRule("PACKAGE_GOVERNANCE_SCENARIO_ADOPTION") {
		t.Fatalf("expected PACKAGE_GOVERNANCE_SCENARIO_ADOPTION to be registered as an external rule")
	}
}

func TestPathWithinDir(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "repo", "scenarios", "git-control-tower")

	if !pathWithinDir(base, base) {
		t.Fatalf("expected base to be within itself")
	}
	if !pathWithinDir(filepath.Join(base, ".vrooli", "service.json"), base) {
		t.Fatalf("expected child path to be within base")
	}
	if pathWithinDir(filepath.Join(string(filepath.Separator), "repo", "scenarios", "other", ".vrooli", "service.json"), base) {
		t.Fatalf("expected sibling scenario to not be within base")
	}
	if pathWithinDir(filepath.Join(base, "..", "other"), base) {
		t.Fatalf("expected .. traversal to not be within base")
	}
}

// fixViaURL is a test helper that calls the stack-governor fix endpoint directly,
// bypassing service discovery.
func fixViaURL(client *http.Client, baseURL string, ctx context.Context, scenarioNames []string, ruleIDs []string, dryRun bool) ([]ExternalFixResult, error) {
	payload, _ := json.Marshal(stackGovernorFixRequest{
		ScenarioNames: scenarioNames,
		RuleIDs:       ruleIDs,
		DryRun:        dryRun,
	})

	endpoint := fmt.Sprintf("%s/api/v1/fix", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("scenario-stack-governor fix responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var parsed stackGovernorFixResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	results := make([]ExternalFixResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		changes := make([]ExternalFixChange, 0, len(r.Changes))
		for _, c := range r.Changes {
			changes = append(changes, ExternalFixChange(c))
		}
		results = append(results, ExternalFixResult{
			ScenarioName: r.ScenarioName,
			RuleID:       r.RuleID,
			Fixed:        r.Fixed,
			FilePath:     r.FilePath,
			Changes:      changes,
			Error:        r.Error,
		})
	}

	return results, nil
}

func TestStackGovernorFix_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/fix" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected content-type: %s", ct)
		}

		var req stackGovernorFixRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if len(req.ScenarioNames) != 1 || req.ScenarioNames[0] != "test-scenario" {
			t.Errorf("unexpected scenario_names: %v", req.ScenarioNames)
		}
		if len(req.RuleIDs) != 1 || req.RuleIDs[0] != "MAKEFILE_STRUCTURE" {
			t.Errorf("unexpected rule_ids: %v", req.RuleIDs)
		}
		if req.DryRun {
			t.Errorf("expected dry_run=false")
		}

		resp := stackGovernorFixResponse{
			Results: []stackGovernorFixResult{
				{
					ScenarioName: "test-scenario",
					RuleID:       "MAKEFILE_STRUCTURE",
					Fixed:        true,
					FilePath:     "scenarios/test-scenario/Makefile",
					Changes: []stackGovernorFixChange{
						{Type: "add_target", Detail: "Added missing 'test' target"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer srv.Close()

	results, err := fixViaURL(srv.Client(), srv.URL, context.Background(), []string{"test-scenario"}, []string{"MAKEFILE_STRUCTURE"}, false)
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario_name=test-scenario, got %s", r.ScenarioName)
	}
	if r.RuleID != "MAKEFILE_STRUCTURE" {
		t.Errorf("expected rule_id=MAKEFILE_STRUCTURE, got %s", r.RuleID)
	}
	if !r.Fixed {
		t.Error("expected fixed=true")
	}
	if len(r.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(r.Changes))
	}
	if r.Changes[0].Type != "add_target" {
		t.Errorf("expected change type=add_target, got %s", r.Changes[0].Type)
	}
}

func TestStackGovernorFix_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fixViaURL(srv.Client(), srv.URL, context.Background(), []string{"test-scenario"}, []string{"MAKEFILE_STRUCTURE"}, false)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code 500, got: %s", err.Error())
	}
}

func TestStackGovernorFix_Unavailable(t *testing.T) {
	client := &http.Client{Timeout: 1 * time.Second}
	_, err := fixViaURL(client, "http://127.0.0.1:1", context.Background(), []string{"test-scenario"}, []string{"MAKEFILE_STRUCTURE"}, false)
	if err == nil {
		t.Fatal("expected error for unavailable server")
	}
}
