// DOC: docs/concepts/ARCHITECTURE.md#data-flow
// DOC: docs/internal/ERROR_SEMANTICS.md
// DOC: docs/internal/INVARIANTS.md

package main

// session_handlers.go holds the in-memory types that the sessions adapter
// (sessions_adapter.go) and a few legacy callers still depend on:
//   - idempotencyCache: bounded TTL cache keyed by X-Idempotency-Key that
//     replays session creations safely.
//   - SessionResponse / sessionToResponse: the historical JSON shape, kept
//     because the idempotency cache stores it and several tests round-trip
//     through it. The wire shape itself lives in proto now.
//
// HTTP handlers for sessions have moved to handlers/sessions; routes are
// mounted by sessionsH.Module() in main.go.

import (
	"sync"
	"time"
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
		delete(c.entries, key)
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
	if len(c.entries) > 100 {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
	}
}

// normalizeAgentType maps a free-form string to one of the closed-set
// AgentType values. Unrecognized inputs become AgentTypeNone (rather than
// erroring) so a future client that knows about a new agent kind can roll
// forward without breaking older API builds.
func normalizeAgentType(s string) AgentType {
	switch AgentType(s) {
	case AgentTypeCodex:
		return AgentTypeCodex
	case AgentTypeClaude:
		return AgentTypeClaude
	case AgentTypeNone:
		return AgentTypeNone
	default:
		return AgentTypeNone
	}
}

// SessionResponse is the JSON representation of a session retained for the
// idempotency cache and tests. The on-wire shape is now defined by the
// SessionsService proto.
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

// sessionToResponse converts an internal Session to the historical JSON
// shape. [REQ:P1-001a] Includes expiration policy in response.
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
