// Package mocks holds an in-memory search.Repository for testing callers.
package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"knowledge-observatory/internal/search"
)

// Repository is an in-memory search.Repository.
type Repository struct {
	mu      sync.Mutex
	History []search.History

	// Err, when set, is returned by every method.
	Err error
}

var _ search.Repository = (*Repository)(nil)

// New returns an empty repository.
func New() *Repository { return &Repository{} }

func (r *Repository) InsertHistory(_ context.Context, h search.History) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return "", r.Err
	}
	if h.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if h.ID == "" {
		h.ID = fmt.Sprintf("search-%d", len(r.History)+1)
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now().UTC()
	}
	r.History = append(r.History, h)
	return h.ID, nil
}

func (r *Repository) RecentHistory(_ context.Context, limit int) ([]search.History, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return nil, r.Err
	}
	if limit <= 0 {
		limit = 50
	}
	out := make([]search.History, 0, limit)
	for i := len(r.History) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.History[i])
	}
	return out, nil
}

func (r *Repository) CountHistory(context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	return int64(len(r.History)), nil
}
