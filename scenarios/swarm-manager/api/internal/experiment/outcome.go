// Package experiment defines the swarm-manager-owned outcome schema for prompt
// A/B experiment analysis.
package experiment

// OutcomeDataV1 is the swarm-manager-defined payload stored inside the opaque
// experiment outcome data field in prompt-manager.
type OutcomeDataV1 struct {
	ExecutionID    string  `json:"executionId"`
	Classification string  `json:"classification"` // ready|ready_with_notes|needs_work|not_assessable
	BacklogKind    string  `json:"backlogKind"`
	BacklogName    string  `json:"backlogName"`
	Purpose        string  `json:"purpose"` // process|fixup|followup
	HadFixup       bool    `json:"hadFixup"`
	DurationSecs   float64 `json:"durationSecs,omitempty"`
}

// OutcomeSchemaVersion is the current schema version for SM outcome data.
const OutcomeSchemaVersion = 1
