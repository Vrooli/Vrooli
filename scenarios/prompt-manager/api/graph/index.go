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

// GraphIndexStore manages the persisted graph index.
type GraphIndexStore struct {
	storeDir string
	builder  graphBuilder
	mu       sync.Mutex
}

// NewIndexStore creates a new graph index store.
func NewIndexStore(storeDir string, builder graphBuilder) *GraphIndexStore {
	return &GraphIndexStore{
		storeDir: storeDir,
		builder:  builder,
	}
}

// indexPath returns the path to the graph index file.
func (s *GraphIndexStore) indexPath() string {
	return filepath.Join(s.storeDir, "indexes", "graph.index.json")
}

// Get loads the index from file, regenerating if missing.
func (s *GraphIndexStore) Get(ctx context.Context) (*GraphIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try reading existing index
	data, err := os.ReadFile(s.indexPath())
	if err == nil {
		var idx GraphIndex
		if err := json.Unmarshal(data, &idx); err == nil {
			return &idx, nil
		}
	}

	// Regenerate if missing or corrupt
	return s.regenerateLocked(ctx)
}

// Invalidate deletes the index file so it will be regenerated on next read.
func (s *GraphIndexStore) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = os.Remove(s.indexPath())
}

// Regenerate rebuilds the index from scratch.
func (s *GraphIndexStore) Regenerate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.regenerateLocked(ctx)
	return err
}

// regenerateLocked performs the actual regeneration. Caller must hold s.mu.
func (s *GraphIndexStore) regenerateLocked(ctx context.Context) (*GraphIndex, error) {
	g, err := s.builder.Build(ctx)
	if err != nil {
		return nil, err
	}

	idx := NewGraphIndex(g)

	// Ensure directory exists
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
