package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

type mockScheduleRepo struct {
	mu        sync.RWMutex
	schedules map[uuid.UUID]*database.ScheduleIndex

	nextRunCalls atomic.Int32
	lastRunCalls atomic.Int32
}

func newMockScheduleRepo(schedules ...*database.ScheduleIndex) *mockScheduleRepo {
	repo := &mockScheduleRepo{
		schedules: make(map[uuid.UUID]*database.ScheduleIndex),
	}
	for _, s := range schedules {
		if s == nil {
			continue
		}
		repo.schedules[s.ID] = s
	}
	return repo
}

func (m *mockScheduleRepo) ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.ScheduleIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*database.ScheduleIndex
	for _, schedule := range m.schedules {
		if schedule == nil {
			continue
		}
		if workflowID != nil && schedule.WorkflowID != *workflowID {
			continue
		}
		if activeOnly && !schedule.IsActive {
			continue
		}
		result = append(result, schedule)
	}
	return result, nil
}

func (m *mockScheduleRepo) GetSchedule(ctx context.Context, id uuid.UUID) (*database.ScheduleIndex, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if schedule, ok := m.schedules[id]; ok && schedule != nil {
		return schedule, nil
	}
	return nil, database.ErrNotFound
}

func (m *mockScheduleRepo) UpdateScheduleNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	m.nextRunCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	schedule, ok := m.schedules[id]
	if !ok || schedule == nil {
		return database.ErrNotFound
	}
	schedule.NextRunAt = &nextRun
	return nil
}

func (m *mockScheduleRepo) UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
	m.lastRunCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	schedule, ok := m.schedules[id]
	if !ok || schedule == nil {
		return database.ErrNotFound
	}
	schedule.LastRunAt = &lastRun
	return nil
}

type fakeExecutor struct {
	calls  atomic.Int32
	lastID atomic.Value

	exec *database.ExecutionIndex
	err  error
}

func (f *fakeExecutor) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	f.calls.Add(1)
	f.lastID.Store(workflowID)
	if f.err != nil {
		return nil, f.err
	}
	if f.exec != nil {
		return f.exec, nil
	}
	return &database.ExecutionIndex{ID: uuid.New(), WorkflowID: workflowID, Status: database.ExecutionStatusCompleted, StartedAt: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
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
	executor := &fakeExecutor{}
	notifier := &fakeNotifier{}

	s := New(repo, executor, notifier, testLogger())
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer s.Stop()

	if got := s.RegisteredCount(); got != 0 {
		t.Fatalf("expected 0 registered schedules, got %d", got)
	}
}

func TestSchedulerStartWithActiveSchedules(t *testing.T) {
	workflowID := uuid.New()
	s1 := newSchedule(uuid.New(), workflowID, "*/5 * * * * *", true)
	s2 := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", true)
	s3 := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", false)

	repo := newMockScheduleRepo(s1, s2, s3)
	executor := &fakeExecutor{}
	notifier := &fakeNotifier{}

	s := New(repo, executor, notifier, testLogger())
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer s.Stop()

	if got := s.RegisteredCount(); got != 2 {
		t.Fatalf("expected 2 registered schedules, got %d", got)
	}
	if repo.nextRunCalls.Load() != 2 {
		t.Fatalf("expected nextRunCalls=2, got %d", repo.nextRunCalls.Load())
	}
}

func TestSchedulerCannotStartTwice(t *testing.T) {
	repo := newMockScheduleRepo()
	executor := &fakeExecutor{}
	s := New(repo, executor, nil, testLogger())

	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer s.Stop()

	if err := s.Start(); err == nil {
		t.Fatalf("expected error on second start")
	}
}

func TestSchedulerRegisterSchedule(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", true)

	repo := newMockScheduleRepo()
	repo.schedules[schedule.ID] = schedule

	s := New(repo, &fakeExecutor{}, nil, testLogger())
	if err := s.RegisterSchedule(schedule); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := s.RegisteredCount(); got != 1 {
		t.Fatalf("expected 1 registered schedule, got %d", got)
	}
}

func TestSchedulerRegisterInactiveSchedule(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", false)

	repo := newMockScheduleRepo()
	repo.schedules[schedule.ID] = schedule

	s := New(repo, &fakeExecutor{}, nil, testLogger())
	if err := s.RegisterSchedule(schedule); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := s.RegisteredCount(); got != 0 {
		t.Fatalf("expected 0 registered schedules, got %d", got)
	}
}

func TestSchedulerUnregisterSchedule(t *testing.T) {
	workflowID := uuid.New()
	schedule := newSchedule(uuid.New(), workflowID, "*/10 * * * * *", true)

	repo := newMockScheduleRepo(schedule)
	s := New(repo, &fakeExecutor{}, nil, testLogger())
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
	s := New(repo, &fakeExecutor{}, nil, testLogger())
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
	executor := &fakeExecutor{
		exec: &database.ExecutionIndex{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusCompleted,
			StartedAt:  time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
	notifier := &fakeNotifier{}

	s := New(repo, executor, notifier, testLogger())
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer s.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if executor.calls.Load() > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if executor.calls.Load() == 0 {
		t.Fatalf("expected scheduled execution to run")
	}
	if repo.lastRunCalls.Load() == 0 {
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
	executor := &fakeExecutor{err: errors.New("boom")}
	notifier := &fakeNotifier{}

	s := New(repo, executor, notifier, testLogger())
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer s.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if executor.calls.Load() > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if executor.calls.Load() == 0 {
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

	s := New(repo, &fakeExecutor{}, nil, testLogger())
	if err := s.Start(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("expected no error stopping scheduler, got %v", err)
	}
}

func TestSchedulerConcurrentRegistration(t *testing.T) {
	repo := newMockScheduleRepo()
	s := New(repo, &fakeExecutor{}, nil, testLogger())

	const n = 25
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			schedule := newSchedule(uuid.New(), uuid.New(), "*/10 * * * * *", true)
			repo.mu.Lock()
			repo.schedules[schedule.ID] = schedule
			repo.mu.Unlock()
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

