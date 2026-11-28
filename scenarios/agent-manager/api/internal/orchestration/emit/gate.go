// Package emit centralizes run-event emission through a single choke point.
//
// Every event a runner or phase publishes flows through Gate. Today the
// wrapped sink owns persistence and broadcast policy; tomorrow's invariants
// (dedupe, ordering, throttling, backpressure, audit hooks) attach here so
// the codebase has one obvious place to find — and one obvious place to
// change — emission policy.
//
// Contract:
//   - RunExecutor instantiates exactly one Gate per run.
//   - Phases consume only *Gate; never raw runner.EventSink.
//   - The Gate's external interface is identical to runner.EventSink so it
//     can be passed through existing seams (runner.Execute, recovery tail).
//
// DOC: scenarios/agent-manager/docs/concepts/ARCHITECTURE.md
// (invariant 1: "emit.Gate is the single Emit choke point").
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md (Gate seam table).
package emit

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// Gate is the single emission choke point for a run.
//
// A nil *Gate is a valid no-op sink so callers don't need defensive checks
// during construction; this matches the noOpEventSink fallback used when
// no broadcaster or store is wired up.
type Gate struct {
	sink runner.EventSink
}

// NewGate wraps an EventSink.
//
// Pass nil to construct a no-op gate (events are silently discarded). The
// no-op shape is used for code paths without a broadcaster or event store
// — production runs always supply a real sink.
func NewGate(sink runner.EventSink) *Gate {
	return &Gate{sink: sink}
}

// Emit forwards the event to the wrapped sink. The wrapped sink's
// persist-before-broadcast contract is preserved as-is in Phase 1; future
// invariants (dedupe by event ID, ordering guarantees) attach here.
//
// Typed-operational events (the eventlog package's payloads) flow through
// this same method — the Gate is event-shape-agnostic. The eventlog
// package is the source of truth for "what the event_type/schema_version
// pair means"; the Gate's job is purely transport.
func (g *Gate) Emit(event *domain.RunEvent) error {
	if g == nil || g.sink == nil {
		return nil
	}
	return g.sink.Emit(event)
}

// Close releases the wrapped sink's resources.
func (g *Gate) Close() error {
	if g == nil || g.sink == nil {
		return nil
	}
	return g.sink.Close()
}

// LastSequence forwards to a SequencedEventSink if the wrapped sink supports
// it, returning 0 otherwise. This keeps the recovery tailer (which uses
// LastSequence to advance TranscriptCursor) compatible with gate-wrapped sinks.
func (g *Gate) LastSequence() int64 {
	if g == nil || g.sink == nil {
		return 0
	}
	if seq, ok := g.sink.(runner.SequencedEventSink); ok {
		return seq.LastSequence()
	}
	return 0
}

// Sink returns the wrapped sink. Provided for the limited set of internal
// callers (e.g. recovery tail) that still consume the runner.EventSink
// interface directly during the migration. Phase 6 removes this.
func (g *Gate) Sink() runner.EventSink {
	if g == nil {
		return nil
	}
	return g.sink
}
