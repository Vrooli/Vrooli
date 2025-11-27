package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/logutil"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// mockRepository implements database.Repository for testing
type mockRepository struct{}

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
func (m *mockRepository) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepository) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *mockRepository) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepository) DeleteExecution(ctx context.Context, id uuid.UUID) error {
	return nil
}
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
func (m *mockRepository) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
	return nil
}
func (m *mockRepository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}
func (m *mockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *mockRepository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *mockRepository) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *mockRepository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}
func (m *mockRepository) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	return nil
}
func (m *mockRepository) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	return nil, nil
}
func (m *mockRepository) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	return nil, nil
}

// Ensure mockRepository implements the interface at compile time
var _ database.Repository = (*mockRepository)(nil)

type workflowServiceMock struct {
	listWorkflowVersionsFn    func(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*workflow.WorkflowVersionSummary, error)
	getWorkflowVersionFn      func(ctx context.Context, workflowID uuid.UUID, version int) (*workflow.WorkflowVersionSummary, error)
	restoreWorkflowVersionFn  func(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error)
	describeExecutionExportFn func(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error)
	executeAdhocWorkflowFn    func(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error)
	automationHealthy         bool
	automationErr             error
}

// Ensure workflowServiceMock stays in sync with the WorkflowService interface
var _ WorkflowService = (*workflowServiceMock)(nil)

func (m *workflowServiceMock) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*workflow.WorkflowVersionSummary, error) {
	if m.listWorkflowVersionsFn != nil {
		return m.listWorkflowVersionsFn(ctx, workflowID, limit, offset)
	}
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*workflow.WorkflowVersionSummary, error) {
	if m.getWorkflowVersionFn != nil {
		return m.getWorkflowVersionFn(ctx, workflowID, version)
	}
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error) {
	if m.restoreWorkflowVersionFn != nil {
		return m.restoreWorkflowVersionFn(ctx, workflowID, version, changeDescription)
	}
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	if m.executeAdhocWorkflowFn != nil {
		return m.executeAdhocWorkflowFn(ctx, flowDefinition, parameters, name)
	}
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*export.ExecutionTimeline, error) {
	return nil, nil
}

func (m *workflowServiceMock) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	if m.describeExecutionExportFn != nil {
		return m.describeExecutionExportFn(ctx, executionID)
	}
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *workflowServiceMock) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) CreateProject(ctx context.Context, project *database.Project) error {
	return errors.New("not implemented")
}

func (m *workflowServiceMock) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) GetProject(ctx context.Context, projectID uuid.UUID) (*database.Project, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) UpdateProject(ctx context.Context, project *database.Project) error {
	return errors.New("not implemented")
}

func (m *workflowServiceMock) DeleteProject(ctx context.Context, projectID uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *workflowServiceMock) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, errors.New("not implemented")
}

func (m *workflowServiceMock) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *workflowServiceMock) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	return errors.New("not implemented")
}

func (m *workflowServiceMock) CheckAutomationHealth(ctx context.Context) (bool, error) {
	if m.automationErr != nil {
		return false, m.automationErr
	}
	if m.automationHealthy {
		return true, nil
	}
	return false, nil
}

// TestNewHandler verifies handler initialization with all dependencies
func TestNewHandler(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard) // Suppress output during tests

	// Create mock dependencies
	repo := &mockRepository{}
	hub := wsHub.NewHub(log)

	// Initialize handler
	handler := NewHandler(repo, hub, log, true, nil)

	if handler == nil {
		t.Fatal("Expected handler to be initialized, got nil")
	}

	if handler.log == nil {
		t.Error("Expected logger to be set")
	}

	if handler.repo == nil {
		t.Error("Expected repository to be set")
	}

	if handler.wsHub == nil {
		t.Error("Expected WebSocket hub to be set")
	}

	if handler.workflowService == nil {
		t.Error("Expected workflow service to be initialized")
	}
}

// TestHandlerUpgraderConfiguration verifies WebSocket upgrader settings
func TestHandlerUpgraderConfiguration(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := &mockRepository{}
	hub := wsHub.NewHub(log)

	handler := NewHandler(repo, hub, log, true, nil)

	// Verify upgrader allows cross-origin by default (needed for iframe embedding)
	if handler.upgrader.CheckOrigin == nil {
		t.Error("Expected CheckOrigin function to be set")
	}
}

func TestHandlerWebSocketOriginEnforcement(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := &mockRepository{}
	hub := wsHub.NewHub(log)

	allowed := []string{"http://allowed.local"}
	handler := NewHandler(repo, hub, log, false, allowed)

	goodReq := httptest.NewRequest(http.MethodGet, "/ws", nil)
	goodReq.Header.Set("Origin", "http://allowed.local")
	if !handler.upgrader.CheckOrigin(goodReq) {
		t.Fatal("expected allowed origin to pass CheckOrigin")
	}

	badReq := httptest.NewRequest(http.MethodGet, "/ws", nil)
	badReq.Header.Set("Origin", "http://evil.local")
	if handler.upgrader.CheckOrigin(badReq) {
		t.Fatal("expected unauthorized origin to be rejected")
	}
}

// TestHandlerDependencies verifies all critical dependencies are present
func TestHandlerDependencies(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	tests := []struct {
		name        string
		repo        database.Repository
		wsHub       *wsHub.Hub
		shouldPanic bool
	}{
		{
			name:        "all dependencies present",
			repo:        &mockRepository{},
			wsHub:       wsHub.NewHub(log),
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.shouldPanic {
					t.Errorf("Expected panic=%v, got panic=%v", tt.shouldPanic, r != nil)
				}
			}()

			handler := NewHandler(tt.repo, tt.wsHub, log, true, nil)
			if !tt.shouldPanic && handler == nil {
				t.Error("Expected handler to be created")
			}
		})
	}
}

func withWorkflowRouteContext(r *http.Request, param, value string) *http.Request {
	ctx := chi.RouteContext(r.Context())
	if ctx == nil {
		ctx = chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
	}
	ctx.URLParams.Add(param, value)
	return r
}

func TestListWorkflowVersionsHandlerSuccess(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handler := &Handler{
		workflowService: &workflowServiceMock{
			listWorkflowVersionsFn: func(_ context.Context, workflowID uuid.UUID, limit, offset int) ([]*workflow.WorkflowVersionSummary, error) {
				if limit != 50 || offset != 0 {
					t.Fatalf("unexpected pagination: limit=%d offset=%d", limit, offset)
				}
				return []*workflow.WorkflowVersionSummary{
					{
						Version:           3,
						WorkflowID:        workflowID,
						CreatedAt:         time.Unix(1731540000, 0),
						CreatedBy:         "autosave",
						ChangeDescription: "Autosave",
					},
				}, nil
			},
		},
		log: logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/id/versions", nil)
	workflowID := uuid.New()
	req = withWorkflowRouteContext(req, "id", workflowID.String())
	resp := httptest.NewRecorder()

	handler.ListWorkflowVersions(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var body struct {
		Versions []workflowVersionResponse `json:"versions"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Versions) != 1 || body.Versions[0].Version != 3 {
		t.Fatalf("unexpected versions payload: %+v", body.Versions)
	}
	if body.Versions[0].WorkflowID != workflowID {
		t.Fatalf("expected workflow id %s, got %s", workflowID, body.Versions[0].WorkflowID)
	}
}

func TestListWorkflowVersionsHandlerNotFound(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handler := &Handler{
		workflowService: &workflowServiceMock{
			listWorkflowVersionsFn: func(context.Context, uuid.UUID, int, int) ([]*workflow.WorkflowVersionSummary, error) {
				return nil, database.ErrNotFound
			},
		},
		log: logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/id/versions", nil)
	req = withWorkflowRouteContext(req, "id", uuid.New().String())
	resp := httptest.NewRecorder()

	handler.ListWorkflowVersions(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}
}

func TestGetWorkflowVersionHandlerSuccess(t *testing.T) {
	created := time.Now().UTC()
	workflowID := uuid.New()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handler := &Handler{
		workflowService: &workflowServiceMock{
			getWorkflowVersionFn: func(context.Context, uuid.UUID, int) (*workflow.WorkflowVersionSummary, error) {
				return &workflow.WorkflowVersionSummary{
					Version:    2,
					WorkflowID: workflowID,
					CreatedAt:  created,
				}, nil
			},
		},
		log: logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/id/versions/2", nil)
	req = withWorkflowRouteContext(req, "id", workflowID.String())
	req = withWorkflowRouteContext(req, "version", "2")
	resp := httptest.NewRecorder()

	handler.GetWorkflowVersion(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var response workflowVersionResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode workflow version: %v", err)
	}
	if response.Version != 2 || response.WorkflowID != workflowID {
		t.Fatalf("unexpected version payload: %+v", response)
	}
	if response.CreatedAt == "" {
		t.Fatalf("expected created_at to be populated")
	}
}

func TestRestoreWorkflowVersionConflict(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handler := &Handler{
		workflowService: &workflowServiceMock{
			getWorkflowVersionFn: func(context.Context, uuid.UUID, int) (*workflow.WorkflowVersionSummary, error) {
				return &workflow.WorkflowVersionSummary{Version: 3}, nil
			},
			restoreWorkflowVersionFn: func(context.Context, uuid.UUID, int, string) (*database.Workflow, error) {
				return nil, services.ErrWorkflowVersionConflict
			},
		},
		log: logger,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/id/versions/3/restore", nil)
	req = withWorkflowRouteContext(req, "id", uuid.New().String())
	req = withWorkflowRouteContext(req, "version", "3")
	resp := httptest.NewRecorder()

	handler.RestoreWorkflowVersion(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", resp.Code)
	}
}

func TestExecuteAdhocWorkflowSuccess(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-ADHOC-SUCCESS] executes adhoc workflow without persistence", func(t *testing.T) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)

		executionID := uuid.New()
		handler := &Handler{
			workflowService: &workflowServiceMock{
				executeAdhocWorkflowFn: func(ctx context.Context, flowDef map[string]any, params map[string]any, name string) (*database.Execution, error) {
					// Verify workflow name passed correctly
					if name != "test-workflow" {
						t.Errorf("expected workflow name 'test-workflow', got '%s'", name)
					}
					// Verify flow definition structure
					if flowDef == nil {
						t.Error("expected flow definition to be non-nil")
					}
					return &database.Execution{
						ID:              executionID,
						Status:          "pending",
						TriggerType:     "adhoc",
						WorkflowVersion: 0,
					}, nil
				},
			},
			log: logger,
		}

		reqBody := `{
			"flow_definition": {
				"nodes": [{"id": "1", "type": "navigate", "data": {"url": "https://example.com"}}],
				"edges": []
			},
			"parameters": {},
			"metadata": {"name": "test-workflow"}
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/execute-adhoc", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		handler.ExecuteAdhocWorkflow(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result["execution_id"] != executionID.String() {
			t.Errorf("expected execution_id %s, got %v", executionID, result["execution_id"])
		}

		if result["workflow_id"] != nil {
			t.Error("expected workflow_id to be null for adhoc execution")
		}

		if result["status"] != "pending" {
			t.Errorf("expected status 'pending', got %v", result["status"])
		}
	})
}

func TestExecuteAdhocWorkflowMissingFlowDefinition(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	handler := &Handler{
		workflowService: &workflowServiceMock{},
		log:             logger,
	}

	reqBody := `{"parameters": {}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/execute-adhoc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	handler.ExecuteAdhocWorkflow(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestExecuteAdhocWorkflowInvalidJSON(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	handler := &Handler{
		workflowService: &workflowServiceMock{},
		log:             logger,
	}

	reqBody := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/execute-adhoc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	handler.ExecuteAdhocWorkflow(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestExecuteAdhocWorkflowServiceError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	handler := &Handler{
		workflowService: &workflowServiceMock{
			executeAdhocWorkflowFn: func(ctx context.Context, flowDef map[string]any, params map[string]any, name string) (*database.Execution, error) {
				return nil, errors.New("workflow compilation failed")
			},
		},
		log: logger,
	}

	reqBody := `{
		"flow_definition": {
			"nodes": [{"id": "1", "type": "navigate"}],
			"edges": []
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/execute-adhoc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	handler.ExecuteAdhocWorkflow(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.Code)
	}
}
