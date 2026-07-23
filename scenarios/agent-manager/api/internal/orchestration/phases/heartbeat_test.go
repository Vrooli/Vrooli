package phases

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

func TestSendHeartbeatPersistsOnlyTheDurableHeartbeatColumn(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "heartbeat", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	checkpoint := &domain.RunCheckpoint{}
	SendHeartbeat(ctx, HeartbeatLoopInput{Deps: Deps{Runs: repos.Runs, Clock: func() time.Time { return now }}, Run: run, Checkpoint: checkpoint, Mu: &sync.Mutex{}})
	if run.LastHeartbeat == nil || !run.LastHeartbeat.Equal(now) || !checkpoint.LastHeartbeat.Equal(now) {
		t.Fatalf("in-memory heartbeat = run=%v checkpoint=%s", run.LastHeartbeat, checkpoint.LastHeartbeat)
	}
	persisted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || persisted.LastHeartbeat == nil || !persisted.LastHeartbeat.Equal(now) {
		t.Fatalf("persisted heartbeat = %+v, err=%v", persisted, err)
	}
}

func TestRunHeartbeatLoopStopsWithoutWaitingForNextTick(t *testing.T) {
	stop := make(chan struct{})
	done := make(chan struct{})
	close(stop)
	run := &domain.Run{ID: uuid.New(), Tag: "heartbeat-stop"}
	levers := config.DefaultLevers()
	levers.Heartbeat.RunHeartbeatInterval = time.Hour
	go RunHeartbeatLoop(context.Background(), HeartbeatLoopInput{Deps: Deps{Clock: time.Now}, Run: run, Levers: levers, Stop: stop, Done: done})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat loop did not stop")
	}
}
