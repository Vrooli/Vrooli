package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"prompt-manager/store"
	"prompt-manager/teamconfig"
	"strings"
	"testing"

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

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Test Team")); err != nil {
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
	all, _, err := teamStore.GetDecisions(ctx, "team-1", "", "", 0)
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

// --- Status filter tests ---

func TestGetDecisions_StatusFilter_Pending(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for _, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "D1", Rationale: "R1", Status: store.DecisionStatusPending},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-1", Decision: "D2", Rationale: "R2", Status: store.DecisionStatusAccepted},
		{ID: "dec-3", At: "2025-01-01T02:00:00Z", By: "agent-1", Decision: "D3", Rationale: "R3", Status: store.DecisionStatusPending},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions?status=pending", nil)
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
		t.Fatalf("expected 2 pending entries, got %d", len(resp.Entries))
	}
	for _, e := range resp.Entries {
		if e.Status != store.DecisionStatusPending {
			t.Errorf("expected status pending, got %s", e.Status)
		}
	}
}

func TestGetDecisions_StatusFilter_Accepted(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for _, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "D1", Rationale: "R1", Status: store.DecisionStatusPending},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-1", Decision: "D2", Rationale: "R2", Status: store.DecisionStatusAccepted},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions?status=accepted", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetDecisions(w, req)

	var resp DecisionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 accepted entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Status != store.DecisionStatusAccepted {
		t.Errorf("expected status accepted, got %s", resp.Entries[0].Status)
	}
}

func TestGetDecisions_StatusFilter_Combined(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for _, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "D1", Rationale: "R1", Context: "auth", Status: store.DecisionStatusPending},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-1", Decision: "D2", Rationale: "R2", Context: "auth", Status: store.DecisionStatusAccepted},
		{ID: "dec-3", At: "2025-01-01T02:00:00Z", By: "agent-1", Decision: "D3", Rationale: "R3", Context: "cache", Status: store.DecisionStatusPending},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions?status=pending&context=auth", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetDecisions(w, req)

	var resp DecisionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry (pending+auth), got %d", len(resp.Entries))
	}
	if resp.Entries[0].ID != "dec-1" {
		t.Errorf("expected dec-1, got %s", resp.Entries[0].ID)
	}
}

func TestGetDecisions_StatusFilter_EmptyStatusMatchesPending(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	// Pre-existing decisions with no status field (empty string)
	for _, e := range []store.DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Old decision", Rationale: "R1"},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-1", Decision: "New decision", Rationale: "R2", Status: store.DecisionStatusAccepted},
	} {
		e := e
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions?status=pending", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetDecisions(w, req)

	var resp DecisionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry (empty status treated as pending), got %d", len(resp.Entries))
	}
	if resp.Entries[0].ID != "dec-1" {
		t.Errorf("expected dec-1, got %s", resp.Entries[0].ID)
	}
}

// --- Auto-pending test ---

func TestAddDecision_AlwaysSetsStatusPending(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	body, _ := json.Marshal(AddDecisionRequest{
		By:        "agent-1",
		Decision:  "A decision",
		Rationale: "Some reason",
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
	if entry.Status != store.DecisionStatusPending {
		t.Errorf("expected status 'pending', got: %s", entry.Status)
	}
}

// --- Approval enforcement tests ---

// setupApprovalTestHandlers sets up handlers with a team in approval mode and a member agent.
func setupApprovalTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore) {
	t.Helper()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	ctx := context.Background()

	// Create team in approval mode
	team := newIndependentTestTeam("team-approval", "Approval Team")
	team.DecisionMode = teamconfig.DecisionModeApproval
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	// Create agent and add as team member
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-approval", AgentID: "agent-1", Status: "active",
	}); err != nil {
		t.Fatalf("set team member: %v", err)
	}

	return handlers, teamStore
}

func TestUpdateDecision_ApprovalMode_AgentBlockedFromAccepting(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusPending}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusAccepted
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp approvalError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Error != "decision_approval_required" {
		t.Errorf("expected error 'decision_approval_required', got: %s", errResp.Error)
	}
}

func TestUpdateDecision_ApprovalMode_AgentBlockedFromRejecting(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusPending}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusRejected
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_ApprovalMode_AgentCanSetPending(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusRunning}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusPending
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_ApprovalMode_AgentCanSetRunning_WhenAccepted(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusAccepted}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusRunning
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_ApprovalMode_AgentCannotSetRunning_WhenPending(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusPending}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusRunning
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp approvalError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errResp.Error != "decision_not_accepted" {
		t.Errorf("expected error 'decision_not_accepted', got: %s", errResp.Error)
	}
}

func TestUpdateDecision_ApprovalMode_AgentCanSetCompleted_WhenRunning(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusRunning}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusCompleted
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_ApprovalMode_AgentCannotSetCompleted_WhenAccepted(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusAccepted}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusCompleted
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_ApprovalMode_HumanCanSetAnyStatus(t *testing.T) {
	handlers, teamStore := setupApprovalTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusPending}
	if err := teamStore.AppendDecision(ctx, "team-approval", &entry); err != nil {
		t.Fatal(err)
	}

	// Human (no X-Caller-ID) can set accepted directly
	status := store.DecisionStatusAccepted
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-approval/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-approval", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for human caller, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_YoloMode_AgentCanSetAnyStatus(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	// Default team (team-1) is yolo mode (no DecisionMode set)
	entry := store.DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "R", Status: store.DecisionStatusPending}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	status := store.DecisionStatusAccepted
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "agent-1")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for yolo mode, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Single-proposal accept-as-proposed tests ---

func TestUpdateDecisionHandler_AcceptSingleProposalNoSelected(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-sp", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Action as proposed", Rationale: "Single-proposal"}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	status := "accepted"
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status})
	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-sp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-sp"})
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
		t.Errorf("status = %q, want accepted", updated.Status)
	}
	if !updated.AcceptedAsProposed {
		t.Errorf("expected AcceptedAsProposed=true on single-proposal accept")
	}
	if updated.Selected != "" {
		t.Errorf("Selected = %q, want empty on accept-as-proposed", updated.Selected)
	}
}

func TestUpdateDecisionHandler_AcceptSingleProposalRejectsExplicitSelected(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{ID: "dec-sp", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Action as proposed", Rationale: "Single-proposal"}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	status := "accepted"
	other := "__other__"
	freeform := "accept as proposed"
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status, Selected: &other, Freeform: &freeform})
	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-sp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-sp"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "single-proposal decisions are accepted with no") {
		t.Errorf("expected migration message, got: %s", w.Body.String())
	}
}

func TestUpdateDecisionHandler_AcceptMultiOptionPreservesNoFlag(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	entry := store.DecisionEntry{
		ID: "dec-mo", At: "2025-01-01T00:00:00Z", By: "agent-1",
		Decision: "Pick one", Rationale: "needs choice",
		Options: []store.DecisionOption{{Key: "A", Label: "alpha"}, {Key: "B", Label: "beta"}},
	}
	if err := teamStore.AppendDecision(ctx, "team-1", &entry); err != nil {
		t.Fatal(err)
	}

	status := "accepted"
	sel := "A"
	body, _ := json.Marshal(UpdateDecisionRequest{Status: &status, Selected: &sel})
	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-mo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-mo"})
	w := httptest.NewRecorder()

	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated store.DecisionEntry
	_ = json.NewDecoder(w.Body).Decode(&updated)
	if updated.AcceptedAsProposed {
		t.Errorf("multi-option accept must not set AcceptedAsProposed")
	}
	if updated.Selected != "A" {
		t.Errorf("Selected = %q, want A", updated.Selected)
	}
}

// --- Pagination / total count tests ---

func TestGetDecisions_TotalCount(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		e := store.DecisionEntry{
			ID:        fmt.Sprintf("dec-%d", i+1),
			At:        fmt.Sprintf("2025-01-01T%02d:00:00Z", i),
			By:        "agent-1",
			Decision:  fmt.Sprintf("Decision %d", i+1),
			Rationale: "Because",
		}
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	// Request last=5, total should be 15
	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/decisions?last=5", nil)
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
	if len(resp.Entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(resp.Entries))
	}
	if resp.Total != 15 {
		t.Errorf("expected total=15, got %d", resp.Total)
	}
	if resp.Last != 5 {
		t.Errorf("expected last=5, got %d", resp.Last)
	}
}

func TestGetDecisions_DefaultLast10(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		e := store.DecisionEntry{
			ID:        fmt.Sprintf("dec-%d", i+1),
			At:        fmt.Sprintf("2025-01-01T%02d:00:00Z", i),
			By:        "agent-1",
			Decision:  fmt.Sprintf("Decision %d", i+1),
			Rationale: "Because",
		}
		if err := teamStore.AppendDecision(ctx, "team-1", &e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	// No last= param, default should be 10
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
	if len(resp.Entries) != 10 {
		t.Fatalf("expected 10 entries (default last=10), got %d", len(resp.Entries))
	}
	if resp.Total != 20 {
		t.Errorf("expected total=20, got %d", resp.Total)
	}
	if resp.Last != 10 {
		t.Errorf("expected last=10, got %d", resp.Last)
	}
}

// --- DecisionModifications tests ---

func acceptWithModifications(t *testing.T, h *Handlers, teamID, decisionID string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/teams/%s/decisions/%s", teamID, decisionID), bytes.NewReader(raw))
	req = mux.SetURLVars(req, map[string]string{"id": teamID, "decisionId": decisionID})
	w := httptest.NewRecorder()
	h.UpdateDecisionHandler(w, req)
	return w
}

func seedPendingOptionDecision(t *testing.T, ts *store.FileTeamStore, teamID, id string) {
	t.Helper()
	e := store.DecisionEntry{
		ID: id, At: "2026-04-24T00:00:00Z", By: "agent-1",
		Topic: "Pick approach", Status: store.DecisionStatusPending,
		Options: []store.DecisionOption{{Key: "A", Label: "a", Rationale: "ra"}},
	}
	if err := ts.AppendDecision(context.Background(), teamID, &e); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestUpdateDecision_ModificationsPersisted(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingOptionDecision(t, ts, "team-1", "dec-mod-1")

	w := acceptWithModifications(t, handlers, "team-1", "dec-mod-1", map[string]any{
		"selected": "A",
		"modifications": map[string]any{
			"excluded_clauses": []string{"relocate existing items"},
			"rationale":        "items stay in their current initiative",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got store.DecisionEntry
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Modifications == nil {
		t.Fatalf("expected modifications persisted")
	}
	if len(got.Modifications.ExcludedClauses) != 1 || got.Modifications.ExcludedClauses[0] != "relocate existing items" {
		t.Errorf("excluded_clauses mismatch: %+v", got.Modifications.ExcludedClauses)
	}
	if got.Modifications.Rationale != "items stay in their current initiative" {
		t.Errorf("rationale mismatch: %q", got.Modifications.Rationale)
	}
	if got.Status != store.DecisionStatusAccepted {
		t.Errorf("expected implicit accept, got %q", got.Status)
	}
}

func TestUpdateDecision_EmptyModificationsRejected(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingOptionDecision(t, ts, "team-1", "dec-mod-2")

	w := acceptWithModifications(t, handlers, "team-1", "dec-mod-2", map[string]any{
		"selected":      "A",
		"modifications": map[string]any{},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["field"] != "modifications" {
		t.Errorf("expected field=modifications, got %v", body["field"])
	}
}

func TestUpdateDecision_ModificationsEmptyStringRejected(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingOptionDecision(t, ts, "team-1", "dec-mod-3")

	w := acceptWithModifications(t, handlers, "team-1", "dec-mod-3", map[string]any{
		"selected": "A",
		"modifications": map[string]any{
			"excluded_clauses": []string{"", "ok"},
		},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDecision_ModificationsImmutable(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingOptionDecision(t, ts, "team-1", "dec-mod-4")

	w := acceptWithModifications(t, handlers, "team-1", "dec-mod-4", map[string]any{
		"selected": "A",
		"modifications": map[string]any{
			"rationale": "first",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("first accept: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w2 := acceptWithModifications(t, handlers, "team-1", "dec-mod-4", map[string]any{
		"modifications": map[string]any{
			"rationale": "second",
		},
	})
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("second mutation: expected 400, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestUpdateDecision_WithoutModificationsLeavesNil(t *testing.T) {
	handlers, ts := setupDecisionTestHandlers(t)
	seedPendingOptionDecision(t, ts, "team-1", "dec-mod-5")

	w := acceptWithModifications(t, handlers, "team-1", "dec-mod-5", map[string]any{
		"selected": "A",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got store.DecisionEntry
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Modifications != nil {
		t.Errorf("expected modifications nil when absent, got %+v", got.Modifications)
	}
}
