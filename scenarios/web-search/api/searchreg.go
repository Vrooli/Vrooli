package main

import "sync"

// tokenHolder caches the per-provider control tokens search-hub mints on
// registration, so re-registration can echo the token back as the ownership
// proof that stops another actor from hijacking a provider_id. It is in-memory:
// on restart it starts empty and the hub treats the next registration as
// first-contact (and re-mints). web-search declares no control endpoints
// (reindex/config), so the tokens are used only for the ownership echo.
type tokenHolder struct {
	mu     sync.Mutex
	tokens map[string]string
}

func newTokenHolder() *tokenHolder {
	return &tokenHolder{tokens: make(map[string]string)}
}

func (h *tokenHolder) set(providerID, token string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tokens[providerID] = token
}

func (h *tokenHolder) get(providerID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tokens[providerID]
}
