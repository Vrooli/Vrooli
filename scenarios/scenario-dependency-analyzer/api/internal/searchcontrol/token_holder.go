package searchcontrol

import "sync"

// TokenStore is the in-memory home for the PER-PROVIDER control tokens search-hub
// mints for SDA's leaves at self-registration. SDA federates THREE providers
// (.dependencies, .scenarios, .resources), each with its OWN token in search-hub
// — a single shared token would present one leaf's secret for another and be
// rejected as a mismatch. The registration goroutine calls Set(providerID, token)
// when the hub echoes each provider's token; the control gate reads them via
// Get / Match. A restart loses them (memory only) and the next boot's
// re-registration re-acquires each — search-hub persists the authoritative copy.
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]string // provider_id -> control token
}

// NewTokenStore returns an empty store (no tokens until Set).
func NewTokenStore() *TokenStore { return &TokenStore{tokens: map[string]string{}} }

// Set stores one provider's control token (called from the registration callback).
func (s *TokenStore) Set(providerID, token string) {
	s.mu.Lock()
	if s.tokens == nil {
		s.tokens = map[string]string{}
	}
	s.tokens[providerID] = token
	s.mu.Unlock()
}

// Get returns the cached token for providerID ("" until Set runs for it). Used
// by the registration ControlToken callback to present the right per-provider
// token (an empty token is accepted by search-hub on re-registration).
func (s *TokenStore) Get(providerID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens[providerID]
}

// Match reports whether presented equals any cached provider token (non-empty).
// Used by scenario-scoped control RPCs (Reindex/status/cancel) that carry no
// provider_id: a caller holding any of this scenario's leaf tokens is authorized.
func (s *TokenStore) Match(presented string) bool {
	if presented == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tokens {
		if t != "" && constantTimeEqual(presented, t) {
			return true
		}
	}
	return false
}
