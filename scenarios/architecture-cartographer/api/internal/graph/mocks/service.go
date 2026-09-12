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
	MetaCalls    atomic.Int64
	ListCalls    atomic.Int64
	ClearCalls   atomic.Int64

	RetentionPreview    graph.SnapshotRetentionPreview
	RetentionResult     graph.RetentionResult
	RetentionErr        error
	RetentionApplyCalls int
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

func (f *FakeService) LatestSnapshotMeta(_ context.Context, scenario string) (graph.GraphSnapshotMeta, error) {
	f.MetaCalls.Add(1)
	if f.NextErr != nil {
		return graph.GraphSnapshotMeta{}, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.Snapshots) - 1; i >= 0; i-- {
		s := f.Snapshots[i]
		if s.Scenario == scenario {
			return graph.GraphSnapshotMeta{
				ID:                s.ID,
				Scenario:          s.Scenario,
				ContentHash:       s.ContentHash,
				SourceFingerprint: s.SourceFingerprint,
				ExtractedAt:       s.ExtractedAt,
				ExtractionMS:      s.ExtractionMS,
			}, nil
		}
	}
	return graph.GraphSnapshotMeta{}, graph.ErrSnapshotNotFound{ID: "scenario=" + scenario}
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

// PreviewSnapshotRetention reports what retention would remove, computed from
// the fake's in-memory snapshots so a handler test sees realistic numbers.
func (f *FakeService) PreviewSnapshotRetention(_ context.Context, keepPerScenario int) (graph.SnapshotRetentionPreview, error) {
	if f.RetentionErr != nil {
		return graph.SnapshotRetentionPreview{}, f.RetentionErr
	}
	return f.RetentionPreview, nil
}

// ApplySnapshotRetention returns the canned retention result.
func (f *FakeService) ApplySnapshotRetention(_ context.Context, keepPerScenario int) (graph.RetentionResult, error) {
	f.RetentionApplyCalls++
	if f.RetentionErr != nil {
		return graph.RetentionResult{}, f.RetentionErr
	}
	return f.RetentionResult, nil
}

var _ graph.Service = (*FakeService)(nil)
