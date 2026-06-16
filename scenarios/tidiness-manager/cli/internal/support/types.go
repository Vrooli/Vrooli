package support

import commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

type CommandRun struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMS int64  `json:"duration_ms"`
	Success    bool   `json:"success"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason,omitempty"`
}

type LongFile struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Threshold int    `json:"threshold"`
}

type LightScanResult struct {
	Scenario        string      `json:"scenario"`
	DurationMS      int64       `json:"duration_ms"`
	LintOutput      *CommandRun `json:"lint_output,omitempty"`
	TypeOutput      *CommandRun `json:"type_output,omitempty"`
	LongFiles       []LongFile  `json:"long_files"`
	TotalFiles      int         `json:"total_files"`
	TotalLines      int         `json:"total_lines"`
	LintIssuesCount int         `json:"lint_issues"`
	TypeIssuesCount int         `json:"type_issues"`
	LongFilesCount  int         `json:"long_files_count"`
}

type TidinessScanResponse struct {
	Scenario   string                       `json:"scenario"`
	Status     string                       `json:"status"`
	Findings   []TidinessFinding            `json:"findings"`
	Violations []TidinessFinding            `json:"violations"`
	Summary    TidinessScanSummary          `json:"summary"`
	Assessment *commonv1.MaturityAssessment `json:"assessment"`
}

type TidinessScanSummary struct {
	TotalFindings int `json:"total_findings"`
	LongFiles     int `json:"long_files"`
	Complexity    int `json:"complexity"`
	Duplication   int `json:"duplication"`
	TechDebt      int `json:"tech_debt"`
	Coupling      int `json:"coupling"`
}

type TidinessFinding struct {
	ID                     string         `json:"id"`
	RuleID                 string         `json:"rule_id"`
	Scenario               string         `json:"scenario"`
	FilePath               string         `json:"file_path,omitempty"`
	Symbol                 string         `json:"symbol,omitempty"`
	LineNumber             int            `json:"line_number,omitempty"`
	Category               string         `json:"category"`
	Severity               string         `json:"severity"`
	Title                  string         `json:"title"`
	Description            string         `json:"description"`
	Evidence               map[string]any `json:"evidence,omitempty"`
	WhyItMatters           string         `json:"why_it_matters"`
	RecommendedRemediation string         `json:"recommended_remediation"`
	Remediation            string         `json:"remediation"`
	CampaignGroupHint      string         `json:"campaign_group_hint,omitempty"`
}

type Issue struct {
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

type StalenessInfo struct {
	LastScanAt    *string `json:"last_scan_at,omitempty"`
	IsStale       bool    `json:"is_stale"`
	ModifiedFiles int     `json:"modified_files,omitempty"`
	StaleReason   string  `json:"stale_reason,omitempty"`
	RescanCommand string  `json:"rescan_command,omitempty"`
}

type Campaign struct {
	ID                 int    `json:"id"`
	Scenario           string `json:"scenario"`
	Status             string `json:"status"`
	MaxSessions        int    `json:"max_sessions"`
	MaxFilesPerSession int    `json:"max_files_per_session"`
	CurrentSession     int    `json:"current_session"`
	FilesVisited       int    `json:"files_visited"`
	FilesTotal         int    `json:"files_total"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type CampaignsResponse struct {
	Campaigns []Campaign `json:"campaigns"`
	Count     int        `json:"count"`
}

type CampaignEnvelope struct {
	Campaign Campaign `json:"campaign"`
}

type Recommendation struct {
	FilePath         string   `json:"file_path"`
	Language         string   `json:"language"`
	LineCount        int      `json:"line_count"`
	VisitCount       int      `json:"visit_count"`
	RefactorPriority float64  `json:"refactor_priority"`
	StalenessScore   float64  `json:"staleness_score"`
	ComplexityMax    *int     `json:"complexity_max,omitempty"`
	TodoCount        int      `json:"todo_count"`
	FixmeCount       int      `json:"fixme_count"`
	HackCount        int      `json:"hack_count"`
	HasTestFile      *bool    `json:"has_test_file,omitempty"`
	Reasons          []string `json:"reasons,omitempty"`
}

type RecommendationsResponse struct {
	Scenario        string           `json:"scenario"`
	Recommendations []Recommendation `json:"recommendations"`
	Count           int              `json:"count"`
	Warning         string           `json:"warning,omitempty"`
}

type SmartScanRequest struct {
	Scenario    string   `json:"scenario"`
	Files       []string `json:"files"`
	ForceRescan bool     `json:"force_rescan,omitempty"`
	CampaignID  *int     `json:"campaign_id,omitempty"`
}

type SmartScanResult struct {
	SessionID     string        `json:"session_id"`
	FilesAnalyzed int           `json:"files_analyzed"`
	IssuesFound   int           `json:"issues_found"`
	BatchResults  []BatchResult `json:"batch_results"`
	Duration      string        `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
}

type BatchResult struct {
	BatchID  int       `json:"batch_id"`
	Files    []string  `json:"files"`
	Issues   []AIIssue `json:"issues"`
	Duration string    `json:"duration"`
	Error    string    `json:"error,omitempty"`
}

type AIIssue struct {
	FilePath         string `json:"file_path"`
	Category         string `json:"category"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	LineNumber       *int   `json:"line_number,omitempty"`
	RemediationSteps string `json:"remediation_steps,omitempty"`
}

type ScenarioSummary struct {
	Scenario  string `json:"scenario"`
	Total     int    `json:"total"`
	Lint      int    `json:"lint"`
	Type      int    `json:"type"`
	LongFiles int    `json:"long_files"`
}

type ScenariosResponse struct {
	Scenarios []ScenarioSummary `json:"scenarios"`
	Count     int               `json:"count"`
}

type ScenarioFile struct {
	Path        string `json:"path"`
	Lines       int    `json:"lines"`
	TotalIssues int    `json:"totalIssues"`
	VisitCount  int    `json:"visitCount"`
}

type ScenarioDetail struct {
	Scenario    string         `json:"scenario"`
	LightIssues int            `json:"lightIssues"`
	AIIssues    int            `json:"aiIssues"`
	LongFiles   int            `json:"longFiles"`
	Files       []ScenarioFile `json:"files"`
}

type IssueUpdateResponse struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

type TidinessScoreResponse struct {
	Scenario   string               `json:"scenario"`
	Score      float64              `json:"score"`
	Violations int                  `json:"violations"`
	LastScan   *string              `json:"last_scan,omitempty"`
	Breakdown  *TidinessBreakdown   `json:"breakdown,omitempty"`
	Metrics    *TidinessMetricsInfo `json:"metrics,omitempty"`
}

type TidinessBreakdown struct {
	LintScore         float64 `json:"lint_score"`
	TypeSafetyScore   float64 `json:"type_safety_score"`
	ComplexityScore   float64 `json:"complexity_score"`
	FileLengthScore   float64 `json:"file_length_score"`
	TestCoverageScore float64 `json:"test_coverage_score"`
	TechDebtScore     float64 `json:"tech_debt_score"`
	CommentsScore     float64 `json:"comments_score"`
	DuplicationScore  float64 `json:"duplication_score"`
	LintIssues        int     `json:"lint_issues"`
	TypeIssues        int     `json:"type_issues"`
	LongFiles         int     `json:"long_files"`
	ComplexFunctions  int     `json:"complex_functions"`
	TechDebtMarkers   int     `json:"tech_debt_markers"`
	DuplicationIssues int     `json:"duplication_issues"`
	TestedFiles       int     `json:"tested_files"`
	TestableFiles     int     `json:"testable_files"`
}

type TidinessMetricsInfo struct {
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
