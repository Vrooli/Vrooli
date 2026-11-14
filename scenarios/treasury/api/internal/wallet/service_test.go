package wallet

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	secp "github.com/go-webauthn/x/crypto/secp256k1"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"

	"treasury/internal/book"
)

type memoryStore map[string]string

func (m memoryStore) Put(service, key, value string) error { m[service+"/"+key] = value; return nil }
func (m memoryStore) Get(service, key string) (string, error) {
	v, ok := m[service+"/"+key]
	if !ok {
		return "", ca.ErrNotFound
	}
	return v, nil
}
func (m memoryStore) Delete(service, key string) error { delete(m, service+"/"+key); return nil }
func setup(t *testing.T) (*Service, *ca.Authority, CreateInput) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(Schema)))
	books := book.NewService(book.NewSQLiteRepository(db), schedule.System())
	_, err := books.Create(ctx, book.CreateInput{ID: "research", Name: "Research", BeneficiaryIdentity: "local-operator"})
	require.NoError(t, err)
	authority, err := ca.NewAuthority(memoryStore{})
	require.NoError(t, err)
	dir := t.TempDir()
	s := NewService(NewSQLiteRepository(db), books, authority, schedule.System(), func(_ context.Context, name string) (string, error) { return filepath.Join(dir, name), nil })
	return s, authority, CreateInput{ID: "wallet-test", BookID: "research", Network: "eip155:84532", OperatorIdentity: "local-operator"}
}

func TestAddressKnownSecp256k1Vector(t *testing.T) {
	// Published non-secret scalar1 is a derivation vector only, never provisioned.
	b := make([]byte, 32)
	b[31] = 1
	k := secp.PrivKeyFromBytes(b)
	defer k.Zero()
	require.Equal(t, "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf", Address(k.PubKey()))
}

func TestMalformedOrOutOfRangeCustodyScalarsNeverNormalizeIntoAnotherWallet(t *testing.T) {
	for name, scalar := range map[string]string{
		"zero":        strings.Repeat("0", 64),
		"curve_order": "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141",
		"short":       strings.Repeat("1", 62),
		"long":        strings.Repeat("1", 66),
		"nonhex":      strings.Repeat("z", 64),
	} {
		t.Run(name, func(t *testing.T) {
			s, authority, in := setup(t)
			ctx := context.Background()
			original, err := s.Create(ctx, in)
			require.NoError(t, err)
			original, err = s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
			require.NoError(t, err)
			identity, err := ca.ParseIdentity(original.CustodyReference)
			require.NoError(t, err)
			require.NoError(t, authority.Put(identity, keyField, scalar))
			_, err = s.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
			require.ErrorIs(t, err, ErrMismatch)
			_, err = s.Prove(ctx, in.ID, in.Network, in.OperatorIdentity, "invalid_scalar_nonce_20261027")
			require.ErrorIs(t, err, ErrMismatch)
			_, err = s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
			require.ErrorIs(t, err, ErrMismatch)
			_, err = s.Create(ctx, in)
			require.ErrorIs(t, err, ErrMismatch)
			stored, err := s.repository.Get(ctx, in.ID)
			require.NoError(t, err)
			require.Equal(t, original.Address, stored.Address)
			require.Equal(t, original.PublicKey, stored.PublicKey)
			require.Equal(t, original.CustodyReference, stored.CustodyReference)
		})
	}
}

func TestWalletCreateIdempotentAndScopeBound(t *testing.T) {
	s, _, in := setup(t)
	ctx := context.Background()
	a, err := s.Create(ctx, in)
	require.NoError(t, err)
	require.Len(t, a.Address, 42)
	b, err := s.Create(ctx, in)
	require.NoError(t, err)
	require.Equal(t, a, b)
	changed := in
	changed.Network = "eip155:8453"
	_, err = s.Create(ctx, changed)
	require.ErrorIs(t, err, ErrMismatch)
	_, err = s.Get(ctx, a.ID, "eip155:8453", in.OperatorIdentity)
	require.ErrorIs(t, err, ErrMismatch)
	changed = in
	changed.OperatorIdentity = "fictional-artist"
	_, err = s.Create(ctx, changed)
	require.Error(t, err)
}

func TestWalletControlProofRejectsChangedNetworkMessageAddressAndNonce(t *testing.T) {
	s, _, in := setup(t)
	ctx := context.Background()
	_, err := s.Create(ctx, in)
	require.NoError(t, err)
	nonce := "research_nonce_20261027"
	p, err := s.Prove(ctx, in.ID, in.Network, in.OperatorIdentity, nonce)
	require.NoError(t, err)
	require.True(t, Verify(p, nonce))
	for _, mutate := range []func(*Proof){func(p *Proof) { p.Scheme = "payment" }, func(p *Proof) { p.Wallet.Network = "eip155:8453" }, func(p *Proof) { p.Message += "pay" }, func(p *Proof) { p.Wallet.Address = "0x" + strings.Repeat("1", 40) }, func(p *Proof) { p.Signature = "0x" + strings.Repeat("0", 130) }, func(p *Proof) { p.Wallet.BookID = "other" }, func(p *Proof) { p.Wallet.OperatorIdentity = "artist" }} {
		q := p
		mutate(&q)
		require.False(t, Verify(q, nonce))
	}
	require.False(t, Verify(p, "different_nonce_20261027"))
	_, err = s.Prove(ctx, in.ID, in.Network, in.OperatorIdentity, "pay\nanything")
	require.ErrorIs(t, err, ErrInvalid)
}

func TestWalletFreshHandleAndMissingKeyNeverRemints(t *testing.T) {
	s, a, in := setup(t)
	ctx := context.Background()
	w, err := s.Create(ctx, in)
	require.NoError(t, err)
	fresh := NewService(s.repository, s.books, a, s.clock, s.recoveryPath)
	got, err := fresh.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, w.Address, got.Address)
	id, err := ca.ParseIdentity(w.CustodyReference)
	require.NoError(t, err)
	require.NoError(t, a.Delete(id, keyField))
	_, err = fresh.Create(ctx, in)
	require.Error(t, err)
	require.False(t, a.Status(id, keyField).Configured)
}

func TestWalletRecoveryIsEncryptedScopedAndRestorable(t *testing.T) {
	s, a, in := setup(t)
	ctx := context.Background()
	w, err := s.Create(ctx, in)
	require.NoError(t, err)
	w, err = s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, "same_host_encrypted_verified", w.RecoveryStatus)
	path, err := s.recoveryPath(ctx, backupName(w.ID))
	require.NoError(t, err)
	blob, err := os.ReadFile(path)
	require.NoError(t, err)
	id, err := ca.ParseIdentity(w.CustodyReference)
	require.NoError(t, err)
	raw, err := a.Resolve(id, keyField)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(blob), raw), "encrypted recovery contains plaintext custody material")
	pass, err := a.Resolve(id, recoveryField)
	require.NoError(t, err)
	fresh, err := ca.NewAuthority(memoryStore{})
	require.NoError(t, err)
	require.Error(t, fresh.RestoreRecovery(blob, "deliberately_wrong_passphrase"))
	require.False(t, fresh.Status(id, keyField).Configured)
	require.NoError(t, fresh.RestoreRecovery(blob, pass))
	restored, err := fresh.Resolve(id, keyField)
	require.NoError(t, err)
	b, err := hex.DecodeString(restored)
	require.NoError(t, err)
	k := secp.PrivKeyFromBytes(b)
	defer k.Zero()
	require.Equal(t, w.Address, Address(k.PubKey()))
	clear(b)
	stat, err := os.Stat(path)
	require.NoError(t, err)
	require.EqualValues(t, 0o600, stat.Mode().Perm())
	freshSvc := NewService(s.repository, s.books, a, s.clock, s.recoveryPath)
	again, err := freshSvc.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, w.RecoverySHA256, again.RecoverySHA256)
	require.NoError(t, os.WriteFile(path, append(blob, byte(1)), 0o600))
	_, err = freshSvc.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.Error(t, err)
}

func TestQualifiedRecoveryRejectsChangedCiphertextBeforeOpening(t *testing.T) {
	s, authority, in := setup(t)
	ctx := context.Background()
	_, err := s.Create(ctx, in)
	require.NoError(t, err)
	w, err := s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	path, err := s.recoveryPath(ctx, backupName(w.ID))
	require.NoError(t, err)
	original, err := os.ReadFile(path)
	require.NoError(t, err)
	id, err := ca.ParseIdentity(w.CustodyReference)
	require.NoError(t, err)
	keyStatus := authority.Status(id, keyField)
	passphraseStatus := authority.Status(id, recoveryField)
	for name, corrupted := range map[string][]byte{
		"invalid_json":       []byte("not a recovery envelope"),
		"unsupported_policy": []byte(`{"version":999,"iterations":0}`),
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(path, corrupted, 0o600))
			_, err := s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
			require.ErrorIs(t, err, ErrMismatch)
			require.ErrorContains(t, err, "recovery checksum mismatch")
			keyAfter := authority.Status(id, keyField)
			passphraseAfter := authority.Status(id, recoveryField)
			require.Equal(t, keyStatus.Configured, keyAfter.Configured)
			require.Equal(t, keyStatus.Version, keyAfter.Version)
			require.Equal(t, passphraseStatus.Configured, passphraseAfter.Configured)
			require.Equal(t, passphraseStatus.Version, passphraseAfter.Version)
			read, err := s.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
			require.NoError(t, err)
			require.Equal(t, w, read)
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, corrupted, after)
		})
	}
	require.NoError(t, os.WriteFile(path, original, 0o600))
	again, err := s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, w, again)
}

func TestWalletRefusesTestModeLiveCredentials(t *testing.T) {
	s, _, in := setup(t)
	_, err := s.Create(database.WithTestMode(context.Background()), in)
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestWalletCustodyIdentitySeparatesNetworkAndBookAcrossRepositories(t *testing.T) {
	first, authority, in := setup(t)
	second, _, _ := setup(t)
	second.custody = authority
	a, err := first.Create(context.Background(), in)
	require.NoError(t, err)
	in.Network = "eip155:8453"
	b, err := second.Create(context.Background(), in)
	require.NoError(t, err)
	require.NotEqual(t, a.CustodyReference, b.CustodyReference)
	require.NotEqual(t, a.Address, b.Address)
}

func TestWalletProviderFailureAndReferenceTamperingRefuse(t *testing.T) {
	s, authority, in := setup(t)
	ctx := context.Background()
	unavailable, err := ca.Unavailable("isolated unavailable provider")
	require.NoError(t, err)
	s.custody = unavailable
	_, err = s.Create(ctx, in)
	require.ErrorIs(t, err, ErrUnavailable)
	s.custody = authority
	w, err := s.Create(ctx, in)
	require.NoError(t, err)
	_, err = s.repository.(*SQLiteRepository).db.ExecContext(ctx, `UPDATE research_wallets SET custody_reference=? WHERE id=?`, reference(CreateInput{ID: "different", BookID: in.BookID, Network: in.Network, OperatorIdentity: in.OperatorIdentity}, strings.Repeat("0", 32)), w.ID)
	require.NoError(t, err)
	_, err = s.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.ErrorIs(t, err, ErrMismatch)
}

func TestWalletRecoveryPassphraseLossCannotRemintOrReplaceBundle(t *testing.T) {
	s, a, in := setup(t)
	ctx := context.Background()
	_, err := s.Create(ctx, in)
	require.NoError(t, err)
	w, err := s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	path, err := s.recoveryPath(ctx, backupName(w.ID))
	require.NoError(t, err)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	id, err := ca.ParseIdentity(w.CustodyReference)
	require.NoError(t, err)
	require.NoError(t, a.Delete(id, recoveryField))
	_, err = s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.Error(t, err)
	require.False(t, a.Status(id, recoveryField).Configured)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestWalletRecoveryRestoresOnlyMissingWitnessedScalar(t *testing.T) {
	s, a, in := setup(t)
	ctx := context.Background()
	_, err := s.Create(ctx, in)
	require.NoError(t, err)
	w, err := s.QualifyRecovery(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	id, err := ca.ParseIdentity(w.CustodyReference)
	require.NoError(t, err)
	require.NoError(t, a.Delete(id, keyField))
	_, err = s.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.Error(t, err)
	restored, err := s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, w.Address, restored.Address)
	require.Equal(t, w.PublicKey, restored.PublicKey)
	again, err := s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.NoError(t, err)
	require.Equal(t, w.Address, again.Address)
	require.NoError(t, a.Delete(id, keyField))
	_, err = s.repository.(*SQLiteRepository).db.ExecContext(ctx, `UPDATE research_wallets SET address=? WHERE id=?`, "0x1111111111111111111111111111111111111111", w.ID)
	require.NoError(t, err)
	_, err = s.Recover(ctx, in.ID, in.Network, in.OperatorIdentity)
	require.ErrorIs(t, err, ErrMismatch)
	require.False(t, a.Status(id, keyField).Configured, "mismatched staged recovery must never write active custody")
}

func TestIndependentCustodyRepositoriesDoNotAliasEqualWalletAndBookIds(t *testing.T) {
	first, authority, in := setup(t)
	second, _, _ := setup(t)
	second.custody = authority
	a, err := first.Create(context.Background(), in)
	require.NoError(t, err)
	b, err := second.Create(context.Background(), in)
	require.NoError(t, err)
	require.NotEqual(t, a.CustodyReference, b.CustodyReference, "independent Treasury databases must not silently share custody from equal display IDs")
	require.NotEqual(t, a.Address, b.Address)
}
