// Package mocks holds the registry domain's co-located test fakes. Deleting
// internal/registry/ takes these with it.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"vrooli-bridge/internal/registry"

	"github.com/google/uuid"
)

// FakeRepository is an in-memory registry.Repository with per-method error
// knobs and atomic call counters. Used by service tests to drive the service
// against a controllable persistence layer without sqlite.
type FakeRepository struct {
	mu    sync.Mutex
	nodes map[string]registry.Node

	CreateErr        error
	GetErr           error
	ListErr          error
	UpdateErr        error
	RevokeErr        error
	RemoveErr        error
	TouchErr         error
	TouchLastSeenIDs []string

	CreateCalls atomic.Int64
	RevokeCalls atomic.Int64
	TouchCalls  atomic.Int64

	// now is the timestamp Create stamps; tests may set it for determinism.
	Now time.Time
}

// NewFakeRepository constructs an empty fake.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{nodes: make(map[string]registry.Node)}
}

var _ registry.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) now() time.Time {
	if !f.Now.IsZero() {
		return f.Now
	}
	return time.Unix(0, 0).UTC()
}

func (f *FakeRepository) Create(_ context.Context, n registry.Node) (registry.Node, error) {
	f.CreateCalls.Add(1)
	if f.CreateErr != nil {
		return registry.Node{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = f.now()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}
	f.nodes[n.ID] = n
	return n, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (registry.Node, error) {
	if f.GetErr != nil {
		return registry.Node{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[id]
	if !ok {
		return registry.Node{}, registry.ErrNodeNotFound{ID: id}
	}
	return n, nil
}

func (f *FakeRepository) GetByPairingCorrelation(_ context.Context, correlationID string) (registry.Node, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.nodes {
		if n.PairingCorrelationID == correlationID {
			return n, nil
		}
	}
	return registry.Node{}, registry.ErrNodeNotFound{ID: correlationID}
}

func (f *FakeRepository) List(_ context.Context) ([]registry.Node, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]registry.Node, 0, len(f.nodes))
	for _, n := range f.nodes {
		out = append(out, n)
	}
	return out, nil
}

func (f *FakeRepository) Update(_ context.Context, n registry.Node) (registry.Node, error) {
	if f.UpdateErr != nil {
		return registry.Node{}, f.UpdateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.nodes[n.ID]
	if !ok {
		return registry.Node{}, registry.ErrNodeNotFound{ID: n.ID}
	}
	existing.Name = n.Name
	existing.Endpoint = n.Endpoint
	existing.Capabilities = n.Capabilities
	existing.Scopes = n.Scopes
	existing.Revision = n.Revision
	existing.UpdatedAt = f.now()
	f.nodes[n.ID] = existing
	return existing, nil
}

func (f *FakeRepository) Revoke(_ context.Context, id string) (registry.Node, error) {
	f.RevokeCalls.Add(1)
	if f.RevokeErr != nil {
		return registry.Node{}, f.RevokeErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.nodes[id]
	if !ok {
		return registry.Node{}, registry.ErrNodeNotFound{ID: id}
	}
	if existing.Revoked() {
		return existing, nil
	}
	existing.RevokedAt = f.now()
	existing.UpdatedAt = f.now()
	f.nodes[id] = existing
	return existing, nil
}

func (f *FakeRepository) Remove(_ context.Context, id string) error {
	if f.RemoveErr != nil {
		return f.RemoveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[id]
	if !ok {
		return registry.ErrNodeNotFound{ID: id}
	}
	if !n.Revoked() {
		return registry.ErrNodeActive{ID: id}
	}
	delete(f.nodes, id)
	return nil
}

func (f *FakeRepository) TouchLastSeen(_ context.Context, id string, t time.Time) error {
	f.TouchCalls.Add(1)
	if f.TouchErr != nil {
		return f.TouchErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.TouchLastSeenIDs = append(f.TouchLastSeenIDs, id)
	if existing, ok := f.nodes[id]; ok {
		existing.LastSeenAt = t
		f.nodes[id] = existing
	}
	return nil
}

// Seed inserts a node directly for test setup, bypassing Create's stamping.
func (f *FakeRepository) Seed(n registry.Node) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nodes[n.ID] = n
}
