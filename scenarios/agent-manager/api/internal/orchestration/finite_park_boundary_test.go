package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"context"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
	"time"
)

func TestFiniteParkBoundaryRefusesBeforeRunAndLedgerEffects(t *testing.T) {
	for _, name := range []string{"expired", "revoked", "wrong-owner", "changed-profile", "beyond-deadline", "unaccepted-slot"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			o, a, p, task, profile, key, now := nativeEffortFixture(t)
			req := effortRequest(t, o, p, task, profile, key, "park-boundary")
			if _, e := o.CreateRun(ctx, req); !errors.Is(e, spawn.ErrDispatcherClosed) {
				t.Fatal(e)
			}
			run, e := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
			if e != nil || run == nil {
				t.Fatal(e)
			}
			run.Status = domain.RunStatusRunning
			run.SessionID = "disposable-fixture-session"
			var deadline *time.Time
			switch name {
			case "expired":
				*now = p.Deadline
			case "revoked":
				if e = a.Revoke(ctx, "fixture-human", p.ID); e != nil {
					t.Fatal(e)
				}
			case "wrong-owner":
				run.OwnerSubject = "other"
			case "changed-profile":
				profile.Name = "changed"
				if e = o.profiles.Update(ctx, profile); e != nil {
					t.Fatal(e)
				}
			case "beyond-deadline":
				d := p.Deadline.Add(time.Second)
				deadline = &d
			case "unaccepted-slot":
				run.IdempotencyKey = "foreign"
			}
			if e = o.runs.Update(ctx, run); e != nil {
				t.Fatal(e)
			}
			beforeRun, e := o.runs.Get(ctx, run.ID)
			if e != nil {
				t.Fatal(e)
			}
			beforeLedger, e := a.Store.Get(ctx, p.ID)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = o.ParkRun(ctx, ParkRunInput{RunID: run.ID, Producer: "fixture", Key: "receipt", Deadline: deadline}); !errors.Is(e, effortauthority.ErrRefused) {
				t.Fatalf("finite authority gate was not the refusal: %v", e)
			}
			afterRun, e := o.runs.Get(ctx, run.ID)
			if e != nil {
				t.Fatal(e)
			}
			afterLedger, e := a.Store.Get(ctx, p.ID)
			if e != nil {
				t.Fatal(e)
			}
			if effortauthority.Digest(beforeRun) != effortauthority.Digest(afterRun) || effortauthority.Digest(beforeLedger) != effortauthority.Digest(afterLedger) {
				t.Fatal("refused park changed run/ledger")
			}
		})
	}
}
func TestFiniteParkKeepsReservationAndOriginalDeadline(t *testing.T) {
	ctx := context.Background()
	o, a, p, task, profile, key, now := nativeEffortFixture(t)
	req := effortRequest(t, o, p, task, profile, key, "park-compatible")
	if _, e := o.CreateRun(ctx, req); !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatal(e)
	}
	run, e := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if e != nil {
		t.Fatal(e)
	}
	run.Status = domain.RunStatusRunning
	run.SessionID = "disposable-fixture-session"
	if e = o.runs.Update(ctx, run); e != nil {
		t.Fatal(e)
	}
	*now = p.Deadline.Add(-time.Minute)
	before, e := a.Store.Get(ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	parked, e := o.ParkRun(ctx, ParkRunInput{RunID: run.ID, Producer: "fixture", Key: "receipt"})
	if e != nil {
		t.Fatal(e)
	}
	if parked.AwaitHandle == nil || !parked.AwaitHandle.Deadline.Equal(p.Deadline) {
		t.Fatal("park exceeded original authority deadline")
	}
	after, e := a.Store.Get(ctx, p.ID)
	if e != nil || effortauthority.Digest(before) != effortauthority.Digest(after) || after.Reservations[req.IdempotencyKey].Terminal {
		t.Fatal("park freed or changed finite budget", e)
	}
}
func TestFiniteFactoryDoesNotBlockOrdinaryContinuation(t *testing.T) {
	o, _, _, _, _, _, _ := nativeEffortFixture(t)
	if e := o.checkEffortContinuation(context.Background(), &domain.Run{}); e != nil {
		t.Fatal("finite factory changed ordinary lifecycle", e)
	}
}
