package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/services/dochealing"
)

type fakeDocHealingService struct {
	job        *dochealing.HealJob
	err        error
	lastReq    dochealing.HealRequest
	autoResult *dochealing.AutoFixResult
	autoErr    error
}

func (f *fakeDocHealingService) StartHealing(_ context.Context, req dochealing.HealRequest) (*dochealing.HealJob, error) {
	f.lastReq = req
	return f.job, f.err
}

func (f *fakeDocHealingService) GetJob(_ context.Context, _ string) (*dochealing.HealJob, error) {
	return f.job, f.err
}

func (f *fakeDocHealingService) ApproveJob(_ context.Context, _ string, _ string) (*dochealing.HealJob, error) {
	return f.job, f.err
}

func (f *fakeDocHealingService) RejectJob(_ context.Context, _ string, _ string, _ string) (*dochealing.HealJob, error) {
	return f.job, f.err
}

func (f *fakeDocHealingService) AutoFix(_ context.Context, _ string, _ bool) (*dochealing.AutoFixResult, error) {
	return f.autoResult, f.autoErr
}

func TestHandleDocsHeal(t *testing.T) {
	service := &fakeDocHealingService{
		job: &dochealing.HealJob{
			JobID:        "job-1",
			ScenarioName: "alpha",
			Status:       dochealing.StatusRunning,
		},
	}
	srv := &Server{docHealingService: service}

	body, _ := json.Marshal(DocHealRequest{AutoApprove: true})
	req := httptest.NewRequest("POST", "/api/v1/scenarios/alpha/docs/heal", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	srv.handleDocsHeal(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if service.lastReq.ScenarioName != "alpha" {
		t.Fatalf("expected scenario alpha, got %s", service.lastReq.ScenarioName)
	}
	var decoded DocHealJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.JobID != "job-1" {
		t.Fatalf("expected job id, got %s", decoded.JobID)
	}
}

func TestHandleDocsHealStatus(t *testing.T) {
	service := &fakeDocHealingService{
		job: &dochealing.HealJob{
			JobID:        "job-2",
			ScenarioName: "beta",
			Status:       dochealing.StatusNeedsReview,
		},
	}
	srv := &Server{docHealingService: service}

	req := httptest.NewRequest("GET", "/api/v1/docs/heal/job-2", nil)
	req = mux.SetURLVars(req, map[string]string{"job_id": "job-2"})
	rec := httptest.NewRecorder()

	srv.handleDocsHealStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded DocHealJobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.Status != dochealing.StatusNeedsReview {
		t.Fatalf("expected needs_review status, got %s", decoded.Status)
	}
}

func TestHandleDocsHealApproveReject(t *testing.T) {
	service := &fakeDocHealingService{
		job: &dochealing.HealJob{
			JobID:        "job-3",
			ScenarioName: "gamma",
			Status:       dochealing.StatusApproved,
		},
	}
	srv := &Server{docHealingService: service}

	approveReq := httptest.NewRequest("POST", "/api/v1/docs/heal/job-3/approve", nil)
	approveReq = mux.SetURLVars(approveReq, map[string]string{"job_id": "job-3"})
	approveRec := httptest.NewRecorder()
	srv.handleDocsHealApprove(approveRec, approveReq)

	if approveRec.Code != http.StatusOK {
		t.Fatalf("expected approve status 200, got %d", approveRec.Code)
	}

	rejectReq := httptest.NewRequest("POST", "/api/v1/docs/heal/job-3/reject", nil)
	rejectReq = mux.SetURLVars(rejectReq, map[string]string{"job_id": "job-3"})
	rejectRec := httptest.NewRecorder()
	srv.handleDocsHealReject(rejectRec, rejectReq)

	if rejectRec.Code != http.StatusOK {
		t.Fatalf("expected reject status 200, got %d", rejectRec.Code)
	}
}

func TestHandleDocsAutoFix(t *testing.T) {
	before := 0.6
	after := 1.0
	service := &fakeDocHealingService{
		autoResult: &dochealing.AutoFixResult{
			ScenarioName: "delta",
			Moved: []dochealing.MovedDoc{
				{FromPath: "ARCHITECTURE.md", ToPath: "docs/concepts/ARCHITECTURE.md", DocType: "architecture"},
			},
			HealthBefore: before,
			HealthAfter:  after,
		},
	}
	srv := &Server{docHealingService: service}

	body, _ := json.Marshal(DocAutoFixRequest{DryRun: false})
	req := httptest.NewRequest("POST", "/api/v1/scenarios/delta/docs/autofix", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "delta"})
	rec := httptest.NewRecorder()

	srv.handleDocsAutoFix(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded DocAutoFixResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.ScenarioName != "delta" {
		t.Fatalf("expected scenario delta, got %s", decoded.ScenarioName)
	}
	if len(decoded.Moved) != 1 {
		t.Fatalf("expected 1 moved file, got %d", len(decoded.Moved))
	}
	if decoded.HealthBefore != before {
		t.Errorf("expected health_before %.1f, got %.1f", before, decoded.HealthBefore)
	}
	if decoded.HealthAfter != after {
		t.Errorf("expected health_after %.1f, got %.1f", after, decoded.HealthAfter)
	}
}

func TestHandleDocsAutoFix_ServiceUnavailable(t *testing.T) {
	srv := &Server{docHealingService: nil}

	req := httptest.NewRequest("POST", "/api/v1/scenarios/delta/docs/autofix", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "delta"})
	rec := httptest.NewRecorder()

	srv.handleDocsAutoFix(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
}
