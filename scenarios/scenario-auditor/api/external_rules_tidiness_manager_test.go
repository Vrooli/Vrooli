package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExternalRules_TidinessManagerRegistered(t *testing.T) {
	for _, ruleID := range []string{
		"TS_CONFIG_STRICT",
		"ESLINT_SAFETY_RULES",
		"TS_DANGEROUS_PATTERNS",
		"ESLINT_TYPED_CONFIG",
		"NODE_BUILD_TYPECHECK",
		"TESTING_CONFIG_LINT_STRICT",
		"GO_MOD_PRESENT_FOR_API_OR_CLI",
		"GO_LINT_CONFIG_PRESENT",
		"GO_LINT_REQUIRED_LINTERS",
		"MAKEFILE_QUALITY_GATES",
	} {
		if !isExternalRule(ruleID) {
			t.Fatalf("expected %s to be registered as an external rule", ruleID)
		}
	}
}

func TestTidinessManagerFix_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scan/type-safety/fix" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req tidinessManagerFixRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.ScenarioName != "test-scenario" {
			t.Errorf("unexpected scenario_name: %s", req.ScenarioName)
		}

		resp := tidinessManagerScanResponse{
			Scenario:                     "test-scenario",
			TSConfigFound:                true,
			TSConfigStrict:               true,
			TSConfigNoUnchecked:          true,
			TSConfigHasProtectiveComment: true,
			Violations:                   []tidinessManagerViolation{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	results, err := tidinessManagerFixViaURL(srv.Client(), srv.URL, context.Background(), []string{"test-scenario"}, []string{"TS_CONFIG_STRICT"}, false)
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
	if r.RuleID != "TS_CONFIG_STRICT" {
		t.Errorf("expected rule_id=TS_CONFIG_STRICT, got %s", r.RuleID)
	}
	if !r.Fixed {
		t.Error("expected fixed=true")
	}
}

func TestTidinessManagerFix_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	results, err := tidinessManagerFixViaURL(srv.Client(), srv.URL, context.Background(), []string{"test-scenario"}, []string{"TS_CONFIG_STRICT"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false for server error")
	}
	if results[0].Error == "" {
		t.Error("expected error message for server error")
	}
}

func TestTidinessManagerFix_Unavailable(t *testing.T) {
	client := &http.Client{Timeout: 1 * time.Second}
	results, err := tidinessManagerFixViaURL(client, "http://127.0.0.1:1", context.Background(), []string{"test-scenario"}, []string{"TS_CONFIG_STRICT"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false for unavailable server")
	}
}

func TestTidinessManagerFix_NonFixableRule(t *testing.T) {
	p := newTidinessManagerProvider().(*tidinessManagerProvider)
	_, err := p.Fix(context.Background(), []string{"test"}, []string{"ESLINT_SAFETY_RULES"}, false)
	if err == nil {
		t.Fatal("expected error for non-fixable rule")
	}
}

// tidinessManagerFixViaURL is a test helper that calls the fix endpoint directly
func tidinessManagerFixViaURL(client *http.Client, baseURL string, ctx context.Context, scenarioNames []string, ruleIDs []string, dryRun bool) ([]ExternalFixResult, error) {
	var results []ExternalFixResult
	for _, name := range scenarioNames {
		cleaned := strings.TrimSpace(name)
		if cleaned == "" {
			continue
		}

		if dryRun {
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Changes: []ExternalFixChange{
					{Type: "dry_run", Detail: "Would fix tsconfig.json"},
				},
			})
			continue
		}

		payload, _ := json.Marshal(tidinessManagerFixRequest{ScenarioName: cleaned})
		endpoint := fmt.Sprintf("%s/api/v1/scan/type-safety/fix", baseURL)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Error:        err.Error(),
			})
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Error:        fmt.Sprintf("responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(b))),
			})
			continue
		}

		var parsed tidinessManagerFixResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Error:        err.Error(),
			})
			continue
		}

		hasViolations := false
		for _, v := range parsed.Violations {
			if v.RuleID == "TS_CONFIG_STRICT" {
				hasViolations = true
				break
			}
		}

		results = append(results, ExternalFixResult{
			ScenarioName: cleaned,
			RuleID:       "TS_CONFIG_STRICT",
			Fixed:        !hasViolations,
			FilePath:     "ui/tsconfig.json",
			Changes: []ExternalFixChange{
				{Type: "config_update", Detail: "Updated tsconfig.json"},
			},
		})
	}

	return results, nil
}
