package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"

	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/auth"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/auth/authv1connect"
)

const onboardingAccessTokenCookieName = "onboarding_access_token"

// onboardingAuthConnectServer is a same-origin browser facade. Onboarding
// does not own account state, passwords, signing keys, or refresh policy; it
// forwards Login to scenario-authenticator and keeps the issued access token
// in an HttpOnly cookie for the API authentication middleware.
type onboardingAuthConnectServer struct{}

func (onboardingAuthConnectServer) Login(ctx context.Context, req *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
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
	tokens := response.Msg.GetTokens()
	out := connect.NewResponse(&authv1.LoginResponse{Email: account.GetEmail(), UserId: account.GetId()})
	maxAge := 1
	if expiresAt := tokens.GetAccessTokenExpiresAt(); expiresAt != nil {
		if seconds := int(time.Until(expiresAt.AsTime()).Seconds()); seconds > 0 {
			maxAge = seconds
		}
	}
	secure := false
	if configured := strings.TrimSpace(os.Getenv("VROOLI_AUTH_COOKIE_SECURE")); configured != "" {
		secure, _ = strconv.ParseBool(configured)
	} else {
		secure = strings.EqualFold(strings.TrimSpace(req.Header().Get("X-Forwarded-Proto")), "https")
	}
	out.Header().Add("Set-Cookie", (&http.Cookie{
		Name: onboardingAccessTokenCookieName, Value: tokens.GetAccessToken(), Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	}).String())
	return out, nil
}

var _ authconnect.AuthServiceHandler = onboardingAuthConnectServer{}

func mountOnboardingAuthHandler(router *mux.Router) {
	path, handler := authconnect.NewAuthServiceHandler(onboardingAuthConnectServer{})
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
