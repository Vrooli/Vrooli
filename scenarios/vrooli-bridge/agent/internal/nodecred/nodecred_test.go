package nodecred

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-P0-002] The node keypair is stable across loads (so its registered
// public key keeps verifying) and the private seed is written owner-only.
func TestLoadOrCreate_StableAndOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node.key")

	a, err := LoadOrCreate(path)
	require.NoError(t, err)
	b, err := LoadOrCreate(path)
	require.NoError(t, err)
	require.Equal(t, a.PublicKeyBase64(), b.PublicKeyBase64())

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

// [REQ:BRG-P0-002] A signature the agent produces verifies against its published
// public key over the canonical (node_id, ts) payload — the wire contract the
// control-plane verifier (api internal/nodeauth) relies on. This is the
// cross-module format check; if it drifts, mutual auth silently breaks.
func TestSign_MatchesCanonicalWireFormat(t *testing.T) {
	cred, err := LoadOrCreate(filepath.Join(t.TempDir(), "node.key"))
	require.NoError(t, err)

	ts := time.Unix(1_700_000_000, 0)
	sig := cred.Sign("node-7", ts)

	pubBytes, err := base64.StdEncoding.DecodeString(cred.PublicKeyBase64())
	require.NoError(t, err)
	wantPayload := []byte("node-7\n" + strconv.FormatInt(ts.Unix(), 10))
	require.True(t, ed25519.Verify(ed25519.PublicKey(pubBytes), wantPayload, sig),
		"signature must verify over '<node_id>\\n<unix>' (the nodeauth contract)")
}

// [REQ:BRG-P0-002] Headers and token carry the same proof in the two transports.
func TestHeadersAndToken_Encoding(t *testing.T) {
	cred, err := LoadOrCreate(filepath.Join(t.TempDir(), "node.key"))
	require.NoError(t, err)
	ts := time.Unix(1_700_000_000, 0)

	h := cred.Headers("node-7", ts)
	require.Equal(t, "node-7", h[HeaderNode])
	require.Equal(t, "1700000000", h[HeaderTS])
	require.NotEmpty(t, h[HeaderSig])

	token := cred.Token("node-7", ts)
	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3)
	require.Equal(t, "node-7", parts[0])
	require.Equal(t, "1700000000", parts[1])
	require.Equal(t, h[HeaderSig], parts[2], "header and token carry the same signature")
}
