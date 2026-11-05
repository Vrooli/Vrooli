// Package mocks holds in-memory fakes for the graph domain.
package mocks

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"architecture-cartographer/internal/graph"

	"github.com/google/uuid"
)

// FakeRepository satisfies graph.Repository.
type FakeRepository struct {
	mu sync.Mutex

	Snapshots []graph.GraphSnapshot

	SaveErr  error
	GetErr   error
	FindErr  error
	ListErr  error
	ClearErr error

	SaveCalls       atomic.Int64
	GetCalls        atomic.Int64
	MetaCalls       atomic.Int64
	FindCalls       atomic.Int64
	SourceFindCalls atomic.Int64
	ListCalls       atomic.Int64
	ClearCalls      atomic.Int64
	PruneCalls      atomic.Int64

	PruneErr error
}

func (f *FakeRepository) SaveSnapshot(_ context.Context, s graph.GraphSnapshot) (graph.GraphSnapshot, error) {
	f.SaveCalls.Add(1)
	if f.SaveErr != nil {
		return graph.GraphSnapshot{}, f.SaveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.ExtractedAt.IsZero() {
		s.ExtractedAt = time.Now().UTC()
	}
	// Replace if (scenario, content_hash) already present.
	for i, existing := range f.Snapshots {
		if existing.Scenario == s.Scenario && existing.ContentHash == s.ContentHash {
			f.Snapshots[i] = s
			return s, nil
		}
	}
	f.Snapshots = append(f.Snapshots, s)
	return s, nil
}

func (f *FakeRepository) GetSnapshot(_ context.Context, id string) (graph.GraphSnapshot, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return graph.GraphSnapshot{}, f.GetErr
	}
	return f.findSnapshot(func(s graph.GraphSnapshot) bool { return s.ID == id }, id)
}

func (f *FakeRepository) LatestSnapshotMeta(_ context.Context, scenario string) (graph.GraphSnapshotMeta, error) {
	f.MetaCalls.Add(1)
	if f.GetErr != nil {
		return graph.GraphSnapshotMeta{}, f.GetErr
	}
	s, err := f.findLatestSnapshot(func(s graph.GraphSnapshot) bool { return s.Scenario == scenario }, "scenario="+scenario)
	if err != nil {
		return graph.GraphSnapshotMeta{}, err
	}
	return graph.GraphSnapshotMeta{
		ID:                s.ID,
		Scenario:          s.Scenario,
		ContentHash:       s.ContentHash,
		SourceFingerprint: s.SourceFingerprint,
		ExtractedAt:       s.ExtractedAt,
		ExtractionMS:      s.ExtractionMS,
	}, nil
}

func (f *FakeRepository) FindByHash(_ context.Context, scenario, contentHash string) (graph.GraphSnapshot, error) {
	f.FindCalls.Add(1)
	if f.FindErr != nil {
		return graph.GraphSnapshot{}, f.FindErr
	}
	return f.findSnapshot(
		func(s graph.GraphSnapshot) bool { return s.Scenario == scenario && s.ContentHash == contentHash },
		scenario+":"+contentHash,
	)
}

func (f *FakeRepository) FindBySourceFingerprint(_ context.Context, scenario, sourceFingerprint string) (graph.GraphSnapshot, error) {
	f.SourceFindCalls.Add(1)
	if f.FindErr != nil {
		return graph.GraphSnapshot{}, f.FindErr
	}
	return f.findLatestSnapshot(
		func(s graph.GraphSnapshot) bool {
			return s.Scenario == scenario && s.SourceFingerprint == sourceFingerprint
		},
		scenario+":"+sourceFingerprint,
	)
}

func (f *FakeRepository) findSnapshot(match func(graph.GraphSnapshot) bool, notFoundID string) (graph.GraphSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.Snapshots {
		if match(s) {
			return s, nil
		}
	}
	return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{ID: notFoundID}
}

func (f *FakeRepository) findLatestSnapshot(match func(graph.GraphSnapshot) bool, notFoundID string) (graph.GraphSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.Snapshots) - 1; i >= 0; i-- {
		s := f.Snapshots[i]
		if match(s) {
			return s, nil
		}
	}
	return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{ID: notFoundID}
}

func (f *FakeRepository) ListSnapshots(_ context.Context, filter graph.ListSnapshotsFilter) (graph.SnapshotPage, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return graph.SnapshotPage{}, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []graph.GraphSnapshot
	for _, s := range f.Snapshots {
		if filter.Scenario != "" && s.Scenario != filter.Scenario {
			continue
		}
		out = append(out, s)
	}
	return graph.SnapshotPage{Snapshots: out}, nil
}

func (f *FakeRepository) ClearSnapshots(_ context.Context, scenario string) (int, error) {
	f.ClearCalls.Add(1)
	if f.ClearErr != nil {
		return 0, f.ClearErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	kept := f.Snapshots[:0]
	deleted := 0
	for _, s := range f.Snapshots {
		if s.Scenario == scenario {
			deleted++
			continue
		}
		kept = append(kept, s)
	}
	f.Snapshots = kept
	return deleted, nil
}

// PruneSnapshots enforces retention in memory, keeping the newest N per
// scenario. The ordering mirrors the SQLite implementation (extracted_at
// descending, then id descending) so a test written against the fake describes
// the same behaviour production has.
func (f *FakeRepository) PruneSnapshots(_ context.Context, policy graph.RetentionPolicy) (graph.RetentionResult, error) {
	f.PruneCalls.Add(1)
	if f.PruneErr != nil {
		return graph.RetentionResult{}, f.PruneErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	kept, removed, bytes := applyRetention(f.Snapshots, policy)
	scenarios := map[string]struct{}{}
	for _, s := range f.Snapshots {
		scenarios[s.Scenario] = struct{}{}
	}
	f.Snapshots = kept

	return graph.RetentionResult{
		ScenariosScanned: len(scenarios),
		RowsRemoved:      removed,
		BytesReclaimed:   bytes,
	}, nil
}

// ReclaimableSnapshotBytes reports what PruneSnapshots would remove.
func (f *FakeRepository) ReclaimableSnapshotBytes(_ context.Context, policy graph.RetentionPolicy) (int64, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, removed, bytes := applyRetention(f.Snapshots, policy)
	return bytes, removed, nil
}

// SnapshotPayloadBytes reports the total live payload across all snapshots.
func (f *FakeRepository) SnapshotPayloadBytes(context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var total int64
	for _, s := range f.Snapshots {
		total += approximatePayloadBytes(s)
	}
	return total, nil
}

// SnapshotCounts reports how many snapshots each scenario holds.
func (f *FakeRepository) SnapshotCounts(context.Context) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := map[string]int{}
	for _, s := range f.Snapshots {
		counts[s.Scenario]++
	}
	return counts, nil
}

// applyRetention returns the surviving snapshots, the number removed, and the
// approximate payload bytes those removals free.
func applyRetention(snapshots []graph.GraphSnapshot, policy graph.RetentionPolicy) ([]graph.GraphSnapshot, int, int64) {
	keep := policy.KeepPerScenario
	if keep < 1 {
		keep = graph.DefaultSnapshotRetentionKeep
	}

	byScenario := map[string][]graph.GraphSnapshot{}
	order := make([]string, 0)
	for _, s := range snapshots {
		if _, seen := byScenario[s.Scenario]; !seen {
			order = append(order, s.Scenario)
		}
		byScenario[s.Scenario] = append(byScenario[s.Scenario], s)
	}

	var (
		kept    []graph.GraphSnapshot
		removed int
		bytes   int64
	)
	for _, scenario := range order {
		group := byScenario[scenario]
		sort.SliceStable(group, func(i, j int) bool {
			if !group[i].ExtractedAt.Equal(group[j].ExtractedAt) {
				return group[i].ExtractedAt.After(group[j].ExtractedAt)
			}
			return group[i].ID > group[j].ID
		})
		for i, s := range group {
			if i < keep {
				kept = append(kept, s)
				continue
			}
			removed++
			bytes += approximatePayloadBytes(s)
		}
	}
	return kept, removed, bytes
}

// approximatePayloadBytes stands in for length(payload) in the fake.
func approximatePayloadBytes(s graph.GraphSnapshot) int64 {
	return int64(len(s.Files)+len(s.Symbols)+len(s.Imports)+len(s.Packages)) * 64
}

var _ graph.Repository = (*FakeRepository)(nil)

// FakeCodeGraphAdapter is the canned-graph adapter every test reaches
// for. The absence of go-code-graph / typescript-code-graph scenarios
// does not block cartographer development because this fake is the
// real seam every test consumes.
type FakeCodeGraphAdapter struct {
	NameValue      string
	LanguagesValue []graph.Language
	Raw            graph.RawGraph
	ExtractErr     error
	ExtractCalls   atomic.Int64
	LastScenario   string
}

func (a *FakeCodeGraphAdapter) Name() string {
	if a.NameValue == "" {
		return "fake"
	}
	return a.NameValue
}

func (a *FakeCodeGraphAdapter) SupportedLanguages() []graph.Language {
	return a.LanguagesValue
}

func (a *FakeCodeGraphAdapter) Extract(_ context.Context, scenario string) (graph.RawGraph, error) {
	a.ExtractCalls.Add(1)
	a.LastScenario = scenario
	if a.ExtractErr != nil {
		return graph.RawGraph{}, a.ExtractErr
	}
	return a.Raw, nil
}

var _ graph.CodeGraphAdapter = (*FakeCodeGraphAdapter)(nil)
