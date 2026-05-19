// Package validation_run orchestrates one (skill|tool, golden)
// validation end-to-end (OT-P0-004 + OT-P0-005). Start returns
// immediately with a queued run; an internal worker advances the
// lifecycle and persists a terminal validation_record.
package validation_run

import (
	"fmt"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	vr "development-toolchain-validator/internal/validation_record"
)

// Status enumerates the operational state of a run.
type Status int

const (
	StatusUnspecified Status = 0
	StatusQueued      Status = 1
	StatusRunning     Status = 2
	StatusEvaluating  Status = 3
	StatusTerminal    Status = 4
)

// Run is the domain shape for one validation run.
type Run struct {
	ID         string
	TupleKind  vr.TupleKind
	SubjectID  string
	GoldenSlug string

	Status          Status
	TerminalVerdict vr.Verdict

	AgentManagerRunID string

	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time

	ErrorMessage string

	// ForceReRun is operator-supplied; the worker preserves it on the
	// in-memory run for reasoning (e.g., a future deduper might skip
	// runs that don't carry it).
	ForceReRun bool
}

// StartInput is the explicit DTO Service.Start accepts.
type StartInput struct {
	TupleKind  vr.TupleKind
	SubjectID  string
	GoldenSlug string
	Force      bool
}

// RunSummary is the data the agent-manager adapter returns when a
// sandboxed run reaches terminal state.
type RunSummary struct {
	AgentManagerRunID string
	StartedAt         time.Time
	EndedAt           time.Time
	TokensUsed        int64
	CostUSDMicro      int64
	DiffHash          string
	// DiffPaths is the list of files mutated by the sandboxed run.
	// Used by the evaluator to apply manifest path-globs.
	DiffPaths []manifest.DiffFile
}

// ToolResult is the data the tool runner returns.
type ToolResult struct {
	Name        string
	Succeeded   bool
	StartedAt   time.Time
	EndedAt     time.Time
	ErrorReason string
}

// ErrRunNotFound is the typed sentinel returned when Get can't find
// the run.
type ErrRunNotFound struct {
	ID string
}

func (e ErrRunNotFound) Error() string {
	return fmt.Sprintf("validation run %q not found", e.ID)
}

// ErrInvalidRun is the typed sentinel returned when input validation
// fails.
type ErrInvalidRun struct {
	Field  string
	Reason string
}

func (e ErrInvalidRun) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrDependencyUnavailable is the typed sentinel returned when a
// required outbound dependency (agent-manager, prompt-manager) is not
// running. Handlers translate to CodeUnavailable.
type ErrDependencyUnavailable struct {
	Dependency string
	Reason     string
}

func (e ErrDependencyUnavailable) Error() string {
	return fmt.Sprintf("%s unavailable: %s", e.Dependency, e.Reason)
}
