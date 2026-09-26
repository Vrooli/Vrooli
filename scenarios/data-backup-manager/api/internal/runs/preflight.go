package runs

import (
	"context"
	"time"

	"data-backup-manager/internal/preflight"

	"github.com/vrooli/api-core/schedule"
)

type preflightTargets struct{ lookup TargetLookup }

func (a preflightTargets) TargetForRun(ctx context.Context, id string) (preflight.Target, error) {
	t, err := a.lookup.TargetForRun(ctx, id)
	if err != nil {
		return preflight.Target{}, err
	}
	return preflight.Target{ID: t.ID, Kind: t.Kind, Locator: t.Locator}, nil
}

type preflightDestinations struct{ lookup DestinationLookup }

func (a preflightDestinations) DestinationForRun(ctx context.Context, id string) (preflight.Destination, error) {
	d, err := a.lookup.DestinationForRun(ctx, id)
	if err != nil {
		return preflight.Destination{}, err
	}
	return preflight.Destination{
		ID:          d.ID,
		Name:        d.Name,
		BackendKind: d.BackendKind,
		Location:    d.Location,
	}, nil
}

func (s *service) runPreflight(ctx context.Context, plan PlanForRun) preflight.Result {
	var clk schedule.Clock = s.deps.Clock
	result := preflight.Check(ctx, preflight.Input{
		Plan:             preflight.Plan{TargetIDs: plan.TargetIDs, DestinationIDs: plan.DestinationIDs},
		Targets:          preflightTargets{lookup: s.deps.Targets},
		Destinations:     preflightDestinations{lookup: s.deps.Destinations},
		Engine:           s.deps.Engine,
		Sources:          s.deps.Sources,
		Clock:            clk,
		CheckSourcePaths: s.deps.PreflightSourcePaths,
		Readiness:        s.deps.Readiness,
	})
	// Last-known-good is historical evidence, not a readiness probe. Attach
	// the oldest known successful target backup to each grouped incident so a
	// shared failure cannot overstate the protection window.
	if len(result.Incidents) > 0 {
		if statuses, err := s.deps.Repo.TargetStatuses(ctx, plan.TargetIDs); err == nil {
			lastGood := make(map[string]time.Time, len(statuses))
			for _, status := range statuses {
				lastGood[status.TargetID] = status.LastSuccessAt
			}
			for i := range result.Incidents {
				for _, targetID := range result.Incidents[i].TargetIDs {
					candidate := lastGood[targetID]
					if candidate.IsZero() {
						continue
					}
					if result.Incidents[i].LastKnownGood.IsZero() || candidate.Before(result.Incidents[i].LastKnownGood) {
						result.Incidents[i].LastKnownGood = candidate
					}
				}
			}
		}
	}
	return result
}
