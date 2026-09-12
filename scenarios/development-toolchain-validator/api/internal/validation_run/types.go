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

// ToolResult is the data the tool runner returns. It carries the
// two-layer signal the evaluator needs: did the tool actually run (Ran),
// and — if it ran — was its success expectation met (ExpectationMet).
// The collapse of these into a single boolean is what made the old
// runner unable to distinguish "couldn't run the tool" (run failure)
// from "tool ran and found problems" (tool/template regression).
type ToolResult struct {
	Name string

	// Ran is true when the tool process actually executed — even if it
	// exited non-zero. It is false only when the tool could not be
	// launched at all (binary missing, unknown tool, a required
	// preparatory command failed, timeout before any execution).
	Ran bool
	// ExpectationMet is meaningful only when Ran: true means the tool's
	// per-tool success expectation (all phases pass / score >= floor) held.
	ExpectationMet bool

	// Detail is a human-readable summary of the expectation result
	// (e.g. "all 14 phases passed", "2 phase(s) failed: smoke, unit",
	// "score 92.0 < floor 96").
	Detail string
	// RawOutput is the captured stdout+stderr of the tool invocation,
	// persisted for triage.
	RawOutput []byte

	StartedAt time.Time
	EndedAt   time.Time
	// ErrorReason explains why the tool could not run, or why its
	// expectation was not met. Empty on a clean pass.
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
