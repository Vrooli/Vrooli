package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// stubReviewInspector lets tests control the live-review guard.
type stubReviewInspector struct{ live bool }

func (s stubReviewInspector) HasLiveReviewRound(_, _ string) bool { return s.live }

func recoverReq(t *testing.T, h *Handler, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/"+name+"/recover-review", nil)
	} else {
		r = httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/"+name+"/recover-review", bytes.NewReader([]byte(body)))
	}
	r = mux.SetURLVars(r, map[string]string{"kind": "execute", "name": name})
	rec := httptest.NewRecorder()
	h.RecoverReview(rec, r)
	return rec
}

func TestRecoverReview_InReviewToReviewPending(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	h.SetReviewRoundInspector(stubReviewInspector{live: false})
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "orphan", Title: "Orphan", Status: StatusInReview, Kind: KindExecute,
	})

	rec := recoverReq(t, h, "orphan", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := h.store.LoadItem(KindExecute, "orphan")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if updated.Status != StatusReviewPending {
		t.Fatalf("status = %q, want %q", updated.Status, StatusReviewPending)
	}

	// Audit record should exist.
	decisionsDir := filepath.Join(rootDir, backlogKindDirs[KindExecute], "orphan", "review", "decisions")
	entries, err := os.ReadDir(decisionsDir)
	if err != nil {
		t.Fatalf("read decisions dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-recover.json") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a -recover.json audit record, got %v", entries)
	}
}

func TestRecoverReview_InReviewToBacklog(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	h.SetReviewRoundInspector(stubReviewInspector{live: false})
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "never-built", Title: "Never Built", Status: StatusInReview, Kind: KindExecute,
	})

	rec := recoverReq(t, h, "never-built", `{"to":"backlog","rationale":"never started"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	updated, _ := h.store.LoadItem(KindExecute, "never-built")
	if updated.Status != StatusBacklog {
		t.Fatalf("status = %q, want %q", updated.Status, StatusBacklog)
	}
}

func TestRecoverReview_ActiveRound_409(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	h.SetReviewRoundInspector(stubReviewInspector{live: true}) // a real review is in flight
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "in-flight", Title: "In Flight", Status: StatusInReview, Kind: KindExecute,
	})

	rec := recoverReq(t, h, "in-flight", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	// Status must be untouched.
	updated, _ := h.store.LoadItem(KindExecute, "in-flight")
	if updated.Status != StatusInReview {
		t.Fatalf("status changed to %q; a live review must not be short-circuited", updated.Status)
	}
}

func TestRecoverReview_NonReviewStatus_400(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	h.SetReviewRoundInspector(stubReviewInspector{live: false})
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "ready-item", Title: "Ready", Status: StatusReady, Kind: KindExecute,
	})

	rec := recoverReq(t, h, "ready-item", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRecoverReview_BadTarget_400(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "x", Title: "X", Status: StatusInReview, Kind: KindExecute,
	})
	rec := recoverReq(t, h, "x", `{"to":"completed"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for terminal target, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRecoverReview_RecoverOrphanedReview covers the sweeper entry point: it
// recovers an in_review item and no-ops on an already-advanced one.
func TestRecoverReview_RecoverOrphanedReview(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "swept", Title: "Swept", Status: StatusInReview, Kind: KindExecute,
	})
	if err := h.RecoverOrphanedReview(context.TODO(), "execute", "swept", "no live round"); err != nil {
		t.Fatalf("RecoverOrphanedReview: %v", err)
	}
	updated, _ := h.store.LoadItem(KindExecute, "swept")
	if updated.Status != StatusReviewPending {
		t.Fatalf("status = %q, want %q", updated.Status, StatusReviewPending)
	}

	// Already-advanced item: no-op, no error.
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "done", Title: "Done", Status: StatusCompleted, Kind: KindExecute,
	})
	if err := h.RecoverOrphanedReview(context.TODO(), "execute", "done", "x"); err != nil {
		t.Fatalf("RecoverOrphanedReview on terminal: %v", err)
	}
	again, _ := h.store.LoadItem(KindExecute, "done")
	if again.Status != StatusCompleted {
		t.Fatalf("terminal item was modified: %q", again.Status)
	}
}

// TestReviewDecide_RejectsInReview characterizes the dead-end this feature
// resolves: review-decide cannot exit an in_review item (it requires
// review_pending), which is why recover-review exists.
func TestReviewDecide_RejectsInReview(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name: "stuck", Title: "Stuck", Status: StatusInReview, Kind: KindExecute,
	})
	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/stuck/review-decide", bytes.NewReader(body))
	r = mux.SetURLVars(r, map[string]string{"kind": "execute", "name": "stuck"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, r)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected review-decide to reject in_review with 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
