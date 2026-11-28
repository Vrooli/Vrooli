package channel

import (
	"io"
	"log"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	"github.com/vrooli/vrooli/packages/proto/sealing"

	"vrooli-bridge/agent/internal/config"
	"vrooli-bridge/agent/internal/credentialgrant"
	"vrooli-bridge/agent/internal/credentialpush"
	"vrooli-bridge/agent/internal/nodecred"
	"vrooli-bridge/agent/internal/storeunlock"
)

// lockedUntilUnlockedSink refuses durable writes until the node's store
// passphrase is held, as the node's encrypted-file store does.
type lockedUntilUnlockedSink struct {
	holder *storeunlock.Holder
	stored map[string]string
}

func (s *lockedUntilUnlockedSink) Put(logicalID, field, value string) error {
	if !s.holder.Held() {
		return credentialpush.StoreRefusal{State: "locked", Recovery: "run `vrooli credentials store unlock` on the node and retry"}
	}
	s.stored[logicalID+":"+field] = value
	return nil
}

func (s *lockedUntilUnlockedSink) Delete(string, string) error { return nil }

func sealedPush(t *testing.T, enc *nodecred.EncryptionCredential, grant credentialgrant.Grant, value string) *channelv1.CredentialPush {
	t.Helper()
	aad := sealing.CredentialContext(grant.NodeID, grant.LogicalID, grant.Field, grant.Generation)
	sealed, err := sealing.Seal(enc.PublicKey(), []byte(value), aad)
	require.NoError(t, err)
	return &channelv1.CredentialPush{
		GrantId: grant.ID, NodeId: grant.NodeID, LogicalId: grant.LogicalID, Field: grant.Field,
		Generation: grant.Generation, Retention: grant.Retention, SealedValue: sealed, Aad: aad,
	}
}

// A durable credential that arrives before the store passphrase is held and
// written once the passphrase arrives, instead of staying refused until the
// next reconnect (2026-09-15: vrooli/openrouter on minimouse).
func TestDurablePushRefusedByALockedStoreIsAppliedOnceTheStoreUnlocks(t *testing.T) {
	enc, err := nodecred.LoadOrCreateEncryption(filepath.Join(t.TempDir(), "encryption.key"))
	require.NoError(t, err)
	durable := credentialgrant.Grant{
		ID: "grant-openrouter", NodeID: "node-1", LogicalID: "vrooli/openrouter", Field: "api-key",
		Class: credentialgrant.ClassUserPrompt, Retention: credentialgrant.RetentionDurable, Generation: 1,
	}
	passphrase := credentialgrant.Grant{
		ID: "grant-store", NodeID: "node-1", LogicalID: "vrooli-bridge/node-credential-store/machine-1", Field: "passphrase",
		Class: credentialgrant.ClassUserPrompt, Retention: credentialgrant.RetentionEphemeral, Generation: 1,
	}
	holder := &storeunlock.Holder{}
	sink := &lockedUntilUnlockedSink{holder: holder, stored: map[string]string{}}
	c := &Client{
		cfg:            config.Config{NodeID: "node-1"},
		encryption:     enc,
		grantStore:     credentialgrant.NewMemoryStore(durable, passphrase),
		credentialSink: sink,
		storeUnlock:    holder,
		logger:         log.New(io.Discard, "", 0),
	}

	c.handleCredentialPush(sealedPush(t, enc, durable, "sk-or-value"))
	require.Empty(t, sink.stored, "a locked store cannot accept the value yet")

	c.handleCredentialPush(sealedPush(t, enc, passphrase, "store-passphrase"))
	require.True(t, holder.Held())
	require.Equal(t, "sk-or-value", sink.stored["vrooli/openrouter:api-key"])

	c.lockedMu.Lock()
	defer c.lockedMu.Unlock()
	require.Empty(t, c.lockedPushes, "an applied push is not held again")
}
