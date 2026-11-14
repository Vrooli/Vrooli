package aisearch

import (
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

// BacklogStoreAdapter bridges backlog.Store (whose LoadAll takes a kinds
// filter) to the aisearch.BacklogReader interface (which always enumerates
// everything). Zero-value is not usable; construct with NewBacklogStoreAdapter.
type BacklogStoreAdapter struct {
	store backlog.Store
}

// NewBacklogStoreAdapter wraps a backlog.Store for use as an aisearch reader.
func NewBacklogStoreAdapter(store backlog.Store) *BacklogStoreAdapter {
	return &BacklogStoreAdapter{store: store}
}

// LoadAll returns every backlog item across all kinds.
func (a *BacklogStoreAdapter) LoadAll() ([]backlog.BacklogItem, error) {
	return a.store.LoadAll(nil)
}

// LoadItem proxies directly to the underlying store.
func (a *BacklogStoreAdapter) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	return a.store.LoadItem(kind, name)
}

// GoalServiceAdapter exposes goals.Service as an aisearch reader.
type GoalServiceAdapter struct {
	service *goals.Service
}

// NewGoalServiceAdapter wraps a goals.Service for use as an aisearch reader.
func NewGoalServiceAdapter(service *goals.Service) *GoalServiceAdapter {
	return &GoalServiceAdapter{service: service}
}

// List returns every goal on disk.
func (a *GoalServiceAdapter) List() ([]goals.Goal, error) {
	list, err := a.service.List()
	if err != nil {
		return nil, err
	}
	out := make([]goals.Goal, 0, len(list))
	for _, goal := range list {
		out = append(out, goal.Goal)
	}
	return out, nil
}

// Get loads a single goal by name.
func (a *GoalServiceAdapter) Get(name string) (*goals.Goal, error) {
	goal, err := a.service.Get(name)
	if err != nil {
		return nil, err
	}
	return &goal.Goal, nil
}
