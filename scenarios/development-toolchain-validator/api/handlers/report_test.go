package handlers

import (
	"context"
	"development-toolchain-validator/domain/expectation"
	"development-toolchain-validator/domain/report"
	"development-toolchain-validator/domain/skill"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// --- mocks (handler-level) ---

type stubConnectionLister struct {
	conns []*skill.Connection
}

func (s *stubConnectionLister) List(_ context.Context, _ skill.ListOptions) ([]*skill.Connection, error) {
	return s.conns, nil
}

type stubExpectationLister struct{}

func (s *stubExpectationLister) ListStructural(_ context.Context, _ expectation.ListOptions) ([]*expectation.StructuralExpectation, error) {
	return nil, nil
}

func (s *stubExpectationLister) ListCLI(_ context.Context, _ expectation.ListOptions) ([]*expectation.CLIAssertion, error) {
	return nil, nil
}

type stubResultReader struct{}

func (s *stubResultReader) CLIResultsByReference(_ context.Context, _ string) ([]*report.CLIResultRow, error) {
	return nil, nil
}

func newTestReportHandler() *ReportHandler {
	svc := report.NewService(
		&stubConnectionLister{},
		&stubExpectationLister{},
		&stubResultReader{},
	)
	return NewReportHandler(svc)
}

func TestReportHandler_Conflicts(t *testing.T) {
	h := newTestReportHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/reports/conflicts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp report.ConflictsReport
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.TotalCount != 0 {
		t.Errorf("expected 0 conflicts, got %d", resp.TotalCount)
	}
}

func TestReportHandler_Drift(t *testing.T) {
	h := newTestReportHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	body := `{"current_hashes": {"skill-a": "hash1"}}`
	req := httptest.NewRequest("POST", "/api/v1/reports/drift", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp report.DriftReport
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
}

func TestReportHandler_Drift_MissingHashes(t *testing.T) {
	h := newTestReportHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/reports/drift", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportHandler_Maturity(t *testing.T) {
	h := newTestReportHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/reports/maturity", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp report.MaturityReport
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
}

func TestReportHandler_ToolBaselines(t *testing.T) {
	h := newTestReportHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/reports/tool-baselines", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp report.ToolBaselinesReport
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
}

func TestReportHandler_ConflictsWithFilter(t *testing.T) {
	h := newTestReportHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/v1/reports/conflicts?reference_id=ref1&skill_id=skill-a", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReportHandler_Drift_InvalidBody(t *testing.T) {
	h := newTestReportHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/api/v1/reports/drift", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
