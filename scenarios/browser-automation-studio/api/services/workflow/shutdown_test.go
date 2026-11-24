package workflow

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	executor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

func TestShutdownClosesAdmissionCancelsAndJoinsOwnedRunners(t *testing.T) {
	service := &WorkflowService{}
	releaseAdmission, releaseSlot, err := service.beginExecutionAdmission()
	if err != nil {
		t.Fatal(err)
	}

	cancelObserved := make(chan struct{})
	runnerDone := make(chan struct{})
	executionID := uuid.New()
	service.storeExecutionCancel(executionID, func() { close(cancelObserved) }, runnerDone)

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- service.Shutdown(context.Background()) }()
	deadline := time.After(time.Second)
	for {
		newAdmission, newSlot, err := service.beginExecutionAdmission()
		if errors.Is(err, ErrWorkflowServiceShuttingDown) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		newAdmission()
		newSlot()
		select {
		case err := <-shutdownDone:
			t.Fatalf("Shutdown returned before the in-flight admission registered its runner: %v", err)
		case <-deadline:
			t.Fatal("Shutdown did not close execution admission")
		default:
		}
	}

	releaseAdmission()
	select {
	case <-cancelObserved:
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not cancel the owned runner")
	}
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before runner cleanup completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	releaseSlot()
	close(runnerDone)
	if err := <-shutdownDone; err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestExecutionCapacityIsSharedAcrossConcurrentStartPathsAndCancellation(t *testing.T) {
	for _, mode := range []string{"manual", "saved", "adhoc"} {
		t.Run(mode, func(t *testing.T) {
			started := make(chan struct{})
			var calls atomic.Int32
			service, repo, invoke := newCompletionFixture(t, mode, func(ctx context.Context, _ executor.Request) error {
				if calls.Add(1) == 1 {
					close(started)
					<-ctx.Done()
					return ctx.Err()
				}
				return nil
			})
			service.maxConcurrentExecutions = 1

			first := invoke(context.Background(), false)
			if first.err != nil {
				t.Fatalf("first execution admission: %v", first.err)
			}
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("first runner did not start")
			}
			firstID := repo.index.ID

			if rejected := invoke(context.Background(), false); !errors.Is(rejected.err, ErrWorkflowExecutionCapacity) {
				t.Fatalf("full-cap start error = %v, want capacity rejection", rejected.err)
			}
			if repo.index.ID != firstID {
				t.Fatal("capacity rejection persisted another execution")
			}

			if err := service.StopExecution(context.Background(), firstID); err != nil {
				t.Fatalf("cancel first execution: %v", err)
			}
			service.executionLifecycleMu.Lock()
			slotsInUse := service.executionSlotsInUse
			service.executionLifecycleMu.Unlock()
			if slotsInUse != 0 {
				t.Fatalf("cancelled runner retained execution permit: %d", slotsInUse)
			}

			next := invoke(context.Background(), true)
			if next.err != nil {
				t.Fatalf("start after cancellation released permit: %+v", next)
			}
			if mode == "manual" {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				terminal, err := service.WaitForExecution(ctx, repo.index.ID)
				if err != nil || terminal.Status != database.ExecutionStatusCompleted {
					t.Fatalf("manual runner after cancellation: terminal=%+v err=%v", terminal, err)
				}
			} else if next.status != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED {
				t.Fatalf("start after cancellation released permit: %+v", next)
			}
		})
	}
}

func TestExecutionCapacityPermitReleasedAfterRunnerFailure(t *testing.T) {
	var calls atomic.Int32
	service, _, invoke := newCompletionFixture(t, "saved", func(context.Context, executor.Request) error {
		if calls.Add(1) == 1 {
			return errors.New("fixture runner failure")
		}
		return nil
	})
	service.maxConcurrentExecutions = 1
	failed := invoke(context.Background(), true)
	if failed.err != nil || failed.status != basbase.ExecutionStatus_EXECUTION_STATUS_FAILED {
		t.Fatalf("first runner result: %+v", failed)
	}
	succeeded := invoke(context.Background(), true)
	if succeeded.err != nil || succeeded.status != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED {
		t.Fatalf("second runner after failure: %+v", succeeded)
	}
}

func TestExecutionCapacityConcurrentStartsNeverExceedLimit(t *testing.T) {
	service := &WorkflowService{maxConcurrentExecutions: 3}
	const attempts = 24
	start := make(chan struct{})
	var wg sync.WaitGroup
	var accepted atomic.Int32
	var releaseMu sync.Mutex
	var releases []func()
	var admissionReleases []func()
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			releaseAdmission, releaseSlot, err := service.beginExecutionAdmission()
			if err == nil {
				accepted.Add(1)
				releaseMu.Lock()
				releases = append(releases, releaseSlot)
				admissionReleases = append(admissionReleases, releaseAdmission)
				releaseMu.Unlock()
			} else if !errors.Is(err, ErrWorkflowExecutionCapacity) {
				t.Errorf("start admission: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := accepted.Load(); got != 3 {
		t.Fatalf("concurrent starts admitted = %d, want cap 3", got)
	}
	for _, release := range admissionReleases {
		release()
	}
	for _, release := range releases {
		release()
	}
	if releaseAdmission, releaseSlot, err := service.beginExecutionAdmission(); err != nil {
		t.Fatalf("capacity remained consumed after permits released: %v", err)
	} else {
		releaseAdmission()
		releaseSlot()
	}
}

func TestExecutionCapacityDefaultComesFromConfigOwner(t *testing.T) {
	service := &WorkflowService{}
	releaseAdmissions := make([]func(), 0, config.DefaultMaxConcurrentExecutions)
	releaseSlots := make([]func(), 0, config.DefaultMaxConcurrentExecutions)
	for i := 0; i < config.DefaultMaxConcurrentExecutions; i++ {
		releaseAdmission, releaseSlot, err := service.beginExecutionAdmission()
		if err != nil {
			t.Fatalf("admit default slot %d: %v", i+1, err)
		}
		releaseAdmissions = append(releaseAdmissions, releaseAdmission)
		releaseSlots = append(releaseSlots, releaseSlot)
	}
	if _, _, err := service.beginExecutionAdmission(); !errors.Is(err, ErrWorkflowExecutionCapacity) {
		t.Fatalf("admission beyond config default = %v, want capacity refusal at %d", err, config.DefaultMaxConcurrentExecutions)
	}
	for _, release := range releaseAdmissions {
		release()
	}
	for _, release := range releaseSlots {
		release()
	}
}

func TestShutdownDeadlineReportsUnjoinedRunner(t *testing.T) {
	service := &WorkflowService{}
	runnerDone := make(chan struct{})
	executionID := uuid.New()
	service.storeExecutionCancel(executionID, func() {}, runnerDone)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := service.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown error = %v, want deadline exceeded", err)
	}
	select {
	case <-runnerDone:
		t.Fatal("test runner unexpectedly completed")
	default:
	}
}

type workflowRestartRepository struct {
	database.Repository
	row *database.ExecutionIndex
}

func (r *workflowRestartRepository) ListExecutions(_ context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	if r.row == nil || r.row.Status != query.Status {
		return nil, 0, nil
	}
	copy := *r.row
	return []*database.ExecutionIndex{&copy}, 1, nil
}

func (r *workflowRestartRepository) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if r.row == nil || r.row.ID != id {
		return nil, nil
	}
	copy := *r.row
	return &copy, nil
}

func (r *workflowRestartRepository) UpdateExecutionStatus(_ context.Context, id uuid.UUID, status string, message *string, completed *time.Time, updated time.Time) error {
	if r.row == nil || r.row.ID != id {
		return nil
	}
	r.row.Status = status
	if message != nil {
		r.row.ErrorMessage = *message
	}
	r.row.CompletedAt = completed
	r.row.UpdatedAt = updated
	return nil
}

type workflowRestartDriver struct {
	sessions []driver.ObservedSession
	closed   []driver.ObservedSession
}

func (d *workflowRestartDriver) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	return append([]driver.ObservedSession(nil), d.sessions...), nil
}

func (d *workflowRestartDriver) ForceCloseSession(_ context.Context, id, ownerID, leaseID string) error {
	for i, session := range d.sessions {
		if session.ID != id || session.OwnerExecutionID != ownerID || session.LeaseID != leaseID {
			continue
		}
		d.closed = append(d.closed, session)
		d.sessions = append(d.sessions[:i], d.sessions[i+1:]...)
		return nil
	}
	return nil
}
