package main

import (
	"context"
	"sync"
	"sync/atomic"

	"swarm-manager/internal/backlog"
)

// nextActionProjectionCache serves one computed next-action projection to every
// reader within a data generation.
//
// The projection is the most expensive read in the scenario, and both readers —
// the operator inbox feed and the Plan board — need all of it. Computing it
// independently per reader meant a single board interaction paid for it twice,
// and the board additionally resolved every item a second time through its own
// loop.
//
// Freshness is generation-based rather than time-based on purpose: this is an
// operator inbox, so a decision the operator just made must be visible on the
// very next read. Every mutating service already announces its writes through
// the graph invalidation seam, and a bump on that signal drops the projection
// before any reader can observe the stale one.
type nextActionProjectionCache struct {
	feed nextActionFeed

	generation atomic.Uint64

	mu       sync.Mutex
	cached   nextActionProjection
	cachedAt uint64
	valid    bool
	// refreshOnRead is required when the projection includes an external
	// source, such as Offer Desk's release ladder, whose writes cannot emit
	// Swarm Manager's local invalidation event.
	refreshOnRead bool
}

func newNextActionProjectionCache(feed nextActionFeed) *nextActionProjectionCache {
	return &nextActionProjectionCache{feed: feed}
}

// SetRefreshOnRead disables cross-scenario stale reads while retaining the
// holder's single-computation behavior for callers that do not use external
// projection inputs.
func (c *nextActionProjectionCache) SetRefreshOnRead(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshOnRead = enabled
	c.valid = false
}

// Invalidate marks every cached projection stale. It is safe to call from any
// goroutine and never blocks on an in-flight computation.
func (c *nextActionProjectionCache) Invalidate() {
	c.generation.Add(1)
}

// projection returns the projection for the current data generation, computing
// it if this generation has not been computed yet.
func (c *nextActionProjectionCache) projection(ctx context.Context) (nextActionProjection, error) {
	// The generation is read before the computation and stored with it. A
	// mutation that lands while the projection is being built therefore leaves
	// the stored generation behind the current one, and the next read
	// recomputes rather than serving state the mutation already invalidated.
	generation := c.generation.Load()

	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.refreshOnRead && c.valid && c.cachedAt == generation {
		return c.cached, nil
	}
	projection, err := c.feed.project(ctx)
	if err != nil {
		return nextActionProjection{}, err
	}
	c.cached, c.cachedAt, c.valid = projection, generation, true
	return projection, nil
}

// Entries serves the ranked operator inbox.
func (c *nextActionProjectionCache) Entries(ctx context.Context) ([]nextActionFeedEntry, error) {
	projection, err := c.projection(ctx)
	if err != nil {
		return nil, err
	}
	return projection.entries, nil
}

// ResolveNextActions serves the Plan board every item's resolved action in one
// call, replacing the board's own per-item resolution loop.
func (c *nextActionProjectionCache) ResolveNextActions(ctx context.Context) (map[string]backlog.NextActionProjection, error) {
	projection, err := c.projection(ctx)
	if err != nil {
		return nil, err
	}
	return projection.actions, nil
}
