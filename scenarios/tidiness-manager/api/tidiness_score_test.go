package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestComputeScore tests the score calculation logic with various inputs
func TestComputeScore(t *testing.T) {
	calculator := &TidinessScoreCalculator{}

	tests := []struct {
		name           string
		issues         *issueCountsResult
		metrics        *fileMetricsResult
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:        "perfect score - no issues",
			issues:      &issueCountsResult{},
			metrics:     &fileMetricsResult{},
			expectedMin: 100,
			expectedMax: 100,
		},
		{
			name: "lint issues reduce score",
			issues: &issueCountsResult{
				Total: 10,
				Lint:  10,
			},
			metrics:     &fileMetricsResult{},
			expectedMin: 89,
			expectedMax: 91,
		},
		{
			name: "type errors reduce score more than lint",
			issues: &issueCountsResult{
				Total: 5,
				Type:  5,
			},
			metrics:     &fileMetricsResult{},
			expectedMin: 89,
			expectedMax: 91,
		},
		{
			name:   "long files reduce score",
			issues: &issueCountsResult{},
			metrics: &fileMetricsResult{
				LongFileCount: 5,
			},
			expectedMin: 84,
			expectedMax: 86,
		},
		{
			name:   "high complexity reduces score",
			issues: &issueCountsResult{},
			metrics: &fileMetricsResult{
				HighComplexityCount: 10,
			},
			expectedMin: 79,
			expectedMax: 81,
		},
		{
			name:   "tech debt markers reduce score",
			issues: &issueCountsResult{},
			metrics: &fileMetricsResult{
				TechDebtMarkers: 20,
			},
			expectedMin: 89,
			expectedMax: 91,
		},
		{
			name:   "missing tests reduce score",
			issues: &issueCountsResult{},
			metrics: &fileMetricsResult{
				MissingTestsCount: 10,
			},
			expectedMin: 89,
			expectedMax: 91,
		},
		{
			name:   "duplication reduces score",
			issues: &issueCountsResult{},
			metrics: &fileMetricsResult{
				DuplicationPct: 20.0,
			},
			expectedMin: 89,
			expectedMax: 91,
		},
		{
			name: "combined issues produce lower score",
			issues: &issueCountsResult{
				Total: 15,
				Lint:  10,
				Type:  5,
			},
			metrics: &fileMetricsResult{
				LongFileCount:       3,
				HighComplexityCount: 5,
				TechDebtMarkers:     10,
			},
			expectedMin: 50,
			expectedMax: 70,
		},
		{
			name: "very messy code bottoms out at 0",
			issues: &issueCountsResult{
				Total: 100,
				Lint:  50,
				Type:  50,
			},
			metrics: &fileMetricsResult{
				LongFileCount:       20,
				HighComplexityCount: 20,
				TechDebtMarkers:     100,
				MissingTestsCount:   50,
				DuplicationPct:      50.0,
			},
			expectedMin: 0,
			expectedMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculator.computeScore(tt.issues, tt.metrics)

			if score < tt.expectedMin || score > tt.expectedMax {
				t.Errorf("computeScore() = %v, expected between %v and %v",
					score, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// TestScoreWeightsAreReasonable verifies weight values make sense
func TestScoreWeightsAreReasonable(t *testing.T) {
	// Type errors should be more severe than lint issues
	if ScoreWeights.TypeError <= ScoreWeights.LintIssue {
		t.Error("TypeError weight should be greater than LintIssue weight")
	}

	// Long files should be a significant penalty
	if ScoreWeights.LongFile < 2.0 {
		t.Error("LongFile weight should be at least 2.0")
	}

	// All weights should be positive
	if ScoreWeights.LintIssue <= 0 ||
		ScoreWeights.TypeError <= 0 ||
		ScoreWeights.LongFile <= 0 ||
		ScoreWeights.HighComplexity <= 0 ||
		ScoreWeights.TechDebtMarker <= 0 {
		t.Error("All weights should be positive")
	}
}

// TestThresholdsAreReasonable verifies threshold values
func TestThresholdsAreReasonable(t *testing.T) {
	// Long file threshold should be in a reasonable range
	if Thresholds.LongFileLines < 200 || Thresholds.LongFileLines > 1000 {
		t.Errorf("LongFileLines threshold %d is outside reasonable range (200-1000)",
			Thresholds.LongFileLines)
	}

	// Complexity threshold should be reasonable
	if Thresholds.HighComplexity < 5 || Thresholds.HighComplexity > 20 {
		t.Errorf("HighComplexity threshold %d is outside reasonable range (5-20)",
			Thresholds.HighComplexity)
	}

	// Comment ratio should be a small percentage
	if Thresholds.LowCommentRatio < 0.01 || Thresholds.LowCommentRatio > 0.20 {
		t.Errorf("LowCommentRatio threshold %f is outside reasonable range (0.01-0.20)",
			Thresholds.LowCommentRatio)
	}
}

// TestTidinessScoreResponseFormat verifies the response structure
func TestTidinessScoreResponseFormat(t *testing.T) {
	response := TidinessScoreResponse{
		Scenario:   "test-scenario",
		Score:      85.5,
		Violations: 12,
		Breakdown: &TidinessBreakdown{
			LintIssues: 5,
			TypeIssues: 3,
			LongFiles:  2,
		},
		Metrics: &TidinessMetricsSummary{
			TotalFiles:    100,
			TotalLines:    10000,
			AvgFileLength: 100.0,
		},
	}

	// Verify JSON serialization works
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Verify expected fields are present
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	requiredFields := []string{"scenario", "score", "violations"}
	for _, field := range requiredFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("Response missing required field: %s", field)
		}
	}

	// Verify score is the expected value
	if parsed["score"].(float64) != 85.5 {
		t.Errorf("Score mismatch: got %v, want 85.5", parsed["score"])
	}
}

// TestHandleGetTidinessScoreWithDB tests the handler with a real database
func TestHandleGetTidinessScoreWithDB(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	testScenario := "test-tidiness-score"

	// Clean up before and after
	cleanupTestScenario := func() {
		srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", testScenario)
		srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", testScenario)
		srv.db.ExecContext(ctx, "DELETE FROM scan_history WHERE scenario = $1", testScenario)
	}
	cleanupTestScenario()
	defer cleanupTestScenario()

	// Insert some test data
	_, err := srv.db.ExecContext(ctx, `
		INSERT INTO issues (scenario, file_path, category, severity, title, status)
		VALUES
			($1, 'file1.go', 'lint', 'warning', 'Lint issue 1', 'open'),
			($1, 'file2.go', 'lint', 'warning', 'Lint issue 2', 'open'),
			($1, 'file3.go', 'type', 'error', 'Type error 1', 'open')
	`, testScenario)
	if err != nil {
		t.Fatalf("Failed to insert test issues: %v", err)
	}

	_, err = srv.db.ExecContext(ctx, `
		INSERT INTO file_metrics (scenario, file_path, line_count, todo_count, fixme_count, hack_count, has_test_file, comment_to_code_ratio)
		VALUES
			($1, 'file1.go', 100, 2, 1, 0, true, 0.10),
			($1, 'file2.go', 500, 0, 0, 0, false, 0.05),
			($1, 'file3.go', 200, 1, 0, 1, true, 0.08)
	`, testScenario)
	if err != nil {
		t.Fatalf("Failed to insert test file metrics: %v", err)
	}

	// Make the request
	req := httptest.NewRequest("GET", "/api/v1/scenarios/"+testScenario+"/tidiness", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": testScenario})
	rr := httptest.NewRecorder()

	srv.handleGetTidinessScore(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response TidinessScoreResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response fields
	if response.Scenario != testScenario {
		t.Errorf("Scenario mismatch: got %s, want %s", response.Scenario, testScenario)
	}

	if response.Score < 0 || response.Score > 100 {
		t.Errorf("Score %f is outside valid range (0-100)", response.Score)
	}

	// We inserted 3 issues and 1 long file (500 lines > 400 threshold)
	// Score should be less than 100
	if response.Score >= 100 {
		t.Error("Score should be less than 100 given the test data")
	}

	if response.Violations < 1 {
		t.Error("Expected at least 1 violation")
	}

	if response.Breakdown == nil {
		t.Error("Breakdown should not be nil")
	} else {
		if response.Breakdown.LintIssues != 2 {
			t.Errorf("Expected 2 lint issues, got %d", response.Breakdown.LintIssues)
		}
		if response.Breakdown.TypeIssues != 1 {
			t.Errorf("Expected 1 type issue, got %d", response.Breakdown.TypeIssues)
		}
	}

	if response.Metrics == nil {
		t.Error("Metrics should not be nil")
	} else {
		if response.Metrics.TotalFiles != 3 {
			t.Errorf("Expected 3 total files, got %d", response.Metrics.TotalFiles)
		}
	}
}

// TestHandleGetTidinessScoreLegacyRoute tests the /api/v1/scan/{scenario} route
func TestHandleGetTidinessScoreLegacyRoute(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	testScenario := "test-legacy-route"

	// Make the request using legacy route format
	req := httptest.NewRequest("GET", "/api/v1/scan/"+testScenario, nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": testScenario})
	rr := httptest.NewRecorder()

	srv.handleGetTidinessScore(rr, req)

	// Verify response - should work even with no data
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response TidinessScoreResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// With no issues, score should be 100
	if response.Score != 100 {
		t.Errorf("Expected score 100 for empty scenario, got %f", response.Score)
	}

	if response.Scenario != testScenario {
		t.Errorf("Scenario mismatch: got %s, want %s", response.Scenario, testScenario)
	}
}

// TestHandleGetTidinessScoreMissingScenario tests error handling for missing scenario
func TestHandleGetTidinessScoreMissingScenario(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	req := httptest.NewRequest("GET", "/api/v1/scenarios//tidiness", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": ""})
	rr := httptest.NewRecorder()

	srv.handleGetTidinessScore(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing scenario, got %d", rr.Code)
	}
}

// TestScoreIsIdempotent verifies that calculating the score multiple times gives the same result
func TestScoreIsIdempotent(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	testScenario := "test-idempotent"

	// Clean up
	srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", testScenario)
	srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", testScenario)
	defer func() {
		srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", testScenario)
		srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", testScenario)
	}()

	// Insert test data
	srv.db.ExecContext(ctx, `
		INSERT INTO issues (scenario, file_path, category, severity, title, status)
		VALUES ($1, 'test.go', 'lint', 'warning', 'Test', 'open')
	`, testScenario)

	calculator := NewTidinessScoreCalculator(srv.db)

	// Calculate score multiple times
	result1, err := calculator.Calculate(ctx, testScenario)
	if err != nil {
		t.Fatalf("First calculation failed: %v", err)
	}

	result2, err := calculator.Calculate(ctx, testScenario)
	if err != nil {
		t.Fatalf("Second calculation failed: %v", err)
	}

	result3, err := calculator.Calculate(ctx, testScenario)
	if err != nil {
		t.Fatalf("Third calculation failed: %v", err)
	}

	// All results should be identical
	if result1.Score != result2.Score || result2.Score != result3.Score {
		t.Errorf("Score is not idempotent: %f, %f, %f",
			result1.Score, result2.Score, result3.Score)
	}

	if result1.Violations != result2.Violations || result2.Violations != result3.Violations {
		t.Errorf("Violations count is not idempotent: %d, %d, %d",
			result1.Violations, result2.Violations, result3.Violations)
	}
}

// TestScoreReflectsIssueResolution verifies score improves when issues are resolved
func TestScoreReflectsIssueResolution(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	testScenario := "test-resolution"

	// Clean up
	srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", testScenario)
	defer srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", testScenario)

	// Insert multiple issues
	srv.db.ExecContext(ctx, `
		INSERT INTO issues (scenario, file_path, category, severity, title, status)
		VALUES
			($1, 'file1.go', 'lint', 'warning', 'Issue 1', 'open'),
			($1, 'file2.go', 'lint', 'warning', 'Issue 2', 'open'),
			($1, 'file3.go', 'lint', 'warning', 'Issue 3', 'open'),
			($1, 'file4.go', 'lint', 'warning', 'Issue 4', 'open'),
			($1, 'file5.go', 'lint', 'warning', 'Issue 5', 'open')
	`, testScenario)

	calculator := NewTidinessScoreCalculator(srv.db)

	// Calculate initial score
	before, err := calculator.Calculate(ctx, testScenario)
	if err != nil {
		t.Fatalf("Initial calculation failed: %v", err)
	}

	// Resolve some issues
	srv.db.ExecContext(ctx, `
		UPDATE issues SET status = 'resolved'
		WHERE scenario = $1 AND file_path IN ('file1.go', 'file2.go', 'file3.go')
	`, testScenario)

	// Calculate new score
	after, err := calculator.Calculate(ctx, testScenario)
	if err != nil {
		t.Fatalf("Post-resolution calculation failed: %v", err)
	}

	// Score should improve after resolving issues
	if after.Score <= before.Score {
		t.Errorf("Score should improve after resolving issues: before=%f, after=%f",
			before.Score, after.Score)
	}

	// Violations should decrease
	if after.Violations >= before.Violations {
		t.Errorf("Violations should decrease after resolving issues: before=%d, after=%d",
			before.Violations, after.Violations)
	}
}
