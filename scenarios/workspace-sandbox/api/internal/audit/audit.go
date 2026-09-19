// Package audit owns the seam for recording sandbox audit events.
//
// Why this package exists (Round 4 Phase 6):
//
// Before this seam, three production sites constructed
// `&types.AuditEvent{...}` literals inline and called
// `repo.LogAuditEvent` directly:
//
//   - internal/sandbox/service_audit.go::logAuditEventWith — the
//     Service-level path that builds the immutable sandbox-state
//     snapshot for forensic analysis.
//   - internal/sandbox/orphan_reconciler.go::logOrphanAuditEvent —
//     the orphan-cleanup path (no Sandbox object available).
//   - internal/gc/gc.go — the GC-collected path.
//
// Each site re-derived event timestamps, default ActorType, and
// error-handling. Tests that wanted to assert audit content scanned
// `FakeRepository.AuditEvents` because there was no shared
// recording surface.
//
// This package consolidates all of that. Production wires
// RepoEmitter, which stamps EventTime via the injected Clock and
// persists through Repository.LogAuditEvent. Tests wire FakeEmitter
// (in testutil/mocks) and assert via assertx.AssertAuditEvents.
//
// The Event struct is the seam contract — callers pass the fields
// they care about; the emitter handles the rest. The
// `&types.AuditEvent{...}` literal lives only inside this package,
// so the Phase 6 DOD invariant
//
//	grep -rn "&types.AuditEvent{" internal/ | grep -v _test.go
//	    | grep -v internal/audit/ | grep -v internal/testutil/
//
// returns zero hits.
package audit

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"

	"github.com/vrooli/api-core/schedule"
)

// Event is the input shape for Emitter.Emit. It mirrors the subset
// of types.AuditEvent that callers populate; ID and EventTime are
// assigned by the emitter (or by the underlying repository, which
// stamps them as a safety net).
//
// All fields except EventType are optional. ActorType defaults to
// "system" if left empty — preserving the legacy default that
// reconcilers and GC paths relied on.
type Event struct {
	// EventType identifies the operation (e.g. "created", "approved",
	// "gc_collected", "sandbox.orphan-cleaned"). Required.
	EventType string

	// SandboxID identifies the sandbox the event is about. Pointer so
	// orphan paths (where the sandbox isn't in the repo by definition)
	// can pass a non-nil id while system-wide events can pass nil.
	SandboxID *uuid.UUID

	// Actor is the principal that triggered the event ("system",
	// "user@x.com", "agent-manager", etc.). Empty is allowed and
	// surfaces as NULL on the wire.
	Actor string

	// ActorType categorizes Actor ("user", "system", "agent",
	// "gc"). Empty defaults to "system".
	ActorType string

	// Source identifies the originating approval surface — see
	// types.ApprovalSource. Most call sites leave this as
	// SourceUnspecified.
	Source types.ApprovalSource

	// Details is the free-form payload. nil is safe; callers that
	// pass nil get an empty JSON object on disk.
	Details map[string]interface{}

	// SandboxState is the immutable snapshot of the sandbox at event
	// time. Populated by service_audit.logAuditEventWith for
	// Service-level events; nil for orphan / GC-only events.
	SandboxState map[string]interface{}
}

// Emitter is the seam for recording audit events. Production wires
// RepoEmitter; tests wire testutil/mocks.FakeEmitter.
//
// Implementations must:
//
//   - Stamp EventTime from an authoritative schedule.
//   - Default ActorType to "system" when the input leaves it empty.
//   - Propagate the underlying persistence error to the caller. The
//     audit policy (log-and-continue vs return-to-caller) is the
//     caller's choice — this contract does NOT swallow errors.
type Emitter interface {
	Emit(ctx context.Context, event Event) error
}

// LogFunc is the narrow contract RepoEmitter wraps. The method-set
// of repository.Repository.LogAuditEvent satisfies it without
// importing the repository package, keeping audit free of the
// repo's compile-time surface.
type LogFunc func(ctx context.Context, event *types.AuditEvent) error

// RepoEmitter is the production Emitter. It builds a
// *types.AuditEvent from the Event input, stamps EventTime via the
// clock, and persists through the supplied LogFunc.
type RepoEmitter struct {
	log   LogFunc
	clock schedule.Clock
}

// NewRepoEmitter wires a RepoEmitter. Both args are required —
// passing nil panics on construction so misconfigured boot paths
// fail loudly instead of silently dropping audit events.
func NewRepoEmitter(log LogFunc, clk schedule.Clock) *RepoEmitter {
	if log == nil {
		panic("audit.NewRepoEmitter: log function is required")
	}
	if clk == nil {
		panic("audit.NewRepoEmitter: clock is required")
	}
	return &RepoEmitter{log: log, clock: clk}
}

// Emit implements Emitter. EventType is the only required field;
// every other field is normalized:
//
//   - ActorType defaults to "system" when empty.
//   - EventTime is stamped from the injected schedule.
//   - ID is stamped here too (the repository would otherwise stamp
//     it on insert; doing it here means the persisted row has a
//     deterministic ID even if the caller wants to read it back
//     before commit, e.g. in a tx).
func (r *RepoEmitter) Emit(ctx context.Context, e Event) error {
	if e.EventType == "" {
		return errors.New("audit.Emit: EventType is required")
	}
	actorType := e.ActorType
	if actorType == "" {
		actorType = "system"
	}
	return r.log(ctx, &types.AuditEvent{
		ID:           uuid.New(),
		SandboxID:    e.SandboxID,
		EventType:    e.EventType,
		EventTime:    r.clock.Now().UTC(),
		Actor:        e.Actor,
		ActorType:    actorType,
		Source:       e.Source,
		Details:      e.Details,
		SandboxState: e.SandboxState,
	})
}

// Compile-time check.
var _ Emitter = (*RepoEmitter)(nil)
