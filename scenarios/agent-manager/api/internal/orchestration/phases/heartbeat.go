// Run heartbeat loop.
//
// The heartbeat goroutine fires once at start, then on Heartbeat.RunHeartbeatInterval.
// Its job is to update Run.LastHeartbeat (and the corresponding checkpoint
// row) so the reconciler can detect stalled runs. Heartbeat updates are
// best-effort: a failed write logs a warn event but does not interrupt
// execution.
//
// The heartbeat goroutine is started by RunExecutor and stopped via the
// stop channel; this file holds the pure loop body so the executor only
// owns the goroutine handle, not the cadence logic.

package phases

import (
	"context"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// HeartbeatState contains loop-owned success timestamps. Values are copied
// before starting the goroutine; the heartbeat never borrows a Run/Checkpoint.
type HeartbeatState struct {
	LastRunHeartbeat        time.Time
	LastCheckpointHeartbeat time.Time
}

// HeartbeatLoopInput is the explicit input to RunHeartbeatLoop.
type HeartbeatLoopInput struct {
	Deps        Deps
	RunID       uuid.UUID
	Tag         string
	State       HeartbeatState
	Levers      config.Levers
	Stop        <-chan struct{}
	Done        chan<- struct{}
	Checkpoints repository.CheckpointRepository
}

// RunHeartbeatLoop sends periodic heartbeats until Stop is closed or the
// caller's ctx is cancelled. Closes Done when it exits.
func RunHeartbeatLoop(ctx context.Context, in HeartbeatLoopInput) {
	defer close(in.Done)
	// A contained panic stops heartbeats (the reconciler will eventually reap
	// the run as stale) but must not take down the API with it.
	defer obs.RecoverToFailure("run heartbeat loop", nil)

	hbLog := obs.Component("heartbeat").With(
		obs.KeyRunID, in.RunID.String(),
		"tag", in.Tag,
	)
	hbLog.Info("heartbeat loop starting", "interval", in.Levers.Heartbeat.RunHeartbeatInterval.String())

	in.State = SendHeartbeat(ctx, in)

	ticker := time.NewTicker(in.Levers.Heartbeat.RunHeartbeatInterval)
	defer ticker.Stop()

	heartbeatCount := 1
	for {
		select {
		case <-in.Stop:
			hbLog.Info("heartbeat loop stopping", "sent", heartbeatCount)
			return
		case <-ctx.Done():
			hbLog.Info("heartbeat loop context cancelled", "sent", heartbeatCount)
			return
		case <-ticker.C:
			heartbeatCount++
			in.State = SendHeartbeat(ctx, in)
		}
	}
}

// SendHeartbeat persists one heartbeat through the repository owners and
// returns loop-local success timestamps. It never mutates executor-owned state.
//
// Heartbeat misses on either target emit a heartbeat.miss event so the
// stats engine can track miss rates per target without parsing log
// strings. Each target reports its own last successful write, not a failed or
// lifecycle-fenced attempt, for operator triage.
func SendHeartbeat(ctx context.Context, in HeartbeatLoopInput) HeartbeatState {
	now := in.Deps.Now()
	state := in.State
	runID := in.RunID

	hbLog := obs.Component("heartbeat").With(
		obs.KeyRunID, runID.String(),
		"tag", in.Tag,
	)

	if in.Deps.Runs != nil {
		// Status-guarded, single-column update: never resurrect a run that has
		// moved off running/starting (e.g. parked mid-turn by another goroutine,
		// or already terminal). A full-row Update from the stale in-memory run
		// would clobber a concurrent park/stop transition; TouchHeartbeat closes
		// that race at the SQL layer (updated=false ⇒ no row matched, not an error).
		updated, err := in.Deps.Runs.TouchHeartbeat(ctx, runID, now)
		switch {
		case err != nil:
			hbLog.Error("heartbeat update failed", obs.KeyError, err.Error())
			EmitHeartbeatMiss(ctx, in.Deps, runID, eventlog.HeartbeatMissPayload{
				Target:        eventlog.HeartbeatTargetRun,
				AttemptNo:     1,
				LastSuccessAt: heartbeatTime(state.LastRunHeartbeat),
				Message:       err.Error(),
			})
		case !updated:
			hbLog.Debug("heartbeat skipped; run no longer running/starting")
		default:
			state.LastRunHeartbeat = now
			hbLog.Debug("heartbeat updated", "at", now.Format(time.RFC3339))
		}
	}

	if in.Checkpoints != nil {
		if err := in.Checkpoints.Heartbeat(ctx, runID); err != nil {
			EmitHeartbeatMiss(ctx, in.Deps, runID, eventlog.HeartbeatMissPayload{
				Target:        eventlog.HeartbeatTargetCheckpoint,
				AttemptNo:     1,
				LastSuccessAt: heartbeatTime(state.LastCheckpointHeartbeat),
				Message:       err.Error(),
			})
		} else {
			state.LastCheckpointHeartbeat = now
		}
	}
	return state
}

func heartbeatTime(at time.Time) *time.Time {
	if at.IsZero() {
		return nil
	}
	return &at
}
