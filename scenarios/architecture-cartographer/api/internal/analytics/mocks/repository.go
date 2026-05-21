// Package mocks holds in-memory fakes for the analytics domain.
//
// FakeRepository satisfies analytics.Repository for service tests.
// FakeService satisfies analytics.Service for handler tests.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"architecture-cartographer/internal/analytics"

	"github.com/google/uuid"
)

// FakeRepository is the in-memory analytics.Repository fake.
type FakeRepository struct {
	mu sync.Mutex

	Events     []analytics.Event
	Placements []analytics.Placement
	Overrides  []analytics.Override

	AppendEventErr     error
	ListEventsErr      error
	AppendPlacementErr error
	ListPlacementsErr  error
	AppendOverrideErr  error
	StatsErr           error

	AppendEventCalls     atomic.Int64
	ListEventsCalls      atomic.Int64
	AppendPlacementCalls atomic.Int64
	ListPlacementsCalls  atomic.Int64
	AppendOverrideCalls  atomic.Int64
	StatsCalls           atomic.Int64
}

func (f *FakeRepository) AppendEvent(_ context.Context, e analytics.Event) (analytics.Event, error) {
	f.AppendEventCalls.Add(1)
	if f.AppendEventErr != nil {
		return analytics.Event{}, f.AppendEventErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.RecordedAt.IsZero() {
		e.RecordedAt = time.Now().UTC()
	}
	f.Events = append(f.Events, e)
	return e, nil
}

func (f *FakeRepository) ListEvents(_ context.Context, filter analytics.EventFilter) (analytics.EventPage, error) {
	f.ListEventsCalls.Add(1)
	if f.ListEventsErr != nil {
		return analytics.EventPage{}, f.ListEventsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []analytics.Event
	for _, e := range f.Events {
		if filter.Scenario != "" && e.Scenario != filter.Scenario {
			continue
		}
		if !filter.Since.IsZero() && e.RecordedAt.Before(filter.Since) {
			continue
		}
		if len(filter.Kinds) > 0 {
			matched := false
			for _, k := range filter.Kinds {
				if k == e.Kind {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, e)
	}
	return analytics.EventPage{Events: out}, nil
}

func (f *FakeRepository) AppendPlacement(_ context.Context, p analytics.Placement) (analytics.Placement, error) {
	f.AppendPlacementCalls.Add(1)
	if f.AppendPlacementErr != nil {
		return analytics.Placement{}, f.AppendPlacementErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.RecordedAt.IsZero() {
		p.RecordedAt = time.Now().UTC()
	}
	f.Placements = append(f.Placements, p)
	return p, nil
}

func (f *FakeRepository) ListPlacements(_ context.Context, filter analytics.PlacementFilter) (analytics.PlacementPage, error) {
	f.ListPlacementsCalls.Add(1)
	if f.ListPlacementsErr != nil {
		return analytics.PlacementPage{}, f.ListPlacementsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []analytics.Placement
	for _, p := range f.Placements {
		if filter.Scenario != "" && p.Scenario != filter.Scenario {
			continue
		}
		if len(filter.Outcomes) > 0 {
			matched := false
			for _, o := range filter.Outcomes {
				if o == p.Outcome {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, p)
	}
	return analytics.PlacementPage{Placements: out}, nil
}

func (f *FakeRepository) AppendOverride(_ context.Context, o analytics.Override) (analytics.Override, error) {
	f.AppendOverrideCalls.Add(1)
	if f.AppendOverrideErr != nil {
		return analytics.Override{}, f.AppendOverrideErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	if o.RecordedAt.IsZero() {
		o.RecordedAt = time.Now().UTC()
	}
	f.Overrides = append(f.Overrides, o)
	return o, nil
}

func (f *FakeRepository) Stats(_ context.Context, scenario string) (analytics.StatsSummary, error) {
	f.StatsCalls.Add(1)
	if f.StatsErr != nil {
		return analytics.StatsSummary{}, f.StatsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	ss := analytics.StatsSummary{Scenario: scenario}
	for _, e := range f.Events {
		if e.Scenario != scenario {
			continue
		}
		switch e.Kind {
		case analytics.EventKindConflictDetected:
			ss.ConflictsDetected++
		case analytics.EventKindConflictResolved:
			ss.ConflictsResolved++
		case analytics.EventKindConflictForceResolved:
			ss.ConflictsForceResolved++
		case analytics.EventKindVerdictProduced:
			ss.VerdictObservationCount++
		}
	}
	for _, p := range f.Placements {
		if p.Scenario != scenario {
			continue
		}
		switch p.Outcome {
		case "auto_placed":
			ss.PlacementsAuto++
		case "suggested":
			ss.PlacementsSuggest++
		}
	}
	for _, o := range f.Overrides {
		if o.Scenario == scenario {
			ss.Overrides++
		}
	}
	if ss.VerdictObservationCount < analytics.MinVerdictObservations {
		ss.VerdictSuccessRateSuppressed = true
	} else {
		ss.VerdictSuccessRate = 1.0 - float64(ss.Overrides)/float64(ss.VerdictObservationCount)
		if ss.VerdictSuccessRate < 0 {
			ss.VerdictSuccessRate = 0
		}
	}
	return ss, nil
}

var _ analytics.Repository = (*FakeRepository)(nil)
