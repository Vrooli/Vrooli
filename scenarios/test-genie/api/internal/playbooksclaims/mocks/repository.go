// Package mocks provides test doubles for playbooksclaims.Repository.
package mocks

import (
	"context"
	"sync"
	"time"

	"test-genie/internal/playbooksclaims"
)

// FakeRepository is an in-memory Repository for tests.
type FakeRepository struct {
	mu     sync.Mutex
	claims map[string]playbooksclaims.Claim
}

var _ playbooksclaims.Repository = (*FakeRepository)(nil)

// NewFakeRepository returns an empty FakeRepository.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{claims: make(map[string]playbooksclaims.Claim)}
}

// TryAcquire implements playbooksclaims.Repository.
func (f *FakeRepository) TryAcquire(ctx context.Context, in playbooksclaims.AcquireInput, now time.Time, ttl time.Duration) (playbooksclaims.Claim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.claims[in.ScenarioName]; ok {
		if existing.ExpiresAt.After(now) {
			return playbooksclaims.Claim{}, &playbooksclaims.ErrBusy{Holder: existing}
		}
	}
	c := playbooksclaims.Claim{
		ScenarioName: in.ScenarioName,
		RunID:        in.RunID,
		Mode:         in.Mode,
		StartedBy:    in.StartedBy,
		AcquiredAt:   now,
		HeartbeatAt:  now,
		ExpiresAt:    now.Add(ttl),
	}
	f.claims[in.ScenarioName] = c
	return c, nil
}

// Heartbeat implements playbooksclaims.Repository.
func (f *FakeRepository) Heartbeat(ctx context.Context, scenarioName, runID string, now time.Time, ttl time.Duration) (playbooksclaims.Claim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.claims[scenarioName]
	if !ok {
		return playbooksclaims.Claim{}, playbooksclaims.ErrNotFound
	}
	if c.RunID != runID {
		return playbooksclaims.Claim{}, playbooksclaims.ErrLeaseMismatch
	}
	c.HeartbeatAt = now
	c.ExpiresAt = now.Add(ttl)
	f.claims[scenarioName] = c
	return c, nil
}

// Release implements playbooksclaims.Repository.
func (f *FakeRepository) Release(ctx context.Context, scenarioName, runID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.claims[scenarioName]
	if !ok {
		return playbooksclaims.ErrNotFound
	}
	if c.RunID != runID {
		return playbooksclaims.ErrLeaseMismatch
	}
	delete(f.claims, scenarioName)
	return nil
}

// Get implements playbooksclaims.Repository.
func (f *FakeRepository) Get(ctx context.Context, scenarioName string) (playbooksclaims.Claim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.claims[scenarioName]
	if !ok {
		return playbooksclaims.Claim{}, playbooksclaims.ErrNotFound
	}
	return c, nil
}

// List implements playbooksclaims.Repository.
func (f *FakeRepository) List(ctx context.Context) ([]playbooksclaims.Claim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]playbooksclaims.Claim, 0, len(f.claims))
	for _, c := range f.claims {
		out = append(out, c)
	}
	return out, nil
}

// ForceBreak implements playbooksclaims.Repository.
func (f *FakeRepository) ForceBreak(ctx context.Context, scenarioName string) (playbooksclaims.Claim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.claims[scenarioName]
	if !ok {
		return playbooksclaims.Claim{}, playbooksclaims.ErrNotFound
	}
	delete(f.claims, scenarioName)
	return c, nil
}
