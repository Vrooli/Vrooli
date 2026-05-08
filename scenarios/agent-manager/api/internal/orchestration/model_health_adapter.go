package orchestration

import (
	"context"
	"log/slog"

	"agent-manager/internal/fallback"
	"agent-manager/internal/health"
)

// healthMarkerAdapter bridges the executor's ModelHealthReporter seam
// to the persisted health.Store. Every Mark* call records an audit row
// triggered by the run that observed the outcome, so the snapshot view
// always reflects the latest classification.
//
// Adapter writes are best-effort: a failed insert is logged via slog
// but never propagated up — the run must not fail because the audit
// table is unavailable.
type healthMarkerAdapter struct {
	store *health.Store
	runID string
}

func newHealthMarkerAdapter(store *health.Store, runID string) *healthMarkerAdapter {
	return &healthMarkerAdapter{store: store, runID: runID}
}

// MarkModelHealthy records a StatusOK observation. Reason is intentionally
// empty for healthy outcomes — the column is reserved for failure
// classification.
func (a *healthMarkerAdapter) MarkModelHealthy(runnerType, modelID string) {
	if a == nil || a.store == nil {
		return
	}
	if err := a.store.RecordModel(context.Background(), runnerType, modelID, health.StatusOK, "", "", a.triggeredBy()); err != nil {
		slog.Warn("health: record ok failed", "err", err, "runner", runnerType, "model", modelID)
	}
}

// MarkModelUnavailable records a StatusFailed observation. The Reason
// column is populated with fallback.ReasonUnknown when no classified
// signal is available; the Message column carries the freeform error.
//
// The current MarkModel* seam predates the typed *fallback.ClassifiedError
// flow — Phase 2.6 wired classification through reportHealth but the
// reporter signature is still string-based. Phase 3 widens the seam to
// accept ClassifiedError directly so the Reason isn't downgraded here.
func (a *healthMarkerAdapter) MarkModelUnavailable(runnerType, modelID, message string) {
	if a == nil || a.store == nil {
		return
	}
	if err := a.store.RecordModel(context.Background(), runnerType, modelID, health.StatusFailed, string(fallback.ReasonUnknown), message, a.triggeredBy()); err != nil {
		slog.Warn("health: record failed failed", "err", err, "runner", runnerType, "model", modelID)
	}
}

func (a *healthMarkerAdapter) triggeredBy() string {
	if a == nil || a.runID == "" {
		return "runtime"
	}
	return a.runID
}
