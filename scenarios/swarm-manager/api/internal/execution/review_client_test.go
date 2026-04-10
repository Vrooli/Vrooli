package execution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTriggerReview_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/review/run" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req ReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ScenarioName != "web-console" {
			t.Fatalf("expected scenarioName web-console, got %s", req.ScenarioName)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(reviewRunResponse{JobID: "job-123"})
	}))
	defer server.Close()

	client := &HTTPReviewClient{httpClient: server.Client()}
	// Override discovery by using direct URL — wrap in a test-friendly way
	jobID, err := triggerReviewDirect(client, server.URL, context.Background(), ReviewRequest{
		ScenarioName: "web-console",
	})
	if err != nil {
		t.Fatalf("TriggerReview error: %v", err)
	}
	if jobID != "job-123" {
		t.Fatalf("expected job-123, got %s", jobID)
	}
}

func TestTriggerReview_ReusesExistingJobOnConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(reviewRunConflictResponse{
			Error: "a review run is already in progress for this scenario",
			JobID: "job-existing",
		})
	}))
	defer server.Close()

	client := &HTTPReviewClient{httpClient: server.Client()}
	jobID, err := client.triggerReviewAtBaseURL(context.Background(), server.URL, ReviewRequest{
		ScenarioName: "web-console",
	})
	if err != nil {
		t.Fatalf("TriggerReview error: %v", err)
	}
	if jobID != "job-existing" {
		t.Fatalf("expected existing job id, got %s", jobID)
	}
}

func TestPollReview_Running(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/review/run/job-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(reviewJobStatus{
			JobID:  "job-1",
			Status: "running",
		})
	}))
	defer server.Close()

	client := &HTTPReviewClient{httpClient: server.Client()}
	result, done, err := pollReviewDirect(client, server.URL, context.Background(), "job-1")
	if err != nil {
		t.Fatalf("PollReview error: %v", err)
	}
	if done {
		t.Fatal("expected not done")
	}
	if result != nil {
		t.Fatal("expected nil result for running job")
	}
}

func TestPollReview_Completed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(reviewJobStatus{
			JobID:  "job-2",
			Status: "completed",
			Summary: &reviewSummaryResponse{
				ScenarioName: "web-console",
				Readiness:    "green",
				Dimensions:   json.RawMessage(`{"tests":{"available":true,"passed":true,"total":5,"passedCount":5,"failedCount":0}}`),
				Timestamp:    "2026-03-24T10:00:00Z",
			},
		})
	}))
	defer server.Close()

	client := &HTTPReviewClient{httpClient: server.Client()}
	result, done, err := pollReviewDirect(client, server.URL, context.Background(), "job-2")
	if err != nil {
		t.Fatalf("PollReview error: %v", err)
	}
	if !done {
		t.Fatal("expected done")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Classification != "ready" {
		t.Fatalf("expected classification ready, got %s", result.Classification)
	}
	if result.JobID != "job-2" {
		t.Fatalf("expected job-2, got %s", result.JobID)
	}
}

func TestPollReview_Failed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(reviewJobStatus{
			JobID:  "job-3",
			Status: "failed",
			Error:  "timeout",
		})
	}))
	defer server.Close()

	client := &HTTPReviewClient{httpClient: server.Client()}
	result, done, err := pollReviewDirect(client, server.URL, context.Background(), "job-3")
	if err != nil {
		t.Fatalf("PollReview error: %v", err)
	}
	if !done {
		t.Fatal("expected done")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Classification != "not_assessable" {
		t.Fatalf("expected not_assessable, got %s", result.Classification)
	}
}

// triggerReviewDirect calls the review trigger against a known base URL, bypassing discovery.
func triggerReviewDirect(c *HTTPReviewClient, baseURL string, ctx context.Context, req ReviewRequest) (string, error) {
	return c.triggerReviewAtBaseURL(ctx, baseURL, req)
}

// pollReviewDirect calls the review poll against a known base URL, bypassing discovery.
func pollReviewDirect(c *HTTPReviewClient, baseURL string, ctx context.Context, jobID string) (*ReviewResult, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/review/run/"+jobID, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	var job reviewJobStatus
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, false, err
	}
	return mapJobToResult(job)
}

func TestPing_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("expected HEAD, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &HTTPReviewClient{httpClient: server.Client()}
	// Bypass discovery by calling Ping with a context that won't resolve.
	// Instead, use pingDirect helper.
	err := pingDirect(client, server.URL, context.Background())
	if err != nil {
		t.Fatalf("Ping should succeed: %v", err)
	}
}

func TestPing_Unavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := &HTTPReviewClient{httpClient: server.Client()}
	err := pingDirect(client, server.URL, context.Background())
	if err == nil {
		t.Fatal("Ping should fail for 503")
	}
}

// pingDirect calls the health endpoint against a known base URL, bypassing discovery.
func pingDirect(c *HTTPReviewClient, baseURL string, ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, baseURL+"/api/v1/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return http.ErrAbortHandler
	}
	return nil
}

func TestMapReadinessToClassification(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"green", "ready"},
		{"yellow", "ready_with_notes"},
		{"red", "needs_work"},
		{"", "not_assessable"},
		{"unknown", "not_assessable"},
	}
	for _, tc := range tests {
		got := mapReadinessToClassification(tc.input)
		if got != tc.want {
			t.Errorf("mapReadinessToClassification(%q): got %s, want %s", tc.input, got, tc.want)
		}
	}
}
