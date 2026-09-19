package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCohortReportRejectsUnboundedSelection(t *testing.T) {
	h := &Handler{}
	rr := httptest.NewRecorder()
	h.GetCohortReport(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs/cohort-report", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCohortReportRoutePrecedesRunIDRoute(t *testing.T) {
	_, router := setupTestHandler(t)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs/cohort-report", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
