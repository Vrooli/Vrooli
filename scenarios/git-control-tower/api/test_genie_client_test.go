package main

import (
	"context"
	"net/http"
	"testing"

	"git-control-tower/internal/testutil/httpx"
	"github.com/vrooli/api-core/discovery"
)

func TestTestGenieClient_ExecuteSuite(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/executions": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodPost)
			req := httpx.DecodeJSON[TestExecutionRequest](t, r)
			if req.ScenarioName != "git-control-tower" {
				t.Errorf("unexpected scenario name: %s", req.ScenarioName)
			}
			httpx.WriteJSON(t, w, http.StatusOK, TestExecutionResult{
				ExecutionID:  "exec-001",
				ScenarioName: req.ScenarioName,
				Success:      true,
				StartedAt:    "2026-03-10T12:00:00Z",
				CompletedAt:  "2026-03-10T12:05:00Z",
				Phases: []TestPhaseResult{
					{Name: "build", Status: "passed", DurationSeconds: 30},
					{Name: "unit", Status: "passed", DurationSeconds: 60},
				},
				PhaseSummary: TestPhaseSummary{Total: 2, Passed: 2, DurationSeconds: 90},
			})
		},
	})

	client := &TestGenieClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "test-genie",
		},
	}

	result, err := client.ExecuteSuite(context.Background(), TestExecutionRequest{
		ScenarioName: "git-control-tower",
		Preset:       "comprehensive",
	})
	if err != nil {
		t.Fatalf("ExecuteSuite returned error: %v", err)
	}
	if result.ExecutionID != "exec-001" {
		t.Errorf("expected execution ID exec-001, got %s", result.ExecutionID)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if len(result.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(result.Phases))
	}
}

func TestTestGenieClient_ExecuteSuite_ServerError(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/executions": func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(t, w, http.StatusInternalServerError, map[string]string{"error": "internal failure"})
		},
	})

	client := &TestGenieClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "test-genie",
		},
	}

	_, err := client.ExecuteSuite(context.Background(), TestExecutionRequest{
		ScenarioName: "git-control-tower",
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestTestGenieClient_ListExecutions(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/executions": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			scenario := r.URL.Query().Get("scenario")
			if scenario != "git-control-tower" {
				t.Errorf("expected scenario=git-control-tower, got %s", scenario)
			}
			httpx.WriteJSON(t, w, http.StatusOK, TestExecutionListResponse{
				Items: []TestExecutionResult{
					{ExecutionID: "exec-001", ScenarioName: scenario, Success: true},
					{ExecutionID: "exec-002", ScenarioName: scenario, Success: false},
				},
				Count: 2,
			})
		},
	})

	client := &TestGenieClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "test-genie",
		},
	}

	result, err := client.ListExecutions(context.Background(), "git-control-tower", 10)
	if err != nil {
		t.Fatalf("ListExecutions returned error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestTestGenieClient_GetExecution(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/executions/exec-001": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			httpx.WriteJSON(t, w, http.StatusOK, TestExecutionResult{
				ExecutionID:  "exec-001",
				ScenarioName: "git-control-tower",
				Success:      true,
				StartedAt:    "2026-03-10T12:00:00Z",
				PhaseSummary: TestPhaseSummary{Total: 3, Passed: 3, DurationSeconds: 120},
			})
		},
	})

	client := &TestGenieClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "test-genie",
		},
	}

	result, err := client.GetExecution(context.Background(), "exec-001")
	if err != nil {
		t.Fatalf("GetExecution returned error: %v", err)
	}
	if result.ExecutionID != "exec-001" {
		t.Errorf("expected execution ID exec-001, got %s", result.ExecutionID)
	}
	if result.PhaseSummary.Total != 3 {
		t.Errorf("expected 3 total phases, got %d", result.PhaseSummary.Total)
	}
}
