package credentialgrant

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"vrooli-bridge/internal/channelsign"

	"github.com/vrooli/vrooli/packages/proto/sealing"
)

type testSigner struct{ key ed25519.PrivateKey }

func (s testSigner) Sign(msg []byte) []byte { return ed25519.Sign(s.key, msg) }

func TestSealPushUsesIndependentNodeKeyAndBindsAddress(t *testing.T) {
	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	nodeKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	grant := Grant{ID: "g1", NodeID: "node-1", LogicalID: "vrooli/test", Field: "api-key", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 4}
	payload, err := SealPush(testSigner{key: signingKey}, grant, "node-1", nodeKey.PublicKey().Bytes(), "fixture-secret")
	require.NoError(t, err)
	frame, err := channelsign.Open(signingKey.Public().(ed25519.PublicKey), payload)
	require.NoError(t, err)
	push := frame.GetCredentialPush()
	require.NotNil(t, push)
	plain, err := sealing.Open(nodeKey, push.GetSealedValue(), push.GetAad())
	require.NoError(t, err)
	require.Equal(t, "fixture-secret", string(plain))
	require.Equal(t, sealing.CredentialContext("node-1", "vrooli/test", "api-key", 4), push.GetAad())
	require.NotContains(t, string(payload), "fixture-secret")
}

var _ channelsign.Signer = testSigner{}
