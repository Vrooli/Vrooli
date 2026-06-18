package channel

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vrooli-bridge/agent/internal/config"
	"vrooli-bridge/agent/internal/health"
	"vrooli-bridge/agent/internal/nodecred"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
)

func TestHandshake_ReflectsConfigAndBuildTarget(t *testing.T) {
	cfg := config.Config{
		NodeID:       "node-1",
		Capabilities: []string{"scenario test*"},
	}
	hs := NewClient(cfg).Handshake()

	require.Equal(t, ProtocolVersion, hs.GetProtocolVersion())
	require.Equal(t, "node-1", hs.GetNodeId())
	require.Equal(t, runtime.GOOS, hs.GetOs())
	require.Equal(t, runtime.GOARCH, hs.GetArch())
	require.Equal(t, []string{"scenario test*"}, hs.GetCapabilities())
	require.NotEmpty(t, hs.GetAgentVersion(), "agent version is the build fingerprint")
}

func TestDial_UnpairedReturnsNotConfigured(t *testing.T) {
	err := NewClient(config.Config{}).Dial(context.Background())
	require.ErrorIs(t, err, ErrNotConfigured)
}

// TestProtocolVersionMatchesProto guards the one invariant that would silently
// mis-drive a node: the agent's implemented ProtocolVersion must equal the
// CHANNEL_PROTOCOL_VERSION the proto documents (1).
func TestProtocolVersionMatchesProto(t *testing.T) {
	require.Equal(t, uint32(1), ProtocolVersion)
}

// fakeControlPlane stands up the node-facing edge of the control plane: the SSE
// dial-out endpoint (which it holds open) and the PresenceService heartbeat
// handler (which records what the agent reports). It is the in-process double
// the live-dial test drives — no real control plane needed.
type fakeControlPlane struct {
	mu           sync.Mutex
	heartbeats   []*channelv1.Heartbeat
	hbAuthSig    string // X-Bridge-Sig of the most recent heartbeat
	hbAuthNode   string
	hbAuthTS     string
	sseToken     string // ?token= of the most recent dial-out
	sseOpened    atomic.Int64
	closeStreams chan struct{}
}

func newFakeControlPlane() *fakeControlPlane {
	return &fakeControlPlane{closeStreams: make(chan struct{})}
}

func (f *fakeControlPlane) ReportHeartbeat(_ context.Context, req *connect.Request[presencev1.ReportHeartbeatRequest]) (*connect.Response[presencev1.ReportHeartbeatResponse], error) {
	f.mu.Lock()
	f.heartbeats = append(f.heartbeats, req.Msg.GetHeartbeat())
	f.hbAuthSig = req.Header().Get("X-Bridge-Sig")
	f.hbAuthNode = req.Header().Get("X-Bridge-Node")
	f.hbAuthTS = req.Header().Get("X-Bridge-Ts")
	f.mu.Unlock()
	return connect.NewResponse(&presencev1.ReportHeartbeatResponse{
		Compatibility: channelv1.CompatibilityStatus_COMPATIBILITY_STATUS_OK,
	}), nil
}

func (f *fakeControlPlane) heartbeatCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.heartbeats)
}

func (f *fakeControlPlane) handler() http.Handler {
	mux := http.NewServeMux()
	path, h := presence_v1connect.NewPresenceServiceHandler(f)
	mux.Handle(path, h)
	mux.HandleFunc(channelEventsPath, func(w http.ResponseWriter, r *http.Request) {
		require := func(ok bool) {
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
			}
		}
		f.mu.Lock()
		f.sseToken = r.URL.Query().Get("token")
		f.mu.Unlock()
		require(r.URL.Query().Get("node") != "" || r.URL.Query().Get("token") != "")
		f.sseOpened.Add(1)
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flush", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, ": connected\n\n")
		flusher.Flush()
		select {
		case <-r.Context().Done():
		case <-f.closeStreams:
		}
	})
	return mux
}

// [REQ:BRG-P0-003] TestDial_HoldsChannelAndHeartbeats is the Phase-1 dial-out
// acceptance test: a paired agent opens the SSE stream purely OUTBOUND and
// reports heartbeats carrying its health snapshot, until its context is
// cancelled (clean shutdown → nil). The node never opens a listening port.
func TestDial_HoldsChannelAndHeartbeats(t *testing.T) {
	fcp := newFakeControlPlane()
	srv := httptest.NewServer(fcp.handler())
	defer srv.Close()

	cfg := config.Config{
		ControlPlaneURL:   srv.URL,
		NodeID:            "node-7",
		HeartbeatInterval: 10 * time.Millisecond,
	}
	client := NewClient(cfg,
		WithHTTPClient(srv.Client()),
		WithSampler(health.Fixed{Snap: health.Snapshot{ToolchainPresent: true, DiskHeadroomBytes: 1 << 30}}),
		WithBackoff(time.Millisecond, 5*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	dialDone := make(chan error, 1)
	go func() { dialDone <- client.Dial(ctx) }()

	require.Eventually(t, func() bool { return fcp.heartbeatCount() >= 3 }, 2*time.Second, 5*time.Millisecond,
		"agent should report heartbeats on the configured cadence")
	require.GreaterOrEqual(t, fcp.sseOpened.Load(), int64(1), "agent should hold the dial-out SSE stream")

	hb := fcp.heartbeats[0]
	require.Equal(t, "node-7", hb.GetNodeId())
	require.Equal(t, uint64(1), hb.GetSequence(), "sequence is per-connection, starting at 1")
	require.True(t, hb.GetHealth().GetToolchainPresent())
	require.Equal(t, int64(1<<30), hb.GetHealth().GetDiskHeadroomBytes())

	cancel()
	select {
	case err := <-dialDone:
		require.NoError(t, err, "ctx-cancelled shutdown is clean")
	case <-time.After(2 * time.Second):
		t.Fatal("Dial did not return after context cancellation")
	}
}

// [REQ:BRG-P0-002] When credentialed, the agent signs its heartbeats (X-Bridge-*
// headers) and binds the dial-out token to its key — the live half of mutual
// auth. The captured proof verifies against the node's public key.
func TestDial_SignsWhenCredentialed(t *testing.T) {
	fcp := newFakeControlPlane()
	srv := httptest.NewServer(fcp.handler())
	defer srv.Close()

	cred, err := nodecred.LoadOrCreate(filepath.Join(t.TempDir(), "node.key"))
	if err != nil {
		t.Fatalf("load cred: %v", err)
	}
	cfg := config.Config{ControlPlaneURL: srv.URL, NodeID: "node-7", HeartbeatInterval: 10 * time.Millisecond}
	client := NewClient(cfg,
		WithHTTPClient(srv.Client()),
		WithSampler(health.Fixed{Snap: health.Snapshot{ToolchainPresent: true}}),
		WithCredential(cred),
		WithBackoff(time.Millisecond, 5*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = client.Dial(ctx) }()

	require.Eventually(t, func() bool { return fcp.heartbeatCount() >= 1 }, 2*time.Second, 5*time.Millisecond)

	fcp.mu.Lock()
	sig, node, tsStr, token := fcp.hbAuthSig, fcp.hbAuthNode, fcp.hbAuthTS, fcp.sseToken
	fcp.mu.Unlock()

	require.Equal(t, "node-7", node, "heartbeat carries the signing node id")
	require.NotEmpty(t, token, "dial-out is token-authed, not ?node=")

	pubBytes, err := base64.StdEncoding.DecodeString(cred.PublicKeyBase64())
	require.NoError(t, err)
	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	require.NoError(t, err)
	require.True(t, ed25519.Verify(ed25519.PublicKey(pubBytes), []byte("node-7\n"+tsStr), sigBytes),
		"heartbeat signature must verify against the node key")
}

// [REQ:BRG-P0-003] TestDial_ReconnectsAfterStreamDrop proves the backoff/
// reconnect loop: when the control plane drops the SSE stream, the agent dials
// again rather than giving up, so a flaky network never permanently parts a
// node from the fleet.
func TestDial_ReconnectsAfterStreamDrop(t *testing.T) {
	fcp := newFakeControlPlane()
	srv := httptest.NewServer(fcp.handler())
	defer srv.Close()

	cfg := config.Config{
		ControlPlaneURL:   srv.URL,
		NodeID:            "node-9",
		HeartbeatInterval: 10 * time.Millisecond,
	}
	client := NewClient(cfg,
		WithHTTPClient(srv.Client()),
		WithSampler(health.Fixed{Snap: health.Snapshot{ToolchainPresent: true}}),
		WithBackoff(time.Millisecond, 5*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = client.Dial(ctx) }()

	require.Eventually(t, func() bool { return fcp.sseOpened.Load() >= 1 }, 2*time.Second, 5*time.Millisecond)
	close(fcp.closeStreams) // drop every held stream

	require.Eventually(t, func() bool { return fcp.sseOpened.Load() >= 2 }, 2*time.Second, 5*time.Millisecond,
		"agent should reopen the dial-out stream after a drop")
}
