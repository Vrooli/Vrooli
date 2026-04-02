package graph

import (
	"time"
)

// Dispatch implements EventDispatcher by forwarding events to a Broadcaster.
type Dispatch struct {
	broadcaster Broadcaster
	invalidator CacheInvalidator
}

// NewDispatch creates a Dispatch that forwards events to the given Broadcaster.
func NewDispatch(b Broadcaster, invalidator CacheInvalidator) *Dispatch {
	return &Dispatch{
		broadcaster: b,
		invalidator: invalidator,
	}
}

// DispatchNodeUpdate emits a node-update event.
func (d *Dispatch) DispatchNodeUpdate(nodeType, nodeID string, data any) {
	if d.broadcaster == nil {
		return
	}
	d.broadcaster.BroadcastUpdate(WSNodeUpdate, Node{
		ID:   nodeID,
		Type: nodeType,
		Data: data,
	})
}

// DispatchNodeAdd emits a node-add event.
func (d *Dispatch) DispatchNodeAdd(node Node) {
	if d.broadcaster == nil {
		return
	}
	d.broadcaster.BroadcastUpdate(WSNodeAdd, node)
}

// DispatchEdgeChange emits an edge-add or edge-remove event.
func (d *Dispatch) DispatchEdgeChange(action string, edge Edge) {
	if d.broadcaster == nil {
		return
	}
	d.broadcaster.BroadcastUpdate(action, edge)
}

// DispatchInvalidate clears cached lenses and broadcasts an invalidate message.
func (d *Dispatch) DispatchInvalidate(lenses ...string) {
	normalized := normalizeLensStrings(lenses)
	if len(normalized) == 0 {
		return
	}

	if d.invalidator != nil {
		d.invalidator.Invalidate(normalized...)
	}
	if d.broadcaster == nil {
		return
	}

	d.broadcaster.BroadcastUpdate(WSInvalidate, InvalidationPayload{
		Lenses: normalized,
	})
}

// DispatchInvalidateWithFocus clears cached flow/operations projections for a
// specific focus node and broadcasts a focus-aware invalidation message.
func (d *Dispatch) DispatchInvalidateWithFocus(focusNodeID string) {
	if focusNodeID == "" {
		return
	}
	if d.invalidator != nil {
		d.invalidator.InvalidateFocus(focusNodeID)
	}
	if d.broadcaster == nil {
		return
	}
	d.broadcaster.BroadcastUpdate(WSInvalidate, InvalidationPayload{
		Lenses:      []Lens{LensOperations},
		FocusNodeID: focusNodeID,
	})
}

func normalizeLensStrings(values []string) []Lens {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[Lens]struct{}, len(values))
	result := make([]Lens, 0, len(values))
	for _, raw := range values {
		lens := Lens(raw)
		if !ValidateLens(lens) {
			continue
		}
		if _, exists := seen[lens]; exists {
			continue
		}
		seen[lens] = struct{}{}
		result = append(result, lens)
	}
	return result
}

// NewWSMessage creates a timestamped WebSocket message envelope.
func NewWSMessage(eventType string, data any) WSMessage {
	return WSMessage{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}
