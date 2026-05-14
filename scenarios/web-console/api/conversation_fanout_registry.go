package main

import "sync"

// ConversationFanoutRegistry owns the per-session conversation event
// fan-outs. SessionManager invokes Create/Delete via lifecycle hooks; readers
// (conversation router, terminal WS, hook handlers) look up the fanout for a
// session by id.
//
// Lives in package main so SessionManager (which moves to its own sub-package)
// can stay decoupled from ConversationEvent.
type ConversationFanoutRegistry struct {
	mu      sync.RWMutex
	fanouts map[string]*ConversationFanout
}

// NewConversationFanoutRegistry returns an empty registry.
func NewConversationFanoutRegistry() *ConversationFanoutRegistry {
	return &ConversationFanoutRegistry{fanouts: make(map[string]*ConversationFanout)}
}

// Create installs a fresh fanout for the given session id.
func (r *ConversationFanoutRegistry) Create(sessionID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.fanouts[sessionID] = NewConversationFanout(sessionID)
	r.mu.Unlock()
}

// Delete removes the fanout for the given session id, if any.
func (r *ConversationFanoutRegistry) Delete(sessionID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	delete(r.fanouts, sessionID)
	r.mu.Unlock()
}

// Get returns the fanout for the given session id, or nil if unknown or if
// the registry itself is nil (some test fixtures construct a Server without
// wiring fanouts).
func (r *ConversationFanoutRegistry) Get(sessionID string) *ConversationFanout {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.fanouts[sessionID]
}

// AttachToManager installs lifecycle hooks on sm so the registry stays in
// sync with the session set. Returns the registry for fluent wiring.
func (r *ConversationFanoutRegistry) AttachToManager(sm *SessionManager) *ConversationFanoutRegistry {
	sm.SetLifecycleHooks(r.Create, r.Delete)
	return r
}
