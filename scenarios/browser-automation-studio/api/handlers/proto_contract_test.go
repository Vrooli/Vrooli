package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
)

// protoContractWorkflowService is a lightweight WorkflowService stub for proto contract tests.
type protoContractWorkflowService struct {
	execution   *database.Execution
	executions  []database.Execution
	timeline    *export.ExecutionTimeline
	screenshots []*database.Screenshot
	previewFn   func(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error)
}

var _ WorkflowService = (*protoContractWorkflowService)(nil)

func (s *protoContractWorkflowService) GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error) {
	return s.execution, nil
}

func (s *protoContractWorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	result := make([]*database.Execution, 0, len(s.executions))
	for i := range s.executions {
		result = append(result, &s.executions[i])
	}
	return result, nil
}

func (s *protoContractWorkflowService) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*export.ExecutionTimeline, error) {
	return s.timeline, nil
}

func (s *protoContractWorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return s.screenshots, nil
}

func (s *protoContractWorkflowService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	if s.previewFn != nil {
		return s.previewFn(ctx, executionID)
	}
	return nil, errors.New("not implemented")
}

// Unused WorkflowService methods.
func (s *protoContractWorkflowService) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*workflow.WorkflowVersionSummary, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*workflow.WorkflowVersionSummary, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	return errors.New("not implemented")
}
func (s *protoContractWorkflowService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	return errors.New("not implemented")
}
func (s *protoContractWorkflowService) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) CreateProject(ctx context.Context, project *database.Project) error {
	return errors.New("not implemented")
}
func (s *protoContractWorkflowService) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) GetProject(ctx context.Context, projectID uuid.UUID) (*database.Project, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) UpdateProject(ctx context.Context, project *database.Project) error {
	return errors.New("not implemented")
}
func (s *protoContractWorkflowService) DeleteProject(ctx context.Context, projectID uuid.UUID) error {
	return errors.New("not implemented")
}
func (s *protoContractWorkflowService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *protoContractWorkflowService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return errors.New("not implemented")
}
func (s *protoContractWorkflowService) CheckAutomationHealth(ctx context.Context) (bool, error) {
	return false, errors.New("not implemented")
}

func newHandlerForProtoContract(t *testing.T, svc *protoContractWorkflowService) *Handler {
	t.Helper()
	return &Handler{
		workflowService: svc,
		log:             logrus.New(),
	}
}

func addRouteParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestGetExecutionProtoJSON(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()
	handler := newHandlerForProtoContract(t, &protoContractWorkflowService{
		execution: &database.Execution{
			ID:          execID,
			WorkflowID:  workflowID,
			Status:      "running",
			TriggerType: "manual",
			Progress:    50,
			StartedAt:   time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	})

	req := addRouteParam(httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+execID.String(), nil), "id", execID.String())
	rr := httptest.NewRecorder()
	handler.GetExecution(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["workflow_id"] != workflowID.String() {
		t.Fatalf("expected workflow_id %s, got %v", workflowID.String(), body["workflow_id"])
	}
	if _, ok := body["workflowId"]; ok {
		t.Fatalf("did not expect camelCase workflowId, got %v", body["workflowId"])
	}
}

func TestListExecutionsProtoJSON(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()
	handler := newHandlerForProtoContract(t, &protoContractWorkflowService{
		executions: []database.Execution{
			{
				ID:          execID,
				WorkflowID:  workflowID,
				Status:      "completed",
				TriggerType: "manual",
				Progress:    100,
				StartedAt:   time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	rr := httptest.NewRecorder()
	handler.ListExecutions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	executions, ok := body["executions"].([]any)
	if !ok || len(executions) != 1 {
		t.Fatalf("expected executions array length 1, got %v", body["executions"])
	}
	execObj, ok := executions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected execution object, got %T", executions[0])
	}
	if execObj["workflow_id"] != workflowID.String() {
		t.Fatalf("expected workflow_id %s, got %v", workflowID.String(), execObj["workflow_id"])
	}
	if _, ok := execObj["workflowId"]; ok {
		t.Fatalf("did not expect camelCase workflowId")
	}
}

func TestGetExecutionTimelineProtoJSON(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()
	handler := newHandlerForProtoContract(t, &protoContractWorkflowService{
		timeline: &export.ExecutionTimeline{
			ExecutionID: execID,
			WorkflowID:  workflowID,
			Status:      "completed",
			Progress:    100,
			StartedAt:   time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC),
			Frames: []export.TimelineFrame{
				{
					StepIndex: 1,
					NodeID:    "node-1",
					StepType:  "navigate",
					Status:    "completed",
					Success:   true,
				},
			},
			Logs: []export.TimelineLog{{
				ID:        "log-1",
				Level:     "info",
				Message:   "ok",
				Timestamp: time.Date(2024, 12, 1, 12, 0, 1, 0, time.UTC),
			}},
		},
	})

	req := addRouteParam(httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+execID.String()+"/timeline", nil), "id", execID.String())
	rr := httptest.NewRecorder()
	handler.GetExecutionTimeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	frames, ok := body["frames"].([]any)
	if !ok || len(frames) != 1 {
		t.Fatalf("expected frames array length 1, got %v", body["frames"])
	}
	frameObj, ok := frames[0].(map[string]any)
	if !ok {
		t.Fatalf("expected frame object, got %T", frames[0])
	}
	if _, ok := frameObj["step_index"]; !ok {
		t.Fatalf("expected step_index field in timeline frame")
	}
	if _, ok := frameObj["stepIndex"]; ok {
		t.Fatalf("did not expect camelCase stepIndex field")
	}
}

func TestDescribeExecutionExportProtoJSON(t *testing.T) {
	execID := uuid.New()
	specID := "spec-123"
	handler := newHandlerForProtoContract(t, &protoContractWorkflowService{
		execution: &database.Execution{ID: execID},
		timeline: &export.ExecutionTimeline{
			ExecutionID: execID,
			Status:      "completed",
			Progress:    100,
			StartedAt:   time.Now(),
		},
	})

	body := strings.NewReader(`{}`)
	req := addRouteParam(httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+execID.String()+"/export", body), "id", execID.String())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Inject a fake preview via the workflowService mock.
	handler.workflowService = &protoContractWorkflowService{
		execution: &database.Execution{ID: execID},
		timeline:  handler.workflowService.(*protoContractWorkflowService).timeline,
		previewFn: func(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
			return &workflow.ExecutionExportPreview{
				ExecutionID:         execID,
				SpecID:              specID,
				Status:              "ready",
				Message:             "ready",
				CapturedFrameCount:  1,
				AvailableAssetCount: 1,
				TotalDurationMs:     1000,
			}, nil
		},
	}

	handler.PostExecutionExport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var bodyMap map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &bodyMap); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if bodyMap["spec_id"] != specID {
		t.Fatalf("expected spec_id %s, got %v", specID, bodyMap["spec_id"])
	}
	if _, ok := bodyMap["specId"]; ok {
		t.Fatalf("did not expect camelCase specId")
	}
}

func TestGetExecutionScreenshotsProtoJSON(t *testing.T) {
	execID := uuid.New()
	handler := newHandlerForProtoContract(t, &protoContractWorkflowService{
		screenshots: []*database.Screenshot{
			{
				ID:           uuid.New(),
				ExecutionID:  execID,
				StepName:     "navigate",
				Timestamp:    time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC),
				StorageURL:   "https://example.com/full.png",
				ThumbnailURL: "",
				Width:        800,
				Height:       600,
				SizeBytes:    1024,
				Metadata: database.JSONMap{
					"step_index":    1,
					"thumbnail_url": "https://example.com/thumb.png",
				},
			},
		},
	})

	req := addRouteParam(httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+execID.String()+"/screenshots", nil), "id", execID.String())
	rr := httptest.NewRecorder()
	handler.GetExecutionScreenshots(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	shots, ok := body["screenshots"].([]any)
	if !ok || len(shots) != 1 {
		t.Fatalf("expected screenshots array length 1, got %v", body["screenshots"])
	}
	shotObj, ok := shots[0].(map[string]any)
	if !ok {
		t.Fatalf("expected screenshot object, got %T", shots[0])
	}
	if shotObj["storage_url"] != "https://example.com/full.png" {
		t.Fatalf("expected storage_url, got %v", shotObj["storage_url"])
	}
	if _, ok := shotObj["storageUrl"]; ok {
		t.Fatalf("did not expect camelCase storageUrl")
	}
	if shotObj["step_index"] != float64(1) { // JSON numbers unmarshal to float64
		t.Fatalf("expected step_index 1, got %v", shotObj["step_index"])
	}
}
