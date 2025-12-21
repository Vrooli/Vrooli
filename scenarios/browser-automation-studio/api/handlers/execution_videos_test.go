package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

type executionServiceStub struct {
	videos []workflow.ExecutionVideoArtifact
	err    error
}

func (s *executionServiceStub) ExecuteWorkflow(context.Context, uuid.UUID, map[string]any) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) ExecuteWorkflowAPI(context.Context, *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) ExecuteWorkflowAPIWithOptions(context.Context, *basapi.ExecuteWorkflowRequest, *workflow.ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) ExecuteAdhocWorkflowAPI(context.Context, *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) ExecuteAdhocWorkflowAPIWithOptions(context.Context, *basexecution.ExecuteAdhocRequest, *workflow.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) StopExecution(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}
func (s *executionServiceStub) ResumeExecution(context.Context, uuid.UUID, map[string]any) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) ListExecutions(context.Context, *uuid.UUID, int, int) ([]*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) GetExecution(context.Context, uuid.UUID) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) UpdateExecution(context.Context, *database.ExecutionIndex) error {
	return errors.New("not implemented")
}
func (s *executionServiceStub) GetExecutionScreenshots(context.Context, uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) GetExecutionVideoArtifacts(context.Context, uuid.UUID) ([]workflow.ExecutionVideoArtifact, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.videos, nil
}
func (s *executionServiceStub) GetExecutionTraceArtifacts(context.Context, uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) GetExecutionHarArtifacts(context.Context, uuid.UUID) ([]workflow.ExecutionFileArtifact, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) HydrateExecutionProto(context.Context, *database.ExecutionIndex) (*basexecution.Execution, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) GetExecutionTimeline(context.Context, uuid.UUID) (*workflow.ExecutionTimeline, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) GetExecutionTimelineProto(context.Context, uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) DescribeExecutionExport(context.Context, uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	return nil, errors.New("not implemented")
}
func (s *executionServiceStub) ExportToFolder(context.Context, uuid.UUID, string, storage.StorageInterface) error {
	return errors.New("not implemented")
}

func withRouteParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

type apiErrorPayload struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func TestGetExecutionRecordedVideos_Success(t *testing.T) {
	executionID := uuid.New()
	handler := &Handler{
		executionService: &executionServiceStub{
			videos: []workflow.ExecutionVideoArtifact{
				{
					ArtifactID:  "video-1",
					StorageURL:  "/api/v1/artifacts/sample.webm",
					ContentType: "video/webm",
					Label:       "video-1",
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-videos", nil)
	req = withRouteParam(req, "id", executionID.String())
	rec := httptest.NewRecorder()

	handler.GetExecutionRecordedVideos(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload executionVideosResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.ExecutionID != executionID.String() {
		t.Fatalf("expected execution_id %s, got %s", executionID.String(), payload.ExecutionID)
	}
	if len(payload.Videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(payload.Videos))
	}
}

func TestGetExecutionRecordedVideos_NotFound(t *testing.T) {
	executionID := uuid.New()
	handler := &Handler{
		executionService: &executionServiceStub{
			err: database.ErrNotFound,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/recorded-videos", nil)
	req = withRouteParam(req, "id", executionID.String())
	rec := httptest.NewRecorder()

	handler.GetExecutionRecordedVideos(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var payload apiErrorPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if payload.Code != ErrExecutionNotFound.Code {
		t.Fatalf("expected error code %s, got %s", ErrExecutionNotFound.Code, payload.Code)
	}
}

func TestPostExecutionExport_WebmReplayFramesRejected(t *testing.T) {
	handler := &Handler{}
	executionID := uuid.New()

	body := []byte(`{"format":"webm","render_source":"replay_frames"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withRouteParam(req, "id", executionID.String())
	rec := httptest.NewRecorder()

	handler.PostExecutionExport(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var payload apiErrorPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if payload.Code != ErrInvalidRequest.Code {
		t.Fatalf("expected error code %s, got %s", ErrInvalidRequest.Code, payload.Code)
	}
	if payload.Details["error"] != "webm format is only available for recorded video exports" {
		t.Fatalf("unexpected error detail: %#v", payload.Details["error"])
	}
}
