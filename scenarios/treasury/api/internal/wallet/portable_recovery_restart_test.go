package wallet

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"

	"treasury/internal/book"
)

// TestPortableRecoverySurvivesIndependentOwnerRestart keeps key custody and
// Treasury metadata in separate durable stores. It wipes the source host's
// credential file, imports only the encrypted key bundle on an independent
// owner, reopens both destination stores, and proves the recovered address.
func TestPortableRecoverySurvivesIndependentOwnerRestart(t *testing.T) {
	ctx := context.Background()
	clock := schedule.System()
	in := CreateInput{ID: "restart-recovered-wallet", BookID: "restart-book", Network: "eip155:84532", OperatorIdentity: "restart-test-operator"}
	const (
		operatorSecret   = "operator-held test recovery secret with 40 bytes"
		sourcePassphrase = "fixture source authority passphrase, test-only"
		nodePassphrase   = "fixture replacement authority passphrase, test-only"
	)

	sourceDir := t.TempDir()
	sourceDBPath := filepath.Join(sourceDir, "treasury.sqlite")
	sourceDB := fileWalletDB(t, sourceDBPath)
	sourceAuthorityPath := filepath.Join(sourceDir, "credentials.enc.json")
	sourceAuthority, err := ca.OpenEncryptedFile(sourceAuthorityPath, sourcePassphrase)
	require.NoError(t, err)
	sourceBooks := book.NewService(book.NewSQLiteRepository(database.NewFromPrimary(sourceDB)), clock)
	_, err = sourceBooks.Create(ctx, book.CreateInput{ID: in.BookID, Name: "Recovery restart fixture", BeneficiaryIdentity: in.OperatorIdentity})
	require.NoError(t, err)
	source := NewService(NewSQLiteRepository(database.NewFromPrimary(sourceDB)), sourceBooks, sourceAuthority, clock,
		func(_ context.Context, name string) (string, error) { return filepath.Join(sourceDir, name), nil })
	original, err := source.Create(ctx, in)
	require.NoError(t, err)
	bundle, err := source.ExportPortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, operatorSecret)
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(original.CustodyReference)
	require.NoError(t, err)
	manifest, err := ca.InspectRecovery(bundle, operatorSecret)
	require.NoError(t, err)
	require.Equal(t, []ca.RecoveryEntry{{Identity: identity, Field: keyField}}, manifest.Entries)
	require.NoError(t, sourceDB.Close())

	// Removing the host-owned encrypted credential store is equivalent to
	// losing that host. A fresh authority proves the old scalar is unavailable.
	require.NoError(t, os.Remove(sourceAuthorityPath))
	wipedSourceAuthority, err := ca.OpenEncryptedFile(sourceAuthorityPath, sourcePassphrase)
	require.NoError(t, err)
	_, err = wipedSourceAuthority.Resolve(identity, keyField)
	require.ErrorIs(t, err, ca.ErrUnconfigured)

	nodeDir := t.TempDir()
	nodeDBPath := filepath.Join(nodeDir, "treasury.sqlite")
	nodeDB := fileWalletDB(t, nodeDBPath)
	nodeBooks := book.NewService(book.NewSQLiteRepository(database.NewFromPrimary(nodeDB)), clock)
	_, err = nodeBooks.Create(ctx, book.CreateInput{ID: in.BookID, Name: "Recovery restart fixture", BeneficiaryIdentity: in.OperatorIdentity})
	require.NoError(t, err)
	nodeRepository := NewSQLiteRepository(database.NewFromPrimary(nodeDB))
	_, err = nodeRepository.Prepare(ctx, original)
	require.NoError(t, err)
	require.NoError(t, nodeRepository.SetPublic(ctx, original.ID, original.Address, original.PublicKey))
	require.NoError(t, nodeRepository.RecordMint(ctx, original.ID, false))
	nodeBundlePath := filepath.Join(nodeDir, "wallet-key.bundle")
	require.NoError(t, os.WriteFile(nodeBundlePath, bundle, 0o600))
	transportedBundle, err := os.ReadFile(nodeBundlePath)
	require.NoError(t, err)

	nodeAuthorityPath := filepath.Join(nodeDir, "credentials.enc.json")
	nodeAuthority, err := ca.OpenEncryptedFile(nodeAuthorityPath, nodePassphrase)
	require.NoError(t, err)
	nodeService := NewService(nodeRepository, nodeBooks, nodeAuthority, clock, nil)
	_, err = nodeService.RestorePortableRecoveryBundle(ctx, original.ID, original.Network, original.OperatorIdentity, transportedBundle, operatorSecret)
	require.NoError(t, err)

	// Simulate process restart on the replacement node: neither SQL handles nor
	// Credential Authority instances from before the close participate now.
	require.NoError(t, nodeDB.Close())
	serviceDB := fileWalletDB(t, nodeDBPath)
	serviceBooks := book.NewService(book.NewSQLiteRepository(database.NewFromPrimary(serviceDB)), clock)
	freshAuthority, err := ca.OpenEncryptedFile(nodeAuthorityPath, nodePassphrase)
	require.NoError(t, err)
	freshService := NewService(NewSQLiteRepository(database.NewFromPrimary(serviceDB)), serviceBooks, freshAuthority, clock, nil)
	proof, err := freshService.Prove(ctx, original.ID, original.Network, original.OperatorIdentity, "second_node_after_restart_20261112")
	require.NoError(t, err)
	require.True(t, proof.Verified)
	require.Equal(t, original.Address, proof.Wallet.Address)

	// Rotate by creating a distinct root, then revoke the restored old scalar.
	// The old immutable address remains bound to its metadata but cannot sign.
	replacementInput := in
	replacementInput.ID = "restart-recovered-wallet-successor"
	replacement, err := freshService.Create(ctx, replacementInput)
	require.NoError(t, err)
	require.NotEqual(t, original.Address, replacement.Address)
	newProof, err := freshService.Prove(ctx, replacement.ID, replacement.Network, replacement.OperatorIdentity, "second_node_rotated_20261112")
	require.NoError(t, err)
	require.True(t, newProof.Verified)
	require.NoError(t, freshAuthority.Delete(identity, keyField))
	_, err = freshService.Prove(ctx, original.ID, original.Network, original.OperatorIdentity, "second_node_revoked_20261112")
	require.Error(t, err)
}
