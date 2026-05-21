// Package mocks holds in-memory fakes for the signals domain.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/signals"
)

// FakeSignal is a deterministic signal whose Score() returns canned
// values. Used by aggregator tests to drive tier dispatch + tie
// handling without standing up real day-one signals.
type FakeSignal struct {
	NameValue      string
	Weight         float64
	Returns        []signals.Score
	Available      bool
	UnavailableMsg string

	ScoreCalls atomic.Int64
}

func (f *FakeSignal) Name() string         { return f.NameValue }
func (f *FakeSignal) DefaultWeight() float64 { return f.Weight }
func (f *FakeSignal) Score(_ context.Context, _ signals.GraphContext, _ graph.Chunk) []signals.Score {
	f.ScoreCalls.Add(1)
	out := make([]signals.Score, len(f.Returns))
	copy(out, f.Returns)
	return out
}
func (f *FakeSignal) IsAvailable(_ context.Context) (bool, string) {
	return f.Available, f.UnavailableMsg
}

var _ signals.Signal = (*FakeSignal)(nil)

// FakeSnapshotProvider satisfies signals.SnapshotProvider.
type FakeSnapshotProvider struct {
	mu sync.Mutex

	BySnapshot map[string]graph.GraphSnapshot
	Err        error
	Calls      atomic.Int64
}

func (f *FakeSnapshotProvider) GetLatestSnapshot(_ context.Context, scenario string) (graph.GraphSnapshot, error) {
	f.Calls.Add(1)
	if f.Err != nil {
		return graph.GraphSnapshot{}, f.Err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.BySnapshot[scenario]
	if !ok {
		return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{ID: scenario}
	}
	return s, nil
}

var _ signals.SnapshotProvider = (*FakeSnapshotProvider)(nil)

// FakeManifestProvider satisfies signals.ManifestProvider.
type FakeManifestProvider struct {
	Manifest manifest.ManifestDefinition
	Err      error
	Calls    atomic.Int64
}

func (f *FakeManifestProvider) GetManifest(_ context.Context, _ string) (manifest.ManifestDefinition, error) {
	f.Calls.Add(1)
	if f.Err != nil {
		return manifest.ManifestDefinition{}, f.Err
	}
	return f.Manifest, nil
}

var _ signals.ManifestProvider = (*FakeManifestProvider)(nil)

// FakeService satisfies signals.Service for handler tests.
type FakeService struct {
	Verdict   signals.Verdict
	Signals   []signals.SignalDescriptor
	NextErr   error
	ScoreCalls   atomic.Int64
	ExplainCalls atomic.Int64
	ListCalls    atomic.Int64
}

func (f *FakeService) ScoreChunk(_ context.Context, _ signals.ScoreInput) (signals.Verdict, error) {
	f.ScoreCalls.Add(1)
	if f.NextErr != nil {
		return signals.Verdict{}, f.NextErr
	}
	return f.Verdict, nil
}

func (f *FakeService) ExplainVerdict(_ context.Context, _ signals.ScoreInput) (signals.Verdict, error) {
	f.ExplainCalls.Add(1)
	if f.NextErr != nil {
		return signals.Verdict{}, f.NextErr
	}
	return f.Verdict, nil
}

func (f *FakeService) ListSignals(_ context.Context, _ string) ([]signals.SignalDescriptor, error) {
	f.ListCalls.Add(1)
	if f.NextErr != nil {
		return nil, f.NextErr
	}
	return f.Signals, nil
}

var _ signals.Service = (*FakeService)(nil)
