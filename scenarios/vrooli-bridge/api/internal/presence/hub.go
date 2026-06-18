package presence

import (
	"sort"
	"sync"

	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/compat"
)

// Hub is the in-memory presence tracker. It is safe for concurrent use and
// constructed once in main.go, shared with the registry read path (it satisfies
// the registry handler's Presence seam via IsOnline) and the channel handler
// (which opens a Conn per dial-out connection and feeds heartbeats).
//
// A node is "online" while it holds at least one live channel connection. The
// Hub tracks the live *Conn set per node (not just a count) so an atomic
// revocation can force-close every held channel for a node (Disconnect). The
// set still tolerates a brief reconnect overlap — a node that reconnects before
// its old connection's Close lands never flickers offline.
type Hub struct {
	clock clock.Clock

	mu     sync.Mutex
	conns  map[string]map[*Conn]struct{} // nodeID -> live connections
	health map[string]HealthSnapshot     // nodeID -> latest self-reported health
	compat map[string]compat.Status      // nodeID -> protocol-compatibility verdict
}

// NewHub constructs an empty Hub.
func NewHub(clk clock.Clock) *Hub {
	return &Hub{
		clock:  clk,
		conns:  make(map[string]map[*Conn]struct{}),
		health: make(map[string]HealthSnapshot),
		compat: make(map[string]compat.Status),
	}
}

// pushBuffer bounds a connection's outbound frame queue. The control plane
// pushes job/provision/control frames here; the SSE handler drains them to the
// node. A wedged node whose buffer fills causes Push to report non-delivery
// (the dispatcher then aborts the run) rather than blocking the dispatch path.
const pushBuffer = 16

// Conn represents one held dial-out channel connection. Close marks the node
// one connection closer to offline; it is idempotent so the channel handler can
// defer it safely. Done is closed when the connection should end — either via
// Close (the node hung up) or Disconnect (the control plane severed it, e.g.
// atomic revocation); the SSE handler selects on it to stop holding the stream.
type Conn struct {
	hub    *Hub
	nodeID string

	done chan struct{}
	out  chan []byte
	once sync.Once
}

// Done returns a channel closed when this connection should terminate. The SSE
// handler selects on it so a server-side revocation drops the held stream
// immediately, not on the next keepalive.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Out returns the connection's outbound frame channel. The SSE handler selects
// on it and writes each payload to the node as an SSE `data:` event. Each
// payload is one already-serialised channel.ServerFrame (the proto translation
// happens at the handler/dispatch boundary so the presence domain stays
// proto-free).
func (c *Conn) Out() <-chan []byte { return c.out }

// Connect registers a new live connection for nodeID and returns its Conn. The
// node is online for as long as any Conn is open.
func (h *Hub) Connect(nodeID string) *Conn {
	c := &Conn{hub: h, nodeID: nodeID, done: make(chan struct{}), out: make(chan []byte, pushBuffer)}
	h.mu.Lock()
	set := h.conns[nodeID]
	if set == nil {
		set = make(map[*Conn]struct{})
		h.conns[nodeID] = set
	}
	set[c] = struct{}{}
	h.mu.Unlock()
	return c
}

// Close drops this connection, marking the node offline when its last
// connection closes. Idempotent.
func (c *Conn) Close() {
	c.once.Do(func() {
		close(c.done)
		h := c.hub
		h.mu.Lock()
		defer h.mu.Unlock()
		if set := h.conns[c.nodeID]; set != nil {
			delete(set, c)
			if len(set) == 0 {
				delete(h.conns, c.nodeID)
				// Health is retained until the node reconnects and re-reports.
			}
		}
	})
}

// Disconnect force-closes every live channel for nodeID and clears its presence
// + health. It is the in-memory half of atomic revocation: a revoked node's
// held SSE stream drops immediately (its Conn.Done fires) and it reads offline
// at once. Returns the number of connections dropped. Idempotent (a node with
// no live channel is a no-op).
func (h *Hub) Disconnect(nodeID string) int {
	h.mu.Lock()
	set := h.conns[nodeID]
	conns := make([]*Conn, 0, len(set))
	for c := range set {
		conns = append(conns, c)
	}
	delete(h.conns, nodeID)
	delete(h.health, nodeID)
	delete(h.compat, nodeID)
	h.mu.Unlock()

	for _, c := range conns {
		c.once.Do(func() { close(c.done) })
	}
	return len(conns)
}

// SetCompatibility records a node's protocol-compatibility verdict (computed
// from the version it reported when it dialed out). It is stored on the live
// layer; dispatch and the fleet roll read it via Compatibility/Dispatchable to
// exclude a version-drifted node from work rather than mis-drive it.
func (h *Hub) SetCompatibility(nodeID string, status compat.Status) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.compat[nodeID] = status
}

// Compatibility returns a node's recorded protocol-compatibility verdict. A node
// with no recorded verdict (never reported a version) reads as
// compat.StatusUnspecified, which is dispatchable (back-compat).
func (h *Hub) Compatibility(nodeID string) compat.Status {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.compat[nodeID]
}

// Dispatchable reports whether the node may currently receive WORK: it must be
// online AND its protocol verdict must not be a flagged (needs-update /
// incompatible) one. This is the seam dispatch and the fleet roll gate on.
func (h *Hub) Dispatchable(nodeID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.conns[nodeID]) == 0 {
		return false
	}
	return h.compat[nodeID].Dispatchable()
}

// IsOnline reports whether nodeID currently holds a dial-out channel. This is
// the method the registry handler's Presence seam calls.
func (h *Hub) IsOnline(nodeID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns[nodeID]) > 0
}

// Push enqueues an already-serialised ServerFrame payload to every live channel
// the node holds, returning the number of connections it reached. It is
// non-blocking: a connection whose outbound buffer is full is skipped (and not
// counted), so a single wedged node can never stall the dispatch path. A return
// of 0 means the node is offline or wedged — the caller (dispatch) treats that
// as non-delivery. This is the control-plane → node push half of the channel
// (JobPush in Phase 3, ProvisionCommand in Phase 4).
func (h *Hub) Push(nodeID string, payload []byte) int {
	h.mu.Lock()
	set := h.conns[nodeID]
	conns := make([]*Conn, 0, len(set))
	for c := range set {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	delivered := 0
	for _, c := range conns {
		select {
		case c.out <- payload:
			delivered++
		default:
			// Buffer full: the node is not draining; skip it.
		}
	}
	return delivered
}

// Heartbeat records the node's latest self-reported health. A heartbeat from a
// node with no open connection is still stored (the connection may be racing
// the first heartbeat), but only surfaces once the node is online.
func (h *Hub) Heartbeat(nodeID string, snap HealthSnapshot) {
	if snap.ReportedAt.IsZero() {
		snap.ReportedAt = h.clock.Now().UTC()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.health[nodeID] = snap
}

// Health returns the node's last-reported health and whether any has been
// reported.
func (h *Hub) Health(nodeID string) (HealthSnapshot, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	snap, ok := h.health[nodeID]
	return snap, ok
}

// OnlineNodes returns the ids of every currently-online node, sorted for
// deterministic output.
func (h *Hub) OnlineNodes() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.conns))
	for id := range h.conns {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Presence returns a point-in-time view of every online node with its health
// overlay. Offline nodes are omitted — durable identity comes from the
// registry; this is the ephemeral live layer.
func (h *Hub) Presence() []NodePresence {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]NodePresence, 0, len(h.conns))
	for id := range h.conns {
		np := NodePresence{NodeID: id, Online: true, Compatibility: h.compat[id]}
		if snap, ok := h.health[id]; ok {
			np.Health = snap
			np.HasHealth = true
		}
		out = append(out, np)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}
