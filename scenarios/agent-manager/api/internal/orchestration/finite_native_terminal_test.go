package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"context"
	"errors"
	"fmt"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"testing"
)

type unitTerminalFixture struct {
	err   error
	calls int
}

func (u *unitTerminalFixture) Terminal(context.Context, string) error { u.calls++; return u.err }
func TestFiniteUnknownUnitCannotReleaseBudget(t *testing.T) {
	o, a, p, task, profile, key, _ := nativeEffortFixture(t)
	req := effortRequest(t, o, p, task, profile, key, "unit-gate")
	_, e := o.CreateRun(context.Background(), req)
	if !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatal(e)
	}
	r, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey)
	if e != nil {
		t.Fatal(e)
	}
	r.Status = domain.RunStatusComplete
	if e = o.runs.Update(context.Background(), r); e != nil {
		t.Fatal(e)
	}
	b := *r.ResolvedConfig.Admission.Effort
	for _, owner := range []interface {
		Terminal(context.Context, string) error
	}{nil, &unitTerminalFixture{err: errors.New("unknown native descendants")}} {
		o.finiteNativeTerminal = owner
		before, _ := a.Store.Get(context.Background(), p.ID)
		if o.SyncEffortTerminal(context.Background(), b, r.IdempotencyKey, r.ID.String()) == nil {
			t.Fatal("AM row/wrapper outcome released occupancy")
		}
		after, _ := a.Store.Get(context.Background(), p.ID)
		if effortauthority.Digest(before) != effortauthority.Digest(after) {
			t.Fatal("unknown settlement changed ledger")
		}
	}
	owner := &unitTerminalFixture{}
	o.finiteNativeTerminal = owner
	if e = o.SyncEffortTerminal(context.Background(), b, r.IdempotencyKey, r.ID.String()); e != nil || owner.calls != 1 {
		t.Fatal("verified fixture native unit terminal refused", e)
	}
	record, _ := a.Store.Get(context.Background(), p.ID)
	if !record.Reservations[r.IdempotencyKey].Terminal {
		t.Fatal("known terminal was not settled")
	}
}

func TestFiniteMissingOrDisabledInstallationRefusesBeforeReservation(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprint(disabled), func(t *testing.T) {
			o, a, p, task, profile, key, _ := nativeEffortFixture(t)
			o.finiteNativeFactory = nil
			if disabled {
				f, e := runner.NewFiniteNativeFactory(isolation.Manifest{})
				if e != nil {
					t.Fatal(e)
				}
				o.finiteNativeFactory = f
			}
			req := effortRequest(t, o, p, task, profile, key, "unconfigured-native")
			before, _ := a.Store.Get(context.Background(), p.ID)
			if _, e := o.CreateRun(context.Background(), req); !errors.Is(e, effortauthority.ErrRefused) {
				t.Fatal("unconfigured route accepted", e)
			}
			after, _ := a.Store.Get(context.Background(), p.ID)
			row, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey)
			if e != nil || row != nil || effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("unconfigured request changed run or ledger", e)
			}
		})
	}
}

func TestFiniteDifferentInstalledBindingRefusesBeforeReservation(t *testing.T) {
	o, a, p, task, profile, key, _ := nativeEffortFixture(t)
	different := p
	different.ID = "other-approved-policy"
	if e := a.Approve(context.Background(), "fixture-human", different); e != nil {
		t.Fatal(e)
	}
	req := effortRequest(t, o, different, task, profile, key, "other-policy")
	before, _ := a.Store.Get(context.Background(), different.ID)
	if _, e := o.CreateRun(context.Background(), req); !errors.Is(e, effortauthority.ErrRefused) {
		t.Fatal("different binding accepted", e)
	}
	after, _ := a.Store.Get(context.Background(), different.ID)
	row, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey)
	if e != nil || row != nil || effortauthority.Digest(before) != effortauthority.Digest(after) {
		t.Fatal("different binding changed row or ledger", e)
	}
}
