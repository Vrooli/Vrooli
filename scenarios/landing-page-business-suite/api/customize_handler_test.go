package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	contenthttp "landing-page-business-suite-api/handlers/content"
	"landing-page-business-suite-api/internal/testutil"
)

func TestHandleCustomizeQueuesJob(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customize", bytes.NewBufferString(`{
		"scenario_id":"landing-page-business-suite",
		"brief":"Improve pricing copy",
		"assets":["logo.svg"],
		"preview":true
	}`))
	recorder := httptest.NewRecorder()

	contenthttp.Customize(time.Now)(recorder, req)

	testutil.RequireHTTPStatus(t, recorder, http.StatusAccepted)
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var response struct {
		JobID   string `json:"job_id"`
		Status  string `json:"status"`
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.JobID == "" || response.Status != "queued" || response.AgentID == "" {
		t.Fatalf("unexpected queue response: %#v", response)
	}
}

func TestHandleCustomizeRejectsMalformedRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customize", bytes.NewBufferString(`{"scenario_id":`))
	recorder := httptest.NewRecorder()

	contenthttp.Customize(time.Now)(recorder, req)

	testutil.RequireHTTPStatus(t, recorder, http.StatusBadRequest)
}
