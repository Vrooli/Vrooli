package aisearch

import (
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
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

// InitiativeStoreAdapter exposes initiatives.Store as an aisearch reader.
// The Service-level API returns rollups; aisearch only needs raw initiatives,
// so we bypass rollup computation and read from the Store directly.
type InitiativeStoreAdapter struct {
	store *initiatives.Store
}

// NewInitiativeStoreAdapter wraps an initiatives.Store for use as an aisearch
// reader.
func NewInitiativeStoreAdapter(store *initiatives.Store) *InitiativeStoreAdapter {
	return &InitiativeStoreAdapter{store: store}
}

// List returns every initiative on disk.
func (a *InitiativeStoreAdapter) List() ([]initiatives.Initiative, error) {
	return a.store.LoadAll()
}

// Get loads a single initiative by name.
func (a *InitiativeStoreAdapter) Get(name string) (*initiatives.Initiative, error) {
	return a.store.Load(name)
}
