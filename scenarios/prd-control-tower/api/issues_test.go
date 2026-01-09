package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func resetIssueCache() {
	scenarioIssuesCacheMu.Lock()
	scenarioIssuesCache = make(map[string]*scenarioIssuesCacheEntry)
	scenarioIssuesCacheMu.Unlock()
}

func withIssueTrackerServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse server url: %v", err)
	}
	port, _ := strconv.Atoi(parsed.Port())

	issueTrackerHTTPClient = srv.Client()
	issueTrackerAPIPortResolver = func(context.Context) (int, error) { return port, nil }
	issueTrackerUIPortResolver = func(context.Context) (int, error) { return port, nil }

	t.Cleanup(func() {
		issueTrackerHTTPClient = &http.Client{Timeout: 15 * time.Second}
		issueTrackerAPIPortResolver = locateIssueTrackerAPIPort
		issueTrackerUIPortResolver = locateIssueTrackerUIPort
	})
}

func TestHandleGetScenarioIssuesStatus_Success(t *testing.T) {
	resetIssueCache()

	withIssueTrackerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		_ = r.ParseForm()
		if got := r.FormValue("target_id"); got != "alpha" {
			t.Fatalf("expected target alpha, got %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"issues":[{"id":"issue-1","title":"Missing vision","status":"open","priority":"high","reporter":{"name":"QA","email":"qa@example.com"},"metadata":{"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}}],"count":1}}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/status?entity_type=scenario&entity_name=alpha", nil)
	rec := httptest.NewRecorder()

	handleGetScenarioIssuesStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var summary ScenarioIssuesSummary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if summary.TotalCount != 1 || summary.OpenCount != 1 {
		t.Fatalf("unexpected counts: %+v", summary)
	}
	if summary.Issues[0].IssueURL == "" {
		t.Fatal("expected issue URL to be populated")
	}
}

func TestHandleGetScenarioIssuesStatus_UsesCacheOnFailure(t *testing.T) {
	resetIssueCache()

	var hitCount atomic.Int32

	withIssueTrackerServer(t, func(w http.ResponseWriter, r *http.Request) {
		hitCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"issues":[],"count":0}}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/status?entity_type=scenario&entity_name=beta", nil)
	rec := httptest.NewRecorder()
	handleGetScenarioIssuesStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if hitCount.Load() != 1 {
		t.Fatalf("expected exactly one upstream call, got %d", hitCount.Load())
	}

	originalTTL := scenarioIssuesCacheTTL
	scenarioIssuesCacheTTL = time.Nanosecond
	t.Cleanup(func() { scenarioIssuesCacheTTL = originalTTL })

	// Force resolver failure to trigger stale cache path
	issueTrackerAPIPortResolver = func(context.Context) (int, error) {
		return 0, errors.New("port missing")
	}

	rec2 := httptest.NewRecorder()
	handleGetScenarioIssuesStatus(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for cached fallback, got %d", rec2.Code)
	}

	var summary ScenarioIssuesSummary
	if err := json.NewDecoder(rec2.Body).Decode(&summary); err != nil {
		t.Fatalf("failed to decode cached response: %v", err)
	}
	if !summary.FromCache || !summary.Stale {
		t.Fatalf("expected cached stale result, got %+v", summary)
	}
}

func TestHandleSubmitIssueReport_Success(t *testing.T) {
	resetIssueCache()

	withIssueTrackerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid payload: %v", err)
		}
		if payload["title"] != "Report quality issues" {
			t.Fatalf("unexpected title %v", payload["title"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"issue_id":"issue-xyz","issue":{"id":"issue-xyz","title":"X","status":"open","priority":"medium","reporter":{"name":"","email":""},"metadata":{"created_at":"","updated_at":""}}}}`))
	})

	payload := ScenarioIssueReportRequest{
		EntityType:  "scenario",
		EntityName:  "gamma",
		Source:      "quality_scanner",
		Title:       "Report quality issues",
		Description: "details",
		Selections: []IssueReportSelection{
			{ID: "missing-section", Title: "Missing section", Detail: "Vision", Category: "structure", Severity: "high"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/report", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleSubmitIssueReport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response IssueReportResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.IssueID != "issue-xyz" {
		t.Fatalf("unexpected issue id %s", response.IssueID)
	}
}

func TestHandleSubmitIssueReport_Validation(t *testing.T) {
	resetIssueCache()

	payload := ScenarioIssueReportRequest{
		EntityType:  "scenario",
		EntityName:  "delta",
		Title:       "Incomplete",
		Description: "",
		Selections:  nil,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/report", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handleSubmitIssueReport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected validation error, got %d", rec.Code)
	}
}
