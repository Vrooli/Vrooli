// Package sandboxprovenance defines the cross-scenario contract for
// per-run provenance records produced by workspace-sandbox and consumed by
// agent-manager, web-console, and Git Control Tower.
//
// The package is the source of truth for field names, allowed values, and
// the schema version. Both the writer (workspace-sandbox apply-at-run-end)
// and the readers (GCT pending-AI hardening, web-console UI, agent-manager
// run detail) import from here so a coordinated bump catches drift at
// compile time rather than silently miswiring fields.
//
// Schema versioning: SchemaVersion is bumped only via a coordination commit
// touching every consumer at once. Until that happens, all consumers MUST
// produce/accept SchemaVersion 1.0.0 records. Readers that encounter a
// future version MUST fail loud (do not best-effort interpret).
package sandboxprovenance

import (
	"errors"
	"fmt"
)

// SchemaVersion is the canonical version for the cross-scenario provenance
// record contract. Coordinated with
// scenarios/swarm-manager/execute/gct-pending-ai-provenance-hardening.
const SchemaVersion = "1.0.0"

// Field name constants — the wire keys used in JSON payloads and as DB
// column-name basis. Both writer and readers reference these constants
// instead of literal strings so a rename catches at compile time.
const (
	FieldRunOutcome     = "runOutcome"
	FieldState          = "state"
	FieldConversationID = "conversationId"
	FieldCostUSD        = "costUsd"
	FieldSchemaVersion  = "schemaVersion"
)

// RunOutcome enumerates the terminal outcomes an agent run can reach.
// Empty is allowed for legacy records written before the auditability
// rollout.
type RunOutcome string

const (
	RunOutcomeSuccess   RunOutcome = "success"
	RunOutcomeFailure   RunOutcome = "failure"
	RunOutcomeCancelled RunOutcome = "cancelled"
	RunOutcomeTimeout   RunOutcome = "timeout"
)

// IsValid reports whether o is a recognised outcome (empty allowed).
func (o RunOutcome) IsValid() bool {
	switch o {
	case "", RunOutcomeSuccess, RunOutcomeFailure, RunOutcomeCancelled, RunOutcomeTimeout:
		return true
	}
	return false
}

// FileState enumerates per-file lifecycle states. Empty is allowed for
// legacy records.
type FileState string

const (
	FileStateApplied       FileState = "applied"
	FileStatePendingReview FileState = "pending-review"
	FileStateDenied        FileState = "denied"
)

// IsValid reports whether s is a recognised state (empty allowed).
func (s FileState) IsValid() bool {
	switch s {
	case "", FileStateApplied, FileStatePendingReview, FileStateDenied:
		return true
	}
	return false
}

// Record is the canonical provenance record shape exchanged between
// workspace-sandbox and its consumers. Construct one of these on the
// writer side and validate before persisting; the reader side decodes the
// same shape and enforces the same invariants.
type Record struct {
	SchemaVersion  string     `json:"schemaVersion"`
	RunID          string     `json:"runId"`
	RunOutcome     RunOutcome `json:"runOutcome,omitempty"`
	State          FileState  `json:"state,omitempty"`
	ConversationID string     `json:"conversationId,omitempty"`
	CostUSD        float64    `json:"costUsd,omitempty"`
}

// ErrUnknownSchemaVersion is returned by Validate when the SchemaVersion
// field is non-empty and does not match this package's known version.
// Readers MUST fail loud on this error rather than best-effort interpret.
var ErrUnknownSchemaVersion = errors.New("sandbox-provenance: unknown schema version")

// Validate enforces the v1.0.0 invariants:
//   - SchemaVersion either matches SchemaVersion or is empty (legacy).
//   - RunID is non-empty.
//   - RunOutcome ∈ {empty, success, failure, cancelled, timeout}.
//   - State ∈ {empty, applied, pending-review, denied}.
//   - CostUSD ≥ 0.
func (r Record) Validate() error {
	if r.SchemaVersion != "" && r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: got %q, want %q", ErrUnknownSchemaVersion, r.SchemaVersion, SchemaVersion)
	}
	if r.RunID == "" {
		return errors.New("sandbox-provenance: runId is required")
	}
	if !r.RunOutcome.IsValid() {
		return fmt.Errorf("sandbox-provenance: invalid runOutcome %q", r.RunOutcome)
	}
	if !r.State.IsValid() {
		return fmt.Errorf("sandbox-provenance: invalid state %q", r.State)
	}
	if r.CostUSD < 0 {
		return fmt.Errorf("sandbox-provenance: costUsd must be ≥ 0, got %v", r.CostUSD)
	}
	return nil
}
