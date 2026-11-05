package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/fixtures"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

type purgeEventDeleteFailure struct {
	event.Store
	err error
}

func (s purgeEventDeleteFailure) Delete(ctx context.Context, id uuid.UUID) error {
	if s.err != nil {
		return s.err
	}
	return s.Store.Delete(ctx, id)
}

func TestPurgeDataReportsCommittedRowsAndCleansCascadedEvents(t *testing.T) {
	cases := []struct {
		name        string
		target      PurgeTarget
		failCleanup bool
	}{
		{name: "run target partial result", target: PurgeTargetRuns, failCleanup: true},
		{name: "task cascade partial result", target: PurgeTargetTasks, failCleanup: true},
		{name: "task cascade cleans sidecar", target: PurgeTargetTasks},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repos, eventStore, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			profile := fixtures.NewAgentProfile(t)
			if err := repos.Profiles.Create(ctx, profile); err != nil {
				t.Fatal(err)
			}
			taskStatus := domain.TaskStatusQueued
			if tc.target == PurgeTargetTasks {
				taskStatus = domain.TaskStatusCancelled
			}
			task := fixtures.NewTask(t, fixtures.WithTaskTitle("purge partial task"), fixtures.WithTaskStatus(taskStatus))
			if err := repos.Tasks.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			run := fixtures.NewRun(t, task.ID, profile.ID, fixtures.WithRunStatus(domain.RunStatusComplete))
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			if err := eventStore.Append(ctx, run.ID, fixtures.NewRunEvent(t, run.ID)); err != nil {
				t.Fatal(err)
			}

			cleanupErr := errors.New("event store unavailable")
			var deleteErr error
			if tc.failCleanup {
				deleteErr = cleanupErr
			}
			o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(purgeEventDeleteFailure{Store: eventStore, err: deleteErr}))
			pattern := "^purge partial"
			if tc.target == PurgeTargetRuns {
				run.Tag = "purge partial run"
				if err := repos.Runs.Update(ctx, run); err != nil {
					t.Fatal(err)
				}
				pattern = "^purge partial run$"
			}
			result, err := o.PurgeData(ctx, PurgeRequest{Pattern: pattern, Targets: []PurgeTarget{tc.target}})
			if tc.failCleanup {
				if !errors.Is(err, cleanupErr) || !strings.Contains(err.Error(), run.ID.String()) {
					t.Fatalf("PurgeData error=%v, want contextual event cleanup error for run %s", err, run.ID)
				}
			} else if err != nil {
				t.Fatalf("PurgeData successful sidecar cleanup: %v", err)
			}
			if result == nil {
				t.Fatal("PurgeData returned nil result after committed deletion")
			}
			wantTasks, wantRuns := 0, 1
			if tc.target == PurgeTargetTasks {
				wantTasks = 1
			}
			if got := result.Deleted; got.Tasks != wantTasks || got.Runs != wantRuns {
				t.Fatalf("partial deleted counts=%+v, want tasks=%d runs=%d", got, wantTasks, wantRuns)
			}
			if got, err := repos.Runs.Get(ctx, run.ID); err != nil || got != nil {
				t.Fatalf("run after committed purge = %+v, %v; want absent", got, err)
			}
			if tc.target == PurgeTargetTasks {
				if got, err := repos.Tasks.Get(ctx, task.ID); err != nil || got != nil {
					t.Fatalf("task after committed purge = %+v, %v; want absent", got, err)
				}
			}
			gotEvents, err := eventStore.Count(ctx, run.ID)
			wantEvents := int64(0)
			if tc.failCleanup {
				wantEvents = 1
			}
			if err != nil || gotEvents != wantEvents {
				t.Fatalf("event sidecar count=%d err=%v, want %d", gotEvents, err, wantEvents)
			}
		})
	}
}

func TestDeleteTaskReportsCommittedCascadeWhenEventCleanupFails(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	profile := fixtures.NewAgentProfile(t)
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	task := fixtures.NewTask(t, fixtures.WithTaskTitle("delete task with attached run"), fixtures.WithTaskStatus(domain.TaskStatusCancelled))
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := fixtures.NewRun(t, task.ID, profile.ID, fixtures.WithRunStatus(domain.RunStatusComplete))
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := eventStore.Append(ctx, run.ID, fixtures.NewRunEvent(t, run.ID)); err != nil {
		t.Fatal(err)
	}

	cleanupErr := errors.New("event store unavailable")
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(purgeEventDeleteFailure{Store: eventStore, err: cleanupErr}))
	err := o.DeleteTask(ctx, task.ID)
	if !errors.Is(err, cleanupErr) || !strings.Contains(err.Error(), task.ID.String()) || !strings.Contains(err.Error(), run.ID.String()) {
		t.Fatalf("DeleteTask error=%v, want contextual partial cleanup error", err)
	}
	if got, err := repos.Tasks.Get(ctx, task.ID); err != nil || got != nil {
		t.Fatalf("task after committed delete = %+v, %v; want absent", got, err)
	}
	if got, err := repos.Runs.Get(ctx, run.ID); err != nil || got != nil {
		t.Fatalf("run after committed cascade = %+v, %v; want absent", got, err)
	}
	if got, err := eventStore.Count(ctx, run.ID); err != nil || got != 1 {
		t.Fatalf("separate event sidecar should remain for failed cleanup retry, events=%d err=%v", got, err)
	}
}

type failNthRunDelete struct {
	repository.RunRepository
	calls    int
	failAt   int
	failedID uuid.UUID
	err      error
}

func (r *failNthRunDelete) Delete(ctx context.Context, id uuid.UUID, status domain.RunStatus, version int64) error {
	r.calls++
	if r.calls == r.failAt {
		r.failedID = id
		return r.err
	}
	return r.RunRepository.Delete(ctx, id, status, version)
}

func TestPurgeDataReturnsEarlierCommittedRunCountWhenLaterDeleteCASFails(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	profile := fixtures.NewAgentProfile(t)
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	task := fixtures.NewTask(t, fixtures.WithTaskTitle("keep task"))
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"purge-run-a", "purge-run-b"} {
		run := fixtures.NewRun(t, task.ID, profile.ID, fixtures.WithRunStatus(domain.RunStatusComplete))
		run.Tag = tag
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	casErr := errors.New("run changed during purge")
	wrappedRuns := &failNthRunDelete{RunRepository: repos.Runs, failAt: 2, err: casErr}
	o := New(repos.Profiles, repos.Tasks, wrappedRuns, WithEvents(eventStore))
	result, err := o.PurgeData(ctx, PurgeRequest{Pattern: "^purge-run-", Targets: []PurgeTarget{PurgeTargetRuns}})
	if !errors.Is(err, casErr) || !strings.Contains(err.Error(), wrappedRuns.failedID.String()) {
		t.Fatalf("PurgeData error=%v, want contextual second-run CAS failure", err)
	}
	if result == nil || result.Deleted.Runs != 1 {
		t.Fatalf("PurgeData partial result=%+v, want one earlier committed Run deletion", result)
	}
	if got, err := repos.Runs.Get(ctx, wrappedRuns.failedID); err != nil || got == nil {
		t.Fatalf("CAS-failed run=%+v err=%v; want retained", got, err)
	}
}
