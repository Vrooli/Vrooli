// Package status provides health and status monitoring for Test Genie.
package status

// Response represents the health check response from the API.
type Response struct {
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

// ExecutionSummary provides a brief summary of the last execution.
type ExecutionSummary struct {
	Scenario     string `json:"scenario"`
	Success      bool   `json:"success"`
	CompletedAt  string `json:"completedAt"`
	PhaseSummary struct {
		Total  int64 `json:"total"`
		Failed int64 `json:"failed"`
	} `json:"phaseSummary"`
}
