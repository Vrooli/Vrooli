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
	ScenarioName string           `json:"scenarioName"`
	Readiness    Readiness        `json:"readiness"`
	Dimensions   ReviewDimensions `json:"dimensions"`
	Capabilities map[string]bool  `json:"capabilities"`
	Timestamp    string           `json:"timestamp"`
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
	Available  bool    `json:"available"`
	Score      float64 `json:"score"`
	Violations int     `json:"violations"`
	Stale      bool    `json:"stale"`
	LastScan   string  `json:"lastScan,omitempty"`
}

// TestsDimension summarizes test-genie results.
type TestsDimension struct {
	Available   bool   `json:"available"`
	Passed      bool   `json:"passed"`
	Total       int    `json:"total"`
	PassedCount int    `json:"passedCount"`
	FailedCount int    `json:"failedCount"`
	LastRun     string `json:"lastRun,omitempty"`
}

// StandardsDimension summarizes scenario-auditor results.
type StandardsDimension struct {
	Available          bool `json:"available"`
	BlockingViolations int  `json:"blockingViolations"`
	Warnings           int  `json:"warnings"`
	TotalViolations    int  `json:"totalViolations"`
}

// VisualDimension summarizes visual capture data.
type VisualDimension struct {
	Available       bool `json:"available"`
	ScreenshotCount int  `json:"screenshotCount"`
	Stale           bool `json:"stale"`
}

// ProvenanceDimension summarizes AI provenance data.
type ProvenanceDimension struct {
	Available   bool `json:"available"`
	TracedFiles int  `json:"tracedFiles"`
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
	ScenarioName  string   `json:"scenarioName"`
	Checks        []string `json:"checks,omitempty"`
	ExpectedPaths []string `json:"expectedPaths,omitempty"`
	SandboxID     string   `json:"sandboxId,omitempty"`
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
