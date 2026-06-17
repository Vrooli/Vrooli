// Package realtime is the presence + server-push layer of Device Sync Hub. A
// trusted device opens one long-lived SSE stream; the hub tracks it as a live
// presence connection and fans server-side events (item-arrived, item-deleted,
// presence-changed, pairing-requested) out to the right devices in near-real-time.
//
// Transport choice: SSE, not WebSocket. Every client *action* already rides a
// Connect RPC, so the only thing the server needs is a one-directional
// server->client push channel — exactly what SSE is, with zero extra
// dependencies (pure net/http + http.Flusher). See docs/concepts/DECISIONS.
//
// The hub is deliberately ignorant of the transfer and devices domains: its
// public surface speaks in plain ids (owner, device, item). Thin adapters at
// the handler layer satisfy transfer.Notifier and the devices pairing hook by
// calling Emit*; that keeps this package a leaf with no domain imports and no
// import cycle.
package realtime

import (
	"sync"
	"time"

	"device-sync-hub/internal/clock"
)

// EventKind tags a server-pushed event. Mapped to the proto EventType enum at
// the SSE handler edge.
type EventKind int

const (
	// EventItemArrived — a new item is available to the receiving device.
	EventItemArrived EventKind = iota + 1
	// EventItemDeleted — an item was deleted or purged.
	EventItemDeleted
	// EventPresenceChanged — the set of online devices changed.
	EventPresenceChanged
	// EventPairingRequested — a device is requesting to pair (approve banner).
	EventPairingRequested
)

// DevicePresence is one device's online state in a presence snapshot.
type DevicePresence struct {
	DeviceID string
	Online   bool
}

// PairingInfo describes the device awaiting approval.
type PairingInfo struct {
	DeviceID string
	Name     string
	Kind     string
}

// Event is one server-pushed message. Only the fields relevant to Kind are set.
type Event struct {
	Kind EventKind
	At   time.Time
	// ItemID / TargetDeviceID are set for item events.
	ItemID         string
	TargetDeviceID string
	// Presence is the full current online snapshot, set for EventPresenceChanged.
	Presence []DevicePresence
	// Pairing is set for EventPairingRequested.
	Pairing *PairingInfo
}

// subChanBuffer bounds each subscriber's queue. A slow client that can't keep
// up has events dropped (non-blocking send) rather than stalling fan-out — it
// recovers correctness by re-listing items / devices on its next interaction.
const subChanBuffer = 32

type subscriber struct {
	ownerID  string
	deviceID string
	ch       chan Event
}

// Hub tracks live presence connections and fans events to subscribers. Safe for
// concurrent use. Construct once in main.go and share across handlers.
type Hub struct {
	clock clock.Clock

	mu   sync.Mutex
	subs map[string]map[*subscriber]struct{} // keyed by ownerID
}

// NewHub constructs an empty Hub. clk supplies event timestamps so tests pin them.
func NewHub(clk clock.Clock) *Hub {
	if clk == nil {
		clk = clock.System{}
	}
	return &Hub{clock: clk, subs: make(map[string]map[*subscriber]struct{})}
}

// Subscription is a live SSE connection's handle. The handler ranges over
// Events() until the request context is cancelled, then calls Close exactly once.
type Subscription struct {
	hub *Hub
	sub *subscriber
}

// Events returns the channel the handler streams to the client.
func (s *Subscription) Events() <-chan Event { return s.sub.ch }

// Close removes the subscription and updates presence. Idempotent-safe to call
// once per Subscribe; the handler defers it.
func (s *Subscription) Close() { s.hub.unsubscribe(s.sub) }

// Subscribe registers a new live connection for (ownerID, deviceID), marks the
// device online, and broadcasts the updated presence snapshot to the owner's
// devices. The returned Subscription's channel is immediately primed with a
// presence snapshot so a just-connected client renders state without waiting.
func (h *Hub) Subscribe(ownerID, deviceID string) *Subscription {
	sub := &subscriber{ownerID: ownerID, deviceID: deviceID, ch: make(chan Event, subChanBuffer)}
	h.mu.Lock()
	if h.subs[ownerID] == nil {
		h.subs[ownerID] = make(map[*subscriber]struct{})
	}
	h.subs[ownerID][sub] = struct{}{}
	snapshot := h.presenceLocked(ownerID)
	h.mu.Unlock()

	// Prime this subscriber with the current snapshot, then tell everyone the
	// set changed (this device just came online).
	sub.ch <- Event{Kind: EventPresenceChanged, At: h.now(), Presence: snapshot}
	h.broadcastPresence(ownerID)
	return &Subscription{hub: h, sub: sub}
}

func (h *Hub) unsubscribe(sub *subscriber) {
	h.mu.Lock()
	if set := h.subs[sub.ownerID]; set != nil {
		if _, ok := set[sub]; ok {
			delete(set, sub)
			close(sub.ch)
			if len(set) == 0 {
				delete(h.subs, sub.ownerID)
			}
		}
	}
	h.mu.Unlock()
	h.broadcastPresence(sub.ownerID)
}

// OnlineDevices returns the owner's currently-online device ids. Used by the
// devices read path to overlay live presence onto the stored device list.
func (h *Hub) OnlineDevices(ownerID string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	seen := make(map[string]struct{})
	var out []string
	for sub := range h.subs[ownerID] {
		if sub.deviceID == "" {
			continue
		}
		if _, dup := seen[sub.deviceID]; dup {
			continue
		}
		seen[sub.deviceID] = struct{}{}
		out = append(out, sub.deviceID)
	}
	return out
}

// EmitItemArrived pushes an item-arrived event to the devices that may pull the
// item: every online device of the owner for a broadcast item, or just the
// target (and origin) device for a directed item.
func (h *Hub) EmitItemArrived(ownerID, itemID, targetDeviceID, originDeviceID string) {
	h.deliver(ownerID, targetDeviceID, originDeviceID, Event{
		Kind: EventItemArrived, At: h.now(), ItemID: itemID, TargetDeviceID: targetDeviceID,
	})
}

// EmitItemDeleted pushes an item-deleted event with the same ACL as arrival.
func (h *Hub) EmitItemDeleted(ownerID, itemID, targetDeviceID, originDeviceID string) {
	h.deliver(ownerID, targetDeviceID, originDeviceID, Event{
		Kind: EventItemDeleted, At: h.now(), ItemID: itemID, TargetDeviceID: targetDeviceID,
	})
}

// EmitPairingRequested pushes the approve/reject banner to every online device
// of the owner (any trusted device may approve a pending request).
func (h *Hub) EmitPairingRequested(ownerID string, info PairingInfo) {
	h.deliver(ownerID, "", "", Event{
		Kind: EventPairingRequested, At: h.now(), Pairing: &info,
	})
}

// deliver fans evt to the owner's subscribers honoring the item delivery ACL:
// when target is empty the event goes to all of the owner's devices; otherwise
// only to subscribers whose device is the target or the origin.
func (h *Hub) deliver(ownerID, targetDeviceID, originDeviceID string, evt Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subs[ownerID] {
		if targetDeviceID != "" && sub.deviceID != targetDeviceID && sub.deviceID != originDeviceID {
			continue
		}
		send(sub, evt)
	}
}

func (h *Hub) broadcastPresence(ownerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	snapshot := h.presenceLocked(ownerID)
	evt := Event{Kind: EventPresenceChanged, At: h.now(), Presence: snapshot}
	for sub := range h.subs[ownerID] {
		send(sub, evt)
	}
}

// presenceLocked computes the owner's online device snapshot. Caller holds mu.
func (h *Hub) presenceLocked(ownerID string) []DevicePresence {
	seen := make(map[string]struct{})
	var out []DevicePresence
	for sub := range h.subs[ownerID] {
		if sub.deviceID == "" {
			continue
		}
		if _, dup := seen[sub.deviceID]; dup {
			continue
		}
		seen[sub.deviceID] = struct{}{}
		out = append(out, DevicePresence{DeviceID: sub.deviceID, Online: true})
	}
	return out
}

// send is a non-blocking enqueue: a full subscriber queue drops the event
// rather than stalling the whole fan-out under the hub lock.
func send(sub *subscriber, evt Event) {
	select {
	case sub.ch <- evt:
	default:
	}
}

func (h *Hub) now() time.Time { return h.clock.Now().UTC() }
