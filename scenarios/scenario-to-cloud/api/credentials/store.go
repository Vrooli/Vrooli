package credentials

import (
	"context"
	"sort"
	"sync"

	"scenario-to-cloud/domain"
)

// Store is the durable ledger seam. The persistence package implements it on
// PostgreSQL/SQLite; MemoryStore implements it for tests and canary scans.
// Nothing stored through this interface is ever a credential value.
type Store interface {
	UpsertBinding(ctx context.Context, binding *domain.CredentialBinding) error
	GetBinding(ctx context.Context, deploymentID, bindingID string) (*domain.CredentialBinding, error)
	ListBindings(ctx context.Context, deploymentID string) ([]domain.CredentialBinding, error)
	FindBindingsByDescriptor(ctx context.Context, descriptor domain.CredentialDescriptor) ([]domain.CredentialBinding, error)
	RecordAck(ctx context.Context, ack domain.CredentialAck) error
	ListAcks(ctx context.Context, bindingID string) ([]domain.CredentialAck, error)
	SaveRotation(ctx context.Context, rotation *domain.CredentialRotation) error
	GetRotation(ctx context.Context, id string) (*domain.CredentialRotation, error)
	ListRotations(ctx context.Context, deploymentID string) ([]domain.CredentialRotation, error)
}

// MemoryStore is an in-process Store. Its Dump method lets the canary scanner
// inspect every retained byte.
type MemoryStore struct {
	mu        sync.Mutex
	bindings  map[string]domain.CredentialBinding
	acks      map[string]map[string]domain.CredentialAck
	rotations map[string]domain.CredentialRotation
}

// NewMemoryStore returns an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{bindings: map[string]domain.CredentialBinding{}, acks: map[string]map[string]domain.CredentialAck{}, rotations: map[string]domain.CredentialRotation{}}
}

func (m *MemoryStore) UpsertBinding(_ context.Context, binding *domain.CredentialBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bindings[binding.ID] = cloneBinding(*binding)
	return nil
}

func (m *MemoryStore) GetBinding(_ context.Context, deploymentID, bindingID string) (*domain.CredentialBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	binding, ok := m.bindings[bindingID]
	if !ok || binding.DeploymentID != deploymentID {
		return nil, nil
	}
	out := cloneBinding(binding)
	return &out, nil
}

func (m *MemoryStore) ListBindings(_ context.Context, deploymentID string) ([]domain.CredentialBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.CredentialBinding{}
	for _, binding := range m.bindings {
		if binding.DeploymentID == deploymentID {
			out = append(out, cloneBinding(binding))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *MemoryStore) FindBindingsByDescriptor(_ context.Context, descriptor domain.CredentialDescriptor) ([]domain.CredentialBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.CredentialBinding{}
	for _, binding := range m.bindings {
		if binding.Descriptor == descriptor {
			out = append(out, cloneBinding(binding))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *MemoryStore) RecordAck(_ context.Context, ack domain.CredentialAck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acks[ack.BindingID] == nil {
		m.acks[ack.BindingID] = map[string]domain.CredentialAck{}
	}
	m.acks[ack.BindingID][ack.Consumer] = ack
	return nil
}

func (m *MemoryStore) ListAcks(_ context.Context, bindingID string) ([]domain.CredentialAck, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.CredentialAck{}
	for _, ack := range m.acks[bindingID] {
		out = append(out, ack)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Consumer < out[j].Consumer })
	return out, nil
}

func (m *MemoryStore) SaveRotation(_ context.Context, rotation *domain.CredentialRotation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rotations[rotation.ID] = cloneRotation(*rotation)
	return nil
}

func (m *MemoryStore) GetRotation(_ context.Context, id string) (*domain.CredentialRotation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rotation, ok := m.rotations[id]
	if !ok {
		return nil, nil
	}
	out := cloneRotation(rotation)
	return &out, nil
}

func (m *MemoryStore) ListRotations(_ context.Context, deploymentID string) ([]domain.CredentialRotation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.CredentialRotation{}
	for _, rotation := range m.rotations {
		if rotation.DeploymentID == deploymentID {
			out = append(out, cloneRotation(rotation))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// Dump returns every retained record for leakage scanning.
func (m *MemoryStore) Dump() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{"bindings": m.bindings, "acks": m.acks, "rotations": m.rotations}
}

func cloneBinding(b domain.CredentialBinding) domain.CredentialBinding {
	b.ConsumerRefs = append([]string(nil), b.ConsumerRefs...)
	if b.PreviousVersion != nil {
		prev := *b.PreviousVersion
		b.PreviousVersion = &prev
	}
	return b
}

func cloneRotation(r domain.CredentialRotation) domain.CredentialRotation {
	r.Consumers = append([]domain.CredentialConsumerProgress(nil), r.Consumers...)
	r.Unreached = append([]string(nil), r.Unreached...)
	r.Receipts = append([]domain.CredentialReceipt(nil), r.Receipts...)
	if r.PendingOperatorInput != nil {
		h := *r.PendingOperatorInput
		r.PendingOperatorInput = &h
	}
	if r.Error != nil {
		e := *r.Error
		r.Error = &e
	}
	if r.BreakGlass != nil {
		w := *r.BreakGlass
		r.BreakGlass = &w
	}
	return r
}
