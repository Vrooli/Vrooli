package duplex

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type reserveAdmissionStub struct {
	reservation Reservation
	err         error
}

func (a reserveAdmissionStub) Authorize(context.Context, Request) error { return a.err }
func (a reserveAdmissionStub) Reserve(context.Context, Request) (Reservation, error) {
	return a.reservation, a.err
}

type countingReservation struct{ releases atomic.Int32 }

func (r *countingReservation) Settle(uint64) error { return nil }
func (r *countingReservation) Release()            { r.releases.Add(1) }

func TestCoordinatorReleasesReservationReturnedWithError(t *testing.T) {
	reservation := &countingReservation{}
	provider := &fakeProvider{}
	coordinator, err := NewCoordinator(provider, reserveAdmissionStub{reservation: reservation, err: errors.New("denied")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = coordinator.Open(context.Background(), Request{}); err == nil {
		t.Fatal("expected admission error")
	}
	if got := reservation.releases.Load(); got != 1 {
		t.Fatalf("reservation released %d times, want 1", got)
	}
	if provider.openSeen {
		t.Fatal("provider opened after admission error")
	}
}

func TestCoordinatorRejectsNilReservationBeforeProviderOpen(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, err := NewCoordinator(provider, reserveAdmissionStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = coordinator.Open(context.Background(), Request{}); !errors.Is(err, ErrNilReservation) {
		t.Fatalf("got %v, want ErrNilReservation", err)
	}
	if provider.openSeen {
		t.Fatal("provider opened without a reservation")
	}
}

func TestBudgetSettleAndReleaseExactlyOnce(t *testing.T) {
	b := NewBudget(100)
	r, err := b.Reserve(70)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Settle(25); err != nil {
		t.Fatal(err)
	}
	r.Release()
	if !errors.Is(r.Settle(1), ErrFinalized) {
		t.Fatal("second settlement was accepted")
	}
	s := b.Snapshot()
	if s.ReservedMicros != 0 || s.SpentMicros != 25 {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
	r2, err := b.Reserve(75)
	if err != nil {
		t.Fatal(err)
	}
	r2.Release()
	r2.Release()
	if got := b.Snapshot(); got.ReservedMicros != 0 || got.SpentMicros != 25 {
		t.Fatalf("release was not idempotent: %+v", got)
	}
}

func TestBudgetConcurrentAdmissionPreventsOverspend(t *testing.T) {
	b := NewBudget(100)
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := b.Reserve(1); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := successes.Load(); got != 100 {
		t.Fatalf("admitted %d reservations, want 100", got)
	}
	if got := b.Snapshot(); got.ReservedMicros != 100 || got.SpentMicros != 0 {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestBudgetCleanupRaceSettlesOrReleasesOnce(t *testing.T) {
	b := NewBudget(50)
	r, err := b.Reserve(50)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_ = r.Settle(20)
			} else {
				r.Release()
			}
		}(i)
	}
	wg.Wait()
	s := b.Snapshot()
	if s.ReservedMicros != 0 || s.SpentMicros > 20 {
		t.Fatalf("cleanup race leaked/duplicated budget: %+v", s)
	}
}
