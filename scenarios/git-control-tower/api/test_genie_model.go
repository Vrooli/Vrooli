package main

// TestExecutionRequest mirrors POST /api/v1/executions body in test-genie.
type TestExecutionRequest struct {
	ScenarioName string   `json:"scenarioName"`
	Preset       string   `json:"preset,omitempty"`
	Phases       []string `json:"phases,omitempty"`
	Skip         []string `json:"skip,omitempty"`
	FailFast     bool     `json:"failFast,omitempty"`
}

// TestExecutionResult mirrors test-genie SuiteExecutionResult.
type TestExecutionResult struct {
	ExecutionID  string            `json:"executionId"`
	ScenarioName string            `json:"scenarioName"`
	Success      bool              `json:"success"`
	StartedAt    string            `json:"startedAt"`
	CompletedAt  string            `json:"completedAt,omitempty"`
	PresetUsed   string            `json:"preset,omitempty"`
	Phases       []TestPhaseResult `json:"phases"`
	PhaseSummary TestPhaseSummary  `json:"phaseSummary"`
	Warnings     []string          `json:"warnings,omitempty"`
}

// TestPhaseResult represents one phase of a test execution.
type TestPhaseResult struct {
	Name            string            `json:"name"`
	Status          string            `json:"status"` // "passed" | "failed"
	DurationSeconds int               `json:"durationSeconds"`
	LogPath         string            `json:"logPath,omitempty"`
	Error           string            `json:"error,omitempty"`
	Classification  string            `json:"classification,omitempty"`
	Remediation     string            `json:"remediation,omitempty"`
	Observations    []TestObservation `json:"observations,omitempty"`
}

// TestPhaseSummary aggregates phase results.
type TestPhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}

// TestObservation is a single observation within a phase.
type TestObservation struct {
	Icon    string `json:"icon,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	Section string `json:"section,omitempty"`
	Text    string `json:"text"`
}

// TestExecutionListResponse wraps a list of test executions.
type TestExecutionListResponse struct {
	Items []TestExecutionResult `json:"items"`
	Count int                   `json:"count"`
}
