package presence_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	channelH "vrooli-bridge/handlers/channel"
	"vrooli-bridge/internal/compat"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/testutil/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type noopLastSeen struct{}

func (noopLastSeen) TouchLastSeen(context.Context, string, time.Time) error { return nil }

// startChannelServer mounts the real channel module on an httptest server and
// returns it plus the shared hub. The server stands in for the control plane;
// the test client stands in for the node-agent.
func startChannelServer(t *testing.T) (*httptest.Server, *presence.Hub) {
	t.Helper()
	hub := presence.NewHub(mocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)))
	r := mux.NewRouter()
	// nil verifier → Phase-1 ?node= behaviour (mutual-auth enforcement is
	// exercised in handlers/channel/heartbeat_handler_test.go).
	channelH.Module(hub, noopLastSeen{}, nil, log.New(io.Discard, "", 0)).Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, hub
}

// [REQ:BRG-P0-003] The node establishes the channel purely OUTBOUND: it makes a
// GET to the control plane and holds the response stream. No listener is opened
// on the node side (the test client opens no port — it is the connection
// initiator, exactly the NAT/firewall-proof dial-out model). While the stream
// is held the node reads online; when it disconnects it reads offline.
func TestDialOut_FlipsPresenceOnlineThenOffline(t *testing.T) {
	srv, hub := startChannelServer(t)

	require.False(t, hub.IsOnline("n1"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/channel/events?node=n1", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Read the greeting so we know the handler has registered the connection.
	buf := make([]byte, 16)
	_, err = resp.Body.Read(buf)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return hub.IsOnline("n1") }, time.Second, 5*time.Millisecond,
		"node is online while the dial-out stream is held")

	// The node disconnects (here by cancelling the outbound request).
	cancel()
	_ = resp.Body.Close()

	require.Eventually(t, func() bool { return !hub.IsOnline("n1") }, 2*time.Second, 5*time.Millisecond,
		"node flips offline when the dial-out stream drops")
}

// [REQ:BRG-P1-001] The dial-out records the node's protocol-compatibility
// verdict from the version it advertises (?pv=). A current-version node is
// dispatchable; a node that omits ?pv= (back-compat) is also dispatchable.
func TestDialOut_RecordsProtocolCompatibility(t *testing.T) {
	srv, hub := startChannelServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		srv.URL+"/api/v1/channel/events?node=n1&pv="+fmt.Sprint(compat.ProtocolVersion), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	buf := make([]byte, 16)
	_, err = resp.Body.Read(buf)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return hub.Dispatchable("n1") }, time.Second, 5*time.Millisecond,
		"a current-protocol node is dispatchable")
	require.Equal(t, compat.StatusOK, hub.Compatibility("n1"))
}

// A dial-out attempt without a node credential (the Phase-1 stub ?node=) is
// rejected — an anonymous stream never registers presence.
func TestDialOut_RequiresNodeID(t *testing.T) {
	srv, hub := startChannelServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/channel/events")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Empty(t, hub.OnlineNodes())
}
