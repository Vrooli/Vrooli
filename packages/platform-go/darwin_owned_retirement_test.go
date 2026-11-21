//go:build !windows

package platform

import (
	"context"
	"errors"
	"testing"
)

type darwinRevokeTransitionFixture struct {
	darwinOwnedStore
	afterRevoke func()
}

func (s darwinRevokeTransitionFixture) Revoke(r darwinOwnedRecord) error {
	if err := s.darwinOwnedStore.Revoke(r); err != nil {
		return err
	}
	s.afterRevoke()
	return nil
}

// A manager label remains exact, but its current kernel root changed after the
// initial inspection. Revocation I/O is an observable gap before bootout.
func TestDarwinOperatorStopRejectsRootChangedDuringRevocation(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	b.store = darwinRevokeTransitionFixture{b.store, func() {
		f.process = darwinOwnedProcess{PID: 77, Seconds: 200, Micros: 9, UniqueID: 777, ExecVersion: 1}
	}}
	before := len(f.calls)
	receipt, err := b.stop(context.Background(), ref)
	if !errors.Is(err, ErrOperationIdentity) || receipt.NativeQuiescent || len(f.calls) != before {
		t.Fatalf("changed manager root was retired: receipt=%+v error=%v effects=%v", receipt, err, f.calls[before:])
	}
}

func TestDarwinOperatorReleaseRejectsRootStartedDuringRevocation(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.process = darwinOwnedProcess{}
	b.store = darwinRevokeTransitionFixture{b.store, func() {
		f.process = darwinOwnedProcess{PID: 77, Seconds: 200, Micros: 9, UniqueID: 777, ExecVersion: 1}
	}}
	before := len(f.calls)
	err := b.release(context.Background(), ref)
	if !errors.Is(err, ErrOperationIdentity) || len(f.calls) != before {
		t.Fatalf("new manager root was released: error=%v effects=%v", err, f.calls[before:])
	}
}
