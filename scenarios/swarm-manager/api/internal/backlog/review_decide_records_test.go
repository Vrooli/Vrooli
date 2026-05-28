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

type fakeStubCreator struct {
	calls []fakeStubCall
	id    string
	err   error
}

type fakeStubCall struct {
	Kind, Name, DecidedBy string
	Status                BacklogStatus
}

func (f *fakeStubCreator) CreateBacklogStub(_ context.Context, kind, name string, status BacklogStatus, decidedBy string) (string, error) {
	f.calls = append(f.calls, fakeStubCall{kind, name, decidedBy, status})
	return f.id, f.err
}

func TestReviewDecide_StubCreated_OnAccept(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "rec-stub-accept", Status: StatusReviewPending, Kind: KindFix,
	})
	stub := &fakeStubCreator{id: "rec-12345"}
	h.SetRecordStubCreator(stub)

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept, DecidedBy: "agent-x"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/fix/rec-stub-accept/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "fix", "name": "rec-stub-accept"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 stub call, got %d", len(stub.calls))
	}
	c := stub.calls[0]
	if c.Kind != "fix" || c.Name != "rec-stub-accept" || c.Status != StatusCompleted || c.DecidedBy != "agent-x" {
		t.Errorf("stub call = %+v", c)
	}
	var resp ReviewDecideResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RecordStubID != "rec-12345" {
		t.Errorf("response RecordStubID = %q, want rec-12345", resp.RecordStubID)
	}
}

func TestReviewDecide_StubSkipped_WhenNoRecord(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "rec-stub-skip", Status: StatusReviewPending, Kind: KindFix,
	})
	stub := &fakeStubCreator{id: "rec-X"}
	h.SetRecordStubCreator(stub)

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept, NoRecord: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/fix/rec-stub-skip/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "fix", "name": "rec-stub-skip"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(stub.calls) != 0 {
		t.Errorf("expected zero stub calls with NoRecord=true, got %d", len(stub.calls))
	}
	var resp ReviewDecideResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.RecordStubID != "" {
		t.Errorf("expected empty RecordStubID, got %q", resp.RecordStubID)
	}
}

func TestReviewDecide_StubFailureDoesNotBlockTerminal(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindFix, BacklogItem{
		Name: "rec-stub-err", Status: StatusReviewPending, Kind: KindFix,
	})
	stub := &fakeStubCreator{err: errors.New("disk full")}
	h.SetRecordStubCreator(stub)

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/fix/rec-stub-err/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "fix", "name": "rec-stub-err"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	// Terminal transition must succeed even when stub creation fails.
	if rec.Code != http.StatusOK {
		t.Fatalf("stub failure must not fail the terminal transition; got %d: %s", rec.Code, rec.Body.String())
	}
	var resp ReviewDecideResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Status != string(StatusCompleted) {
		t.Errorf("status = %q, want completed", resp.Status)
	}
	if resp.RecordStubID != "" {
		t.Errorf("expected empty RecordStubID on stub error, got %q", resp.RecordStubID)
	}
}
