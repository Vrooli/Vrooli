package review

type summaryResponse struct {
	ScenarioName string          `json:"scenarioName"`
	Readiness    string          `json:"readiness"`
	Dimensions   dimensions      `json:"dimensions"`
	Capabilities map[string]bool `json:"capabilities"`
	Timestamp    string          `json:"timestamp"`
}

type dimensions struct {
	CodeQuality *codeQualityDimension `json:"codeQuality,omitempty"`
	Tests       *testsDimension       `json:"tests,omitempty"`
	Standards   *standardsDimension   `json:"standards,omitempty"`
	Visual      *visualDimension      `json:"visual,omitempty"`
	Provenance  *provenanceDimension  `json:"provenance,omitempty"`
}

type codeQualityIssue struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type codeQualityDimension struct {
	Available  bool               `json:"available"`
	Score      float64            `json:"score"`
	Violations int                `json:"violations"`
	Stale      bool               `json:"stale"`
	LastScan   string             `json:"lastScan,omitempty"`
	TopIssues  []codeQualityIssue `json:"topIssues,omitempty"`
}

type testFailure struct {
	Phase          string `json:"phase"`
	Error          string `json:"error,omitempty"`
	Classification string `json:"classification,omitempty"`
	Remediation    string `json:"remediation,omitempty"`
}

type testsDimension struct {
	Available   bool          `json:"available"`
	Passed      bool          `json:"passed"`
	Total       int           `json:"total"`
	PassedCount int           `json:"passedCount"`
	FailedCount int           `json:"failedCount"`
	LastRun     string        `json:"lastRun,omitempty"`
	Failures    []testFailure `json:"failures,omitempty"`
}

type standardsViolationDetail struct {
	FilePath       string `json:"filePath"`
	LineNumber     int    `json:"lineNumber"`
	Title          string `json:"title"`
	Severity       string `json:"severity"`
	Recommendation string `json:"recommendation,omitempty"`
}

type standardsDimension struct {
	Available          bool                       `json:"available"`
	BlockingViolations int                        `json:"blockingViolations"`
	Warnings           int                        `json:"warnings"`
	TotalViolations    int                        `json:"totalViolations"`
	TopViolations      []standardsViolationDetail `json:"topViolations,omitempty"`
}

type visualCaptureMeta struct {
	CapturedAt      string `json:"capturedAt"`
	CommitHash      string `json:"commitHash,omitempty"`
	ScreenshotCount int    `json:"screenshotCount"`
}

type visualDimension struct {
	Available       bool               `json:"available"`
	ScreenshotCount int                `json:"screenshotCount"`
	Stale           bool               `json:"stale"`
	LatestCapture   *visualCaptureMeta `json:"latestCapture,omitempty"`
}

type provenanceDimension struct {
	Available     bool     `json:"available"`
	TracedFiles   int      `json:"tracedFiles"`
	UntracedFiles []string `json:"untracedFiles,omitempty"`
}

type runRequest struct {
	ScenarioName string   `json:"scenarioName"`
	Checks       []string `json:"checks,omitempty"`
	Details      int      `json:"details,omitempty"`
}

type runResponse struct {
	JobID string `json:"jobId"`
}

type jobStatusResponse struct {
	JobID     string            `json:"jobId"`
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks"`
	Summary   *summaryResponse  `json:"summary,omitempty"`
	StartedAt string            `json:"startedAt"`
	Error     string            `json:"error,omitempty"`
}
