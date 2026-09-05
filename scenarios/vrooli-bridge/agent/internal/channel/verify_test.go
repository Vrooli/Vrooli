package channel

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// abortFrame is a convenient observable frame: acting on it cancels a registered
// run, so a test can assert whether a frame reached a handler by whether the run
// was cancelled.
func abortFrame(runID string) *channelv1.ServerFrame {
	return &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Abort{
		Abort: &channelv1.AbortJob{RunId: runID, Reason: "test"},
	}}
}

// cancelled reports whether ctx has been cancelled (the run's execution stopped).
func cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// [REQ:BRG-P0-002] A frame validly signed by the pinned control-plane key is
// verified and acted on; nothing is counted as rejected.
func TestHandleServerFrame_ValidSignedFrameActs(t *testing.T) {
	c, priv := signedClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("run-1", cancel)

	c.handleServerFrame(signFrame(t, priv, abortFrame("run-1")))

	require.True(t, cancelled(ctx), "a validly signed frame must reach the handler")
	require.Equal(t, uint64(0), c.rejectedFrames.Load(), "a valid frame is not counted as rejected")
}

// [REQ:BRG-P0-002] A frame signed by a DIFFERENT key than the pinned one (the
// impostor control plane) never reaches a handler and is counted.
func TestHandleServerFrame_RejectsWrongKey(t *testing.T) {
	c, _ := signedClient(t)
	// An attacker's key, not the pinned one.
	_, attacker, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("run-2", cancel)

	c.handleServerFrame(signFrame(t, attacker, abortFrame("run-2")))

	require.False(t, cancelled(ctx), "a wrong-key frame must NOT reach the handler")
	require.Equal(t, uint64(1), c.rejectedFrames.Load(), "the rejection is counted")
}

// [REQ:BRG-P0-002] A frame whose signed bytes were altered after signing fails
// verification and never reaches a handler.
func TestHandleServerFrame_RejectsTampered(t *testing.T) {
	c, priv := signedClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("run-3", cancel)

	// Sign one frame, then swap the signed inner bytes for a different frame while
	// keeping the original signature.
	valid := signFrame(t, priv, abortFrame("run-3"))
	var env channelv1.SignedServerFrame
	require.NoError(t, protojson.Unmarshal([]byte(valid), &env))
	other, err := proto.Marshal(abortFrame("run-3-EVIL"))
	require.NoError(t, err)
	env.Frame = other // signature no longer matches Frame
	bad, err := protojson.Marshal(&env)
	require.NoError(t, err)

	c.handleServerFrame(string(bad))

	require.False(t, cancelled(ctx), "a tampered frame must NOT reach the handler")
	require.Equal(t, uint64(1), c.rejectedFrames.Load())
}

// [REQ:BRG-P0-002] An unsigned envelope (attacker omits the signature) is
// rejected — there is no unsigned-frame acceptance path.
func TestHandleServerFrame_RejectsUnsigned(t *testing.T) {
	c, _ := signedClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("run-4", cancel)

	inner, err := proto.Marshal(abortFrame("run-4"))
	require.NoError(t, err)
	env := &channelv1.SignedServerFrame{Frame: inner} // no signature
	payload, err := protojson.Marshal(env)
	require.NoError(t, err)

	c.handleServerFrame(string(payload))

	require.False(t, cancelled(ctx), "an unsigned frame must NOT reach the handler")
	require.Equal(t, uint64(1), c.rejectedFrames.Load())
}

// [REQ:BRG-P0-002] A bare (pre-signing) ServerFrame — not wrapped in a signed
// envelope — is rejected, proving there is no legacy unsigned-frame path.
func TestHandleServerFrame_RejectsBareServerFrame(t *testing.T) {
	c, _ := signedClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("run-5", cancel)

	bare, err := protojson.Marshal(abortFrame("run-5"))
	require.NoError(t, err)

	c.handleServerFrame(string(bare))

	require.False(t, cancelled(ctx), "a bare unsigned ServerFrame must NOT reach the handler")
	require.Equal(t, uint64(1), c.rejectedFrames.Load())
}

// [REQ:BRG-P0-002] A client with no pinned key (a mis-wired build) refuses every
// frame rather than trusting it.
func TestHandleServerFrame_NoVerifierRefuses(t *testing.T) {
	c := &Client{logger: log.New(io.Discard, "", 0)} // no cpVerifier
	ctx, cancel := context.WithCancel(context.Background())
	c.registerJob("run-6", cancel)

	// Even a well-formed signed frame is refused when nothing is pinned.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	c.handleServerFrame(signFrame(t, priv, abortFrame("run-6")))

	require.False(t, cancelled(ctx), "no pin ⇒ no frame is trusted")
	require.Equal(t, uint64(1), c.rejectedFrames.Load())
}

func TestHandleServerFrame_RejectsCredentialPushWithoutLocalGrant(t *testing.T) {
	c, priv := signedClient(t)
	c.cfg.NodeID = "node-1"
	frame := &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_CredentialPush{CredentialPush: &channelv1.CredentialPush{
		NodeId: "node-1", LogicalId: "vrooli/test", Field: "api-key", Generation: 1, Retention: "durable",
	}}}
	c.handleServerFrame(signFrame(t, priv, frame))
	require.Equal(t, uint64(1), c.rejectedCredentialPushes.Load())
	require.Equal(t, uint64(0), c.rejectedFrames.Load())
}
