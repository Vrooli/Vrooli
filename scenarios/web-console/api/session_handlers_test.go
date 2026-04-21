package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// newTestServer creates a Server with real PTY processes — use for
// integration-style tests that need actual shell I/O.
func newTestServer() *Server {
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    NewSessionManager(),
		events:      NewEventLogger(100),
		metrics:     NewMetrics(),
		aiChain:     NewAIProviderChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    NewAIProviderConfigStore(),
		idempotency: newIdempotencyCache(),
		workspace:   NewMemWorkspaceStore(),
	}
	srv.conversations = NewConversationStore()
	srv.codexCheckpointStore = NewInMemoryCodexCheckpointStore()
	srv.ttsSummarization = NewTTSSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)
	return srv
}

// newFakeTestServer creates a Server with pipe-backed fake PTYs — use for
// fast, deterministic handler tests that don't need a real shell.
func newFakeTestServer() *Server {
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    NewSessionManagerWithFactory(newFakePTYFactory()),
		events:      NewEventLogger(100),
		metrics:     NewMetrics(),
		aiChain:     NewAIProviderChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    NewAIProviderConfigStore(),
		idempotency: newIdempotencyCache(),
		workspace:   NewMemWorkspaceStore(),
	}
	srv.conversations = NewConversationStore()
	srv.codexCheckpointStore = NewInMemoryCodexCheckpointStore()
	srv.ttsSummarization = NewTTSSummarizationService(srv.ttsSummarizer, srv.getTTSSummarizeConfig)
	return srv
}

// [REQ:P0-002a] PTY Session Backend - create endpoint
func TestHandleCreateSession(t *testing.T) {
	srv := newTestServer()

	body := strings.NewReader(`{"cols": 80, "rows": 24}`)
	req := httptest.NewRequest("POST", "/api/v1/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleCreateSession(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp SessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.ID == "" {
		t.Error("response ID should not be empty")
	}
	if resp.Cols != 80 {
		t.Errorf("expected cols=80, got %d", resp.Cols)
	}

	// Cleanup
	_ = srv.sessions.Delete(resp.ID)
}

// Failure path: malformed JSON body returns 400 with structured error
func TestHandleCreateSession_MalformedJSON(t *testing.T) {
	srv := newFakeTestServer()

	body := strings.NewReader(`{invalid json}`)
	req := httptest.NewRequest("POST", "/api/v1/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleCreateSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("error response should be valid JSON: %v", err)
	}
	if errResp.Code != "invalid_body" {
		t.Errorf("expected code=invalid_body, got %s", errResp.Code)
	}
	if errResp.Error == "" {
		t.Error("error message should not be empty")
	}
}

// Failure path: session limit returns 429 with structured error
func TestHandleCreateSession_SessionLimit(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessions.cfg.MaxSessions = 1

	// Create one session to hit the limit
	s1, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("first session create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(s1.ID) }()

	// Attempt to create a second session via the handler
	req := httptest.NewRequest("POST", "/api/v1/sessions", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleCreateSession(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("error response should be valid JSON: %v", err)
	}
	if errResp.Code != "session_limit_reached" {
		t.Errorf("expected code=session_limit_reached, got %s", errResp.Code)
	}
}

// Failure path: PTY spawn failure returns 500 with structured error
func TestHandleCreateSession_PTYSpawnFailed(t *testing.T) {
	failingFactory := func(spec SessionLaunchSpec) (PTY, error) {
		return nil, fmt.Errorf("shell not found: %s", spec.Shell)
	}
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    NewSessionManagerWithFactory(failingFactory),
		idempotency: newIdempotencyCache(),
	}

	req := httptest.NewRequest("POST", "/api/v1/sessions", strings.NewReader(`{"shell": "/nonexistent"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleCreateSession(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("error response should be valid JSON: %v", err)
	}
	if errResp.Code != "pty_spawn_failed" {
		t.Errorf("expected code=pty_spawn_failed, got %s", errResp.Code)
	}
	// Ensure internal details are not leaked
	if strings.Contains(errResp.Error, "shell not found") {
		t.Error("error message should not leak internal PTY error details to client")
	}
}

// Failure path: get/delete not-found returns JSON error
func TestHandleGetSession_NotFound_JSONError(t *testing.T) {
	srv := newFakeTestServer()

	req := httptest.NewRequest("GET", "/api/v1/sessions/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleGetSession(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("error response should be valid JSON: %v", err)
	}
	if errResp.Code != "session_not_found" {
		t.Errorf("expected code=session_not_found, got %s", errResp.Code)
	}
}

// DELETE is idempotent: deleting a non-existent session returns 204 with no body.
func TestHandleDeleteSession_NotFound_Idempotent_NoBody(t *testing.T) {
	srv := newFakeTestServer()

	req := httptest.NewRequest("DELETE", "/api/v1/sessions/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleDeleteSession(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 (idempotent), got %d", rec.Code)
	}
}

// Verify sentinel errors wrap correctly
func TestSentinelErrors(t *testing.T) {
	limitErr := fmt.Errorf("%w (%d)", ErrSessionLimitReached, 5)
	if !errors.Is(limitErr, ErrSessionLimitReached) {
		t.Error("wrapped ErrSessionLimitReached should be detectable via errors.Is")
	}

	ptyErr := fmt.Errorf("%w: some detail", ErrPTYSpawnFailed)
	if !errors.Is(ptyErr, ErrPTYSpawnFailed) {
		t.Error("wrapped ErrPTYSpawnFailed should be detectable via errors.Is")
	}
}

// Verify sanitizeID prevents injection and truncates long IDs
func TestSanitizeID(t *testing.T) {
	// Normal ID
	if got := sanitizeID("abc-123"); got != "abc-123" {
		t.Errorf("normal ID: got %q", got)
	}
	// Control characters stripped
	if got := sanitizeID("abc\x00\ndef"); got != "abcdef" {
		t.Errorf("control chars: got %q", got)
	}
	// Long ID truncated
	longID := strings.Repeat("x", 100)
	got := sanitizeID(longID)
	if len(got) > 44 { // 40 + "..."
		t.Errorf("long ID should be truncated, got length %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated ID should end with ...")
	}
}

// --- writeJSON utility tests ---

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

// --- Error category and recovery tests ---

// Verify all error responses include category and recovery fields
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

// Verify session_limit_reached error is retryable
func TestErrorResponse_SessionLimit_Retryable(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessions.cfg.MaxSessions = 1

	s1, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(s1.ID) }()

	req := httptest.NewRequest("POST", "/api/v1/sessions", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCreateSession(rec, req)

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !errResp.Retry {
		t.Error("session_limit_reached should be retryable")
	}
	if errResp.Category != "resource_limit" {
		t.Errorf("expected category=resource_limit, got %s", errResp.Category)
	}
}

// Verify error catalog covers all known error codes
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

// --- Change axis invariant tests ---
// These tests encode structural invariants that must hold when adding
// new error codes, categories, or recovery hints. They validate the
// extension point contract rather than specific values.

// Every error catalog entry must have a valid category and non-empty recovery hint.
// This prevents adding new error codes that silently break the client contract.
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

// writeCatalogError with an unknown code should still return a valid
// structured error (the fallback path). This ensures new codes can
// be referenced in handlers before being added to the catalog during
// development without crashing.
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

// Config variation: multiple session limits should work correctly
// [REQ:P1-001a] Session Policy Controls
func TestSessionLimit_VariousLimits(t *testing.T) {
	limits := []int{1, 3, 5, 10}
	for _, limit := range limits {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			sm := NewSessionManagerWithFactory(newFakePTYFactory())
			sm.cfg.MaxSessions = limit

			var sessions []*Session
			for i := 0; i < limit; i++ {
				s, err := sm.Create("", 0, 0, "", nil)
				if err != nil {
					t.Fatalf("session %d: unexpected error: %v", i, err)
				}
				sessions = append(sessions, s)
			}

			// Next one should fail
			_, err := sm.Create("", 0, 0, "", nil)
			if err == nil {
				t.Errorf("session %d should be rejected when MaxSessions=%d", limit+1, limit)
			}

			// Cleanup
			for _, s := range sessions {
				_ = sm.Delete(s.ID)
			}
		})
	}
}

// Verify X-Request-ID header is set by middleware
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

// --- Session CRUD happy-path tests (use real PTY) ---

// [REQ:P0-003a] Session Persistence Store - list endpoint
func TestHandleListSessions(t *testing.T) {
	srv := newTestServer()

	sess, _ := srv.sessions.Create("", 80, 24, "", nil)
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	req := httptest.NewRequest("GET", "/api/v1/sessions", nil)
	rec := httptest.NewRecorder()

	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var sessions []SessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(sessions) == 0 {
		t.Error("expected at least 1 session")
	}
}

func TestHandleGetSession(t *testing.T) {
	srv := newTestServer()

	sess, _ := srv.sessions.Create("", 80, 24, "", nil)
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	req := httptest.NewRequest("GET", "/api/v1/sessions/"+sess.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	rec := httptest.NewRecorder()

	srv.handleGetSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp SessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, resp.ID)
	}
}

func TestHandleGetSession_NotFound(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/api/v1/sessions/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleGetSession(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteSession(t *testing.T) {
	srv := newTestServer()

	sess, _ := srv.sessions.Create("", 80, 24, "", nil)

	req := httptest.NewRequest("DELETE", "/api/v1/sessions/"+sess.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	rec := httptest.NewRecorder()

	srv.handleDeleteSession(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}

	_, ok := srv.sessions.Get(sess.ID)
	if ok {
		t.Error("session should not exist after delete")
	}
}

// DELETE is idempotent: deleting a non-existent session returns 204 (not 404),
// so that retries and replays are safe.
func TestHandleDeleteSession_NotFound_Idempotent(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("DELETE", "/api/v1/sessions/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleDeleteSession(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 (idempotent delete), got %d", rec.Code)
	}
}

// --- Replay / Idempotency Tests ---

// Deleting the same session twice: first deletes it, second is a no-op 204.
// Metrics and events should only fire once.
func TestDeleteSession_Replay_MetricsOnce(t *testing.T) {
	srv := newFakeTestServer()

	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	deletedBefore := srv.metrics.SessionsDeleted.Load()
	eventsBefore := srv.events.Count()

	// First delete — actually removes the session
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("DELETE", "/api/v1/sessions/"+sess.ID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
		rec := httptest.NewRecorder()
		srv.handleDeleteSession(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("attempt %d: expected 204, got %d", i+1, rec.Code)
		}
	}

	// Metrics should increment exactly once (from the first real delete)
	deletedAfter := srv.metrics.SessionsDeleted.Load()
	if got := deletedAfter - deletedBefore; got != 1 {
		t.Errorf("expected SessionsDeleted to increment by 1, got %d", got)
	}

	// Only one deletion event should be emitted
	eventsAfter := srv.events.Count()
	if got := eventsAfter - eventsBefore; got != 1 {
		t.Errorf("expected 1 deletion event, got %d", got)
	}
}

// Session creation with idempotency key: same key returns the same session.
func TestCreateSession_IdempotencyKey_ReturnsCache(t *testing.T) {
	srv := newFakeTestServer()

	key := "test-idem-key-123"

	// First creation
	body1 := strings.NewReader(`{"cols": 80, "rows": 24}`)
	req1 := httptest.NewRequest("POST", "/api/v1/sessions", body1)
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Idempotency-Key", key)
	rec1 := httptest.NewRecorder()
	srv.handleCreateSession(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d: %s", rec1.Code, rec1.Body.String())
	}
	var resp1 SessionResponse
	if err := json.Unmarshal(rec1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("unmarshal first: %v", err)
	}

	// Replay with same key — should return the cached session, no new session
	body2 := strings.NewReader(`{"cols": 80, "rows": 24}`)
	req2 := httptest.NewRequest("POST", "/api/v1/sessions", body2)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	srv.handleCreateSession(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("replay: expected 201, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var resp2 SessionResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal replay: %v", err)
	}

	// Same session ID returned
	if resp1.ID != resp2.ID {
		t.Errorf("replay should return same session: first=%s, replay=%s", resp1.ID, resp2.ID)
	}

	// Only one session should exist
	sessions := srv.sessions.List()
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}

	// Metrics should show only 1 creation
	if got := srv.metrics.SessionsCreated.Load(); got != 1 {
		t.Errorf("expected SessionsCreated=1, got %d", got)
	}

	// Cleanup
	_ = srv.sessions.Delete(resp1.ID)
}

// Without idempotency key, two requests create two sessions.
func TestCreateSession_NoIdempotencyKey_CreatesTwoSessions(t *testing.T) {
	srv := newFakeTestServer()

	for i := 0; i < 2; i++ {
		body := strings.NewReader(`{}`)
		req := httptest.NewRequest("POST", "/api/v1/sessions", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.handleCreateSession(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %d: expected 201, got %d", i+1, rec.Code)
		}
	}

	sessions := srv.sessions.List()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions without idempotency key, got %d", len(sessions))
	}

	// Cleanup
	for _, s := range sessions {
		_ = srv.sessions.Delete(s.ID)
	}
}

// [REQ:P0-003b] Idempotency cache: expired entries are cleaned up on Get
func TestIdempotencyCache_TTLExpiry(t *testing.T) {
	c := &idempotencyCache{
		entries: make(map[string]idempotencyEntry),
		ttl:     10 * time.Millisecond, // very short TTL
	}

	resp := SessionResponse{ID: "s1"}
	c.Set("key1", resp)

	// Immediately retrievable
	got, ok := c.Get("key1")
	if !ok || got.ID != "s1" {
		t.Fatal("entry should be retrievable immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)

	// Should return false and clean up
	_, ok = c.Get("key1")
	if ok {
		t.Error("expired entry should not be returned by Get")
	}

	// Entry should be deleted from map
	c.mu.Lock()
	_, stillThere := c.entries["key1"]
	c.mu.Unlock()
	if stillThere {
		t.Error("expired entry should be removed from cache map after Get")
	}
}

// [REQ:P0-003b] Idempotency cache: eviction scan triggered when >100 entries
func TestIdempotencyCache_EvictionScan(t *testing.T) {
	c := &idempotencyCache{
		entries: make(map[string]idempotencyEntry),
		ttl:     time.Hour,
	}

	// Pre-populate with 100 expired entries
	for i := 0; i < 100; i++ {
		c.entries[fmt.Sprintf("expired-%d", i)] = idempotencyEntry{
			expiresAt: time.Now().Add(-time.Minute), // already expired
		}
	}

	// This Set should trigger eviction scan (>100 entries)
	c.Set("fresh-key", SessionResponse{ID: "fresh"})

	c.mu.Lock()
	count := len(c.entries)
	c.mu.Unlock()

	// All expired entries should be evicted, leaving only "fresh-key"
	if count != 1 {
		t.Errorf("expected 1 entry after eviction (fresh-key), got %d", count)
	}

	// Fresh entry should still be retrievable
	got, ok := c.Get("fresh-key")
	if !ok || got.ID != "fresh" {
		t.Error("fresh entry should survive eviction scan")
	}
}

// Policy update replay: emits event only once for same policy.
func TestUpdatePolicy_Replay_EventOnlyOnChange(t *testing.T) {
	srv := newFakeTestServer()

	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	eventsBefore := srv.events.Count()

	// Set policy to "1h" preset — should emit event
	for i := 0; i < 3; i++ {
		body := strings.NewReader(`{"mode": "preset", "duration": "1h"}`)
		req := httptest.NewRequest("PUT", "/api/v1/sessions/"+sess.ID+"/policy", body)
		req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.handleUpdatePolicy(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200, got %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	// Only 1 event should be emitted (first call changes policy, repeats are no-ops)
	eventsAfter := srv.events.Count()
	if got := eventsAfter - eventsBefore; got != 1 {
		t.Errorf("expected 1 policy event (first change only), got %d", got)
	}
}
