// DOC: docs/concepts/ARCHITECTURE.md#data-flow
// DOC: docs/internal/ERROR-SEMANTICS.md
// DOC: docs/internal/INVARIANTS.md

package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// idempotencyEntry caches the result of a session creation keyed by
// the client-provided X-Idempotency-Key header. Entries expire after
// idempotencyTTL so memory is bounded.
type idempotencyEntry struct {
	response  SessionResponse
	expiresAt time.Time
}

// idempotencyCache is a bounded, TTL-scoped cache that prevents duplicate
// session creation when clients retry with the same idempotency key.
type idempotencyCache struct {
	mu      sync.Mutex
	entries map[string]idempotencyEntry
	ttl     time.Duration
}

const idempotencyTTL = 5 * time.Minute

func newIdempotencyCache() *idempotencyCache {
	return &idempotencyCache{
		entries: make(map[string]idempotencyEntry),
		ttl:     idempotencyTTL,
	}
}

// Get returns the cached response for a key, or false if not found/expired.
func (c *idempotencyCache) Get(key string) (SessionResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(c.entries, key) // clean up expired
		return SessionResponse{}, false
	}
	return entry.response, true
}

// Set stores a response under the given key with TTL.
func (c *idempotencyCache) Set(key string, resp SessionResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = idempotencyEntry{
		response:  resp,
		expiresAt: time.Now().Add(c.ttl),
	}
	// Opportunistic eviction: remove expired entries when cache grows
	if len(c.entries) > 100 {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
	}
}

// CreateSessionRequest is the JSON body for creating a new session.
type CreateSessionRequest struct {
	Shell   string            `json:"shell,omitempty"`
	Cols    int               `json:"cols,omitempty"`
	Rows    int               `json:"rows,omitempty"`
	Backend string            `json:"backend,omitempty"`
	Policy  *CreatePolicySpec `json:"policy,omitempty"`
}

// CreatePolicySpec allows setting a policy at session creation time.
type CreatePolicySpec struct {
	Mode     string `json:"mode"`
	Duration string `json:"duration,omitempty"`
}

// SessionResponse is the JSON representation of a session.
type SessionResponse struct {
	ID              string           `json:"id"`
	Shell           string           `json:"shell"`
	CreatedAt       string           `json:"created_at"`
	Cols            int              `json:"cols"`
	Rows            int              `json:"rows"`
	Backend         BackendID        `json:"backend"`
	SurvivesRestart bool             `json:"survives_restart"`
	Policy          ExpirationPolicy `json:"policy"`
	Busy            bool             `json:"busy"`
	Recovered       bool             `json:"recovered,omitempty"`
}

// sessionToResponse converts an internal Session to the JSON-safe response
// format. Timestamps are serialized as UTC ISO 8601 strings.
// [REQ:P1-001a] Includes expiration policy in response
func sessionToResponse(s *Session) SessionResponse {
	return SessionResponse{
		ID:              s.ID,
		Shell:           s.Shell,
		CreatedAt:       s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Cols:            int(s.Cols),
		Rows:            int(s.Rows),
		Backend:         s.Backend,
		SurvivesRestart: s.Backend == BackendPersistent,
		Policy:          s.GetPolicy(),
		Busy:            s.HasChildProcess(),
		Recovered:       s.recovered,
	}
}

// classifyCreateError maps a session creation error to the appropriate HTTP
// error response. This is the single place where the decision "which error code
// does this creation failure produce?" is made.
func classifyCreateError(err error) appError {
	switch {
	case errors.Is(err, ErrSessionLimitReached):
		return errorCatalog["session_limit_reached"]
	case errors.Is(err, ErrBackendUnavailable):
		return errorCatalog["backend_unavailable"]
	case errors.Is(err, ErrBackendUnknown):
		return errorCatalog["backend_unknown"]
	case errors.Is(err, ErrPTYSpawnFailed):
		return errorCatalog["pty_spawn_failed"]
	default:
		ae := errorCatalog["internal_error"]
		ae.Message = "An unexpected error occurred while creating the session. Please try again."
		return ae
	}
}

// handleCreateSession creates a new terminal session.
// POST /api/v1/sessions
// Replay-safe: if the client includes an X-Idempotency-Key header, repeated
// requests with the same key return the cached response instead of creating
// duplicate sessions. Keys are valid for 5 minutes.
// [REQ:P0-002a] PTY Session Backend
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	reqID := getRequestID(r)

	// Check idempotency key — if this is a retry, return the cached response.
	idempotencyKey := r.Header.Get("X-Idempotency-Key")
	if idempotencyKey != "" {
		if cached, ok := s.idempotency.Get(idempotencyKey); ok {
			log.Printf("create-session [%s]: idempotency hit for key %q, returning cached session %s", reqID, idempotencyKey, cached.ID)
			writeJSON(w, http.StatusCreated, cached)
			return
		}
	}

	var req CreateSessionRequest
	if !decodeJSON(w, r, &req) {
		log.Printf("create-session [%s]: malformed JSON body", reqID)
		return
	}

	// Resolve policy from request or defaults
	var policy *ExpirationPolicy
	if req.Policy != nil {
		p := ExpirationPolicy{
			Mode:     PolicyMode(req.Policy.Mode),
			Duration: req.Policy.Duration,
		}
		if err := ValidatePolicy(p); err != nil {
			writeCatalogError(w, "invalid_policy", err.Error())
			return
		}
		policy = &p
	}

	sess, err := s.sessions.Create(req.Shell, uint16(req.Cols), uint16(req.Rows), BackendID(req.Backend), policy)
	if err != nil {
		log.Printf("create-session [%s]: %v", reqID, err)
		writeAppError(w, classifyCreateError(err))
		return
	}

	// [REQ:P1-004a] Emit session lifecycle event
	s.events.Emit(EventSessionCreated, sess.ID, map[string]string{
		"shell":   sess.Shell,
		"cols":    fmt.Sprintf("%d", sess.Cols),
		"rows":    fmt.Sprintf("%d", sess.Rows),
		"backend": string(sess.Backend),
	})
	s.metrics.SessionsCreated.Add(1)
	s.metrics.ActiveSessions.Add(1)

	resp := sessionToResponse(sess)

	// Cache the response for idempotent replays
	if idempotencyKey != "" {
		s.idempotency.Set(idempotencyKey, resp)
	}

	writeJSON(w, http.StatusCreated, resp)
}

// handleListSessions returns all active sessions.
// GET /api/v1/sessions
// [REQ:P0-003a] Session Persistence Store
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.sessions.List()
	result := make([]SessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, sessionToResponse(sess))
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGetSession returns a single session.
// GET /api/v1/sessions/{id}
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	writeJSON(w, http.StatusOK, sessionToResponse(sess))
}

// handleDeleteSession terminates a session.
// DELETE /api/v1/sessions/{id}
// Idempotent: returns 204 even if the session was already deleted, so that
// retries and replays are safe. Events and metrics are only recorded when a
// session is actually removed.
func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.sessions.Delete(id); err == nil {
		if s.conversations != nil {
			s.conversations.DeleteSession(id)
		}
		if s.codexCheckpointStore != nil {
			_ = s.codexCheckpointStore.DeleteSession(id)
		}
		// [REQ:P1-004a] Emit session lifecycle event — only on actual deletion
		s.events.Emit(EventSessionDeleted, id, nil)
		s.metrics.SessionsDeleted.Add(1)
		s.metrics.ActiveSessions.Add(-1)
	}
	// 204 regardless: the post-condition "session does not exist" is satisfied
	w.WriteHeader(http.StatusNoContent)
}

// --- Session Policy HTTP Handlers ---
// These handlers operate on the /sessions/{id}/policy sub-resource.
// Domain logic (validation, TTL resolution, sweeper) lives in session_policy.go.

// PolicyResponse is the JSON shape for policy endpoints.
type PolicyResponse struct {
	SessionID string           `json:"session_id"`
	Policy    ExpirationPolicy `json:"policy"`
	ExpiresAt *string          `json:"expires_at,omitempty"`
	TTL       *float64         `json:"ttl_seconds,omitempty"`
}

// buildPolicyResponse constructs a PolicyResponse with computed TTL and expiry
// fields. This is the single place where the decision "does this policy have a
// finite lifetime?" is made and translated into response fields.
func buildPolicyResponse(sess *Session, policy ExpirationPolicy) PolicyResponse {
	resp := PolicyResponse{
		SessionID: sess.ID,
		Policy:    policy,
	}
	ttl := ResolveTTL(policy)
	if ttl > 0 {
		expiresAt := sess.CreatedAt.Add(ttl)
		expiresStr := expiresAt.UTC().Format(time.RFC3339)
		resp.ExpiresAt = &expiresStr
		remaining := time.Until(expiresAt).Seconds()
		if remaining < 0 {
			remaining = 0
		}
		resp.TTL = &remaining
	}
	return resp
}

// handleGetPolicy returns the current expiration policy for a session.
// GET /api/v1/sessions/{id}/policy
// [REQ:P1-001a] Expiration Policy Engine
func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	writeJSON(w, http.StatusOK, buildPolicyResponse(sess, sess.GetPolicy()))
}

// UpdatePolicyRequest is the JSON body for updating a session's expiration policy.
type UpdatePolicyRequest struct {
	Mode     string `json:"mode"`
	Duration string `json:"duration,omitempty"`
}

// handleUpdatePolicy sets the expiration policy for a session.
// PUT /api/v1/sessions/{id}/policy
// [REQ:P1-001a] Expiration Policy Engine
func (s *Server) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	var req UpdatePolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	policy := ExpirationPolicy{
		Mode:     PolicyMode(req.Mode),
		Duration: req.Duration,
	}

	if err := ValidatePolicy(policy); err != nil {
		writeCatalogError(w, "invalid_policy", err.Error())
		return
	}

	// Only emit an event if the policy actually changed, so that replaying
	// the same PUT is a no-op for the event log and audit trail.
	oldPolicy := sess.GetPolicy()
	sess.SetPolicy(policy)

	// Persist policy update in metadata store
	if s.sessionStore != nil {
		_ = s.sessionStore.UpdatePolicy(sess.ID, policy)
	}

	if oldPolicy.Mode != policy.Mode || oldPolicy.Duration != policy.Duration {
		s.events.Emit(EventSessionPolicyUpdate, sess.ID, map[string]string{
			"mode":     req.Mode,
			"duration": req.Duration,
		})
	}

	writeJSON(w, http.StatusOK, buildPolicyResponse(sess, policy))
}
