package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rulespkg "scenario-auditor/rules"
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

func TestStackGovernorRun_MapsPackageGovernanceFindingToViolation(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioDir := filepath.Join(repoRoot, "scenarios", "alpha")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir repo contract dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"), []byte(`{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "layout": {
    "scenarioDir": "scenarios",
    "resourceDir": "resources",
    "packagesDir": "packages",
    "templates": {
      "scenarioDir": "templates/scenarios"
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/run" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp := stackGovernorRunResponse{
			RepoRoot: repoRoot,
			Results: []stackGovernorRuleResult{
				{
					RuleID: "PACKAGE_GOVERNANCE_SCENARIO_ADOPTION",
					Passed: false,
					Findings: []stackGovernorFinding{
						{
							Level:   "error",
							Message: `alpha: real scenario "alpha" uses workspace:* for shared package adoption`,
							Evidence: []stackGovernorEvidence{
								{Type: "file", Ref: filepath.Join(scenarioDir, "ui", "package.json")},
								{Type: "note", Detail: "Replace workspace-star shared-package references with file:."},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	provider := &stackGovernorProvider{
		client: srv.Client(),
		ruleLookup: map[string]rulespkg.Rule{
			"PACKAGE_GOVERNANCE_SCENARIO_ADOPTION": {
				ID:       "PACKAGE_GOVERNANCE_SCENARIO_ADOPTION",
				Name:     "Scenario shared-package adoption follows package governance policy",
				Standard: "stack-governance",
			},
		},
	}

	violations, err := provider.runAgainstBaseURL(context.Background(), srv.URL, "alpha", []string{"PACKAGE_GOVERNANCE_SCENARIO_ADOPTION"})
	if err != nil {
		t.Fatalf("runAgainstBaseURL: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %#v", len(violations), violations)
	}
	if violations[0].ScenarioName != "alpha" {
		t.Fatalf("scenario name = %q", violations[0].ScenarioName)
	}
	if violations[0].Type != "PACKAGE_GOVERNANCE_SCENARIO_ADOPTION" {
		t.Fatalf("type = %q", violations[0].Type)
	}
	if violations[0].Severity != "high" {
		t.Fatalf("severity = %q", violations[0].Severity)
	}
	if !strings.Contains(violations[0].Description, "workspace:*") {
		t.Fatalf("description = %q", violations[0].Description)
	}
	if violations[0].FilePath != filepath.Join(scenarioDir, "ui", "package.json") {
		t.Fatalf("file path = %q", violations[0].FilePath)
	}
}
