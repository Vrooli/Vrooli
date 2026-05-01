package main

import (
	"context"
	"net/http"
	"testing"

	"git-control-tower/internal/testutil/httpx"
	"github.com/vrooli/api-core/discovery"
)

func TestAuditorClient_StartCheck(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/standards/check/my-scenario": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodPost)
			httpx.WriteJSON(t, w, http.StatusAccepted, AuditorCheckJobResponse{
				JobID: "standards-abc123",
				Status: AuditorJobStatus{
					ID:       "standards-abc123",
					Scenario: "my-scenario",
					ScanType: "full",
					Status:   "running",
					Message:  "Standards scan started",
				},
			})
		},
	})

	client := newTestAuditorClient(server.URL)

	result, err := client.StartCheck(context.Background(), "my-scenario", "full")
	if err != nil {
		t.Fatalf("StartCheck returned error: %v", err)
	}
	if result.JobID != "standards-abc123" {
		t.Errorf("expected job_id standards-abc123, got %s", result.JobID)
	}
	if result.Status.Status != "running" {
		t.Errorf("expected status running, got %s", result.Status.Status)
	}
}

func TestAuditorClient_StartCheck_ServerError(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/standards/check/my-scenario": func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(t, w, http.StatusInternalServerError, map[string]string{"error": "internal failure"})
		},
	})

	client := newTestAuditorClient(server.URL)

	_, err := client.StartCheck(context.Background(), "my-scenario", "full")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestAuditorClient_GetJobStatus(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/standards/check/jobs/standards-abc123": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			httpx.WriteJSON(t, w, http.StatusOK, AuditorJobStatus{
				ID:       "standards-abc123",
				Scenario: "my-scenario",
				Status:   "completed",
				Result: &AuditorCheckResult{
					CheckID:      "standards-abc123",
					Status:       "completed",
					FilesScanned: 42,
					Violations: []AuditorViolation{
						{ID: "v1", Type: "MAKEFILE_STRUCTURE", Severity: "high", Title: "Missing target", Source: "scenario-stack-governor"},
					},
				},
			})
		},
	})

	client := newTestAuditorClient(server.URL)

	result, err := client.GetJobStatus(context.Background(), "standards-abc123")
	if err != nil {
		t.Fatalf("GetJobStatus returned error: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("expected status completed, got %s", result.Status)
	}
	if result.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Result.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result.Result.Violations))
	}
}

func TestAuditorClient_ListRules(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/rules": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			httpx.WriteJSON(t, w, http.StatusOK, AuditorRulesListResponse{
				Rules: map[string]AuditorRule{
					"MAKEFILE_STRUCTURE": {ID: "MAKEFILE_STRUCTURE", Name: "Makefile Structure", Category: "makefile", Severity: "high", Enabled: true},
					"GO_CLI_WORKSPACE":   {ID: "GO_CLI_WORKSPACE", Name: "Go CLI Workspace", Category: "go", Severity: "high", Enabled: true},
				},
				Count: 2,
				Total: 2,
			})
		},
	})

	client := newTestAuditorClient(server.URL)

	result, err := client.ListRules(context.Background())
	if err != nil {
		t.Fatalf("ListRules returned error: %v", err)
	}
	if len(result.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(result.Rules))
	}
}

func TestAuditorClient_ApplyFix(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/standards/fix": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodPost)
			httpx.WriteJSON(t, w, http.StatusOK, AuditorFixResponse{
				Results: []AuditorFixResult{
					{ScenarioName: "my-scenario", RuleID: "MAKEFILE_STRUCTURE", Fixed: true, FilePath: "Makefile", Changes: []AuditorFixChange{{Type: "line", Detail: "Added start target"}}},
				},
				Count: 1,
			})
		},
	})

	client := newTestAuditorClient(server.URL)

	result, err := client.ApplyFix(context.Background(), AuditorFixRequest{
		ScenarioNames: []string{"my-scenario"},
		RuleIDs:       []string{"MAKEFILE_STRUCTURE"},
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("ApplyFix returned error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 fix result, got %d", len(result.Results))
	}
	if !result.Results[0].Fixed {
		t.Error("expected fix to be applied")
	}
}

func TestAuditorClient_GetViolations(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/standards/violations": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			scenario := r.URL.Query().Get("scenario")
			if scenario != "my-scenario" {
				t.Errorf("expected scenario=my-scenario, got %s", scenario)
			}
			httpx.WriteJSON(t, w, http.StatusOK, AuditorViolationsResponse{
				Violations: []AuditorViolation{
					{ID: "v1", ScenarioName: "my-scenario", Type: "MAKEFILE_STRUCTURE", Severity: "high", Title: "Missing target"},
				},
			})
		},
	})

	client := newTestAuditorClient(server.URL)

	result, err := client.GetViolations(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("GetViolations returned error: %v", err)
	}
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result.Violations))
	}
}

func newTestAuditorClient(serverURL string) *AuditorClient {
	return &AuditorClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(serverURL),
			serviceName: "scenario-auditor",
		},
	}
}
