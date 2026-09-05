package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"web-console/internal/audioports"
	"web-console/internal/config"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
	"web-console/session"

	intai "web-console/internal/ai"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	sessionsH "web-console/handlers/sessions"
	intsessions "web-console/internal/sessions"
	"web-console/internal/sessionstore"
	intworkspace "web-console/internal/workspace"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

// newTestServer creates a Server with real PTY processes — use for
// integration-style tests that need actual shell I/O.
func newTestServer() *Server {
	sm := newSessionManager()
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    sm,
		hub:         NewConversationHub(),
		events:      events.NewLogger(100),
		metrics:     metrics.New(),
		aiChain:     intai.NewChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    intai.NewMemConfigStore(),
		idempotency: intsessions.NewIdempotencyCache(),
		workspace:   intworkspace.NewMemStore(),
	}
	srv.conversations = NewConversationStore()
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()
	srv.speechProcessor = audioports.PassthroughSpeechTextProcessor{}
	srv.summarizeAutoPolicy = defaultSummarizeAutoPolicy()
	srv.lastTTSBySource = map[string]conversationAppendSnapshot{}
	srv.lastTTSAckBySrc = map[string]ttsAckSnapshot{}
	srv.ai = intai.NewService(srv.aiChain, srv.aiConfig, nil, srv.events, &srv.metrics.AIGenerations, &srv.metrics.AISuggestions)
	return srv
}

// newFakeTestServer creates a Server with pipe-backed fake PTYs — use for
// fast, deterministic handler tests that don't need a real shell.
func newFakeTestServer() *Server {
	return newFakeTestServerWithFactory(ptyfake.NewFactory())
}

// newFakeTestServerWithFactory is newFakeTestServer with control over the
// PTY factory, for tests that need a backend which fails writes in a
// specific way.
func newFakeTestServerWithFactory(factory pty.Factory) *Server {
	sm := newSessionManagerWithFactory(factory)
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    sm,
		hub:         NewConversationHub(),
		events:      events.NewLogger(100),
		metrics:     metrics.New(),
		aiChain:     intai.NewChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    intai.NewMemConfigStore(),
		idempotency: intsessions.NewIdempotencyCache(),
		workspace:   intworkspace.NewMemStore(),
	}
	srv.conversations = NewConversationStore()
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()
	srv.speechProcessor = audioports.PassthroughSpeechTextProcessor{}
	srv.summarizeAutoPolicy = defaultSummarizeAutoPolicy()
	srv.lastTTSBySource = map[string]conversationAppendSnapshot{}
	srv.lastTTSAckBySrc = map[string]ttsAckSnapshot{}
	srv.ai = intai.NewService(srv.aiChain, srv.aiConfig, nil, srv.events, &srv.metrics.AIGenerations, &srv.metrics.AISuggestions)
	return srv
}

// --- Session CRUD happy paths (Connect handler) ---

func TestHandleCreateSession(t *testing.T) {
	srv := newTestServer()

	sess, err := callCreate(t, srv, 80, 24, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sess.GetId() == "" {
		t.Error("response ID should not be empty")
	}
	if sess.GetCols() != 80 {
		t.Errorf("expected cols=80, got %d", sess.GetCols())
	}

	_ = srv.sessions.Delete(context.Background(), sess.GetId())
}

func TestHandleCreateSession_SessionLimit(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessions.SetConfigField(func(c *config.Config) { c.MaxSessions = 1 })

	s1, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("first session create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), s1.ID) }()

	_, err = callCreate(t, srv, 0, 0, "")
	if got := connectCode(err); got != connect.CodeResourceExhausted {
		t.Errorf("expected CodeResourceExhausted, got %s (err=%v)", got, err)
	}
}

func TestHandleCreateSession_PTYSpawnFailed(t *testing.T) {
	failingFactory := func(spec pty.LaunchSpec) (pty.PTY, error) {
		return nil, fmt.Errorf("shell not found: %s", spec.Shell)
	}
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    newSessionManagerWithFactory(failingFactory),
		events:      events.NewLogger(10),
		metrics:     metrics.New(),
		idempotency: intsessions.NewIdempotencyCache(),
	}

	_, err := callCreate(t, srv, 0, 0, "")
	if got := connectCode(err); got != connect.CodeInternal {
		t.Errorf("expected CodeInternal, got %s (err=%v)", got, err)
	}
	// Internal PTY details must not leak into the error message.
	if err != nil && strings.Contains(err.Error(), "shell not found") {
		t.Errorf("error should not leak internal PTY details, got %v", err)
	}
}

func TestHandleGetSession_NotFound_ReturnsNotFound(t *testing.T) {
	srv := newFakeTestServer()
	_, err := callGet(t, srv, "nonexistent")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %s (err=%v)", got, err)
	}
}

func TestHandleDeleteSession_NotFound_Idempotent(t *testing.T) {
	srv := newFakeTestServer()
	if err := callDelete(t, srv, "nonexistent"); err != nil {
		t.Errorf("delete unknown session should be idempotent (no error), got %v", err)
	}
}

func TestSentinelErrors(t *testing.T) {
	limitErr := fmt.Errorf("%w (%d)", session.ErrSessionLimitReached, 5)
	if !errors.Is(limitErr, session.ErrSessionLimitReached) {
		t.Error("wrapped session.ErrSessionLimitReached should be detectable via errors.Is")
	}

	ptyErr := fmt.Errorf("%w: some detail", session.ErrPTYSpawnFailed)
	if !errors.Is(ptyErr, session.ErrPTYSpawnFailed) {
		t.Error("wrapped session.ErrPTYSpawnFailed should be detectable via errors.Is")
	}
}

func TestSanitizeID(t *testing.T) {
	if got := sanitizeID("abc-123"); got != "abc-123" {
		t.Errorf("normal ID: got %q", got)
	}
	if got := sanitizeID("abc\x00\ndef"); got != "abcdef" {
		t.Errorf("control chars: got %q", got)
	}
	longID := strings.Repeat("x", 100)
	got := sanitizeID(longID)
	if len(got) > 44 {
		t.Errorf("long ID should be truncated, got length %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated ID should end with ...")
	}
}

// --- writeJSON utility tests (unchanged — utility still exists) ---

func TestWriteJSON_SetsContentTypeAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]string{"id": "abc"})

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["id"] != "abc" {
		t.Errorf("expected id=abc, got %s", body["id"])
	}
}

func TestWriteJSON_EncodesSlice(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, []string{"a", "b"})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body []string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body) != 2 || body[0] != "a" || body[1] != "b" {
		t.Errorf("expected [a b], got %v", body)
	}
}

// --- Error category / recovery tests (unchanged — catalog remains) ---

func TestErrorResponse_CategoryAndRecovery(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		category string
		retry    bool
	}{
		{"invalid_body has validation category", "invalid_body", "validation", false},
		{"session_limit_reached has resource_limit category", "session_limit_reached", "resource_limit", true},
		{"pty_spawn_failed has dependency category", "pty_spawn_failed", "dependency", false},
		{"internal_error has internal category", "internal_error", "internal", true},
		{"session_not_found has validation category", "session_not_found", "validation", false},
		{"session_terminated has dependency category", "session_terminated", "dependency", false},
		{"profile_not_found has validation category", "profile_not_found", "validation", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeCatalogError(rec, tt.code, "test message")

			var errResp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if errResp.Category != tt.category {
				t.Errorf("expected category=%s, got %s", tt.category, errResp.Category)
			}
			if errResp.Recovery == "" {
				t.Error("recovery hint should not be empty")
			}
			if errResp.Retry != tt.retry {
				t.Errorf("expected retry=%v, got %v", tt.retry, errResp.Retry)
			}
		})
	}
}

func TestErrorCatalog_Completeness(t *testing.T) {
	expectedCodes := []string{
		"invalid_body", "session_limit_reached", "pty_spawn_failed",
		"internal_error", "session_not_found", "session_terminated",
		"ai_provider_unavailable", "invalid_policy", "profile_not_found",
	}
	for _, code := range expectedCodes {
		if _, ok := errorCatalog[code]; !ok {
			t.Errorf("error catalog missing code: %s", code)
		}
	}
}

func TestErrorCatalog_StructuralInvariants(t *testing.T) {
	validCategories := map[string]bool{
		"validation":     true,
		"resource_limit": true,
		"dependency":     true,
		"internal":       true,
	}

	for code, ae := range errorCatalog {
		t.Run(code, func(t *testing.T) {
			if ae.Code == "" {
				t.Errorf("error catalog entry %q has empty Code", code)
			}
			if ae.Code != code {
				t.Errorf("error catalog entry %q: Code=%q doesn't match map key", code, ae.Code)
			}
			if !validCategories[ae.Category] {
				t.Errorf("error catalog entry %q: invalid category %q", code, ae.Category)
			}
			if ae.Message == "" {
				t.Errorf("error catalog entry %q has empty Message", code)
			}
			if ae.Recovery == "" {
				t.Errorf("error catalog entry %q has empty Recovery hint", code)
			}
			if ae.Status == 0 {
				t.Errorf("error catalog entry %q has zero Status", code)
			}
		})
	}
}

func TestWriteCatalogError_UnknownCode_Fallback(t *testing.T) {
	rec := httptest.NewRecorder()
	writeCatalogError(rec, "new_future_code", "test")

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if errResp.Code != "new_future_code" {
		t.Errorf("expected code=new_future_code, got %s", errResp.Code)
	}
	if errResp.Category != "internal" {
		t.Errorf("unknown codes should default to internal category, got %s", errResp.Category)
	}
	if errResp.Recovery == "" {
		t.Error("fallback recovery hint should not be empty")
	}
}

// [REQ:P1-001a] Session Policy Controls
func TestSessionLimit_VariousLimits(t *testing.T) {
	limits := []int{1, 3, 5, 10}
	for _, limit := range limits {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			sm := newSessionManagerWithFactory(ptyfake.NewFactory())
			sm.SetConfigField(func(c *config.Config) { c.MaxSessions = limit })

			var sessions []*session.Session
			for i := 0; i < limit; i++ {
				s, err := sm.Create(context.Background(), "", 0, 0, "", nil)
				if err != nil {
					t.Fatalf("session %d: unexpected error: %v", i, err)
				}
				sessions = append(sessions, s)
			}

			_, err := sm.Create(context.Background(), "", 0, 0, "", nil)
			if err == nil {
				t.Errorf("session %d should be rejected when MaxSessions=%d", limit+1, limit)
			}

			for _, s := range sessions {
				_ = sm.Delete(context.Background(), s.ID)
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	reqID := rec.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Error("X-Request-ID header should be set")
	}
	if !strings.HasPrefix(reqID, "req-") {
		t.Errorf("request ID should start with 'req-', got %q", reqID)
	}
}

// --- Session CRUD additional Connect-handler tests ---

// [REQ:P0-003a] Session Persistence Store - list endpoint
func TestHandleListSessions(t *testing.T) {
	srv := newTestServer()

	sess, _ := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	got, err := callList(t, srv)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected at least 1 session")
	}
}

func TestHandleGetSession(t *testing.T) {
	srv := newTestServer()
	sess, _ := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	got, err := callGet(t, srv, sess.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.GetId() != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, got.GetId())
	}
}

func TestHandleGetSession_NotFound(t *testing.T) {
	srv := newTestServer()
	_, err := callGet(t, srv, "nonexistent")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %s", got)
	}
}

func TestHandleDeleteSession(t *testing.T) {
	srv := newTestServer()
	sess, _ := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)

	if err := callDelete(t, srv, sess.ID); err != nil {
		t.Errorf("delete: %v", err)
	}

	if _, ok := srv.sessions.Get(sess.ID); ok {
		t.Error("session should not exist after delete")
	}
}

func TestArchivePreservesTranscriptAndUnarchiveRestoresVisibility(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	srv.sessionStore = store
	srv.sessions.SetStore(store)

	sess, err := srv.sessions.Create(ctx, "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, result := srv.conversations.AppendUserEvent(ctx, sess.ID, "test", "keep this transcript"); !result.Appended {
		t.Fatalf("append conversation: %+v", result)
	}

	h := sessionsH.NewConnectHandler(sessionsH.Deps{Service: &sessionsH.Adapter{
		Manager:            srv.sessions,
		Store:              store,
		Idempotency:        srv.idempotency,
		Events:             srv.events,
		Metrics:            srv.metrics,
		Conversations:      srv.conversations,
		ArchiveGracePeriod: time.Hour,
	}})
	if _, err := h.Archive(ctx, connect.NewRequest(&sessionsv1.ArchiveRequest{Id: sess.ID})); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if got := srv.conversations.CountSessionEvents(ctx, sess.ID); got != 1 {
		t.Fatalf("conversation count after archive = %d, want 1", got)
	}
	meta, err := store.Get(ctx, sess.ID)
	if err != nil || meta.ArchivedAt.IsZero() {
		t.Fatalf("archived session row = %+v, err=%v", meta, err)
	}
	if listed, err := h.List(ctx, connect.NewRequest(&sessionsv1.ListRequest{})); err != nil || len(listed.Msg.GetSessions()) != 0 {
		t.Fatalf("live listing after archive = %+v, err=%v", listed, err)
	}

	if _, err := h.Unarchive(ctx, connect.NewRequest(&sessionsv1.UnarchiveRequest{Id: sess.ID})); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	meta, err = store.Get(ctx, sess.ID)
	if err != nil || !meta.ArchivedAt.IsZero() {
		t.Fatalf("unarchived session row = %+v, err=%v", meta, err)
	}
	if got := srv.conversations.CountSessionEvents(ctx, sess.ID); got != 1 {
		t.Fatalf("conversation count after unarchive = %d, want 1", got)
	}
	_ = srv.sessions.Delete(ctx, sess.ID)
}

func TestPermanentDeleteStillDestroysSessionAndTranscript(t *testing.T) { // [REQ:REQ-P0-003a]
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	srv.sessionStore = store
	srv.sessions.SetStore(store)
	sess, err := srv.sessions.Create(ctx, "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, result := srv.conversations.AppendAssistantEvent(ctx, sess.ID, "test", "delete this transcript"); !result.Appended {
		t.Fatalf("append conversation: %+v", result)
	}
	if err := callDelete(t, srv, sess.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(ctx, sess.ID); err == nil {
		t.Fatal("session row survived permanent deletion")
	}
	if got := srv.conversations.CountSessionEvents(ctx, sess.ID); got != 0 {
		t.Fatalf("conversation count after permanent delete = %d, want 0", got)
	}
}

func TestPermanentDeleteDestroysArchivedPersistedSession(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	srv.sessionStore = store
	srv.sessions.SetStore(store)
	if err := store.Save(ctx, sessionstore.Metadata{ID: "archived-only", Status: sessionstore.StatusDismissed, ArchivedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, result := srv.conversations.AppendAssistantEvent(ctx, "archived-only", "test", "delete archived transcript"); !result.Appended {
		t.Fatalf("append conversation: %+v", result)
	}
	if err := callDelete(t, srv, "archived-only"); err != nil {
		t.Fatalf("delete archived: %v", err)
	}
	if _, err := store.Get(ctx, "archived-only"); err == nil {
		t.Fatal("archived session row survived permanent deletion")
	}
	if got := srv.conversations.CountSessionEvents(ctx, "archived-only"); got != 0 {
		t.Fatalf("archived conversation count after permanent delete = %d", got)
	}
}

func TestArchivePruneDefaultsToDryRunAndDeletesNothing(t *testing.T) {
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	old := time.Now().UTC().Add(-30 * 24 * time.Hour)
	if err := store.Save(ctx, sessionstore.Metadata{ID: "dry-run-empty", ArchivedAt: old}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, sessionstore.Metadata{ID: "legacy-visible-not-prunable", Status: sessionstore.StatusDismissed, Created: old}); err != nil {
		t.Fatal(err)
	}
	adapter := &sessionsH.Adapter{
		Manager: srv.sessions, Store: store, Conversations: srv.conversations,
		Events: srv.events, Metrics: srv.metrics, Workspace: srv.workspace,
		RetentionPolicy: func() sessionsH.ArchiveRetentionPolicy {
			return sessionsH.ArchiveRetentionPolicy{MessageLessAge: 7 * 24 * time.Hour}
		},
	}

	result, err := adapter.PruneArchive(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.DryRun || len(result.Actions) != 1 || result.Actions[0].Kind != sessionsH.PruneTranscript {
		t.Fatalf("dry-run result = %+v", result)
	}
	if _, err := store.Get(ctx, "dry-run-empty"); err != nil {
		t.Fatalf("dry run deleted archived row: %v", err)
	}
	snapshot, err := adapter.GetArchiveRetention(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Stats.EntryCount != 2 {
		t.Fatalf("visible archive count = %d, want explicit + legacy entries", snapshot.Stats.EntryCount)
	}
	if _, err := store.Get(ctx, "legacy-visible-not-prunable"); err != nil {
		t.Fatalf("legacy row was touched: %v", err)
	}
}

func TestArchivePruneHomeFirstPreservesConversationAndDecaysRestoreState(t *testing.T) {
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	old := time.Now().UTC().Add(-30 * 24 * time.Hour)
	meta := sessionstore.Metadata{ID: "home-prune", AgentType: sessionstore.AgentCodex, AgentSessionID: "agent-1", ArchivedAt: old}
	if err := store.Save(ctx, meta); err != nil {
		t.Fatal(err)
	}
	if _, result := srv.conversations.AppendAssistantEvent(ctx, meta.ID, "test", "preserve me"); !result.Appended {
		t.Fatalf("append: %+v", result)
	}
	historyPresent := true
	adapter := &sessionsH.Adapter{
		Manager: srv.sessions, Store: store, Conversations: srv.conversations,
		Events: srv.events, Metrics: srv.metrics, Workspace: srv.workspace,
		RetentionPolicy: func() sessionsH.ArchiveRetentionPolicy {
			return sessionsH.ArchiveRetentionPolicy{AgentHomeAge: 7 * 24 * time.Hour}
		},
		AgentHistoryPresent: func(sessionstore.Metadata) bool { return historyPresent },
		AgentHistorySize:    func(sessionstore.Metadata) (int64, error) { return 4096, nil },
		PruneAgentHistory: func(sessionstore.Metadata) (int64, error) {
			historyPresent = false
			return 4096, nil
		},
	}

	result, err := adapter.PruneArchive(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.DryRun || len(result.Actions) != 1 || result.Actions[0].Kind != sessionsH.PruneAgentHome || !result.Actions[0].Applied {
		t.Fatalf("apply result = %+v", result)
	}
	if got := srv.conversations.CountSessionEvents(ctx, meta.ID); got != 1 {
		t.Fatalf("conversation count after home prune = %d, want 1", got)
	}
	rows, err := adapter.ListArchived(ctx)
	if err != nil || len(rows) != 1 || rows[0].RestoreState != sessionsH.RestoreStateReadOnly {
		t.Fatalf("archive after home prune = %+v, err=%v", rows, err)
	}
}

func TestArchivePruneDeletesOnlyOldMessageLessTranscripts(t *testing.T) {
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	old := time.Now().UTC().Add(-30 * 24 * time.Hour)
	for _, id := range []string{"old-empty", "old-with-message"} {
		if err := store.Save(ctx, sessionstore.Metadata{ID: id, ArchivedAt: old}); err != nil {
			t.Fatal(err)
		}
	}
	if _, result := srv.conversations.AppendUserEvent(ctx, "old-with-message", "test", "do not delete"); !result.Appended {
		t.Fatalf("append: %+v", result)
	}
	adapter := &sessionsH.Adapter{
		Manager: srv.sessions, Store: store, Conversations: srv.conversations,
		Events: srv.events, Metrics: srv.metrics, Workspace: srv.workspace,
		RetentionPolicy: func() sessionsH.ArchiveRetentionPolicy {
			return sessionsH.ArchiveRetentionPolicy{MessageLessAge: 7 * 24 * time.Hour}
		},
	}

	if _, err := adapter.PruneArchive(ctx, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "old-empty"); err == nil {
		t.Fatal("old message-less archive survived apply")
	}
	if _, err := store.Get(ctx, "old-with-message"); err != nil {
		t.Fatalf("message-bearing archive was deleted: %v", err)
	}
	if got := srv.conversations.CountSessionEvents(ctx, "old-with-message"); got != 1 {
		t.Fatalf("message-bearing transcript count = %d, want 1", got)
	}
}

func TestListArchivedCollapsesLineageAtNewestRow(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	archivedOld := time.Date(2026, 8, 18, 18, 0, 0, 0, time.UTC)
	archivedNew := archivedOld.Add(time.Hour)
	if err := store.Save(ctx, sessionstore.Metadata{ID: "old", AgentType: sessionstore.AgentCodex, RecoveredInto: "new", ArchivedAt: archivedOld}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, sessionstore.Metadata{ID: "new", AgentType: sessionstore.AgentCodex, ArchivedAt: archivedNew, LastRolloutPath: "/history/new"}); err != nil {
		t.Fatal(err)
	}
	if _, result := srv.conversations.AppendAssistantEvent(ctx, "new", "test", "newest transcript"); !result.Appended {
		t.Fatalf("append: %+v", result)
	}
	if err := srv.workspace.UpsertPane(ctx, intworkspace.Pane{SessionID: "new", Name: "Newest pane", HeaderColor: "#123456"}); err != nil {
		t.Fatal(err)
	}
	adapter := &sessionsH.Adapter{
		Store: store, Conversations: srv.conversations, Workspace: srv.workspace,
		AgentHistoryPresent: func(sessionstore.Metadata) bool { return true },
	}
	rows, err := adapter.ListArchived(ctx)
	if err != nil {
		t.Fatalf("list archived: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "new" || rows[0].MessageCount != 1 || rows[0].PaneName != "Newest pane" {
		t.Fatalf("collapsed rows = %+v", rows)
	}
	if rows[0].RestoreState != sessionsH.RestoreStateReopenable {
		t.Fatalf("restore state = %q", rows[0].RestoreState)
	}
}

func TestListArchivedProjectsHonestNonReopenableStates(t *testing.T) { // [REQ:REQ-P0-003c]
	ctx := context.Background()
	srv := newFakeTestServer()
	store := sessionstore.NewInMemory()
	now := time.Date(2026, 8, 18, 19, 0, 0, 0, time.UTC)
	_ = store.Save(ctx, sessionstore.Metadata{ID: "claude", AgentType: sessionstore.AgentClaude, ArchivedAt: now})
	_ = store.Save(ctx, sessionstore.Metadata{ID: "shell", AgentType: sessionstore.AgentNone, ArchivedAt: now.Add(-time.Minute)})
	if _, result := srv.conversations.AppendUserEvent(ctx, "claude", "test", "preserved"); !result.Appended {
		t.Fatalf("append: %+v", result)
	}
	adapter := &sessionsH.Adapter{Store: store, Conversations: srv.conversations}
	rows, err := adapter.ListArchived(ctx)
	if err != nil {
		t.Fatalf("list archived: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	byID := map[string]sessionsH.ArchivedSession{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	claude := byID["claude"]
	if claude.RestoreState != sessionsH.RestoreStateReadOnly || claude.RestoreStateReason != "claude session id is required (resuming the wrong project is unsafe)" {
		t.Fatalf("claude projection = %+v", claude)
	}
	shell := byID["shell"]
	if shell.RestoreState != sessionsH.RestoreStateNothingToRestore || shell.RestoreStateReason == "" {
		t.Fatalf("shell projection = %+v", shell)
	}
}

func TestListArchivedMarksCrashOrphans(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	if err := store.Save(ctx, sessionstore.Metadata{
		ID: "crash-orphan", AgentType: sessionstore.AgentCodex,
		Status: sessionstore.StatusAwaitingRecovery, OrphanedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := (&sessionsH.Adapter{Store: store}).ListArchived(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !rows[0].AwaitingRecovery {
		t.Fatalf("archive crash marker = %+v", rows)
	}
}

// --- Replay / Idempotency Tests ---

// Deleting the same session 3 times: only one real delete; metrics + events fire once.
func TestDeleteSession_Replay_MetricsOnce(t *testing.T) {
	srv := newFakeTestServer()

	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	deletedBefore := srv.metrics.SessionsDeleted.Load()
	eventsBefore := srv.events.Count()

	for i := 0; i < 3; i++ {
		if err := callDelete(t, srv, sess.ID); err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}

	if got := srv.metrics.SessionsDeleted.Load() - deletedBefore; got != 1 {
		t.Errorf("expected SessionsDeleted to increment by 1, got %d", got)
	}
	if got := srv.events.Count() - eventsBefore; got != 1 {
		t.Errorf("expected 1 deletion event, got %d", got)
	}
}

func TestCreateSession_IdempotencyKey_ReturnsCache(t *testing.T) {
	srv := newFakeTestServer()
	key := "test-idem-key-123"

	first, err := callCreate(t, srv, 80, 24, key)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	second, err := callCreate(t, srv, 80, 24, key)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	if first.GetId() != second.GetId() {
		t.Errorf("replay should return same session: first=%s, replay=%s", first.GetId(), second.GetId())
	}
	if sessions := srv.sessions.List(); len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
	if got := srv.metrics.SessionsCreated.Load(); got != 1 {
		t.Errorf("expected SessionsCreated=1, got %d", got)
	}

	_ = srv.sessions.Delete(context.Background(), first.GetId())
}

func TestCreateSession_NoIdempotencyKey_CreatesTwoSessions(t *testing.T) {
	srv := newFakeTestServer()

	for i := 0; i < 2; i++ {
		if _, err := callCreate(t, srv, 0, 0, ""); err != nil {
			t.Fatalf("create %d: %v", i+1, err)
		}
	}

	sessions := srv.sessions.List()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions without idempotency key, got %d", len(sessions))
	}
	for _, s := range sessions {
		_ = srv.sessions.Delete(context.Background(), s.ID)
	}
}

// Policy update replay: emits event only once for same policy.
func TestUpdatePolicy_Replay_EventOnlyOnChange(t *testing.T) {
	srv := newFakeTestServer()

	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	eventsBefore := srv.events.Count()

	for i := 0; i < 3; i++ {
		if _, err := callUpdatePolicy(t, srv, sess.ID, "preset", "1h"); err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}

	if got := srv.events.Count() - eventsBefore; got != 1 {
		t.Errorf("expected 1 policy event (first change only), got %d", got)
	}
}

// guard: the proto type for CreateRequest is referenced so a removed field
// breaks compilation.
var (
	_ = sessionsv1.CreateRequest{}
	_ = context.Background
)
