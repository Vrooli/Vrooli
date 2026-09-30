package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type fakeRecordCreator struct {
	calls []BacklogRecordRequest
	id    string
	err   error
}

func (f *fakeRecordCreator) CreateBacklogRecord(_ context.Context, req BacklogRecordRequest) (string, error) {
	f.calls = append(f.calls, req)
	return f.id, f.err
}

func TestReviewDecide_RecordCaptured_OnAccept(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "rec-cap-accept", Status: StatusReviewPending, Kind: KindFix,
		Title: "Fix the silence race", Description: "Debounce VAD stop events.",
		Milestone: "voice-reliability", AcceptanceAllow: []string{"scenarios/web-console/**"},
	})
	creator := &fakeRecordCreator{id: "rec-12345"}
	h.SetRecordCreator(creator)

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept, DecidedBy: "agent-x"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/fix/rec-cap-accept/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "fix", "name": "rec-cap-accept"})
	rec := httptest.NewRecorder()
	decideReviewMutation(t, h, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(creator.calls) != 1 {
		t.Fatalf("expected 1 capture call, got %d", len(creator.calls))
	}
	c := creator.calls[0]
	// The hook must hand the records seam the full item context (so the record
	// is born filled with the lesson, not an empty stub).
	if c.Kind != "fix" || c.Name != "rec-cap-accept" || c.Status != StatusCompleted || c.DecidedBy != "agent-x" {
		t.Errorf("capture request core = %+v", c)
	}
	if c.Title != "Fix the silence race" || c.Description != "Debounce VAD stop events." {
		t.Errorf("capture request did not carry item title/description: %+v", c)
	}
	if c.Milestone != "voice-reliability" {
		t.Errorf("capture request milestone = %q, want voice-reliability", c.Milestone)
	}
	if len(c.AcceptanceAllow) != 1 || c.AcceptanceAllow[0] != "scenarios/web-console/**" {
		t.Errorf("capture request acceptance globs = %v", c.AcceptanceAllow)
	}
	var resp ReviewDecideResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RecordID != "rec-12345" {
		t.Errorf("response RecordID = %q, want rec-12345", resp.RecordID)
	}
}

func TestReviewDecide_RecordSkipped_WhenNoRecord(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "rec-cap-skip", Status: StatusReviewPending, Kind: KindFix,
	})
	creator := &fakeRecordCreator{id: "rec-X"}
	h.SetRecordCreator(creator)

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept, NoRecord: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/fix/rec-cap-skip/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "fix", "name": "rec-cap-skip"})
	rec := httptest.NewRecorder()
	decideReviewMutation(t, h, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(creator.calls) != 0 {
		t.Errorf("expected zero capture calls with NoRecord=true, got %d", len(creator.calls))
	}
	var resp ReviewDecideResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.RecordID != "" {
		t.Errorf("expected empty RecordID, got %q", resp.RecordID)
	}
}

func TestReviewDecide_CaptureFailureDoesNotBlockTerminal(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "rec-cap-err", Status: StatusReviewPending, Kind: KindFix,
	})
	creator := &fakeRecordCreator{err: errors.New("disk full")}
	h.SetRecordCreator(creator)

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/fix/rec-cap-err/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "fix", "name": "rec-cap-err"})
	rec := httptest.NewRecorder()
	decideReviewMutation(t, h, rec, req)

	// Terminal transition must succeed even when record capture fails.
	if rec.Code != http.StatusOK {
		t.Fatalf("capture failure must not fail the terminal transition; got %d: %s", rec.Code, rec.Body.String())
	}
	var resp ReviewDecideResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Status != string(StatusCompleted) {
		t.Errorf("status = %q, want completed", resp.Status)
	}
	if resp.RecordID != "" {
		t.Errorf("expected empty RecordID on capture error, got %q", resp.RecordID)
	}
}
