package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/types"

	"github.com/vrooli/api-core/scheduletest"
)

func TestRepoEmitter_StampsEventTimeAndDefaults(t *testing.T) {
	frozen := time.Date(2026, 4, 29, 15, 30, 0, 0, time.UTC)
	clk := scheduletest.New(frozen)

	var captured *types.AuditEvent
	log := func(ctx context.Context, ev *types.AuditEvent) error {
		captured = ev
		return nil
	}
	emitter := audit.NewRepoEmitter(log, clk)

	id := uuid.New()
	if err := emitter.Emit(context.Background(), audit.Event{
		EventType: "created",
		SandboxID: &id,
		Actor:     "alice",
		Details:   map[string]interface{}{"reason": "test"},
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	if captured == nil {
		t.Fatal("captured event was nil")
	}
	if captured.EventType != "created" {
		t.Errorf("EventType = %q, want created", captured.EventType)
	}
	if captured.SandboxID == nil || *captured.SandboxID != id {
		t.Errorf("SandboxID = %v, want %v", captured.SandboxID, id)
	}
	if captured.Actor != "alice" {
		t.Errorf("Actor = %q, want alice", captured.Actor)
	}
	if captured.ActorType != "system" {
		t.Errorf("ActorType = %q, want system (default)", captured.ActorType)
	}
	if !captured.EventTime.Equal(frozen) {
		t.Errorf("EventTime = %v, want %v", captured.EventTime, frozen)
	}
	if captured.ID == uuid.Nil {
		t.Errorf("ID was uuid.Nil — emitter must stamp")
	}
	if captured.Details["reason"] != "test" {
		t.Errorf("Details lost: %v", captured.Details)
	}
}

func TestRepoEmitter_PreservesExplicitActorTypeAndSource(t *testing.T) {
	clk := scheduletest.New(time.Now())
	var captured *types.AuditEvent
	log := func(ctx context.Context, ev *types.AuditEvent) error {
		captured = ev
		return nil
	}
	emitter := audit.NewRepoEmitter(log, clk)

	if err := emitter.Emit(context.Background(), audit.Event{
		EventType: "rejected",
		Actor:     "system",
		ActorType: "gc",
		Source:    types.SourceWorkspaceSandboxGC,
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if captured.ActorType != "gc" {
		t.Errorf("ActorType = %q, want gc", captured.ActorType)
	}
	if captured.Source != types.SourceWorkspaceSandboxGC {
		t.Errorf("Source = %q, want %q", captured.Source, types.SourceWorkspaceSandboxGC)
	}
}

func TestRepoEmitter_RejectsEmptyEventType(t *testing.T) {
	clk := scheduletest.New(time.Now())
	log := func(ctx context.Context, ev *types.AuditEvent) error {
		t.Errorf("log should not be called for invalid Emit; got %+v", ev)
		return nil
	}
	emitter := audit.NewRepoEmitter(log, clk)

	err := emitter.Emit(context.Background(), audit.Event{EventType: ""})
	if err == nil {
		t.Error("Emit with empty EventType should fail")
	}
}

func TestRepoEmitter_PropagatesLogError(t *testing.T) {
	clk := scheduletest.New(time.Now())
	wantErr := errors.New("disk full")
	log := func(ctx context.Context, ev *types.AuditEvent) error {
		return wantErr
	}
	emitter := audit.NewRepoEmitter(log, clk)

	err := emitter.Emit(context.Background(), audit.Event{EventType: "x"})
	if !errors.Is(err, wantErr) {
		t.Errorf("Emit = %v, want %v", err, wantErr)
	}
}

func TestNewRepoEmitter_RequiresLog(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when log is nil")
		}
	}()
	audit.NewRepoEmitter(nil, scheduletest.New(time.Now()))
}

func TestNewRepoEmitter_RequiresClock(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when clock is nil")
		}
	}()
	audit.NewRepoEmitter(func(ctx context.Context, ev *types.AuditEvent) error { return nil }, nil)
}

func TestRepoEmitter_EmitsUTC(t *testing.T) {
	// FakeClock returns wall time in whatever zone callers set; the
	// emitter must convert to UTC before persisting so audit queries
	// across time zones compare consistently.
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	wall := time.Date(2026, 4, 29, 11, 45, 0, 0, loc)
	clk := scheduletest.New(wall)

	var captured *types.AuditEvent
	log := func(ctx context.Context, ev *types.AuditEvent) error {
		captured = ev
		return nil
	}
	emitter := audit.NewRepoEmitter(log, clk)
	if err := emitter.Emit(context.Background(), audit.Event{EventType: "x"}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if captured.EventTime.Location() != time.UTC {
		t.Errorf("EventTime.Location = %v, want UTC", captured.EventTime.Location())
	}
	if !captured.EventTime.Equal(wall) {
		t.Errorf("EventTime = %v, want equal to wall %v", captured.EventTime, wall)
	}
}
