package fetch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
)

// DiskLeaseGuard keeps a complete install lease live while synchronous writers
// run. A definitive heartbeat refusal (the lease is no longer active) cancels
// the work context immediately. A heartbeat that only TIMES OUT — the host
// capacity ledger serializes on one fleet-wide admission lock that other
// scenarios' lifecycle commands can hold for tens of seconds — is retried at a
// short cadence until diskLeaseStaleBudget has passed since the last confirmed
// renewal; the lease TTL (15 min) is well beyond that budget, so retrying never
// outlives admission, while one slow ledger call no longer kills a multi-GB
// download. Callers must leave the reservation active if cleanup cannot be
// verified.
type DiskLeaseGuard struct {
	lease  capacity.DiskJobLease
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	err    error
}

// diskLeaseStaleBudget bounds how long timed-out heartbeats are retried before
// the guard treats admission as lost (a third of the 15-minute disk lease TTL).
// diskLeaseRetryInterval is the retry cadence while renewals are timing out, and
// diskLeaseAttemptTimeout bounds one renewal attempt. Variables so tests can
// shrink them.
var (
	diskLeaseStaleBudget    = 5 * time.Minute
	diskLeaseRetryInterval  = 10 * time.Second
	diskLeaseAttemptTimeout = 10 * time.Second
)

func MaintainDiskLease(ctx context.Context, lease capacity.DiskJobLease, interval time.Duration) (context.Context, *DiskLeaseGuard) {
	workCtx, cancel := context.WithCancel(ctx)
	g := &DiskLeaseGuard{lease: lease, cancel: cancel, done: make(chan struct{})}
	if interval <= 0 {
		interval = 45 * time.Second
	}
	go func() {
		defer close(g.done)
		lastRenewed := time.Now()
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			select {
			case <-workCtx.Done():
				return
			case <-timer.C:
				hbCtx, stop := context.WithTimeout(context.WithoutCancel(workCtx), diskLeaseAttemptTimeout)
				err := lease.Heartbeat(hbCtx)
				timedOut := hbCtx.Err() != nil || errors.Is(err, context.DeadlineExceeded)
				stop()
				if err == nil {
					lastRenewed = time.Now()
					timer.Reset(interval)
					continue
				}
				if timedOut && time.Since(lastRenewed) < diskLeaseStaleBudget {
					timer.Reset(diskLeaseRetryInterval)
					continue
				}
				g.mu.Lock()
				g.err = fmt.Errorf("disk lease heartbeat failed (last renewed %s ago): %w", time.Since(lastRenewed).Round(time.Second), err)
				g.mu.Unlock()
				cancel()
				return
			}
		}
	}()
	return workCtx, g
}

func (g *DiskLeaseGuard) Stop() error {
	if g == nil {
		return nil
	}
	g.cancel()
	<-g.done
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}
