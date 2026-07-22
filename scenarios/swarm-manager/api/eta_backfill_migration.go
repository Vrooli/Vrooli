package main

import (
	"log/slog"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eta"
	"swarm-manager/internal/eventlog"
)

// migrationNameBackfillETADurationSamplesV1 is the sentinel for the one-time
// backfill that walks every completed backlog item on disk and emits a coarse
// backlog.duration_sample event derived from its spec created→updated span.
// This seeds the ETA engine's cold-start distribution from history the event
// log never recorded (most completions predate the queue/execution
// instrumentation). Gated on a system.migration_applied event with this name.
const migrationNameBackfillETADurationSamplesV1 = "backfill_eta_duration_samples_v1"

// migrationNameSeedGoalsFromTagsV1 is the sentinel for the one-time seeding of
// the four de-facto v1 goals from their existing tags. Gated so an operator who
// deletes a seeded goal does not see it resurrected on the next boot.
const migrationNameSeedGoalsFromTagsV1 = "seed_goals_from_tags_v1"

// backfillETADurationSamples derives one backfill-origin duration sample per
// historical completed item that does not already have one, and emits it. It
// returns the number of samples emitted.
func (s *Server) backfillETADurationSamples(alreadySampled map[string]struct{}) int {
	store := backlog.NewFileStore(s.scenarioRoot)
	items, err := store.LoadAll(nil)
	if err != nil {
		slog.Error("backfill_eta_duration_samples_v1: load backlog failed", "err", err)
		return 0
	}

	var completed []eta.CompletedItem
	for _, it := range items {
		if it.Status != backlog.StatusCompleted {
			continue
		}
		ref := string(it.Kind) + "/" + it.Name
		completed = append(completed, eta.CompletedItem{
			Ref:        ref,
			Kind:       string(it.Kind),
			Effort:     it.Effort,
			Milestone: it.Milestone,
			Created:    it.Created,
			Completed:  it.Updated,
		})
	}

	samples, rep := eta.BuildBackfillSamples(completed, alreadySampled)
	for _, sample := range samples {
		s.emitter.EmitBacklogDurationSample(sample.Ref, eventlog.DurationSamplePayload{
			EffortClass:   sample.EffortClass,
			DurationHours: sample.DurationHours,
			Origin:        eventlog.DurationOriginBackfill,
			Kind:          sample.Kind,
			Milestone:    sample.Milestone,
		})
	}
	slog.Info("backfill_eta_duration_samples_v1 built",
		"produced", rep.Produced,
		"skipped_no_time", rep.SkippedNoTime,
		"skipped_already", rep.SkippedAlready)
	return rep.Produced
}

// refsWithDurationSamples returns the set of backlog refs that already carry a
// duration sample, so a partial prior run is never double-counted.
func refsWithDurationSamples(events []eventlog.Event) map[string]struct{} {
	seen := make(map[string]struct{})
	for _, e := range events {
		if e.EventType == eventlog.EventBacklogDurationSample {
			seen[e.EntityID] = struct{}{}
		}
	}
	return seen
}
