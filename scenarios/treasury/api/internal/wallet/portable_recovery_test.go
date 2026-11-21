package wallet

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
	"treasury/internal/book"
)

func newRecoveryBookService() (book.Service, *databasetest.SliceRepo[book.Book]) {
	repo := databasetest.NewSliceRepo(
		func(record book.Book) string { return record.ID },
		func(record *book.Book, id string) { record.ID = id },
		func(string) error { return book.ErrNotFound },
	)
	return book.NewService(repo, schedule.System()), repo
}

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

	// Exercise the owner-level rotation contract after recovery. Rotation
	// creates a fresh address and durably revokes both the restored scalar and
	// any future restore of its old encrypted bundle.
	replacement, _ := prepareAndCommitRotationFixture(t, ctx, restored, original, "wallet-rotation-successor")
	require.NotEqual(t, original.Address, replacement.Address)
	newProof, err := restored.Prove(ctx, replacement.ID, replacement.Network, replacement.OperatorIdentity, "rotated_root_nonce_20261115")
	require.NoError(t, err)
	require.True(t, newProof.Verified)
	require.False(t, restoredAuthority.Status(identity, keyField).Configured)
	_, err = restored.Prove(ctx, original.ID, original.Network, original.OperatorIdentity, "revoked_node_nonce_20261115")
	require.ErrorIs(t, err, ErrMismatch, "the recovered root is durably revoked on the replacement owner")
	_, err = restored.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, operatorSecret)
	require.ErrorIs(t, err, ErrMismatch, "a revoked old bundle cannot re-enable the root")

	// This is a deliberately test-only random wallet; no network or broadcast
	// path is invoked by this local recovery proof.
	require.NotEqual(t, "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf", original.Address)
	require.NotContains(t, string(bundle), strings.TrimPrefix(original.Address, "0x"))
}

func TestPortableRecoveryAuthenticatedWitnessBootstrapsFreshTreasury(t *testing.T) {
	ctx := context.Background()
	const secret = "operator-held test recovery secret with 40 bytes"
	source, _, input := setup(t)
	original, err := source.Create(ctx, input)
	require.NoError(t, err)
	bundle, err := source.ExportPortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, secret)
	require.NoError(t, err)
	witness, err := source.ExportPortableRecoveryWitness(ctx, original.ID, original.Network, original.OperatorIdentity, secret)
	require.NoError(t, err)

	clean, authority, _ := setup(t) // independent CA and clean Treasury DB
	cleanBooks, cleanBookRepo := newRecoveryBookService()
	clean.books = cleanBooks // clean node must bootstrap the authenticated beneficiary book through the real book service
	_, err = clean.repository.Get(ctx, original.ID)
	require.ErrorIs(t, err, ErrNotFound)

	badMetadata := append([]byte(nil), witness...)
	badMetadata[len(badMetadata)-3] ^= 1
	_, err = clean.RestorePortableRecoveryWithWitness(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, badMetadata, secret)
	require.Error(t, err, "metadata tampering must be rejected before a wallet row or key is created")
	_, err = clean.repository.Get(ctx, original.ID)
	require.ErrorIs(t, err, ErrNotFound)

	badBundle := append([]byte(nil), bundle...)
	badBundle[len(badBundle)-1] ^= 1
	_, err = clean.RestorePortableRecoveryWithWitness(ctx, original.ID, original.Network, original.OperatorIdentity, badBundle, witness, secret)
	require.Error(t, err, "bundle tampering must be rejected before metadata import")
	_, err = clean.repository.Get(ctx, original.ID)
	require.ErrorIs(t, err, ErrNotFound)

	recovered, err := clean.RestorePortableRecoveryWithWitness(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, witness, secret)
	require.NoError(t, err)
	require.Equal(t, original.Address, recovered.Address)
	require.Equal(t, original.PublicKey, recovered.PublicKey)
	bootstrappedBook, err := cleanBooks.Get(ctx, original.BookID)
	require.NoError(t, err)
	require.Equal(t, original.BookID, bootstrappedBook.ID)
	require.Equal(t, original.OperatorIdentity, bootstrappedBook.BeneficiaryIdentity)
	identity, err := ca.ParseIdentity(original.CustodyReference)
	require.NoError(t, err)
	require.True(t, authority.Status(identity, keyField).Configured)

	again, err := clean.RestorePortableRecoveryWithWitness(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, witness, secret)
	require.NoError(t, err, "repeating the same authenticated restore is idempotent and must not replace the key")
	require.Equal(t, original.PublicKey, again.PublicKey)
	require.Len(t, cleanBookRepo.Items, 1, "retry must not duplicate the beneficiary book")
	proof, err := clean.Prove(ctx, original.ID, original.Network, original.OperatorIdentity, "fresh_node_bootstrap_nonce_20261115")
	require.NoError(t, err)
	require.True(t, proof.Verified)
	require.Equal(t, original.Address, proof.Wallet.Address)

	// Fault boundary: CA durably accepted the recovered key, then the process
	// stopped before the SQL key_minted marker committed. A retry may finish
	// this exact witness idempotently, while a different stored key is refused.
	interrupted, interruptedAuthority, _ := setup(t)
	require.NoError(t, interruptedAuthority.RestoreRecovery(bundle, secret))
	retried, err := interrupted.RestorePortableRecoveryWithWitness(ctx, original.ID, original.Network, original.OperatorIdentity, bundle, witness, secret)
	require.NoError(t, err)
	require.Equal(t, original.Address, retried.Address)
	minted, err := interrupted.repository.Minted(ctx, original.ID, false)
	require.NoError(t, err)
	require.True(t, minted, "retry completes the durable DB witness after CA persistence")
}
