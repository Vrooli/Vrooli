package capacity

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type scopeLease struct {
	released  atomic.Int32
	heartbeat func(context.Context) error
	release   func(context.Context) error
}

func (l *scopeLease) Heartbeat(ctx context.Context) error { return l.heartbeat(ctx) }
func (l *scopeLease) Release(ctx context.Context) error {
	l.released.Add(1)
	return l.release(ctx)
}

func TestScopeCancellationRetainsClaimUntilWorkerReaped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	lease := &scopeLease{heartbeat: func(context.Context) error { return nil }, release: func(ctx context.Context) error { return ctx.Err() }}
	s := Maintain(ctx, lease)
	cancel()
	<-s.Context().Done()
	if lease.released.Load() != 0 {
		t.Fatal("cancellation released capacity before worker exit")
	}
	if err := s.Release(ctx); err != nil {
		t.Fatalf("cleanup inherited cancellation: %v", err)
	}
	if err := s.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if lease.released.Load() != 1 {
		t.Fatal("successful release was repeated")
	}
}

func TestScopeRenewalLossCancelsWorkAndRetainsReleaseFailureForRetry(t *testing.T) {
	lost := errors.New("claim fenced")
	lease := &scopeLease{heartbeat: func(context.Context) error { return lost }}
	lease.release = func(context.Context) error {
		if lease.released.Load() == 1 {
			return errors.New("ledger temporarily unavailable")
		}
		return nil
	}
	s := maintain(context.Background(), lease, time.Millisecond)
	select {
	case <-s.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("renewal loss did not cancel work")
	}
	if !errors.Is(context.Cause(s.Context()), lost) {
		t.Fatalf("cause = %v", context.Cause(s.Context()))
	}
	if lease.released.Load() != 0 {
		t.Fatal("renewal loss released a running worker's claim")
	}
	if err := s.Release(context.Background()); err == nil {
		t.Fatalf("first release = %v", err)
	}
	if err := s.Release(context.Background()); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if lease.released.Load() != 2 || s.lease != nil {
		t.Fatal("failed release was not retried")
	}
}

func TestScopeSuccessfulReleaseDoesNotCancelConsumer(t *testing.T) {
	lease := &scopeLease{heartbeat: func(context.Context) error { return nil }, release: func(context.Context) error { return nil }}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := Maintain(ctx, lease)
	if err := s.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Context().Err(); err != nil {
		t.Fatalf("successful cleanup canceled consumer: %v", err)
	}
}

func TestScopeCanStopRenewalWhileRetainingClaimForVerifiedCleanup(t *testing.T) {
	var heartbeats atomic.Int32
	lease := &scopeLease{
		heartbeat: func(context.Context) error { heartbeats.Add(1); return nil },
		release:   func(context.Context) error { return nil },
	}
	s := maintain(context.Background(), lease, time.Millisecond)
	deadline := time.Now().Add(time.Second)
	for heartbeats.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if heartbeats.Load() == 0 {
		t.Fatal("renewal loop did not heartbeat")
	}
	s.StopRenew()
	if lease.released.Load() != 0 {
		t.Fatal("stopping renewal released the retained claim")
	}
	stoppedAt := heartbeats.Load()
	time.Sleep(10 * time.Millisecond)
	if got := heartbeats.Load(); got != stoppedAt {
		t.Fatalf("heartbeat continued after StopRenew: before=%d after=%d", stoppedAt, got)
	}
	if err := s.Release(context.Background()); err != nil {
		t.Fatalf("release after verified cleanup: %v", err)
	}
	if lease.released.Load() != 1 {
		t.Fatalf("release count=%d, want 1", lease.released.Load())
	}
}
