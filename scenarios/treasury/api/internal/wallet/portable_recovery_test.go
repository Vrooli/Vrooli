package wallet

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestPortableRecoveryBundleRestoresOnlyTheWitnessedKeyOnIndependentCustody(t *testing.T) {
	ctx := context.Background()
	const operatorSecret = "operator-held test recovery secret with 40 bytes"
	source, sourceAuthority, input := setup(t)
	original, err := source.Create(ctx, input)
	require.NoError(t, err)
	bundle, err := source.ExportPortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, operatorSecret)
	require.NoError(t, err)
	require.NotEmpty(t, bundle)
	identity, err := ca.ParseIdentity(original.CustodyReference)
	require.NoError(t, err)
	require.False(t, sourceAuthority.Status(identity, recoveryField).Configured, "the operator-held secret must not be persisted in same-host Credential Authority")
	manifest, err := ca.InspectRecovery(bundle, operatorSecret)
	require.NoError(t, err)
	require.Equal(t, []ca.RecoveryEntry{{Identity: identity, Field: keyField}}, manifest.Entries, "the portable bundle contains only this wallet's signing scalar")

	// Model loss of the source host's entire key store before the restore
	// owner is initialized. Recovery must depend on the encrypted bundle and
	// operator-held secret, never on a still-live source copy.
	require.NoError(t, sourceAuthority.Delete(identity, keyField))
	require.False(t, sourceAuthority.Status(identity, keyField).Configured, "source host key store must be wiped before recovery")

	// A second isolated owner receives only the durable wallet metadata, as a
	// database backup would, plus the encrypted bundle. Its Credential Authority
	// starts empty and cannot remint the already-witnessed key.
	restored, restoredAuthority, _ := setup(t)
	_, err = restored.repository.Prepare(ctx, original)
	require.NoError(t, err)
	require.NoError(t, restored.repository.SetPublic(ctx, original.ID, original.Address, original.PublicKey))
	require.NoError(t, restored.repository.RecordMint(ctx, original.ID, false))
	_, err = restored.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, "wrong operator-held recovery secret with 40 bytes")
	require.Error(t, err)
	require.False(t, restoredAuthority.Status(identity, keyField).Configured, "wrong secret must not populate the second custody owner")
	_, err = restored.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, append(append([]byte(nil), bundle...), 0), operatorSecret)
	require.Error(t, err, "corrupt ciphertext must not populate the second custody owner")
	require.False(t, restoredAuthority.Status(identity, keyField).Configured)
	unwitnessed, unwitnessedAuthority, _ := setup(t)
	_, err = unwitnessed.repository.Prepare(ctx, original)
	require.NoError(t, err)
	require.NoError(t, unwitnessed.repository.SetPublic(ctx, original.ID, original.Address, original.PublicKey))
	_, err = unwitnessed.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, operatorSecret)
	require.Error(t, err, "portable recovery must never mint or restore a key whose prior creation is unwitnessed")
	require.False(t, unwitnessedAuthority.Status(identity, keyField).Configured)

	recovered, err := restored.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, operatorSecret)
	require.NoError(t, err)
	require.Equal(t, original.Address, recovered.Address)
	require.Equal(t, original.PublicKey, recovered.PublicKey)
	read, err := restored.Get(ctx, original.ID, original.Network, original.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, original.Address, read.Address)

	// Prove control from the recovered key itself, using a fresh challenge on
	// the replacement custody owner. Matching metadata alone is not a control
	// proof; the restored owner must produce a valid signature after recovery.
	proof, err := restored.Prove(ctx, original.ID, original.Network, original.OperatorIdentity, "recovered_node_nonce_20261112")
	require.NoError(t, err)
	require.True(t, proof.Verified)
	require.Equal(t, original.Address, proof.Wallet.Address)

	_, err = restored.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, operatorSecret)
	require.Error(t, err, "recovery never overwrites a currently configured key")

	// Exercise root-wallet rotation as an explicit revoke-old / prove-new
	// sequence. The root's address is immutable, so rotation creates a new
	// wallet identity and revokes the prior scalar; it never mutates the old
	// public binding or silently overwrites its key.
	replacementInput := input
	replacementInput.ID = "wallet-rotation-successor"
	replacement, err := source.Create(ctx, replacementInput)
	require.NoError(t, err)
	require.NotEqual(t, original.Address, replacement.Address)
	newProof, err := source.Prove(ctx, replacement.ID, replacement.Network, replacement.OperatorIdentity, "rotated_root_nonce_20261112")
	require.NoError(t, err)
	require.True(t, newProof.Verified)
	require.NoError(t, sourceAuthority.Delete(identity, keyField))
	require.NoError(t, restoredAuthority.Delete(identity, keyField))
	_, err = source.Prove(ctx, original.ID, original.Network, original.OperatorIdentity, "revoked_root_nonce_20261112")
	require.Error(t, err, "revoked root key cannot produce a fresh control proof")
	_, err = restored.Prove(ctx, original.ID, original.Network, original.OperatorIdentity, "revoked_node_nonce_20261112")
	require.Error(t, err, "the recovered copy must also be revoked on the replacement owner")

	// This is a deliberately test-only random wallet; no network or broadcast
	// path is invoked by this local recovery proof.
	require.NotEqual(t, "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf", original.Address)
	require.NotContains(t, string(bundle), strings.TrimPrefix(original.Address, "0x"))
}
