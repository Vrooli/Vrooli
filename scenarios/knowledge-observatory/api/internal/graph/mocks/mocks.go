// Package mocks holds an in-memory graph.Repository for testing callers.
package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"knowledge-observatory/internal/graph"
)

// Repository is an in-memory graph.Repository keyed like the real unique
// constraint on (source_id, target_id, relationship_type).
type Repository struct {
	mu    sync.Mutex
	Edges map[string]graph.Edge

	// Err, when set, is returned by every method.
	Err error
}

var _ graph.Repository = (*Repository)(nil)

// New returns an empty repository.
func New() *Repository { return &Repository{Edges: map[string]graph.Edge{}} }

func edgeKey(e graph.Edge) string {
	return fmt.Sprintf("%s\x00%s\x00%s", e.SourceID, e.TargetID, e.RelationshipType)
}

func (r *Repository) UpsertEdges(_ context.Context, edges []graph.Edge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	for _, e := range edges {
		// Mirror the SQLite implementation: degenerate edges are skipped, not
		// rejected, so a bad pair never discards a batch.
		if e.SourceID == "" || e.TargetID == "" || e.SourceID == e.TargetID {
			continue
		}
		if e.RelationshipType == "" {
			e.RelationshipType = "semantic_similarity"
		}
		if e.ID == "" {
			e.ID = edgeKey(e)
		}
		e.DiscoveredAt = time.Now().UTC()
		r.Edges[edgeKey(e)] = e
	}
	return nil
}

func (r *Repository) ListEdges(_ context.Context, limit int) ([]graph.Edge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return nil, r.Err
	}
	if limit <= 0 {
		limit = 500
	}
	out := make([]graph.Edge, 0, len(r.Edges))
	for _, e := range r.Edges {
		if len(out) == limit {
			break
		}
		out = append(out, e)
	}
	return out, nil
}

func (r *Repository) CountEdges(context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	return int64(len(r.Edges)), nil
}
