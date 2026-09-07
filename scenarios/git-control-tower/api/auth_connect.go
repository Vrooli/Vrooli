package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auth"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auth/auth_v1connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

// authConnectServer is a same-origin browser facade. GCT does not own account
// state, passwords, signing keys, or refresh-token policy; it forwards Login
// to scenario-authenticator over the typed AccountsService contract.
type authConnectServer struct{}

func (authConnectServer) Login(ctx context.Context, req *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	email := strings.TrimSpace(req.Msg.GetEmail())
	password := req.Msg.GetPassword()
	if email == "" || password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and password are required"))
	}

	base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	if err != nil || strings.TrimSpace(base) == "" {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-authenticator is unavailable"))
	}
	client := accountsconnect.NewAccountsServiceClient(http.DefaultClient, strings.TrimRight(base, "/"))
	response, err := client.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{
		Email: email, Password: password, Realm: "default",
	}))
	if err != nil {
		switch connect.CodeOf(err) {
		case connect.CodeUnauthenticated:
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid email or password"))
		case connect.CodeInvalidArgument:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid sign-in details"))
		default:
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-authenticator is unavailable"))
		}
	}
	if response == nil || response.Msg.GetTokens() == nil || strings.TrimSpace(response.Msg.GetTokens().GetAccessToken()) == "" {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario-authenticator returned no access token"))
	}
	account := response.Msg.GetAccount()
	return connect.NewResponse(&authv1.LoginResponse{
		AccessToken:  response.Msg.GetTokens().GetAccessToken(),
		RefreshToken: response.Msg.GetTokens().GetRefreshToken(),
		Email:        account.GetEmail(),
		UserId:       account.GetId(),
	}), nil
}

var _ authconnect.AuthServiceHandler = authConnectServer{}
