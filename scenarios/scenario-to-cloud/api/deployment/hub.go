package deployment

import (
	"fmt"
	"sync"
	"time"
)

// Broadcaster is the interface for broadcasting progress events.
// This abstraction allows RunVPSDeployWithProgress to work with both the real Hub
// and a no-op implementation for callers that don't need progress tracking.
type Broadcaster interface {
	Broadcast(deploymentID string, event Event)
}

// Hub manages SSE subscriptions for deployment progress.
// It allows multiple clients to subscribe to progress updates for a deployment.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string][]chan Event),
	}
}

// Subscribe creates a new subscription channel for a deployment.
// Returns a channel that will receive progress events.
func (h *Hub) Subscribe(deploymentID string) chan Event {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan Event, 100) // Buffer for handling slow clients
	h.subscribers[deploymentID] = append(h.subscribers[deploymentID], ch)
	return ch
}

// Unsubscribe removes a subscription channel for a deployment.
func (h *Hub) Unsubscribe(deploymentID string, ch chan Event) {
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
func (h *Hub) Broadcast(deploymentID string, event Event) {
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

// BroadcastInvestigation sends an investigation progress event to all subscribers.
// This method satisfies the investigation.Broadcaster interface.
func (h *Hub) BroadcastInvestigation(deploymentID, invID, eventType string, progress float64, message string) {
	h.Broadcast(deploymentID, Event{
		Type:      eventType,
		Step:      "investigation",
		StepTitle: "Investigation",
		Progress:  progress,
		Message:   fmt.Sprintf("[%s] %s", invID[:8], message),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// HasSubscribers returns true if there are active subscribers for a deployment.
func (h *Hub) HasSubscribers(deploymentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers[deploymentID]) > 0
}

// CleanupDeployment removes all subscribers for a completed deployment.
func (h *Hub) CleanupDeployment(deploymentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, ch := range h.subscribers[deploymentID] {
		close(ch)
	}
	delete(h.subscribers, deploymentID)
}

// GetEmitter returns an Emitter that broadcasts to the hub.
// This is used by the deployment pipeline to emit events.
func (h *Hub) GetEmitter(deploymentID string) Emitter {
	return &hubEmitter{
		hub:          h,
		deploymentID: deploymentID,
	}
}

// hubEmitter implements Emitter by broadcasting to a Hub.
type hubEmitter struct {
	hub          *Hub
	deploymentID string
}

// Emit broadcasts the event to all subscribers.
func (e *hubEmitter) Emit(event Event) {
	e.hub.Broadcast(e.deploymentID, event)
}

// Close cleans up the deployment from the hub.
func (e *hubEmitter) Close() {
	// Don't cleanup here - clients may still be reading final events
	// The hub will be cleaned up when all clients disconnect
}
