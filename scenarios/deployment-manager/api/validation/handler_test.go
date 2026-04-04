package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// inMemoryRepo implements Repository for tests.
type inMemoryRepo struct {
	records map[string]*Record
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{records: make(map[string]*Record)}
}

func (r *inMemoryRepo) Create(_ context.Context, rec *Record) error {
	r.records[rec.ID] = rec
	return nil
}

func (r *inMemoryRepo) Get(_ context.Context, id string) (*Record, error) {
	rec, ok := r.records[id]
	if !ok {
		return nil, nil
	}
	return rec, nil
}

func (r *inMemoryRepo) ListByProfile(_ context.Context, profileID string) ([]*Record, error) {
	var out []*Record
	for _, rec := range r.records {
		if rec.ProfileID == profileID {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (r *inMemoryRepo) UpdateStatus(_ context.Context, id, status string) error {
	if rec, ok := r.records[id]; ok {
		rec.Status = status
	}
	return nil
}

func (r *inMemoryRepo) UpdateReview(_ context.Context, id, decision, reviewer, notes string) error {
	if rec, ok := r.records[id]; ok {
		rec.ReviewDecision = decision
		rec.ReviewedBy = reviewer
		rec.ReviewNotes = notes
		now := time.Now()
		rec.ReviewedAt = &now
	}
	return nil
}

func (r *inMemoryRepo) UpdateVideo(_ context.Context, id, videoPath string, sizeBytes, durationMs int64) error {
	if rec, ok := r.records[id]; ok {
		rec.VideoURL = videoPath
		rec.VideoSizeBytes = sizeBytes
		rec.VideoDurationMs = durationMs
	}
	return nil
}

func noopLog(_ string, _ map[string]interface{}) {}

func TestCreate(t *testing.T) {
	repo := newInMemoryRepo()
	h := NewHandlerForTest(repo, "/tmp/videos", noopLog)

	body, _ := json.Marshal(Request{ProfileID: "prof-1", GitCommitHash: "abc123", RecordVideo: true})
	req := httptest.NewRequest("POST", "/api/v1/validations", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var rec Record
	if err := json.NewDecoder(w.Body).Decode(&rec); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if rec.ProfileID != "prof-1" {
		t.Fatalf("expected profile_id prof-1, got %s", rec.ProfileID)
	}
	if rec.GitCommitHash != "abc123" {
		t.Fatalf("expected git_commit_hash abc123, got %s", rec.GitCommitHash)
	}
	if rec.Status != "pending" {
		t.Fatalf("expected status pending, got %s", rec.Status)
	}
}

func TestCreate_MissingProfileID(t *testing.T) {
	h := NewHandlerForTest(newInMemoryRepo(), "/tmp", noopLog)

	body, _ := json.Marshal(Request{GitCommitHash: "abc123"})
	req := httptest.NewRequest("POST", "/api/v1/validations", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_MissingCommitHash(t *testing.T) {
	h := NewHandlerForTest(newInMemoryRepo(), "/tmp", noopLog)

	body, _ := json.Marshal(Request{ProfileID: "prof-1"})
	req := httptest.NewRequest("POST", "/api/v1/validations", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGet(t *testing.T) {
	repo := newInMemoryRepo()
	repo.records["val-1"] = &Record{ID: "val-1", ProfileID: "p1", Status: "pending", CreatedAt: time.Now()}
	h := NewHandlerForTest(repo, "/tmp", noopLog)

	req := httptest.NewRequest("GET", "/api/v1/validations/val-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "val-1"})
	w := httptest.NewRecorder()

	h.Get(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	h := NewHandlerForTest(newInMemoryRepo(), "/tmp", noopLog)

	req := httptest.NewRequest("GET", "/api/v1/validations/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()

	h.Get(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSubmitReview(t *testing.T) {
	repo := newInMemoryRepo()
	repo.records["val-1"] = &Record{ID: "val-1", ProfileID: "p1", Status: "review_required", CreatedAt: time.Now()}
	h := NewHandlerForTest(repo, "/tmp", noopLog)

	body, _ := json.Marshal(ReviewRequest{Decision: "approved", Reviewer: "admin"})
	req := httptest.NewRequest("POST", "/api/v1/validations/val-1/review", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "val-1"})
	w := httptest.NewRecorder()

	h.SubmitReview(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if repo.records["val-1"].ReviewDecision != "approved" {
		t.Fatal("expected review decision to be approved")
	}

	var resp ReviewResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %s", resp.Status)
	}
	if resp.Decision != "approved" {
		t.Fatalf("expected decision approved, got %s", resp.Decision)
	}
}

func TestSubmitReview_DefaultsReviewerToOperator(t *testing.T) {
	repo := newInMemoryRepo()
	repo.records["val-1"] = &Record{ID: "val-1", ProfileID: "p1", Status: "review_required", CreatedAt: time.Now()}
	h := NewHandlerForTest(repo, "/tmp", noopLog)

	// Send review with no reviewer field.
	body, _ := json.Marshal(ReviewRequest{Decision: "approved"})
	req := httptest.NewRequest("POST", "/api/v1/validations/val-1/review", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "val-1"})
	w := httptest.NewRecorder()

	h.SubmitReview(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if repo.records["val-1"].ReviewedBy != "operator" {
		t.Fatalf("expected reviewed_by 'operator', got %q", repo.records["val-1"].ReviewedBy)
	}
}

func TestSubmitReview_NoBridgingWithoutCommitHash(t *testing.T) {
	repo := newInMemoryRepo()
	// Record has no GitCommitHash (legacy).
	repo.records["val-1"] = &Record{ID: "val-1", ProfileID: "p1", Status: "review_required", CreatedAt: time.Now()}
	h := NewHandlerForTest(repo, "/tmp", noopLog)

	body, _ := json.Marshal(ReviewRequest{Decision: "approved", Reviewer: "admin"})
	req := httptest.NewRequest("POST", "/api/v1/validations/val-1/review", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "val-1"})
	w := httptest.NewRecorder()

	h.SubmitReview(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReviewResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ApprovalID != "" {
		t.Fatalf("expected empty approval_id for legacy validation, got %q", resp.ApprovalID)
	}
	if resp.ApprovalStatus != "" {
		t.Fatalf("expected empty approval_status for legacy validation, got %q", resp.ApprovalStatus)
	}
}

func TestSubmitReview_NotFound(t *testing.T) {
	h := NewHandlerForTest(newInMemoryRepo(), "/tmp", noopLog)

	body, _ := json.Marshal(ReviewRequest{Decision: "approved"})
	req := httptest.NewRequest("POST", "/api/v1/validations/missing/review", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()

	h.SubmitReview(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSubmitReview_InvalidDecision(t *testing.T) {
	h := NewHandlerForTest(newInMemoryRepo(), "/tmp", noopLog)

	body, _ := json.Marshal(ReviewRequest{Decision: "maybe"})
	req := httptest.NewRequest("POST", "/api/v1/validations/val-1/review", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "val-1"})
	w := httptest.NewRecorder()

	h.SubmitReview(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListByProfile(t *testing.T) {
	repo := newInMemoryRepo()
	repo.records["v1"] = &Record{ID: "v1", ProfileID: "p1", Status: "passed", CreatedAt: time.Now()}
	repo.records["v2"] = &Record{ID: "v2", ProfileID: "p1", Status: "failed", CreatedAt: time.Now()}
	repo.records["v3"] = &Record{ID: "v3", ProfileID: "p2", Status: "pending", CreatedAt: time.Now()}
	h := NewHandlerForTest(repo, "/tmp", noopLog)

	req := httptest.NewRequest("GET", "/api/v1/profiles/p1/validations", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "p1"})
	w := httptest.NewRecorder()

	h.ListByProfile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []*Record
	if err := json.NewDecoder(w.Body).Decode(&records); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records for p1, got %d", len(records))
	}
}
