package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTestExecutionList_Empty(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.OK(&TestExecutionListResponse{
		Items: []TestExecutionResult{},
		Count: 0,
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestHandleTestExecution_TestGenieUnavailable(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.ServiceUnavailable("test-genie is not available")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestHandleTestExecutionDetail_NotFound(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.NotFound("execution not found")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}
