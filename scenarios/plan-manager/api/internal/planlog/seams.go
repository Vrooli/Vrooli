package planlog

import "context"

// Resolver maps a `plan_or_execution` handle to the canonical plan id and (when
// the handle was an execution) the execution id, so a ledger entry binds to the
// right scope. Production wraps the plans Service (slug→id) and the execution
// store (execution id → plan id); tests inject a fake. A nil Resolver degrades
// to treating the handle verbatim as a plan id (no execution binding).
type Resolver interface {
	// Resolve returns (planID, executionID). ok=false when the handle resolves to
	// neither a known plan nor a known execution.
	Resolve(ctx context.Context, planOrExecution string) (planID, executionID string, ok bool, err error)
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

// --- default (stubbed) downstream sinks ---------------------------------------
//
// v1 ships pending stubs mirroring the documented VelocitySink/MoM pattern: the
// local entry is durable and the agent retries with `plan-manager log sync` once
// an operator wires a live downstream. The concrete production adapters (a
// scenario-qa bug filer and a `swarm-manager records create` writer) land behind
// these same seams as a follow-up — exactly like the deferred MoM emit on
// VelocitySink — so the live wire never changes the call sites or the
// durable-local + retryable contract.
//
// TODO(downstream): wire the real scenario-qa BugReporter and swarm-manager
// RecordWriter here once their ingest contracts are pinned. Until then the
// defaults keep bug/record entries PENDING and retryable.

// pendingBugReporter is the default BugReporter: it never reaches a downstream,
// signalling unavailability so the entry stays PENDING and retryable.
type pendingBugReporter struct{}

// DefaultBugReporter returns the documented pending stub.
func DefaultBugReporter() BugReporter { return pendingBugReporter{} }

func (pendingBugReporter) FileBug(context.Context, Entry) (DownstreamRef, error) {
	return DownstreamRef{System: "scenario-qa", Kind: "bug_report"}, ErrDownstreamUnavailable{
		System: "scenario-qa",
		Reason: "downstream bug forwarding is not wired; retry with `plan-manager log sync` once configured",
	}
}

// pendingRecordWriter is the default RecordWriter (pending stub).
type pendingRecordWriter struct{}

// DefaultRecordWriter returns the documented pending stub.
func DefaultRecordWriter() RecordWriter { return pendingRecordWriter{} }

func (pendingRecordWriter) WriteRecord(context.Context, Entry) (DownstreamRef, error) {
	return DownstreamRef{System: "swarm-manager", Kind: "record"}, ErrDownstreamUnavailable{
		System: "swarm-manager",
		Reason: "downstream record forwarding is not wired; retry with `plan-manager log sync` once configured",
	}
}
