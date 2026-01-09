package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// TidinessScoreResponse is the response format expected by ecosystem-manager
// See: scenarios/ecosystem-manager/api/pkg/autosteer/metrics_refactor.go:48-83
type TidinessScoreResponse struct {
	Scenario   string                  `json:"scenario"`
	Score      float64                 `json:"score"`
	Violations int                     `json:"violations"`
	LastScan   *time.Time              `json:"last_scan,omitempty"`
	Breakdown  *TidinessBreakdown      `json:"breakdown,omitempty"`
	Metrics    *TidinessMetricsSummary `json:"metrics,omitempty"`
}

// TidinessBreakdown provides detailed issue counts by category
type TidinessBreakdown struct {
	LintIssues        int `json:"lint_issues"`
	TypeIssues        int `json:"type_issues"`
	LongFiles         int `json:"long_files"`
	ComplexFunctions  int `json:"complex_functions"`
	TechDebtMarkers   int `json:"tech_debt_markers"`
	DuplicationIssues int `json:"duplication_issues"`
}

// TidinessMetricsSummary provides aggregate code metrics
type TidinessMetricsSummary struct {
	TotalFiles     int     `json:"total_files"`
	TotalLines     int     `json:"total_lines"`
	AvgFileLength  float64 `json:"avg_file_length"`
	MaxComplexity  int     `json:"max_complexity"`
	AvgComplexity  float64 `json:"avg_complexity"`
	DuplicationPct float64 `json:"duplication_pct"`
}

// TidinessScoreCalculator computes tidiness scores from database metrics
type TidinessScoreCalculator struct {
	db *sql.DB
}

// NewTidinessScoreCalculator creates a new calculator instance
func NewTidinessScoreCalculator(db *sql.DB) *TidinessScoreCalculator {
	return &TidinessScoreCalculator{db: db}
}

// ScoreWeights defines the penalty weights for each issue type
// These are tuned to produce a 0-100 scale where 100 is perfectly clean
var ScoreWeights = struct {
	LintIssue       float64
	TypeError       float64
	LongFile        float64
	HighComplexity  float64
	TechDebtMarker  float64
	MissingTests    float64
	LowCommentRatio float64
	DuplicationPct  float64
}{
	LintIssue:       1.0, // -1 per lint issue
	TypeError:       2.0, // -2 per type error (more severe)
	LongFile:        3.0, // -3 per file > threshold
	HighComplexity:  2.0, // -2 per function with complexity > 10
	TechDebtMarker:  0.5, // -0.5 per TODO/FIXME/HACK
	MissingTests:    1.0, // -1 per file without tests
	LowCommentRatio: 0.5, // -0.5 per file with < 5% comments
	DuplicationPct:  0.5, // -0.5 per 1% duplication
}

// Thresholds for determining issues
var Thresholds = struct {
	LongFileLines   int
	HighComplexity  int
	LowCommentRatio float64
}{
	LongFileLines:   400,  // Files over 400 lines are flagged
	HighComplexity:  10,   // Functions with complexity > 10 are flagged
	LowCommentRatio: 0.05, // Files with < 5% comments are flagged
}

// Calculate computes the tidiness score for a scenario
func (c *TidinessScoreCalculator) Calculate(ctx context.Context, scenario string) (*TidinessScoreResponse, error) {
	// Get issue counts from issues table
	issueCounts, err := c.getIssueCounts(ctx, scenario)
	if err != nil {
		return nil, err
	}

	// Get file metrics aggregates
	fileMetrics, err := c.getFileMetricsAggregates(ctx, scenario)
	if err != nil {
		return nil, err
	}

	// Get last scan time
	lastScan, err := c.getLastScanTime(ctx, scenario)
	if err != nil {
		// Non-fatal - just won't include last_scan
		lastScan = nil
	}

	// Calculate the score
	score := c.computeScore(issueCounts, fileMetrics)

	// Build breakdown
	breakdown := &TidinessBreakdown{
		LintIssues:        issueCounts.Lint,
		TypeIssues:        issueCounts.Type,
		LongFiles:         fileMetrics.LongFileCount,
		ComplexFunctions:  fileMetrics.HighComplexityCount,
		TechDebtMarkers:   fileMetrics.TechDebtMarkers,
		DuplicationIssues: 0, // Derived from duplication_pct if available
	}

	// Build metrics summary
	metrics := &TidinessMetricsSummary{
		TotalFiles:     fileMetrics.TotalFiles,
		TotalLines:     fileMetrics.TotalLines,
		AvgFileLength:  fileMetrics.AvgFileLength,
		MaxComplexity:  fileMetrics.MaxComplexity,
		AvgComplexity:  fileMetrics.AvgComplexity,
		DuplicationPct: fileMetrics.DuplicationPct,
	}

	// Total violations = all issues + structural problems
	violations := issueCounts.Total + fileMetrics.LongFileCount + fileMetrics.HighComplexityCount

	return &TidinessScoreResponse{
		Scenario:   scenario,
		Score:      score,
		Violations: violations,
		LastScan:   lastScan,
		Breakdown:  breakdown,
		Metrics:    metrics,
	}, nil
}

// issueCountsResult holds issue counts by category
type issueCountsResult struct {
	Total int
	Lint  int
	Type  int
	AI    int
}

// fileMetricsResult holds aggregated file metrics
type fileMetricsResult struct {
	TotalFiles          int
	TotalLines          int
	AvgFileLength       float64
	LongFileCount       int
	HighComplexityCount int
	MaxComplexity       int
	AvgComplexity       float64
	TechDebtMarkers     int
	MissingTestsCount   int
	LowCommentCount     int
	DuplicationPct      float64
}

func (c *TidinessScoreCalculator) getIssueCounts(ctx context.Context, scenario string) (*issueCountsResult, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN category = 'lint' THEN 1 END) as lint,
			COUNT(CASE WHEN category = 'type' THEN 1 END) as type_issues,
			COUNT(CASE WHEN category IN ('ai', 'complexity', 'length', 'duplication') THEN 1 END) as ai
		FROM issues
		WHERE scenario = $1 AND status = 'open'
	`

	var result issueCountsResult
	err := c.db.QueryRowContext(ctx, query, scenario).Scan(
		&result.Total,
		&result.Lint,
		&result.Type,
		&result.AI,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &issueCountsResult{}, nil
		}
		return nil, err
	}

	return &result, nil
}

func (c *TidinessScoreCalculator) getFileMetricsAggregates(ctx context.Context, scenario string) (*fileMetricsResult, error) {
	query := `
		SELECT
			COUNT(*) as total_files,
			COALESCE(SUM(line_count), 0) as total_lines,
			COALESCE(AVG(line_count), 0) as avg_file_length,
			COUNT(CASE WHEN line_count > $2 THEN 1 END) as long_files,
			COUNT(CASE WHEN complexity_max > $3 THEN 1 END) as high_complexity,
			COALESCE(MAX(complexity_max), 0) as max_complexity,
			COALESCE(AVG(complexity_avg), 0) as avg_complexity,
			COALESCE(SUM(todo_count + fixme_count + hack_count), 0) as tech_debt,
			COUNT(CASE WHEN has_test_file = false THEN 1 END) as missing_tests,
			COUNT(CASE WHEN comment_to_code_ratio < $4 THEN 1 END) as low_comments,
			COALESCE(AVG(duplication_pct), 0) as duplication_pct
		FROM file_metrics
		WHERE scenario = $1
	`

	var result fileMetricsResult
	var maxComplexity, techDebt sql.NullInt64
	var avgComplexity, duplicationPct sql.NullFloat64

	err := c.db.QueryRowContext(ctx, query, scenario, Thresholds.LongFileLines, Thresholds.HighComplexity, Thresholds.LowCommentRatio).Scan(
		&result.TotalFiles,
		&result.TotalLines,
		&result.AvgFileLength,
		&result.LongFileCount,
		&result.HighComplexityCount,
		&maxComplexity,
		&avgComplexity,
		&techDebt,
		&result.MissingTestsCount,
		&result.LowCommentCount,
		&duplicationPct,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &fileMetricsResult{}, nil
		}
		return nil, err
	}

	if maxComplexity.Valid {
		result.MaxComplexity = int(maxComplexity.Int64)
	}
	if avgComplexity.Valid {
		result.AvgComplexity = avgComplexity.Float64
	}
	if techDebt.Valid {
		result.TechDebtMarkers = int(techDebt.Int64)
	}
	if duplicationPct.Valid {
		result.DuplicationPct = duplicationPct.Float64
	}

	return &result, nil
}

func (c *TidinessScoreCalculator) getLastScanTime(ctx context.Context, scenario string) (*time.Time, error) {
	query := `
		SELECT MAX(created_at)
		FROM scan_history
		WHERE scenario = $1
	`

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

func (c *TidinessScoreCalculator) computeScore(issues *issueCountsResult, metrics *fileMetricsResult) float64 {
	// Start with perfect score
	score := 100.0

	// Deduct for lint issues
	score -= float64(issues.Lint) * ScoreWeights.LintIssue

	// Deduct for type errors (more severe)
	score -= float64(issues.Type) * ScoreWeights.TypeError

	// Deduct for long files
	score -= float64(metrics.LongFileCount) * ScoreWeights.LongFile

	// Deduct for high complexity functions
	score -= float64(metrics.HighComplexityCount) * ScoreWeights.HighComplexity

	// Deduct for tech debt markers
	score -= float64(metrics.TechDebtMarkers) * ScoreWeights.TechDebtMarker

	// Deduct for missing tests
	score -= float64(metrics.MissingTestsCount) * ScoreWeights.MissingTests

	// Deduct for low comment ratio files
	score -= float64(metrics.LowCommentCount) * ScoreWeights.LowCommentRatio

	// Deduct for duplication (0.5 per 1% duplication)
	score -= metrics.DuplicationPct * ScoreWeights.DuplicationPct

	// Clamp to 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// handleGetTidinessScore handles GET /api/v1/scenarios/{scenario}/tidiness
// Also handles GET /api/v1/scan/{scenario} for ecosystem-manager compatibility
func (s *Server) handleGetTidinessScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]
	if scenario == "" {
		scenario = vars["name"] // fallback for /api/v1/scan/{name} route
	}

	if scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario parameter is required")
		return
	}

	// Normalize scenario name if scan coordinator is available
	if s.scanCoordinator != nil {
		normalized, err := s.scanCoordinator.NormalizeScenarioName(scenario)
		if err == nil {
			scenario = normalized
		}
		// If normalization fails, use the original name - the query will just return empty results
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
