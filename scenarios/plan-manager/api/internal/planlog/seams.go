package planlog

import "context"

// PhaseRef is the small phase view the log domain needs for phase flag
// normalization without importing the plans domain.
type PhaseRef struct {
	ID    string
	Order int
	Title string
}

// Scope is the resolved plan/execution context for a log entry.
type Scope struct {
	PlanID         string
	ExecutionID    string
	CurrentPhaseID string
	Phases         []PhaseRef
}

// Resolver maps a `plan_or_execution` handle to the canonical plan id and (when
// the handle was an execution) the execution id/current phase, so a ledger entry
// binds to the right scope. Production wraps the plans Service (slug→id) and the
// execution store (execution id → plan id/current phase); tests inject a fake. A
// nil Resolver degrades to treating the handle verbatim as a plan id (no
// execution binding).
type Resolver interface {
	// Resolve returns the canonical scope. ok=false when the handle resolves to
	// neither a known plan nor a known execution.
	Resolve(ctx context.Context, planOrExecution string) (Scope, bool, error)
}

// BugReporter forwards a bug_report entry to the downstream issue tracker
// (scenario-qa). Production wraps the approved bug-filing mechanism; tests inject
// a fake. A failed forward is NEVER fatal — the service keeps the local entry and
// marks it sync_failed/pending for retry via SyncEntry.
type BugReporter interface {
	FileBug(ctx context.Context, entry Entry) (DownstreamRef, error)
}

// RecordWriter forwards a record entry to Swarm Manager records. Same graceful
// degradation contract as BugReporter.
type RecordWriter interface {
	WriteRecord(ctx context.Context, entry Entry) (DownstreamRef, error)
}

// --- explicit degraded downstream sinks --------------------------------------
//
// Production module wiring installs live scenario-qa and swarm-manager adapters.
// These explicit pending sinks remain the degraded/test fallback: the local
// entry is durable and the agent retries with `plan-manager log sync` once the
// downstream is reachable.

// pendingBugReporter is the default BugReporter: it never reaches a downstream,
// signalling unavailability so the entry stays PENDING and retryable.
type pendingBugReporter struct{}

// DefaultBugReporter returns the explicit pending fallback.
func DefaultBugReporter() BugReporter { return pendingBugReporter{} }

func (pendingBugReporter) FileBug(context.Context, Entry) (DownstreamRef, error) {
	return DownstreamRef{System: "scenario-qa", Kind: "bug_report"}, ErrDownstreamUnavailable{
		System: "scenario-qa",
		Reason: "downstream bug forwarding is not wired; retry with `plan-manager log sync` once configured",
	}
}

// pendingRecordWriter is the explicit pending RecordWriter fallback.
type pendingRecordWriter struct{}

// DefaultRecordWriter returns the explicit pending fallback.
func DefaultRecordWriter() RecordWriter { return pendingRecordWriter{} }

func (pendingRecordWriter) WriteRecord(context.Context, Entry) (DownstreamRef, error) {
	return DownstreamRef{System: "swarm-manager", Kind: "record"}, ErrDownstreamUnavailable{
		System: "swarm-manager",
		Reason: "downstream record forwarding is not wired; retry with `plan-manager log sync` once configured",
	}
}
