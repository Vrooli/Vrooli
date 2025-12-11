package recovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// mockRepo implements the minimal repository interface for testing recovery.
type mockRepo struct {
	staleExecutions  []*database.Execution
	lastStepIndexes  map[uuid.UUID]int
	interruptedCalls []uuid.UUID
	findErr          error
	markErr          error
	stepErr          error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		lastStepIndexes: make(map[uuid.UUID]int),
	}
}

func (m *mockRepo) FindStaleExecutions(ctx context.Context, threshold time.Duration) ([]*database.Execution, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.staleExecutions, nil
}

func (m *mockRepo) MarkExecutionInterrupted(ctx context.Context, id uuid.UUID, reason string) error {
	if m.markErr != nil {
		return m.markErr
	}
	m.interruptedCalls = append(m.interruptedCalls, id)
	return nil
}

func (m *mockRepo) GetLastSuccessfulStepIndex(ctx context.Context, executionID uuid.UUID) (int, error) {
	if m.stepErr != nil {
		return -1, m.stepErr
	}
	if idx, ok := m.lastStepIndexes[executionID]; ok {
		return idx, nil
	}
	return -1, nil
}

func (m *mockRepo) UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error {
	return nil
}

// Additional methods to satisfy full Repository interface (stubs)
func (m *mockRepo) CreateProject(ctx context.Context, project *database.Project) error { return nil }
func (m *mockRepo) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepo) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepo) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepo) UpdateProject(ctx context.Context, project *database.Project) error { return nil }
func (m *mockRepo) DeleteProject(ctx context.Context, id uuid.UUID) error             { return nil }
func (m *mockRepo) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *mockRepo) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, nil
}
func (m *mockRepo) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return nil, nil
}
func (m *mockRepo) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error { return nil }
func (m *mockRepo) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepo) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepo) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepo) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error { return nil }
func (m *mockRepo) DeleteWorkflow(ctx context.Context, id uuid.UUID) error                { return nil }
func (m *mockRepo) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *mockRepo) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepo) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepo) CreateWorkflowVersion(ctx context.Context, version *database.WorkflowVersion) error {
	return nil
}
func (m *mockRepo) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.WorkflowVersion, error) {
	return nil, nil
}
func (m *mockRepo) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.WorkflowVersion, error) {
	return nil, nil
}
func (m *mockRepo) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepo) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *mockRepo) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepo) DeleteExecution(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepo) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *mockRepo) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *mockRepo) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *mockRepo) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return nil, nil
}
func (m *mockRepo) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	return nil
}
func (m *mockRepo) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionArtifact, error) {
	return nil, nil
}
func (m *mockRepo) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *mockRepo) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *mockRepo) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
	return nil
}
func (m *mockRepo) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}
func (m *mockRepo) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *mockRepo) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}
func (m *mockRepo) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	return nil
}
func (m *mockRepo) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	return nil, nil
}
func (m *mockRepo) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	return nil, nil
}
func (m *mockRepo) CreateExport(ctx context.Context, export *database.Export) error { return nil }
func (m *mockRepo) GetExport(ctx context.Context, id uuid.UUID) (*database.Export, error) {
	return nil, nil
}
func (m *mockRepo) UpdateExport(ctx context.Context, export *database.Export) error { return nil }
func (m *mockRepo) DeleteExport(ctx context.Context, id uuid.UUID) error            { return nil }
func (m *mockRepo) ListExports(ctx context.Context, limit, offset int) ([]*database.ExportWithDetails, error) {
	return nil, nil
}
func (m *mockRepo) ListExportsByExecution(ctx context.Context, executionID uuid.UUID) ([]*database.Export, error) {
	return nil, nil
}
func (m *mockRepo) ListExportsByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.Export, error) {
	return nil, nil
}
func (m *mockRepo) GetCompletedSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return nil, nil
}
func (m *mockRepo) GetResumableExecution(ctx context.Context, id uuid.UUID) (*database.Execution, int, error) {
	return nil, -1, nil
}

func TestRecoverStaleExecutions_NoStaleExecutions(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(repo, log)
	result, err := svc.RecoverStaleExecutions(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.TotalStale != 0 {
		t.Errorf("expected 0 stale executions, got %d", result.TotalStale)
	}
	if result.Recovered != 0 {
		t.Errorf("expected 0 recovered, got %d", result.Recovered)
	}
}

func TestRecoverStaleExecutions_RecoversSingleExecution(t *testing.T) {
	t.Parallel()
	execID := uuid.New()
	workflowID := uuid.New()

	repo := newMockRepo()
	repo.staleExecutions = []*database.Execution{
		{
			ID:         execID,
			WorkflowID: workflowID,
			Status:     "running",
			StartedAt:  time.Now().Add(-10 * time.Minute),
		},
	}
	repo.lastStepIndexes[execID] = 5 // Had 5 successful steps

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(repo, log)
	result, err := svc.RecoverStaleExecutions(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.TotalStale != 1 {
		t.Errorf("expected 1 stale execution, got %d", result.TotalStale)
	}
	if result.Recovered != 1 {
		t.Errorf("expected 1 recovered, got %d", result.Recovered)
	}
	if result.Resumable != 1 {
		t.Errorf("expected 1 resumable, got %d", result.Resumable)
	}
	if len(repo.interruptedCalls) != 1 || repo.interruptedCalls[0] != execID {
		t.Errorf("expected MarkExecutionInterrupted to be called with %v, got %v", execID, repo.interruptedCalls)
	}
}

func TestRecoverStaleExecutions_NonResumableExecution(t *testing.T) {
	t.Parallel()
	execID := uuid.New()
	workflowID := uuid.New()

	repo := newMockRepo()
	repo.staleExecutions = []*database.Execution{
		{
			ID:         execID,
			WorkflowID: workflowID,
			Status:     "pending",
			StartedAt:  time.Now().Add(-10 * time.Minute),
		},
	}
	// No successful steps - lastStepIndexes not set, defaults to -1

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(repo, log)
	result, err := svc.RecoverStaleExecutions(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.TotalStale != 1 {
		t.Errorf("expected 1 stale execution, got %d", result.TotalStale)
	}
	if result.Recovered != 1 {
		t.Errorf("expected 1 recovered, got %d", result.Recovered)
	}
	if result.Resumable != 0 {
		t.Errorf("expected 0 resumable (no successful steps), got %d", result.Resumable)
	}
}

func TestRecoverStaleExecutions_MultipleExecutions(t *testing.T) {
	t.Parallel()
	exec1 := uuid.New()
	exec2 := uuid.New()
	exec3 := uuid.New()
	workflowID := uuid.New()

	repo := newMockRepo()
	repo.staleExecutions = []*database.Execution{
		{ID: exec1, WorkflowID: workflowID, Status: "running"},
		{ID: exec2, WorkflowID: workflowID, Status: "running"},
		{ID: exec3, WorkflowID: workflowID, Status: "pending"},
	}
	repo.lastStepIndexes[exec1] = 3
	repo.lastStepIndexes[exec2] = 0
	// exec3 has no steps

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(repo, log)
	result, err := svc.RecoverStaleExecutions(context.Background())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.TotalStale != 3 {
		t.Errorf("expected 3 stale executions, got %d", result.TotalStale)
	}
	if result.Recovered != 3 {
		t.Errorf("expected 3 recovered, got %d", result.Recovered)
	}
	if result.Resumable != 2 {
		t.Errorf("expected 2 resumable (exec1 and exec2 have steps), got %d", result.Resumable)
	}
}

func TestRecoverStaleExecutions_FindError(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	repo.findErr = errors.New("database connection failed")

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(repo, log)
	_, err := svc.RecoverStaleExecutions(context.Background())

	if err == nil {
		t.Error("expected error when FindStaleExecutions fails")
	}
}

func TestRecoverStaleExecutions_MarkInterruptedError(t *testing.T) {
	t.Parallel()
	execID := uuid.New()

	repo := newMockRepo()
	repo.staleExecutions = []*database.Execution{
		{ID: execID, WorkflowID: uuid.New(), Status: "running"},
	}
	repo.markErr = errors.New("update failed")

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(repo, log)
	result, err := svc.RecoverStaleExecutions(context.Background())

	if err != nil {
		t.Errorf("expected no error (partial recovery allowed), got %v", err)
	}
	if result.TotalStale != 1 {
		t.Errorf("expected 1 stale execution, got %d", result.TotalStale)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed recovery, got %d", result.Failed)
	}
	if result.Recovered != 0 {
		t.Errorf("expected 0 recovered, got %d", result.Recovered)
	}
}

func TestWithStaleThreshold(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	log := logrus.New()

	customThreshold := 10 * time.Minute
	svc := NewService(repo, log, WithStaleThreshold(customThreshold))

	if svc.staleThreshold != customThreshold {
		t.Errorf("expected stale threshold %v, got %v", customThreshold, svc.staleThreshold)
	}
}

func TestRecoveryResult_DurationTracking(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(repo, log)
	result, _ := svc.RecoverStaleExecutions(context.Background())

	if result.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
	if result.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
	if result.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", result.DurationMs)
	}
}
