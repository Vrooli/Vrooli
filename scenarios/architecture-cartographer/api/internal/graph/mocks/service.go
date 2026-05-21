package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"architecture-cartographer/internal/graph"
)

// FakeService satisfies graph.Service for handler tests.
type FakeService struct {
	mu sync.Mutex

	Snapshots []graph.GraphSnapshot
	NextErr   error

	FromCache bool

	ExtractCalls atomic.Int64
	GetCalls     atomic.Int64
	ListCalls    atomic.Int64
	ClearCalls   atomic.Int64
}

func (f *FakeService) ExtractGraph(_ context.Context, in graph.ExtractGraphInput) (graph.GraphSnapshot, bool, error) {
	f.ExtractCalls.Add(1)
	if f.NextErr != nil {
		return graph.GraphSnapshot{}, false, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.Snapshots) == 0 {
		return graph.GraphSnapshot{Scenario: in.Scenario}, f.FromCache, nil
	}
	return f.Snapshots[0], f.FromCache, nil
}

func (f *FakeService) GetSnapshot(_ context.Context, id string) (graph.GraphSnapshot, error) {
	f.GetCalls.Add(1)
	if f.NextErr != nil {
		return graph.GraphSnapshot{}, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.Snapshots {
		if s.ID == id {
			return s, nil
		}
	}
	return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{ID: id}
}

func (f *FakeService) ListSnapshots(_ context.Context, _ graph.ListSnapshotsFilter) (graph.SnapshotPage, error) {
	f.ListCalls.Add(1)
	if f.NextErr != nil {
		return graph.SnapshotPage{}, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]graph.GraphSnapshot, len(f.Snapshots))
	copy(out, f.Snapshots)
	return graph.SnapshotPage{Snapshots: out}, nil
}

func (f *FakeService) ClearSnapshots(_ context.Context, _ string, dryRun bool) (int, bool, error) {
	f.ClearCalls.Add(1)
	if f.NextErr != nil {
		return 0, dryRun, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	n := len(f.Snapshots)
	if !dryRun {
		f.Snapshots = nil
	}
	return n, dryRun, nil
}

var _ graph.Service = (*FakeService)(nil)
