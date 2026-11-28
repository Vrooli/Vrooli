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
	mu       sync.Mutex
	nodes    map[string]registry.Node
	receipts map[string]registry.NodeUpdateReceipt

	CreateErr        error
	GetErr           error
	ListErr          error
	UpdateErr        error
	RevokeErr        error
	RemoveErr        error
	TouchErr         error
	ArchitectureErr  error
	TouchLastSeenIDs []string

	CreateCalls atomic.Int64
	RevokeCalls atomic.Int64
	TouchCalls  atomic.Int64

	// now is the timestamp Create stamps; tests may set it for determinism.
	Now time.Time
}

// NewFakeRepository constructs an empty fake.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{nodes: make(map[string]registry.Node), receipts: make(map[string]registry.NodeUpdateReceipt)}
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
	if n.OwnerRevision <= 0 {
		n.OwnerRevision = 1
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

func (f *FakeRepository) UpdateCapabilityInventory(_ context.Context, id string, observations []registry.CapabilityObservation, probedAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[id]
	if !ok {
		return registry.ErrNodeNotFound{ID: id}
	}
	n.CapabilityInventory = append([]registry.CapabilityObservation(nil), observations...)
	n.CapabilityProbedAt = probedAt
	f.nodes[id] = n
	return nil
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
	existing.OwnerRevision++
	f.nodes[n.ID] = existing
	return existing, nil
}

func (f *FakeRepository) UpdateCAS(_ context.Context, m registry.CASMutation) (registry.UpdateCASResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if prior, ok := f.receipts[m.RequestID]; ok {
		if prior.ActorID == m.ActorID && prior.TargetID == m.TargetID && prior.Subject == m.Subject && prior.ExpectedRevision == m.ExpectedRevision && prior.PayloadDigest == m.PayloadDigest {
			return registry.UpdateCASResult{Node: f.nodes[m.TargetID], Receipt: prior}, nil
		}
		return registry.UpdateCASResult{}, registry.ErrRequestIdentityConflict{RequestID: m.RequestID}
	}
	n, ok := f.nodes[m.TargetID]
	if !ok {
		return registry.UpdateCASResult{}, registry.ErrNodeNotFound{ID: m.TargetID}
	}
	if n.Revoked() {
		return registry.UpdateCASResult{}, registry.ErrNodeRevoked{ID: m.TargetID}
	}
	if n.OwnerRevision != m.ExpectedRevision {
		return registry.UpdateCASResult{}, registry.ErrRevisionConflict{ID: m.TargetID, Expected: m.ExpectedRevision, Current: n.OwnerRevision}
	}
	n.Name = m.Node.Name
	n.Endpoint = m.Node.Endpoint
	n.Capabilities = m.Node.Capabilities
	n.Scopes = m.Node.Scopes
	n.Revision = m.Node.Revision
	n.Kind = m.Node.Kind
	n.DeploymentTier = m.Node.DeploymentTier
	n.OwnerRevision++
	n.UpdatedAt = f.now()
	f.nodes[m.TargetID] = n
	r := registry.NodeUpdateReceipt{RequestID: m.RequestID, ActorID: m.ActorID, TargetID: m.TargetID, Subject: m.Subject, ExpectedRevision: m.ExpectedRevision, NewRevision: n.OwnerRevision, PayloadDigest: m.PayloadDigest, CreatedAt: f.now()}
	f.receipts[m.RequestID] = r
	return registry.UpdateCASResult{Node: n, Receipt: r}, nil
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

func (f *FakeRepository) UpdateArchitecture(_ context.Context, id, machineArch, binaryArch string) error {
	if f.ArchitectureErr != nil {
		return f.ArchitectureErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[id]
	if !ok {
		return registry.ErrNodeNotFound{ID: id}
	}
	if machineArch != "" {
		n.Arch = machineArch
	}
	n.MachineArch = machineArch
	n.BinaryArch = binaryArch
	n.UpdatedAt = f.now()
	f.nodes[id] = n
	return nil
}

func (f *FakeRepository) SetEnrollmentGeneration(_ context.Context, id string, generation int64) (registry.Node, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[id]
	if !ok {
		return registry.Node{}, registry.ErrNodeNotFound{ID: id}
	}
	if generation <= 0 {
		return registry.Node{}, registry.ErrInvalidNode{Field: "enrollment_generation", Reason: "must be positive"}
	}
	if generation < n.EnrollmentGeneration {
		return registry.Node{}, registry.ErrEnrollmentGenerationConflict{ID: id, Current: n.EnrollmentGeneration, Attempted: generation}
	}
	n.EnrollmentGeneration = generation
	n.UpdatedAt = f.now()
	f.nodes[id] = n
	return n, nil
}

// Seed inserts a node directly for test setup, bypassing Create's stamping.
func (f *FakeRepository) Seed(n registry.Node) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nodes[n.ID] = n
}
