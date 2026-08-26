package credentialpush

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"vrooli-bridge/agent/internal/credentialgrant"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	"github.com/vrooli/vrooli/packages/proto/sealing"
)

type sink struct{ value string }

func (s *sink) Put(_, _, value string) error { s.value = value; return nil }
func (s *sink) Delete(_, _ string) error     { s.value = ""; return nil }

func TestApplyRejectsUngrantedPushWithoutLocalConsent(t *testing.T) {
	private, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	result, err := Apply(&channelv1.CredentialPush{NodeId: "node-1", LogicalId: "vrooli/test", Field: "api-key", Generation: 1, Retention: credentialgrant.RetentionDurable}, "node-1", private, credentialgrant.NewMemoryStore(), nil)
	require.NoError(t, err)
	require.True(t, result.Rejected)
	require.Equal(t, "address is not granted to this node", result.Receipt.GetReason())
}

func TestApplyDurableDecryptsAndZeroesPlaintext(t *testing.T) {
	private, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	grant := credentialgrant.Grant{ID: "g1", NodeID: "node-1", LogicalID: "vrooli/test", Field: "api-key", Class: credentialgrant.ClassUserPrompt, Retention: credentialgrant.RetentionDurable, Generation: 1}
	grants := credentialgrant.NewMemoryStore(grant)
	aad := []byte("credential-aad")
	sealed, err := sealing.Seal(private.PublicKey().Bytes(), []byte("fixture-secret"), aad)
	require.NoError(t, err)
	var got sink
	result, err := Apply(&channelv1.CredentialPush{GrantId: "g1", NodeId: "node-1", LogicalId: "vrooli/test", Field: "api-key", Generation: 1, Retention: credentialgrant.RetentionDurable, SealedValue: sealed, Aad: aad}, "node-1", private, grants, &got)
	require.NoError(t, err)
	require.True(t, result.Receipt.GetAccepted())
	require.Equal(t, "fixture-secret", got.value)
}

func TestApplyEphemeralReturnsBufferAndCallerCanZeroIt(t *testing.T) {
	private, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	grant := credentialgrant.Grant{ID: "g1", NodeID: "node-1", LogicalID: "vrooli/test", Field: "api-key", Class: credentialgrant.ClassInfrastructure, Retention: credentialgrant.RetentionEphemeral, Generation: 1}
	aad := []byte("credential-aad")
	sealed, err := sealing.Seal(private.PublicKey().Bytes(), []byte("fixture-secret"), aad)
	require.NoError(t, err)
	result, err := Apply(&channelv1.CredentialPush{GrantId: "g1", NodeId: "node-1", LogicalId: "vrooli/test", Field: "api-key", Generation: 1, Retention: credentialgrant.RetentionEphemeral, SealedValue: sealed, Aad: aad}, "node-1", private, credentialgrant.NewMemoryStore(grant), nil)
	require.NoError(t, err)
	require.True(t, result.Receipt.GetAccepted())
	require.Equal(t, "fixture-secret", string(result.Ephemeral))
	Zero(result.Ephemeral)
	require.Equal(t, make([]byte, len(result.Ephemeral)), result.Ephemeral)
}

func TestEphemeralStoreConsumesAndReplacesWithoutPersistence(t *testing.T) {
	store := NewEphemeralStore()
	require.NoError(t, store.Put("vrooli/test", "token", []byte("first")))
	require.NoError(t, store.Put("vrooli/test", "token", []byte("second")))
	value, ok := store.Take("vrooli/test", "token")
	require.True(t, ok)
	require.Equal(t, "second", string(value))
	Zero(value)
	_, ok = store.Take("vrooli/test", "token")
	require.False(t, ok, "an ephemeral credential is single-use")
}
