// Package mocks holds the runs domain's co-located test fakes. Deleting
// internal/runs/ takes these with it.
package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"vrooli-bridge/internal/runs"

	"github.com/google/uuid"
)

// FakeRepository is an in-memory runs.Repository with per-method error knobs.
// Used by service tests to drive the service against a controllable persistence
// layer without sqlite.
type FakeRepository struct {
	mu     sync.Mutex
	runs   map[string]runs.Run
	events map[string][]runs.RunEvent

	CreateErr      error
	GetErr         error
	ListErr        error
	UpdateErr      error
	AppendEventErr error
	ListEventsErr  error

	// Now is the timestamp Create stamps; tests may set it for determinism.
	Now time.Time
}

// NewFakeRepository constructs an empty fake.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		runs:   make(map[string]runs.Run),
		events: make(map[string][]runs.RunEvent),
	}
}

var _ runs.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) now() time.Time {
	if !f.Now.IsZero() {
		return f.Now
	}
	return time.Unix(0, 0).UTC()
}

func (f *FakeRepository) Create(_ context.Context, r runs.Run) (runs.Run, error) {
	if f.CreateErr != nil {
		return runs.Run{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = f.now()
	}
	if r.Status == runs.StatusUnspecified {
		r.Status = runs.StatusQueued
	}
	f.runs[r.ID] = r
	return r, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (runs.Run, error) {
	if f.GetErr != nil {
		return runs.Run{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runs[id]
	if !ok {
		return runs.Run{}, runs.ErrRunNotFound{ID: id}
	}
	return r, nil
}

func (f *FakeRepository) List(_ context.Context, filter runs.ListFilter) ([]runs.Run, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]runs.Run, 0, len(f.runs))
	for _, r := range f.runs {
		if filter.NodeID != "" && r.NodeID != filter.NodeID {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (f *FakeRepository) Update(_ context.Context, r runs.Run) (runs.Run, error) {
	if f.UpdateErr != nil {
		return runs.Run{}, f.UpdateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.runs[r.ID]
	if !ok {
		return runs.Run{}, runs.ErrRunNotFound{ID: r.ID}
	}
	existing.Status = r.Status
	existing.ExitCode = r.ExitCode
	existing.StartedAt = r.StartedAt
	existing.FinishedAt = r.FinishedAt
	existing.ArtifactRefs = r.ArtifactRefs
	f.runs[r.ID] = existing
	return existing, nil
}

func (f *FakeRepository) AppendEvent(_ context.Context, ev runs.RunEvent) error {
	if f.AppendEventErr != nil {
		return f.AppendEventErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	// Mirror the sqlite INSERT OR IGNORE de-dup on (run_id, sequence).
	for _, existing := range f.events[ev.RunID] {
		if existing.Sequence == ev.Sequence {
			return nil
		}
	}
	f.events[ev.RunID] = append(f.events[ev.RunID], ev)
	return nil
}

func (f *FakeRepository) ListEvents(_ context.Context, runID string) ([]runs.RunEvent, error) {
	if f.ListEventsErr != nil {
		return nil, f.ListEventsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	evs := append([]runs.RunEvent(nil), f.events[runID]...)
	sort.Slice(evs, func(i, j int) bool { return evs[i].Sequence < evs[j].Sequence })
	return evs, nil
}

// Seed inserts a run directly for test setup, bypassing Create's stamping.
func (f *FakeRepository) Seed(r runs.Run) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runs[r.ID] = r
}
