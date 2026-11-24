package orchestration

import (
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestHeartbeatLoopInputSnapshotsExecutorOwnedState(t *testing.T) {
	runAt := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	checkpointAt := runAt.Add(time.Minute)
	run := &domain.Run{
		ID:            uuid.New(),
		Tag:           "starting-run",
		Status:        domain.RunStatusStarting,
		LastHeartbeat: &runAt,
	}
	checkpoint := &domain.RunCheckpoint{LastHeartbeat: checkpointAt}
	executor := &RunExecutor{run: run, checkpoint: checkpoint}

	input := executor.heartbeatLoopInput()
	if input.RunID != run.ID || input.Tag != "starting-run" {
		t.Fatalf("heartbeat identity = (%s, %q), want (%s, %q)", input.RunID, input.Tag, run.ID, "starting-run")
	}
	if !input.State.LastRunHeartbeat.Equal(runAt) || !input.State.LastCheckpointHeartbeat.Equal(checkpointAt) {
		t.Fatalf("heartbeat timestamps = (%s, %s), want (%s, %s)", input.State.LastRunHeartbeat, input.State.LastCheckpointHeartbeat, runAt, checkpointAt)
	}

	// The heartbeat goroutine receives a value snapshot. Later executor-side
	// startup persistence must not change the identity or timestamps it sees.
	run.ID = uuid.New()
	run.Tag = "updated-run"
	runAt = runAt.Add(time.Hour)
	run.LastHeartbeat = &runAt
	checkpoint.LastHeartbeat = checkpointAt.Add(time.Hour)
	if input.RunID == run.ID || input.Tag != "starting-run" {
		t.Fatalf("executor mutation changed heartbeat identity: (%s, %q)", input.RunID, input.Tag)
	}
	if !input.State.LastRunHeartbeat.Equal(time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)) || !input.State.LastCheckpointHeartbeat.Equal(checkpointAt) {
		t.Fatalf("executor mutation changed heartbeat timestamps: (%s, %s)", input.State.LastRunHeartbeat, input.State.LastCheckpointHeartbeat)
	}
}
