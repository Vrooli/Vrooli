package bindings

import (
	"context"
	"sync"
	"time"
)

// UsageReader answers program-name → run count for the library corpus.
type UsageReader func(context.Context) (map[string]int64, error)

// UsageCache memoizes a UsageReader for the library search corpus.
//
// Search Hub calls the library corpus several times a minute (queries plus
// its registry probe), and the usage read was previously the full portfolio
// scan on every call — measured at ~1.9 s and ~0.8 idle cores on a
// production-sized programs table. The cache turns that into one read per
// TTL: callers inside the TTL share the same map, concurrent callers on an
// empty cache share one computation, and once the TTL lapses the stale map is
// served immediately while a single background refresh replaces it, so a
// search never blocks on the scan after the first one.
type UsageCache struct {
	reader  UsageReader
	ttl     time.Duration
	timeout time.Duration
	now     func() time.Time
	base    context.Context

	mu         sync.Mutex
	usage      map[string]int64
	computedAt time.Time
	inflight   chan struct{} // closed when the current computation finishes
	lastErr    error
}

// NewUsageCache wraps reader. base bounds background refreshes so they stop
// with the server instead of with the request that triggered them; ttl <= 0
// disables caching by expiring every entry immediately (still single-flight).
func NewUsageCache(base context.Context, reader UsageReader, ttl time.Duration) *UsageCache {
	if base == nil {
		base = context.Background()
	}
	return &UsageCache{reader: reader, ttl: ttl, timeout: 30 * time.Second, now: time.Now, base: base}
}

// Read returns the cached usage map. The first call (and any call while the
// cache is empty) computes synchronously, sharing one computation between
// concurrent callers. Later calls return the cached map; once it is older
// than the TTL the stale map is returned and one refresh runs in the
// background. Callers must treat the map as read-only.
func (c *UsageCache) Read(ctx context.Context) (map[string]int64, error) {
	c.mu.Lock()
	if c.usage != nil {
		usage := c.usage
		stale := c.now().Sub(c.computedAt) >= c.ttl
		if stale && c.inflight == nil {
			c.startLocked()
		}
		c.mu.Unlock()
		return usage, nil
	}
	if c.inflight == nil {
		c.startLocked()
	}
	done := c.inflight
	c.mu.Unlock()
	select {
	case <-done:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.usage == nil {
		return nil, c.lastErr
	}
	return c.usage, nil
}

// startLocked launches one computation; the caller holds c.mu.
func (c *UsageCache) startLocked() {
	done := make(chan struct{})
	c.inflight = done
	go func() {
		callCtx, cancel := context.WithTimeout(c.base, c.timeout)
		defer cancel()
		usage, err := c.reader(callCtx)
		c.mu.Lock()
		if err == nil {
			if usage == nil {
				usage = map[string]int64{}
			}
			c.usage = usage
			c.computedAt = c.now()
			c.lastErr = nil
		} else {
			c.lastErr = err
		}
		c.inflight = nil
		c.mu.Unlock()
		close(done)
	}()
}
