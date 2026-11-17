package wallet

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
	"treasury/internal/book"
)

func TestRotationWritesSignedTombstoneAndFreshNodeRejectsStaleSnapshot(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	const secret = "operator-held test recovery secret with 40 bytes"
	wallet, err := source.Create(ctx, in)
	require.NoError(t, err)
	_, err = source.PersistPortableRecoveryBundle(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, secret)
	require.NoError(t, err)
	// Keep the authentic pre-rotation pair as an immutable stale-snapshot fixture.
	backupRoot := t.TempDir()
	oldBundle, err := source.ExportPortableRecoveryBundle(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, secret)
	require.NoError(t, err)
	witness, err := source.ExportPortableRecoveryWitness(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, secret)
	require.NoError(t, err)

	_, _ = prepareAndCommitRotationFixture(t, ctx, source, wallet, "wallet-revoked-successor")
	identity, err := ca.ParseIdentity(wallet.CustodyReference)
	require.NoError(t, err)
	require.False(t, authority.Status(identity, keyField).Configured, "the old scalar is deleted after the tombstone is durable")

	// The actual artifact tree now contains a signed tombstone and no current
	// bundle; an immutable earlier snapshot still carries oldBundle+witness.
	var root string
	// setup stores artifact paths under a task-owned temporary directory. Resolve
	// the known file name through the Service path function rather than its
	// relative path response.
	root, err = source.recoveryPath(ctx, portableBackupName(wallet.ID))
	require.NoError(t, err)
	tombstoneBytes, err := os.ReadFile(filepath.Join(root, "revocation.json"))
	require.NoError(t, err)
	var tombstone PortableRecoveryRevocation
	require.NoError(t, json.Unmarshal(tombstoneBytes, &tombstone))
	require.True(t, VerifyPortableRecoveryRevocation(tombstone))
	require.Equal(t, wallet.ID, tombstone.WalletID)
	require.Equal(t, "wallet-revoked-successor", tombstone.SuccessorID)
	require.EqualValues(t, 2, tombstone.Epoch)
	_, err = os.Stat(filepath.Join(root, "wallet.bundle"))
	require.ErrorIs(t, err, os.ErrNotExist)

	// A clean replacement node is given the stale pair and the newer signed
	// revocation state selected by DBM's latest-snapshot contract. Treasury
	// refuses before creating book/wallet metadata or restoring a key.
	cleanDB := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, cleanDB, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(Schema)))
	cleanBooks := book.NewService(book.NewSQLiteRepository(database.NewFromPrimary(cleanDB)), schedule.System())
	cleanAuthority, err := ca.NewAuthority(memoryStore{})
	require.NoError(t, err)
	clean := NewService(NewSQLiteRepository(database.NewFromPrimary(cleanDB)), cleanBooks, cleanAuthority, schedule.System(), func(_ context.Context, name string) (string, error) {
		return filepath.Join(backupRoot, name), nil
	})
	_, err = clean.RestorePortableRecoveryWithWitness(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, oldBundle, witness, secret, tombstoneBytes)
	require.ErrorIs(t, err, ErrMismatch)
	cleanRestarted := NewService(clean.repository, clean.books, cleanAuthority, schedule.System(), clean.recoveryPath)
	_, err = cleanRestarted.RestorePortableRecoveryWithWitness(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, oldBundle, witness, secret, tombstoneBytes)
	require.ErrorIs(t, err, ErrMismatch, "a restart and retry with the same stale pre-rotation bundle remains refused")
	_, err = clean.repository.Get(ctx, wallet.ID)
	require.ErrorIs(t, err, ErrNotFound)
	cleanIdentity, err := ca.ParseIdentity(wallet.CustodyReference)
	require.NoError(t, err)
	_, err = cleanAuthority.Resolve(cleanIdentity, keyField)
	require.ErrorIs(t, err, ca.ErrUnconfigured)

}

func TestRotationPreparationKeepsOldScalarUntilOffhostReceipt(t *testing.T) {
	ctx := context.Background()
	source, authority, in := setup(t)
	wallet, err := source.Create(ctx, in)
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(wallet.CustodyReference)
	require.NoError(t, err)
	_, err = source.Rotate(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, "wallet-one-shot-bypass")
	require.ErrorIs(t, err, ErrUnavailable, "the legacy endpoint cannot accept an unverified publisher claim")
	active, err := source.repository.Get(ctx, wallet.ID)
	require.NoError(t, err)
	require.Empty(t, active.RevokedAt)
	require.True(t, authority.Status(identity, keyField).Configured)
	prepared, err := source.PreparePortableRotation(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, "wallet-awaiting-offhost-proof")
	require.NoError(t, err)
	stored, err := source.repository.Get(ctx, wallet.ID)
	require.NoError(t, err)
	require.Empty(t, stored.RevokedAt, "preparation does not revoke metadata before DBM publication proof")
	require.NotEmpty(t, prepared.Tombstone)
	require.True(t, authority.Status(identity, keyField).Configured, "old scalar stays available until off-host publication is verified")
	proof, err := source.Prove(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, "rotation_waits_for_dbm_nonce_20261115")
	require.NoError(t, err)
	require.True(t, proof.Verified)
}

func TestPortableRecoveryPersistRetryCompletesWitnessAfterInterruptedWrite(t *testing.T) {
	ctx := context.Background()
	source, _, in := setup(t)
	const secret = "operator-held test recovery secret with 40 bytes"
	wallet, err := source.Create(ctx, in)
	require.NoError(t, err)
	bundle, err := source.ExportPortableRecoveryBundle(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, secret)
	require.NoError(t, err)
	root, err := source.recoveryPath(ctx, portableBackupName(wallet.ID))
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(root, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "wallet.bundle"), bundle, 0o600))
	// Simulate a process stop after the durable ciphertext commit but before
	// the witness sidecar. The next export must validate and reuse exact-key
	// ciphertext, then finish the missing sidecar safely.
	artifact, err := source.PersistPortableRecoveryBundle(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, secret)
	require.NoError(t, err)
	require.Equal(t, filepath.ToSlash(portableBackupName(wallet.ID)), artifact.RelativePath)
	require.Equal(t, filepath.ToSlash(filepath.Join(portableBackupName(wallet.ID), "witness.json")), artifact.WitnessPath)
	witness, err := os.ReadFile(filepath.Join(root, "witness.json"))
	require.NoError(t, err)
	expected, err := source.ExportPortableRecoveryWitness(ctx, wallet.ID, wallet.Network, wallet.OperatorIdentity, secret)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(witness))
}
