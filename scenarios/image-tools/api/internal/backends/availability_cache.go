package backends

import (
	"context"
	"sync"
	"time"
)

// AvailabilityCache coalesces expensive positive software probes. Unavailable
// runtimes are never cached, so on-demand provisioning becomes visible at once.
// Keys identify the resolved interpreter and its required modules. TTL bounds
// staleness when packages are removed or replaced without restarting the owner.
type AvailabilityCache struct {
	TTL   time.Duration
	mu    sync.Mutex
	key   string
	value Availability
	until time.Time
	busy  chan struct{}
}

func (c *AvailabilityCache) Do(ctx context.Context, key string, probe func() Availability) Availability {
	if c.TTL <= 0 {
		return probe()
	}
	for {
		c.mu.Lock()
		if key == c.key && time.Now().Before(c.until) {
			value := c.value
			c.mu.Unlock()
			return value
		}
		if busy := c.busy; busy != nil {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return Availability{Detail: context.Cause(ctx).Error()}
			case <-busy:
				continue
			}
		}
		c.busy = make(chan struct{})
		c.mu.Unlock()
		value := probe()
		c.mu.Lock()
		if value.Available && ctx.Err() == nil {
			c.key, c.value, c.until = key, value, time.Now().Add(c.TTL)
		}
		close(c.busy)
		c.busy = nil
		c.mu.Unlock()
		return value
	}
}
