package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

func setupDecisionTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore) {
	t.Helper()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	if err := teamStore.Create(context.Background(), &store.Team{
		ID: "team-1", DisplayName: "Test Team", Enabled: true,
	}); err != nil {
		t.Fatalf("create team: %v", err)
	}
	return handlers, teamStore
}

func TestGetDecisions_Empty(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DecisionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(resp.Entries))
	}
}

func TestGetDecisions_WithEntries(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for _, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Use JWT", Rationale: "Stateless", Context: "auth"},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-2", Decision: "Use Redis", Rationale: "Fast", Context: "cache"},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DecisionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Entries))
	}
}

func TestGetDecisions_FilterContext(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for _, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Use JWT", Rationale: "Stateless", Context: "auth"},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-2", Decision: "Use Redis", Rationale: "Fast", Context: "cache"},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions?context=auth", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DecisionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry for context=auth, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Context != "auth" {
		t.Errorf("expected context 'auth', got: %s", resp.Entries[0].Context)
	}
}

func TestGetDecisions_Limit(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for i, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "First", Rationale: "R1"},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-1", Decision: "Second", Rationale: "R2"},
		{ID: "dec-3", At: "2025-01-01T02:00:00Z", By: "agent-1", Decision: "Third", Rationale: "R3"},
	} {
		e := e
		_ = i
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions?last=2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetDecisions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DecisionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries with last=2, got %d", len(resp.Entries))
	}
}

func TestAddDecision_Success(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	body, _ := json.Marshal(AddDecisionRequest{
		By:        "agent-1",
		Decision:  "Use JWT for auth",
		Rationale: "Stateless, works across services",
		Context:   "auth",
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/decisions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddDecision(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var entry store.DecisionEntry
	if err := json.NewDecoder(w.Body).Decode(&entry); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if entry.Decision != "Use JWT for auth" {
		t.Errorf("expected 'Use JWT for auth', got: %s", entry.Decision)
	}
	if entry.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestAddDecision_MissingDecision(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	body, _ := json.Marshal(AddDecisionRequest{
		By:        "agent-1",
		Decision:  "",
		Rationale: "Some rationale",
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/decisions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddDecision(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddDecision_MissingRationale(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	body, _ := json.Marshal(AddDecisionRequest{
		By:        "agent-1",
		Decision:  "Some decision",
		Rationale: "",
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/decisions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddDecision(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddDecision_InvalidJSON(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/decisions", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddDecision(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecisionHandler_Status(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "Reason"}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	status := "accepted"
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated store.DecisionEntry
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Status != "accepted" {
		t.Errorf("expected status 'accepted', got: %s", updated.Status)
	}
}

func TestUpdateDecisionHandler_Fields(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Original", Rationale: "Old reason"}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	newDecision := "Updated decision"
	newRationale := "New reason"
	body, _ := json.Marshal(UpdateDecisionRequest{Decision: &newDecision, Rationale: &newRationale})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated store.DecisionEntry
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Decision != "Updated decision" {
		t.Errorf("expected 'Updated decision', got: %s", updated.Decision)
	}
	if updated.Rationale != "New reason" {
		t.Errorf("expected 'New reason', got: %s", updated.Rationale)
	}
}

func TestUpdateDecisionHandler_NotFound(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	status := "accepted"
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDecisionHandler_Success(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Delete me", Rationale: "Reason"}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/decisions/dec-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.DeleteDecisionHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	all, err := teamStore.GetDecisions(ctx, "team-1", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 decisions after delete, got %d", len(all))
	}
}

func TestDeleteDecisionHandler_NotFound(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/decisions/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.DeleteDecisionHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
