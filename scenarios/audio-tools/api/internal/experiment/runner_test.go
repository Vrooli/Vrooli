package experiment_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/experiment"
	"audio-tools/internal/testutil/mocks"
)

type memBlobs struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (m *memBlobs) Put(_ context.Context, key string, data []byte, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = map[string][]byte{}
	}
	m.data[key] = append([]byte(nil), data...)
	return nil
}

func (m *memBlobs) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.data[key]
	if !ok {
		return nil, errors.New("missing blob")
	}
	return append([]byte(nil), data...), nil
}

func (m *memBlobs) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func newManager(t *testing.T, runner experiment.Runner) (*experiment.Manager, experiment.Repository, *mocks.FakeClock) {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	repo := experiment.NewSQLiteRepository(newSchemaDB(t), clk)
	svc := experiment.NewService(repo, &memBlobs{})
	mgr := experiment.NewManager(experiment.Config{Service: svc, Runner: runner, Clock: clk})
	require.NoError(t, mgr.Start(context.Background()))
	t.Cleanup(mgr.Close)
	return mgr, repo, clk
}

func TestManager_SubmitWaitStoresReportAndRuns(t *testing.T) {
	mgr, repo, _ := newManager(t, func(_ context.Context, exp experiment.Experiment, emit func(int, string)) (experiment.RunResult, error) {
		emit(50, "halfway")
		return experiment.RunResult{
			Report: []byte(`{"ok":true}`),
			Runs: []experiment.Run{{
				Strategy:      "batch",
				ConditionJSON: []byte(`{"clean":true}`),
			}},
		}, nil
	})

	exp, err := mgr.Submit(context.Background(), experiment.SubmitSpec{Name: "default", RecipeJSON: []byte(`{"seed":1}`)})
	require.NoError(t, err)

	done, err := mgr.Wait(context.Background(), exp.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusSucceeded, done.Status)
	require.NotEmpty(t, done.ResultRef)

	stored, err := repo.GetExperiment(context.Background(), exp.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusSucceeded, stored.Status)
	require.NotNil(t, stored.StartedAt)
	require.NotNil(t, stored.FinishedAt)

	runs, err := repo.ListRuns(context.Background(), exp.ID)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, "batch", runs[0].Strategy)
}

func TestManager_WaitContextCancelDoesNotCancelRun(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	mgr, _, _ := newManager(t, func(ctx context.Context, _ experiment.Experiment, _ func(int, string)) (experiment.RunResult, error) {
		close(started)
		select {
		case <-release:
			return experiment.RunResult{Report: []byte(`{"done":true}`)}, nil
		case <-ctx.Done():
			return experiment.RunResult{}, ctx.Err()
		}
	})

	exp, err := mgr.Submit(context.Background(), experiment.SubmitSpec{Name: "survives wait cancel"})
	require.NoError(t, err)
	<-started

	waitCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = mgr.Wait(waitCtx, exp.ID)
	require.ErrorIs(t, err, context.Canceled)

	close(release)
	done, err := mgr.Wait(context.Background(), exp.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusSucceeded, done.Status)
}

func TestManager_CancelRunningExperiment(t *testing.T) {
	started := make(chan struct{})
	mgr, _, _ := newManager(t, func(ctx context.Context, _ experiment.Experiment, _ func(int, string)) (experiment.RunResult, error) {
		close(started)
		<-ctx.Done()
		return experiment.RunResult{}, ctx.Err()
	})

	exp, err := mgr.Submit(context.Background(), experiment.SubmitSpec{Name: "cancel"})
	require.NoError(t, err)
	<-started
	require.NoError(t, mgr.Cancel(exp.ID))

	done, err := mgr.Wait(context.Background(), exp.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusCanceled, done.Status)
}

func TestManager_StreamReportsQueuePosition(t *testing.T) {
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})
	mgr, _, _ := newManager(t, func(ctx context.Context, exp experiment.Experiment, _ func(int, string)) (experiment.RunResult, error) {
		switch exp.Name {
		case "first":
			close(firstStarted)
			select {
			case <-releaseFirst:
			case <-ctx.Done():
				return experiment.RunResult{}, ctx.Err()
			}
		case "second":
			close(secondStarted)
			select {
			case <-releaseSecond:
			case <-ctx.Done():
				return experiment.RunResult{}, ctx.Err()
			}
		}
		return experiment.RunResult{Report: []byte(`{"done":true}`)}, nil
	})

	first, err := mgr.Submit(context.Background(), experiment.SubmitSpec{Name: "first"})
	require.NoError(t, err)
	<-firstStarted
	_, err = mgr.Submit(context.Background(), experiment.SubmitSpec{Name: "second"})
	require.NoError(t, err)
	third, err := mgr.Submit(context.Background(), experiment.SubmitSpec{Name: "third"})
	require.NoError(t, err)

	ch, unsub, err := mgr.Subscribe(third.ID)
	require.NoError(t, err)
	defer unsub()

	initial := requireEvent(t, ch)
	require.Equal(t, experiment.StatusQueued, initial.Status)
	require.True(t, strings.Contains(initial.Message, "1 experiment ahead"), initial.Message)

	close(releaseFirst)
	<-secondStarted
	updated := requireEvent(t, ch)
	require.Equal(t, experiment.StatusQueued, updated.Status)
	require.Equal(t, "queued", updated.Message)

	close(releaseSecond)
	done, err := mgr.Wait(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusSucceeded, done.Status)
}

func TestManager_StartMarksOrphansFailed(t *testing.T) {
	ctx := context.Background()
	clk := mocks.NewFakeClock(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	repo := experiment.NewSQLiteRepository(newSchemaDB(t), clk)
	queued, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "queued"})
	require.NoError(t, err)
	running, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "running", Status: experiment.StatusRunning})
	require.NoError(t, err)

	svc := experiment.NewService(repo, &memBlobs{})
	mgr := experiment.NewManager(experiment.Config{
		Service: svc,
		Clock:   clk,
		Runner: func(context.Context, experiment.Experiment, func(int, string)) (experiment.RunResult, error) {
			return experiment.RunResult{}, nil
		},
	})
	require.NoError(t, mgr.Start(ctx))
	t.Cleanup(mgr.Close)

	gotQueued, err := repo.GetExperiment(ctx, queued.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusFailed, gotQueued.Status)
	require.Equal(t, "interrupted by server restart", gotQueued.Error)

	gotRunning, err := repo.GetExperiment(ctx, running.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusFailed, gotRunning.Status)
}

// failRunRepo wraps a real Repository but fails every CreateRun, simulating a
// metrics-row insert failure during finalize.
type failRunRepo struct {
	experiment.Repository
}

func (failRunRepo) CreateRun(context.Context, experiment.Run) (experiment.Run, error) {
	return experiment.Run{}, errors.New("disk full")
}

func (failRunRepo) CompleteSucceeded(context.Context, experiment.Experiment, []experiment.Run) error {
	return errors.New("disk full")
}

func TestManager_RunPersistFailureMarksFailed(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	repo := failRunRepo{experiment.NewSQLiteRepository(newSchemaDB(t), clk)}
	svc := experiment.NewService(repo, &memBlobs{})
	mgr := experiment.NewManager(experiment.Config{Service: svc, Clock: clk, Runner: func(context.Context, experiment.Experiment, func(int, string)) (experiment.RunResult, error) {
		return experiment.RunResult{
			Report: []byte(`{"ok":true}`),
			Runs:   []experiment.Run{{Strategy: "batch", ConditionJSON: []byte(`{"strategy":"batch"}`)}},
		}, nil
	}})
	require.NoError(t, mgr.Start(context.Background()))
	t.Cleanup(mgr.Close)

	exp, err := mgr.Submit(context.Background(), experiment.SubmitSpec{Name: "persist-fail"})
	require.NoError(t, err)

	done, err := mgr.Wait(context.Background(), exp.ID)
	require.NoError(t, err)
	// A run-row insert failure must NOT be reported as success; the report blob
	// without its backing metrics rows would be a silent partial result.
	require.Equal(t, experiment.StatusFailed, done.Status)
	require.Contains(t, done.Error, "persist successful result")
	require.Empty(t, done.ResultRef)

	stored, err := repo.GetExperiment(context.Background(), exp.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusFailed, stored.Status)
}

func TestService_DeleteExperimentDeletesTerminalRowsAndBlob(t *testing.T) {
	ctx := context.Background()
	clk := mocks.NewFakeClock(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	repo := experiment.NewSQLiteRepository(newSchemaDB(t), clk)
	blobs := &memBlobs{}
	svc := experiment.NewService(repo, blobs)
	exp, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "delete-me"})
	require.NoError(t, err)
	now := clk.Now().UTC()
	exp.Status = experiment.StatusSucceeded
	exp.FinishedAt = &now
	stored, err := svc.StoreReport(ctx, exp, []byte(`{"ok":true}`), "application/json", now)
	require.NoError(t, err)
	_, err = repo.CreateRun(ctx, experiment.Run{
		ExperimentID:  stored.ID,
		Strategy:      "batch",
		ConditionJSON: []byte(`{"strategy":"batch"}`),
	})
	require.NoError(t, err)

	deletedReport, err := svc.DeleteExperiment(ctx, stored)
	require.NoError(t, err)
	require.True(t, deletedReport)
	_, err = repo.GetExperiment(ctx, stored.ID)
	require.ErrorAs(t, err, &experiment.ErrNotFound{})
	runs, err := repo.ListRuns(ctx, stored.ID)
	require.NoError(t, err)
	require.Empty(t, runs)
	_, err = blobs.Get(ctx, stored.ResultRef)
	require.Error(t, err)
}

func requireEvent(t *testing.T, ch <-chan experiment.ProgressEvent) experiment.ProgressEvent {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for progress event")
		return experiment.ProgressEvent{}
	}
}
