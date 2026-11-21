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
	budget   time.Duration
	now      func() time.Time

	mu         sync.Mutex
	have       bool
	at         time.Time
	degraded   bool
	detail     string
	err        error
	refreshing bool
	done       chan struct{}
}

// postureProbeBudget bounds how long a probe waits for a first or too-old
// answer. Posture reads every target's last run and every enabled repository;
// while a large backup writes that can exceed the watchdog's health timeout, and
// a cold cache after a restart would then time out, get restarted, and start
// cold again — a restart loop that aborted runs every few minutes. Past this
// budget the probe answers optimistically and the refresh lands on the next one.
const postureProbeBudget = 2 * time.Second

// NewCachedPosture wraps posture with the stale-while-revalidate cache above.
func NewCachedPosture(posture BackupPosture, fresh, maxStale time.Duration) BackupPosture {
	return newCachedPosture(posture, fresh, maxStale, time.Now)
}

func newCachedPosture(posture BackupPosture, fresh, maxStale time.Duration, now func() time.Time) *cachedPosture {
	return &cachedPosture{inner: posture, fresh: fresh, maxStale: maxStale, timeout: time.Minute, budget: postureProbeBudget, now: now}
}

func (c *cachedPosture) OverdueOrFailed(ctx context.Context) (bool, string, error) {
	c.mu.Lock()
	age := c.now().Sub(c.at)
	switch {
	case c.have && age < c.fresh:
		defer c.mu.Unlock()
		return c.degraded, c.detail, c.err
	case c.have && age < c.maxStale:
		// Serve the stale answer at once; one background refresh catches up.
		c.startRefreshLocked()
		defer c.mu.Unlock()
		return c.degraded, c.detail, c.err
	}
	// Cold, or older than maxStale: wait briefly for a fresh answer, then answer
	// from what we have rather than block the watchdog past its timeout.
	done := c.startRefreshLocked()
	have, degraded, detail := c.have, c.degraded, c.detail
	c.mu.Unlock()

	timer := time.NewTimer(c.budget)
	defer timer.Stop()
	select {
	case <-done:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.degraded, c.detail, c.err
	case <-timer.C:
	case <-ctx.Done():
	}
	if have {
		return degraded, detail, nil
	}
	return false, "", nil
}

// startRefreshLocked starts the single in-flight refresh and returns the
// channel that closes when it finishes. The caller must hold c.mu.
func (c *cachedPosture) startRefreshLocked() chan struct{} {
	if !c.refreshing {
		c.refreshing = true
		c.done = make(chan struct{})
		go c.refresh(c.done)
	}
	return c.done
}

// refresh recomputes under its own deadline: it outlives the probe that started it.
func (c *cachedPosture) refresh(done chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	degraded, detail, err := c.inner.OverdueOrFailed(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshing = false
	if err != nil {
		// Keep the last good answer for the stale path, but surface the error so
		// an answer older than maxStale reports why it could not be refreshed.
		c.err = err
	} else {
		c.have, c.at, c.degraded, c.detail, c.err = true, c.now(), degraded, detail, nil
	}
	close(done)
}
