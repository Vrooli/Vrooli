package httpserver

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/applicability"

	"github.com/gorilla/mux"
)

func TestHandleListPhases(t *testing.T) {
	tmp := t.TempDir()
	orchestratorSvc, err := orchestrator.NewSuiteOrchestrator(tmp)
	if err != nil {
		t.Fatalf("failed to initialize orchestrator: %v", err)
	}
	server := &Server{
		config:       Config{Port: "0", ServiceName: "Test Genie API"},
		router:       mux.NewRouter(),
		phaseCatalog: orchestratorSvc,
		logger:       log.New(io.Discard, "", 0),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phases", nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Items []orchestrator.PhaseDescriptor `json:"items"`
		Count int                            `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Count == 0 || len(payload.Items) == 0 {
		t.Fatalf("expected phase descriptors in response")
	}
}

func TestHandleInspectPhase(t *testing.T) {
	tmp := t.TempDir()
	orchestratorSvc, err := orchestrator.NewSuiteOrchestrator(tmp)
	if err != nil {
		t.Fatalf("failed to initialize orchestrator: %v", err)
	}
	server := &Server{
		config:       Config{Port: "0", ServiceName: "Test Genie API"},
		router:       mux.NewRouter(),
		phaseCatalog: orchestratorSvc,
		logger:       log.New(io.Discard, "", 0),
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phases/unit", nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider":"unit-health"`) {
		t.Fatalf("expected provider metadata in response, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"policy"`) || !strings.Contains(rec.Body.String(), `"runnability"`) {
		t.Fatalf("expected policy and runnability metadata in response, got %s", rec.Body.String())
	}
}

func TestHandlePreviewPhaseApplicabilityFiltersPhase(t *testing.T) {
	planner := &fakeExecutionPlanner{
		result: &execution.ExecutionPlanPreview{
			ScenarioName: "demo",
			Phases: []execution.PlannedPhase{{
				Name:                "unit",
				ApplicabilityStatus: applicability.StatusApplies,
			}},
			NotApplicablePhases: []execution.PlannedPhase{{
				Name:                "search",
				ApplicabilityStatus: applicability.StatusNotApplicable,
				ApplicabilityReasons: []applicability.Reason{{
					Code:    "applicability.default_not_applicable",
					Message: "phase does not apply by default",
				}},
			}},
		},
	}
	server := &Server{
		config:           Config{Port: "0", ServiceName: "Test Genie API"},
		executionPlanner: planner,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phases/applicability?target=demo&phase=search", nil)
	rec := httptest.NewRecorder()
	server.handlePreviewPhaseApplicability(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if planner.lastRequest.ScenarioName != "demo" || len(planner.lastRequest.Phases) != 0 {
		t.Fatalf("expected full target preview without explicit phase selection, got %#v", planner.lastRequest)
	}
	if !strings.Contains(rec.Body.String(), `"status":"not_applicable"`) {
		t.Fatalf("expected not applicable status, got %s", rec.Body.String())
	}
}
