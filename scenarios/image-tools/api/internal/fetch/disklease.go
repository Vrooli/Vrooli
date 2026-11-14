package fetch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
)

// DiskLeaseGuard keeps a complete install lease live while synchronous writers
// run. A heartbeat failure cancels the work context immediately; callers must
// leave the reservation active if cleanup cannot be verified.
type DiskLeaseGuard struct {
	lease  capacity.DiskJobLease
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	err    error
}

func MaintainDiskLease(ctx context.Context, lease capacity.DiskJobLease, interval time.Duration) (context.Context, *DiskLeaseGuard) {
	workCtx, cancel := context.WithCancel(ctx)
	g := &DiskLeaseGuard{lease: lease, cancel: cancel, done: make(chan struct{})}
	if interval <= 0 {
		interval = 45 * time.Second
	}
	go func() {
		defer close(g.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-workCtx.Done():
				return
			case <-ticker.C:
				hbCtx, stop := context.WithTimeout(context.WithoutCancel(workCtx), 10*time.Second)
				err := lease.Heartbeat(hbCtx)
				stop()
				if err != nil {
					g.mu.Lock()
					g.err = fmt.Errorf("disk lease heartbeat failed: %w", err)
					g.mu.Unlock()
					cancel()
					return
				}
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
