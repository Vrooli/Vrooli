package graph

import "time"

// Dispatch implements EventDispatcher by forwarding events to a Broadcaster.
type Dispatch struct {
	broadcaster Broadcaster
}

// NewDispatch creates a Dispatch that forwards events to the given Broadcaster.
func NewDispatch(b Broadcaster) *Dispatch {
	return &Dispatch{broadcaster: b}
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

// NewWSMessage creates a timestamped WebSocket message envelope.
func NewWSMessage(eventType string, data any) WSMessage {
	return WSMessage{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}
