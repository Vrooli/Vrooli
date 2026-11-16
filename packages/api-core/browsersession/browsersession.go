// Package browsersession is the same-origin browser sign-in edge to the
// installation identity provider, scenario-authenticator. A scenario's web UI
// cannot hold a bearer token safely, so it signs in through its own API, which
// forwards the credentials here and keeps the resulting session in HttpOnly
// cookies the page script never sees.
//
// The cookies carry the names api-core authn's scenario-authenticator provider
// already reads (AccessCookie, RefreshCookie), so a scenario that verifies its
// own audience through authn accepts the session with no further wiring. The
// edge owns no passwords, issues no tokens and grants nothing: the provider
// verifies the credentials and scopes the token to the scenario's audience,
// and the scenario still decides what the verified person may do.
//
// The edge is transport-neutral: it reads request headers and writes response
// headers, so a Connect handler and a plain HTTP handler use it alike.
package browsersession

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

const (
	// AccessCookie holds the short-lived access token. api-core authn's
	// scenario-authenticator provider reads it when no bearer is presented.
	AccessCookie = "vrooli_access_token"
	// RefreshCookie holds the rotating refresh token.
	RefreshCookie = "vrooli_refresh_token"
	// refreshLifetime matches scenario-authenticator's rotating refresh window.
	refreshLifetime = 7 * 24 * time.Hour
)

var (
	// ErrInvalidCredentials means the provider refused the credentials or the
	// refresh token.
	ErrInvalidCredentials = errors.New("the email, password or code was not accepted")
	// ErrInvalidInput means the request was incomplete.
	ErrInvalidInput = errors.New("enter your email and password")
	// ErrRateLimited means the provider is throttling sign-in attempts.
	ErrRateLimited = errors.New("too many sign-in attempts; try again later")
	// ErrUnavailable means the provider could not be reached or answered
	// without establishing a session.
	ErrUnavailable = errors.New("the installation sign-in provider is unavailable")
	// ErrNoSession means there is no refresh cookie to renew from.
	ErrNoSession = errors.New("no browser session to renew")
)

// Accounts is the part of scenario-authenticator's AccountsService the edge
// calls.
type Accounts interface {
	Login(context.Context, *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error)
	Refresh(context.Context, *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error)
	Logout(context.Context, *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error)
}

// Edge signs browsers in for one scenario audience.
type Edge struct {
	// Resource is the scenario's registered resource at the provider; tokens
	// are issued for the audience scenario-authenticator:<Resource>.
	Resource string
	// Accounts resolves the provider client per call, so a stopped provider is
	// an explicit ErrUnavailable and never a startup failure.
	Accounts func(context.Context) (Accounts, error)
	// Now is the clock; nil means time.Now.
	Now func() time.Time
}

// New returns an edge for resource that finds scenario-authenticator through
// api-core discovery on each call.
func New(resource string) *Edge {
	client := &http.Client{Timeout: 10 * time.Second}
	return &Edge{Resource: resource, Accounts: func(ctx context.Context) (Accounts, error) {
		base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
		if err != nil {
			return nil, err
		}
		return accountsconnect.NewAccountsServiceClient(client, base), nil
	}}
}

// Credentials is one sign-in attempt. The second-factor fields answer an MFA
// challenge a previous attempt returned.
type Credentials struct {
	Email        string
	Password     string
	TOTPCode     string
	RecoveryCode string
	MFAChallenge string
}

// Outcome is the result of a sign-in or renewal that did not fail.
type Outcome struct {
	// MFARequired means the password was accepted and the provider wants a
	// second factor; resubmit with MFAChallenge and a code. No cookie is set.
	MFARequired  bool
	MFAChallenge string
	// AccessToken is the new session's access token, for a caller that wants
	// to verify it in the same request. It is also in the cookie written to
	// the response; never send it to the browser in a body.
	AccessToken string
}

// SignIn forwards credentials to the provider and, on success, writes the
// session cookies to out. in is the browser request's headers.
func (e *Edge) SignIn(ctx context.Context, in, out http.Header, credentials Credentials) (Outcome, error) {
	email := strings.TrimSpace(credentials.Email)
	if email == "" || credentials.Password == "" {
		return Outcome{}, ErrInvalidInput
	}
	accounts, err := e.accounts(ctx)
	if err != nil {
		return Outcome{}, err
	}
	request := connect.NewRequest(&accountsv1.LoginRequest{
		Email: email, Password: credentials.Password, Resource: e.Resource,
		TotpCode: strings.TrimSpace(credentials.TOTPCode), RecoveryCode: strings.TrimSpace(credentials.RecoveryCode),
		MfaChallenge: strings.TrimSpace(credentials.MFAChallenge),
	})
	forwardMetadata(request.Header(), in)
	response, err := accounts.Login(ctx, request)
	if err != nil {
		return Outcome{}, providerError(err)
	}
	if response.Msg.GetMfaRequired() {
		return Outcome{MFARequired: true, MFAChallenge: response.Msg.GetMfaChallenge()}, nil
	}
	return e.establish(in, out, response.Msg.GetTokens())
}

// Renew exchanges the refresh cookie for a new session and writes the rotated
// cookies to out. A refused refresh clears the cookies so the browser stops
// presenting them; an unreachable provider leaves them for the next attempt.
func (e *Edge) Renew(ctx context.Context, in, out http.Header) (Outcome, error) {
	refresh := cookie(in, RefreshCookie)
	if refresh == "" {
		return Outcome{}, ErrNoSession
	}
	accounts, err := e.accounts(ctx)
	if err != nil {
		return Outcome{}, err
	}
	request := connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: refresh, Resource: e.Resource})
	forwardMetadata(request.Header(), in)
	response, err := accounts.Refresh(ctx, request)
	if err != nil {
		err = providerError(err)
		if errors.Is(err, ErrInvalidCredentials) {
			Clear(in, out)
		}
		return Outcome{}, err
	}
	return e.establish(in, out, response.Msg.GetTokens())
}

// SignOut clears the session cookies and asks the provider to revoke the
// access token. The cookies are cleared even when revocation fails, which is
// reported so the page can say the provider did not confirm it.
func (e *Edge) SignOut(ctx context.Context, in, out http.Header) error {
	Clear(in, out)
	access := cookie(in, AccessCookie)
	if access == "" {
		return nil
	}
	accounts, err := e.accounts(ctx)
	if err != nil {
		return err
	}
	request := connect.NewRequest(&accountsv1.LogoutRequest{AccessToken: access})
	forwardMetadata(request.Header(), in)
	if _, err := accounts.Logout(ctx, request); err != nil {
		return providerError(err)
	}
	return nil
}

// HasSession reports whether the browser presents a session cookie at all.
func HasSession(in http.Header) bool {
	return cookie(in, AccessCookie) != "" || cookie(in, RefreshCookie) != ""
}

// Clear writes expired session cookies to out.
func Clear(in, out http.Header) {
	secure := Secure(in)
	for _, name := range []string{AccessCookie, RefreshCookie} {
		out.Add("Set-Cookie", (&http.Cookie{Name: name, Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
	}
}

// Secure reports whether the browser reached the scenario over HTTPS, so the
// cookies are marked Secure behind the tunnel and stay usable on plain-HTTP
// localhost. The UI server and the tunnel forward X-Forwarded-Proto; a browser
// sends Origin on every sign-in POST.
func Secure(in http.Header) bool {
	if strings.EqualFold(strings.TrimSpace(in.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	origin, err := url.Parse(strings.TrimSpace(in.Get("Origin")))
	return err == nil && origin.Scheme == "https"
}

// ConnectError maps an edge error to the Connect code a page branches on.
func ConnectError(err error) *connect.Error {
	switch {
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrNoSession):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrRateLimited):
		return connect.NewError(connect.CodeResourceExhausted, err)
	default:
		return connect.NewError(connect.CodeUnavailable, ErrUnavailable)
	}
}

func (e *Edge) accounts(ctx context.Context) (Accounts, error) {
	if e == nil || e.Accounts == nil {
		return nil, ErrUnavailable
	}
	accounts, err := e.Accounts(ctx)
	if err != nil || accounts == nil {
		return nil, ErrUnavailable
	}
	return accounts, nil
}

func (e *Edge) establish(in, out http.Header, tokens *accountsv1.TokenPair) (Outcome, error) {
	access := strings.TrimSpace(tokens.GetAccessToken())
	if access == "" {
		return Outcome{}, ErrUnavailable
	}
	now := time.Now
	if e.Now != nil {
		now = e.Now
	}
	lifetime := time.Hour
	if expires := tokens.GetAccessTokenExpiresAt(); expires != nil {
		lifetime = expires.AsTime().Sub(now())
	}
	if lifetime < time.Second {
		lifetime = time.Second
	}
	secure := Secure(in)
	out.Add("Set-Cookie", (&http.Cookie{Name: AccessCookie, Value: access, Path: "/", MaxAge: int(lifetime.Seconds()), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
	if refresh := strings.TrimSpace(tokens.GetRefreshToken()); refresh != "" {
		out.Add("Set-Cookie", (&http.Cookie{Name: RefreshCookie, Value: refresh, Path: "/", MaxAge: int(refreshLifetime.Seconds()), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
	}
	return Outcome{AccessToken: access}, nil
}

// providerError classifies a provider failure without passing its message to
// the browser.
func providerError(err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		return ErrInvalidCredentials
	case connect.CodeInvalidArgument:
		return ErrInvalidInput
	case connect.CodeResourceExhausted:
		return ErrRateLimited
	}
	return ErrUnavailable
}

// forwardMetadata passes the browser's user agent and transport security to
// the provider, which records them on the session.
func forwardMetadata(to, from http.Header) {
	if agent := from.Get("User-Agent"); agent != "" {
		to.Set("User-Agent", agent)
	}
	if Secure(from) {
		to.Set("X-Forwarded-Proto", "https")
	}
}

func cookie(in http.Header, name string) string {
	value, err := (&http.Request{Header: in}).Cookie(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value.Value)
}
