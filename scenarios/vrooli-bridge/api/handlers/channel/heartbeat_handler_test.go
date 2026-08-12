package channel

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
	"testing"
	"time"

	auditmocks "vrooli-bridge/internal/audit/mocks"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/testutil/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

type fakeLastSeen struct {
	ids []string
	err error
}

type fakeDeliveryAckRecorder struct{ acks []runs.DeliveryAck }

func (f *fakeDeliveryAckRecorder) RecordDeliveryAck(_ context.Context, ack runs.DeliveryAck) error {
	f.acks = append(f.acks, ack)
	return nil
}

func (f *fakeLastSeen) TouchLastSeen(_ context.Context, id string, _ time.Time) error {
	f.ids = append(f.ids, id)
	return f.err
}

func newHub() *presence.Hub {
	return presence.NewHub(mocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)))
}

// [REQ:BRG-P0-003] A heartbeat records the node's health in the hub and
// persists its last-seen; the response carries the compatibility verdict.
func TestReportHeartbeat_StoresHealthAndTouchesLastSeen(t *testing.T) {
	hub := newHub()
	ls := &fakeLastSeen{}
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: hub, LastSeen: ls})

	resp, err := h.ReportHeartbeat(context.Background(), connect.NewRequest(&presencev1.ReportHeartbeatRequest{
		Heartbeat: &sharedv1.Heartbeat{
			NodeId: "n1",
			Health: &sharedv1.HealthSnapshot{ToolchainPresent: true, DiskHeadroomBytes: 5 << 30},
		},
	}))
	require.NoError(t, err)
	require.Equal(t, sharedv1.CompatibilityStatus_COMPATIBILITY_STATUS_OK, resp.Msg.Compatibility)

	snap, ok := hub.Health("n1")
	require.True(t, ok)
	require.True(t, snap.Ready())
	require.Equal(t, int64(5<<30), snap.DiskHeadroomBytes)
	require.Equal(t, []string{"n1"}, ls.ids)
}

func TestReportHeartbeat_MissingNodeIDRejected(t *testing.T) {
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: newHub(), LastSeen: &fakeLastSeen{}})
	_, err := h.ReportHeartbeat(context.Background(), connect.NewRequest(&presencev1.ReportHeartbeatRequest{
		Heartbeat: &sharedv1.Heartbeat{Health: &sharedv1.HealthSnapshot{ToolchainPresent: true}},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// fakeCredStore satisfies nodeauth.CredentialStore for enforcement tests.
type fakeCredStore struct{ keys map[string]ed25519.PublicKey }

func (f fakeCredStore) ActivePublicKey(_ context.Context, nodeID string) (ed25519.PublicKey, bool, error) {
	pub, ok := f.keys[nodeID]
	return pub, ok, nil
}

func signedHeartbeatReq(nodeID string, priv ed25519.PrivateKey, ts time.Time) *connect.Request[presencev1.ReportHeartbeatRequest] {
	req := connect.NewRequest(&presencev1.ReportHeartbeatRequest{
		Heartbeat: &sharedv1.Heartbeat{NodeId: nodeID, Health: &sharedv1.HealthSnapshot{ToolchainPresent: true}},
	})
	req.Header().Set(nodeauth.HeaderNode, nodeID)
	req.Header().Set(nodeauth.HeaderTS, strconv.FormatInt(ts.Unix(), 10))
	req.Header().Set(nodeauth.HeaderSig, base64.StdEncoding.EncodeToString(ed25519.Sign(priv, nodeauth.SigningPayload(nodeID, ts))))
	return req
}

// [REQ:BRG-P0-002] With a verifier wired, a heartbeat carrying a valid signature
// from the node's stored credential is accepted.
func TestReportHeartbeat_EnforcesMutualAuth_Accepts(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Unix(1_700_000_000, 0)
	v := nodeauth.NewVerifier(fakeCredStore{keys: map[string]ed25519.PublicKey{"n1": pub}},
		nodeauth.WithClock(func() time.Time { return now }))
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: newHub(), LastSeen: &fakeLastSeen{}, Verifier: v})

	_, err := h.ReportHeartbeat(context.Background(), signedHeartbeatReq("n1", priv, now))
	require.NoError(t, err)
}

// [REQ:BRG-P0-002] An unsigned heartbeat is rejected when enforcement is on.
func TestReportHeartbeat_EnforcesMutualAuth_RejectsUnsigned(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	v := nodeauth.NewVerifier(fakeCredStore{keys: map[string]ed25519.PublicKey{"n1": pub}})
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: newHub(), LastSeen: &fakeLastSeen{}, Verifier: v})

	_, err := h.ReportHeartbeat(context.Background(), connect.NewRequest(&presencev1.ReportHeartbeatRequest{
		Heartbeat: &sharedv1.Heartbeat{NodeId: "n1", Health: &sharedv1.HealthSnapshot{ToolchainPresent: true}},
	}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P0-002] A heartbeat signed by a DIFFERENT node's key (impostor) is
// rejected — the signature does not verify against n1's stored key.
func TestReportHeartbeat_EnforcesMutualAuth_RejectsImpostor(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, impostorPriv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Unix(1_700_000_000, 0)
	v := nodeauth.NewVerifier(fakeCredStore{keys: map[string]ed25519.PublicKey{"n1": pub}},
		nodeauth.WithClock(func() time.Time { return now }))
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: newHub(), LastSeen: &fakeLastSeen{}, Verifier: v})

	_, err := h.ReportHeartbeat(context.Background(), signedHeartbeatReq("n1", impostorPriv, now))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// A failure to persist last-seen must not drop the heartbeat (presence is
// already updated in memory).
func TestReportHeartbeat_LastSeenErrorSwallowed(t *testing.T) {
	hub := newHub()
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: hub, LastSeen: &fakeLastSeen{err: errors.New("db down")}})
	_, err := h.ReportHeartbeat(context.Background(), connect.NewRequest(&presencev1.ReportHeartbeatRequest{
		Heartbeat: &sharedv1.Heartbeat{NodeId: "n1", Health: &sharedv1.HealthSnapshot{ToolchainPresent: true}},
	}))
	require.NoError(t, err)
	_, ok := hub.Health("n1")
	require.True(t, ok, "health stored despite last-seen failure")
}

func TestReportDeliveryAckAcceptsFrameSentToSameNode(t *testing.T) {
	hub := newHub()
	conn := hub.Connect("n1")
	defer conn.Close()
	delivered := hub.PushFrame("n1", "frame-1", []byte("payload"))
	require.Equal(t, 1, delivered)

	recorder := &fakeDeliveryAckRecorder{}
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: hub, DeliveryAckRecorder: recorder})
	req := connect.NewRequest(&presencev1.ReportDeliveryAckRequest{Ack: &sharedv1.DeliveryAck{
		FrameId: "frame-1", RunId: "run-1", ReceivedAt: timestamppb.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)),
	}})
	req.Header().Set(nodeauth.HeaderNode, "n1")
	resp, err := h.ReportDeliveryAck(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.Accepted)
	require.Equal(t, "n1", hub.DeliveryAcks()[0].NodeID)
	require.Equal(t, "frame-1", recorder.acks[0].FrameID)
}

func TestReportDeliveryAckRejectsCrossNodeForge(t *testing.T) {
	hub := newHub()
	conn := hub.Connect("n1")
	defer conn.Close()
	require.Equal(t, 1, hub.PushFrame("n1", "frame-1", []byte("payload")))

	auditSink := &auditmocks.FakeSink{}
	h := NewHeartbeatHandler(HeartbeatDeps{Hub: hub, Audit: auditSink})
	req := connect.NewRequest(&presencev1.ReportDeliveryAckRequest{Ack: &sharedv1.DeliveryAck{FrameId: "frame-1"}})
	req.Header().Set(nodeauth.HeaderNode, "n2")
	_, err := h.ReportDeliveryAck(context.Background(), req)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Empty(t, hub.DeliveryAcks())
	require.Len(t, auditSink.Appended(), 1)
}
