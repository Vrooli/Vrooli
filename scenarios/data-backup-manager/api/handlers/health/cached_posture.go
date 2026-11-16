package health

import (
	"context"
	"sync"
	"time"
)

// cachedPosture answers the health probe from the last backup-posture result so
// a probe never waits on the work behind it. Posture reads every target's last
// run and every enabled plan's repository; while a large backup is writing, that
// took 5 seconds or more, the probe timed out, and the watchdog restarted the
// service in the middle of the run (twice in ten minutes on 2026-11-15).
//
// A result younger than fresh is served as is. An older one is still served at
// once, and one background refresh brings it up to date. A result older than
// maxStale, or none yet, is computed on the spot: an answer that old is not shown
// as the current state.
type cachedPosture struct {
	inner    BackupPosture
	fresh    time.Duration
	maxStale time.Duration
	timeout  time.Duration
	now      func() time.Time

	mu         sync.Mutex
	have       bool
	at         time.Time
	degraded   bool
	detail     string
	err        error
	refreshing bool
}

// NewCachedPosture wraps posture with the stale-while-revalidate cache above.
func NewCachedPosture(posture BackupPosture, fresh, maxStale time.Duration) BackupPosture {
	return newCachedPosture(posture, fresh, maxStale, time.Now)
}

func newCachedPosture(posture BackupPosture, fresh, maxStale time.Duration, now func() time.Time) *cachedPosture {
	return &cachedPosture{inner: posture, fresh: fresh, maxStale: maxStale, timeout: time.Minute, now: now}
}

func (c *cachedPosture) OverdueOrFailed(ctx context.Context) (bool, string, error) {
	c.mu.Lock()
	age := c.now().Sub(c.at)
	switch {
	case c.have && age < c.fresh:
		defer c.mu.Unlock()
		return c.degraded, c.detail, c.err
	case c.have && age < c.maxStale:
		if !c.refreshing {
			c.refreshing = true
			go c.refresh()
		}
		defer c.mu.Unlock()
		return c.degraded, c.detail, c.err
	}
	c.mu.Unlock()

	degraded, detail, err := c.inner.OverdueOrFailed(ctx)
	c.store(degraded, detail, err)
	return degraded, detail, err
}

// refresh recomputes under its own deadline: it outlives the probe that started it.
func (c *cachedPosture) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	degraded, detail, err := c.inner.OverdueOrFailed(ctx)
	c.mu.Lock()
	c.refreshing = false
	c.mu.Unlock()
	if err != nil {
		// Keep the last good answer; it ages out at maxStale and is then recomputed
		// in the open, where the error is shown.
		return
	}
	c.store(degraded, detail, nil)
}

func (c *cachedPosture) store(degraded bool, detail string, err error) {
	if err != nil {
		return
	}
	c.mu.Lock()
	c.have, c.at, c.degraded, c.detail, c.err = true, c.now(), degraded, detail, nil
	c.mu.Unlock()
}
