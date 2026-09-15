package mocks

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/types"

	"github.com/vrooli/api-core/schedule"
)

// FakeEmitter is the canonical audit.Emitter fake. Tests that wire a
// Service or reconciler with a fake emitter (instead of a fake repo)
// can assert on the structured audit-event sequence directly via
// assertx.AssertAuditEvents.
//
// FakeEmitter performs the same normalization the production
// RepoEmitter performs (UUID stamping, EventTime stamping via the
// injected clock, ActorType default), so Events() returns
// *types.AuditEvent values that look exactly like what the
// repository would have stored. Existing assertions written against
// FakeRepository.AuditEvents work unchanged.
type FakeEmitter struct {
	mu     sync.Mutex
	events []*types.AuditEvent

	// EmitErr — when non-nil, every Emit call returns this error
	// after recording the event. Lets tests drive the failure path
	// (e.g. confirm Service swallows audit errors but doesn't
	// return them to the caller).
	EmitErr error

	clock schedule.Clock
}

// NewFakeEmitter constructs a FakeEmitter using the supplied schedule.
// Most tests pass FakeClock so EventTime is deterministic.
func NewFakeEmitter(clk schedule.Clock) *FakeEmitter {
	if clk == nil {
		panic("mocks.NewFakeEmitter: clock is required")
	}
	return &FakeEmitter{clock: clk}
}

// Emit implements audit.Emitter. The recorded event mirrors the
// shape RepoEmitter would have written.
func (f *FakeEmitter) Emit(ctx context.Context, e audit.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	actorType := e.ActorType
	if actorType == "" {
		actorType = "system"
	}
	stored := &types.AuditEvent{
		ID:           uuid.New(),
		SandboxID:    e.SandboxID,
		EventType:    e.EventType,
		EventTime:    f.clock.Now().UTC(),
		Actor:        e.Actor,
		ActorType:    actorType,
		Source:       e.Source,
		Details:      e.Details,
		SandboxState: e.SandboxState,
	}
	f.events = append(f.events, stored)

	if f.EmitErr != nil {
		return f.EmitErr
	}
	return nil
}

// Events returns a snapshot of every recorded event in order. Safe
// to call concurrently with Emit — the returned slice is a copy.
func (f *FakeEmitter) Events() []*types.AuditEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*types.AuditEvent, len(f.events))
	copy(out, f.events)
	return out
}

// Reset clears the recorded events. Useful in table tests where
// each subtest reuses the same emitter.
func (f *FakeEmitter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = nil
}

// Compile-time check.
var _ audit.Emitter = (*FakeEmitter)(nil)
