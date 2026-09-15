package credentialgrant

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/vrooli/packages/proto/sealing"

	"vrooli-bridge/internal/channelsign"
	internalgrant "vrooli-bridge/internal/credentialgrant"
	"vrooli-bridge/internal/presence"
)

type handlerTestSigner struct{ key ed25519.PrivateKey }

func (s handlerTestSigner) Sign(msg []byte) []byte { return ed25519.Sign(s.key, msg) }

// [REQ:BRG-P1-002] The production delivery path sends consent metadata before
// the current value, but the value remains sealed for the node and absent from
// every serialized control-plane payload.
func TestDeliverGrantSendsMetadataAndSealedValueWithoutPlaintext(t *testing.T) {
	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	nodeKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)

	hub := presence.NewHub(scheduletest.New(time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)))
	conn := hub.Connect("node-1")
	defer conn.Close()

	const secret = "handler-fixture-secret"
	h := NewHandler(ModuleDeps{
		Presence: hub,
		Signer:   handlerTestSigner{key: signingKey},
		SealingPublicKey: func(context.Context, string) ([]byte, error) {
			return nodeKey.PublicKey().Bytes(), nil
		},
		ResolveValue: func(context.Context, string, string) (string, error) {
			return secret, nil
		},
	})
	grant := internalgrant.Grant{
		ID: "grant-1", NodeID: "node-1", LogicalID: "vrooli/test", Field: "api-key",
		Class: internalgrant.ClassUserPrompt, Retention: internalgrant.RetentionEphemeral, Generation: 3,
	}

	require.NoError(t, h.deliverGrant(context.Background(), grant))

	metadataPayload := <-conn.Out()
	metadataFrame, err := channelsign.Open(signingKey.Public().(ed25519.PublicKey), metadataPayload)
	require.NoError(t, err)
	require.NotNil(t, metadataFrame.GetCredentialGrant())
	require.NotContains(t, string(metadataPayload), secret)

	valuePayload := <-conn.Out()
	valueFrame, err := channelsign.Open(signingKey.Public().(ed25519.PublicKey), valuePayload)
	require.NoError(t, err)
	require.NotContains(t, string(valuePayload), secret)
	push := valueFrame.GetCredentialPush()
	require.NotNil(t, push)
	plain, err := sealing.Open(nodeKey, push.GetSealedValue(), push.GetAad())
	require.NoError(t, err)
	require.Equal(t, secret, string(plain))
	require.Equal(t, sealing.CredentialContext("node-1", "vrooli/test", "api-key", 3), push.GetAad())
}
