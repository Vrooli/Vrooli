package main

import (
	"sync"
)

// ProgressHub manages SSE subscriptions for deployment progress.
// It allows multiple clients to subscribe to progress updates for a deployment.
type ProgressHub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan ProgressEvent
}

// NewProgressHub creates a new ProgressHub.
func NewProgressHub() *ProgressHub {
	return &ProgressHub{
		subscribers: make(map[string][]chan ProgressEvent),
	}
}

// Subscribe creates a new subscription channel for a deployment.
// Returns a channel that will receive progress events.
func (h *ProgressHub) Subscribe(deploymentID string) chan ProgressEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan ProgressEvent, 100) // Buffer for handling slow clients
	h.subscribers[deploymentID] = append(h.subscribers[deploymentID], ch)
	return ch
}

// Unsubscribe removes a subscription channel for a deployment.
func (h *ProgressHub) Unsubscribe(deploymentID string, ch chan ProgressEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs := h.subscribers[deploymentID]
	for i, sub := range subs {
		if sub == ch {
			// Remove from slice
			h.subscribers[deploymentID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}

	// Clean up empty subscription lists
	if len(h.subscribers[deploymentID]) == 0 {
		delete(h.subscribers, deploymentID)
	}
}

// Broadcast sends a progress event to all subscribers for a deployment.
func (h *ProgressHub) Broadcast(deploymentID string, event ProgressEvent) {
	h.mu.RLock()
	subs := h.subscribers[deploymentID]
	h.mu.RUnlock()

	for _, ch := range subs {
		// Non-blocking send to avoid blocking on slow clients
		select {
		case ch <- event:
		default:
			// Channel buffer full, skip this client
		}
	}
}

// HasSubscribers returns true if there are active subscribers for a deployment.
func (h *ProgressHub) HasSubscribers(deploymentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers[deploymentID]) > 0
}

// CleanupDeployment removes all subscribers for a completed deployment.
func (h *ProgressHub) CleanupDeployment(deploymentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, ch := range h.subscribers[deploymentID] {
		close(ch)
	}
	delete(h.subscribers, deploymentID)
}

// GetEmitter returns a ProgressEmitter that broadcasts to the hub.
// This is used by the deployment pipeline to emit events.
func (h *ProgressHub) GetEmitter(deploymentID string) ProgressEmitter {
	return &hubEmitter{
		hub:          h,
		deploymentID: deploymentID,
	}
}

// hubEmitter implements ProgressEmitter by broadcasting to a ProgressHub.
type hubEmitter struct {
	hub          *ProgressHub
	deploymentID string
}

// Emit broadcasts the event to all subscribers.
func (e *hubEmitter) Emit(event ProgressEvent) {
	e.hub.Broadcast(e.deploymentID, event)
}

// Close cleans up the deployment from the hub.
func (e *hubEmitter) Close() {
	// Don't cleanup here - clients may still be reading final events
	// The hub will be cleaned up when all clients disconnect
}
