package mocks

import (
	"context"
	"sync"

	"network-manager/internal/policy"
)

type Repository struct {
	mu        sync.Mutex
	changes   map[string]policy.Change
	approvals []policy.ApprovalRecord
	rollbacks []policy.RollbackRecord
}

func NewRepository() *Repository {
	return &Repository{changes: map[string]policy.Change{}}
}

func (r *Repository) SaveChange(_ context.Context, change policy.Change) (policy.Change, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.changes[change.ID] = cloneChange(change)
	return change, nil
}

func (r *Repository) GetChange(_ context.Context, id string) (policy.Change, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	change, ok := r.changes[id]
	if !ok {
		return policy.Change{}, policy.ErrNotFound
	}
	return cloneChange(change), nil
}

func (r *Repository) UpdateChange(_ context.Context, change policy.Change) (policy.Change, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.changes[change.ID]; !ok {
		return policy.Change{}, policy.ErrNotFound
	}
	r.changes[change.ID] = cloneChange(change)
	return change, nil
}

func (r *Repository) SaveApproval(_ context.Context, approval policy.ApprovalRecord) (policy.ApprovalRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.approvals = append(r.approvals, approval)
	return approval, nil
}

func (r *Repository) SaveRollback(_ context.Context, rollback policy.RollbackRecord) (policy.RollbackRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rollbacks = append(r.rollbacks, rollback)
	return rollback, nil
}

func cloneChange(change policy.Change) policy.Change {
	change.Values = append([]string(nil), change.Values...)
	change.Effects = append([]string(nil), change.Effects...)
	return change
}

var _ policy.Repository = (*Repository)(nil)
