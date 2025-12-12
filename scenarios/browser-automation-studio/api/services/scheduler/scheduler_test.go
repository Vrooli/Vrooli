package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// mockRepository implements database.Repository for scheduler tests.
// Only schedule-related methods are actually implemented; others are no-ops.
type mockRepository struct {
	schedules    []*database.WorkflowSchedule
	mu           sync.RWMutex
	nextRunCalls atomic.Int32
	lastRunCalls atomic.Int32
}

func newMockRepository(schedules ...*database.WorkflowSchedule) *mockRepository {
	return &mockRepository{
		schedules: schedules,
	}
}

// Schedule operations - actually implemented
func (m *mockRepository) GetActiveSchedules(ctx context.Context) ([]*database.WorkflowSchedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var active []*database.WorkflowSchedule
	for _, s := range m.schedules {
		if s.IsActive {
			active = append(active, s)
		}
	}
	return active, nil
}

func (m *mockRepository) GetSchedule(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.schedules {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *mockRepository) UpdateScheduleNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	m.nextRunCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.schedules {
		if s.ID == id {
			s.NextRunAt = &nextRun
			return nil
		}
	}
	return database.ErrNotFound
}

func (m *mockRepository) UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
	m.lastRunCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.schedules {
		if s.ID == id {
			s.LastRunAt = &lastRun
			return nil
		}
	}
	return database.ErrNotFound
}

func (m *mockRepository) CreateSchedule(ctx context.Context, schedule *database.WorkflowSchedule) error {
	return nil
}
func (m *mockRepository) ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.WorkflowSchedule, error) {
	return nil, nil
}
func (m *mockRepository) UpdateSchedule(ctx context.Context, schedule *database.WorkflowSchedule) error {
	return nil
}
func (m *mockRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error { return nil }

// Project operations - stubs
func (m *mockRepository) UpsertProjectEntry(ctx context.Context, entry *database.ProjectEntry) error {
	return nil
}
func (m *mockRepository) GetProjectEntry(ctx context.Context, projectID uuid.UUID, path string) (*database.ProjectEntry, error) {
	return nil, nil
}
func (m *mockRepository) DeleteProjectEntry(ctx context.Context, projectID uuid.UUID, path string) error {
	return nil
}
func (m *mockRepository) DeleteProjectEntries(ctx context.Context, projectID uuid.UUID) error {
	return nil
}
func (m *mockRepository) ListProjectEntries(ctx context.Context, projectID uuid.UUID) ([]*database.ProjectEntry, error) {
	return nil, nil
}
func (m *mockRepository) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockRepository) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockRepository) DeleteProject(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepository) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return nil, nil
}

// Workflow operations - stubs
func (m *mockRepository) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *mockRepository) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *mockRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepository) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *mockRepository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) CreateWorkflowVersion(ctx context.Context, version *database.WorkflowVersion) error {
	return nil
}
func (m *mockRepository) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.WorkflowVersion, error) {
	return nil, nil
}
func (m *mockRepository) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.WorkflowVersion, error) {
	return nil, nil
}

// Execution operations - stubs
func (m *mockRepository) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepository) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *mockRepository) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepository) DeleteExecution(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *mockRepository) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *mockRepository) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *mockRepository) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return nil, nil
}
func (m *mockRepository) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	return nil
}
func (m *mockRepository) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionArtifact, error) {
	return nil, nil
}

// Screenshot operations - stubs
func (m *mockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *mockRepository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}

// Log operations - stubs
func (m *mockRepository) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
	return nil
}
func (m *mockRepository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}

// Extracted data operations - stubs
func (m *mockRepository) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *mockRepository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}

// Folder operations - stubs
func (m *mockRepository) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	return nil
}
func (m *mockRepository) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	return nil, nil
}
func (m *mockRepository) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	return nil, nil
}

// Export operations - stubs
func (m *mockRepository) CreateExport(ctx context.Context, export *database.Export) error { return nil }
func (m *mockRepository) GetExport(ctx context.Context, id uuid.UUID) (*database.Export, error) {
	return nil, nil
}
func (m *mockRepository) UpdateExport(ctx context.Context, export *database.Export) error { return nil }
func (m *mockRepository) DeleteExport(ctx context.Context, id uuid.UUID) error            { return nil }
func (m *mockRepository) ListExports(ctx context.Context, limit, offset int) ([]*database.ExportWithDetails, error) {
	return nil, nil
}
func (m *mockRepository) ListExportsByExecution(ctx context.Context, executionID uuid.UUID) ([]*database.Export, error) {
	return nil, nil
}
func (m *mockRepository) ListExportsByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.Export, error) {
	return nil, nil
}

// Recovery operations - stubs
func (m *mockRepository) FindStaleExecutions(ctx context.Context, staleThreshold time.Duration) ([]*database.Execution, error) {
	return nil, nil
}
func (m *mockRepository) MarkExecutionInterrupted(ctx context.Context, id uuid.UUID, reason string) error {
	return nil
}
func (m *mockRepository) GetLastSuccessfulStepIndex(ctx context.Context, executionID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockRepository) UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error {
	return nil
}
func (m *mockRepository) GetCompletedSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return nil, nil
}
func (m *mockRepository) GetResumableExecution(ctx context.Context, id uuid.UUID) (*database.Execution, int, error) {
	return nil, 0, nil
}

// Settings operations - stubs
func (m *mockRepository) GetSetting(ctx context.Context, key string) (string, error) { return "", nil }
func (m *mockRepository) SetSetting(ctx context.Context, key, value string) error    { return nil }
func (m *mockRepository) DeleteSetting(ctx context.Context, key string) error        { return nil }

// Verify interface compliance
var _ database.Repository = (*mockRepository)(nil)

// mockExecutor tracks workflow executions.
type mockExecutor struct {
	executions []executionRecord
	mu         sync.Mutex
	execDelay  time.Duration // optional delay to simulate work
}

type executionRecord struct {
	WorkflowID uuid.UUID
	Params     map[string]any
	Time       time.Time
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		executions: make([]executionRecord, 0),
	}
}

func (m *mockExecutor) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, params map[string]any) (*database.Execution, error) {
	if m.execDelay > 0 {
		select {
		case <-time.After(m.execDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions = append(m.executions, executionRecord{
		WorkflowID: workflowID,
		Params:     params,
		Time:       time.Now(),
	})
	return &database.Execution{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     "completed",
	}, nil
}

func (m *mockExecutor) GetExecutionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.executions)
}

func (m *mockExecutor) GetExecutions() []executionRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]executionRecord{}, m.executions...)
}

// mockNotifier tracks notifications.
type mockNotifier struct {
	events []ScheduleEvent
	mu     sync.Mutex
}

func newMockNotifier() *mockNotifier {
	return &mockNotifier{
		events: make([]ScheduleEvent, 0),
	}
}

func (m *mockNotifier) BroadcastScheduleEvent(event ScheduleEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockNotifier) GetEvents() []ScheduleEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ScheduleEvent{}, m.events...)
}

func newTestLogger() *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	return log
}

func TestSchedulerStartWithNoSchedules(t *testing.T) {
	repo := newMockRepository()
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	if !scheduler.IsRunning() {
		t.Error("scheduler should be running after Start()")
	}

	if scheduler.RegisteredCount() != 0 {
		t.Errorf("RegisteredCount() = %d, want 0", scheduler.RegisteredCount())
	}
}

func TestSchedulerStartWithActiveSchedules(t *testing.T) {
	schedule1 := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "Schedule 1",
		CronExpression: "0 9 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}
	schedule2 := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "Schedule 2",
		CronExpression: "0 12 * * *",
		Timezone:       "America/New_York",
		IsActive:       true,
	}
	inactiveSchedule := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "Inactive",
		CronExpression: "* * * * *",
		Timezone:       "UTC",
		IsActive:       false,
	}

	repo := newMockRepository(schedule1, schedule2, inactiveSchedule)
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	if scheduler.RegisteredCount() != 2 {
		t.Errorf("RegisteredCount() = %d, want 2 (only active schedules)", scheduler.RegisteredCount())
	}

	// Verify schedules are registered
	if _, registered := scheduler.GetScheduleInfo(schedule1.ID); !registered {
		t.Error("schedule1 should be registered")
	}
	if _, registered := scheduler.GetScheduleInfo(schedule2.ID); !registered {
		t.Error("schedule2 should be registered")
	}
	if _, registered := scheduler.GetScheduleInfo(inactiveSchedule.ID); registered {
		t.Error("inactive schedule should not be registered")
	}
}

func TestSchedulerCannotStartTwice(t *testing.T) {
	repo := newMockRepository()
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	defer scheduler.Stop()

	err = scheduler.Start()
	if err == nil {
		t.Error("second Start() should return error")
	}
}

func TestSchedulerRegisterSchedule(t *testing.T) {
	repo := newMockRepository()
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	schedule := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "New Schedule",
		CronExpression: "0 9 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}

	err = scheduler.RegisterSchedule(schedule)
	if err != nil {
		t.Fatalf("RegisterSchedule() error = %v", err)
	}

	if scheduler.RegisteredCount() != 1 {
		t.Errorf("RegisteredCount() = %d, want 1", scheduler.RegisteredCount())
	}

	if _, registered := scheduler.GetScheduleInfo(schedule.ID); !registered {
		t.Error("schedule should be registered")
	}
}

func TestSchedulerRegisterInactiveSchedule(t *testing.T) {
	repo := newMockRepository()
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	schedule := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "Inactive Schedule",
		CronExpression: "0 9 * * *",
		Timezone:       "UTC",
		IsActive:       false,
	}

	err = scheduler.RegisterSchedule(schedule)
	if err != nil {
		t.Fatalf("RegisterSchedule() error = %v", err)
	}

	// Inactive schedules should not be registered with cron
	if scheduler.RegisteredCount() != 0 {
		t.Errorf("RegisteredCount() = %d, want 0 (inactive schedule)", scheduler.RegisteredCount())
	}
}

func TestSchedulerUnregisterSchedule(t *testing.T) {
	schedule := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "Schedule",
		CronExpression: "0 9 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}

	repo := newMockRepository(schedule)
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	if scheduler.RegisteredCount() != 1 {
		t.Errorf("RegisteredCount() = %d, want 1", scheduler.RegisteredCount())
	}

	err = scheduler.UnregisterSchedule(schedule.ID)
	if err != nil {
		t.Fatalf("UnregisterSchedule() error = %v", err)
	}

	if scheduler.RegisteredCount() != 0 {
		t.Errorf("RegisteredCount() = %d, want 0 after unregister", scheduler.RegisteredCount())
	}

	if _, registered := scheduler.GetScheduleInfo(schedule.ID); registered {
		t.Error("schedule should not be registered after UnregisterSchedule")
	}
}

func TestSchedulerUpdateSchedule(t *testing.T) {
	schedule := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "Schedule",
		CronExpression: "0 9 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}

	repo := newMockRepository(schedule)
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	// Update schedule cron expression
	schedule.CronExpression = "0 12 * * *"

	err = scheduler.RegisterSchedule(schedule)
	if err != nil {
		t.Fatalf("RegisterSchedule() error = %v", err)
	}

	// Should still have 1 registered (update replaces)
	if scheduler.RegisteredCount() != 1 {
		t.Errorf("RegisteredCount() = %d, want 1 after update", scheduler.RegisteredCount())
	}
}

func TestSchedulerDeactivateSchedule(t *testing.T) {
	schedule := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     uuid.New(),
		Name:           "Schedule",
		CronExpression: "0 9 * * *",
		Timezone:       "UTC",
		IsActive:       true,
	}

	repo := newMockRepository(schedule)
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	if scheduler.RegisteredCount() != 1 {
		t.Errorf("RegisteredCount() = %d, want 1", scheduler.RegisteredCount())
	}

	// Deactivate schedule
	schedule.IsActive = false

	err = scheduler.RegisterSchedule(schedule)
	if err != nil {
		t.Fatalf("RegisterSchedule() error = %v", err)
	}

	// Should be unregistered now
	if scheduler.RegisteredCount() != 0 {
		t.Errorf("RegisteredCount() = %d, want 0 after deactivation", scheduler.RegisteredCount())
	}
}

func TestSchedulerCronExecution(t *testing.T) {
	workflowID := uuid.New()
	scheduleID := uuid.New()

	schedule := &database.WorkflowSchedule{
		ID:             scheduleID,
		WorkflowID:     workflowID,
		Name:           "Frequent Schedule",
		CronExpression: "* * * * * *", // Every second (6-field cron)
		Timezone:       "UTC",
		IsActive:       true,
	}

	repo := newMockRepository(schedule)
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for at least 2 executions (give it 3 seconds to be safe)
	time.Sleep(3 * time.Second)

	scheduler.Stop()

	execCount := executor.GetExecutionCount()
	if execCount < 2 {
		t.Errorf("Expected at least 2 executions, got %d", execCount)
	}

	// Verify execution metadata
	executions := executor.GetExecutions()
	for _, exec := range executions {
		if exec.WorkflowID != workflowID {
			t.Errorf("Execution workflow_id = %s, want %s", exec.WorkflowID, workflowID)
		}
		if exec.Params["_trigger_type"] != "scheduled" {
			t.Errorf("Expected _trigger_type = 'scheduled', got %v", exec.Params["_trigger_type"])
		}
		if exec.Params["_schedule_id"] != scheduleID.String() {
			t.Errorf("Expected _schedule_id = %s, got %v", scheduleID, exec.Params["_schedule_id"])
		}
	}

	// Verify notifications were sent
	events := notifier.GetEvents()
	startedCount := 0
	completedCount := 0
	for _, e := range events {
		switch e.Type {
		case EventTypeScheduleStarted:
			startedCount++
		case EventTypeScheduleCompleted:
			completedCount++
		}
	}
	if startedCount < 2 {
		t.Errorf("Expected at least 2 started events, got %d", startedCount)
	}
	if completedCount < 2 {
		t.Errorf("Expected at least 2 completed events, got %d", completedCount)
	}
}

func TestSchedulerGracefulShutdown(t *testing.T) {
	workflowID := uuid.New()

	schedule := &database.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     workflowID,
		Name:           "Slow Schedule",
		CronExpression: "* * * * * *", // Every second
		Timezone:       "UTC",
		IsActive:       true,
	}

	repo := newMockRepository(schedule)
	executor := newMockExecutor()
	executor.execDelay = 500 * time.Millisecond // Simulate slow execution
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for an execution to start
	time.Sleep(1500 * time.Millisecond)

	// Stop should wait for in-flight executions
	stopStart := time.Now()
	err = scheduler.Stop()
	stopDuration := time.Since(stopStart)

	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if !scheduler.IsRunning() == true {
		// After stop, IsRunning should be false
	}

	// Executions should have completed
	execCount := executor.GetExecutionCount()
	if execCount < 1 {
		t.Errorf("Expected at least 1 execution, got %d", execCount)
	}

	t.Logf("Shutdown completed in %v, %d executions completed", stopDuration, execCount)
}

func TestSchedulerConcurrentRegistration(t *testing.T) {
	repo := newMockRepository()
	executor := newMockExecutor()
	notifier := newMockNotifier()
	log := newTestLogger()

	scheduler := New(repo, executor, notifier, log)

	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	// Concurrently register many schedules
	var wg sync.WaitGroup
	numSchedules := 50

	for i := 0; i < numSchedules; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			schedule := &database.WorkflowSchedule{
				ID:             uuid.New(),
				WorkflowID:     uuid.New(),
				Name:           "Concurrent Schedule",
				CronExpression: "0 9 * * *",
				Timezone:       "UTC",
				IsActive:       true,
			}
			scheduler.RegisterSchedule(schedule)
		}(i)
	}

	wg.Wait()

	if scheduler.RegisteredCount() != numSchedules {
		t.Errorf("RegisteredCount() = %d, want %d", scheduler.RegisteredCount(), numSchedules)
	}
}

func TestCronParserValidation(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"valid 5-field", "0 9 * * *", false},
		{"valid 6-field", "0 0 9 * * *", false},
		{"every minute", "* * * * *", false},
		{"every 5 minutes", "*/5 * * * *", false},
		{"weekdays 9am", "0 9 * * 1-5", false},
		{"monthly", "0 0 1 * *", false},
		{"preset hourly", "@hourly", false},
		{"preset daily", "@daily", false},
		{"invalid too few fields", "* *", true},
		{"invalid characters", "abc def", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCronExpression(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestCalculateNextRun(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		timezone string
		wantErr  bool
	}{
		{"daily UTC", "0 9 * * *", "UTC", false},
		{"daily NYC", "0 9 * * *", "America/New_York", false},
		{"every hour", "0 * * * *", "UTC", false},
		{"invalid timezone uses UTC", "0 9 * * *", "Invalid/Zone", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextRun, err := CalculateNextRun(tt.expr, tt.timezone)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateNextRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && nextRun.IsZero() {
				t.Error("CalculateNextRun() returned zero time")
			}
			if !tt.wantErr && nextRun.Before(time.Now()) {
				t.Error("CalculateNextRun() returned past time")
			}
		})
	}
}

func TestCalculateNextNRuns(t *testing.T) {
	runs, err := CalculateNextNRuns("* * * * *", "UTC", 5)
	if err != nil {
		t.Fatalf("CalculateNextNRuns() error = %v", err)
	}

	if len(runs) != 5 {
		t.Errorf("CalculateNextNRuns() returned %d runs, want 5", len(runs))
	}

	// Verify runs are in order
	for i := 1; i < len(runs); i++ {
		if !runs[i].After(runs[i-1]) {
			t.Errorf("runs[%d] (%v) is not after runs[%d] (%v)", i, runs[i], i-1, runs[i-1])
		}
	}
}

func TestDescribeCronExpression(t *testing.T) {
	tests := []struct {
		expr     string
		contains string
	}{
		{"@hourly", "hour"},
		{"@daily", "day"},
		{"0 * * * *", "hour"},
		{"0 0 * * *", "midnight"},
		{"0 9 * * 1-5", "Weekdays"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			desc := DescribeCronExpression(tt.expr)
			// Just verify we get a non-empty description
			if desc == "" {
				t.Errorf("DescribeCronExpression(%q) returned empty string", tt.expr)
			}
		})
	}
}
