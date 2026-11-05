package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/testutil/executormocks"
	"github.com/vrooli/browser-automation-studio/internal/testutil/schedulemocks"
)

func newMockScheduleRepo(schedules ...*database.ScheduleIndex) *schedulemocks.Repository {
	return schedulemocks.NewRepository(schedules...)
}

type cancellationAwareExecutor struct {
	started   chan struct{}
	canceled  chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

func (f *cancellationAwareExecutor) ExecuteWorkflow(ctx context.Context, _ uuid.UUID, _ map[string]any) (*database.ExecutionIndex, error) {
	f.startOnce.Do(func() { close(f.started) })
	<-ctx.Done()
	f.stopOnce.Do(func() { close(f.canceled) })
	return nil, ctx.Err()
}

type fakeNotifier struct {
	mu     sync.Mutex
	events []ScheduleEvent
}

func (n *fakeNotifier) BroadcastScheduleEvent(event ScheduleEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, event)
}

func (n *fakeNotifier) countByType(eventType string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	count := 0
	for _, evt := range n.events {
		if evt.Type == eventType {
			count++
		}
	}
	return count
}

func testLogger() *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	return log
}

// newTestScheduler creates a Scheduler for tests with the given dependencies.
// CreditService and SettingsRepo are left nil for basic tests.
func newTestScheduler(repo ScheduleRepository, executor WorkflowExecutor, notifier ScheduleNotifier) *Scheduler {
	return New(SchedulerOptions{
		Repo:     repo,
		Executor: executor,
		Notifier: notifier,
		Log:      testLogger(),
		// CreditService and SettingsRepo left nil for tests
	})
}

func newSchedule(id, workflowID uuid.UUID, cronExpr string, active bool) *database.ScheduleIndex {
	s := &database.ScheduleIndex{
		ID:             id,
		WorkflowID:     workflowID,
		Name:           "Test Schedule",
		CronExpression: cronExpr,
		Timezone:       "UTC",
		IsActive:       active,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	_ = s.SetParameters(map[string]any{"foo": "bar"})
	return s
}

func TestSchedulerStartWithNoSchedules(t *testing.T) {
	repo := newMockScheduleRepo()
	executor := &executormocks.Executor{}
	notifier := &fakeNotifier{}

	s := newTestScheduler(repo, executor, notifier)
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !s.IsRunning() {
		t.Fatal("scheduler should report running after Start")
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop scheduler: %v", err)
		}
		if s.IsRunning() {
			t.Error("scheduler should report stopped after Stop")
		}
	})

	if got := s.RegisteredCount(); got != 0 {
		t.Fatalf("expected 0 registered schedules, got %d", got)
	}
}

func TestSchedulerStopContextCancelsScheduledWorkBeforeCronDrain(t *testing.T) {
	executor := &cancellationAwareExecutor{started: make(chan struct{}), canceled: make(chan struct{})}
	s := newTestScheduler(newMockScheduleRepo(), executor, nil)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	go s.createJob(newSchedule(uuid.New(), uuid.New(), "*/5 * * * * *", true))()
	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("scheduled workflow did not start")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.StopContext(stopCtx); err != nil {
		t.Fatalf("StopContext returned error: %v", err)
	}
	select {
	case <-executor.canceled:
	case <-time.After(time.Second):
		t.Fatal("scheduler context did not cancel the running workflow")
	}
}

func TestSchedulerCanRestartAfterContextBoundedStop(t *testing.T) {
	s := newTestScheduler(newMockScheduleRepo(), &executormocks.Executor{}, nil)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.StopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("scheduler could not restart after stop: %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerStartWithActiveSchedules(t *testing.T) {
	workflowID := uuid.New()
	s1 := newSchedule(uuid.New(), workflowID, "*/5 * * * * *", true)
	s2 := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", true)
	s3 := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", false)

	repo := newMockScheduleRepo(s1, s2, s3)
	executor := &executormocks.Executor{}
	notifier := &fakeNotifier{}

	s := newTestScheduler(repo, executor, notifier)
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop scheduler: %v", err)
		}
	})

	if got := s.RegisteredCount(); got != 2 {
		t.Fatalf("expected 2 registered schedules, got %d", got)
	}
	if repo.NextRunCalls.Load() != 2 {
		t.Fatalf("expected nextRunCalls=2, got %d", repo.NextRunCalls.Load())
	}
}

func TestSchedulerCannotStartTwice(t *testing.T) {
	repo := newMockScheduleRepo()
	executor := &executormocks.Executor{}
	s := newTestScheduler(repo, executor, nil)

	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop scheduler: %v", err)
		}
	})

	if err := s.Start(); err == nil {
		t.Fatalf("expected error on second start")
	}
}

func TestSchedulerRegisterSchedule(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", true)

	repo := newMockScheduleRepo()
	repo.Add(schedule)

	s := newTestScheduler(repo, &executormocks.Executor{}, nil)
	if err := s.RegisterSchedule(schedule); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := s.RegisteredCount(); got != 1 {
		t.Fatalf("expected 1 registered schedule, got %d", got)
	}
	if _, registered := s.GetScheduleInfo(schedule.ID); !registered {
		t.Fatal("registered schedule should have entry information")
	}
}

func TestSchedulerRegisterInactiveSchedule(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", false)

	repo := newMockScheduleRepo()
	repo.Add(schedule)

	s := newTestScheduler(repo, &executormocks.Executor{}, nil)
	if err := s.RegisterSchedule(schedule); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := s.RegisteredCount(); got != 0 {
		t.Fatalf("expected 0 registered schedules, got %d", got)
	}
	if _, registered := s.GetScheduleInfo(schedule.ID); registered {
		t.Fatal("unregistered schedule should not have entry information")
	}
}

func TestSchedulerUnregisterSchedule(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", true)

	repo := newMockScheduleRepo(schedule)
	s := newTestScheduler(repo, &executormocks.Executor{}, nil)
	if err := s.RegisterSchedule(schedule); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := s.UnregisterSchedule(schedule.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := s.RegisteredCount(); got != 0 {
		t.Fatalf("expected 0 registered schedules, got %d", got)
	}
}

func TestSchedulerDeactivateSchedule(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", true)

	repo := newMockScheduleRepo(schedule)
	s := newTestScheduler(repo, &executormocks.Executor{}, nil)
	if err := s.RegisterSchedule(schedule); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := s.RegisteredCount(); got != 1 {
		t.Fatalf("expected 1 registered schedule, got %d", got)
	}

	schedule.IsActive = false
	if err := s.RegisterSchedule(schedule); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := s.RegisteredCount(); got != 0 {
		t.Fatalf("expected 0 registered schedules after deactivation, got %d", got)
	}
}

func TestSchedulerCronExecution(t *testing.T) {
	workflowID := uuid.New()
	executionID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/1 * * * * *", true)

	repo := newMockScheduleRepo(schedule)
	called := make(chan struct{})
	var calledOnce sync.Once
	execution := &database.ExecutionIndex{
		ID:         executionID,
		WorkflowID: workflowID,
		Status:     database.ExecutionStatusCompleted,
		StartedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	executor := &executormocks.Executor{
		RunFunc: func(_ context.Context, _ uuid.UUID, _ map[string]any) (*database.ExecutionIndex, error) {
			calledOnce.Do(func() { close(called) })
			return execution, nil
		},
	}
	notifier := &fakeNotifier{}

	s := newTestScheduler(repo, executor, notifier)
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop scheduler: %v", err)
		}
	})

	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for scheduled execution")
	}

	if len(executor.Calls()) == 0 {
		t.Fatalf("expected scheduled execution to run")
	}
	if repo.LastRunCalls.Load() == 0 {
		t.Fatalf("expected last_run_at to be updated")
	}
	if notifier.countByType(EventTypeScheduleStarted) == 0 {
		t.Fatalf("expected schedule_started notification")
	}
	if notifier.countByType(EventTypeScheduleCompleted) == 0 {
		t.Fatalf("expected schedule_completed notification")
	}
}

func TestSchedulerCronExecutionFailureNotifies(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/1 * * * * *", true)

	repo := newMockScheduleRepo(schedule)
	called := make(chan struct{})
	var calledOnce sync.Once
	executor := &executormocks.Executor{RunFunc: func(context.Context, uuid.UUID, map[string]any) (*database.ExecutionIndex, error) {
		calledOnce.Do(func() { close(called) })
		return nil, errors.New("boom")
	}}
	notifier := &fakeNotifier{}

	s := newTestScheduler(repo, executor, notifier)
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop scheduler: %v", err)
		}
	})

	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for scheduled execution")
	}

	if len(executor.Calls()) == 0 {
		t.Fatalf("expected scheduled execution to run")
	}
	if notifier.countByType(EventTypeScheduleFailed) == 0 {
		t.Fatalf("expected schedule_failed notification")
	}
}

func TestSchedulerGracefulShutdown(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/5 * * * * *", true)
	repo := newMockScheduleRepo(schedule)

	s := newTestScheduler(repo, &executormocks.Executor{}, nil)
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("expected no error stopping scheduler, got %v", err)
	}
}

func TestSchedulerConcurrentRegistration(t *testing.T) {
	repo := newMockScheduleRepo()
	s := newTestScheduler(repo, &executormocks.Executor{}, nil)

	const n = 25
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			schedule := newSchedule(uuid.New(), uuid.New(), "*/10 * * * * *", true)
			repo.Add(schedule)
			_ = s.RegisterSchedule(schedule)
		}()
	}
	wg.Wait()

	if got := s.RegisteredCount(); got != n {
		t.Fatalf("expected %d registered schedules, got %d", n, got)
	}
}

func TestCronParserValidation(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{"valid 5-field", "*/5 * * * *", false},
		{"valid 6-field", "0 */5 * * * *", false},
		{"invalid fields", "*/5 * *", true},
		{"invalid chars", "x y z a b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronExpression(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateCronExpression(%q) error=%v, wantErr=%v", tt.expression, err, tt.wantErr)
			}
		})
	}
}

func TestCalculateNextRun(t *testing.T) {
	nextRun, err := CalculateNextRun("*/5 * * * *", "UTC")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if nextRun.IsZero() {
		t.Fatalf("expected non-zero next run")
	}
}

func TestCalculateNextNRuns(t *testing.T) {
	runs, err := CalculateNextNRuns("*/10 * * * *", "UTC", 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
}

func TestDescribeCronExpression(t *testing.T) {
	if got := DescribeCronExpression("*/5 * * * *"); got == "" {
		t.Fatalf("expected a description")
	}
}
