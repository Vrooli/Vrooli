package hyperframes

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
)

type admissionLease struct {
	released     atomic.Bool
	heartbeatErr error
	releaseErr   error
}

func (l *admissionLease) Release(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	l.released.Store(true)
	return l.releaseErr
}
func (l *admissionLease) Heartbeat(ctx context.Context) error { return l.heartbeatErr }

type admissionBroker struct {
	lease    capacity.Lease
	verdict  capacity.Verdict
	err      error
	ram, cpu int64
	owner    string
	closed   bool
}

func (b *admissionBroker) Acquire(_ context.Context, owner string, ram, cpu int64) (capacity.Lease, capacity.Verdict, error) {
	b.owner, b.ram, b.cpu = owner, ram, cpu
	return b.lease, b.verdict, b.err
}
func (b *admissionBroker) Close() error { b.closed = true; return nil }
func factory(b *admissionBroker) renderBrokerFactory {
	return func(context.Context) (renderBroker, error) { return b, nil }
}

func TestRenderAdmissionRefusesUnavailableDeniedAndQueuedBeforeWork(t *testing.T) {
	for _, kind := range []string{"deny", "queue", "degrade", "grant"} {
		t.Run(kind, func(t *testing.T) {
			b := &admissionBroker{verdict: capacity.Verdict{Kind: kind, Reason: "insufficient headroom"}}
			called := false
			r, err := withRenderAdmission(context.Background(), 4096, factory(b), time.Hour, func(context.Context) (Receipt, error) { called = true; return Receipt{}, nil })
			if err == nil || called || !b.closed || r.Admission.Release != "not_acquired" {
				t.Fatalf("unsafe admission: %+v %v called=%v", r, err, called)
			}
		})
	}
	_, err := withRenderAdmission(context.Background(), 4096, func(context.Context) (renderBroker, error) { return nil, errors.New("sensor offline") }, time.Hour, func(context.Context) (Receipt, error) {
		t.Fatal("work after unavailable broker")
		return Receipt{}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "sensor offline") {
		t.Fatalf("missing refusal: %v", err)
	}
}

func TestRenderAdmissionHoldsBudgetUntilWorkReturnsAndReleases(t *testing.T) {
	l := &admissionLease{}
	b := &admissionBroker{lease: l, verdict: capacity.Verdict{Kind: "grant"}}
	r, err := withRenderAdmission(context.Background(), 4096, factory(b), time.Hour, func(context.Context) (Receipt, error) {
		if l.released.Load() {
			t.Fatal("released before render cleanup")
		}
		return Receipt{Status: "completed", OutputSHA256: "retained-output"}, nil
	})
	if err != nil || !l.released.Load() || !b.closed || b.ram != 4<<30 || b.cpu != 2000 || !strings.HasPrefix(b.owner, "hyperframes:") || r.Admission.Release != "released" || r.OutputSHA256 != "retained-output" {
		t.Fatalf("bad admission lifecycle: %+v %v broker=%+v", r, err, b)
	}
}

func TestRenderAdmissionCancellationReleasesWithIndependentCleanupContext(t *testing.T) {
	l := &admissionLease{}
	b := &admissionBroker{lease: l, verdict: capacity.Verdict{Kind: "grant"}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := withRenderAdmission(ctx, 1024, factory(b), time.Hour, func(work context.Context) (Receipt, error) { cancel(); <-work.Done(); return Receipt{}, work.Err() })
	if !errors.Is(err, context.Canceled) || !l.released.Load() || r.Admission.Release != "released" {
		t.Fatalf("cancel cleanup: %+v %v", r, err)
	}
}

func TestRenderAdmissionRenewalFailureCancelsWorkBeforeRelease(t *testing.T) {
	l := &admissionLease{heartbeatErr: errors.New("lease superseded")}
	b := &admissionBroker{lease: l, verdict: capacity.Verdict{Kind: "grant"}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := withRenderAdmission(ctx, 1024, factory(b), time.Millisecond, func(work context.Context) (Receipt, error) {
		<-work.Done()
		if l.released.Load() {
			t.Error("released before owned cleanup joined")
		}
		return Receipt{}, work.Err()
	})
	if err == nil || !strings.Contains(err.Error(), "lease superseded") || !l.released.Load() || r.Admission.Release != "released" {
		t.Fatalf("lost lease not surfaced: %+v %v", r, err)
	}
}

func TestRenderAdmissionRetainsClaimWhenOwnedCleanupIsUnknown(t *testing.T) {
	l := &admissionLease{}
	b := &admissionBroker{lease: l, verdict: capacity.Verdict{Kind: "grant"}}
	r, err := withRenderAdmission(context.Background(), 1024, factory(b), time.Hour, func(context.Context) (Receipt, error) {
		return Receipt{}, errors.Join(errOwnedCleanup, errors.New("scope cannot be read"))
	})
	if !errors.Is(err, errOwnedCleanup) || l.released.Load() || r.Admission.Release != "retained_until_expiry_cleanup_unverified" {
		t.Fatalf("unknown cleanup released: %+v %v", r, err)
	}
}

func TestRenderAdmissionReleaseFailurePreservesOutputIdentityAndFailsInvocation(t *testing.T) {
	l := &admissionLease{releaseErr: errors.New("ledger unavailable")}
	b := &admissionBroker{lease: l, verdict: capacity.Verdict{Kind: "grant"}}
	r, err := withRenderAdmission(context.Background(), 1024, factory(b), time.Hour, func(context.Context) (Receipt, error) { return Receipt{OutputSHA256: "already-published"}, nil })
	if err == nil || !strings.Contains(err.Error(), "ledger unavailable") || r.OutputSHA256 != "already-published" || r.Admission.Release != "failed" {
		t.Fatalf("lost retained output/release error: %+v %v", r, err)
	}
}

func TestOwnedCleanupFailureIsTypedForCapacityRetention(t *testing.T) {
	err := waitOwned(context.Background(), func() error { return nil }, func(context.Context) error { return nil }, func() ([]int, error) { return nil, errors.New("unknown scope") })
	if !errors.Is(err, errOwnedCleanup) {
		t.Fatalf("cleanup failure not typed: %v", err)
	}
}

// Completing work during a heartbeat must not turn cleanup cancellation into
// a false render failure. Synchronization makes this race deterministic.
type blockingHeartbeatLease struct {
	admissionLease
	started chan struct{}
}

func (l *blockingHeartbeatLease) Heartbeat(ctx context.Context) error {
	close(l.started)
	<-ctx.Done()
	return ctx.Err()
}
func TestRenderCompletionCancelsInflightHeartbeatWithoutFalseFailure(t *testing.T) {
	l := &blockingHeartbeatLease{started: make(chan struct{})}
	b := &admissionBroker{lease: l, verdict: capacity.Verdict{Kind: "grant"}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := withRenderAdmission(ctx, 1024, factory(b), time.Millisecond, func(context.Context) (Receipt, error) {
		<-l.started
		return Receipt{Status: "completed"}, nil
	})
	if err != nil || r.Status != "completed" || !l.released.Load() {
		t.Fatalf("completion became failure: %+v %v", r, err)
	}
}
