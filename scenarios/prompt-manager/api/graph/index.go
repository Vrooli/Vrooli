// DOC: docs/concepts/GRAPH.md#index-persistence
package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// graphBuilder builds a complete graph from all sources.
type graphBuilder interface {
	Build(ctx context.Context) (Graph, error)
}

// GraphIndexStore manages the persisted graph index with an in-memory cache.
//
// Performance design:
//   - Get() returns the in-memory cached index if available (no disk I/O).
//   - On cache miss, loads from the on-disk index file.
//   - On disk miss, triggers a synchronous rebuild.
//   - Invalidate() clears the in-memory cache and deletes the disk file.
//   - Regenerate() forces a synchronous rebuild and updates the cache.
type GraphIndexStore struct {
	cacheRoot string
	builder   graphBuilder
	mu        sync.RWMutex
	cached    *GraphIndex // in-memory cache; nil means cold
}

// NewIndexStore creates a new graph index store. cacheRoot is the scenario's
// runtime cache class root (see paths.Roots); the graph index is fully
// reconstructable from authored sources via Regenerate.
func NewIndexStore(cacheRoot string, builder graphBuilder) *GraphIndexStore {
	return &GraphIndexStore{
		cacheRoot: cacheRoot,
		builder:   builder,
	}
}

// indexPath returns the path to the graph index file under the cache root.
func (s *GraphIndexStore) indexPath() string {
	return filepath.Join(s.cacheRoot, "indexes", "graph.index.json")
}

// Get returns the graph index, using the in-memory cache when available.
//
// Fast path (read lock): returns cached *GraphIndex immediately.
// Slow path (write lock): loads from disk or regenerates, then caches.
func (s *GraphIndexStore) Get(ctx context.Context) (*GraphIndex, error) {
	// Fast path: return from in-memory cache (read lock only).
	s.mu.RLock()
	if s.cached != nil {
		idx := s.cached
		s.mu.RUnlock()
		return idx, nil
	}
	s.mu.RUnlock()

	// Slow path: acquire write lock, double-check, then load or rebuild.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check: another goroutine may have populated the cache.
	if s.cached != nil {
		return s.cached, nil
	}

	// Try loading from the on-disk index file.
	data, err := os.ReadFile(s.indexPath())
	if err == nil {
		var idx GraphIndex
		if err := json.Unmarshal(data, &idx); err == nil {
			s.cached = &idx
			return &idx, nil
		}
	}

	// Regenerate if missing or corrupt.
	return s.regenerateLocked(ctx)
}

// Invalidate clears the in-memory cache and deletes the on-disk index file.
// The next Get() call will trigger a synchronous rebuild.
func (s *GraphIndexStore) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = nil
	_ = os.Remove(s.indexPath())
}

// Regenerate rebuilds the index from scratch (synchronous).
func (s *GraphIndexStore) Regenerate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.regenerateLocked(ctx)
	return err
}

// regenerateLocked performs a synchronous rebuild and updates the in-memory cache.
// Caller must hold s.mu (write lock).
func (s *GraphIndexStore) regenerateLocked(ctx context.Context) (*GraphIndex, error) {
	idx, err := s.rebuildAndPersist(ctx)
	if err != nil {
		return nil, err
	}
	s.cached = idx
	return idx, nil
}

// rebuildAndPersist runs the builder and writes the result to disk.
// Caller must hold s.mu (write lock).
func (s *GraphIndexStore) rebuildAndPersist(ctx context.Context) (*GraphIndex, error) {
	g, err := s.builder.Build(ctx)
	if err != nil {
		return nil, err
	}

	idx := NewGraphIndex(g)

	// Ensure directory exists.
	dir := filepath.Dir(s.indexPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(s.indexPath(), data, 0o644); err != nil {
		return nil, err
	}

	return idx, nil
}
