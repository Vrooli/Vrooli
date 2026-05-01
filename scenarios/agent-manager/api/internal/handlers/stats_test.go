// Package handlers provides HTTP handlers for the agent-manager API.
// This file contains tests for stats API endpoints.
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil/assertx"
	"agent-manager/internal/testutil/fixtures"
	"agent-manager/internal/testutil/mocks"

	"github.com/gorilla/mux"
)

// =============================================================================
// STATS HANDLER TEST SETUP
// =============================================================================

// setupStatsTestHandler creates a stats handler with seeded test data.
func setupStatsTestHandler() (*StatsHandler, *mux.Router) {
	statsRepo := mocks.NewFakeStatsRepository()
	snapshot := fixtures.NewStatsSnapshot()
	statsRepo.StatusCounts = snapshot.StatusCounts
	statsRepo.SuccessRate = snapshot.SuccessRate
	statsRepo.DurationStats = snapshot.DurationStats
	statsRepo.CostStats = snapshot.CostStats
	statsRepo.RunnerBreakdown = snapshot.RunnerBreakdown
	statsRepo.TimeSeries = snapshot.TimeSeries

	// Create stats orchestrator
	statsSvc := orchestration.NewStatsOrchestrator(statsRepo)

	// Create handler
	handler := NewStatsHandler(statsSvc)

	// Create router and register routes
	r := mux.NewRouter()
	handler.RegisterRoutes(r)

	return handler, r
}

// =============================================================================
// STATS SUMMARY TESTS
// =============================================================================

func TestGetSummary_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response SummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Summary == nil {
		t.Fatal("expected summary, got nil")
	}

	if response.Summary.StatusCounts == nil {
		t.Error("expected status counts in summary")
	}

	if response.Summary.StatusCounts.Total != 27 {
		t.Errorf("expected total runs 27, got %d", response.Summary.StatusCounts.Total)
	}

	if response.Summary.SuccessRate != 0.85 {
		t.Errorf("expected success rate 0.85, got %f", response.Summary.SuccessRate)
	}
}

func TestGetSummary_WithTimeRange(t *testing.T) {
	_, router := setupStatsTestHandler()

	start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?start="+start+"&end="+end, nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)
}

func TestGetSummary_InvalidTimestamp(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?start=invalid-date", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusBadRequest)
}

// =============================================================================
// STATUS DISTRIBUTION TESTS
// =============================================================================

func TestGetStatusDistribution_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/status-distribution?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response StatusDistributionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.StatusCounts == nil {
		t.Fatal("expected status counts, got nil")
	}

	if response.StatusCounts.Complete != 17 {
		t.Errorf("expected 17 complete runs, got %d", response.StatusCounts.Complete)
	}

	if response.StatusCounts.Failed != 3 {
		t.Errorf("expected 3 failed runs, got %d", response.StatusCounts.Failed)
	}
}

// =============================================================================
// SUCCESS RATE TESTS
// =============================================================================

func TestGetSuccessRate_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/success-rate?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response SuccessRateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.SuccessRate != 0.85 {
		t.Errorf("expected success rate 0.85, got %f", response.SuccessRate)
	}
}

// =============================================================================
// DURATION STATS TESTS
// =============================================================================

func TestGetDurationStats_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/duration?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response DurationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Duration == nil {
		t.Fatal("expected duration stats, got nil")
	}

	if response.Duration.AvgMs != 45000 {
		t.Errorf("expected avg duration 45000ms, got %d", response.Duration.AvgMs)
	}

	if response.Duration.P95Ms != 120000 {
		t.Errorf("expected p95 duration 120000ms, got %d", response.Duration.P95Ms)
	}
}

// =============================================================================
// COST STATS TESTS
// =============================================================================

func TestGetCostStats_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/cost?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response CostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Cost == nil {
		t.Fatal("expected cost stats, got nil")
	}

	if response.Cost.TotalCostUSD != 12.50 {
		t.Errorf("expected total cost $12.50, got $%f", response.Cost.TotalCostUSD)
	}

	if response.Cost.TotalTokens != 650000 {
		t.Errorf("expected total tokens 650000, got %d", response.Cost.TotalTokens)
	}
}

// =============================================================================
// RUNNER BREAKDOWN TESTS
// =============================================================================

func TestGetRunnerBreakdown_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/runners?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response RunnerBreakdownResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Runners) != 2 {
		t.Errorf("expected 2 runners, got %d", len(response.Runners))
	}
}

// =============================================================================
// TIME SERIES TESTS
// =============================================================================

func TestGetTimeSeries_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/time-series?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response TimeSeriesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Buckets) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(response.Buckets))
	}

	if response.BucketDuration == "" {
		t.Error("expected bucket duration to be set")
	}
}

func TestGetTimeSeries_WithCustomBucket(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/time-series?preset=24h&bucket=1h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	var response TimeSeriesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.BucketDuration != "1h0m0s" {
		t.Errorf("expected bucket duration '1h0m0s', got '%s'", response.BucketDuration)
	}
}

func TestGetTimeSeries_InvalidBucket(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/time-series?preset=24h&bucket=invalid", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusBadRequest)
}

// =============================================================================
// FILTER TESTS
// =============================================================================

func TestStatsEndpoints_WithFilters(t *testing.T) {
	_, router := setupStatsTestHandler()

	// Test runner_type filter
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?preset=24h&runner_type=claude-code", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	// Test model filter
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?preset=24h&model=claude-3-opus", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)

	// Test tag_prefix filter
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?preset=24h&tag_prefix=ecosystem-", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assertx.HTTPStatus(t, rr, http.StatusOK)
}

// =============================================================================
// PRESET TESTS
// =============================================================================

func TestTimePresets(t *testing.T) {
	_, router := setupStatsTestHandler()

	presets := []string{"6h", "12h", "24h", "7d", "30d"}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?preset="+preset, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assertx.HTTPStatus(t, rr, http.StatusOK)
		})
	}
}
