package mocks

import (
	"context"
	"sync"

	"network-manager/internal/policy"
)

type Repository struct {
	mu        sync.Mutex
	changes   map[string]policy.Change
	profiles  map[string]policy.Profile
	approvals []policy.ApprovalRecord
	rollbacks []policy.RollbackRecord
}

func NewRepository() *Repository {
	return &Repository{
		changes:  map[string]policy.Change{},
		profiles: map[string]policy.Profile{},
	}
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

func (r *Repository) ListProfiles(_ context.Context, deviceGroup string) ([]policy.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	profiles := make([]policy.Profile, 0, len(r.profiles))
	for _, profile := range r.profiles {
		if deviceGroup != "" && profile.DeviceGroup != deviceGroup {
			continue
		}
		profiles = append(profiles, cloneProfile(profile))
	}
	return profiles, nil
}

func (r *Repository) UpsertProfile(_ context.Context, profile policy.Profile) (policy.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[profile.ID] = cloneProfile(profile)
	return profile, nil
}

func (r *Repository) GetProfile(_ context.Context, id string) (policy.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, ok := r.profiles[id]
	if !ok {
		return policy.Profile{}, policy.ErrNotFound
	}
	return cloneProfile(profile), nil
}

func cloneChange(change policy.Change) policy.Change {
	change.Values = append([]string(nil), change.Values...)
	change.Effects = append([]string(nil), change.Effects...)
	return change
}

func cloneProfile(profile policy.Profile) policy.Profile {
	profile.Effects = append([]string(nil), profile.Effects...)
	return profile
}

var _ policy.Repository = (*Repository)(nil)
