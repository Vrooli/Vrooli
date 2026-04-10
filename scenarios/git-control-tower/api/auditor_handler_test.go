package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAuditorRunCheck_Unavailable(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("scenario-auditor is not available")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleAuditorRunCheck_MissingScenarioName(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenario_name is required")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAuditorJobStatus_Unavailable(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("scenario-auditor is not available")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleAuditorRules_Unavailable(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("scenario-auditor is not available")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleAuditorFix_Unavailable(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("scenario-auditor is not available")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleAuditorFix_MissingScenarioNames(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenario_names is required")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAuditorViolations_Unavailable(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("scenario-auditor is not available")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleAuditorViolations_MissingScenarioName(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenarioName query parameter is required")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
