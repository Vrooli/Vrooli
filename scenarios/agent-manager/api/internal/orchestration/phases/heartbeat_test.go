package phases

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"

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
	checkpoint := &domain.RunCheckpoint{RunID: run.ID, Phase: domain.RunPhaseExecuting}
	if err := repos.Checkpoints.Save(ctx, checkpoint); err != nil {
		t.Fatal(err)
	}
	SendHeartbeat(ctx, HeartbeatLoopInput{Deps: Deps{Runs: repos.Runs, Clock: func() time.Time { return now }}, RunID: run.ID, Tag: run.GetTag(), Checkpoints: repos.Checkpoints})
	if run.LastHeartbeat != nil || !checkpoint.LastHeartbeat.IsZero() {
		t.Fatalf("heartbeat mutated executor-owned state: run=%v checkpoint=%s", run.LastHeartbeat, checkpoint.LastHeartbeat)
	}
	persisted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || persisted.LastHeartbeat == nil || !persisted.LastHeartbeat.Equal(now) {
		t.Fatalf("persisted heartbeat = %+v, err=%v", persisted, err)
	}
	persistedCheckpoint, err := repos.Checkpoints.Get(ctx, run.ID)
	if err != nil || persistedCheckpoint == nil || persistedCheckpoint.LastHeartbeat.IsZero() {
		t.Fatalf("checkpoint heartbeat = %+v, err=%v", persistedCheckpoint, err)
	}
	// Phase persistence still owns its old snapshot; it must not roll back a
	// newer heartbeat after the goroutine stops sharing that mutable pointer.
	checkpoint.Phase = domain.RunPhaseCollectingResults
	if err := repos.Checkpoints.Save(ctx, checkpoint); err != nil {
		t.Fatal(err)
	}
	afterSave, err := repos.Checkpoints.Get(ctx, run.ID)
	if err != nil || afterSave == nil || !afterSave.LastHeartbeat.Equal(persistedCheckpoint.LastHeartbeat) || afterSave.Phase != checkpoint.Phase {
		t.Fatalf("phase save rolled back heartbeat: before=%+v after=%+v err=%v", persistedCheckpoint, afterSave, err)
	}
	// RFC3339Nano omits fractional digits for exact seconds. Both newer
	// timestamps and stale snapshots must still compare chronologically.
	second := persistedCheckpoint.LastHeartbeat.Add(time.Second).Truncate(time.Second)
	for _, stamp := range []time.Time{second, second.Add(time.Nanosecond), second} {
		checkpoint.LastHeartbeat = stamp
		if err := repos.Checkpoints.Save(ctx, checkpoint); err != nil {
			t.Fatal(err)
		}
	}
	afterSave, err = repos.Checkpoints.Get(ctx, run.ID)
	if err != nil || afterSave == nil || !afterSave.LastHeartbeat.Equal(second.Add(time.Nanosecond)) {
		t.Fatalf("checkpoint heartbeat lost fractional successor: %+v err=%v", afterSave, err)
	}
}

func TestSendHeartbeatHonorsLatestLifecycle(t *testing.T) {
	for _, status := range []domain.RunStatus{domain.RunStatusStarting, domain.RunStatusRunning, domain.RunStatusPending, domain.RunStatusParked, domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			ctx := context.Background()
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			task := &domain.Task{ID: uuid.New(), Title: "heartbeat lifecycle", ScopePath: ".", Status: domain.TaskStatusQueued}
			if err := repos.Tasks.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			previous := time.Now().Add(-time.Minute).UTC()
			run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, LastHeartbeat: &previous}
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			now := previous.Add(time.Minute)
			in := HeartbeatLoopInput{Deps: Deps{Runs: repos.Runs, Clock: func() time.Time { return now }}, RunID: run.ID, State: HeartbeatState{LastRunHeartbeat: previous}}
			latest, err := repos.Runs.Get(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			latest.Status = status
			latest.ErrorMsg = "newer owner state"
			if err := repos.Runs.Update(ctx, latest); err != nil {
				t.Fatal(err)
			}
			state := SendHeartbeat(ctx, in)
			persisted, err := repos.Runs.Get(ctx, run.ID)
			if err != nil || persisted == nil || persisted.Status != status || persisted.LifecycleVersion != latest.LifecycleVersion || persisted.ErrorMsg != latest.ErrorMsg {
				t.Fatalf("heartbeat clobbered latest lifecycle: %+v, err=%v", persisted, err)
			}
			want := previous
			if status == domain.RunStatusStarting || status == domain.RunStatusRunning {
				want = now
			}
			if persisted.LastHeartbeat == nil || !persisted.LastHeartbeat.Equal(want) || !state.LastRunHeartbeat.Equal(want) {
				t.Fatalf("heartbeat = %v, state=%v; want %v", persisted.LastHeartbeat, state.LastRunHeartbeat, want)
			}
		})
	}
}

type heartbeatResultRunRepo struct {
	repository.RunRepository
	updated bool
	err     error
}

func (r *heartbeatResultRunRepo) TouchHeartbeat(context.Context, uuid.UUID, time.Time) (bool, error) {
	return r.updated, r.err
}

type heartbeatResultCheckpointRepo struct {
	repository.CheckpointRepository
	err error
}

func (r *heartbeatResultCheckpointRepo) Heartbeat(context.Context, uuid.UUID) error { return r.err }

func TestSendHeartbeatReportsOnlySuccessfulWritesPerTarget(t *testing.T) {
	deps, events := makeDeps()
	runs := &heartbeatResultRunRepo{updated: true}
	checkpoints := &heartbeatResultCheckpointRepo{}
	deps.Runs = runs
	now := time.Now().UTC()
	deps.Clock = func() time.Time { return now }
	in := HeartbeatLoopInput{Deps: deps, RunID: uuid.New(), Checkpoints: checkpoints}
	in.State = SendHeartbeat(context.Background(), in)
	first := now
	now = now.Add(time.Second)
	runs.err = errors.New("run heartbeat failed")
	in.State = SendHeartbeat(context.Background(), in)
	second := now
	now = now.Add(time.Second)
	runs.err, runs.updated = nil, false // latest lifecycle fences this attempt
	checkpoints.err = errors.New("checkpoint heartbeat failed")
	in.State = SendHeartbeat(context.Background(), in)
	if !in.State.LastRunHeartbeat.Equal(first) || !in.State.LastCheckpointHeartbeat.Equal(second) {
		t.Fatalf("unsuccessful attempt advanced success evidence: %+v", in.State)
	}
	misses := events.TypedEvents(in.RunID, domain.EventTypeHeartbeatMiss)
	if len(misses) != 2 {
		t.Fatalf("got %d misses, want run failure and checkpoint failure", len(misses))
	}
	for i, want := range []struct {
		target string
		at     time.Time
	}{{eventlog.HeartbeatTargetRun, first}, {eventlog.HeartbeatTargetCheckpoint, second}} {
		payload := decodeTypedPayload(t, misses[i]).(*eventlog.HeartbeatMissPayload)
		if payload.Target != want.target || payload.LastSuccessAt == nil || !payload.LastSuccessAt.Equal(want.at) {
			t.Fatalf("miss %d = %+v; want last successful %s heartbeat %v", i, payload, want.target, want.at)
		}
	}
}

// The executor serializes its run without the heartbeat mutex during startup.
// Heartbeats must persist independently, without writing that shared object.
func TestSendHeartbeatConcurrentRunPersistence(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "concurrent heartbeat", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusStarting, Phase: domain.RunPhaseInitializing}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	in := HeartbeatLoopInput{Deps: Deps{Runs: repos.Runs, Clock: time.Now}, RunID: run.ID, Tag: run.GetTag()}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 32; i++ {
			in.State = SendHeartbeat(ctx, in)
		}
	}()
	for i := 0; i < 32; i++ {
		run.ProgressPercent = i
		if err := repos.Runs.Update(ctx, run); err != nil {
			<-done
			t.Fatal(err)
		}
	}
	<-done
	if run.LastHeartbeat != nil {
		t.Fatal("heartbeat wrote the executor's run")
	}
	persisted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || persisted.LastHeartbeat == nil || persisted.ProgressPercent != 31 {
		t.Fatalf("concurrent persistence lost heartbeat or progress: %+v, err=%v", persisted, err)
	}
}

func TestRunHeartbeatLoopStopsWithoutWaitingForNextTick(t *testing.T) {
	stop := make(chan struct{})
	done := make(chan struct{})
	close(stop)
	run := &domain.Run{ID: uuid.New(), Tag: "heartbeat-stop"}
	levers := config.DefaultLevers()
	levers.Heartbeat.RunHeartbeatInterval = time.Hour
	go RunHeartbeatLoop(context.Background(), HeartbeatLoopInput{Deps: Deps{Clock: time.Now}, RunID: run.ID, Tag: run.GetTag(), Levers: levers, Stop: stop, Done: done})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat loop did not stop")
	}
}

func TestRunHeartbeatLoopTicksWithoutAgentOutput(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "timer heartbeat", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}

	levers := config.DefaultLevers()
	levers.Heartbeat.RunHeartbeatInterval = 10 * time.Millisecond
	stop := make(chan struct{})
	done := make(chan struct{})
	go RunHeartbeatLoop(ctx, HeartbeatLoopInput{Deps: Deps{Runs: repos.Runs, Clock: time.Now}, RunID: run.ID, Tag: run.GetTag(), Levers: levers, Stop: stop, Done: done})
	var first *time.Time
	deadline := time.After(time.Second)
	for {
		persisted, err := repos.Runs.Get(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if first == nil && persisted.LastHeartbeat != nil {
			observed := *persisted.LastHeartbeat
			first = &observed
		} else if first != nil && persisted.LastHeartbeat != nil && persisted.LastHeartbeat.After(*first) {
			break
		}
		select {
		case <-deadline:
			close(stop)
			<-done
			t.Fatal("timer did not emit a second heartbeat without agent output")
		case <-time.After(5 * time.Millisecond):
		}
	}
	close(stop)
	<-done
}

// panickingHeartbeatRepo simulates a defect inside the heartbeat persist path.
type panickingHeartbeatRepo struct{ repository.RunRepository }

func (panickingHeartbeatRepo) TouchHeartbeat(context.Context, uuid.UUID, time.Time) (bool, error) {
	panic("simulated heartbeat persist defect")
}

// A panic inside the heartbeat loop must be contained: Done still closes so
// the executor is not wedged, and the API stays up.
func TestRunHeartbeatLoopContainsPanic(t *testing.T) {
	stop := make(chan struct{})
	done := make(chan struct{})
	run := &domain.Run{ID: uuid.New(), Tag: "heartbeat-panic"}
	levers := config.DefaultLevers()
	levers.Heartbeat.RunHeartbeatInterval = time.Hour
	go RunHeartbeatLoop(context.Background(), HeartbeatLoopInput{
		Deps:   Deps{Runs: panickingHeartbeatRepo{}, Clock: time.Now},
		RunID:  run.ID,
		Tag:    run.GetTag(),
		Levers: levers,
		Stop:   stop,
		Done:   done,
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat loop did not exit after contained panic")
	}
}
