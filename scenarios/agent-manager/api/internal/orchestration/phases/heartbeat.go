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
	"log"
	"sync"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"
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

	log.Printf("[heartbeat] Starting heartbeat loop for run %s (tag=%s, interval=%v)",
		in.Run.ID, in.Run.GetTag(), in.Levers.Heartbeat.RunHeartbeatInterval)

	SendHeartbeat(ctx, in)

	ticker := time.NewTicker(in.Levers.Heartbeat.RunHeartbeatInterval)
	defer ticker.Stop()

	heartbeatCount := 1
	for {
		select {
		case <-in.Stop:
			log.Printf("[heartbeat] Stopping heartbeat loop for run %s (sent %d heartbeats)",
				in.Run.ID, heartbeatCount)
			return
		case <-ctx.Done():
			log.Printf("[heartbeat] Context cancelled for run %s (sent %d heartbeats)",
				in.Run.ID, heartbeatCount)
			return
		case <-ticker.C:
			heartbeatCount++
			SendHeartbeat(ctx, in)
		}
	}
}

// SendHeartbeat performs one heartbeat update: writes Run.LastHeartbeat,
// persists to the run repo, and pings the checkpoint store.
func SendHeartbeat(ctx context.Context, in HeartbeatLoopInput) {
	if in.Mu != nil {
		in.Mu.Lock()
	}
	now := time.Now()
	in.Run.LastHeartbeat = &now
	if in.Checkpoint != nil {
		in.Checkpoint.LastHeartbeat = now
	}
	runID := in.Run.ID
	tag := in.Run.GetTag()
	if in.Mu != nil {
		in.Mu.Unlock()
	}

	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			log.Printf("[heartbeat] ERROR: Failed to update heartbeat for run %s (tag=%s): %v",
				runID, tag, err)
			EmitSystemEvent(ctx, in.Deps, runID, "warn", "heartbeat update failed: "+err.Error())
		} else {
			log.Printf("[heartbeat] DEBUG: Updated heartbeat for run %s (tag=%s) at %v",
				runID, tag, now.Format(time.RFC3339))
		}
	}

	if in.Checkpoints != nil {
		if err := in.Checkpoints.Heartbeat(ctx, runID); err != nil {
			EmitSystemEvent(ctx, in.Deps, runID, "warn", "heartbeat checkpoint failed: "+err.Error())
		}
	}
}
