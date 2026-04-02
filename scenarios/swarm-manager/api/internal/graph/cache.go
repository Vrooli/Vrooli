package graph

import (
	"context"
	"log"
	"sync"
	"time"
)

const defaultCacheTTL = 30 * time.Second

// cacheKey identifies a unique graph projection by lens and optional focus.
type cacheKey struct {
	Lens        Lens
	FocusNodeID string
}

type cacheEntry struct {
	response  GraphResponse
	expiresAt time.Time
}

type inflightProjection struct {
	done     chan struct{}
	response GraphResponse
	err      error
}

// ProjectionCache memoizes graph projections per lens+focus and coalesces
// concurrent rebuilds. Explicit invalidation is the primary freshness mechanism;
// TTL is a safety net for out-of-band filesystem changes.
type ProjectionCache struct {
	projector Projector
	ttl       time.Duration
	now       func() time.Time

	mu       sync.Mutex
	entries  map[cacheKey]cacheEntry
	inflight map[cacheKey]*inflightProjection
}

// ProjectionCacheConfig configures cache behavior.
type ProjectionCacheConfig struct {
	Projector Projector
	TTL       time.Duration
}

// NewProjectionCache creates a new projection cache.
func NewProjectionCache(cfg ProjectionCacheConfig) *ProjectionCache {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}

	return &ProjectionCache{
		projector: cfg.Projector,
		ttl:       ttl,
		now:       time.Now,
		entries:   make(map[cacheKey]cacheEntry),
		inflight:  make(map[cacheKey]*inflightProjection),
	}
}

// Project returns a cached projection when fresh, otherwise rebuilds it.
func (c *ProjectionCache) Project(ctx context.Context, params ProjectionParams) (GraphResponse, error) {
	key := cacheKey(params)
	now := c.now()

	c.mu.Lock()
	if entry, ok := c.entries[key]; ok && now.Before(entry.expiresAt) {
		resp := entry.response
		c.mu.Unlock()
		return resp, nil
	}

	if inflight, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return GraphResponse{}, ctx.Err()
		case <-inflight.done:
			if inflight.err == nil {
				return inflight.response, nil
			}
			return c.staleOrError(key, inflight.err)
		}
	}

	inflight := &inflightProjection{done: make(chan struct{})}
	c.inflight[key] = inflight
	c.mu.Unlock()

	startedAt := c.now()
	response, err := c.projector.Project(ctx, params)

	c.mu.Lock()
	if err == nil {
		c.entries[key] = cacheEntry{
			response:  response,
			expiresAt: c.now().Add(c.ttl),
		}
	}
	delete(c.inflight, key)
	inflight.response = response
	inflight.err = err
	close(inflight.done)
	c.mu.Unlock()

	if err != nil {
		return c.staleOrError(key, err)
	}

	if params.FocusNodeID != "" {
		log.Printf("[graph] built %s graph (focus=%s) in %s", params.Lens, params.FocusNodeID, c.now().Sub(startedAt))
	} else {
		log.Printf("[graph] built %s graph in %s", params.Lens, c.now().Sub(startedAt))
	}
	return response, nil
}

// Invalidate clears all cached projections for the requested lenses.
// For the flow lens this clears ALL focus variants.
func (c *ProjectionCache) Invalidate(lenses ...Lens) {
	if len(lenses) == 0 {
		return
	}

	lensSet := make(map[Lens]struct{}, len(lenses))
	for _, l := range lenses {
		lensSet[l] = struct{}{}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if _, match := lensSet[key.Lens]; match {
			delete(c.entries, key)
		}
	}
}

// InvalidateFocus clears the cached operations projection for a specific focus node.
func (c *ProjectionCache) InvalidateFocus(focusNodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, cacheKey{Lens: LensOperations, FocusNodeID: focusNodeID})
}

func (c *ProjectionCache) staleOrError(key cacheKey, buildErr error) (GraphResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return GraphResponse{}, buildErr
	}

	log.Printf("[graph] serving stale %s graph after rebuild error: %v", key.Lens, buildErr)
	return entry.response, nil
}
