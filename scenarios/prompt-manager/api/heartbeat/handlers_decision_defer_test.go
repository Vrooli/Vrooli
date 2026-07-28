package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

// helper: send PATCH to UpdateDecisionHandler.
func patchDecision(t *testing.T, h *Handlers, teamID, decisionID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	bs, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/teams/"+teamID+"/decisions/"+decisionID, bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": teamID, "decisionId": decisionID})
	w := httptest.NewRecorder()
	h.UpdateDecisionHandler(w, req)
	return w
}

func seedPendingDecision(t *testing.T, ts *store.FileTeamStore, teamID, decisionID string) {
	t.Helper()
	entry := store.DecisionEntry{
		ID:        decisionID,
		At:        "2025-01-01T00:00:00Z",
		By:        "agent-1",
		Decision:  "Test",
		Rationale: "Reason",
		Status:    store.DecisionStatusPending,
	}
	if err := ts.AppendDecision(context.Background(), teamID, &entry); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateDecision_DeferFromPending_RequiresRevisitAfter(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingDecision(t, ts, "team-1", "dec-1")

	status := store.DecisionStatusDeferred
	w := patchDecision(t, handlers, "team-1", "dec-1", UpdateDecisionRequest{Status: &status})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_DeferRejectsPastDate(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingDecision(t, ts, "team-1", "dec-1")

	status := store.DecisionStatusDeferred
	past := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	w := patchDecision(t, handlers, "team-1", "dec-1", UpdateDecisionRequest{Status: &status, RevisitAfter: &past})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_DeferRejectsMalformedDate(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingDecision(t, ts, "team-1", "dec-1")

	status := store.DecisionStatusDeferred
	bad := "not-a-date"
	w := patchDecision(t, handlers, "team-1", "dec-1", UpdateDecisionRequest{Status: &status, RevisitAfter: &bad})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_DeferRejectsHorizonExceeded(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingDecision(t, ts, "team-1", "dec-1")

	status := store.DecisionStatusDeferred
	tooFar := time.Now().UTC().AddDate(0, 0, store.MaxRevisitAfterDays+5).Format("2006-01-02")
	w := patchDecision(t, handlers, "team-1", "dec-1", UpdateDecisionRequest{Status: &status, RevisitAfter: &tooFar})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_DeferHappyPath(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingDecision(t, ts, "team-1", "dec-1")

	status := store.DecisionStatusDeferred
	revisit := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	w := patchDecision(t, handlers, "team-1", "dec-1", UpdateDecisionRequest{Status: &status, RevisitAfter: &revisit})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got store.DecisionEntry
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.Status != store.DecisionStatusDeferred {
		t.Errorf("status: want deferred, got %q", got.Status)
	}
	if got.RevisitAfter == nil || *got.RevisitAfter != revisit {
		t.Errorf("revisit_after: want %q, got %v", revisit, got.RevisitAfter)
	}
}

func TestUpdateDecision_DeferRejectsIllegalSourceTransition(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	// Seed an accepted decision.
	entry := store.DecisionEntry{ID: "dec-acc", At: "2025-01-01T00:00:00Z", By: "agent-1", Status: store.DecisionStatusAccepted}
	if err := ts.AppendDecision(context.Background(), "team-1", &entry); err != nil {
		t.Fatal(err)
	}
	status := store.DecisionStatusDeferred
	revisit := time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02")
	w := patchDecision(t, handlers, "team-1", "dec-acc", UpdateDecisionRequest{Status: &status, RevisitAfter: &revisit})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_ReDeferInPlace(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	prev := time.Now().UTC().AddDate(0, 0, 3).Format("2006-01-02")
	prevPtr := prev
	entry := store.DecisionEntry{
		ID:           "dec-d",
		At:           "2025-01-01T00:00:00Z",
		By:           "agent-1",
		Status:       store.DecisionStatusDeferred,
		RevisitAfter: &prevPtr,
	}
	if err := ts.AppendDecision(context.Background(), "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusDeferred
	newDate := time.Now().UTC().AddDate(0, 0, 14).Format("2006-01-02")
	w := patchDecision(t, handlers, "team-1", "dec-d", UpdateDecisionRequest{Status: &status, RevisitAfter: &newDate})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got store.DecisionEntry
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.RevisitAfter == nil || *got.RevisitAfter != newDate {
		t.Errorf("revisit_after: want %q, got %v", newDate, got.RevisitAfter)
	}
	if !strings.Contains(got.Notes, "[re-deferred]") || !strings.Contains(got.Notes, prev) || !strings.Contains(got.Notes, newDate) {
		t.Errorf("expected re-defer audit note in notes, got %q", got.Notes)
	}
}

func TestUpdateDecision_AcceptFromDeferred(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	prev := time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02")
	prevPtr := prev
	entry := store.DecisionEntry{ID: "dec-d", At: "2025-01-01T00:00:00Z", By: "agent-1", Status: store.DecisionStatusDeferred, RevisitAfter: &prevPtr}
	if err := ts.AppendDecision(context.Background(), "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusAccepted
	w := patchDecision(t, handlers, "team-1", "dec-d", UpdateDecisionRequest{Status: &status})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got store.DecisionEntry
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.Status != store.DecisionStatusAccepted {
		t.Errorf("status: want accepted, got %q", got.Status)
	}
	if got.RevisitAfter != nil {
		t.Errorf("revisit_after should be cleared on un-defer; got %v", *got.RevisitAfter)
	}
}

func TestUpdateDecision_DeferToCompletedRejected(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	prev := time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02")
	prevPtr := prev
	entry := store.DecisionEntry{ID: "dec-d", At: "2025-01-01T00:00:00Z", By: "agent-1", Status: store.DecisionStatusDeferred, RevisitAfter: &prevPtr}
	if err := ts.AppendDecision(context.Background(), "team-1", &entry); err != nil {
		t.Fatal(err)
	}
	status := store.DecisionStatusCompleted
	w := patchDecision(t, handlers, "team-1", "dec-d", UpdateDecisionRequest{Status: &status})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for deferred → completed, got %d: %s", w.Code, w.Body.String())
	}
}
