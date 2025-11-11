package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
func (m *healthMockRepository) ListExecutionLogs(ctx context.Context, executionID uuid.UUID, level string, limit, offset int) ([]*database.ExecutionLog, error) {
	return nil, nil
}
func (m *healthMockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *healthMockRepository) ListExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
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

func TestHealth(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("[REQ:BAS-FUNC-002] returns degraded when database is healthy but browserless is unavailable", func(t *testing.T) {
		mockRepo := &healthMockRepository{healthy: true}
		// Set browserless to nil to simulate unavailable service
		handler := &Handler{
			repo:        mockRepo,
			browserless: nil,
			log:         log,
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
		// Browserless can be nil here since database failure dominates
		handler := &Handler{
			repo:        mockRepo,
			browserless: nil,
			log:         log,
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
		// Browserless will be nil to simulate it not being available
		handler := &Handler{
			repo:        mockRepo,
			browserless: nil,
			log:         log,
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
		handler := &Handler{
			repo:        nil,
			browserless: nil,
			log:         log,
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
