package ledger

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/api-core/servicetoken"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type serviceCredentialFixture struct {
	value    string
	err      error
	reads    int
	identity string
}

func (f *serviceCredentialFixture) Resolve(_ context.Context, identity, field string) (string, error) {
	expected := f.identity
	if expected == "" {
		expected = "scenario-authenticator/service-principal/treasury-money-ledger/money-ledger"
	}
	if identity != expected || field != "credential" {
		return "", errors.New("wrong authority identity")
	}
	f.reads++
	return f.value, f.err
}

func TestServiceTokenSourceCanUseDistinctAudienceBoundPrincipal(t *testing.T) {
	authority := &serviceCredentialFixture{value: "notice-credential", identity: "scenario-authenticator/service-principal/treasury-notification-hub/notification-hub"}
	exchange := &exchangeFixture{credential: "notice-credential", token: "notice-service-jwt", expires: time.Now().Add(time.Hour)}
	_, handler := accountsconnect.NewAccountsServiceHandler(exchange)
	server := httptest.NewServer(handler)
	defer server.Close()
	source, err := NewServiceTokenSourceFor(authority, server.URL, server.Client(), NoticePrincipal, NoticeResource)
	require.NoError(t, err)
	token, err := source.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, "notice-service-jwt", token)
	require.Equal(t, 1, authority.reads)
	require.Equal(t, 1, exchange.calls)
	if _, err := NewServiceTokenSourceFor(authority, server.URL, server.Client(), "", "notification-hub"); err == nil {
		t.Fatal("service name is required")
	}
}

type exchangeFixture struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	credential string
	token      string
	expires    time.Time
	calls      int
}

func (f *exchangeFixture) ExchangeServiceCredential(_ context.Context, req *connect.Request[accountsv1.ExchangeServiceCredentialRequest]) (*connect.Response[accountsv1.ExchangeServiceCredentialResponse], error) {
	f.calls++
	if req.Msg.Credential != f.credential {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("credential revoked"))
	}
	return connect.NewResponse(&accountsv1.ExchangeServiceCredentialResponse{AccessToken: f.token, AccessTokenExpiresAt: timestamppb.New(f.expires)}), nil
}

func TestServiceIdentityExchangesAuthorityCredentialAndRefreshesAfterRotation(t *testing.T) {
	now := time.Date(2026, 11, 8, 20, 0, 0, 0, time.UTC)
	authority := &serviceCredentialFixture{value: "first-credential"}
	exchange := &exchangeFixture{credential: authority.value, token: "first-service-jwt", expires: now.Add(10 * time.Minute)}
	_, handler := accountsconnect.NewAccountsServiceHandler(exchange)
	server := httptest.NewServer(handler)
	defer server.Close()
	source, err := NewServiceTokenSource(authority, server.URL, server.Client(), servicetoken.WithClock(func() time.Time { return now }))
	require.NoError(t, err)
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			token, err := source.Token(context.Background())
			require.NoError(t, err)
			require.Equal(t, "first-service-jwt", token)
		})
	}
	wg.Wait()
	require.Equal(t, 1, authority.reads)
	require.Equal(t, 1, exchange.calls)
	now = now.Add(9 * time.Minute)
	authority.value, exchange.credential, exchange.token = "rotated-credential", "rotated-credential", "rotated-service-jwt"
	exchange.expires = now.Add(10 * time.Minute)
	token, err := source.Token(context.Background())
	require.NoError(t, err)
	require.Equal(t, "rotated-service-jwt", token)
	require.Equal(t, 2, authority.reads)
	now = now.Add(10 * time.Minute)
	authority.err = errors.New("authority unavailable")
	token, err = source.Token(context.Background())
	require.Error(t, err)
	require.Empty(t, token, "an expired service token cannot survive failed credential refresh")
}

func TestServiceIdentityRefusesRevokedCredentialOrExpiredExchange(t *testing.T) {
	now := time.Now()
	authority := &serviceCredentialFixture{value: "revoked"}
	exchange := &exchangeFixture{credential: "current", token: "expired-service-jwt", expires: now.Add(-time.Second)}
	_, handler := accountsconnect.NewAccountsServiceHandler(exchange)
	server := httptest.NewServer(handler)
	defer server.Close()
	source, err := NewServiceTokenSource(authority, server.URL, server.Client())
	require.NoError(t, err)
	_, err = source.Token(context.Background())
	require.ErrorContains(t, err, "exchange failed")
	authority.value = "current"
	_, err = source.Token(context.Background())
	require.ErrorContains(t, err, "no current token")
}

// The terse texts are deliberate: no reason from the authority or authenticator
// reaches a log line.
func TestServiceIdentityErrorsNeverCarryTheUnderlyingReason(t *testing.T) {
	authority := &serviceCredentialFixture{err: errors.New("authority path /secret/entry unreadable")}
	source, err := NewServiceTokenSource(authority, "http://127.0.0.1:1", nil)
	require.NoError(t, err)
	_, err = source.Token(context.Background())
	require.EqualError(t, err, "Treasury service credential is unavailable")
	require.ErrorIs(t, err, servicetoken.ErrNotIssued)
	authority.err, authority.value = nil, "issued-credential"
	_, err = source.Token(context.Background())
	require.EqualError(t, err, "Treasury service credential exchange failed")
	require.ErrorIs(t, err, servicetoken.ErrExchangeFailed)
}

// Treasury holds one principal per peer, so each peer can accept exactly its own
// and a token minted for one can never satisfy another. The credential a token
// source reads is the one `service-principals sync --apply` issues for that name.
func TestEachPeerPrincipalReadsItsOwnCredentialAddress(t *testing.T) {
	for _, tc := range []struct {
		name, principal, resource, address string
	}{
		{"money ledger", MoneyLedgerPrincipal, MoneyLedgerResource, "scenario-authenticator/service-principal/treasury-money-ledger/money-ledger"},
		{"agent manager", AgentManagerPrincipal, AgentManagerResource, "scenario-authenticator/service-principal/treasury-agent-manager/agent-manager"},
		{"notification hub", NoticePrincipal, NoticeResource, "scenario-authenticator/service-principal/treasury-notification-hub/notification-hub"},
	} {
		authority := &serviceCredentialFixture{value: tc.name + "-credential", identity: tc.address}
		exchange := &exchangeFixture{credential: authority.value, token: tc.name + "-jwt", expires: time.Now().Add(time.Hour)}
		_, handler := accountsconnect.NewAccountsServiceHandler(exchange)
		server := httptest.NewServer(handler)
		source, err := NewServiceTokenSourceFor(authority, server.URL, server.Client(), tc.principal, tc.resource)
		require.NoError(t, err, tc.name)
		token, err := source.Token(context.Background())
		require.NoError(t, err, "%s must read exactly %s", tc.name, tc.address)
		require.Equal(t, tc.name+"-jwt", token)
		server.Close()
	}
	require.Equal(t, hubsender.Principal("treasury"), NoticePrincipal, "the Hub reads the source off the sender principal's name")
	require.Equal(t, hubsender.CredentialIdentity("treasury"), TreasuryNoticeCredentialIdentity)
	require.Equal(t, "scenario-authenticator/service-principal/treasury-money-ledger/money-ledger", TreasuryServiceCredentialIdentity)
	require.Equal(t, "scenario-authenticator/service-principal/treasury-agent-manager/agent-manager", TreasuryAgentManagerCredentialIdentity)
}

// The operator issues these with one command from .vrooli/service-principals.json,
// so the file has to say exactly what the code asks the authenticator for: the
// name, the resource, the audience each peer verifies and its one scope.
func TestTreasuryDeclaresExactlyTheServicePrincipalsItUses(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", filepath.Base(serviceprincipal.DeclarationFile)))
	require.NoError(t, err)
	declared, err := serviceprincipal.ParseDeclarations(raw)
	require.NoError(t, err)
	byName := map[string]serviceprincipal.Declaration{}
	for _, d := range declared {
		_, duplicate := byName[d.Name]
		require.False(t, duplicate, "principal %s is declared twice", d.Name)
		byName[d.Name] = d
	}
	for _, want := range []serviceprincipal.Declaration{
		{Name: MoneyLedgerPrincipal, Resource: MoneyLedgerResource, Audience: "scenario-authenticator:money-ledger", Scopes: []string{"money-ledger:write"}},
		{Name: AgentManagerPrincipal, Resource: AgentManagerResource, Audience: "scenario-authenticator:agent-manager", Scopes: []string{"agent-manager:write"}},
		{Name: NoticePrincipal, Resource: NoticeResource, Audience: hubsender.Audience, Scopes: []string{hubsender.Scope}},
		{Name: lpbsidentity.TreasuryAcceptedReceiptCredit.Principal("treasury"), Resource: lpbsidentity.TreasuryAcceptedReceiptCredit.Resource, Audience: lpbsidentity.TreasuryAcceptedReceiptCredit.Audience, Scopes: []string{lpbsidentity.TreasuryAcceptedReceiptCredit.Scope}},
	} {
		require.Equal(t, want, byName[want.Name], "Treasury must declare %s exactly", want.Name)
	}
	_, shared := byName["treasury"]
	require.False(t, shared, "one name across three resources cannot be issued; each peer has its own principal")
}
