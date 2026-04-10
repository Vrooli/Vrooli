package main

import "regexp"

// Readiness represents the overall review readiness of a scenario.
type Readiness string

const (
	ReadinessGreen  Readiness = "green"
	ReadinessYellow Readiness = "yellow"
	ReadinessRed    Readiness = "red"
)

// validScenarioName matches safe scenario names: alphanumeric, hyphens, underscores.
var validScenarioName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// IsValidScenarioName returns true if the name is safe for path construction.
func IsValidScenarioName(name string) bool {
	return len(name) > 0 && len(name) <= 128 && validScenarioName.MatchString(name)
}

// validReviewChecks is the set of recognized check names for review runs.
var validReviewChecks = map[string]bool{
	"tidiness": true,
	"tests":    true,
	"rules":    true,
}

// ReviewSummaryResponse is the unified review summary for a scenario.
type ReviewSummaryResponse struct {
	ScenarioName      string            `json:"scenarioName"`
	Readiness         Readiness         `json:"readiness"`
	Dimensions        ReviewDimensions  `json:"dimensions"`
	DimensionStatuses map[string]string `json:"dimensionStatuses,omitempty"`
	Capabilities      map[string]bool   `json:"capabilities"`
	Timestamp         string            `json:"timestamp"`
}

// ReviewDimensions holds per-dimension review data.
type ReviewDimensions struct {
	CodeQuality *CodeQualityDimension `json:"codeQuality,omitempty"`
	Tests       *TestsDimension       `json:"tests,omitempty"`
	Standards   *StandardsDimension   `json:"standards,omitempty"`
	Visual      *VisualDimension      `json:"visual,omitempty"`
	Provenance  *ProvenanceDimension  `json:"provenance,omitempty"`
}

// CodeQualityDimension summarizes tidiness-manager results.
type CodeQualityDimension struct {
	Available  bool               `json:"available"`
	Score      float64            `json:"score"`
	Violations int                `json:"violations"`
	Stale      bool               `json:"stale"`
	LastScan   string             `json:"lastScan,omitempty"`
	TopIssues  []CodeQualityIssue `json:"topIssues,omitempty"`
}

// CodeQualityIssue is a category-level issue summary from the tidiness breakdown.
type CodeQualityIssue struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// TestsDimension summarizes test-genie results.
type TestsDimension struct {
	Available   bool          `json:"available"`
	Passed      bool          `json:"passed"`
	Total       int           `json:"total"`
	PassedCount int           `json:"passedCount"`
	FailedCount int           `json:"failedCount"`
	LastRun     string        `json:"lastRun,omitempty"`
	Failures    []TestFailure `json:"failures,omitempty"`
}

// TestFailure describes a single failed test phase.
type TestFailure struct {
	Phase          string `json:"phase"`
	Error          string `json:"error,omitempty"`
	Classification string `json:"classification,omitempty"`
	Remediation    string `json:"remediation,omitempty"`
}

// StandardsDimension summarizes scenario-auditor results.
type StandardsDimension struct {
	Available          bool                       `json:"available"`
	BlockingViolations int                        `json:"blockingViolations"`
	Warnings           int                        `json:"warnings"`
	TotalViolations    int                        `json:"totalViolations"`
	TopViolations      []StandardsViolationDetail `json:"topViolations,omitempty"`
}

// StandardsViolationDetail is a single standards violation for top-K display.
type StandardsViolationDetail struct {
	FilePath       string `json:"filePath"`
	LineNumber     int    `json:"lineNumber"`
	Title          string `json:"title"`
	Severity       string `json:"severity"`
	Recommendation string `json:"recommendation,omitempty"`
}

// VisualDimension summarizes visual capture data.
type VisualDimension struct {
	Available       bool               `json:"available"`
	ScreenshotCount int                `json:"screenshotCount"`
	Stale           bool               `json:"stale"`
	LatestCapture   *VisualCaptureMeta `json:"latestCapture,omitempty"`
}

// VisualCaptureMeta describes the most recent complete visual capture set.
type VisualCaptureMeta struct {
	CapturedAt      string `json:"capturedAt"`
	CommitHash      string `json:"commitHash,omitempty"`
	ScreenshotCount int    `json:"screenshotCount"`
}

// ProvenanceDimension summarizes AI provenance data.
type ProvenanceDimension struct {
	Available     bool     `json:"available"`
	TracedFiles   int      `json:"tracedFiles"`
	UntracedFiles []string `json:"untracedFiles,omitempty"`
}

// CheckStatus represents the status of an individual review check.
type CheckStatus string

const (
	CheckPending   CheckStatus = "pending"
	CheckRunning   CheckStatus = "running"
	CheckCompleted CheckStatus = "completed"
	CheckFailed    CheckStatus = "failed"
	CheckSkipped   CheckStatus = "skipped"
)

// ReviewRunRequest is the request body for POST /api/v1/review/run.
type ReviewRunRequest struct {
	ScenarioName  string               `json:"scenarioName"`
	Checks        []string             `json:"checks,omitempty"`
	ExpectedPaths []string             `json:"expectedPaths,omitempty"`
	SandboxID     string               `json:"sandboxId,omitempty"`
	Details       int                  `json:"details,omitempty"`
	Thresholds    *ReadinessThresholds `json:"thresholds,omitempty"`
}

// ReviewRunResponse is the immediate response from starting a review run.
type ReviewRunResponse struct {
	JobID string `json:"jobId"`
}

// ReviewJobStatus tracks the progress and result of a review run.
type ReviewJobStatus struct {
	JobID     string                 `json:"jobId"`
	Status    string                 `json:"status"`
	Checks    map[string]CheckStatus `json:"checks"`
	Summary   *ReviewSummaryResponse `json:"summary,omitempty"`
	StartedAt string                 `json:"startedAt"`
	Error     string                 `json:"error,omitempty"`
}
