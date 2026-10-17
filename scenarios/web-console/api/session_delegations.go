package main

import (
	"sync"
	"time"
)

// sessionDelegations holds short-lived, session-scoped human-owner delegation
// tokens minted at a verified-human session create. It is deliberately
// IN-MEMORY and never persisted: a delegation is a short-lived credential, and
// durable session state must not carry credentials. The env builder reads it by
// session id when it assembles a PTY's environment, so a coding agent launched
// in that session inherits the delegation and can forward it on attach. A nil
// receiver is safe so test servers that never mint behave as before.
type sessionDelegations struct {
	mu    sync.Mutex
	items map[string]sessionDelegation
}

type sessionDelegation struct {
	token     string
	expiresAt time.Time
}

func newSessionDelegations() *sessionDelegations {
	return &sessionDelegations{items: map[string]sessionDelegation{}}
}

// Put records the delegation for a session. An empty token clears any prior one.
func (d *sessionDelegations) Put(sessionID, token string, expiresAt time.Time) {
	if d == nil || sessionID == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if token == "" {
		delete(d.items, sessionID)
		return
	}
	d.items[sessionID] = sessionDelegation{token: token, expiresAt: expiresAt}
}

// Lookup returns the delegation token for a session when one is present and not
// expired. An expired entry is dropped and reported absent, so a stale
// delegation never reaches a freshly launched agent.
func (d *sessionDelegations) Lookup(sessionID string, now time.Time) (string, bool) {
	if d == nil || sessionID == "" {
		return "", false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	entry, ok := d.items[sessionID]
	if !ok {
		return "", false
	}
	if !entry.expiresAt.IsZero() && !now.Before(entry.expiresAt) {
		delete(d.items, sessionID)
		return "", false
	}
	return entry.token, true
}

// Delete drops the delegation for a session (e.g. when the session ends).
func (d *sessionDelegations) Delete(sessionID string) {
	if d == nil || sessionID == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.items, sessionID)
}

// activeSessionDelegations is the process-wide holder the env builder reads and
// the verified-human create path writes. It is a package var because
// defaultSessionEnv is a free function wired into the session manager before any
// request is served; main.go initializes it during startup.
var activeSessionDelegations = newSessionDelegations()
