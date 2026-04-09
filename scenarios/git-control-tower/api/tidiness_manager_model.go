package main

import "time"

// TidinessScoreResponse mirrors the tidiness-manager score endpoint response.
type TidinessScoreResponse struct {
	Scenario   string                  `json:"scenario"`
	Score      float64                 `json:"score"`
	Violations int                     `json:"violations"`
	LastScan   *time.Time              `json:"last_scan,omitempty"`
	Breakdown  *TidinessBreakdown      `json:"breakdown,omitempty"`
	Metrics    *TidinessMetricsSummary `json:"metrics,omitempty"`
}

// TidinessBreakdown exposes per-dimension sub-scores and raw violation counts.
type TidinessBreakdown struct {
	// Per-dimension sub-scores (0-100)
	LintScore         float64 `json:"lint_score"`
	TypeSafetyScore   float64 `json:"type_safety_score"`
	ComplexityScore   float64 `json:"complexity_score"`
	FileLengthScore   float64 `json:"file_length_score"`
	TestCoverageScore float64 `json:"test_coverage_score"`
	TechDebtScore     float64 `json:"tech_debt_score"`
	CommentsScore     float64 `json:"comments_score"`
	DuplicationScore  float64 `json:"duplication_score"`

	// Raw counts
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

// TidinessIssue mirrors an issue from the tidiness-manager agent API.
type TidinessIssue struct {
	ID               int    `json:"id"`
	Scenario         string `json:"scenario"`
	FilePath         string `json:"file_path"`
	Category         string `json:"category"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	LineNumber       *int   `json:"line_number,omitempty"`
	ColumnNumber     *int   `json:"column_number,omitempty"`
	AgentNotes       string `json:"agent_notes,omitempty"`
	RemediationSteps string `json:"remediation_steps,omitempty"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
}

// TidinessStalenessInfo mirrors the staleness endpoint response.
type TidinessStalenessInfo struct {
	LastScanAt    *string `json:"last_scan_at,omitempty"`
	IsStale       bool    `json:"is_stale"`
	ModifiedFiles int     `json:"modified_files,omitempty"`
	StaleReason   string  `json:"stale_reason,omitempty"`
	RescanCommand string  `json:"rescan_command,omitempty"`
}

// TidinessLightScanProxyRequest is the request body accepted by the proxy handler.
// The handler resolves scenario_name to an absolute path before forwarding.
type TidinessLightScanProxyRequest struct {
	ScenarioName string `json:"scenario_name"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
	Incremental  bool   `json:"incremental,omitempty"`
}

// TidinessLightScanRequest mirrors POST /api/v1/scan/light body sent to tidiness-manager.
type TidinessLightScanRequest struct {
	ScenarioPath string `json:"scenario_path"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
	Incremental  bool   `json:"incremental,omitempty"`
}

// TidinessLightScanResult mirrors the light scan response.
type TidinessLightScanResult struct {
	Scenario        string               `json:"scenario"`
	StartedAt       time.Time            `json:"started_at"`
	CompletedAt     time.Time            `json:"completed_at"`
	Duration        int64                `json:"duration_ms"`
	FileMetrics     []TidinessFileMetric `json:"file_metrics"`
	LongFiles       []TidinessLongFile   `json:"long_files"`
	TotalFiles      int                  `json:"total_files"`
	TotalLines      int                  `json:"total_lines"`
	LintIssuesCount int                  `json:"lint_issues"`
	TypeIssuesCount int                  `json:"type_issues"`
	LongFilesCount  int                  `json:"long_files_count"`
}

// TidinessFileMetric holds per-file metrics from a scan.
type TidinessFileMetric struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Extension string `json:"extension"`
}

// TidinessLongFile flags a file exceeding length thresholds.
type TidinessLongFile struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Threshold int    `json:"threshold"`
}

// TidinessScenarioDetail mirrors the agent scenario detail response.
type TidinessScenarioDetail struct {
	Scenario    string                     `json:"scenario"`
	LightIssues int                        `json:"lightIssues"`
	AIIssues    int                        `json:"aiIssues"`
	LongFiles   int                        `json:"longFiles"`
	Files       []TidinessScenarioFileInfo `json:"files"`
}

// TidinessScenarioFileInfo holds per-file summary in a scenario detail.
type TidinessScenarioFileInfo struct {
	Path        string `json:"path"`
	Lines       int    `json:"lines"`
	TotalIssues int    `json:"totalIssues"`
	VisitCount  int    `json:"visitCount"`
}
