package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// withURLParams adds multiple URL parameters to a request for chi routing
func withURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ============================================================================
// Mock UX Metrics Service
// ============================================================================

type MockUXMetricsService struct {
	GetExecutionMetricsError   error
	ComputeAndSaveMetricsError error
	GetWorkflowAggregateError  error

	ExecutionMetrics  *contracts.ExecutionMetrics
	WorkflowAggregate *contracts.WorkflowMetricsAggregate
	AnalyzeStepError  error
	AnalyzeStepResult *contracts.StepMetrics

	analyzer  *MockUXAnalyzer
	collector *MockUXCollector
}

type MockUXAnalyzer struct {
	AnalyzeStepError      error
	AnalyzeStepResult     *contracts.StepMetrics
	AnalyzeExecutionError error
	AnalyzeExecutionResult *contracts.ExecutionMetrics
}

type MockUXCollector struct {
	OnStepOutcomeError error
	OnCursorUpdateError error
	FlushExecutionError error
}

func (m *MockUXCollector) OnStepOutcome(ctx context.Context, executionID uuid.UUID, outcome uxmetrics.StepOutcomeData) error {
	return m.OnStepOutcomeError
}

func (m *MockUXCollector) OnCursorUpdate(ctx context.Context, executionID uuid.UUID, stepIndex int, point contracts.TimedPoint) error {
	return m.OnCursorUpdateError
}

func (m *MockUXCollector) FlushExecution(ctx context.Context, executionID uuid.UUID) error {
	return m.FlushExecutionError
}

func (m *MockUXAnalyzer) AnalyzeExecution(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	if m.AnalyzeExecutionError != nil {
		return nil, m.AnalyzeExecutionError
	}
	return m.AnalyzeExecutionResult, nil
}

func (m *MockUXAnalyzer) AnalyzeStep(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error) {
	if m.AnalyzeStepError != nil {
		return nil, m.AnalyzeStepError
	}
	return m.AnalyzeStepResult, nil
}

func NewMockUXMetricsService() *MockUXMetricsService {
	return &MockUXMetricsService{
		analyzer:  &MockUXAnalyzer{},
		collector: &MockUXCollector{},
	}
}

func (m *MockUXMetricsService) GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	if m.GetExecutionMetricsError != nil {
		return nil, m.GetExecutionMetricsError
	}
	return m.ExecutionMetrics, nil
}

func (m *MockUXMetricsService) ComputeAndSaveMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	if m.ComputeAndSaveMetricsError != nil {
		return nil, m.ComputeAndSaveMetricsError
	}
	return m.ExecutionMetrics, nil
}

func (m *MockUXMetricsService) GetWorkflowAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*contracts.WorkflowMetricsAggregate, error) {
	if m.GetWorkflowAggregateError != nil {
		return nil, m.GetWorkflowAggregateError
	}
	return m.WorkflowAggregate, nil
}

func (m *MockUXMetricsService) Analyzer() uxmetrics.Analyzer {
	return m.analyzer
}

func (m *MockUXMetricsService) Collector() uxmetrics.Collector {
	return m.collector
}

// Compile-time interface check
var _ uxmetrics.Service = (*MockUXMetricsService)(nil)

// createUXMetricsHandler creates a UXMetricsHandler with mock service for testing
func createUXMetricsHandler() (*UXMetricsHandler, *MockUXMetricsService) {
	mockService := NewMockUXMetricsService()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := NewUXMetricsHandler(mockService, log)
	return handler, mockService
}

// ============================================================================
// GetExecutionMetrics Tests
// ============================================================================

func TestGetExecutionMetrics_InvalidExecutionID(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/invalid-uuid/ux-metrics", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetExecutionMetrics(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetExecutionMetrics_Success(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.ExecutionMetrics = &contracts.ExecutionMetrics{
		ExecutionID:     executionID,
		TotalDurationMs: 1000,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/ux-metrics", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetExecutionMetrics_NotFound(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.ExecutionMetrics = nil // No metrics found

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/ux-metrics", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionMetrics(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetExecutionMetrics_ServiceError(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.GetExecutionMetricsError = errors.New("service unavailable")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/ux-metrics", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.GetExecutionMetrics(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// GetStepMetrics Tests
// ============================================================================

func TestGetStepMetrics_InvalidExecutionID(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/invalid-uuid/ux-metrics/steps/0", nil)
	req = withURLParams(req, map[string]string{"id": "invalid-uuid", "stepIndex": "0"})
	rr := httptest.NewRecorder()

	handler.GetStepMetrics(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetStepMetrics_InvalidStepIndex(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/ux-metrics/steps/invalid", nil)
	req = withURLParams(req, map[string]string{"id": executionID.String(), "stepIndex": "invalid"})
	rr := httptest.NewRecorder()

	handler.GetStepMetrics(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetStepMetrics_Success(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.analyzer.AnalyzeStepResult = &contracts.StepMetrics{
		StepIndex: 0,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/ux-metrics/steps/0", nil)
	req = withURLParams(req, map[string]string{"id": executionID.String(), "stepIndex": "0"})
	rr := httptest.NewRecorder()

	handler.GetStepMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetStepMetrics_NotFound(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.analyzer.AnalyzeStepResult = nil

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/ux-metrics/steps/0", nil)
	req = withURLParams(req, map[string]string{"id": executionID.String(), "stepIndex": "0"})
	rr := httptest.NewRecorder()

	handler.GetStepMetrics(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetStepMetrics_AnalysisError(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.analyzer.AnalyzeStepError = errors.New("analysis failed")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String()+"/ux-metrics/steps/0", nil)
	req = withURLParams(req, map[string]string{"id": executionID.String(), "stepIndex": "0"})
	rr := httptest.NewRecorder()

	handler.GetStepMetrics(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// ComputeMetrics Tests
// ============================================================================

func TestComputeMetrics_InvalidExecutionID(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/invalid-uuid/ux-metrics/compute", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.ComputeMetrics(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestComputeMetrics_Success(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.ExecutionMetrics = &contracts.ExecutionMetrics{
		ExecutionID:     executionID,
		TotalDurationMs: 2000,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/ux-metrics/compute", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.ComputeMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestComputeMetrics_ServiceError(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	executionID := uuid.New()
	mockService.ComputeAndSaveMetricsError = errors.New("computation failed")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+executionID.String()+"/ux-metrics/compute", nil)
	req = withURLParam(req, "id", executionID.String())
	rr := httptest.NewRecorder()

	handler.ComputeMetrics(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// GetWorkflowMetricsAggregate Tests
// ============================================================================

func TestGetWorkflowMetricsAggregate_InvalidWorkflowID(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/invalid-uuid/ux-metrics/aggregate", nil)
	req = withURLParam(req, "id", "invalid-uuid")
	rr := httptest.NewRecorder()

	handler.GetWorkflowMetricsAggregate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetWorkflowMetricsAggregate_Success(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	workflowID := uuid.New()
	mockService.WorkflowAggregate = &contracts.WorkflowMetricsAggregate{
		WorkflowID:     workflowID,
		ExecutionCount: 10,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/ux-metrics/aggregate", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflowMetricsAggregate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetWorkflowMetricsAggregate_WithLimit(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	workflowID := uuid.New()
	mockService.WorkflowAggregate = &contracts.WorkflowMetricsAggregate{
		WorkflowID: workflowID,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/ux-metrics/aggregate?limit=50", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflowMetricsAggregate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestGetWorkflowMetricsAggregate_InvalidLimit(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	workflowID := uuid.New()
	mockService.WorkflowAggregate = &contracts.WorkflowMetricsAggregate{
		WorkflowID: workflowID,
	}

	// Invalid limit should be ignored, defaulting to 10
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/ux-metrics/aggregate?limit=invalid", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflowMetricsAggregate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 (invalid limit ignored), got %d", rr.Code)
	}
}

func TestGetWorkflowMetricsAggregate_LimitTooHigh(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	workflowID := uuid.New()
	mockService.WorkflowAggregate = &contracts.WorkflowMetricsAggregate{
		WorkflowID: workflowID,
	}

	// Limit > 100 should be capped or ignored
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/ux-metrics/aggregate?limit=200", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflowMetricsAggregate(rr, req)

	// Should still work, limit will be capped at 100 or use default
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestGetWorkflowMetricsAggregate_ServiceError(t *testing.T) {
	handler, mockService := createUXMetricsHandler()

	workflowID := uuid.New()
	mockService.GetWorkflowAggregateError = errors.New("aggregation failed")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID.String()+"/ux-metrics/aggregate", nil)
	req = withURLParam(req, "id", workflowID.String())
	rr := httptest.NewRecorder()

	handler.GetWorkflowMetricsAggregate(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

// ============================================================================
// Helper Method Tests
// ============================================================================

func TestUXMetricsHandler_ExtractUUID(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	tests := []struct {
		name      string
		param     string
		expectErr bool
	}{
		{"valid uuid", uuid.NewString(), false},
		{"invalid uuid", "not-a-uuid", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = withURLParam(req, "id", tt.param)

			_, err := handler.extractUUID(req, "id")
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got error: %v", tt.expectErr, err)
			}
		})
	}
}

func TestUXMetricsHandler_JsonOK(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	rr := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	handler.jsonOK(rr, data)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response["key"] != "value" {
		t.Errorf("expected key=value, got %v", response)
	}
}

func TestUXMetricsHandler_JsonError(t *testing.T) {
	handler, _ := createUXMetricsHandler()

	rr := httptest.NewRecorder()

	handler.jsonError(rr, http.StatusBadRequest, "test error")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response["error"] != "test error" {
		t.Errorf("expected error='test error', got %v", response)
	}
}

func TestNewUXMetricsHandler(t *testing.T) {
	mockService := NewMockUXMetricsService()
	log := logrus.New()

	handler := NewUXMetricsHandler(mockService, log)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if handler.service == nil {
		t.Error("expected service to be set")
	}
	if handler.log == nil {
		t.Error("expected log to be set")
	}
}
