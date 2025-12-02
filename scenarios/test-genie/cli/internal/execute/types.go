// Package execute provides shared execution types used across the execute domain.
package execute

// Request represents an execution request.
type Request struct {
	ScenarioName   string   `json:"scenarioName"`
	Preset         string   `json:"preset,omitempty"`
	Phases         []string `json:"phases,omitempty"`
	Skip           []string `json:"skip,omitempty"`
	FailFast       bool     `json:"failFast"`
	SuiteRequestID string   `json:"suiteRequestId,omitempty"`
}

// Response represents the execution response.
type Response struct {
	Success       bool           `json:"success"`
	ExecutionID   string         `json:"executionId"`
	SuiteRequest  string         `json:"suiteRequestId"`
	PresetUsed    string         `json:"presetUsed"`
	StartedAt     string         `json:"startedAt"`
	CompletedAt   string         `json:"completedAt"`
	PhaseSummary  PhaseSummary   `json:"phaseSummary"`
	Phases        []Phase        `json:"phases"`
	Error         string         `json:"error"`
	ErrorMessages []string       `json:"errors"`
	Links         map[string]any `json:"links"`
	Metadata      map[string]any `json:"metadata"`
}

// Phase represents a single execution phase result.
type Phase struct {
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	DurationSeconds float64  `json:"durationSeconds"`
	LogPath         string   `json:"logPath"`
	Error           string   `json:"error"`
	Classification  string   `json:"classification"`
	Remediation     string   `json:"remediation"`
	Observations    []string `json:"observations"`
}

// PhaseSummary provides aggregate phase statistics.
type PhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}
