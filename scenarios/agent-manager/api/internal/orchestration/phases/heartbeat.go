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
	"sync"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
)

// HeartbeatLoopInput is the explicit input to RunHeartbeatLoop.
type HeartbeatLoopInput struct {
	Deps        Deps
	Run         *domain.Run
	Checkpoint  *domain.RunCheckpoint
	Mu          *sync.Mutex
	Levers      config.Levers
	Stop        <-chan struct{}
	Done        chan<- struct{}
	Checkpoints repository.CheckpointRepository
}

// RunHeartbeatLoop sends periodic heartbeats until Stop is closed or the
// caller's ctx is cancelled. Closes Done when it exits.
func RunHeartbeatLoop(ctx context.Context, in HeartbeatLoopInput) {
	defer close(in.Done)

	hbLog := obs.Component("heartbeat").With(
		obs.KeyRunID, in.Run.ID.String(),
		"tag", in.Run.GetTag(),
	)
	hbLog.Info("heartbeat loop starting", "interval", in.Levers.Heartbeat.RunHeartbeatInterval.String())

	SendHeartbeat(ctx, in)

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
			SendHeartbeat(ctx, in)
		}
	}
}

// SendHeartbeat performs one heartbeat update: writes Run.LastHeartbeat,
// persists to the run repo, and pings the checkpoint store.
//
// Heartbeat misses on either target emit a heartbeat.miss event so the
// stats engine can track miss rates per target without parsing log
// strings. The previous freeform "heartbeat update failed: ..." emissions
// are deleted; lastSuccess (Run.LastHeartbeat at the moment of the miss)
// is included in the typed payload for operator triage.
func SendHeartbeat(ctx context.Context, in HeartbeatLoopInput) {
	if in.Mu != nil {
		in.Mu.Lock()
	}
	now := time.Now()
	previousHeartbeat := in.Run.LastHeartbeat
	in.Run.LastHeartbeat = &now
	if in.Checkpoint != nil {
		in.Checkpoint.LastHeartbeat = now
	}
	runID := in.Run.ID
	tag := in.Run.GetTag()
	if in.Mu != nil {
		in.Mu.Unlock()
	}

	hbLog := obs.Component("heartbeat").With(
		obs.KeyRunID, runID.String(),
		"tag", tag,
	)

	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			hbLog.Error("heartbeat update failed", obs.KeyError, err.Error())
			EmitHeartbeatMiss(ctx, in.Deps, runID, eventlog.HeartbeatMissPayload{
				Target:        eventlog.HeartbeatTargetRun,
				AttemptNo:     1,
				LastSuccessAt: previousHeartbeat,
				Message:       err.Error(),
			})
		} else {
			hbLog.Debug("heartbeat updated", "at", now.Format(time.RFC3339))
		}
	}

	if in.Checkpoints != nil {
		if err := in.Checkpoints.Heartbeat(ctx, runID); err != nil {
			EmitHeartbeatMiss(ctx, in.Deps, runID, eventlog.HeartbeatMissPayload{
				Target:        eventlog.HeartbeatTargetCheckpoint,
				AttemptNo:     1,
				LastSuccessAt: previousHeartbeat,
				Message:       err.Error(),
			})
		}
	}
}
