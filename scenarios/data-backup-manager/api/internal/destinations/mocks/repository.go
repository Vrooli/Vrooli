// Package mocks holds co-located test doubles for the destinations domain seams.
// Lives in a mocks/ directory (no _test.go suffix) so sibling _test.go files
// can import it; never linked into production.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"data-backup-manager/internal/destinations"
)

// FakeRepository is an in-memory destinations.Repository. It behaves like a
// real store (keyed by name, with a separate id index and clock-driven
// timestamps) so the service's CRUD logic is genuinely exercised — not stubbed.
// Per-method error knobs inject failures; atomic counters let tests assert call
// counts.
type FakeRepository struct {
	mu sync.Mutex

	byName map[string]destinations.Destination
	byID   map[string]destinations.Destination

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
		byName: map[string]destinations.Destination{},
		byID:   map[string]destinations.Destination{},
		now:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (f *FakeRepository) tick() time.Time {
	f.seq++
	return f.now.Add(time.Duration(f.seq) * time.Second)
}

func (f *FakeRepository) Create(_ context.Context, d destinations.Destination) (destinations.Destination, error) {
	f.Creates.Add(1)
	if f.CreateErr != nil {
		return destinations.Destination{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if d.ID == "" {
		f.ider++
		d.ID = "dst-" + itoa(f.ider)
	}
	ts := f.tick()
	d.CreatedAt = ts
	d.UpdatedAt = ts
	f.byName[d.Name] = d
	f.byID[d.ID] = d
	return d, nil
}

func (f *FakeRepository) Update(_ context.Context, d destinations.Destination) (destinations.Destination, error) {
	f.Updates.Add(1)
	if f.UpdateErr != nil {
		return destinations.Destination{}, f.UpdateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	d.UpdatedAt = f.tick()
	f.byName[d.Name] = d
	f.byID[d.ID] = d
	return d, nil
}

func (f *FakeRepository) GetByID(_ context.Context, id string) (destinations.Destination, error) {
	if f.GetErr != nil {
		return destinations.Destination{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.byID[id]
	if !ok {
		return destinations.Destination{}, destinations.ErrDestinationNotFound{ID: id}
	}
	return d, nil
}

func (f *FakeRepository) GetByName(_ context.Context, name string) (destinations.Destination, error) {
	if f.GetErr != nil {
		return destinations.Destination{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.byName[name]
	if !ok {
		return destinations.Destination{}, destinations.ErrDestinationNotFound{Name: name}
	}
	return d, nil
}

func (f *FakeRepository) List(_ context.Context, limit int) ([]destinations.Destination, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	if limit <= 0 {
		return nil, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]destinations.Destination, 0, len(f.byName))
	for _, d := range f.byName {
		out = append(out, d)
	}
	sortDestinations(out)
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
	d, ok := f.byID[id]
	if !ok {
		return false, nil
	}
	delete(f.byName, d.Name)
	delete(f.byID, id)
	return true, nil
}

// Compile-time guarantee.
var _ destinations.Repository = (*FakeRepository)(nil)

func sortDestinations(ds []destinations.Destination) {
	for i := 1; i < len(ds); i++ {
		for j := i; j > 0; j-- {
			if ds[j-1].Name > ds[j].Name {
				ds[j-1], ds[j] = ds[j], ds[j-1]
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
