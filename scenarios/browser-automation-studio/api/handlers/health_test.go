package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// healthMockRepository implements database.Repository for health check testing
type healthMockRepository struct {
	listProjectsErr error
	healthy         bool
}

func (m *healthMockRepository) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	if m.listProjectsErr != nil {
		return nil, m.listProjectsErr
	}
	if !m.healthy {
		return nil, errors.New("database connection failed")
	}
	return []*database.Project{}, nil
}

// Implement other required methods as no-ops
func (m *healthMockRepository) GetProject(ctx context.Context, projectID uuid.UUID) (*database.Project, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *healthMockRepository) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *healthMockRepository) DeleteProject(ctx context.Context, projectID uuid.UUID) error {
	return nil
}
func (m *healthMockRepository) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, nil
}
func (m *healthMockRepository) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *healthMockRepository) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*database.Workflow, error) {
	return nil, nil
}
func (m *healthMockRepository) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *healthMockRepository) DeleteWorkflow(ctx context.Context, workflowID uuid.UUID) error {
	return nil
}
func (m *healthMockRepository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *healthMockRepository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *healthMockRepository) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, nil
}
func (m *healthMockRepository) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateWorkflowVersion(ctx context.Context, version *database.WorkflowVersion) error {
	return nil
}
func (m *healthMockRepository) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.WorkflowVersion, error) {
	return nil, nil
}
func (m *healthMockRepository) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.WorkflowVersion, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *healthMockRepository) GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *healthMockRepository) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *healthMockRepository) DeleteExecution(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *healthMockRepository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
	return nil
}
func (m *healthMockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *healthMockRepository) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *healthMockRepository) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	return nil
}
func (m *healthMockRepository) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	return nil, nil
}
func (m *healthMockRepository) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*database.Workflow, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *healthMockRepository) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *healthMockRepository) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return nil, nil
}
func (m *healthMockRepository) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionArtifact, error) {
	return nil, nil
}
func (m *healthMockRepository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *healthMockRepository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *healthMockRepository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	return nil
}
func (m *healthMockRepository) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	return nil, nil
}
func (m *healthMockRepository) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	return nil, nil
}

// Project file index operations
func (m *healthMockRepository) UpsertProjectEntry(ctx context.Context, entry *database.ProjectEntry) error {
	return nil
}
func (m *healthMockRepository) GetProjectEntry(ctx context.Context, projectID uuid.UUID, path string) (*database.ProjectEntry, error) {
	return nil, nil
}
func (m *healthMockRepository) DeleteProjectEntry(ctx context.Context, projectID uuid.UUID, path string) error {
	return nil
}
func (m *healthMockRepository) DeleteProjectEntries(ctx context.Context, projectID uuid.UUID) error {
	return nil
}
func (m *healthMockRepository) ListProjectEntries(ctx context.Context, projectID uuid.UUID) ([]*database.ProjectEntry, error) {
	return nil, nil
}

// Export operations
func (m *healthMockRepository) CreateExport(ctx context.Context, export *database.Export) error {
	return nil
}
func (m *healthMockRepository) GetExport(ctx context.Context, id uuid.UUID) (*database.Export, error) {
	return nil, nil
}
func (m *healthMockRepository) UpdateExport(ctx context.Context, export *database.Export) error {
	return nil
}
func (m *healthMockRepository) DeleteExport(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *healthMockRepository) ListExports(ctx context.Context, limit, offset int) ([]*database.ExportWithDetails, error) {
	return nil, nil
}
func (m *healthMockRepository) ListExportsByExecution(ctx context.Context, executionID uuid.UUID) ([]*database.Export, error) {
	return nil, nil
}
func (m *healthMockRepository) ListExportsByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.Export, error) {
	return nil, nil
}

// Recovery operations
func (m *healthMockRepository) FindStaleExecutions(ctx context.Context, staleThreshold time.Duration) ([]*database.Execution, error) {
	return nil, nil
}
func (m *healthMockRepository) MarkExecutionInterrupted(ctx context.Context, id uuid.UUID, reason string) error {
	return nil
}
func (m *healthMockRepository) GetLastSuccessfulStepIndex(ctx context.Context, executionID uuid.UUID) (int, error) {
	return -1, nil
}
func (m *healthMockRepository) UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error {
	return nil
}
func (m *healthMockRepository) GetCompletedSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return nil, nil
}
func (m *healthMockRepository) GetResumableExecution(ctx context.Context, id uuid.UUID) (*database.Execution, int, error) {
	return nil, -1, nil
}

// Settings operations
func (m *healthMockRepository) GetSetting(ctx context.Context, key string) (string, error) {
	return "", nil
}
func (m *healthMockRepository) SetSetting(ctx context.Context, key, value string) error {
	return nil
}
func (m *healthMockRepository) DeleteSetting(ctx context.Context, key string) error {
	return nil
}

// Schedule operations
func (m *healthMockRepository) CreateSchedule(ctx context.Context, schedule *database.WorkflowSchedule) error {
	return nil
}
func (m *healthMockRepository) GetSchedule(ctx context.Context, id uuid.UUID) (*database.WorkflowSchedule, error) {
	return nil, nil
}
func (m *healthMockRepository) ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.WorkflowSchedule, error) {
	return nil, nil
}
func (m *healthMockRepository) UpdateSchedule(ctx context.Context, schedule *database.WorkflowSchedule) error {
	return nil
}
func (m *healthMockRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *healthMockRepository) GetActiveSchedules(ctx context.Context) ([]*database.WorkflowSchedule, error) {
	return nil, nil
}
func (m *healthMockRepository) UpdateScheduleNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	return nil
}
func (m *healthMockRepository) UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
	return nil
}

func TestHealth(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("[REQ:BAS-FUNC-002] returns degraded when database is healthy but browserless is unavailable", func(t *testing.T) {
		mockRepo := &healthMockRepository{healthy: true}
		svc := &workflowServiceMock{
			automationHealthy: false,
			automationErr:     errors.New("automation engine unavailable"),
		}
		handler := &Handler{
			repo:             mockRepo,
			workflowCatalog:  svc,
			executionService: svc,
			exportService:    svc,
			log:              log,
		}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.Health(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for degraded state, got %d", w.Code)
		}

		var response HealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "degraded" {
			t.Errorf("Expected status 'degraded', got '%s'", response.Status)
		}
		if !response.Readiness {
			t.Error("Expected readiness to be true for degraded state")
		}
		if response.Service != "browser-automation-studio-api" {
			t.Errorf("Expected service 'browser-automation-studio-api', got '%s'", response.Service)
		}
	})

	t.Run("[REQ:BAS-FUNC-002] returns unhealthy when database is down", func(t *testing.T) {
		mockRepo := &healthMockRepository{listProjectsErr: errors.New("connection refused")}
		svc := &workflowServiceMock{automationHealthy: true}
		handler := &Handler{
			repo:             mockRepo,
			workflowCatalog:  svc,
			executionService: svc,
			exportService:    svc,
			log:              log,
		}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.Health(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}

		var response HealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
		}
		if response.Readiness {
			t.Error("Expected readiness to be false")
		}
	})

	t.Run("[REQ:BAS-FUNC-002] returns degraded when browserless is down but database is up", func(t *testing.T) {
		mockRepo := &healthMockRepository{healthy: true}
		svc := &workflowServiceMock{
			automationHealthy: false,
			automationErr:     errors.New("automation engine unavailable"),
		}
		handler := &Handler{
			repo:             mockRepo,
			workflowCatalog:  svc,
			executionService: svc,
			exportService:    svc,
			log:              log,
		}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.Health(w, req)

		// Degraded state still returns 200 OK (service is partially functional)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response HealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "degraded" {
			t.Errorf("Expected status 'degraded', got '%s'", response.Status)
		}
		if !response.Readiness {
			t.Error("Expected readiness to be true for degraded state")
		}
	})

	t.Run("[REQ:BAS-FUNC-002] returns error when repository is nil", func(t *testing.T) {
		svc := &workflowServiceMock{automationHealthy: true}
		handler := &Handler{
			repo:             nil,
			workflowCatalog:  svc,
			executionService: svc,
			exportService:    svc,
			log:              log,
		}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler.Health(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}

		var response HealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
		}
		if response.Readiness {
			t.Error("Expected readiness to be false")
		}
	})
}
