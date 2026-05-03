package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prompt-manager/store"
)

func TestPendingQueue_ExcludesDeferredWithFutureDate(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	ctx := context.Background()

	future := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	for _, e := range []store.DecisionEntry{
		{ID: "p1", At: "2025-01-01T00:00:00Z", By: "a", Status: store.DecisionStatusPending},
		{ID: "d1", At: "2025-01-01T01:00:00Z", By: "a", Status: store.DecisionStatusDeferred, RevisitAfter: &future},
	} {
		e := e
		if err := ts.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/decisions/pending", nil)
	w := httptest.NewRecorder()
	handlers.GetAllPendingDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	var resp AllPendingDecisionsResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.TotalCount != 1 {
		t.Fatalf("totalCount: want 1, got %d", resp.TotalCount)
	}
	if resp.Teams[0].Entries[0].ID != "p1" {
		t.Errorf("expected p1 only; got %s", resp.Teams[0].Entries[0].ID)
	}
}

func TestPendingQueue_ResurfacesDeferredOnDueDate(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	ctx := context.Background()

	due := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02") // yesterday
	entry := store.DecisionEntry{
		ID:           "d1",
		At:           "2025-01-01T00:00:00Z",
		By:           "a",
		Status:       store.DecisionStatusDeferred,
		RevisitAfter: &due,
	}
	if err := ts.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/decisions/pending", nil)
	w := httptest.NewRecorder()
	handlers.GetAllPendingDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	var resp AllPendingDecisionsResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.TotalCount != 1 {
		t.Fatalf("totalCount: want 1, got %d (body=%s)", resp.TotalCount, w.Body.String())
	}
	got := resp.Teams[0].Entries[0]
	if got.Status != store.DecisionStatusPending {
		t.Errorf("status: want pending, got %q", got.Status)
	}
	if got.RevisitAfter != nil {
		t.Errorf("revisit_after should be cleared after resurface; got %v", *got.RevisitAfter)
	}
	if !strings.Contains(got.Notes, "[re-surfaced after defer]") || !strings.Contains(got.Notes, due) {
		t.Errorf("expected resurface audit note; got %q", got.Notes)
	}

	// Durability: a second call must not duplicate the note.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/decisions/pending", nil)
	w2 := httptest.NewRecorder()
	handlers.GetAllPendingDecisions(w2, req2)
	var resp2 AllPendingDecisionsResponse
	_ = json.NewDecoder(w2.Body).Decode(&resp2)
	if strings.Count(resp2.Teams[0].Entries[0].Notes, "[re-surfaced") != 1 {
		t.Errorf("resurface note must persist exactly once; got notes %q", resp2.Teams[0].Entries[0].Notes)
	}
}
