package search

import "sync"

// TokenHolder is the in-memory home for the control token search-hub mints for
// this provider at self-registration. The registration goroutine calls Set when
// the hub echoes the token; the search handler's override gate reads it via Get
// on every request. A restart loses it (memory only) and the next boot's
// re-registration re-acquires it — search-hub persists the authoritative copy.
//
// It exists so the boot wiring can hand the handler a stable Token func
// (holder.Get) before the token is known: Get returns "" until Set runs, and the
// gate treats an empty token as "deny", so the override channel is simply closed
// until registration completes.
type TokenHolder struct {
	mu    sync.RWMutex
	token string
}

// NewTokenHolder returns an empty holder (no token until Set).
func NewTokenHolder() *TokenHolder { return &TokenHolder{} }

// Set stores the control token (called from the registration callback).
func (h *TokenHolder) Set(token string) {
	h.mu.Lock()
	h.token = token
	h.mu.Unlock()
}

// Get returns the current token ("" until Set runs).
func (h *TokenHolder) Get() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.token
}
