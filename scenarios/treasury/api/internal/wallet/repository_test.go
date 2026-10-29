package wallet

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	secp "github.com/go-webauthn/x/crypto/secp256k1"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"

	"treasury/internal/book"
)

// A portability fixture must never create a new signing key or passphrase.
// Only the published scalar-1 vector and public fixture strings are admitted.
type noMintTestCustody struct {
	Custody
	mintCalls int
}

func (c *noMintTestCustody) ResolveOrMint(id ca.Identity, field string, witness ca.Witness, _ func() (string, error)) (string, error) {
	return c.Custody.ResolveOrMint(id, field, witness, func() (string, error) {
		c.mintCalls++
		return "", ErrUnavailable
	})
}

func TestPublicVectorRecoveryPortabilityWithSourceUnavailable(t *testing.T) {
	ctx := context.Background()
	clock := schedule.System()
	in := CreateInput{ID: "public-vector-never-fund", BookID: "public-fixture", Network: "eip155:84532", OperatorIdentity: "public-test-operator"}
	const walletPassphrase = "PUBLIC fixture wallet backup passphrase, not a secret"
	const exportPassphrase = "PUBLIC fixture portable export passphrase, not a secret"
	sourceRoot := t.TempDir()
	sourcePath := filepath.Join(sourceRoot, "wallet.sqlite")
	sourceDB := fileWalletDB(t, sourcePath)
	sourceBooks := book.NewService(book.NewSQLiteRepository(sourceDB), clock)
	_, err := sourceBooks.Create(ctx, book.CreateInput{ID: in.BookID, Name: "Never fund this public vector", BeneficiaryIdentity: in.OperatorIdentity})
	require.NoError(t, err)
	repository := NewSQLiteRepository(sourceDB)
	fixture := Wallet{ID: in.ID, BookID: in.BookID, Network: in.Network, OperatorIdentity: in.OperatorIdentity, CustodyReference: reference(in, "11111111111111111111111111111111"), CreatedAt: time.Unix(1, 0).UTC()}
	_, err = repository.Prepare(ctx, fixture)
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(fixture.CustodyReference)
	require.NoError(t, err)
	sourceStore := memoryStore{}
	sourceAuthority, err := ca.NewAuthority(sourceStore)
	require.NoError(t, err)
	scalar := make([]byte, 32)
	scalar[31] = 1 // Universally published vector; never usable as funded custody.
	require.NoError(t, sourceAuthority.Put(identity, keyField, hex.EncodeToString(scalar)))
	clear(scalar)
	require.NoError(t, sourceAuthority.Put(identity, recoveryField, walletPassphrase))
	sourceCustody := &noMintTestCustody{Custody: sourceAuthority}
	source := NewService(repository, sourceBooks, sourceCustody, clock, func(_ context.Context, rel string) (string, error) {
		return filepath.Join(sourceRoot, rel), nil
	})
	w, err := source.Create(ctx, in) // Adopts the existing public vector, never mints.
	require.NoError(t, err)
	require.Equal(t, "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf", w.Address)
	w, err = source.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	backup, err := os.ReadFile(filepath.Join(sourceRoot, backupName(in.ID)))
	require.NoError(t, err)
	digest := sha256.Sum256(backup)
	require.Equal(t, hex.EncodeToString(digest[:]), w.RecoverySHA256)
	portable, err := sourceAuthority.ExportRecovery([]ca.RecoveryEntry{{Identity: identity, Field: keyField}, {Identity: identity, Field: recoveryField}}, exportPassphrase)
	require.NoError(t, err)
	require.Zero(t, sourceCustody.mintCalls)
	require.NoError(t, sourceDB.Close()) // Copy a closed SQLite image, never a live DB.
	image, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	clear(sourceStore)
	require.NoError(t, os.RemoveAll(sourceRoot))
	require.NoDirExists(t, sourceRoot)

	newTarget := func(t *testing.T) (*Service, *ca.Authority, *noMintTestCustody, string) {
		t.Helper()
		root := t.TempDir()
		path := filepath.Join(root, "restored.sqlite")
		require.NoError(t, os.WriteFile(path, image, 0o600))
		db := fileWalletDB(t, path)
		authority, err := ca.NewAuthority(memoryStore{})
		require.NoError(t, err)
		custody := &noMintTestCustody{Custody: authority}
		s := NewService(NewSQLiteRepository(db), book.NewService(book.NewSQLiteRepository(db), clock), custody, clock, func(_ context.Context, rel string) (string, error) {
			return filepath.Join(root, rel), nil
		})
		return s, authority, custody, filepath.Join(root, backupName(in.ID))
	}

	t.Run("wrong_export_passphrase_leaves_fresh_target_empty", func(t *testing.T) {
		store := memoryStore{}
		authority, err := ca.NewAuthority(store)
		require.NoError(t, err)
		require.Error(t, authority.RestoreRecovery(portable, "wrong public fixture passphrase"))
		require.Empty(t, store)
	})
	t.Run("credential_values_do_not_reconstruct_wallet_metadata", func(t *testing.T) {
		authority, err := ca.NewAuthority(memoryStore{})
		require.NoError(t, err)
		require.NoError(t, authority.RestoreRecovery(portable, exportPassphrase))
		db := fileWalletDB(t, filepath.Join(t.TempDir(), "empty.sqlite"))
		custody := &noMintTestCustody{Custody: authority}
		s := NewService(NewSQLiteRepository(db), book.NewService(book.NewSQLiteRepository(db), clock), custody, clock, nil)
		_, err = s.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
		require.ErrorIs(t, err, ErrNotFound)
		require.Zero(t, custody.mintCalls)
	})
	t.Run("fresh_root_and_custody_preserve_binding_and_recovery", func(t *testing.T) {
		s, authority, custody, path := newTarget(t)
		require.NoError(t, authority.RestoreRecovery(portable, exportPassphrase))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, backup, 0o600))
		read, err := s.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
		require.NoError(t, err)
		require.Equal(t, w, read)
		qualified, err := s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
		require.NoError(t, err)
		require.Equal(t, w, qualified)
		// This deletion affects only a known-public value in the TEST map.
		require.NoError(t, authority.Delete(identity, keyField))
		recovered, err := s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
		require.NoError(t, err)
		require.Equal(t, w, recovered)
		again, err := s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
		require.NoError(t, err)
		require.Equal(t, w, again)
		_, err = s.Get(ctx, in.ID, "eip155:8453", in.OperatorIdentity)
		require.ErrorIs(t, err, ErrMismatch)
		_, err = s.Get(ctx, in.ID, in.Network, "other-operator")
		require.ErrorIs(t, err, ErrMismatch)
		persisted, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, backup, persisted)
		require.Zero(t, custody.mintCalls)
		require.NoDirExists(t, sourceRoot)
		require.Empty(t, sourceStore)
	})
	t.Run("missing_or_corrupt_original_backup_cannot_remint", func(t *testing.T) {
		s, authority, custody, path := newTarget(t)
		require.NoError(t, authority.RestoreRecovery(portable, exportPassphrase))
		require.NoError(t, authority.Delete(identity, keyField))
		_, err := s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
		require.ErrorIs(t, err, ErrUnavailable)
		require.False(t, authority.Status(identity, keyField).Configured)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		corrupt := append([]byte(nil), backup...)
		corrupt[len(corrupt)-1] ^= 1
		require.NoError(t, os.WriteFile(path, corrupt, 0o600))
		_, err = s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
		require.ErrorIs(t, err, ErrMismatch)
		require.ErrorContains(t, err, "recovery checksum mismatch")
		require.False(t, authority.Status(identity, keyField).Configured)
		retained, err := s.repository.Get(ctx, in.ID)
		require.NoError(t, err)
		require.Equal(t, w, retained)
		require.Zero(t, custody.mintCalls)
	})
}

func TestCreationMetadataFailureRecoversSameExistingCustody(t *testing.T) {
	s, a, in := setup(t)
	ctx := context.Background()
	db := s.repository.(*SQLiteRepository).db
	_, err := db.ExecContext(ctx, `CREATE TRIGGER fail_wallet_address BEFORE UPDATE OF address ON research_wallets WHEN NEW.address<>'' BEGIN SELECT RAISE(ABORT,'injected public binding failure'); END`)
	require.NoError(t, err)
	_, err = s.Create(ctx, in)
	require.ErrorContains(t, err, "injected public binding failure")
	pending, err := s.repository.Get(ctx, in.ID)
	require.NoError(t, err)
	require.Empty(t, pending.Address)
	id, err := ca.ParseIdentity(pending.CustodyReference)
	require.NoError(t, err)
	raw, err := a.Resolve(id, keyField)
	require.NoError(t, err)
	bytes, err := hex.DecodeString(raw)
	require.NoError(t, err)
	key := secp.PrivKeyFromBytes(bytes)
	defer key.Zero()
	clear(bytes)
	expected := Address(key.PubKey())
	_, err = db.ExecContext(ctx, `DROP TRIGGER fail_wallet_address`)
	require.NoError(t, err)
	ready, err := s.Create(ctx, in)
	require.NoError(t, err)
	require.Equal(t, pending.CustodyReference, ready.CustodyReference)
	require.Equal(t, expected, ready.Address)
	minted, err := s.repository.Minted(ctx, in.ID, false)
	require.NoError(t, err)
	require.True(t, minted)
}

func TestRecoveryMetadataFailureReusesPersistedPassphraseAndCiphertext(t *testing.T) {
	s, _, in := setup(t)
	ctx := context.Background()
	_, err := s.Create(ctx, in)
	require.NoError(t, err)
	db := s.repository.(*SQLiteRepository).db
	_, err = db.ExecContext(ctx, `CREATE TRIGGER fail_wallet_recovery BEFORE UPDATE OF recovery_sha256 ON research_wallets BEGIN SELECT RAISE(ABORT,'injected recovery metadata failure'); END`)
	require.NoError(t, err)
	_, err = s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.ErrorContains(t, err, "injected recovery metadata failure")
	path, err := s.recoveryPath(ctx, backupName(in.ID))
	require.NoError(t, err)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DROP TRIGGER fail_wallet_recovery`)
	require.NoError(t, err)
	w, err := s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, "same_host_encrypted_verified", w.RecoveryStatus)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func fileWalletDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(50)&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(Schema)))
	return db
}

func TestWalletMetadataAndCipherReadbackAfterActualDatabaseReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "wallets.sqlite")
	db := fileWalletDB(t, path)
	store := memoryStore{}
	authority, err := ca.NewAuthority(store)
	require.NoError(t, err)
	clock := schedule.System()
	books := book.NewService(book.NewSQLiteRepository(db), clock)
	_, err = books.Create(ctx, book.CreateInput{ID: "book", Name: "Fixture", BeneficiaryIdentity: "operator"})
	require.NoError(t, err)
	recovery := func(_ context.Context, rel string) (string, error) { return filepath.Join(dir, rel), nil }
	s := NewService(NewSQLiteRepository(db), books, authority, clock, recovery)
	in := CreateInput{ID: "wallet", BookID: "book", Network: "eip155:84532", OperatorIdentity: "operator"}
	w, err := s.Create(ctx, in)
	require.NoError(t, err)
	w, err = s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	freshDB := fileWalletDB(t, path)
	freshAuthority, err := ca.NewAuthority(store)
	require.NoError(t, err)
	fresh := NewService(NewSQLiteRepository(freshDB), book.NewService(book.NewSQLiteRepository(freshDB), clock), freshAuthority, clock, recovery)
	read, err := fresh.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, w.Address, read.Address)
	require.Equal(t, w.CustodyReference, read.CustodyReference)
	again, err := fresh.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, w.RecoverySHA256, again.RecoverySHA256)
}

type heldCustody struct {
	Custody
	entered chan struct{}
	release chan struct{}
}

func (c *heldCustody) ResolveOrMint(id ca.Identity, field string, w ca.Witness, mint func() (string, error)) (string, error) {
	if field == keyField {
		close(c.entered)
		<-c.release
	}
	return c.Custody.ResolveOrMint(id, field, w, mint)
}

func TestCustodyCreationHoldsRepositoryWriterLockAcrossCredentialOperation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "locking.sqlite")
	db := fileWalletDB(t, path)
	other := fileWalletDB(t, path)
	authority, err := ca.NewAuthority(memoryStore{})
	require.NoError(t, err)
	held := &heldCustody{Custody: authority, entered: make(chan struct{}), release: make(chan struct{})}
	clock := schedule.System()
	books := book.NewService(book.NewSQLiteRepository(db), clock)
	_, err = books.Create(ctx, book.CreateInput{ID: "book", Name: "Fixture", BeneficiaryIdentity: "operator"})
	require.NoError(t, err)
	s := NewService(NewSQLiteRepository(db), books, held, clock, nil)
	result := make(chan error, 1)
	go func() {
		_, err := s.Create(ctx, CreateInput{ID: "wallet", BookID: "book", Network: "eip155:84532", OperatorIdentity: "operator"})
		result <- err
	}()
	released := false
	defer func() {
		if !released {
			close(held.release)
		}
	}()
	select {
	case <-held.entered:
	case err := <-result:
		t.Fatalf("creation ended before custody boundary: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("custody boundary was not reached")
	}
	_, err = other.ExecContext(ctx, `UPDATE research_wallets SET key_minted=key_minted WHERE id='wallet'`)
	require.Error(t, err, "another connection must not write while custody creation is in progress")
	close(held.release)
	released = true
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("creation failed to close after custody release")
	}
}
