package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"prompt-manager/store"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// stubSwarmCreator captures the swarm-manager call for assertion or simulates
// a failure. Wire via NewSwarmInitiativeClient + handler.SetSwarmInitiativeClient
// in tests; here we go one level lower: stub the HTTP layer with httptest.
func newSwarmStubServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/initiatives" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAddDecision_RejectsInitiativeMetadataOnNonProposalContext(t *testing.T) {
	handlers, _ := setupDecisionTestHandlers(t)

	body := AddDecisionRequest{
		By:        "agent-1",
		Decision:  "x",
		Rationale: "y",
		Context:   "tech-debt", // not initiative-proposal
		InitiativeMetadata: &store.DecisionInitiativeMetadata{
			Name: "web-console-readiness",
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/decisions", bytes.NewReader(raw))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()
	handlers.AddDecision(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "initiative_metadata") {
		t.Errorf("expected initiative_metadata field error, got: %s", w.Body.String())
	}
}

func TestAcceptInitiativeProposal_AutoCreatesAndPersistsRef(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	// Seed an initiative-proposal decision with metadata.
	entry := &store.DecisionEntry{
		ID:      "dec-1",
		At:      "2026-04-25T00:00:00Z",
		By:      "agent-1",
		Topic:   "Web console readiness",
		Context: store.DecisionContextInitiativeProposal,
		Status:  store.DecisionStatusPending,
		Options: []store.DecisionOption{
			{Key: "A", Label: "Build it", Rationale: "Needed for milestone"},
		},
		InitiativeMetadata: &store.DecisionInitiativeMetadata{
			Name:     "web-console-readiness",
			Priority: 5,
		},
	}
	if err := teamStore.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Stub swarm-manager.
	srv := newSwarmStubServer(t, http.StatusCreated, map[string]any{"name": "web-console-readiness"})
	client := NewSwarmInitiativeClient(0)
	client.testBaseURL = srv.URL
	handlers.SetSwarmInitiativeClient(client)

	body := UpdateDecisionRequest{
		Status:   strPtr(store.DecisionStatusAccepted),
		Selected: strPtr("A"),
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(raw))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()
	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp UpdateDecisionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.AutoCreateOutcome == nil || resp.AutoCreateOutcome.Status != "created" {
		t.Fatalf("expected auto_create_outcome.status=created, got: %+v", resp.AutoCreateOutcome)
	}
	if resp.AutoCreateOutcome.InitiativeRef != "swarm-manager/web-console-readiness" {
		t.Errorf("unexpected ref: %q", resp.AutoCreateOutcome.InitiativeRef)
	}
	if resp.AutoCreateStatus != store.AutoCreateStatusCreated {
		t.Errorf("expected persisted status=created, got %q", resp.AutoCreateStatus)
	}
}

func TestAcceptInitiativeProposal_FailureSurfacesWorkaround(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	entry := &store.DecisionEntry{
		ID:      "dec-1",
		At:      "2026-04-25T00:00:00Z",
		By:      "agent-1",
		Topic:   "Web console readiness",
		Context: store.DecisionContextInitiativeProposal,
		Status:  store.DecisionStatusPending,
		Options: []store.DecisionOption{{Key: "A", Label: "Build it", Rationale: "Needed"}},
		InitiativeMetadata: &store.DecisionInitiativeMetadata{
			Name:     "web-console-readiness",
			Priority: 5,
		},
	}
	if err := teamStore.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Stub swarm-manager returning 409.
	srv := newSwarmStubServer(t, http.StatusConflict, map[string]any{
		"error":   "conflict",
		"message": `initiative "web-console-readiness" already exists`,
	})
	client := NewSwarmInitiativeClient(0)
	client.testBaseURL = srv.URL
	handlers.SetSwarmInitiativeClient(client)

	body := UpdateDecisionRequest{Status: strPtr(store.DecisionStatusAccepted), Selected: strPtr("A")}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(raw))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()
	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (accept survives create failure), got %d: %s", w.Code, w.Body.String())
	}
	var resp UpdateDecisionResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != store.DecisionStatusAccepted {
		t.Errorf("expected decision accepted despite create failure, got status=%q", resp.Status)
	}
	if resp.AutoCreateOutcome == nil || resp.AutoCreateOutcome.Status != "failed" {
		t.Fatalf("expected auto_create_outcome.status=failed, got: %+v", resp.AutoCreateOutcome)
	}
	if !strings.Contains(resp.AutoCreateOutcome.WorkaroundCommand, "swarm-manager initiatives create") {
		t.Errorf("expected pre-filled swarm-manager workaround, got: %q", resp.AutoCreateOutcome.WorkaroundCommand)
	}
	if !strings.Contains(resp.AutoCreateOutcome.ResolveCommand, "decision-update") ||
		!strings.Contains(resp.AutoCreateOutcome.ResolveCommand, "auto-create-status=created") {
		t.Errorf("expected pre-filled decision-update follow-up, got: %q", resp.AutoCreateOutcome.ResolveCommand)
	}
	if resp.AutoCreateStatus != store.AutoCreateStatusFailed {
		t.Errorf("expected persisted status=failed, got %q", resp.AutoCreateStatus)
	}
}

func TestAcceptInitiativeProposal_RequiresMetadata(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	// initiative-proposal decision *without* metadata.
	entry := &store.DecisionEntry{
		ID:      "dec-1",
		At:      "2026-04-25T00:00:00Z",
		By:      "agent-1",
		Topic:   "x",
		Context: store.DecisionContextInitiativeProposal,
		Status:  store.DecisionStatusPending,
		Options: []store.DecisionOption{{Key: "A", Label: "x", Rationale: "y"}},
	}
	if err := teamStore.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("append: %v", err)
	}

	body := UpdateDecisionRequest{Status: strPtr(store.DecisionStatusAccepted), Selected: strPtr("A")}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(raw))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()
	handlers.UpdateDecisionHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "initiative_metadata") {
		t.Errorf("expected initiative_metadata error, got: %s", w.Body.String())
	}
}

func TestDecisionUpdate_AutoCreateStatusManualRecovery(t *testing.T) {
	handlers, teamStore := setupDecisionTestHandlers(t)
	ctx := context.Background()

	// Seed an accepted decision in failed auto-create state.
	entry := &store.DecisionEntry{
		ID:                 "dec-1",
		At:                 "2026-04-25T00:00:00Z",
		By:                 "agent-1",
		Topic:              "x",
		Context:            store.DecisionContextInitiativeProposal,
		Status:             store.DecisionStatusAccepted,
		AutoCreateStatus:   store.AutoCreateStatusFailed,
		AutoCreateError:    "boom",
		InitiativeMetadata: &store.DecisionInitiativeMetadata{Name: "foo"},
	}
	if err := teamStore.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Without ref → rejected.
	body := UpdateDecisionRequest{AutoCreateStatus: strPtr(store.AutoCreateStatusCreated)}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(raw))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w := httptest.NewRecorder()
	handlers.UpdateDecisionHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without ref, got %d: %s", w.Code, w.Body.String())
	}

	// With ref → accepted, persisted.
	ref := "swarm-manager/foo"
	body = UpdateDecisionRequest{
		AutoCreateStatus:        strPtr(store.AutoCreateStatusCreated),
		AutoCreateInitiativeRef: &ref,
	}
	raw, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPatch, "/teams/team-1/decisions/dec-1", bytes.NewReader(raw))
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "decisionId": "dec-1"})
	w = httptest.NewRecorder()
	handlers.UpdateDecisionHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got, _ := handlers.findDecision(ctx, "team-1", "dec-1")
	if got == nil || got.AutoCreateStatus != store.AutoCreateStatusCreated || got.AutoCreateInitiativeRef != ref {
		t.Errorf("expected status=created, ref=%q; got %+v", ref, got)
	}
	if got != nil && got.AutoCreateError != "" {
		t.Errorf("expected error cleared on flip to created, got %q", got.AutoCreateError)
	}
}

func TestValidateInitiativeMetadata_NameRegex(t *testing.T) {
	for _, c := range []struct {
		name string
		ok   bool
	}{
		{"web-console-readiness", true},
		{"abc", true},
		{"abc123", true},
		{"-bad", false},
		{"bad-", false},
		{"Bad", false},
		{"", false},
	} {
		err := validateInitiativeMetadata(&store.DecisionInitiativeMetadata{Name: c.name})
		if c.ok && err != nil {
			t.Errorf("name=%q: expected ok, got err=%v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("name=%q: expected error", c.name)
		}
	}
}

// strPtr returns &s. Tiny helper to keep the test bodies tight.
func strPtr(s string) *string { return &s }

// keep the imports minimal — fmt is referenced indirectly when Errorf paths
// fire above; keep this here in case a future test needs structured assertions.
var _ = fmt.Sprintf
