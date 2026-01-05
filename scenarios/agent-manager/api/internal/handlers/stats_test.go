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
	"agent-manager/internal/repository"

	"github.com/gorilla/mux"
)

// =============================================================================
// STATS HANDLER TEST SETUP
// =============================================================================

// setupStatsTestHandler creates a stats handler with seeded test data.
func setupStatsTestHandler() (*StatsHandler, *mux.Router) {
	// Create in-memory stats repository with seeded data
	statsRepo := repository.NewMemoryStatsRepository()

	// Seed test data
	successRate := 0.85
	statsRepo.SeededStatusCounts = &repository.RunStatusCounts{
		Pending:     2,
		Running:     3,
		Complete:    17,
		Failed:      3,
		Cancelled:   1,
		NeedsReview: 1,
		Total:       27,
	}
	statsRepo.SeededSuccessRate = &successRate
	statsRepo.SeededDurationStats = &repository.DurationStats{
		AvgMs: 45000,
		P50Ms: 30000,
		P95Ms: 120000,
		P99Ms: 180000,
		MinMs: 5000,
		MaxMs: 200000,
		Count: 20,
	}
	statsRepo.SeededCostStats = &repository.CostStats{
		TotalCostUSD:    12.50,
		AvgCostUSD:      0.625,
		InputTokens:     500000,
		OutputTokens:    100000,
		CacheReadTokens: 50000,
		TotalTokens:     650000,
	}
	statsRepo.SeededRunnerBreakdown = []*repository.RunnerBreakdown{
		{
			RunnerType:    "claude-code",
			RunCount:      15,
			SuccessCount:  12,
			FailedCount:   2,
			TotalCostUSD:  10.00,
			AvgDurationMs: 40000,
		},
		{
			RunnerType:    "codex",
			RunCount:      5,
			SuccessCount:  4,
			FailedCount:   1,
			TotalCostUSD:  2.50,
			AvgDurationMs: 55000,
		},
	}
	statsRepo.SeededTimeSeries = []*repository.TimeSeriesBucket{
		{
			Timestamp:     time.Now().Add(-2 * time.Hour).Truncate(time.Hour),
			RunsStarted:   5,
			RunsCompleted: 4,
			RunsFailed:    1,
			TotalCostUSD:  3.00,
			AvgDurationMs: 35000,
		},
		{
			Timestamp:     time.Now().Add(-1 * time.Hour).Truncate(time.Hour),
			RunsStarted:   8,
			RunsCompleted: 7,
			RunsFailed:    0,
			TotalCostUSD:  5.50,
			AvgDurationMs: 42000,
		},
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestGetSummary_InvalidTimestamp(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?start=invalid-date", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// =============================================================================
// STATUS DISTRIBUTION TESTS
// =============================================================================

func TestGetStatusDistribution_Success(t *testing.T) {
	_, router := setupStatsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/status-distribution?preset=24h", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

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

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
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

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d with runner filter, got %d", http.StatusOK, rr.Code)
	}

	// Test model filter
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?preset=24h&model=claude-3-opus", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d with model filter, got %d", http.StatusOK, rr.Code)
	}

	// Test tag_prefix filter
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?preset=24h&tag_prefix=ecosystem-", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d with tag_prefix filter, got %d", http.StatusOK, rr.Code)
	}
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

			if rr.Code != http.StatusOK {
				t.Errorf("expected status %d for preset %s, got %d", http.StatusOK, preset, rr.Code)
			}
		})
	}
}
