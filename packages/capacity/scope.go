package capacity

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// UnavailableError means the broker evaluated a request but did not admit it.
// Schedulers may retry it; transport and cleanup errors are separate failures.
type UnavailableError struct{ Verdict Verdict }

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("capacity admission %s: %s", e.Verdict.Kind, e.Verdict.Reason)
}

// Scope renews a claim while its owner is executing. Losing renewal cancels the
// work context. Cancellation never releases capacity early: the owner must reap
// its worker before calling Release. Failed releases can be retried.
type Scope struct {
	ctx   context.Context
	stop  chan struct{}
	done  chan struct{}
	mu    sync.Mutex
	lease RenewableLease
	close func() error
}

// AcquireScoped acquires CPU/RAM from the shared ledger and owns its connection.
// It applies no waiting policy; the caller's scheduler owns admission retries.
func AcquireScoped(ctx context.Context, request Request) (*Scope, error) {
	broker, err := NewBroker(ctx, "")
	if err != nil {
		return nil, err
	}
	lease, verdict, err := broker.AcquireResources(ctx, request)
	if err != nil || verdict.Kind != "grant" || lease == nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if lease != nil {
			err = errors.Join(err, lease.Release(cleanup))
		}
		err = errors.Join(err, broker.Close())
		if err == nil {
			err = &UnavailableError{Verdict: verdict}
		}
		return nil, err
	}
	s := Maintain(ctx, lease)
	s.close = broker.Close
	return s, nil
}

// Maintain renews an already-owned CPU/RAM or GPU lease. The caller retains any
// enclosing broker connection until Release succeeds.
func Maintain(ctx context.Context, lease RenewableLease) *Scope {
	return maintain(ctx, lease, RecommendedLeaseHeartbeatInterval())
}

// MaintainWithInterval supports leases with a shorter TTL than the shared
// broker's default. Invalid or overlong cadences use the recommended interval;
// the caller must choose a cadence shorter than the lease owner's TTL.
func MaintainWithInterval(ctx context.Context, lease RenewableLease, interval time.Duration) *Scope {
	if interval <= 0 || interval > RecommendedLeaseHeartbeatInterval() {
		interval = RecommendedLeaseHeartbeatInterval()
	}
	return maintain(ctx, lease, interval)
}

func maintain(ctx context.Context, lease RenewableLease, interval time.Duration) *Scope {
	ctx, cancel := context.WithCancelCause(ctx)
	s := &Scope{ctx: ctx, stop: make(chan struct{}), done: make(chan struct{}), lease: lease}
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			case <-ticker.C:
				beat, stop := context.WithTimeout(ctx, 5*time.Second)
				err := lease.Heartbeat(beat)
				stop()
				if err != nil {
					err = fmt.Errorf("capacity renewal lost: %w", err)
					cancel(err)
					return
				}
			}
		}
	}()
	return s
}

func (s *Scope) Context() context.Context { return s.ctx }

// Release stops and joins renewal, then releases using bounded cleanup that
// survives caller cancellation. It is safe to retry after a ledger failure.
func (s *Scope) Release(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done != nil {
		close(s.stop)
		<-s.done
		s.done = nil
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if s.lease != nil {
		if err := s.lease.Release(cleanup); err != nil {
			return fmt.Errorf("release capacity: %w", err)
		}
		s.lease = nil
	}
	if s.close != nil {
		if err := s.close(); err != nil {
			return err
		}
		s.close = nil
	}
	return nil
}
