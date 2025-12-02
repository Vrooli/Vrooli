package main

type HealthResponse struct {
	Status       string                 `json:"status"`
	Service      string                 `json:"service"`
	Version      string                 `json:"version"`
	Dependencies map[string]interface{} `json:"dependencies"`
	Operations   struct {
		Queue struct {
			Pending             int64 `json:"pending"`
			Queued              int64 `json:"queued"`
			Delegated           int64 `json:"delegated"`
			Running             int64 `json:"running"`
			Failed              int64 `json:"failed"`
			OldestQueuedAgeSecs int64 `json:"oldestQueuedAgeSeconds"`
		} `json:"queue"`
		LastExecution *ExecutionSummary `json:"lastExecution"`
	} `json:"operations"`
}

type ExecutionSummary struct {
	Scenario     string `json:"scenario"`
	Success      bool   `json:"success"`
	CompletedAt  string `json:"completedAt"`
	PhaseSummary struct {
		Total  int64 `json:"total"`
		Failed int64 `json:"failed"`
	} `json:"phaseSummary"`
}

type GenerateRequest struct {
	ScenarioName   string   `json:"scenarioName"`
	RequestedTypes []string `json:"requestedTypes,omitempty"`
	CoverageTarget *int     `json:"coverageTarget,omitempty"`
	Priority       string   `json:"priority,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

type GenerateResponse struct {
	ID                string   `json:"id"`
	ScenarioName      string   `json:"scenarioName"`
	Status            string   `json:"status"`
	EstimatedQueueSec int64    `json:"estimatedQueueTimeSeconds"`
	RequestedTypes    []string `json:"requestedTypes"`
	CoverageTarget    *int     `json:"coverageTarget"`
	Priority          string   `json:"priority"`
	IssueID           string   `json:"issueId"`
}

type ExecuteRequest struct {
	ScenarioName   string   `json:"scenarioName"`
	Preset         string   `json:"preset,omitempty"`
	Phases         []string `json:"phases,omitempty"`
	Skip           []string `json:"skip,omitempty"`
	FailFast       bool     `json:"failFast"`
	SuiteRequestID string   `json:"suiteRequestId,omitempty"`
}

type ExecuteResponse struct {
	Success       bool           `json:"success"`
	ExecutionID   string         `json:"executionId"`
	SuiteRequest  string         `json:"suiteRequestId"`
	PresetUsed    string         `json:"presetUsed"`
	StartedAt     string         `json:"startedAt"`
	CompletedAt   string         `json:"completedAt"`
	PhaseSummary  PhaseSummary   `json:"phaseSummary"`
	Phases        []ExecutePhase `json:"phases"`
	Error         string         `json:"error"`
	ErrorMessages []string       `json:"errors"`
	Links         map[string]any `json:"links"`
	Metadata      map[string]any `json:"metadata"`
}

type ExecutePhase struct {
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	DurationSeconds float64  `json:"durationSeconds"`
	LogPath         string   `json:"logPath"`
	Error           string   `json:"error"`
	Classification  string   `json:"classification"`
	Remediation     string   `json:"remediation"`
	Observations    []string `json:"observations"`
}

type PhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}

type RunTestsRequest struct {
	Type string `json:"type,omitempty"`
}

type RunTestsResponse struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	LogPath string `json:"logPath"`
	Command struct {
		Command    []string `json:"command"`
		WorkingDir string   `json:"workingDir"`
	} `json:"command"`
}

type PhaseDescriptor struct {
	Name                  string `json:"name"`
	Optional              bool   `json:"optional"`
	Description           string `json:"description"`
	Source                string `json:"source"`
	DefaultTimeoutSeconds int    `json:"defaultTimeoutSeconds"`
}
