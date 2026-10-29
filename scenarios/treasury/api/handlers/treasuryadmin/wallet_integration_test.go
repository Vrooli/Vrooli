package treasuryadmin_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	ca "github.com/vrooli/vrooli/packages/credential-authority-go"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	"treasury/handlers/treasuryadmin"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/operatorauth"
	"treasury/internal/wallet"
)

// This test backend is never persisted or used with host credentials. Reopening
// its Authority wrapper qualifies wiring, not recovery of a host keystore.
type walletTestCredentials struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *walletTestCredentials) Put(service, field, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[service+"\x00"+field] = value
	return nil
}

func (m *walletTestCredentials) Get(service, field string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[service+"\x00"+field]
	if !ok {
		return "", ca.ErrNotFound
	}
	return value, nil
}

func (m *walletTestCredentials) Delete(service, field string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, service+"\x00"+field)
	return nil
}

func TestResearchWalletActualOwnerHTTPRoundTripAndReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	credentials := &walletTestCredentials{values: map[string]string{}}
	open := func() (*sql.DB, authconnect.TreasuryAdminClient, func()) {
		db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "owner.sqlite")+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(1000)")
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		require.NoError(t, database.EnsureSchemas(ctx, db,
			database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema),
			database.SchemaProviderFunc(mandate.Schema), database.SchemaProviderFunc(instrument.Schema), database.SchemaProviderFunc(wallet.Schema)))
		authority, err := ca.NewAuthority(credentials)
		require.NoError(t, err)
		books := book.NewService(book.NewSQLiteRepository(db), schedule.System())
		wallets := wallet.NewService(wallet.NewSQLiteRepository(db), books, authority, schedule.System(), func(_ context.Context, rel string) (string, error) { return filepath.Join(dir, rel), nil })
		authorizer, err := operatorauth.NewStaticToken("isolated-test-operator")
		require.NoError(t, err)
		_, handler := authconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, books, wallets))
		server := httptest.NewServer(handler)
		return db, authconnect.NewTreasuryAdminClient(server.Client(), server.URL), func() { server.Close(); require.NoError(t, db.Close()) }
	}
	operator := func(h http.Header) { h.Set(operatorauth.HeaderOperatorToken, "isolated-test-operator") }
	db, client, closeOwner := open()
	closed := false
	defer func() {
		if !closed {
			closeOwner()
		}
	}()
	bookRequest := connect.NewRequest(&authv1.CreateBookRequest{Book: &bookv1.Book{Id: "receive-fixture", Name: "Isolated receive fixture", BeneficiaryIdentity: "untrusted-subject"}})
	operator(bookRequest.Header())
	bookResponse, err := client.CreateBook(ctx, bookRequest)
	require.NoError(t, err)
	require.Equal(t, "local-operator", bookResponse.Msg.Book.BeneficiaryIdentity)
	create := connect.NewRequest(&authv1.CreateResearchWalletRequest{Id: "receive-fixture", BookId: "receive-fixture", Network: "eip155:84532"})
	operator(create.Header())
	created, err := client.CreateResearchWallet(ctx, create)
	require.NoError(t, err)
	w := created.Msg.Wallet
	require.Equal(t, "local-operator", w.OperatorIdentity)
	require.Len(t, w.Address, 42)
	require.False(t, w.IndependentRecoveryVerified)
	nonce := "http_control_nonce_20261027"
	proofRequest := connect.NewRequest(&authv1.ProveResearchWalletRequest{Id: w.Id, Network: w.Network, Nonce: nonce})
	operator(proofRequest.Header())
	proofResponse, err := client.ProveResearchWallet(ctx, proofRequest)
	require.NoError(t, err)
	p := proofResponse.Msg.Proof
	require.True(t, p.Verified)
	require.True(t, wallet.Verify(wallet.Proof{Wallet: wallet.Wallet{ID: w.Id, BookID: w.BookId, OperatorIdentity: w.OperatorIdentity, Network: w.Network, Address: w.Address, PublicKey: w.PublicKey}, Message: p.Message, Signature: p.Signature, Scheme: p.Scheme}, nonce))
	qualify := connect.NewRequest(&authv1.ResearchWalletRequest{Id: w.Id, Network: w.Network})
	operator(qualify.Header())
	backup, err := client.QualifyResearchWalletRecovery(ctx, qualify)
	require.NoError(t, err)
	require.Equal(t, "same_host_encrypted_verified", backup.Msg.Wallet.RecoveryStatus)
	require.False(t, backup.Msg.Wallet.IndependentRecoveryVerified)
	require.Len(t, backup.Msg.Wallet.RecoverySha256, 64)
	for _, table := range []string{"budgets", "mandates", "instruments"} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count, "receive provisioning must not create spending controls")
	}
	closeOwner()
	closed = true
	freshDB, fresh, closeFresh := open()
	defer closeFresh()
	get := connect.NewRequest(&authv1.ResearchWalletRequest{Id: w.Id, Network: w.Network})
	operator(get.Header())
	readback, err := fresh.GetResearchWallet(ctx, get)
	require.NoError(t, err)
	require.Equal(t, w.Address, readback.Msg.Wallet.Address)
	require.Equal(t, w.CustodyReference, readback.Msg.Wallet.CustodyReference)
	again, err := fresh.QualifyResearchWalletRecovery(ctx, qualify)
	require.NoError(t, err)
	require.Equal(t, backup.Msg.Wallet.RecoverySha256, again.Msg.Wallet.RecoverySha256)
	// Delete only this isolated TEST backend's scalar, then exercise the real
	// operator RPC. Neither the host credential store nor a host bearer is used.
	fixtureAuthority, err := ca.NewAuthority(credentials)
	require.NoError(t, err)
	identity, err := ca.ParseIdentity(w.CustodyReference)
	require.NoError(t, err)
	require.NoError(t, fixtureAuthority.Delete(identity, "secp256k1-scalar"))
	_, err = fresh.GetResearchWallet(ctx, get)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	restored, err := fresh.RecoverResearchWallet(ctx, get)
	require.NoError(t, err)
	require.Equal(t, w.Address, restored.Msg.Wallet.Address)
	require.Equal(t, w.CustodyReference, restored.Msg.Wallet.CustodyReference)
	idempotent, err := fresh.RecoverResearchWallet(ctx, get)
	require.NoError(t, err)
	require.Equal(t, restored.Msg.Wallet.Address, idempotent.Msg.Wallet.Address)
	proofAfterRestore, err := fresh.ProveResearchWallet(ctx, proofRequest)
	require.NoError(t, err)
	require.True(t, proofAfterRestore.Msg.Proof.Verified)
	wrongNetwork := connect.NewRequest(&authv1.ResearchWalletRequest{Id: w.Id, Network: "eip155:8453"})
	operator(wrongNetwork.Header())
	_, err = fresh.GetResearchWallet(ctx, wrongNetwork)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	create.Header().Set(operatorauth.HeaderAgentToken, "isolated-agent")
	_, err = fresh.CreateResearchWallet(ctx, create)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	var count int
	require.NoError(t, freshDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM research_wallets").Scan(&count))
	require.Equal(t, 1, count)
}
