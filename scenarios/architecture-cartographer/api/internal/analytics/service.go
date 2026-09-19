package analytics

import (
	"context"
	"strings"
)

// Service is the application-layer surface for analytics. Owns
// validation (e.g., kind is recognized) and threshold suppression for
// stats reporting.
type Service interface {
	RecordEvent(ctx context.Context, e Event) (Event, error)
	RecordPlacement(ctx context.Context, p Placement) (Placement, error)
	RecordOverride(ctx context.Context, o Override) (Override, error)
	ListEvents(ctx context.Context, f EventFilter) (EventPage, error)
	ListPlacements(ctx context.Context, f PlacementFilter) (PlacementPage, error)
	Stats(ctx context.Context, scenario string) (StatsSummary, error)
}

type service struct {
	repo Repository
}

// NewService constructs the production Service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

var _ Service = (*service)(nil)

// validKinds is the set of EventKind values RecordEvent accepts. An
// allowlist (rather than parsing a string) ensures kinds always match
// the proto enum.
var validKinds = map[EventKind]struct{}{
	EventKindConflictDetected:      {},
	EventKindConflictAssigned:      {},
	EventKindConflictResolved:      {},
	EventKindConflictReopened:      {},
	EventKindConflictForceResolved: {},
	EventKindVerdictProduced:       {},
	EventKindPlacementAuto:         {},
	EventKindPlacementSuggest:      {},
	EventKindOverrideRecorded:      {},
	EventKindApplyPlanned:          {},
	EventKindApplyRan:              {},
	EventKindApplyBuildGreen:       {},
	EventKindApplyBuildRed:         {},
	EventKindApplyReverted:         {},
}

func (s *service) RecordEvent(ctx context.Context, e Event) (Event, error) {
	if strings.TrimSpace(e.Scenario) == "" {
		return Event{}, ErrInvalidEvent{Field: "scenario", Reason: "required"}
	}
	if _, ok := validKinds[e.Kind]; !ok {
		return Event{}, ErrInvalidEvent{Field: "kind", Reason: "unknown"}
	}
	return s.repo.AppendEvent(ctx, e)
}

func (s *service) RecordPlacement(ctx context.Context, p Placement) (Placement, error) {
	if strings.TrimSpace(p.Scenario) == "" {
		return Placement{}, ErrInvalidEvent{Field: "scenario", Reason: "required"}
	}
	if strings.TrimSpace(p.ChunkID) == "" {
		return Placement{}, ErrInvalidEvent{Field: "chunk_id", Reason: "required"}
	}
	return s.repo.AppendPlacement(ctx, p)
}

func (s *service) RecordOverride(ctx context.Context, o Override) (Override, error) {
	if strings.TrimSpace(o.Scenario) == "" {
		return Override{}, ErrInvalidOverride{Field: "scenario", Reason: "required"}
	}
	if strings.TrimSpace(o.ChunkID) == "" {
		return Override{}, ErrInvalidOverride{Field: "chunk_id", Reason: "required"}
	}
	if strings.TrimSpace(o.ChosenDomain) == "" {
		return Override{}, ErrInvalidOverride{Field: "chosen_domain", Reason: "required"}
	}
	if o.ChosenDomain == o.VerdictDomain {
		return Override{}, ErrInvalidOverride{Field: "chosen_domain", Reason: "must differ from verdict_domain"}
	}
	return s.repo.AppendOverride(ctx, o)
}

func (s *service) ListEvents(ctx context.Context, f EventFilter) (EventPage, error) {
	return s.repo.ListEvents(ctx, f)
}

func (s *service) ListPlacements(ctx context.Context, f PlacementFilter) (PlacementPage, error) {
	return s.repo.ListPlacements(ctx, f)
}

func (s *service) Stats(ctx context.Context, scenario string) (StatsSummary, error) {
	if strings.TrimSpace(scenario) == "" {
		return StatsSummary{}, ErrInvalidEvent{Field: "scenario", Reason: "required"}
	}
	return s.repo.Stats(ctx, scenario)
}
