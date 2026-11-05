package orchestration_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

func TestStopRunPendingCancelsAdmissionWithoutCallingRunnerStop(t *testing.T) {
	for _, claimed := range []bool{false, true} {
		name := "unclaimed"
		if claimed {
			name = "claimed-before-runner-status"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			repos, events, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
			t.Cleanup(dispatcher.Close)

			registry := runner.NewRegistry()
			mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
			mockRunner.SetAvailable(true, "available")
			var stopCalls atomic.Int32
			mockRunner.StopFunc = func(context.Context, uuid.UUID) error {
				stopCalls.Add(1)
				return nil
			}
			mustRegisterRunner(t, registry, mockRunner)
			terminator := orchestration.NewTerminator(repos.Runs, registry, orchestration.TerminatorConfig{
				GracePeriod: time.Millisecond, MaxRetries: 1, BaseBackoff: time.Millisecond,
				MaxBackoff: time.Millisecond, VerifyTimeout: time.Millisecond, KillProcessGroup: false,
			})
			fakeSandbox := &mocks.FakeSandboxProvider{}
			svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
				orchestration.WithEvents(events), orchestration.WithRunners(registry),
				orchestration.WithTerminator(terminator), orchestration.WithSpawnDispatcher(dispatcher),
				orchestration.WithSandbox(fakeSandbox))
			run := pendingStopRun()
			sharedSandboxID := uuid.New()
			run.SandboxID = &sharedSandboxID
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			admission, err := dispatcher.ReserveAdmission(run.ID)
			if err != nil {
				t.Fatal(err)
			}
			var entered, finished chan struct{}
			if claimed {
				entered, finished = make(chan struct{}), make(chan struct{})
				if err := dispatcher.EnqueueAdmitted(&spawn.Job{RunID: run.ID, Fn: func(spawn.StartedFn) {
					close(entered)
					<-admission.Context().Done()
					close(finished)
				}}, admission); err != nil {
					t.Fatal(err)
				}
				<-entered // The dispatcher has claimed this Admission; Run is still Pending.
			}

			if err := svc.StopRun(ctx, run.ID); err != nil {
				t.Fatalf("StopRun pending: %v", err)
			}
			if claimed {
				select {
				case <-finished:
				case <-time.After(time.Second):
					t.Fatal("claimed executor did not observe cooperative cancellation")
				}
			}
			got, err := repos.Runs.Get(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != domain.RunStatusCancelled {
				t.Fatalf("status=%s, want cancelled", got.Status)
			}
			if stopCalls.Load() != 0 {
				t.Fatalf("runner stop calls=%d, pending run must not invoke process stop", stopCalls.Load())
			}
			if fakeSandbox.StopCallCount() != 0 || fakeSandbox.DeleteCallCount() != 0 {
				t.Fatalf("pending cancellation touched shared sandbox: stop=%d delete=%d", fakeSandbox.StopCallCount(), fakeSandbox.DeleteCallCount())
			}
		})
	}
}

func TestStopRunPendingCancellationPersistenceFailureReopensAdmission(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
	t.Cleanup(dispatcher.Close)
	persistErr := errors.New("injected cancellation write failure")
	wrapped := &requestCancellationFailureRepo{RunRepository: repos.Runs, err: persistErr}
	svc := orchestration.New(repos.Profiles, repos.Tasks, wrapped,
		orchestration.WithEvents(events), orchestration.WithSpawnDispatcher(dispatcher))
	run := pendingStopRun()
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	admission, err := dispatcher.ReserveAdmission(run.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.StopRun(ctx, run.ID); !errors.Is(err, persistErr) {
		t.Fatalf("StopRun error=%v, want injected persistence error", err)
	}
	if admission.Cancelled() {
		t.Fatal("admission remained cancelled after durable cancellation failed")
	}
	select {
	case <-admission.Context().Done():
		t.Fatal("executor context was cancelled despite durable cancellation failure")
	default:
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusPending || got.CancelRequestedAt != nil {
		t.Fatalf("failed cancellation mutated Run: status=%s cancelRequestedAt=%v", got.Status, got.CancelRequestedAt)
	}
}

func TestStopRunPendingRereadsAndUsesNormalPathWhenRunStarts(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
	t.Cleanup(dispatcher.Close)
	wrapped := &startingOnCancellationRepo{RunRepository: repos.Runs}
	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	var stopCalls atomic.Int32
	mockRunner.StopFunc = func(context.Context, uuid.UUID) error {
		stopCalls.Add(1)
		return nil
	}
	mustRegisterRunner(t, registry, mockRunner)
	svc := orchestration.New(repos.Profiles, repos.Tasks, wrapped,
		orchestration.WithEvents(events), orchestration.WithRunners(registry), orchestration.WithSpawnDispatcher(dispatcher))
	run := pendingStopRun()
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := dispatcher.ReserveAdmission(run.ID); err != nil {
		t.Fatal(err)
	}

	if err := svc.StopRun(ctx, run.ID); err != nil {
		t.Fatalf("StopRun after pending-to-starting race: %v", err)
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RunStatusCancelled || stopCalls.Load() != 1 {
		t.Fatalf("status=%s runnerStopCalls=%d, want cancelled through normal active stop", got.Status, stopCalls.Load())
	}
}

type requestCancellationFailureRepo struct {
	repository.RunRepository
	err error
}

func (r *requestCancellationFailureRepo) RequestCancellation(context.Context, uuid.UUID, time.Time) (bool, error) {
	return false, r.err
}

type startingOnCancellationRepo struct {
	repository.RunRepository
}

func (r *startingOnCancellationRepo) RequestCancellation(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	run, err := r.RunRepository.Get(ctx, id)
	if err != nil {
		return false, err
	}
	run.Status = domain.RunStatusStarting
	run.Phase = domain.RunPhaseInitializing
	if err := r.RunRepository.Update(ctx, run); err != nil {
		return false, err
	}
	return r.RunRepository.RequestCancellation(ctx, id, at)
}

func pendingStopRun() *domain.Run {
	now := time.Now().UTC()
	return &domain.Run{
		ID: uuid.New(), Tag: uuid.NewString(), RunMode: domain.RunModeSandboxed, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
		Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		CreatedAt:      now, UpdatedAt: now,
	}
}
