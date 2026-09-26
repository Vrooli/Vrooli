package channel

import (
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"log"
	"testing"

	"vrooli-bridge/agent/internal/cpverify"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// testCPKeys returns a fresh control-plane keypair and a Verifier pinned to its
// public half — the in-test stand-in for the key `pair redeem` would persist.
func testCPKeys(t *testing.T) (ed25519.PrivateKey, *cpverify.Verifier) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	v, err := cpverify.NewVerifier(pub)
	require.NoError(t, err)
	return priv, v
}

// signedClient builds a quiet Client already pinned to a control-plane key, plus
// the private key a test uses to sign the frames it feeds the client. This is
// the wiring a paired agent always has (main fails hard without a pin).
func signedClient(t *testing.T) (*Client, ed25519.PrivateKey) {
	t.Helper()
	priv, v := testCPKeys(t)
	return &Client{logger: log.New(io.Discard, "", 0), cpVerifier: v}, priv
}

// signFrame produces the SSE payload the control plane would push: the exact
// wire contract mirrored from api internal/channelsign (proto-serialise the
// ServerFrame, Ed25519-sign those bytes, wrap in a protojson SignedServerFrame).
func signFrame(t *testing.T, priv ed25519.PrivateKey, frame *channelv1.ServerFrame) string {
	t.Helper()
	inner, err := proto.Marshal(frame)
	require.NoError(t, err)
	env := &channelv1.SignedServerFrame{Frame: inner, Signature: ed25519.Sign(priv, inner)}
	payload, err := protojson.Marshal(env)
	require.NoError(t, err)
	return string(payload)
}
