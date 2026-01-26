package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/services/deepsearch"
)

type fakeDeepSearchService struct {
	job *deepsearch.DeepSearchJob
	err error
}

func (f *fakeDeepSearchService) StartSearch(_ context.Context, _ deepsearch.DeepSearchRequest) (*deepsearch.DeepSearchJob, error) {
	return f.job, f.err
}

func (f *fakeDeepSearchService) GetJob(_ context.Context, _ string) (*deepsearch.DeepSearchJob, error) {
	return f.job, f.err
}

func TestHandleDocsSearchDeep(t *testing.T) {
	startedAt := time.Now().UTC()
	service := &fakeDeepSearchService{
		job: &deepsearch.DeepSearchJob{
			JobID:     "job-1",
			Status:    deepsearch.StatusRunning,
			StartedAt: &startedAt,
		},
	}
	srv := &Server{docDeepSearchService: service}

	body, _ := json.Marshal(DeepSearchRequest{Query: "docs", Scope: "global"})
	req := httptest.NewRequest("POST", "/api/v1/docs/search/deep", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleDocsSearchDeep(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded DeepSearchJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.JobID != "job-1" {
		t.Fatalf("expected job id, got %s", decoded.JobID)
	}
}

func TestHandleDocsSearchDeepStatus(t *testing.T) {
	service := &fakeDeepSearchService{
		job: &deepsearch.DeepSearchJob{
			JobID:  "job-2",
			Status: deepsearch.StatusCompleted,
			Results: []deepsearch.DeepSearchResult{
				{Path: "docs/README.md", Relevance: 0.9, Summary: "Readme"},
			},
		},
	}
	srv := &Server{docDeepSearchService: service}

	req := httptest.NewRequest("GET", "/api/v1/docs/search/deep/job-2", nil)
	req = mux.SetURLVars(req, map[string]string{"job_id": "job-2"})
	rec := httptest.NewRecorder()

	srv.handleDocsSearchDeepStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded DeepSearchJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.Status != deepsearch.StatusCompleted {
		t.Fatalf("expected completed status, got %s", decoded.Status)
	}
}
