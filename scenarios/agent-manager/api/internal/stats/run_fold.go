package stats

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
)

// RunSnapshot is an isolated, read-only fold of one run's typed operational
// events. It deliberately has no checkpoint or global engine state: report
// construction must never advance the process-wide aggregation watermark.
type RunSnapshot struct {
	GeneratedAt time.Time         `json:"generated_at"`
	EventCount  int64             `json:"event_count"`
	Fallback    FallbackInsights  `json:"fallback"`
	Health      HealthSummary     `json:"health"`
	Sandbox     SandboxSummary    `json:"sandbox"`
	Heartbeat   HeartbeatSummary  `json:"heartbeat"`
	Checkpoint  CheckpointSummary `json:"checkpoint"`
	Retry       RetrySummary      `json:"retry"`
}

// FoldRun folds one run through the same registered processors as Engine. It
// is intentionally a package function so callers cannot accidentally share a
// global Engine's mutable state or checkpoint.
func FoldRun(ctx context.Context, repo eventlog.Repository, runID uuid.UUID) (*RunSnapshot, error) {
	if repo == nil {
		return nil, fmt.Errorf("stats: event repository is unavailable")
	}

	state := newAggregateState()
	after := int64(-1)
	for {
		records, err := repo.SinceForRun(ctx, runID, after, refreshBatchSize)
		if err != nil {
			return nil, fmt.Errorf("stats: fold run %s: %w", runID, err)
		}
		if len(records) == 0 {
			break
		}
		for _, rec := range records {
			if processor := lookupProcessor(rec.EventType, rec.SchemaVersion); processor != nil {
				processor(state, rec)
				state.totalEvents++
				if !state.earliestRecorded || rec.Timestamp.Before(state.earliestEventAt) {
					state.earliestEventAt = rec.Timestamp
					state.earliestRecorded = true
				}
			}
			after = rec.Sequence
		}
		if len(records) < refreshBatchSize {
			break
		}
	}

	now := time.Now()
	history := state.historyWindow(now)
	return &RunSnapshot{
		GeneratedAt: now,
		EventCount:  state.totalEvents,
		Fallback:    state.buildFallbackInsights(now, history),
		Health:      state.buildHealthSummary(now, history),
		Sandbox:     state.buildSandboxSummary(now, history),
		Heartbeat:   state.buildHeartbeatSummary(now, history),
		Checkpoint:  state.buildCheckpointSummary(now, history),
		Retry:       state.buildRetrySummary(now, history),
	}, nil
}
