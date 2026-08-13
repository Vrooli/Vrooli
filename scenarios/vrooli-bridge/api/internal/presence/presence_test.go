package presence_test

import (
	"testing"
	"time"

	"vrooli-bridge/internal/compat"
	"vrooli-bridge/internal/presence"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
)

func newHub() *presence.Hub {
	return presence.NewHub(scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)))
}

// [REQ:BRG-P0-003] A node is offline until it dials out (Connect) and flips
// back offline when the connection closes (disconnect).
func TestHub_OnlineOnConnectOfflineOnClose(t *testing.T) {
	h := newHub()
	require.False(t, h.IsOnline("n1"))

	conn := h.Connect("n1")
	require.True(t, h.IsOnline("n1"))
	require.Equal(t, []string{"n1"}, h.OnlineNodes())

	conn.Close()
	require.False(t, h.IsOnline("n1"))
	require.Empty(t, h.OnlineNodes())
}

// [REQ:BRG-P0-003] A reconnect overlap never flickers the node offline: while
// two connections are held, closing one keeps the node online.
func TestHub_ReconnectOverlapStaysOnline(t *testing.T) {
	h := newHub()
	first := h.Connect("n1")
	second := h.Connect("n1")
	require.True(t, h.IsOnline("n1"))

	first.Close()
	require.True(t, h.IsOnline("n1"), "still online while the second connection is held")

	second.Close()
	require.False(t, h.IsOnline("n1"))
}

// Close is idempotent (the channel handler defers it).
func TestHub_CloseIsIdempotent(t *testing.T) {
	h := newHub()
	conn := h.Connect("n1")
	conn.Close()
	conn.Close() // must not underflow the count for any other connection
	require.False(t, h.IsOnline("n1"))

	other := h.Connect("n1")
	require.True(t, h.IsOnline("n1"))
	other.Close()
}

// [REQ:BRG-P0-003] Heartbeat stores the self-reported readiness and stamps
// ReportedAt from the clock when unset.
func TestHub_HeartbeatStoresHealth(t *testing.T) {
	h := newHub()
	_, ok := h.Health("n1")
	require.False(t, ok)

	h.Heartbeat("n1", presence.HealthSnapshot{
		ToolchainPresent:   true,
		DiskHeadroomBytes:  10 << 30,
		ContainerRuntimeUp: true,
		Details:            map[string]string{"go": "1.25.0"},
	})

	snap, ok := h.Health("n1")
	require.True(t, ok)
	require.True(t, snap.Ready())
	require.Equal(t, int64(10<<30), snap.DiskHeadroomBytes)
	require.Equal(t, "1.25.0", snap.Details["go"])
	require.False(t, snap.ReportedAt.IsZero(), "ReportedAt stamped from the clock")
}

func TestHub_PresenceOverlaysHealthForOnlineNodes(t *testing.T) {
	h := newHub()
	conn := h.Connect("n1")
	defer conn.Close()
	h.Heartbeat("n1", presence.HealthSnapshot{ToolchainPresent: true})

	// An offline node with stale health is not reported.
	h.Heartbeat("ghost", presence.HealthSnapshot{ToolchainPresent: true})

	snap := h.Presence()
	require.Len(t, snap, 1)
	require.Equal(t, "n1", snap[0].NodeID)
	require.True(t, snap[0].Online)
	require.True(t, snap[0].HasHealth)
	require.True(t, snap[0].Health.Ready())
}

func TestHub_HeartbeatStalenessMakesHalfOpenChannelUndispatchable(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	h := presence.NewHub(clk, presence.WithHeartbeatStaleAfter(45*time.Second))
	conn := h.Connect("n1")
	defer conn.Close()
	h.Heartbeat("n1", presence.HealthSnapshot{ToolchainPresent: true})
	require.True(t, h.IsOnline("n1"))
	require.True(t, h.Dispatchable("n1"))

	clk.Advance(46 * time.Second)
	require.False(t, h.IsOnline("n1"), "a half-open socket cannot remain online forever")
	require.False(t, h.Dispatchable("n1"))
	require.Empty(t, h.Presence())
}

func TestHub_ReadinessFactsFlipIndependently(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	h := presence.NewHub(clk)
	conn := h.Connect("n1")
	facts := h.Readiness("n1")
	require.True(t, facts.ChannelHeld)
	require.False(t, facts.HeartbeatFresh)
	require.True(t, facts.ProtocolCompatible)
	require.True(t, facts.Dispatchable, "dispatchability remains the independent channel+protocol gate until a heartbeat becomes stale")

	h.Heartbeat("n1", presence.HealthSnapshot{ToolchainPresent: true})
	h.SetCompatibility("n1", compat.StatusNeedsUpdate)
	facts = h.Readiness("n1")
	require.True(t, facts.HeartbeatFresh)
	require.True(t, facts.ChannelHeld)
	require.False(t, facts.ProtocolCompatible)
	require.False(t, facts.Dispatchable)

	h.SetCompatibility("n1", compat.StatusOK)
	conn.Close()
	facts = h.Readiness("n1")
	require.True(t, facts.HeartbeatFresh, "health remains independently observable after channel close")
	require.False(t, facts.ChannelHeld)
	require.True(t, facts.ProtocolCompatible)
	require.False(t, facts.Dispatchable)
}

// HealthSnapshot.Ready gates on the toolchain.
func TestHealthSnapshot_Ready(t *testing.T) {
	require.True(t, presence.HealthSnapshot{ToolchainPresent: true}.Ready())
	require.False(t, presence.HealthSnapshot{ToolchainPresent: false}.Ready())
}

// [REQ:BRG-P0-002] Disconnect force-closes a node's live channel(s) and clears
// its presence + health — the in-memory half of atomic revocation. The held
// SSE handler learns via Conn.Done().
func TestHub_DisconnectSeversLiveChannel(t *testing.T) {
	h := newHub()
	conn := h.Connect("n1")
	h.Heartbeat("n1", presence.HealthSnapshot{ToolchainPresent: true})
	require.True(t, h.IsOnline("n1"))

	dropped := h.Disconnect("n1")
	require.Equal(t, 1, dropped)
	require.False(t, h.IsOnline("n1"), "node reads offline immediately")

	// The held connection is signalled to stop.
	select {
	case <-conn.Done():
	default:
		t.Fatal("Conn.Done() was not closed by Disconnect")
	}

	// Health is cleared; presence omits the node.
	_, ok := h.Health("n1")
	require.False(t, ok)
	require.Empty(t, h.Presence())

	// Idempotent: disconnecting an already-gone node is a no-op.
	require.Equal(t, 0, h.Disconnect("n1"))
	// Close after Disconnect must not panic (double-close guarded by once).
	conn.Close()
}
