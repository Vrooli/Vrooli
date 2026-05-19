package httpserver

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
)

type fakeExecutionPlanner struct {
	lastRequest orchestrator.SuiteExecutionRequest
	result      *execution.ExecutionPlanPreview
	err         error
}

func (f *fakeExecutionPlanner) Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error) {
	f.lastRequest = req
	return f.result, f.err
}

func TestServer_handlePreviewExecutionPlan(t *testing.T) {
	planner := &fakeExecutionPlanner{
		result: &execution.ExecutionPlanPreview{
			ScenarioName: "demo",
			PresetUsed:   "quick",
			Phases: []execution.PlannedPhase{
				{
					Name:                     "unit",
					EstimatedDurationSeconds: 42,
					TimeoutSeconds:           900,
					EstimateSource:           execution.EstimateSourceScenarioHistory,
					EstimateConfidence:       execution.EstimateConfidenceMedium,
					EstimateSampleSize:       6,
				},
			},
			Summary: execution.ExecutionPlanSummary{
				PhaseCount:               1,
				EstimatedDurationSeconds: 42,
				TimeoutSeconds:           900,
			},
		},
	}
	srv := &Server{
		config:           Config{Port: "0", ServiceName: "Test Genie API"},
		router:           nil,
		executionPlanner: planner,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/plan", strings.NewReader(`{"scenarioName":"demo","preset":"quick","failFast":true}`))
	w := httptest.NewRecorder()

	srv.handlePreviewExecutionPlan(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if planner.lastRequest.ScenarioName != "demo" || planner.lastRequest.Preset != "quick" || !planner.lastRequest.FailFast {
		t.Fatalf("expected request to propagate, got %#v", planner.lastRequest)
	}
	if !strings.Contains(w.Body.String(), `"estimatedDurationSeconds":42`) {
		t.Fatalf("expected estimate in response, got %s", w.Body.String())
	}
}

func TestServer_handlePreviewExecutionPlanRejectsInvalidPayload(t *testing.T) {
	srv := &Server{
		config:           Config{Port: "0", ServiceName: "Test Genie API"},
		executionPlanner: &fakeExecutionPlanner{},
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/plan", strings.NewReader(`{"scenarioName":" "}`))
	w := httptest.NewRecorder()

	srv.handlePreviewExecutionPlan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
