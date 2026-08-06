package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/store"

	"github.com/gorilla/mux"
)

// setupKnowledgeAddTest provisions an empty team-1 backed by an
// on-disk file store so AddKnowledge persists end-to-end. Returns the
// handler set + team store for follow-up assertions.
func setupKnowledgeAddTest(t *testing.T) (*Handlers, *store.FileTeamStore) {
	t.Helper()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Test Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	return handlers, teamStore
}

// invokeAddKnowledge wires up a request with the supplied body and
// optional X-Vrooli-Attribution header. Returns status + body for
// assertions.
func invokeAddKnowledge(t *testing.T, h *Handlers, teamID string, body any, headerValue string) (int, string, store.KnowledgeEntry) {
	t.Helper()
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/teams/%s/knowledge", teamID), bytes.NewReader(bodyJSON))
	req = mux.SetURLVars(req, map[string]string{"id": teamID})
	if headerValue != "" {
		req.Header.Set(attributionHeaderName, headerValue)
	}

	w := httptest.NewRecorder()
	h.AddKnowledge(w, req)

	var entry store.KnowledgeEntry
	if w.Code == http.StatusCreated {
		if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
			t.Fatalf("decode response: %v\nbody: %s", err, w.Body.String())
		}
	}
	return w.Code, w.Body.String(), entry
}

func TestAddKnowledge_RejectsMissingHeader(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	body := AddKnowledgeRequest{Topic: "audience-scan/2026-05-04/test", Content: "hello"}
	code, body2, _ := invokeAddKnowledge(t, h, "team-1", body, "")
	if code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", code, body2)
	}
	if !strings.Contains(body2, "X-Vrooli-Attribution") {
		t.Errorf("error must mention header name: %s", body2)
	}
}

func TestAddKnowledge_RejectsMalformedHeader(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	body := AddKnowledgeRequest{Topic: "t", Content: "c"}

	for _, hv := range []string{"!!!", "QmFkOiB7", "_not_base64="} {
		code, b, _ := invokeAddKnowledge(t, h, "team-1", body, hv)
		if code != http.StatusBadRequest {
			t.Errorf("header %q: want 400, got %d: %s", hv, code, b)
		}
	}
}

func TestAddKnowledge_RejectsTeamMismatch(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		MemberID:    ptr("researcher"),
		TeamID:      ptr("monetization"),
		RunID:       ptr("run-abc"),
		SpawnOrigin: store.SpawnOriginHeartbeat,
	})
	body := AddKnowledgeRequest{Topic: "t", Content: "c"}
	code, b, _ := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", code, b)
	}
	if !strings.Contains(b, "team_mismatch") {
		t.Errorf("error must call out team_mismatch: %s", b)
	}
}

func TestAddKnowledge_AcceptsOperatorDirect(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindOperatorDirect,
		SpawnOrigin: store.SpawnOriginOperatorCLI,
	})
	body := AddKnowledgeRequest{
		Topic:      "audience-scan/2026-05-04/manual",
		Content:    "operator note",
		CallerNote: "hand-curated from yesterday's email",
	}
	code, raw, entry := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", code, raw)
	}
	if entry.Caller != "operator" {
		t.Errorf("Caller = %q, want 'operator'", entry.Caller)
	}
	if entry.CallerNote != "hand-curated from yesterday's email" {
		t.Errorf("CallerNote not persisted: %q", entry.CallerNote)
	}
	if entry.Attribution.Kind != store.KnowledgeKindOperatorDirect {
		t.Errorf("Attribution.Kind = %q", entry.Attribution.Kind)
	}
}

func TestAddKnowledge_AcceptsAgentMember(t *testing.T) {
	h, teamStore := setupKnowledgeAddTest(t)
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		MemberID:    ptr("researcher"),
		TeamID:      ptr("team-1"),
		RunID:       ptr("01234567-89ab-cdef"),
		SpawnOrigin: store.SpawnOriginHeartbeat,
	})
	body := AddKnowledgeRequest{
		Topic:   "audience-scan/2026-05-04/q2-creators",
		Content: "agent observation",
	}
	code, raw, entry := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", code, raw)
	}
	if entry.Caller != "team-1/researcher" {
		t.Errorf("Caller = %q, want 'team-1/researcher'", entry.Caller)
	}
	if entry.Attribution.RunID == nil || *entry.Attribution.RunID != "01234567-89ab-cdef" {
		t.Errorf("RunID not persisted: %v", entry.Attribution.RunID)
	}

	// Round-trip: persisted entry must round-trip with structured
	// attribution intact (no field dropped on disk).
	stored, err := teamStore.GetKnowledge(context.Background(), "team-1", "", "", 10)
	if err != nil {
		t.Fatalf("GetKnowledge: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored entry, got %d", len(stored))
	}
	if stored[0].Attribution.MemberID == nil || *stored[0].Attribution.MemberID != "researcher" {
		t.Errorf("on-disk MemberID lost: %+v", stored[0].Attribution)
	}
}

func TestAddKnowledge_RejectsLegacyKindAtWrite(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindLegacy,
		SpawnOrigin: store.SpawnOriginLegacy,
	})
	body := AddKnowledgeRequest{Topic: "t", Content: "c"}
	code, raw, _ := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusBadRequest {
		t.Fatalf("want 400 rejecting legacy kind at write, got %d: %s", code, raw)
	}
}

func TestAddKnowledge_RejectsCallerNoteOverCap(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindOperatorDirect,
		SpawnOrigin: store.SpawnOriginOperatorCLI,
	})
	body := AddKnowledgeRequest{
		Topic:      "t",
		Content:    "c",
		CallerNote: strings.Repeat("x", callerNoteMaxLen+1),
	}
	code, raw, _ := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusBadRequest {
		t.Fatalf("want 400 over-cap, got %d: %s", code, raw)
	}
	if !strings.Contains(raw, "caller_note") {
		t.Errorf("error must mention caller_note: %s", raw)
	}
}

func TestAddKnowledge_RejectsMissingTopic(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindOperatorDirect,
		SpawnOrigin: store.SpawnOriginOperatorCLI,
	})
	body := AddKnowledgeRequest{Content: "missing topic"}
	code, _, _ := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", code)
	}
}

func TestAddKnowledge_RejectsMissingContent(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindOperatorDirect,
		SpawnOrigin: store.SpawnOriginOperatorCLI,
	})
	body := AddKnowledgeRequest{Topic: "t"}
	code, _, _ := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", code)
	}
}

func TestAddKnowledge_RejectsAgentMemberMissingMemberID(t *testing.T) {
	h, _ := setupKnowledgeAddTest(t)
	// agent-member with all required fields except member_id — must
	// be rejected before we even touch the body.
	hdr := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		TeamID:      ptr("team-1"),
		RunID:       ptr("run-abc"),
		SpawnOrigin: store.SpawnOriginHeartbeat,
	})
	body := AddKnowledgeRequest{Topic: "t", Content: "c"}
	code, raw, _ := invokeAddKnowledge(t, h, "team-1", body, hdr)
	if code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", code, raw)
	}
	if !strings.Contains(raw, "member_id") {
		t.Errorf("error must call out member_id: %s", raw)
	}
}
