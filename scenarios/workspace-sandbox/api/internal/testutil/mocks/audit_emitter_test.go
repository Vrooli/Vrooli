package mocks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/types"
)

func TestFakeEmitter_RecordsAndStamps(t *testing.T) {
	frozen := time.Date(2026, 4, 29, 16, 0, 0, 0, time.UTC)
	clk := NewFakeClock(frozen)
	f := NewFakeEmitter(clk)

	id := uuid.New()
	if err := f.Emit(context.Background(), audit.Event{
		EventType: "approved",
		SandboxID: &id,
		Actor:     "alice",
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	events := f.Events()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	got := events[0]
	if got.EventType != "approved" {
		t.Errorf("EventType = %q", got.EventType)
	}
	if got.SandboxID == nil || *got.SandboxID != id {
		t.Errorf("SandboxID = %v, want %v", got.SandboxID, id)
	}
	if got.ActorType != "system" {
		t.Errorf("ActorType = %q, want system (default)", got.ActorType)
	}
	if !got.EventTime.Equal(frozen) {
		t.Errorf("EventTime = %v, want %v", got.EventTime, frozen)
	}
	if got.ID == uuid.Nil {
		t.Errorf("ID was uuid.Nil")
	}
}

func TestFakeEmitter_PropagatesEmitErr(t *testing.T) {
	clk := NewFakeClock(time.Now())
	f := NewFakeEmitter(clk)
	want := errors.New("boom")
	f.EmitErr = want

	err := f.Emit(context.Background(), audit.Event{EventType: "x"})
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
	// The event should still be recorded — tests that check both
	// "did Emit get called?" and "did the caller surface the error?"
	// rely on this.
	if len(f.Events()) != 1 {
		t.Errorf("event was not recorded despite EmitErr")
	}
}

func TestFakeEmitter_Reset(t *testing.T) {
	clk := NewFakeClock(time.Now())
	f := NewFakeEmitter(clk)
	_ = f.Emit(context.Background(), audit.Event{EventType: "a"})
	_ = f.Emit(context.Background(), audit.Event{EventType: "b"})
	if len(f.Events()) != 2 {
		t.Fatalf("setup: expected 2 events")
	}
	f.Reset()
	if len(f.Events()) != 0 {
		t.Errorf("Reset did not clear events")
	}
}

func TestFakeEmitter_EventsReturnsSnapshot(t *testing.T) {
	clk := NewFakeClock(time.Now())
	f := NewFakeEmitter(clk)
	_ = f.Emit(context.Background(), audit.Event{EventType: "a"})
	snap := f.Events()
	_ = f.Emit(context.Background(), audit.Event{EventType: "b"})
	if len(snap) != 1 {
		t.Errorf("snapshot mutated by later Emit; got %d", len(snap))
	}
	if len(f.Events()) != 2 {
		t.Errorf("Events() did not return both, got %d", len(f.Events()))
	}
}

func TestFakeEmitter_NilClockPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when clock is nil")
		}
	}()
	NewFakeEmitter(nil)
}

// Sanity: FakeEmitter's recorded shape matches what FakeRepository's
// LogAuditEvent would have stored, so tests can swap between fakes
// without rewriting their assertions.
func TestFakeEmitter_ShapeMatchesRepo(t *testing.T) {
	clk := NewFakeClock(time.Now())
	emitter := NewFakeEmitter(clk)
	if err := emitter.Emit(context.Background(), audit.Event{EventType: "created", Actor: "x"}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	ev := emitter.Events()[0]
	// Verify we can assign to a *types.AuditEvent — meaning Events()
	// returns the same shape FakeRepository.AuditEvents does.
	var _ *types.AuditEvent = ev
}
