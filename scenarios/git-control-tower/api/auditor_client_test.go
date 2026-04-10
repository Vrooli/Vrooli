package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func TestAuditorClient_StartCheck(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/standards/check/my-scenario", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(AuditorCheckJobResponse{
			JobID: "standards-abc123",
			Status: AuditorJobStatus{
				ID:       "standards-abc123",
				Scenario: "my-scenario",
				ScanType: "full",
				Status:   "running",
				Message:  "Standards scan started",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AuditorClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "scenario-auditor",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/standards/check/my-scenario", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal failure"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AuditorClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "scenario-auditor",
		},
	}

	_, err := client.StartCheck(context.Background(), "my-scenario", "full")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestAuditorClient_GetJobStatus(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/standards/check/jobs/standards-abc123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuditorJobStatus{
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
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AuditorClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "scenario-auditor",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuditorRulesListResponse{
			Rules: map[string]AuditorRule{
				"MAKEFILE_STRUCTURE": {ID: "MAKEFILE_STRUCTURE", Name: "Makefile Structure", Category: "makefile", Severity: "high", Enabled: true},
				"GO_CLI_WORKSPACE":   {ID: "GO_CLI_WORKSPACE", Name: "Go CLI Workspace", Category: "go", Severity: "high", Enabled: true},
			},
			Count: 2,
			Total: 2,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AuditorClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "scenario-auditor",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/standards/fix", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuditorFixResponse{
			Results: []AuditorFixResult{
				{ScenarioName: "my-scenario", RuleID: "MAKEFILE_STRUCTURE", Fixed: true, FilePath: "Makefile", Changes: []AuditorFixChange{{Type: "line", Detail: "Added start target"}}},
			},
			Count: 1,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AuditorClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "scenario-auditor",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/standards/violations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		scenario := r.URL.Query().Get("scenario")
		if scenario != "my-scenario" {
			t.Errorf("expected scenario=my-scenario, got %s", scenario)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuditorViolationsResponse{
			Violations: []AuditorViolation{
				{ID: "v1", ScenarioName: "my-scenario", Type: "MAKEFILE_STRUCTURE", Severity: "high", Title: "Missing target"},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AuditorClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "scenario-auditor",
		},
	}

	result, err := client.GetViolations(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("GetViolations returned error: %v", err)
	}
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result.Violations))
	}
}
