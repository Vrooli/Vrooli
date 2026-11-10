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

	_, err = restored.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, operatorSecret)
	require.Error(t, err, "recovery never overwrites a currently configured key")

	// This is a deliberately test-only random wallet; no network or broadcast
	// path is invoked by this local recovery proof.
	require.NotEqual(t, "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf", original.Address)
	require.NotContains(t, string(bundle), strings.TrimPrefix(original.Address, "0x"))
}
