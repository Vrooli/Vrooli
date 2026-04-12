package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// ---------------------------------------------------------------------------
// Helpers for building test inputs
// ---------------------------------------------------------------------------

// makeIssues returns an issueCountsResult with total auto-calculated.
func makeIssues(lint, typ, typeSafety, ai int) *issueCountsResult {
	return &issueCountsResult{
		Total:      lint + typ + typeSafety + ai,
		Lint:       lint,
		Type:       typ,
		TypeSafety: typeSafety,
		AI:         ai,
	}
}

// makeMetrics returns a fileMetricsResult with sensible defaults.
// Override fields after creation as needed.
func makeMetrics(totalFiles, totalCodeLines, testedFiles, testableFiles int) *fileMetricsResult {
	return &fileMetricsResult{
		TotalFiles:           totalFiles,
		TotalLines:           totalCodeLines, // approximate
		TotalCodeLines:       totalCodeLines,
		AvgFileLength:        safeDiv(float64(totalCodeLines), float64(totalFiles)),
		ShortFileCount:       totalFiles, // default: all files under threshold
		FilesWithComplexity:  totalFiles,
		AdequateCommentFiles: totalFiles, // default: all files have adequate comments
		TestedFiles:          testedFiles,
		TestableFiles:        testableFiles,
	}
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// ---------------------------------------------------------------------------
// Unit tests: computeScore
// ---------------------------------------------------------------------------

func TestComputeScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		issues      *issueCountsResult
		metrics     *fileMetricsResult
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "empty codebase scores 100",
			issues:      &issueCountsResult{},
			metrics:     &fileMetricsResult{},
			expectedMin: 100,
			expectedMax: 100,
		},
		{
			name:   "pristine project (50 files, 10 KLOC, all clean)",
			issues: makeIssues(0, 0, 0, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(50, 10000, 50, 50)
				return m
			}(),
			expectedMin: 95,
			expectedMax: 100,
		},
		{
			name:   "well-maintained large project",
			issues: makeIssues(20, 5, 0, 3),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(500, 100000, 400, 500)
				m.HighComplexityCount = 25   // 5% of files
				m.LongFileCount = 15         // 3% of files
				m.ShortFileCount = 485       // 97%
				m.TechDebtMarkers = 150      // 1.5/KLOC
				m.AdequateCommentFiles = 425 // 85%
				m.AvgDuplicationPct = 3.0
				return m
			}(),
			expectedMin: 70,
			expectedMax: 85,
		},
		{
			name:   "average project",
			issues: makeIssues(60, 10, 5, 5),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(200, 40000, 100, 200)
				m.HighComplexityCount = 30   // 15%
				m.LongFileCount = 40         // 20%
				m.ShortFileCount = 160       // 80%
				m.TechDebtMarkers = 200      // 5/KLOC
				m.AdequateCommentFiles = 120 // 60%
				m.AvgDuplicationPct = 8.0
				m.TypeSafetyMarkers = 50 // 1.25/KLOC
				return m
			}(),
			expectedMin: 45,
			expectedMax: 65,
		},
		{
			name:   "messy project",
			issues: makeIssues(40, 15, 10, 10),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 20, 100)
				m.HighComplexityCount = 30  // 30%
				m.LongFileCount = 35        // 35%
				m.ShortFileCount = 65       // 65%
				m.TechDebtMarkers = 180     // 9/KLOC
				m.AdequateCommentFiles = 30 // 30%
				m.AvgDuplicationPct = 15.0
				m.TypeSafetyMarkers = 80 // 4/KLOC
				return m
			}(),
			expectedMin: 20,
			expectedMax: 40,
		},
		{
			name:   "terrible project scores near zero",
			issues: makeIssues(50, 30, 20, 20),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(50, 10000, 0, 50)
				m.HighComplexityCount = 40 // 80%
				m.LongFileCount = 40       // 80%
				m.ShortFileCount = 10      // 20%
				m.TechDebtMarkers = 200    // 20/KLOC
				m.AdequateCommentFiles = 5 // 10%
				m.AvgDuplicationPct = 25.0
				m.TypeSafetyMarkers = 100 // 10/KLOC
				return m
			}(),
			expectedMin: 0,
			expectedMax: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score, breakdown := computeScore(tt.issues, tt.metrics, DefaultDimensions)

			if score < tt.expectedMin || score > tt.expectedMax {
				t.Errorf("score = %.2f, want [%.0f, %.0f]\nbreakdown: lint=%.1f type=%.1f complexity=%.1f length=%.1f test=%.1f debt=%.1f comments=%.1f dup=%.1f",
					score, tt.expectedMin, tt.expectedMax,
					breakdown.LintScore, breakdown.TypeSafetyScore,
					breakdown.ComplexityScore, breakdown.FileLengthScore,
					breakdown.TestCoverageScore, breakdown.TechDebtScore,
					breakdown.CommentsScore, breakdown.DuplicationScore)
			}
		})
	}
}

// TestScoreScaleInvariance verifies that the same issue density at different
// codebase sizes produces the same score (±2 points).
func TestScoreScaleInvariance(t *testing.T) {
	t.Parallel()

	// Small: 10 files, 2 KLOC, 20% tested, 3 lint issues, 10% high complexity
	small := makeMetrics(10, 2000, 2, 10)
	small.HighComplexityCount = 1
	small.LongFileCount = 1
	small.ShortFileCount = 9
	small.TechDebtMarkers = 10 // 5/KLOC
	small.AdequateCommentFiles = 7
	small.AvgDuplicationPct = 5.0
	smallIssues := makeIssues(3, 1, 0, 0)

	// Large: 1000 files, 200 KLOC, same densities
	large := makeMetrics(1000, 200000, 200, 1000)
	large.HighComplexityCount = 100
	large.LongFileCount = 100
	large.ShortFileCount = 900
	large.TechDebtMarkers = 1000 // 5/KLOC
	large.AdequateCommentFiles = 700
	large.AvgDuplicationPct = 5.0
	largeIssues := makeIssues(300, 100, 0, 0)

	scoreSmall, _ := computeScore(smallIssues, small, DefaultDimensions)
	scoreLarge, _ := computeScore(largeIssues, large, DefaultDimensions)

	if math.Abs(scoreSmall-scoreLarge) > 2.0 {
		t.Errorf("scale invariance violated: small=%.2f, large=%.2f (diff=%.2f, max allowed=2.0)",
			scoreSmall, scoreLarge, math.Abs(scoreSmall-scoreLarge))
	}
}

// TestScoreMonotonicity verifies that increasing any issue count never raises the score.
func TestScoreMonotonicity(t *testing.T) {
	t.Parallel()

	base := makeMetrics(100, 20000, 80, 100)
	base.HighComplexityCount = 5
	base.LongFileCount = 5
	base.ShortFileCount = 95
	base.TechDebtMarkers = 20
	base.AdequateCommentFiles = 80
	base.AvgDuplicationPct = 3.0

	baseIssues := makeIssues(10, 5, 2, 1)
	baseScore, _ := computeScore(baseIssues, base, DefaultDimensions)

	// More lint issues
	moreLint := makeIssues(20, 5, 2, 1)
	scoreLint, _ := computeScore(moreLint, base, DefaultDimensions)
	if scoreLint > baseScore {
		t.Errorf("more lint issues raised score: base=%.2f, more=%.2f", baseScore, scoreLint)
	}

	// More type issues
	moreType := makeIssues(10, 15, 2, 1)
	scoreType, _ := computeScore(moreType, base, DefaultDimensions)
	if scoreType > baseScore {
		t.Errorf("more type issues raised score: base=%.2f, more=%.2f", baseScore, scoreType)
	}

	// More high complexity files
	worseComplexity := *base
	worseComplexity.HighComplexityCount = 20
	scoreComplexity, _ := computeScore(baseIssues, &worseComplexity, DefaultDimensions)
	if scoreComplexity > baseScore {
		t.Errorf("more complex files raised score: base=%.2f, more=%.2f", baseScore, scoreComplexity)
	}

	// More long files
	worseLong := *base
	worseLong.LongFileCount = 20
	worseLong.ShortFileCount = 80
	scoreLong, _ := computeScore(baseIssues, &worseLong, DefaultDimensions)
	if scoreLong > baseScore {
		t.Errorf("more long files raised score: base=%.2f, more=%.2f", baseScore, scoreLong)
	}

	// Fewer tested files
	worseCoverage := *base
	worseCoverage.TestedFiles = 40
	scoreCoverage, _ := computeScore(baseIssues, &worseCoverage, DefaultDimensions)
	if scoreCoverage > baseScore {
		t.Errorf("fewer tests raised score: base=%.2f, fewer=%.2f", baseScore, scoreCoverage)
	}

	// More tech debt
	worseDebt := *base
	worseDebt.TechDebtMarkers = 100
	scoreDebt, _ := computeScore(baseIssues, &worseDebt, DefaultDimensions)
	if scoreDebt > baseScore {
		t.Errorf("more tech debt raised score: base=%.2f, more=%.2f", baseScore, scoreDebt)
	}

	// Worse duplication
	worseDup := *base
	worseDup.AvgDuplicationPct = 15.0
	scoreDup, _ := computeScore(baseIssues, &worseDup, DefaultDimensions)
	if scoreDup > baseScore {
		t.Errorf("more duplication raised score: base=%.2f, more=%.2f", baseScore, scoreDup)
	}
}

// TestSingleDimensionIsolation verifies that maxing out one dimension drops
// the score by approximately that dimension's weight.
func TestSingleDimensionIsolation(t *testing.T) {
	t.Parallel()

	// Perfect baseline
	perfect := makeMetrics(100, 20000, 100, 100)
	perfectIssues := makeIssues(0, 0, 0, 0)
	perfectScore, _ := computeScore(perfectIssues, perfect, DefaultDimensions)

	tests := []struct {
		name          string
		issues        *issueCountsResult
		metrics       *fileMetricsResult
		dimensionName string
	}{
		{
			name:          "lint maxed out",
			issues:        makeIssues(100, 0, 0, 0), // 1 per file, saturates at 0.5
			metrics:       makeMetrics(100, 20000, 100, 100),
			dimensionName: "lint",
		},
		{
			name:   "type_safety maxed out",
			issues: makeIssues(0, 50, 50, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 100, 100)
				m.TypeSafetyMarkers = 200 // massive type unsafety
				return m
			}(),
			dimensionName: "type_safety",
		},
		{
			name:   "complexity maxed out",
			issues: makeIssues(0, 0, 0, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 100, 100)
				m.HighComplexityCount = 100 // all files high complexity
				return m
			}(),
			dimensionName: "complexity",
		},
		{
			name:   "file_length maxed out",
			issues: makeIssues(0, 0, 0, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 100, 100)
				m.LongFileCount = 100
				m.ShortFileCount = 0
				return m
			}(),
			dimensionName: "file_length",
		},
		{
			name:   "test_coverage maxed out",
			issues: makeIssues(0, 0, 0, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 0, 100)
				return m
			}(),
			dimensionName: "test_coverage",
		},
		{
			name:   "tech_debt maxed out",
			issues: makeIssues(0, 0, 0, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 100, 100)
				m.TechDebtMarkers = 400 // 20/KLOC, saturates at 10
				return m
			}(),
			dimensionName: "tech_debt",
		},
		{
			name:   "comments maxed out",
			issues: makeIssues(0, 0, 0, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 100, 100)
				m.AdequateCommentFiles = 0
				return m
			}(),
			dimensionName: "comments",
		},
		{
			name:   "duplication maxed out",
			issues: makeIssues(0, 0, 0, 0),
			metrics: func() *fileMetricsResult {
				m := makeMetrics(100, 20000, 100, 100)
				m.AvgDuplicationPct = 40.0 // saturates at 20
				return m
			}(),
			dimensionName: "duplication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score, _ := computeScore(tt.issues, tt.metrics, DefaultDimensions)
			weight := DefaultDimensions[tt.dimensionName].Weight
			expectedScore := perfectScore - weight

			// Allow ±1 tolerance for floating-point rounding
			if math.Abs(score-expectedScore) > 1.0 {
				t.Errorf("dimension %q maxed: score=%.2f, expected ~%.2f (perfect=%.2f, weight=%.0f)",
					tt.dimensionName, score, expectedScore, perfectScore, weight)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Unit tests: helpers
// ---------------------------------------------------------------------------

func TestSaturate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value, k, expected float64
	}{
		{0, 10, 100}, // no issues = perfect
		{5, 10, 50},  // half saturated
		{10, 10, 0},  // fully saturated
		{20, 10, 0},  // over-saturated clamps to 0
		{0, 0, 100},  // zero k = always perfect (guard)
		{5, 0, 100},  // zero k = always perfect (guard)
		{0, -1, 100}, // negative k = always perfect (guard)
	}

	for _, tt := range tests {
		result := saturate(tt.value, tt.k)
		if math.Abs(result-tt.expected) > 0.01 {
			t.Errorf("saturate(%.1f, %.1f) = %.2f, want %.2f", tt.value, tt.k, result, tt.expected)
		}
	}
}

func TestRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		clean, total int
		expected     float64
	}{
		{100, 100, 100}, // all clean
		{50, 100, 50},   // half clean
		{0, 100, 0},     // none clean
		{0, 0, 100},     // no items = perfect
		{5, 0, 100},     // zero total edge case
	}

	for _, tt := range tests {
		result := ratio(tt.clean, tt.total)
		if math.Abs(result-tt.expected) > 0.01 {
			t.Errorf("ratio(%d, %d) = %.2f, want %.2f", tt.clean, tt.total, result, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// Structural tests: configuration
// ---------------------------------------------------------------------------

func TestDimensionWeightsSumTo100(t *testing.T) {
	t.Parallel()
	sum := 0.0
	for _, d := range DefaultDimensions {
		sum += d.Weight
	}
	if math.Abs(sum-100.0) > 0.01 {
		t.Errorf("dimension weights sum to %.2f, expected 100.0", sum)
	}
}

func TestAllSaturationConstantsPositive(t *testing.T) {
	t.Parallel()
	for name, d := range DefaultDimensions {
		if d.Saturation < 0 {
			t.Errorf("dimension %q has negative saturation: %.2f", name, d.Saturation)
		}
	}
}

func TestAllWeightsPositive(t *testing.T) {
	t.Parallel()
	for name, d := range DefaultDimensions {
		if d.Weight <= 0 {
			t.Errorf("dimension %q has non-positive weight: %.2f", name, d.Weight)
		}
	}
}

func TestValidateDimensionWeightsPanicsOnBadSum(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for bad weight sum, got none")
		}
	}()

	bad := map[string]DimensionConfig{
		"lint": {Weight: 50},
		"type": {Weight: 60},
	}
	validateDimensionWeights(bad)
}

// ---------------------------------------------------------------------------
// Structural tests: response format
// ---------------------------------------------------------------------------

func TestTidinessScoreResponseFormat(t *testing.T) {
	t.Parallel()

	response := TidinessScoreResponse{
		Scenario:   "test-scenario",
		Score:      72.5,
		Violations: 8,
		Breakdown: &TidinessBreakdown{
			LintScore:         85.0,
			TypeSafetyScore:   90.0,
			ComplexityScore:   70.0,
			FileLengthScore:   80.0,
			TestCoverageScore: 60.0,
			TechDebtScore:     75.0,
			CommentsScore:     50.0,
			DuplicationScore:  95.0,
			LintIssues:        5,
			TypeIssues:        3,
			LongFiles:         2,
			ComplexFunctions:  4,
			TechDebtMarkers:   15,
			TypeSafetyMarkers: 8,
			TestedFiles:       30,
			TestableFiles:     50,
		},
		Metrics: &TidinessMetricsSummary{
			TotalFiles:      100,
			TotalLines:      15000,
			TotalCodeLines:  12000,
			KLOC:            12.0,
			AvgFileLength:   150.0,
			TestCoveragePct: 60.0,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for _, field := range []string{"scenario", "score", "violations", "breakdown", "metrics"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Verify breakdown has dimension scores
	bd := parsed["breakdown"].(map[string]interface{})
	for _, field := range []string{
		"lint_score", "type_safety_score", "complexity_score", "file_length_score",
		"test_coverage_score", "tech_debt_score", "comments_score", "duplication_score",
	} {
		if _, ok := bd[field]; !ok {
			t.Errorf("breakdown missing field: %s", field)
		}
	}

	// Verify breakdown has raw counts (GCT compatibility)
	for _, field := range []string{
		"lint_issues", "type_issues", "long_files", "complex_functions",
		"tech_debt_markers", "duplication_issues",
	} {
		if _, ok := bd[field]; !ok {
			t.Errorf("breakdown missing raw count field: %s", field)
		}
	}

	// Verify metrics has new fields
	met := parsed["metrics"].(map[string]interface{})
	for _, field := range []string{"total_code_lines", "kloc", "test_coverage_pct"} {
		if _, ok := met[field]; !ok {
			t.Errorf("metrics missing field: %s", field)
		}
	}

	// Round-trip: unmarshal back into typed struct
	var roundTrip TidinessScoreResponse
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("round-trip unmarshal failed: %v", err)
	}
	if roundTrip.Score != 72.5 {
		t.Errorf("round-trip score: got %.2f, want 72.5", roundTrip.Score)
	}
	if roundTrip.Breakdown.LintScore != 85.0 {
		t.Errorf("round-trip lint_score: got %.2f, want 85.0", roundTrip.Breakdown.LintScore)
	}
}

// ---------------------------------------------------------------------------
// Integration tests (require database)
// ---------------------------------------------------------------------------

func TestHandleGetTidinessScoreWithDB(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	scenario := "test-tidiness-score-v2"

	cleanup := func() {
		_, _ = srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", scenario)
		_, _ = srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", scenario)
		_, _ = srv.db.ExecContext(ctx, "DELETE FROM scan_history WHERE scenario = $1", scenario)
	}
	cleanup()
	defer cleanup()

	// Insert realistic data: 3 files, 2 lint issues, 1 type error
	_, err := srv.db.ExecContext(ctx, `
		INSERT INTO issues (scenario, file_path, category, severity, title, status)
		VALUES
			($1, 'file1.go', 'lint', 'warning', 'Lint issue 1', 'open'),
			($1, 'file2.go', 'lint', 'warning', 'Lint issue 2', 'open'),
			($1, 'file3.go', 'type', 'error', 'Type error 1', 'open')
	`, scenario)
	if err != nil {
		t.Fatalf("failed to insert issues: %v", err)
	}

	_, err = srv.db.ExecContext(ctx, `
		INSERT INTO file_metrics (scenario, file_path, line_count, code_lines, todo_count, fixme_count, hack_count, has_test_file, comment_to_code_ratio)
		VALUES
			($1, 'file1.go', 100, 80, 2, 1, 0, true, 0.10),
			($1, 'file2.go', 500, 400, 0, 0, 0, false, 0.02),
			($1, 'file3.go', 200, 160, 1, 0, 1, true, 0.08)
	`, scenario)
	if err != nil {
		t.Fatalf("failed to insert file metrics: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/scenarios/"+scenario+"/tidiness", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": scenario})
	rr := httptest.NewRecorder()
	srv.handleGetTidinessScore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp TidinessScoreResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Scenario != scenario {
		t.Errorf("scenario: got %q, want %q", resp.Scenario, scenario)
	}
	if resp.Score < 0 || resp.Score > 100 {
		t.Errorf("score %.2f outside [0, 100]", resp.Score)
	}
	// With 3 issues and mixed quality, score should be between 30 and 95
	if resp.Score < 30 || resp.Score > 95 {
		t.Errorf("score %.2f outside expected range [30, 95] for moderate test data", resp.Score)
	}
	if resp.Violations != 3 {
		t.Errorf("violations: got %d, want 3", resp.Violations)
	}
	if resp.Breakdown == nil {
		t.Fatal("breakdown should not be nil")
	}
	if resp.Breakdown.LintIssues != 2 {
		t.Errorf("lint issues: got %d, want 2", resp.Breakdown.LintIssues)
	}
	if resp.Breakdown.TypeIssues != 1 {
		t.Errorf("type issues: got %d, want 1", resp.Breakdown.TypeIssues)
	}
	if resp.Metrics == nil {
		t.Fatal("metrics should not be nil")
	}
	if resp.Metrics.TotalFiles != 3 {
		t.Errorf("total files: got %d, want 3", resp.Metrics.TotalFiles)
	}
	if resp.Metrics.KLOC <= 0 {
		t.Errorf("KLOC should be positive, got %.4f", resp.Metrics.KLOC)
	}
}

func TestHandleGetTidinessScoreEmptyScenario(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	req := httptest.NewRequest("GET", "/api/v1/scenarios/test-empty-v2/tidiness", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-empty-v2"})
	rr := httptest.NewRecorder()
	srv.handleGetTidinessScore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp TidinessScoreResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if resp.Score != 100 {
		t.Errorf("empty scenario should score 100, got %.2f", resp.Score)
	}
}

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
		t.Errorf("expected 400 for missing scenario, got %d", rr.Code)
	}
}

func TestScoreIsIdempotent(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	scenario := "test-idempotent-v2"

	_, _ = srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", scenario)
	_, _ = srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", scenario)
	defer func() {
		_, _ = srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", scenario)
		_, _ = srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", scenario)
	}()

	_, _ = srv.db.ExecContext(ctx, `
		INSERT INTO issues (scenario, file_path, category, severity, title, status)
		VALUES ($1, 'test.go', 'lint', 'warning', 'Test', 'open')
	`, scenario)

	calc := NewTidinessScoreCalculator(srv.db)

	r1, err := calc.Calculate(ctx, scenario)
	if err != nil {
		t.Fatalf("calc 1 failed: %v", err)
	}
	r2, err := calc.Calculate(ctx, scenario)
	if err != nil {
		t.Fatalf("calc 2 failed: %v", err)
	}
	r3, err := calc.Calculate(ctx, scenario)
	if err != nil {
		t.Fatalf("calc 3 failed: %v", err)
	}

	if r1.Score != r2.Score || r2.Score != r3.Score {
		t.Errorf("not idempotent: %.2f, %.2f, %.2f", r1.Score, r2.Score, r3.Score)
	}
	if r1.Violations != r2.Violations || r2.Violations != r3.Violations {
		t.Errorf("violations not idempotent: %d, %d, %d", r1.Violations, r2.Violations, r3.Violations)
	}
}

func TestScoreReflectsIssueResolution(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	scenario := "test-resolution-v2"

	_, _ = srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", scenario)
	_, _ = srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", scenario)
	defer func() { _, _ = srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", scenario) }()
	defer func() { _, _ = srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", scenario) }()

	for i := 0; i < 20; i++ {
		path := fmt.Sprintf("f%d.go", i+1)
		_, _ = srv.db.ExecContext(ctx, `
			INSERT INTO file_metrics (
				scenario, file_path, line_count, code_lines, todo_count, fixme_count, hack_count,
				has_test_file, comment_to_code_ratio, complexity_max, complexity_avg, duplication_pct
			)
			VALUES ($1, $2, 200, 160, 0, 0, 0, true, 0.10, 3, 2.0, 0.0)
		`, scenario, path)
	}

	_, _ = srv.db.ExecContext(ctx, `
		INSERT INTO issues (scenario, file_path, category, severity, title, status)
		VALUES
			($1, 'f1.go', 'lint', 'warning', 'I1', 'open'),
			($1, 'f2.go', 'lint', 'warning', 'I2', 'open'),
			($1, 'f3.go', 'lint', 'warning', 'I3', 'open'),
			($1, 'f4.go', 'lint', 'warning', 'I4', 'open'),
			($1, 'f5.go', 'lint', 'warning', 'I5', 'open')
	`, scenario)

	calc := NewTidinessScoreCalculator(srv.db)

	before, err := calc.Calculate(ctx, scenario)
	if err != nil {
		t.Fatalf("initial calc failed: %v", err)
	}

	_, _ = srv.db.ExecContext(ctx, `
		UPDATE issues SET status = 'resolved'
		WHERE scenario = $1 AND file_path IN ('f1.go', 'f2.go', 'f3.go')
	`, scenario)

	after, err := calc.Calculate(ctx, scenario)
	if err != nil {
		t.Fatalf("post-resolve calc failed: %v", err)
	}

	if after.Score <= before.Score {
		t.Errorf("score should improve: before=%.2f, after=%.2f", before.Score, after.Score)
	}
	if after.Violations >= before.Violations {
		t.Errorf("violations should decrease: before=%d, after=%d", before.Violations, after.Violations)
	}
}

func TestScoreScaleInvariance_Integration(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil || srv.db == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	small := "test-scale-small"
	large := "test-scale-large"

	cleanup := func() {
		for _, s := range []string{small, large} {
			_, _ = srv.db.ExecContext(ctx, "DELETE FROM issues WHERE scenario = $1", s)
			_, _ = srv.db.ExecContext(ctx, "DELETE FROM file_metrics WHERE scenario = $1", s)
		}
	}
	cleanup()
	defer cleanup()

	// Small: 10 files, 2 lint issues (0.2 per file)
	for i := 0; i < 10; i++ {
		path := "file" + string(rune('a'+i)) + ".go"
		_, _ = srv.db.ExecContext(ctx, `
			INSERT INTO file_metrics (scenario, file_path, line_count, code_lines, todo_count, fixme_count, hack_count, has_test_file, comment_to_code_ratio)
			VALUES ($1, $2, 200, 160, 1, 0, 0, true, 0.10)
		`, small, path)
	}
	_, _ = srv.db.ExecContext(ctx, `
		INSERT INTO issues (scenario, file_path, category, severity, title, status)
		VALUES ($1, 'filea.go', 'lint', 'warning', 'L1', 'open'),
			   ($1, 'fileb.go', 'lint', 'warning', 'L2', 'open')
	`, small)

	// Large: 100 files, 20 lint issues (0.2 per file — same density)
	for i := 0; i < 100; i++ {
		path := "file" + string(rune('a'+(i/26))) + string(rune('a'+(i%26))) + ".go"
		_, _ = srv.db.ExecContext(ctx, `
			INSERT INTO file_metrics (scenario, file_path, line_count, code_lines, todo_count, fixme_count, hack_count, has_test_file, comment_to_code_ratio)
			VALUES ($1, $2, 200, 160, 1, 0, 0, true, 0.10)
		`, large, path)
	}
	for i := 0; i < 20; i++ {
		path := "file" + string(rune('a'+(i/26))) + string(rune('a'+(i%26))) + ".go"
		_, _ = srv.db.ExecContext(ctx, `
			INSERT INTO issues (scenario, file_path, category, severity, title, status)
			VALUES ($1, $2, 'lint', 'warning', 'L', 'open')
		`, large, path)
	}

	calc := NewTidinessScoreCalculator(srv.db)

	rSmall, err := calc.Calculate(ctx, small)
	if err != nil {
		t.Fatalf("small calc failed: %v", err)
	}
	rLarge, err := calc.Calculate(ctx, large)
	if err != nil {
		t.Fatalf("large calc failed: %v", err)
	}

	diff := math.Abs(rSmall.Score - rLarge.Score)
	if diff > 2.0 {
		t.Errorf("scale invariance violated: small=%.2f, large=%.2f (diff=%.2f)",
			rSmall.Score, rLarge.Score, diff)
	}
}
