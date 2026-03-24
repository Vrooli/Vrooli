package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCalculateReadiness_AllGreen(t *testing.T) {
	t.Parallel()
	dims := ReviewDimensions{
		Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
		Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
		CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
	}
	got := CalculateReadiness(dims)
	if got != ReadinessGreen {
		t.Errorf("expected green, got %s", got)
	}
}

func TestCalculateReadiness_Yellow(t *testing.T) {
	t.Parallel()
	// Only screenshots available → yellow
	dims := ReviewDimensions{
		Visual: &VisualDimension{Available: true, ScreenshotCount: 1},
	}
	got := CalculateReadiness(dims)
	if got != ReadinessYellow {
		t.Errorf("expected yellow, got %s", got)
	}
}

func TestCalculateReadiness_Red(t *testing.T) {
	t.Parallel()
	dims := ReviewDimensions{}
	got := CalculateReadiness(dims)
	if got != ReadinessRed {
		t.Errorf("expected red, got %s", got)
	}
}

func TestCalculateReadiness_MissingDimensions(t *testing.T) {
	t.Parallel()
	// Tests exist but fail, no screenshots, quality stale → should be yellow (hasTests is true)
	dims := ReviewDimensions{
		Tests:       &TestsDimension{Available: true, Total: 2, Passed: false, PassedCount: 1, FailedCount: 1},
		CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: true},
	}
	got := CalculateReadiness(dims)
	if got != ReadinessYellow {
		t.Errorf("expected yellow, got %s", got)
	}
}

func TestCalculateReadiness_QualityBelowThreshold(t *testing.T) {
	t.Parallel()
	// Everything present but quality score < 60 → yellow (not green)
	dims := ReviewDimensions{
		Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
		Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
		CodeQuality: &CodeQualityDimension{Available: true, Score: 50, Stale: false},
	}
	got := CalculateReadiness(dims)
	if got != ReadinessYellow {
		t.Errorf("expected yellow, got %s", got)
	}
}

func TestHandleReviewSummary_MissingScenarioName(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenarioName query parameter is required")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReviewRun_MissingScenarioName(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenarioName is required")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReviewJobStatus_NotFound(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.NotFound("review job not found")
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestReviewJobStore_Lifecycle(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()

	// Create
	job := store.Create("job-1", []string{"tidiness", "tests"})
	if job.Status != "running" {
		t.Errorf("expected running, got %s", job.Status)
	}
	if job.Checks["tidiness"] != CheckPending {
		t.Errorf("expected pending, got %s", job.Checks["tidiness"])
	}

	// Get
	got, ok := store.Get("job-1")
	if !ok {
		t.Fatal("expected job to be found")
	}
	if got.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", got.JobID)
	}

	// UpdateCheck
	store.UpdateCheck("job-1", "tidiness", CheckRunning)
	got, _ = store.Get("job-1")
	if got.Checks["tidiness"] != CheckRunning {
		t.Errorf("expected running, got %s", got.Checks["tidiness"])
	}

	// Complete
	summary := &ReviewSummaryResponse{ScenarioName: "test", Readiness: ReadinessGreen}
	store.Complete("job-1", summary)
	got, _ = store.Get("job-1")
	if got.Status != "completed" {
		t.Errorf("expected completed, got %s", got.Status)
	}
	if got.Summary == nil || got.Summary.Readiness != ReadinessGreen {
		t.Error("expected green readiness in summary")
	}

	// Fail a different job
	store.Create("job-2", []string{"rules"})
	store.Fail("job-2", "something went wrong")
	got, _ = store.Get("job-2")
	if got.Status != "failed" {
		t.Errorf("expected failed, got %s", got.Status)
	}
	if got.Error != "something went wrong" {
		t.Errorf("expected error message, got %s", got.Error)
	}

	// Not found
	_, ok = store.Get("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent job")
	}
}
