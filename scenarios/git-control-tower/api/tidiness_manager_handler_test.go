package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTidinessScore_Unavailable(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("tidiness-manager is not available")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleTidinessIssues_MissingScenarioName(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenarioName query parameter is required")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleTidinessStaleness_Unavailable(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("tidiness-manager is not available")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleTidinessLightScan_Unavailable(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("tidiness-manager is not available")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleTidinessScenarioDetail_MissingScenarioName(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenarioName query parameter is required")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
