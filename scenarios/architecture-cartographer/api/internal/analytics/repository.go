package analytics

import (
	"context"
	"time"
)

// Repository is the persistence seam for the analytics domain.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests wire mocks.FakeRepository.
//
// Append-only: no Update / Delete methods.
type Repository interface {
	// AppendEvent persists e. The implementation populates ID and
	// RecordedAt when zero-valued. Returns the persisted event.
	AppendEvent(ctx context.Context, e Event) (Event, error)

	// ListEvents paginates events matching the filter.
	ListEvents(ctx context.Context, f EventFilter) (EventPage, error)

	// AppendPlacement persists p. ID/RecordedAt populated when zero.
	AppendPlacement(ctx context.Context, p Placement) (Placement, error)

	// ListPlacements paginates placement rows matching the filter.
	ListPlacements(ctx context.Context, f PlacementFilter) (PlacementPage, error)

	// AppendOverride persists o. ID/RecordedAt populated when zero.
	AppendOverride(ctx context.Context, o Override) (Override, error)

	// Stats returns aggregated counts + verdict success rate inputs
	// for the given scenario.
	Stats(ctx context.Context, scenario string) (StatsSummary, error)
}

// EventFilter scopes ListEvents.
type EventFilter struct {
	Scenario  string
	Kinds     []EventKind
	Since     time.Time
	PageSize  int
	PageToken string
}

// EventPage is the paginated result.
type EventPage struct {
	Events        []Event
	NextPageToken string
}

// PlacementFilter scopes ListPlacements.
type PlacementFilter struct {
	Scenario  string
	Outcomes  []string
	PageSize  int
	PageToken string
}

type PlacementPage struct {
	Placements    []Placement
	NextPageToken string
}
