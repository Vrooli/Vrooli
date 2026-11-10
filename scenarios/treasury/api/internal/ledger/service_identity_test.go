package ledger

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type serviceCredentialFixture struct {
	value string
	err   error
	reads int
}

func (f *serviceCredentialFixture) Resolve(_ context.Context, identity, field string) (string, error) {
	if identity != TreasuryServiceCredentialIdentity || field != "credential" {
		return "", errors.New("wrong authority identity")
	}
	f.reads++
	return f.value, f.err
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
	source, err := NewServiceTokenSource(authority, server.URL, server.Client())
	require.NoError(t, err)
	source.now = func() time.Time { return now }
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
