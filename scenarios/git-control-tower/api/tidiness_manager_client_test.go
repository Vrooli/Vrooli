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

func TestTidinessManagerClient_GetTidinessScore(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/scenarios/my-scenario/tidiness", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TidinessScoreResponse{
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
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &TidinessManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "tidiness-manager",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/scenarios/my-scenario/tidiness", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal failure"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &TidinessManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "tidiness-manager",
		},
	}

	_, err := client.GetTidinessScore(context.Background(), "my-scenario")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestTidinessManagerClient_GetIssues(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agent/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		scenario := r.URL.Query().Get("scenario")
		if scenario != "my-scenario" {
			t.Errorf("expected scenario=my-scenario, got %s", scenario)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]TidinessIssue{
			{ID: 1, Scenario: scenario, FilePath: "api/main.go", Category: "lint", Severity: "medium", Title: "unused var", Status: "open"},
			{ID: 2, Scenario: scenario, FilePath: "api/main.go", Category: "type", Severity: "high", Title: "missing return", Status: "open"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &TidinessManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "tidiness-manager",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agent/staleness", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		lastScan := "2026-03-10T12:00:00Z"
		_ = json.NewEncoder(w).Encode(TidinessStalenessInfo{
			LastScanAt:    &lastScan,
			IsStale:       true,
			ModifiedFiles: 3,
			StaleReason:   "3 files modified since last scan",
			RescanCommand: "tidiness-manager scan my-scenario",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &TidinessManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "tidiness-manager",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/scan/light", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req TidinessLightScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.ScenarioPath == "" {
			t.Error("expected non-empty scenario_path")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TidinessLightScanResult{
			Scenario:        "my-scenario",
			TotalFiles:      28,
			TotalLines:      4200,
			LintIssuesCount: 5,
			TypeIssuesCount: 2,
			LongFilesCount:  3,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &TidinessManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "tidiness-manager",
		},
	}

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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agent/scenarios/my-scenario", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TidinessScenarioDetail{
			Scenario:    "my-scenario",
			LightIssues: 5,
			AIIssues:    2,
			LongFiles:   3,
			Files: []TidinessScenarioFileInfo{
				{Path: "api/main.go", Lines: 200, TotalIssues: 3, VisitCount: 1},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &TidinessManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "tidiness-manager",
		},
	}

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
