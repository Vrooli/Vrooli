package mocks

import (
	"context"
	"sync"

	"network-manager/internal/snapshot"
)

type Repository struct {
	mu    sync.Mutex
	items []snapshot.Snapshot
}

func NewRepository() *Repository { return &Repository{} }

func (r *Repository) Create(_ context.Context, s snapshot.Snapshot) (snapshot.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.ID == "" {
		s.ID = "snapshot-test"
	}
	r.items = append([]snapshot.Snapshot{s}, r.items...)
	return s, nil
}

func (r *Repository) List(context.Context) ([]snapshot.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]snapshot.Snapshot, len(r.items))
	copy(out, r.items)
	return out, nil
}

func (r *Repository) Get(_ context.Context, id string) (snapshot.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return snapshot.Snapshot{}, snapshot.ErrNotFound
}

func (r *Repository) Count(context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.items), nil
}

var _ snapshot.Repository = (*Repository)(nil)
