package orchestration

import (
	"context"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
)

// This fault occurs after the actual native row exists, before its authority
// receipt commits. It never changes dispatcher behavior or human verification.
type lostFiniteAcceptStore struct {
	effortauthority.Store
	failed bool
}

func (s *lostFiniteAcceptStore) Update(ctx context.Context, id string, fn func(*effortauthority.Record) error) error {
	record, e := s.Store.Get(ctx, id)
	if e != nil {
		return e
	}
	before := map[string]string{}
	for k, v := range record.Reservations {
		before[k] = v.RunID
	}
	if e = fn(&record); e != nil {
		return e
	}
	for k, v := range record.Reservations {
		if before[k] == "" && v.RunID != "" && !s.failed {
			s.failed = true
			return errors.New("disposable native acceptance receipt lost")
		}
	}
	return s.Store.Update(ctx, id, fn)
}
func TestFiniteNativeLostAcceptanceReceiptRepairsExactRunWithoutRedispatch(t *testing.T) {
	o, a, p, task, profile, key, _ := nativeEffortFixture(t)
	base := a.Store
	fault := &lostFiniteAcceptStore{Store: base}
	a.Store = fault
	request := effortRequest(t, o, p, task, profile, key, "lost-native-receipt")
	if _, e := o.CreateRun(context.Background(), request); e == nil || !fault.failed {
		t.Fatal("native acceptance fault not reached", e)
	}
	run, e := o.runs.GetByIdempotencyKey(context.Background(), request.IdempotencyKey)
	if e != nil || run == nil {
		t.Fatal("native row did not retain original unknown outcome", e)
	}
	record, e := base.Get(context.Background(), p.ID)
	if e != nil {
		t.Fatal(e)
	}
	receipt := record.Reservations[request.IdempotencyKey]
	if !receipt.NativeBound || receipt.RunID != "" || len(record.Reservations) != 1 || receipt.Terminal {
		t.Fatal("lost receipt released or fabricated native acceptance")
	}
	// Replay enters actual native idempotency qualification. The deliberately
	// closed dispatcher makes any redispatch an error, so success proves repair.
	replay, e := o.CreateRun(context.Background(), request)
	if e != nil || replay == nil || replay.ID != run.ID {
		t.Fatal("exact native receipt could not repair without redispatch", e)
	}
	record, e = base.Get(context.Background(), p.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(record.Reservations) != 1 || record.Reservations[request.IdempotencyKey].RunID != run.ID.String() || record.Reservations[request.IdempotencyKey].Terminal {
		t.Fatal("receipt repair replaced original or released occupied allowance")
	}
}
