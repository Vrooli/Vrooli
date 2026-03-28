package graph

import (
	"context"
	"log"
	"sync"
	"time"
)

const defaultCacheTTL = 30 * time.Second

type cacheEntry struct {
	response  GraphResponse
	expiresAt time.Time
}

type inflightProjection struct {
	done     chan struct{}
	response GraphResponse
	err      error
}

// ProjectionCache memoizes graph projections per lens and coalesces concurrent
// rebuilds. Explicit invalidation is the primary freshness mechanism; TTL is a
// safety net for out-of-band filesystem changes.
type ProjectionCache struct {
	projector Projector
	ttl       time.Duration
	now       func() time.Time

	mu       sync.Mutex
	entries  map[Lens]cacheEntry
	inflight map[Lens]*inflightProjection
}

// ProjectionCacheConfig configures cache behavior.
type ProjectionCacheConfig struct {
	Projector Projector
	TTL       time.Duration
}

// NewProjectionCache creates a new per-lens projection cache.
func NewProjectionCache(cfg ProjectionCacheConfig) *ProjectionCache {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}

	return &ProjectionCache{
		projector: cfg.Projector,
		ttl:       ttl,
		now:       time.Now,
		entries:   make(map[Lens]cacheEntry),
		inflight:  make(map[Lens]*inflightProjection),
	}
}

// Project returns a cached lens projection when fresh, otherwise rebuilds it.
func (c *ProjectionCache) Project(ctx context.Context, lens Lens) (GraphResponse, error) {
	now := c.now()

	c.mu.Lock()
	if entry, ok := c.entries[lens]; ok && now.Before(entry.expiresAt) {
		resp := entry.response
		c.mu.Unlock()
		return resp, nil
	}

	if inflight, ok := c.inflight[lens]; ok {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return GraphResponse{}, ctx.Err()
		case <-inflight.done:
			if inflight.err == nil {
				return inflight.response, nil
			}
			return c.staleOrError(lens, inflight.err)
		}
	}

	inflight := &inflightProjection{done: make(chan struct{})}
	c.inflight[lens] = inflight
	c.mu.Unlock()

	startedAt := c.now()
	response, err := c.projector.Project(ctx, lens)

	c.mu.Lock()
	if err == nil {
		c.entries[lens] = cacheEntry{
			response:  response,
			expiresAt: c.now().Add(c.ttl),
		}
	}
	delete(c.inflight, lens)
	inflight.response = response
	inflight.err = err
	close(inflight.done)
	c.mu.Unlock()

	if err != nil {
		return c.staleOrError(lens, err)
	}

	log.Printf("[graph] built %s graph in %s", lens, c.now().Sub(startedAt))
	return response, nil
}

// Invalidate clears cached projections for the requested lenses.
func (c *ProjectionCache) Invalidate(lenses ...Lens) {
	if len(lenses) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, lens := range lenses {
		delete(c.entries, lens)
	}
}

func (c *ProjectionCache) staleOrError(lens Lens, buildErr error) (GraphResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[lens]
	if !ok {
		return GraphResponse{}, buildErr
	}

	log.Printf("[graph] serving stale %s graph after rebuild error: %v", lens, buildErr)
	return entry.response, nil
}
