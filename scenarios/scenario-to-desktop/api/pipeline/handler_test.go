package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"scenario-to-desktop-api/shared/errors"
)

// mockOrchestrator implements Orchestrator for testing
type mockOrchestrator struct {
	runResult       *Status
	runError        error
	getResult       *Status
	getFound        bool
	cancelSuccess   bool
	pipelines       []*Status
	resumeResult    *Status
	resumeError     error
	idleResult      *Status
	idleError       error
	startResult     *Status
	startError      error
}

func (m *mockOrchestrator) RunPipeline(ctx context.Context, config *Config) (*Status, error) {
	if m.runError != nil {
		return nil, m.runError
	}
	return m.runResult, nil
}

func (m *mockOrchestrator) CreateIdlePipeline(config *Config) (*Status, error) {
	if m.idleError != nil {
		return nil, m.idleError
	}
	return m.idleResult, nil
}

func (m *mockOrchestrator) StartPipeline(ctx context.Context, pipelineID string) (*Status, error) {
	if m.startError != nil {
		return nil, m.startError
	}
	return m.startResult, nil
}

func (m *mockOrchestrator) GetStatus(pipelineID string) (*Status, bool) {
	return m.getResult, m.getFound
}

func (m *mockOrchestrator) CancelPipeline(pipelineID string) bool {
	return m.cancelSuccess
}

func (m *mockOrchestrator) ListPipelines() []*Status {
	return m.pipelines
}

func (m *mockOrchestrator) ResumePipeline(ctx context.Context, pipelineID string, config *Config) (*Status, error) {
	if m.resumeError != nil {
		return nil, m.resumeError
	}
	return m.resumeResult, nil
}

func TestNewHandler(t *testing.T) {
	h := NewHandler()
	if h == nil {
		t.Fatal("expected handler to be created")
	}
	if h.basePath != "/api/v1/pipeline" {
		t.Errorf("expected default basePath, got %q", h.basePath)
	}
}

func TestWithOrchestrator(t *testing.T) {
	orch := &mockOrchestrator{}
	h := NewHandler(WithOrchestrator(orch))
	if h.orchestrator != orch {
		t.Error("expected orchestrator to be set")
	}
}

func TestWithBasePath(t *testing.T) {
	h := NewHandler(WithBasePath("/custom/path"))
	if h.basePath != "/custom/path" {
		t.Errorf("expected custom basePath, got %q", h.basePath)
	}
}

func TestRegisterRoutes(t *testing.T) {
	h := NewHandler()
	router := mux.NewRouter()

	h.RegisterRoutes(router)

	// Verify routes are registered
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/pipeline/run"},
		{http.MethodGet, "/api/v1/pipeline/test-id"},
		{http.MethodPost, "/api/v1/pipeline/test-id/cancel"},
		{http.MethodGet, "/api/v1/pipelines"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		match := &mux.RouteMatch{}
		if !router.Match(req, match) {
			t.Errorf("expected route %s %s to be registered", tt.method, tt.path)
		}
	}
}

func TestHandleRun(t *testing.T) {
	t.Run("orchestrator not configured", func(t *testing.T) {
		h := NewHandler()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/run", nil)
		rr := httptest.NewRecorder()

		h.handleRun(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		h := NewHandler(WithOrchestrator(&mockOrchestrator{}))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/run", bytes.NewBufferString("not json"))
		rr := httptest.NewRecorder()

		h.handleRun(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("missing scenario_name", func(t *testing.T) {
		h := NewHandler(WithOrchestrator(&mockOrchestrator{}))
		body, _ := json.Marshal(Config{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/run", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.handleRun(rr, req)

		// 422 Unprocessable Entity is the correct status for validation errors (missing required fields)
		// This is more semantically correct than 400 Bad Request which indicates malformed request syntax
		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", rr.Code)
		}
	})

	t.Run("orchestrator error", func(t *testing.T) {
		orch := &mockOrchestrator{runError: context.DeadlineExceeded}
		h := NewHandler(WithOrchestrator(orch))
		body, _ := json.Marshal(Config{ScenarioName: "test"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/run", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.handleRun(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("successful run", func(t *testing.T) {
		orch := &mockOrchestrator{
			runResult: &Status{PipelineID: "pipeline-123", Status: StatusRunning},
		}
		h := NewHandler(WithOrchestrator(orch), WithBasePath("/api/v1/pipeline"))
		body, _ := json.Marshal(Config{ScenarioName: "test-scenario"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/run", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.handleRun(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", rr.Code)
		}

		var resp RunResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.PipelineID != "pipeline-123" {
			t.Errorf("expected pipeline_id 'pipeline-123', got %q", resp.PipelineID)
		}
		if resp.StatusURL != "/api/v1/pipeline/pipeline-123" {
			t.Errorf("expected status URL to include pipeline ID, got %q", resp.StatusURL)
		}
	})
}

func TestHandleGetStatus(t *testing.T) {
	router := mux.NewRouter()

	t.Run("orchestrator not configured", func(t *testing.T) {
		h := NewHandler()
		router.HandleFunc("/api/v1/pipeline/{id}", h.handleGetStatus).Methods("GET")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/test-id", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("pipeline not found", func(t *testing.T) {
		orch := &mockOrchestrator{getFound: false}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}", h.handleGetStatus).Methods("GET")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/nonexistent", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("pipeline found", func(t *testing.T) {
		orch := &mockOrchestrator{
			getResult: &Status{PipelineID: "found-123", Status: StatusRunning},
			getFound:  true,
		}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}", h.handleGetStatus).Methods("GET")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/found-123", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}

func TestHandleCancel(t *testing.T) {
	t.Run("orchestrator not configured", func(t *testing.T) {
		h := NewHandler()
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/cancel", h.handleCancel).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/test-id/cancel", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("cancel successful", func(t *testing.T) {
		orch := &mockOrchestrator{cancelSuccess: true}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/cancel", h.handleCancel).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/running-123/cancel", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("pipeline not found", func(t *testing.T) {
		orch := &mockOrchestrator{
			cancelSuccess: false,
			getFound:      false,
		}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/cancel", h.handleCancel).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/nonexistent/cancel", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("pipeline already completed", func(t *testing.T) {
		orch := &mockOrchestrator{
			cancelSuccess: false,
			getResult:     &Status{PipelineID: "done-123", Status: StatusCompleted},
			getFound:      true,
		}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/cancel", h.handleCancel).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/done-123/cancel", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp CancelResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Message != "Pipeline has already completed" {
			t.Errorf("expected completed message, got %q", resp.Message)
		}
	})
}

func TestHandleResume(t *testing.T) {
	t.Run("orchestrator not configured", func(t *testing.T) {
		h := NewHandler()
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/resume", h.handleResume).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/test-id/resume", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("successful resume", func(t *testing.T) {
		orch := &mockOrchestrator{
			resumeResult: &Status{
				PipelineID: "resumed-123",
				Status:     StatusRunning,
				Config: &Config{
					ScenarioName:     "test",
					ResumeFromStage:  "generate",
					ParentPipelineID: "original-123",
				},
			},
		}
		h := NewHandler(WithOrchestrator(orch), WithBasePath("/api/v1/pipeline"))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/resume", h.handleResume).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/original-123/resume", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", rr.Code)
		}

		var resp ResumeResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.PipelineID != "resumed-123" {
			t.Errorf("expected pipeline_id 'resumed-123', got %q", resp.PipelineID)
		}
		if resp.ParentPipelineID != "original-123" {
			t.Errorf("expected parent_pipeline_id 'original-123', got %q", resp.ParentPipelineID)
		}
		if resp.ResumeFromStage != "generate" {
			t.Errorf("expected resume_from_stage 'generate', got %q", resp.ResumeFromStage)
		}
		if resp.StatusURL != "/api/v1/pipeline/resumed-123" {
			t.Errorf("expected status URL '/api/v1/pipeline/resumed-123', got %q", resp.StatusURL)
		}
	})

	t.Run("pipeline not found", func(t *testing.T) {
		orch := &mockOrchestrator{
			// Return a proper domain error for "not found" which maps to HTTP 404
			resumeError: errors.ErrPipelineNotFound("nonexistent"),
		}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/resume", h.handleResume).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/nonexistent/resume", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("invalid resume state", func(t *testing.T) {
		orch := &mockOrchestrator{
			// Return a proper domain error for "not resumable" which maps to HTTP 400
			resumeError: errors.ErrPipelineNotResumable("running-123", "pipeline is currently running"),
		}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/resume", h.handleResume).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/running-123/resume", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("with config override", func(t *testing.T) {
		orch := &mockOrchestrator{
			resumeResult: &Status{
				PipelineID: "resumed-456",
				Status:     StatusRunning,
				Config: &Config{
					ScenarioName:     "test",
					ResumeFromStage:  "generate",
					StopAfterStage:   "build",
					ParentPipelineID: "original-456",
				},
			},
		}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/resume", h.handleResume).Methods("POST")

		body, _ := json.Marshal(Config{StopAfterStage: "build"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/original-456/resume", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", rr.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		orch := &mockOrchestrator{}
		h := NewHandler(WithOrchestrator(orch))
		r := mux.NewRouter()
		r.HandleFunc("/api/v1/pipeline/{id}/resume", h.handleResume).Methods("POST")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/test-123/resume", bytes.NewBufferString("not json"))
		req.ContentLength = 8 // Set content length to indicate body is present
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestHandleList(t *testing.T) {
	t.Run("orchestrator not configured", func(t *testing.T) {
		h := NewHandler()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
		rr := httptest.NewRecorder()

		h.handleList(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		orch := &mockOrchestrator{pipelines: []*Status{}}
		h := NewHandler(WithOrchestrator(orch))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
		rr := httptest.NewRecorder()

		h.handleList(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp ListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Pipelines) != 0 {
			t.Errorf("expected 0 pipelines, got %d", len(resp.Pipelines))
		}
	})

	t.Run("with pipelines", func(t *testing.T) {
		orch := &mockOrchestrator{
			pipelines: []*Status{
				{PipelineID: "p1", Status: StatusRunning},
				{PipelineID: "p2", Status: StatusCompleted},
			},
		}
		h := NewHandler(WithOrchestrator(orch))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
		rr := httptest.NewRecorder()

		h.handleList(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp ListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Pipelines) != 2 {
			t.Errorf("expected 2 pipelines, got %d", len(resp.Pipelines))
		}
	})
}

// NOTE: TestWriteJSON and TestWriteError were removed as the methods were
// refactored into shared/http/response.go (WriteJSON, WriteError). Those
// utilities have their own tests in shared/http/response_test.go.
