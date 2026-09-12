package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"architecture-cartographer/internal/analytics"
)

// FakeService satisfies analytics.Service for handler tests.
type FakeService struct {
	mu sync.Mutex

	Events     []analytics.Event
	Placements []analytics.Placement
	Overrides  []analytics.Override
	Summary    analytics.StatsSummary

	RecordEventErr     error
	RecordPlacementErr error
	RecordOverrideErr  error
	ListEventsErr      error
	ListPlacementsErr  error
	StatsErr           error

	RecordEventCalls     atomic.Int64
	RecordPlacementCalls atomic.Int64
	RecordOverrideCalls  atomic.Int64
	ListEventsCalls      atomic.Int64
	ListPlacementsCalls  atomic.Int64
	StatsCalls           atomic.Int64
}

func (f *FakeService) RecordEvent(_ context.Context, e analytics.Event) (analytics.Event, error) {
	f.RecordEventCalls.Add(1)
	if f.RecordEventErr != nil {
		return analytics.Event{}, f.RecordEventErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Events = append(f.Events, e)
	return e, nil
}

func (f *FakeService) RecordPlacement(_ context.Context, p analytics.Placement) (analytics.Placement, error) {
	f.RecordPlacementCalls.Add(1)
	if f.RecordPlacementErr != nil {
		return analytics.Placement{}, f.RecordPlacementErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Placements = append(f.Placements, p)
	return p, nil
}

func (f *FakeService) RecordOverride(_ context.Context, o analytics.Override) (analytics.Override, error) {
	f.RecordOverrideCalls.Add(1)
	if f.RecordOverrideErr != nil {
		return analytics.Override{}, f.RecordOverrideErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Overrides = append(f.Overrides, o)
	return o, nil
}

func (f *FakeService) ListEvents(_ context.Context, _ analytics.EventFilter) (analytics.EventPage, error) {
	f.ListEventsCalls.Add(1)
	if f.ListEventsErr != nil {
		return analytics.EventPage{}, f.ListEventsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]analytics.Event, len(f.Events))
	copy(out, f.Events)
	return analytics.EventPage{Events: out}, nil
}

func (f *FakeService) ListPlacements(_ context.Context, _ analytics.PlacementFilter) (analytics.PlacementPage, error) {
	f.ListPlacementsCalls.Add(1)
	if f.ListPlacementsErr != nil {
		return analytics.PlacementPage{}, f.ListPlacementsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]analytics.Placement, len(f.Placements))
	copy(out, f.Placements)
	return analytics.PlacementPage{Placements: out}, nil
}

func (f *FakeService) Stats(_ context.Context, _ string) (analytics.StatsSummary, error) {
	f.StatsCalls.Add(1)
	if f.StatsErr != nil {
		return analytics.StatsSummary{}, f.StatsErr
	}
	return f.Summary, nil
}

var _ analytics.Service = (*FakeService)(nil)
