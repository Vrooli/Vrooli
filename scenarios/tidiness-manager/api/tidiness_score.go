package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// TidinessScoreResponse is the top-level response for the tidiness score endpoint.
type TidinessScoreResponse struct {
	Scenario   string                  `json:"scenario"`
	Score      float64                 `json:"score"`
	Violations int                     `json:"violations"`
	LastScan   *time.Time              `json:"last_scan,omitempty"`
	Breakdown  *TidinessBreakdown      `json:"breakdown,omitempty"`
	Metrics    *TidinessMetricsSummary `json:"metrics,omitempty"`
}

// TidinessBreakdown exposes per-dimension sub-scores (0-100) and raw counts.
type TidinessBreakdown struct {
	// Per-dimension sub-scores (0-100, where 100 = perfectly clean)
	LintScore         float64 `json:"lint_score"`
	TypeSafetyScore   float64 `json:"type_safety_score"`
	ComplexityScore   float64 `json:"complexity_score"`
	FileLengthScore   float64 `json:"file_length_score"`
	TestCoverageScore float64 `json:"test_coverage_score"`
	TechDebtScore     float64 `json:"tech_debt_score"`
	CommentsScore     float64 `json:"comments_score"`
	DuplicationScore  float64 `json:"duplication_score"`

	// Raw counts for display and downstream consumers (e.g. GCT buildCodeQualityIssues)
	LintIssues        int `json:"lint_issues"`
	TypeIssues        int `json:"type_issues"`
	LongFiles         int `json:"long_files"`
	ComplexFunctions  int `json:"complex_functions"`
	TechDebtMarkers   int `json:"tech_debt_markers"`
	DuplicationIssues int `json:"duplication_issues"`
	TypeSafetyMarkers int `json:"type_safety_markers"`
	TestedFiles       int `json:"tested_files"`
	TestableFiles     int `json:"testable_files"`
}

// TidinessMetricsSummary holds aggregate code metrics.
type TidinessMetricsSummary struct {
	TotalFiles      int     `json:"total_files"`
	TotalLines      int     `json:"total_lines"`
	TotalCodeLines  int     `json:"total_code_lines"`
	KLOC            float64 `json:"kloc"`
	AvgFileLength   float64 `json:"avg_file_length"`
	MaxComplexity   int     `json:"max_complexity"`
	AvgComplexity   float64 `json:"avg_complexity"`
	DuplicationPct  float64 `json:"duplication_pct"`
	TestCoveragePct float64 `json:"test_coverage_pct"`
}

// ---------------------------------------------------------------------------
// Dimension configuration
// ---------------------------------------------------------------------------

// DimensionConfig defines how a single quality dimension is scored.
//   - Weight: points this dimension contributes (all weights must sum to 100).
//   - Saturation: for density-based dimensions, the density at which the
//     sub-score reaches 0. Higher = more forgiving.
//   - Threshold: for ratio-based dimensions, the boundary value that separates
//     "clean" from "dirty" (e.g., max complexity, min comment ratio).
type DimensionConfig struct {
	Weight     float64
	Saturation float64
	Threshold  float64
}

// DefaultDimensions defines the scoring weights and thresholds for each
// quality dimension. Adjust these to tune score sensitivity.
//
// Dimension types:
//   - Density-based (use Saturation): lint, type_safety, tech_debt, duplication
//   - Ratio-based (use Threshold): complexity, file_length, comments
//   - Pure ratio (no config): test_coverage
var DefaultDimensions = map[string]DimensionConfig{
	"lint":          {Weight: 20, Saturation: 0.5},  // lint issues per file
	"type_safety":   {Weight: 20, Saturation: 5.0},  // markers per KLOC
	"complexity":    {Weight: 15, Threshold: 10},    // max cyclomatic complexity
	"file_length":   {Weight: 10, Threshold: 400},   // line count (test files: 2.5×)
	"test_coverage": {Weight: 15},                   // pure ratio
	"tech_debt":     {Weight: 10, Saturation: 10.0}, // markers per KLOC
	"comments":      {Weight: 5, Threshold: 0.05},   // min comment-to-code ratio
	"duplication":   {Weight: 5, Saturation: 20.0},  // avg duplication %
}

// testFileLengthMultiplier is applied to the file_length threshold for test files.
const testFileLengthMultiplier = 2.5

func init() {
	validateDimensionWeights(DefaultDimensions)
}

// validateDimensionWeights panics if weights do not sum to 100.
func validateDimensionWeights(dims map[string]DimensionConfig) {
	sum := 0.0
	for _, d := range dims {
		sum += d.Weight
	}
	if math.Abs(sum-100.0) > 0.01 {
		panic(fmt.Sprintf("dimension weights must sum to 100, got %.2f", sum))
	}
}

// ---------------------------------------------------------------------------
// Calculator
// ---------------------------------------------------------------------------

// TidinessScoreCalculator computes tidiness scores from database metrics.
type TidinessScoreCalculator struct {
	db         *sql.DB
	dimensions map[string]DimensionConfig
}

// NewTidinessScoreCalculator creates a calculator with default dimensions.
func NewTidinessScoreCalculator(db *sql.DB) *TidinessScoreCalculator {
	return &TidinessScoreCalculator{db: db, dimensions: DefaultDimensions}
}

// Calculate computes the tidiness score for a scenario.
func (c *TidinessScoreCalculator) Calculate(ctx context.Context, scenario string) (*TidinessScoreResponse, error) {
	issues, err := c.getIssueCounts(ctx, scenario)
	if err != nil {
		return nil, err
	}

	metrics, err := c.getFileMetricsAggregates(ctx, scenario)
	if err != nil {
		return nil, err
	}

	lastScan, _ := c.getLastScanTime(ctx, scenario)

	score, breakdown := computeScore(issues, metrics, c.dimensions)

	kloc := math.Max(float64(metrics.TotalCodeLines), 1) / 1000.0
	testCoveragePct := 0.0
	if metrics.TestableFiles > 0 {
		testCoveragePct = float64(metrics.TestedFiles) / float64(metrics.TestableFiles) * 100
	}

	return &TidinessScoreResponse{
		Scenario:   scenario,
		Score:      score,
		Violations: issues.Total,
		LastScan:   lastScan,
		Breakdown:  breakdown,
		Metrics: &TidinessMetricsSummary{
			TotalFiles:      metrics.TotalFiles,
			TotalLines:      metrics.TotalLines,
			TotalCodeLines:  metrics.TotalCodeLines,
			KLOC:            kloc,
			AvgFileLength:   metrics.AvgFileLength,
			MaxComplexity:   metrics.MaxComplexity,
			AvgComplexity:   metrics.AvgComplexity,
			DuplicationPct:  metrics.AvgDuplicationPct,
			TestCoveragePct: testCoveragePct,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Score computation (pure functions)
// ---------------------------------------------------------------------------

// computeScore calculates the weighted score and builds the breakdown.
// Each dimension produces a sub-score from 0-100. The final score is the
// weighted sum: Σ(weight_i × subScore_i / 100), always in [0, 100].
func computeScore(issues *issueCountsResult, metrics *fileMetricsResult, dims map[string]DimensionConfig) (float64, *TidinessBreakdown) {
	totalFiles := math.Max(float64(metrics.TotalFiles), 1)
	kloc := math.Max(float64(metrics.TotalCodeLines), 1) / 1000.0

	// Lint: density = lint issues per file
	lintScore := saturate(float64(issues.Lint)/totalFiles, dims["lint"].Saturation)

	// Type Safety: density = (type issues + type-safety code markers) per KLOC
	typeDensity := float64(issues.Type+issues.TypeSafety+metrics.TypeSafetyMarkers) / kloc
	typeSafetyScore := saturate(typeDensity, dims["type_safety"].Saturation)

	// Complexity: ratio of files under threshold (files without complexity data are excluded)
	complexityScore := ratio(metrics.FilesWithComplexity-metrics.HighComplexityCount, metrics.FilesWithComplexity)

	// File Length: ratio of files under length threshold
	fileLengthScore := ratio(metrics.ShortFileCount, metrics.TotalFiles)

	// Test Coverage: ratio of testable files that have tests
	testCoverageScore := ratio(metrics.TestedFiles, metrics.TestableFiles)

	// Tech Debt: density = (TODO+FIXME+HACK) per KLOC
	techDebtScore := saturate(float64(metrics.TechDebtMarkers)/kloc, dims["tech_debt"].Saturation)

	// Comments: ratio of files meeting the comment threshold
	commentsScore := ratio(metrics.AdequateCommentFiles, metrics.TotalFiles)

	// Duplication: average duplication percentage
	duplicationScore := saturate(metrics.AvgDuplicationPct, dims["duplication"].Saturation)

	// Weighted sum
	finalScore := 0.0
	dimScores := map[string]float64{
		"lint":          lintScore,
		"type_safety":   typeSafetyScore,
		"complexity":    complexityScore,
		"file_length":   fileLengthScore,
		"test_coverage": testCoverageScore,
		"tech_debt":     techDebtScore,
		"comments":      commentsScore,
		"duplication":   duplicationScore,
	}
	for name, subScore := range dimScores {
		finalScore += dims[name].Weight * subScore / 100.0
	}

	// Count files exceeding duplication threshold (10%) for the raw count
	duplicationIssueCount := 0
	if metrics.AvgDuplicationPct > 10 {
		duplicationIssueCount = 1
	}

	breakdown := &TidinessBreakdown{
		LintScore:         lintScore,
		TypeSafetyScore:   typeSafetyScore,
		ComplexityScore:   complexityScore,
		FileLengthScore:   fileLengthScore,
		TestCoverageScore: testCoverageScore,
		TechDebtScore:     techDebtScore,
		CommentsScore:     commentsScore,
		DuplicationScore:  duplicationScore,

		LintIssues:        issues.Lint,
		TypeIssues:        issues.Type,
		LongFiles:         metrics.LongFileCount,
		ComplexFunctions:  metrics.HighComplexityCount,
		TechDebtMarkers:   metrics.TechDebtMarkers,
		DuplicationIssues: duplicationIssueCount,
		TypeSafetyMarkers: metrics.TypeSafetyMarkers,
		TestedFiles:       metrics.TestedFiles,
		TestableFiles:     metrics.TestableFiles,
	}

	return finalScore, breakdown
}

// saturate returns a sub-score using linear degradation:
// score = 100 × max(0, 1 − value/k).
// When value = 0, score = 100. When value ≥ k, score = 0.
func saturate(value, k float64) float64 {
	if k <= 0 {
		return 100
	}
	return math.Max(0, 1-value/k) * 100
}

// ratio returns a sub-score based on the proportion of "clean" items:
// score = 100 × clean/total. If total is 0, returns 100 (no items = no problems).
func ratio(clean, total int) float64 {
	if total <= 0 {
		return 100
	}
	return float64(clean) / float64(total) * 100
}

// ---------------------------------------------------------------------------
// Database queries
// ---------------------------------------------------------------------------

// issueCountsResult holds issue counts by category.
type issueCountsResult struct {
	Total         int
	Lint          int
	Type          int
	TypeSafety    int
	Complexity    int
	Length        int
	Duplication   int
	TechnicalDebt int
	Coupling      int
	AI            int
}

// fileMetricsResult holds aggregated file metrics.
type fileMetricsResult struct {
	TotalFiles           int
	TotalLines           int
	TotalCodeLines       int
	AvgFileLength        float64
	LongFileCount        int
	ShortFileCount       int
	HighComplexityCount  int
	FilesWithComplexity  int
	MaxComplexity        int
	AvgComplexity        float64
	TechDebtMarkers      int
	TestedFiles          int
	TestableFiles        int
	AdequateCommentFiles int
	AvgDuplicationPct    float64
	TypeSafetyMarkers    int
}

func (c *TidinessScoreCalculator) getIssueCounts(ctx context.Context, scenario string) (*issueCountsResult, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN category = 'lint' THEN 1 END),
			COUNT(CASE WHEN category = 'type' THEN 1 END),
			COUNT(CASE WHEN category = 'type_safety' THEN 1 END),
			COUNT(CASE WHEN category = 'complexity' THEN 1 END),
			COUNT(CASE WHEN category = 'length' THEN 1 END),
			COUNT(CASE WHEN category = 'duplication' THEN 1 END),
			COUNT(CASE WHEN category = 'technical_debt' THEN 1 END),
			COUNT(CASE WHEN category = 'coupling' THEN 1 END),
			COUNT(CASE WHEN category IN ('ai', 'dead_code', 'style') THEN 1 END)
		FROM issues
		WHERE scenario = $1 AND status = 'open'
	`

	var r issueCountsResult
	err := c.db.QueryRowContext(ctx, query, scenario).Scan(
		&r.Total, &r.Lint, &r.Type, &r.TypeSafety,
		&r.Complexity, &r.Length, &r.Duplication,
		&r.TechnicalDebt, &r.Coupling, &r.AI,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &issueCountsResult{}, nil
		}
		return nil, err
	}
	return &r, nil
}

// isTestFileSQL is the SQL condition for identifying test files.
// Used in multiple places to partition testable vs test files.
const isTestFileSQL = `(
	file_path LIKE '%%\_test.go'
	OR file_path LIKE '%%.test.ts' OR file_path LIKE '%%.test.tsx'
	OR file_path LIKE '%%.spec.ts' OR file_path LIKE '%%.spec.tsx'
	OR file_path LIKE '%%.test.js' OR file_path LIKE '%%.test.jsx'
	OR file_path LIKE '%%.spec.js' OR file_path LIKE '%%.spec.jsx'
	OR file_path LIKE '%%/tests/%%' OR file_path LIKE '%%/__tests__/%%'
)`

func (c *TidinessScoreCalculator) getFileMetricsAggregates(ctx context.Context, scenario string) (*fileMetricsResult, error) {
	lengthThreshold := int(c.dimensions["file_length"].Threshold)
	testLengthThreshold := int(float64(lengthThreshold) * testFileLengthMultiplier)
	complexityThreshold := int(c.dimensions["complexity"].Threshold)
	commentRatioThreshold := c.dimensions["comments"].Threshold

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_files,
			COALESCE(SUM(line_count), 0) as total_lines,
			COALESCE(SUM(code_lines), 0) as total_code_lines,
			COALESCE(AVG(line_count), 0) as avg_file_length,
			-- Long files (over threshold, test-aware)
			COUNT(CASE WHEN
				(%s AND line_count > $2)
				OR (NOT %s AND line_count > $3)
			THEN 1 END) as long_files,
			-- Short files (under threshold, test-aware)
			COUNT(CASE WHEN
				(%s AND line_count <= $2)
				OR (NOT %s AND line_count <= $3)
			THEN 1 END) as short_files,
			-- Complexity
			COUNT(CASE WHEN complexity_max > $4 THEN 1 END) as high_complexity,
			COUNT(CASE WHEN complexity_max IS NOT NULL THEN 1 END) as files_with_complexity,
			COALESCE(MAX(complexity_max), 0) as max_complexity,
			COALESCE(AVG(CASE WHEN complexity_max IS NOT NULL THEN complexity_avg END), 0) as avg_complexity,
			-- Tech debt
			COALESCE(SUM(todo_count + fixme_count + hack_count), 0) as tech_debt,
			-- Test coverage (testable = not a test file; tested = has_test_file among testable)
			COUNT(CASE WHEN NOT %s AND has_test_file = true THEN 1 END) as tested_files,
			COUNT(CASE WHEN NOT %s THEN 1 END) as testable_files,
			-- Comments
			COUNT(CASE WHEN comment_to_code_ratio >= $5 THEN 1 END) as adequate_comments,
			-- Duplication
			COALESCE(AVG(duplication_pct), 0) as avg_duplication_pct,
			-- Type safety markers
			COALESCE(SUM(
				COALESCE(as_any_count, 0) +
				COALESCE(as_type_assertion_count, 0) +
				COALESCE(ts_ignore_count, 0) +
				COALESCE(non_null_assertion_count, 0)
			), 0) as type_safety_markers
		FROM file_metrics
		WHERE scenario = $1
	`, isTestFileSQL, isTestFileSQL, isTestFileSQL, isTestFileSQL, isTestFileSQL, isTestFileSQL)

	var r fileMetricsResult
	var maxComplexity, techDebt, typeSafetyMarkers sql.NullInt64
	var avgComplexity, duplicationPct sql.NullFloat64

	err := c.db.QueryRowContext(ctx, query, scenario,
		testLengthThreshold, lengthThreshold, complexityThreshold, commentRatioThreshold,
	).Scan(
		&r.TotalFiles,
		&r.TotalLines,
		&r.TotalCodeLines,
		&r.AvgFileLength,
		&r.LongFileCount,
		&r.ShortFileCount,
		&r.HighComplexityCount,
		&r.FilesWithComplexity,
		&maxComplexity,
		&avgComplexity,
		&techDebt,
		&r.TestedFiles,
		&r.TestableFiles,
		&r.AdequateCommentFiles,
		&duplicationPct,
		&typeSafetyMarkers,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &fileMetricsResult{}, nil
		}
		return nil, err
	}

	if maxComplexity.Valid {
		r.MaxComplexity = int(maxComplexity.Int64)
	}
	if avgComplexity.Valid {
		r.AvgComplexity = avgComplexity.Float64
	}
	if techDebt.Valid {
		r.TechDebtMarkers = int(techDebt.Int64)
	}
	if duplicationPct.Valid {
		r.AvgDuplicationPct = duplicationPct.Float64
	}
	if typeSafetyMarkers.Valid {
		r.TypeSafetyMarkers = int(typeSafetyMarkers.Int64)
	}

	return &r, nil
}

func (c *TidinessScoreCalculator) getLastScanTime(ctx context.Context, scenario string) (*time.Time, error) {
	query := `SELECT MAX(created_at) FROM scan_history WHERE scenario = $1`

	var lastScan sql.NullTime
	err := c.db.QueryRowContext(ctx, query, scenario).Scan(&lastScan)
	if err != nil {
		return nil, err
	}
	if lastScan.Valid {
		return &lastScan.Time, nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// HTTP handler
// ---------------------------------------------------------------------------

// handleGetTidinessScore handles GET /api/v1/scenarios/{scenario}/tidiness
// for consumers of scenario tidiness metrics.
func (s *Server) handleGetTidinessScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]
	if scenario == "" {
		scenario = vars["name"]
	}
	if scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario parameter is required")
		return
	}

	if s.scanCoordinator != nil {
		if normalized, err := s.scanCoordinator.NormalizeScenarioName(scenario); err == nil {
			scenario = normalized
		}
	}

	calculator := NewTidinessScoreCalculator(s.db)
	result, err := calculator.Calculate(r.Context(), scenario)
	if err != nil {
		s.log("failed to calculate tidiness score", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenario,
		})
		respondError(w, http.StatusInternalServerError, "failed to calculate tidiness score")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
