package faultinject

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
)

// TestArmOutsideTestModeIsRefused [REQ:STC-P0-040] (AUTH-08 groundwork,
// P02-O04): a production context can never arm a fault.
func TestArmOutsideTestModeIsRefused(t *testing.T) {
	r := New()
	err := r.Arm(context.Background(), TransportSend, Behaviour{Kind: KindFail})
	if !errors.Is(err, ErrNotTestMode) {
		t.Fatalf("expected ErrNotTestMode, got %v", err)
	}
	if r.Armed(TransportSend) {
		t.Fatalf("refused arm must leave the point disarmed")
	}
}

// TestHitNeverFiresOutsideTestMode [REQ:STC-P0-040]: even a registry armed
// through a test-mode context is inert for a production context.
func TestHitNeverFiresOutsideTestMode(t *testing.T) {
	r := New()
	testCtx := database.WithTestMode(context.Background())
	if err := r.Arm(testCtx, WorkerBeforeCommit, Behaviour{Kind: KindFail}); err != nil {
		t.Fatalf("arm in test mode: %v", err)
	}
	prodCtx := WithRegistry(context.Background(), r)
	for _, p := range Points() {
		if err := Hit(prodCtx, p); err != nil {
			t.Fatalf("Hit fired at %s outside test mode: %v", p, err)
		}
		if err := r.Hit(prodCtx, p); err != nil {
			t.Fatalf("Registry.Hit fired at %s outside test mode: %v", p, err)
		}
	}
	if r.Hits(WorkerBeforeCommit) != 0 {
		t.Fatalf("hit counter must stay zero outside test mode")
	}
	if err := Hit(context.Background(), TransportSend); err != nil {
		t.Fatalf("Hit with no registry must be nil, got %v", err)
	}
	if err := (*Registry)(nil).Hit(testCtx, TransportSend); err != nil {
		t.Fatalf("nil registry must be inert, got %v", err)
	}
}

func TestArmedPointFiresTypedFaultInTestMode(t *testing.T) {
	r := New()
	ctx := WithRegistry(database.WithTestMode(context.Background()), r)
	if err := r.Arm(ctx, TransportReply, Behaviour{Kind: KindDropReply, Once: true}); err != nil {
		t.Fatalf("arm: %v", err)
	}
	err := Hit(ctx, TransportReply)
	var f *Fault
	if !errors.As(err, &f) || f.Point != TransportReply || f.Kind != KindDropReply {
		t.Fatalf("expected drop_reply fault at transport_reply, got %v", err)
	}
	if !errors.Is(err, ErrReplyDropped) {
		t.Fatalf("drop_reply fault must unwrap to ErrReplyDropped")
	}
	if err := Hit(ctx, TransportReply); err != nil {
		t.Fatalf("Once must disarm after the first hit, got %v", err)
	}
	if r.Hits(TransportReply) != 1 {
		t.Fatalf("expected one hit, got %d", r.Hits(TransportReply))
	}
	if err := Hit(ctx, DataBeforeBackup); err != nil {
		t.Fatalf("unarmed point must not fire, got %v", err)
	}
}

func TestCrashBehaviourUsesCrashFunc(t *testing.T) {
	r := New()
	var crashed *Fault
	r.SetCrashFunc(func(f *Fault) { crashed = f })
	ctx := WithRegistry(database.WithTestMode(context.Background()), r)
	if err := r.Arm(ctx, ActivationAfterSwitch, Behaviour{Kind: KindCrash}); err != nil {
		t.Fatalf("arm: %v", err)
	}
	err := Hit(ctx, ActivationAfterSwitch)
	if crashed == nil || crashed.Point != ActivationAfterSwitch {
		t.Fatalf("crash func not invoked: %v", err)
	}
	var f *Fault
	if !errors.As(err, &f) || f.Kind != KindCrash {
		t.Fatalf("crash must also return the fault, got %v", err)
	}
}

func TestDelayRespectsContext(t *testing.T) {
	r := New()
	base := WithRegistry(database.WithTestMode(context.Background()), r)
	if err := r.Arm(base, DataAfterRestore, Behaviour{Kind: KindDelay, Delay: time.Minute}); err != nil {
		t.Fatalf("arm: %v", err)
	}
	ctx, cancel := context.WithTimeout(base, 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := Hit(ctx, DataAfterRestore)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("delay must stop at ctx done")
	}
	if err := r.Arm(base, DataBeforeBackup, Behaviour{Kind: KindDelay, Delay: time.Millisecond}); err != nil {
		t.Fatalf("arm: %v", err)
	}
	if err := Hit(base, DataBeforeBackup); err != nil {
		t.Fatalf("short delay must return nil, got %v", err)
	}
}

func TestArmRejectsUnknownPointOrKind(t *testing.T) {
	r := New()
	ctx := database.WithTestMode(context.Background())
	if err := r.Arm(ctx, Point("shell_exec"), Behaviour{Kind: KindFail}); !errors.Is(err, ErrUnknownPoint) {
		t.Fatalf("expected ErrUnknownPoint, got %v", err)
	}
	if err := r.Arm(ctx, TransportSend, Behaviour{Kind: Kind("corrupt")}); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("expected ErrUnknownKind, got %v", err)
	}
	var zero Registry
	if err := zero.Arm(ctx, TransportSend, Behaviour{Kind: KindFail}); err != nil {
		t.Fatalf("zero registry must be usable after Arm: %v", err)
	}
	if !zero.Armed(TransportSend) {
		t.Fatalf("zero registry did not record arm")
	}
}

func TestPointsAreStableAndComplete(t *testing.T) {
	want := []Point{
		"transport_send", "transport_reply",
		"worker_before_commit", "worker_after_effect",
		"activation_before_switch", "activation_after_switch",
		"data_before_backup", "data_after_restore",
	}
	got := Points()
	if len(got) != len(want) {
		t.Fatalf("expected %d points, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("point %d: expected %s, got %s", i, want[i], got[i])
		}
	}
}
