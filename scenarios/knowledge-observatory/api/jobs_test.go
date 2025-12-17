package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/ports"
)

type fakeJobStore struct {
	enqueued ports.DocumentIngestJobRequest
	jobID    string
	status   ports.JobStatus
	found    bool
}

func (f *fakeJobStore) EnqueueDocumentIngest(ctx context.Context, req ports.DocumentIngestJobRequest) (string, error) {
	f.enqueued = req
	if f.jobID == "" {
		f.jobID = "job-1"
	}
	return f.jobID, nil
}
func (f *fakeJobStore) GetJob(ctx context.Context, jobID string) (ports.JobStatus, bool, error) {
	return f.status, f.found, nil
}
func (f *fakeJobStore) ClaimNextPendingJob(ctx context.Context) (ports.DocumentIngestJob, bool, error) {
	return ports.DocumentIngestJob{}, false, nil
}
func (f *fakeJobStore) UpdateJobProgress(ctx context.Context, jobID string, completedChunks, totalChunks int) error {
	return nil
}
func (f *fakeJobStore) CompleteJob(ctx context.Context, jobID string, status string, errorMessage string) error {
	return nil
}

func TestHandleCreateIngestJob(t *testing.T) {
	js := &fakeJobStore{jobID: "job-123"}
	srv := &Server{jobStore: js}

	body := map[string]interface{}{
		"namespace": "ecosystem-manager",
		"content":   "hello world",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/ingest/jobs", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	srv.handleCreateIngestJob(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var decoded CreateIngestJobResponse
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.JobID != "job-123" {
		t.Fatalf("job_id=%q", decoded.JobID)
	}
	if decoded.Status != "pending" {
		t.Fatalf("status=%q", decoded.Status)
	}
	if js.enqueued.Namespace != "ecosystem-manager" || js.enqueued.Content != "hello world" {
		t.Fatalf("enqueued=%+v", js.enqueued)
	}
}

func TestHandleGetIngestJobNotFound(t *testing.T) {
	js := &fakeJobStore{found: false}
	srv := &Server{jobStore: js}

	req := httptest.NewRequest("GET", "/api/v1/ingest/jobs/job-1", nil)
	req = mux.SetURLVars(req, map[string]string{"job_id": "job-1"})
	rec := httptest.NewRecorder()

	srv.handleGetIngestJob(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetIngestJobFound(t *testing.T) {
	js := &fakeJobStore{
		found: true,
		status: ports.JobStatus{
			JobID:           "job-1",
			Status:          "running",
			TotalChunks:     10,
			CompletedChunks: 3,
		},
	}
	srv := &Server{jobStore: js}

	req := httptest.NewRequest("GET", "/api/v1/ingest/jobs/job-1", nil)
	req = mux.SetURLVars(req, map[string]string{"job_id": "job-1"})
	rec := httptest.NewRecorder()

	srv.handleGetIngestJob(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

