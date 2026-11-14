package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/vrooli/api-core/discovery"
	httpx "github.com/vrooli/api-core/servertest"
)

func TestTidinessManagerClient_GetTidinessScore(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/scenarios/my-scenario/tidiness": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			httpx.WriteJSON(t, w, http.StatusOK, TidinessScoreResponse{
				Scenario:   "my-scenario",
				Score:      72.5,
				Violations: 14,
				Breakdown: &TidinessBreakdown{
					LintIssues: 5, TypeIssues: 2, LongFiles: 3,
					ComplexFunctions: 1, TechDebtMarkers: 2, DuplicationIssues: 1,
				},
				Metrics: &TidinessMetricsSummary{
					TotalFiles: 28, TotalLines: 4200, AvgFileLength: 150,
					MaxComplexity: 12, AvgComplexity: 3.2, DuplicationPct: 4.5,
				},
			})
		},
	})

	client := newTestTidinessManagerClient(server.URL)

	result, err := client.GetTidinessScore(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("GetTidinessScore returned error: %v", err)
	}
	if result.Scenario != "my-scenario" {
		t.Errorf("expected scenario my-scenario, got %s", result.Scenario)
	}
	if result.Score != 72.5 {
		t.Errorf("expected score 72.5, got %f", result.Score)
	}
	if result.Violations != 14 {
		t.Errorf("expected 14 violations, got %d", result.Violations)
	}
}

func TestTidinessManagerClient_GetTidinessScore_ServerError(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/scenarios/my-scenario/tidiness": func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(t, w, http.StatusInternalServerError, map[string]string{"error": "internal failure"})
		},
	})

	client := newTestTidinessManagerClient(server.URL)

	_, err := client.GetTidinessScore(context.Background(), "my-scenario")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestTidinessManagerClient_GetIssues(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/agent/issues": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			scenario := r.URL.Query().Get("scenario")
			if scenario != "my-scenario" {
				t.Errorf("expected scenario=my-scenario, got %s", scenario)
			}
			httpx.WriteJSON(t, w, http.StatusOK, []TidinessIssue{
				{ID: 1, Scenario: scenario, FilePath: "api/main.go", Category: "lint", Severity: "medium", Title: "unused var", Status: "open"},
				{ID: 2, Scenario: scenario, FilePath: "api/main.go", Category: "type", Severity: "high", Title: "missing return", Status: "open"},
			})
		},
	})

	client := newTestTidinessManagerClient(server.URL)

	result, err := client.GetIssues(context.Background(), "my-scenario", "", "", "", 100)
	if err != nil {
		t.Fatalf("GetIssues returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result))
	}
}

func TestTidinessManagerClient_GetStaleness(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/agent/staleness": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			lastScan := "2026-03-10T12:00:00Z"
			httpx.WriteJSON(t, w, http.StatusOK, TidinessStalenessInfo{
				LastScanAt:    &lastScan,
				IsStale:       true,
				ModifiedFiles: 3,
				StaleReason:   "3 files modified since last scan",
				RescanCommand: "tidiness-manager scan my-scenario",
			})
		},
	})

	client := newTestTidinessManagerClient(server.URL)

	result, err := client.GetStaleness(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("GetStaleness returned error: %v", err)
	}
	if !result.IsStale {
		t.Error("expected IsStale=true")
	}
	if result.ModifiedFiles != 3 {
		t.Errorf("expected 3 modified files, got %d", result.ModifiedFiles)
	}
}

func TestTidinessManagerClient_TriggerLightScan(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/scan/light": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodPost)
			req := httpx.DecodeJSON[TidinessLightScanRequest](t, r)
			if req.ScenarioPath == "" {
				t.Error("expected non-empty scenario_path")
			}
			httpx.WriteJSON(t, w, http.StatusOK, TidinessLightScanResult{
				Scenario:        "my-scenario",
				TotalFiles:      28,
				TotalLines:      4200,
				LintIssuesCount: 5,
				TypeIssuesCount: 2,
				LongFilesCount:  3,
			})
		},
	})

	client := newTestTidinessManagerClient(server.URL)

	result, err := client.TriggerLightScan(context.Background(), TidinessLightScanRequest{
		ScenarioPath: "/scenarios/my-scenario",
		TimeoutSec:   60,
	})
	if err != nil {
		t.Fatalf("TriggerLightScan returned error: %v", err)
	}
	if result.TotalFiles != 28 {
		t.Errorf("expected 28 total files, got %d", result.TotalFiles)
	}
}

func TestTidinessManagerClient_GetScenarioDetail(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/agent/scenarios/my-scenario": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodGet)
			httpx.WriteJSON(t, w, http.StatusOK, TidinessScenarioDetail{
				Scenario:    "my-scenario",
				LightIssues: 5,
				AIIssues:    2,
				LongFiles:   3,
				Files: []TidinessScenarioFileInfo{
					{Path: "api/main.go", Lines: 200, TotalIssues: 3, VisitCount: 1},
				},
			})
		},
	})

	client := newTestTidinessManagerClient(server.URL)

	result, err := client.GetScenarioDetail(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("GetScenarioDetail returned error: %v", err)
	}
	if result.Scenario != "my-scenario" {
		t.Errorf("expected scenario my-scenario, got %s", result.Scenario)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}
}

func newTestTidinessManagerClient(serverURL string) *TidinessManagerClient {
	return &TidinessManagerClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(serverURL),
			serviceName: "tidiness-manager",
		},
	}
}
