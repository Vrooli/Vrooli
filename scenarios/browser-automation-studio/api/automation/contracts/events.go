// Package contracts defines the event and telemetry types for workflow execution.
//
// # Signal Surface Overview
//
// Browser Automation Studio emits structured signals to enable observability,
// debugging, and automation. Signals fall into three categories:
//
// ## 1. Execution Lifecycle Events (EventKind)
//
// These events track workflow execution state transitions:
//
//   - execution.started: Workflow begins running (status: pending → running)
//   - execution.progress: Periodic progress updates during execution
//   - execution.completed: Workflow finished successfully (status: running → completed)
//   - execution.failed: Workflow terminated with an error (status: running → failed)
//   - execution.cancelled: User cancelled the execution (status: running → cancelled)
//   - execution.interrupted: Recovered after crash/restart (status: running → interrupted)
//
// ## 2. Step-Level Events
//
// These events track individual step execution:
//
//   - step.started: Step begins execution
//   - step.completed: Step finished successfully
//   - step.failed: Step terminated with error (may trigger retry)
//   - step.screenshot: Screenshot captured mid-step
//   - step.telemetry: Mid-step telemetry (console, network, heartbeat)
//   - step.heartbeat: Liveness signal during long-running steps
//
// ## 3. Execution Status Values
//
// The database.Execution.Status field uses these values:
//
//   - "pending": Created but not yet started
//   - "running": Currently executing steps
//   - "completed": All steps finished successfully
//   - "failed": Stopped due to error
//   - "cancelled": User-initiated cancellation
//   - "interrupted": Recovered after service restart (resumable if checkpoint exists)
//
// ## 4. Health & Telemetry
//
// The /health endpoint exposes operational metrics:
//
//   - goroutines: Active goroutine count
//   - uptime_seconds: Time since service start
//   - heap_alloc_mb: Current heap allocation
//   - gc_pause_total_ms: Total GC pause time
//
// Dependencies health checks include database, storage, and automation engine.
//
// ## Signal Stability Guarantees
//
// Event kinds and status values are stable identifiers. Changes to these values
// constitute breaking changes. Payload shapes within events may evolve with
// schema_version bumps.
package contracts

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// EventKind enumerates structured events emitted by the executor/recorder layer.
// These values are stable identifiers; changes are breaking.
type EventKind string

const (
	EventKindExecutionStarted     EventKind = "execution.started"
	EventKindExecutionProgress    EventKind = "execution.progress"
	EventKindExecutionCompleted   EventKind = "execution.completed"
	EventKindExecutionFailed      EventKind = "execution.failed"
	EventKindExecutionCancelled   EventKind = "execution.cancelled"
	EventKindExecutionInterrupted EventKind = "execution.interrupted" // Set when recovered after crash/restart
	EventKindStepStarted          EventKind = "step.started"
	EventKindStepCompleted        EventKind = "step.completed"
	EventKindStepFailed           EventKind = "step.failed"
	EventKindStepScreenshot       EventKind = "step.screenshot"
	EventKindStepTelemetry        EventKind = "step.telemetry"
	EventKindStepHeartbeat        EventKind = "step.heartbeat"
)

// ExecutionStatus defines the canonical status values for workflow executions.
// These are stored in database.Execution.Status and emitted in event payloads.
// Use these constants rather than string literals to ensure consistency.
type ExecutionStatus string

const (
	ExecutionStatusPending     ExecutionStatus = "pending"
	ExecutionStatusRunning     ExecutionStatus = "running"
	ExecutionStatusCompleted   ExecutionStatus = "completed"
	ExecutionStatusFailed      ExecutionStatus = "failed"
	ExecutionStatusCancelled   ExecutionStatus = "cancelled"
	ExecutionStatusInterrupted ExecutionStatus = "interrupted"
)

// IsTerminal returns true if the status represents a final execution state.
func (s ExecutionStatus) IsTerminal() bool {
	switch s {
	case ExecutionStatusCompleted, ExecutionStatusFailed, ExecutionStatusCancelled, ExecutionStatusInterrupted:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
func (s ExecutionStatus) String() string {
	return string(s)
}

// EventEnvelope wraps payloads with ordering/backpressure metadata. Sequence is
// monotonic per execution; per-attempt ordering is enforced by EventSink
// without altering payload shapes consumed by WebSocket clients.
type EventEnvelope struct {
	SchemaVersion  string       `json:"schema_version"`
	PayloadVersion string       `json:"payload_version"`
	Kind           EventKind    `json:"kind"`
	ExecutionID    uuid.UUID    `json:"execution_id"`
	WorkflowID     uuid.UUID    `json:"workflow_id"`
	StepIndex      *int         `json:"step_index,omitempty"`
	Attempt        *int         `json:"attempt,omitempty"`
	Sequence       uint64       `json:"sequence,omitempty"` // Monotonic per execution.
	Timestamp      time.Time    `json:"timestamp"`
	Payload        any          `json:"payload,omitempty"`
	Drops          DropCounters `json:"drops,omitempty"` // Populated when telemetry is dropped due to backpressure.
}

// DropCounters surface how many telemetry/heartbeat events were dropped under
// backpressure. Completion/failure events must never be dropped.
type DropCounters struct {
	Dropped       uint64 `json:"dropped,omitempty"`
	OldestDropped uint64 `json:"oldest_dropped,omitempty"`
}

// EventBufferLimits define per-execution/per-attempt queue sizing used by
// EventSink backpressure handling.
type EventBufferLimits struct {
	PerExecution int
	PerAttempt   int
}

// DefaultEventBufferLimits align with the refactor plan: keep bounded buffers
// while guaranteeing that completion/failure events are not dropped.
var DefaultEventBufferLimits = EventBufferLimits{
	PerExecution: 200,
	PerAttempt:   50,
}

// Validate ensures configured queue sizes are sane.
func (l EventBufferLimits) Validate() error {
	if l.PerExecution <= 0 {
		return errors.New("per-execution buffer must be positive")
	}
	if l.PerAttempt <= 0 {
		return errors.New("per-attempt buffer must be positive")
	}
	return nil
}

// TelemetryKind describes the flavor of mid-step telemetry emitted before a
// StepOutcome is available.
type TelemetryKind string

const (
	TelemetryKindHeartbeat TelemetryKind = "heartbeat"
	TelemetryKindConsole   TelemetryKind = "console"
	TelemetryKindNetwork   TelemetryKind = "network"
	TelemetryKindRetry     TelemetryKind = "retry"
	TelemetryKindProgress  TelemetryKind = "progress"
)

// StepTelemetry carries per-attempt telemetry emitted while a step is inflight.
// Sequence ordering is scoped to the attempt so ordering can be enforced
// independently from the execution-level sequence used in EventEnvelope.
type StepTelemetry struct {
	SchemaVersion  string              `json:"schema_version"`
	PayloadVersion string              `json:"payload_version"`
	ExecutionID    uuid.UUID           `json:"execution_id,omitempty"`
	StepIndex      int                 `json:"step_index"`
	Attempt        int                 `json:"attempt"`
	Kind           TelemetryKind       `json:"kind"`
	Sequence       uint64              `json:"sequence,omitempty"`   // Monotonic per attempt.
	Timestamp      time.Time           `json:"timestamp"`            // UTC
	ElapsedMs      int64               `json:"elapsed_ms,omitempty"` // Relative to step start.
	Heartbeat      *HeartbeatTelemetry `json:"heartbeat,omitempty"`
	Console        []ConsoleLogEntry   `json:"console,omitempty"` // Incremental console lines.
	Network        []NetworkEvent      `json:"network,omitempty"` // Incremental network entries.
	Retry          *RetryTelemetry     `json:"retry,omitempty"`   // Retry notices.
	Note           string              `json:"note,omitempty"`
}

// HeartbeatTelemetry conveys liveness while a step attempt is in-flight.
type HeartbeatTelemetry struct {
	Progress int    `json:"progress,omitempty"` // 0-100, executor-defined meaning.
	Message  string `json:"message,omitempty"`
}

// RetryTelemetry surfaces retry metadata without waiting for the final outcome.
type RetryTelemetry struct {
	Attempt     int          `json:"attempt"`                // Current attempt number (1-indexed).
	MaxAttempts int          `json:"max_attempts,omitempty"` // Max attempts allowed for the instruction.
	Reason      *StepFailure `json:"reason,omitempty"`       // Failure prompting the retry.
}
