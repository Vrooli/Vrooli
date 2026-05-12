// Package runs persists verification history in SQLite. Repository writes
// one row per verify run/check; service exposes list and lookup. Flows
// are filesystem-truth; this domain owns the trail of verifying them.
package runs

import "time"

// Status enumerates the three terminal states of a verification of a
// single flow.
//
//   - "passed"  — pipeline produced/checked artifacts cleanly.
//   - "failed"  — pipeline ran but produced a counterexample or staleness.
//   - "error"   — pipeline could not run (e.g. quint missing, IO failure).
type Status string

const (
	StatusPassed Status = "passed"
	StatusFailed Status = "failed"
	StatusError  Status = "error"
)

// Mode mirrors pipeline.Mode for storage. Recorded so the history can
// distinguish "check" runs from "regenerate" runs at a glance.
type Mode string

const (
	ModeRun   Mode = "run"
	ModeCheck Mode = "check"
)

// Run is one persisted verification of one flow. The triple of hashes
// (source/model/gen) is reserved for Phase F+ — Phase E records the
// minimum the UI needs (id, flow, status, output, timings).
//
// FailureReason narrows a "failed" status into a typed category so the
// UI can render a distinct, recoverable state. MissingArtifacts holds
// the per-flow artifact paths reported by *pipeline.FreshnessError —
// the UI's "Needs generate" affordance lists them and offers a
// one-click generate. Both fields are zero-valued for passing runs.
type Run struct {
	ID               string    `json:"id"`
	FlowID           string    `json:"flowId"`
	FlowPath         string    `json:"flowPath"`
	Root             string    `json:"root"`
	SourceSHA256     string    `json:"sourceSha256,omitempty"`
	ModelSHA256      string    `json:"modelSha256,omitempty"`
	GenSHA256        string    `json:"genSha256,omitempty"`
	Mode             Mode      `json:"mode"`
	Status           Status    `json:"status"`
	Counterexample   string    `json:"counterexample,omitempty"`
	ErrorMessage     string    `json:"errorMessage,omitempty"`
	FailureReason    string    `json:"failureReason,omitempty"`
	MissingArtifacts []string  `json:"missingArtifacts,omitempty"`
	Output           string    `json:"output,omitempty"`
	StartedAt        time.Time `json:"startedAt"`
	FinishedAt       time.Time `json:"finishedAt"`
	DurationMs       int64     `json:"durationMs"`
}

// ErrNotFound is returned by Repository.Get when no row matches the id.
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return "verification run not found: " + e.ID }
