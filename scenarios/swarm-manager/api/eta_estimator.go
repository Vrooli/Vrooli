package main

import (
	"context"
	"encoding/json"

	"swarm-manager/internal/eta"
	"swarm-manager/internal/eventlog"
)

// newETAEstimator builds a fresh ETA estimator from the current
// backlog.duration_sample events and the live execute-lane capacity. It is the
// bridge between the event-sourced calibration samples (phase 1) and the pure
// eta engine (phase 4): the event log stays the single source of truth, and the
// estimator is rebuilt per read so newly-accrued live samples take effect
// immediately.
//
// Returns (nil, nil) when the event log is not wired, so callers omit the ETA
// band rather than failing the request.
func (s *Server) newETAEstimator() (*eta.Estimator, error) {
	if s.eventRepo == nil {
		return nil, nil
	}
	events, err := s.eventRepo.All(context.Background())
	if err != nil {
		return nil, err
	}
	samples := durationSamplesFromEvents(events)

	capacity := 1
	if s.executionSvc != nil {
		capacity = s.executionSvc.ExecuteLaneCapacity()
	}
	return eta.NewEstimator(samples, nil, capacity, eta.DefaultTrials, eta.DefaultSeed), nil
}

// durationSamplesFromEvents extracts the ETA calibration samples from the event
// log. Malformed or non-positive samples are skipped rather than erroring — a
// bad row abstains, it does not poison the distribution.
func durationSamplesFromEvents(events []eventlog.Event) []eta.Sample {
	var out []eta.Sample
	for _, ev := range events {
		if ev.EventType != eventlog.EventBacklogDurationSample || len(ev.Metadata) == 0 {
			continue
		}
		var p eventlog.DurationSamplePayload
		if err := json.Unmarshal(ev.Metadata, &p); err != nil {
			continue
		}
		if p.DurationHours <= 0 {
			continue
		}
		out = append(out, eta.Sample{
			EffortClass:   p.EffortClass,
			DurationHours: p.DurationHours,
			Origin:        p.Origin,
		})
	}
	return out
}
