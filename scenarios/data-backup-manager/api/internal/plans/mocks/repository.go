// Package mocks holds co-located test doubles for the plans domain seams.
// Lives in a mocks/ directory (no _test.go suffix) so sibling _test.go files
// can import it; never linked into production.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"data-backup-manager/internal/plans"
)

// FakeRepository is an in-memory plans.Repository. It behaves like a real store
// (keyed by id) so the service's CRUD logic is genuinely exercised — not
// stubbed. Per-method error knobs inject failures; atomic counters let tests
// assert call counts.
type FakeRepository struct {
	mu sync.Mutex

	byID map[string]plans.Plan

	now  time.Time
	seq  int
	ider int

	CreateErr error
	UpdateErr error
	GetErr    error
	ListErr   error
	DeleteErr error

	Creates atomic.Int64
	Updates atomic.Int64
	Deletes atomic.Int64
}

// NewFakeRepository returns an empty store with a deterministic clock base.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		byID: map[string]plans.Plan{},
		now:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (f *FakeRepository) tick() time.Time {
	f.seq++
	return f.now.Add(time.Duration(f.seq) * time.Second)
}

func (f *FakeRepository) Create(_ context.Context, p plans.Plan) (plans.Plan, error) {
	f.Creates.Add(1)
	if f.CreateErr != nil {
		return plans.Plan{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if p.ID == "" {
		f.ider++
		p.ID = "plan-" + itoa(f.ider)
	}
	ts := f.tick()
	p.CreatedAt = ts
	p.UpdatedAt = ts
	f.byID[p.ID] = p
	return p, nil
}

func (f *FakeRepository) Update(_ context.Context, p plans.Plan) (plans.Plan, error) {
	f.Updates.Add(1)
	if f.UpdateErr != nil {
		return plans.Plan{}, f.UpdateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.byID[p.ID]
	if !ok {
		return plans.Plan{}, plans.ErrPlanNotFound{ID: p.ID}
	}
	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = f.tick()
	f.byID[p.ID] = p
	return p, nil
}

func (f *FakeRepository) GetByID(_ context.Context, id string) (plans.Plan, error) {
	if f.GetErr != nil {
		return plans.Plan{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok {
		return plans.Plan{}, plans.ErrPlanNotFound{ID: id}
	}
	return p, nil
}

func (f *FakeRepository) List(_ context.Context, limit int) ([]plans.Plan, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	if limit <= 0 {
		return nil, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]plans.Plan, 0, len(f.byID))
	for _, p := range f.byID {
		out = append(out, p)
	}
	sortPlans(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeRepository) Delete(_ context.Context, id string) (bool, error) {
	f.Deletes.Add(1)
	if f.DeleteErr != nil {
		return false, f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.byID[id]
	if !ok {
		return false, nil
	}
	delete(f.byID, id)
	return true, nil
}

// Compile-time guarantee.
var _ plans.Repository = (*FakeRepository)(nil)

func sortPlans(ps []plans.Plan) {
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0; j-- {
			if ps[j-1].Name > ps[j].Name {
				ps[j-1], ps[j] = ps[j], ps[j-1]
			} else {
				break
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
