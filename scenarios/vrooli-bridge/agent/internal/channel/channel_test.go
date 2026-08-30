package channel

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
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
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
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
	require.Equal(t, MachineArchitecture(), hs.GetMachineArch())
	require.Equal(t, runtime.GOARCH, hs.GetBinaryArch())
	require.Equal(t, []string{"scenario test*"}, hs.GetCapabilities())
	require.NotEmpty(t, hs.GetAgentVersion(), "agent version is the build fingerprint")
}

func TestDial_UnpairedReturnsNotConfigured(t *testing.T) {
	err := NewClient(config.Config{}).Dial(context.Background())
	require.ErrorIs(t, err, ErrNotConfigured)
}

// TestProtocolVersionMatchesProto guards the one invariant that would silently
// mis-drive a node: the agent's implemented ProtocolVersion must equal the
// CHANNEL_PROTOCOL_VERSION the proto documents (2).
func TestProtocolVersionMatchesProto(t *testing.T) {
	require.Equal(t, uint32(2), ProtocolVersion)
}

func TestHomeForVrooliBinaryStripsInstalledLayout(t *testing.T) {
	require.Equal(t, "/Users/tester", homeForVrooliBinary("/Users/tester/.vrooli/bin/vrooli"))
	require.Equal(t, "", homeForVrooliBinary("vrooli"))
}

// fakeControlPlane stands up the node-facing edge of the control plane: the SSE
// dial-out endpoint (which it holds open) and the PresenceService heartbeat
// handler (which records what the agent reports). It is the in-process double
// the live-dial test drives — no real control plane needed.
type fakeControlPlane struct {
	mu           sync.Mutex
	heartbeats   []*sharedv1.Heartbeat
	hbAuthSig    string // X-Bridge-Sig of the most recent heartbeat
	hbAuthNode   string
	hbAuthTS     string
	sseToken     string // ?token= of the most recent dial-out
	sseProtocol  string
	sseMachine   string
	sseBinary    string
	sseOpened    atomic.Int64
	deliveryAcks []*sharedv1.DeliveryAck
	closeStreams chan struct{}
	// pushFrames carries already-serialised SSE payloads (a signed
	// SignedServerFrame envelope) the control plane pushes down a held stream, so
	// a test can deliver a real server frame over the actual dial-out transport.
	pushFrames chan string
}

func newFakeControlPlane() *fakeControlPlane {
	return &fakeControlPlane{closeStreams: make(chan struct{}), pushFrames: make(chan string, 8)}
}

func (f *fakeControlPlane) ReportHeartbeat(_ context.Context, req *connect.Request[presencev1.ReportHeartbeatRequest]) (*connect.Response[presencev1.ReportHeartbeatResponse], error) {
	f.mu.Lock()
	f.heartbeats = append(f.heartbeats, req.Msg.GetHeartbeat())
	f.hbAuthSig = req.Header().Get("X-Bridge-Sig")
	f.hbAuthNode = req.Header().Get("X-Bridge-Node")
	f.hbAuthTS = req.Header().Get("X-Bridge-Ts")
	f.mu.Unlock()
	return connect.NewResponse(&presencev1.ReportHeartbeatResponse{
		Compatibility: sharedv1.CompatibilityStatus_COMPATIBILITY_STATUS_OK,
	}), nil
}

func (f *fakeControlPlane) ReportDeliveryAck(_ context.Context, req *connect.Request[presencev1.ReportDeliveryAckRequest]) (*connect.Response[presencev1.ReportDeliveryAckResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deliveryAcks = append(f.deliveryAcks, req.Msg.GetAck())
	return connect.NewResponse(&presencev1.ReportDeliveryAckResponse{Accepted: req.Msg.GetAck().GetFrameId() != ""}), nil
}

func (f *fakeControlPlane) ReportSessionFrame(_ context.Context, _ *connect.Request[presencev1.ReportSessionFrameRequest]) (*connect.Response[presencev1.ReportSessionFrameResponse], error) {
	return connect.NewResponse(&presencev1.ReportSessionFrameResponse{Accepted: true}), nil
}

func (f *fakeControlPlane) ReportRelayResponse(_ context.Context, _ *connect.Request[presencev1.ReportRelayResponseRequest]) (*connect.Response[presencev1.ReportRelayResponseResponse], error) {
	return connect.NewResponse(&presencev1.ReportRelayResponseResponse{Accepted: true}), nil
}

func (f *fakeControlPlane) ReportCredentialReceipt(_ context.Context, _ *connect.Request[presencev1.ReportCredentialReceiptRequest]) (*connect.Response[presencev1.ReportCredentialReceiptResponse], error) {
	return connect.NewResponse(&presencev1.ReportCredentialReceiptResponse{Accepted: true}), nil
}

func (f *fakeControlPlane) deliveryAckCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.deliveryAcks)
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
		f.sseProtocol = r.URL.Query().Get("pv")
		f.sseMachine = r.URL.Query().Get("machine_arch")
		f.sseBinary = r.URL.Query().Get("binary_arch")
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
		for {
			select {
			case <-r.Context().Done():
				return
			case <-f.closeStreams:
				return
			case payload := <-f.pushFrames:
				_, _ = io.WriteString(w, "data: "+payload+"\n\n")
				flusher.Flush()
			}
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
	fcp.mu.Lock()
	require.Equal(t, "2", fcp.sseProtocol)
	require.Equal(t, MachineArchitecture(), fcp.sseMachine)
	require.Equal(t, runtime.GOARCH, fcp.sseBinary)
	fcp.mu.Unlock()

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

// [REQ:BRG-P0-002] TestDial_VerifiesSignedFramesOverTheWire is the G7 acceptance
// test: over a REAL dial-out SSE loop, a frame signed by the pinned control-plane
// key is verified and acted on, while a frame signed by a DIFFERENT key (an
// impostor control plane) is rejected before any handler sees it and surfaces on
// the node's heartbeat health as rejected_cp_frames. This exercises the whole
// path — SSE read → envelope decode → signature verify → dispatch — not just the
// verifier in isolation.
func TestDial_VerifiesSignedFramesOverTheWire(t *testing.T) {
	fcp := newFakeControlPlane()
	srv := httptest.NewServer(fcp.handler())
	defer srv.Close()

	priv, verifier := testCPKeys(t)

	cfg := config.Config{ControlPlaneURL: srv.URL, NodeID: "node-int", HeartbeatInterval: 10 * time.Millisecond}
	client := NewClient(cfg,
		WithHTTPClient(srv.Client()),
		WithSampler(health.Fixed{Snap: health.Snapshot{ToolchainPresent: true}}),
		WithCPVerifier(verifier),
		WithBackoff(time.Millisecond, 5*time.Millisecond),
	)

	// A registered run whose cancel func fires iff a verified AbortJob reaches the
	// dispatch handler — the observable proxy for "the frame was acted on".
	runCtx, runCancel := context.WithCancel(context.Background())
	client.registerJob("run-int", runCancel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = client.Dial(ctx) }()

	require.Eventually(t, func() bool { return fcp.sseOpened.Load() >= 1 }, 2*time.Second, 5*time.Millisecond,
		"agent holds the dial-out stream")

	// Impostor: a provision frame signed by a DIFFERENT key must be rejected.
	_, attacker, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	provision := &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Provision{
		Provision: &channelv1.ProvisionCommand{OpId: "op-evil", TargetRevision: "deadbeef"},
	}}
	fcp.pushFrames <- signFrame(t, attacker, provision)

	require.Eventually(t, func() bool { return client.rejectedFrames.Load() >= 1 }, 2*time.Second, 5*time.Millisecond,
		"an impostor-signed frame is rejected and counted")
	require.False(t, cancelled(runCtx), "the impostor frame must not have reached any handler")

	// The rejection surfaces on the heartbeat health details for the operator.
	require.Eventually(t, func() bool {
		fcp.mu.Lock()
		defer fcp.mu.Unlock()
		for _, hb := range fcp.heartbeats {
			if hb.GetHealth().GetDetails()["rejected_cp_frames"] == "1" {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond, "rejected frame count is surfaced on the heartbeat")

	// Legitimate: a frame signed by the PINNED key is verified and acted on.
	abort := &channelv1.ServerFrame{FrameId: "frame-abort", Payload: &channelv1.ServerFrame_Abort{
		Abort: &channelv1.AbortJob{RunId: "run-int", Reason: "operator abort"},
	}}
	fcp.pushFrames <- signFrame(t, priv, abort)

	require.Eventually(t, func() bool { return fcp.deliveryAckCount() == 1 }, 2*time.Second, 5*time.Millisecond,
		"the agent acknowledges the verified frame before dispatching it")
	require.Eventually(t, func() bool { return cancelled(runCtx) }, 2*time.Second, 5*time.Millisecond,
		"a validly signed AbortJob reaches the dispatch handler over the real transport")
	require.Equal(t, uint64(1), client.rejectedFrames.Load(), "the valid frame is not counted as rejected")
}
