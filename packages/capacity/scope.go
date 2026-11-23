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
		// The ledger's writer lock can be held by another process for up to its
		// SQLite busy timeout. Give release a window that outlives a normal busy
		// wait (and its retries), while an enclosing caller deadline still wins.
		cleanup, cancel := detachedCleanupContext(ctx, 45*time.Second)
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
//
// A renewal that only TIMES OUT is retried every scopeRenewRetryInterval until
// two cadences have passed since the last confirmed renewal: the cadence is a
// third of the broker TTL, so the claim never lapses while retrying, and one
// slow ledger call (the admission lock is fleet-wide and other lifecycle
// commands hold it for tens of seconds) no longer cancels a long job. Any other
// renewal error cancels the work immediately.
func Maintain(ctx context.Context, lease RenewableLease) *Scope {
	interval := RecommendedLeaseHeartbeatInterval()
	return maintain(ctx, lease, interval, 2*interval)
}

// MaintainWithInterval supports leases with a shorter TTL than the shared
// broker's default. Invalid or overlong cadences use the recommended interval;
// the caller must choose a cadence shorter than the lease owner's TTL. Because
// that is the only promise, any renewal failure cancels the work immediately.
func MaintainWithInterval(ctx context.Context, lease RenewableLease, interval time.Duration) *Scope {
	if interval <= 0 || interval > RecommendedLeaseHeartbeatInterval() {
		interval = RecommendedLeaseHeartbeatInterval()
	}
	return maintain(ctx, lease, interval, 0)
}

// scopeRenewRetryInterval is the retry cadence while renewals time out.
var scopeRenewRetryInterval = 10 * time.Second

func maintain(ctx context.Context, lease RenewableLease, interval, staleBudget time.Duration) *Scope {
	ctx, cancel := context.WithCancelCause(ctx)
	s := &Scope{ctx: ctx, stop: make(chan struct{}), done: make(chan struct{}), lease: lease}
	go func() {
		defer close(s.done)
		lastRenewed := time.Now()
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			case <-timer.C:
				beat, stop := context.WithTimeout(ctx, 5*time.Second)
				err := lease.Heartbeat(beat)
				timedOut := beat.Err() != nil || errors.Is(err, context.DeadlineExceeded)
				stop()
				if err == nil {
					lastRenewed = time.Now()
					timer.Reset(interval)
					continue
				}
				if timedOut && ctx.Err() == nil && time.Since(lastRenewed) < staleBudget {
					timer.Reset(min(scopeRenewRetryInterval, interval))
					continue
				}
				err = fmt.Errorf("capacity renewal lost: %w", err)
				cancel(err)
				return
			}
		}
	}()
	return s
}

func (s *Scope) Context() context.Context { return s.ctx }

// StopRenew stops and joins the heartbeat loop without releasing the claim.
// It is useful when a producer has exited but cleanup cannot be verified: the
// lease remains retained in the ledger while no caller goroutine is leaked.
// Release may still be called later after owner-verified cleanup.
func (s *Scope) StopRenew() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopRenewLocked()
}

func (s *Scope) stopRenewLocked() {
	if s.done == nil {
		return
	}
	close(s.stop)
	<-s.done
	s.done = nil
}

// Release stops and joins renewal, then releases using bounded cleanup that
// survives caller cancellation. It is safe to retry after a ledger failure.
func (s *Scope) Release(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopRenewLocked()
	// The ledger's writer lock can be held by another process for up to its
	// SQLite busy timeout. Give release a window that outlives a normal busy
	// wait (and its retries), while an enclosing caller deadline still wins.
	cleanup, cancel := detachedCleanupContext(ctx, 45*time.Second)
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

// detachedCleanupContext ignores caller cancellation but retains an earlier
// caller deadline. Cleanup can finish after a canceled job, while an enclosing
// owner timeout still bounds how long the shared broker operation may run.
func detachedCleanupContext(ctx context.Context, max time.Duration) (context.Context, context.CancelFunc) {
	deadline := time.Now().Add(max)
	if callerDeadline, ok := ctx.Deadline(); ok && callerDeadline.Before(deadline) {
		deadline = callerDeadline
	}
	return context.WithDeadline(context.Background(), deadline)
}
