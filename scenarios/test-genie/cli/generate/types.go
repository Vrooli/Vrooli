// Package generate provides test suite generation capabilities.
package generate

// Request represents a suite generation request.
type Request struct {
	ScenarioName   string   `json:"scenarioName"`
	RequestedTypes []string `json:"requestedTypes,omitempty"`
	CoverageTarget *int     `json:"coverageTarget,omitempty"`
	Priority       string   `json:"priority,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

// Response represents the suite generation response.
type Response struct {
	ID                string   `json:"id"`
	ScenarioName      string   `json:"scenarioName"`
	Status            string   `json:"status"`
	EstimatedQueueSec int64    `json:"estimatedQueueTimeSeconds"`
	RequestedTypes    []string `json:"requestedTypes"`
	CoverageTarget    *int     `json:"coverageTarget"`
	Priority          string   `json:"priority"`
	IssueID           string   `json:"issueId"`
}

// Args holds parsed CLI inputs for the generate command.
type Args struct {
	Scenario  string
	Types     string
	Coverage  int
	Priority  string
	Notes     string
	NotesFile string
	JSON      bool
}
