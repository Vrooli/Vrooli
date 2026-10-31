package surfaces

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
	accounts "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"net/http/httptest"
	"testing"
)

type operatorAccountFixture struct {
	accounts_v1connect.UnimplementedAccountsServiceHandler
}

func (operatorAccountFixture) Login(_ context.Context, r *connect.Request[accounts.LoginRequest]) (*connect.Response[accounts.LoginResponse], error) {
	if r.Msg.Email != "owner@example.test" || r.Msg.Password != "correct" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("private database detail"))
	}
	if r.Header().Get("Cookie") != "" || r.Header().Get("Authorization") != "" {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unexpected forwarded authority"))
	}
	return connect.NewResponse(&accounts.LoginResponse{Tokens: &accounts.TokenPair{AccessToken: "issued-token"}}), nil
}
func TestOperatorLoginUsesDiscoveredOwnerAndPrivateResponse(t *testing.T) {
	_, handler := accounts_v1connect.NewAccountsServiceHandler(operatorAccountFixture{})
	authenticator := httptest.NewServer(handler)
	defer authenticator.Close()
	h := &OperatorHandler{resolve: func(_ context.Context, name string) (string, error) {
		require.Equal(t, "scenario-authenticator", name)
		return authenticator.URL, nil
	}, client: authenticator.Client()}
	router := mux.NewRouter()
	operatorModule(h).Mount(router)
	portal := httptest.NewServer(router)
	defer portal.Close()
	client := surfacesv1connect.NewOperatorSessionServiceClient(portal.Client(), portal.URL)
	r := connect.NewRequest(&accounts.LoginRequest{Email: "owner@example.test", Password: "correct"})
	r.Header().Set("Cookie", "browser-cookie")
	r.Header().Set("Authorization", "Bearer unrelated")
	result, err := client.Login(context.Background(), r)
	require.NoError(t, err)
	require.Equal(t, "issued-token", result.Msg.Tokens.AccessToken)
	require.Equal(t, "no-store", result.Header().Get("Cache-Control"))
	r.Msg.Password = "wrong"
	_, err = client.Login(context.Background(), r)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "private database detail")
}
