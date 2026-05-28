// Package mocks holds in-memory fakes for the signals domain.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/boundaries"
)

// FakeSignal is a deterministic signal whose Score() returns canned
// values. Used by aggregator tests to drive tier dispatch + tie
// handling without standing up real day-one signals.
//
// Set Returns to emit scores; set Abstain to emit an abstention.
// If both are zero, FakeSignal returns an empty ScoreResult so tests
// can exercise the aggregator's broken-contract synthesis path.
type FakeSignal struct {
	NameValue      string
	Weight         float64
	Returns        []signals.Score
	Abstain        *signals.Abstention
	Available      bool
	UnavailableMsg string

	ScoreCalls atomic.Int64
}

func (f *FakeSignal) Name() string           { return f.NameValue }
func (f *FakeSignal) DefaultWeight() float64 { return f.Weight }
func (f *FakeSignal) Score(_ context.Context, _ signals.GraphContext, _ graph.Chunk) signals.ScoreResult {
	f.ScoreCalls.Add(1)
	out := signals.ScoreResult{}
	if len(f.Returns) > 0 {
		out.Scores = make([]signals.Score, len(f.Returns))
		copy(out.Scores, f.Returns)
	}
	if f.Abstain != nil {
		cp := *f.Abstain
		out.Abstention = &cp
	}
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

// FakeDomainMapProvider satisfies signals.DomainMapProvider.
type FakeDomainMapProvider struct {
	DomainMap domains.DerivedDomainMap
	Err       error
	Calls     atomic.Int64
}

func (f *FakeDomainMapProvider) GetDomainMap(_ context.Context, _ string) (domains.DerivedDomainMap, error) {
	f.Calls.Add(1)
	if f.Err != nil {
		return domains.DerivedDomainMap{}, f.Err
	}
	return f.DomainMap, nil
}

var _ signals.DomainMapProvider = (*FakeDomainMapProvider)(nil)

// FakeService satisfies signals.Service for handler tests.
type FakeService struct {
	Verdict       signals.Verdict
	BatchVerdicts []signals.Verdict
	Signals       []signals.SignalDescriptor
	Boundary      boundaries.Report
	NextErr       error
	ScoreCalls    atomic.Int64
	BatchCalls    atomic.Int64
	ExplainCalls  atomic.Int64
	ListCalls     atomic.Int64
	BoundaryCalls atomic.Int64
}

// ScoreBatch returns the canned batch verdict slice (or, when nil, a
// slice filled with the single Verdict value).
func (f *FakeService) ScoreBatch(_ context.Context, in signals.ScoreBatchInput) ([]signals.Verdict, error) {
	f.BatchCalls.Add(1)
	if f.NextErr != nil {
		return nil, f.NextErr
	}
	if f.BatchVerdicts != nil {
		return f.BatchVerdicts, nil
	}
	out := make([]signals.Verdict, len(in.Chunks))
	for i := range in.Chunks {
		out[i] = f.Verdict
	}
	return out, nil
}

// BoundaryHealth returns the canned report.
func (f *FakeService) BoundaryHealth(_ context.Context, _ string) (boundaries.Report, error) {
	f.BoundaryCalls.Add(1)
	if f.NextErr != nil {
		return boundaries.Report{}, f.NextErr
	}
	return f.Boundary, nil
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
