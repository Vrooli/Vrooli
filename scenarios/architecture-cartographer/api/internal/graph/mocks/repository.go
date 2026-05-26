// Package mocks holds in-memory fakes for the graph domain.
package mocks

import (
	"context"
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

	SaveCalls  atomic.Int64
	GetCalls   atomic.Int64
	FindCalls  atomic.Int64
	ListCalls  atomic.Int64
	ClearCalls atomic.Int64
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
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.Snapshots {
		if s.ID == id {
			return s, nil
		}
	}
	return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{ID: id}
}

func (f *FakeRepository) FindByHash(_ context.Context, scenario, contentHash string) (graph.GraphSnapshot, error) {
	f.FindCalls.Add(1)
	if f.FindErr != nil {
		return graph.GraphSnapshot{}, f.FindErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.Snapshots {
		if s.Scenario == scenario && s.ContentHash == contentHash {
			return s, nil
		}
	}
	return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{ID: scenario + ":" + contentHash}
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
