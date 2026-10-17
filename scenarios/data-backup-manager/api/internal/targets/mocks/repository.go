// Package mocks holds co-located test doubles for the targets domain seams.
// Lives in a mocks/ directory (no _test.go suffix) so sibling _test.go files
// can import it; never linked into production.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"data-backup-manager/internal/targets"
)

// FakeRepository is an in-memory targets.Repository. It behaves like a real
// store (keyed by owner+name, with a separate id index and clock-driven
// timestamps) so the service's idempotent-upsert logic is genuinely exercised
// — not stubbed. Per-method error knobs inject failures; atomic counters let
// tests assert call counts.
type FakeRepository struct {
	mu sync.Mutex

	byKey map[string]targets.Target // "owner\x00name" -> target
	byID  map[string]targets.Target

	// Monotonic clock for deterministic CreatedAt/UpdatedAt. Defaults to a
	// fixed base advanced by one second per write.
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
		byKey: map[string]targets.Target{},
		byID:  map[string]targets.Target{},
		now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func keyOf(owner, name string) string { return owner + "\x00" + name }

func (f *FakeRepository) tick() time.Time {
	f.seq++
	return f.now.Add(time.Duration(f.seq) * time.Second)
}

func (f *FakeRepository) Create(_ context.Context, t targets.Target) (targets.Target, error) {
	f.Creates.Add(1)
	if f.CreateErr != nil {
		return targets.Target{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.ID == "" {
		f.ider++
		t.ID = "tgt-" + itoa(f.ider)
	}
	ts := f.tick()
	t.CreatedAt = ts
	t.UpdatedAt = ts
	f.byKey[keyOf(t.Owner, t.Name)] = t
	f.byID[t.ID] = t
	return t, nil
}

func (f *FakeRepository) Update(_ context.Context, t targets.Target) (targets.Target, error) {
	f.Updates.Add(1)
	if f.UpdateErr != nil {
		return targets.Target{}, f.UpdateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t.UpdatedAt = f.tick()
	f.byKey[keyOf(t.Owner, t.Name)] = t
	f.byID[t.ID] = t
	return t, nil
}

func (f *FakeRepository) GetByOwnerName(_ context.Context, owner, name string) (targets.Target, error) {
	if f.GetErr != nil {
		return targets.Target{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byKey[keyOf(owner, name)]
	if !ok {
		return targets.Target{}, targets.ErrTargetNotFound{Owner: owner, Name: name}
	}
	return t, nil
}

func (f *FakeRepository) GetByID(_ context.Context, id string) (targets.Target, error) {
	if f.GetErr != nil {
		return targets.Target{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[id]
	if !ok {
		return targets.Target{}, targets.ErrTargetNotFound{ID: id}
	}
	return t, nil
}

func (f *FakeRepository) List(_ context.Context, owner string, limit int) ([]targets.Target, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	if limit <= 0 {
		return nil, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]targets.Target, 0, len(f.byKey))
	for _, t := range f.byKey {
		if owner != "" && t.Owner != owner {
			continue
		}
		out = append(out, t)
	}
	// Deterministic order: owner, then name.
	sortTargets(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeRepository) DeleteByOwnerName(_ context.Context, owner, name string) (bool, error) {
	f.Deletes.Add(1)
	if f.DeleteErr != nil {
		return false, f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	k := keyOf(owner, name)
	t, ok := f.byKey[k]
	if !ok {
		return false, nil
	}
	delete(f.byKey, k)
	delete(f.byID, t.ID)
	return true, nil
}

// Compile-time guarantee.
var _ targets.Repository = (*FakeRepository)(nil)

func sortTargets(ts []targets.Target) {
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0; j-- {
			a, b := ts[j-1], ts[j]
			if a.Owner > b.Owner || (a.Owner == b.Owner && a.Name > b.Name) {
				ts[j-1], ts[j] = ts[j], ts[j-1]
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
