package identity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

// Typed outcomes of a forwarded sign-in. Invalid credentials and a locked
// account both collapse to ErrInvalidCredentials so nothing leaks which.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidInput       = errors.New("invalid sign-in input")
	ErrAuthUnavailable    = errors.New("scenario-authenticator is unavailable")
)

// URLResolver resolves the authenticator's base URL by scenario name.
type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// Session is what a successful sign-in or refresh returns. The hub keeps
// neither token; it hands both to the browser as HttpOnly cookies.
type Session struct {
	AccessToken, RefreshToken, Email, UserID string
}

// Forwarder relays sign-in to scenario-authenticator. It owns no credential
// logic: it stores nothing, mints nothing, and verifies no password.
type Forwarder struct {
	Resolver URLResolver
	Client   connect.HTTPClient
}

func (f Forwarder) Login(ctx context.Context, email, password string) (Session, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return Session{}, fmt.Errorf("%w: email and password are required", ErrInvalidInput)
	}
	client, err := f.client(ctx)
	if err != nil {
		return Session{}, err
	}
	resp, err := client.Login(ctx, connect.NewRequest(&accountsv1.LoginRequest{Email: email, Password: password}))
	if err != nil {
		return Session{}, mapAuthError(err)
	}
	return sessionFrom(resp.Msg.GetAccount(), resp.Msg.GetTokens())
}

func (f Forwarder) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return Session{}, ErrInvalidCredentials
	}
	client, err := f.client(ctx)
	if err != nil {
		return Session{}, err
	}
	resp, err := client.Refresh(ctx, connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: refreshToken}))
	if err != nil {
		return Session{}, mapAuthError(err)
	}
	return sessionFrom(nil, resp.Msg.GetTokens())
}

func (f Forwarder) client(ctx context.Context) (accountsconnect.AccountsServiceClient, error) {
	if f.Resolver == nil {
		return nil, fmt.Errorf("%w: no resolver configured", ErrAuthUnavailable)
	}
	base, err := f.Resolver.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	httpClient := f.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return accountsconnect.NewAccountsServiceClient(httpClient, strings.TrimRight(base, "/")), nil
}

func sessionFrom(account *accountsv1.Account, tokens *accountsv1.TokenPair) (Session, error) {
	if tokens == nil || tokens.GetAccessToken() == "" {
		return Session{}, fmt.Errorf("%w: authenticator returned no token", ErrAuthUnavailable)
	}
	session := Session{AccessToken: tokens.GetAccessToken(), RefreshToken: tokens.GetRefreshToken()}
	if account != nil {
		session.Email = account.GetEmail()
		session.UserID = account.GetId()
	}
	return session, nil
}

func mapAuthError(err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		return ErrInvalidCredentials
	case connect.CodeInvalidArgument:
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	default:
		return fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
}
