package browsersession

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// provider is a scenario-authenticator stand-in that records what it was
// asked and answers as configured.
type provider struct {
	login     *accountsv1.LoginRequest
	loginErr  error
	loginResp *accountsv1.LoginResponse
	refresh   *accountsv1.RefreshRequest
	refreshFn func() (*accountsv1.RefreshResponse, error)
	loggedOut string
	logoutErr error
	agent     string
}

func (p *provider) Login(_ context.Context, r *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	p.login, p.agent = r.Msg, r.Header().Get("User-Agent")
	if p.loginErr != nil {
		return nil, p.loginErr
	}
	return connect.NewResponse(p.loginResp), nil
}

func (p *provider) Refresh(_ context.Context, r *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
	p.refresh = r.Msg
	response, err := p.refreshFn()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (p *provider) Logout(_ context.Context, r *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error) {
	p.loggedOut = r.Msg.GetAccessToken()
	if p.logoutErr != nil {
		return nil, p.logoutErr
	}
	return connect.NewResponse(&accountsv1.LogoutResponse{}), nil
}

var fixedNow = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

func edge(p Accounts) *Edge {
	return &Edge{Resource: "money-ledger", Accounts: func(context.Context) (Accounts, error) { return p, nil }, Now: func() time.Time { return fixedNow }}
}

func tokens(access, refresh string) *accountsv1.TokenPair {
	return &accountsv1.TokenPair{AccessToken: access, RefreshToken: refresh, AccessTokenExpiresAt: timestamppb.New(fixedNow.Add(30 * time.Minute))}
}

func cookies(out http.Header) map[string]*http.Cookie {
	found := map[string]*http.Cookie{}
	for _, c := range (&http.Response{Header: out}).Cookies() {
		found[c.Name] = c
	}
	return found
}

func browser(cookieHeader string) http.Header {
	in := http.Header{"User-Agent": {"fixture-browser"}}
	if cookieHeader != "" {
		in.Set("Cookie", cookieHeader)
	}
	return in
}

func TestSignInScopesTheSessionToTheResourceAndKeepsItOutOfScript(t *testing.T) {
	p := &provider{loginResp: &accountsv1.LoginResponse{Tokens: tokens("access-1", "refresh-1")}}
	out := http.Header{}
	outcome, err := edge(p).SignIn(context.Background(), browser(""), out, Credentials{Email: " owner@example.test ", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	if p.login.GetResource() != "money-ledger" || p.login.GetEmail() != "owner@example.test" || p.agent != "fixture-browser" {
		t.Fatalf("provider asked %+v with agent %q", p.login, p.agent)
	}
	if outcome.AccessToken != "access-1" || outcome.MFARequired {
		t.Fatalf("outcome %+v", outcome)
	}
	set := cookies(out)
	access, refresh := set[AccessCookie], set[RefreshCookie]
	if access == nil || access.Value != "access-1" || !access.HttpOnly || access.SameSite != http.SameSiteLaxMode || access.Path != "/" || access.MaxAge != 1800 {
		t.Fatalf("access cookie %+v", access)
	}
	if refresh == nil || refresh.Value != "refresh-1" || !refresh.HttpOnly || refresh.MaxAge != int(refreshLifetime.Seconds()) {
		t.Fatalf("refresh cookie %+v", refresh)
	}
	if access.Secure || refresh.Secure {
		t.Fatal("plain-HTTP localhost cookies must not be Secure, or the browser drops them")
	}
}

func TestCookiesAreSecureBehindHTTPS(t *testing.T) {
	for name, in := range map[string]http.Header{
		"forwarded proto": {"X-Forwarded-Proto": {"https"}},
		"https origin":    {"Origin": {"https://money-ledger.example.test"}},
	} {
		t.Run(name, func(t *testing.T) {
			p := &provider{loginResp: &accountsv1.LoginResponse{Tokens: tokens("a", "r")}}
			out := http.Header{}
			if _, err := edge(p).SignIn(context.Background(), in, out, Credentials{Email: "e", Password: "p"}); err != nil {
				t.Fatal(err)
			}
			for _, c := range cookies(out) {
				if !c.Secure {
					t.Fatalf("%s cookie is not Secure", c.Name)
				}
			}
		})
	}
}

func TestSignInReturnsTheMFAChallengeWithoutASession(t *testing.T) {
	p := &provider{loginResp: &accountsv1.LoginResponse{MfaRequired: true, MfaChallenge: "challenge-1"}}
	out := http.Header{}
	outcome, err := edge(p).SignIn(context.Background(), browser(""), out, Credentials{Email: "e", Password: "p"})
	if err != nil || !outcome.MFARequired || outcome.MFAChallenge != "challenge-1" {
		t.Fatalf("outcome %+v err %v", outcome, err)
	}
	if len(out.Values("Set-Cookie")) != 0 {
		t.Fatal("no session before the second factor")
	}
	p.loginResp = &accountsv1.LoginResponse{Tokens: tokens("a", "r")}
	if _, err := edge(p).SignIn(context.Background(), browser(""), out, Credentials{Email: "e", Password: "p", TOTPCode: " 123456 ", MFAChallenge: "challenge-1"}); err != nil {
		t.Fatal(err)
	}
	if p.login.GetTotpCode() != "123456" || p.login.GetMfaChallenge() != "challenge-1" {
		t.Fatalf("second factor not forwarded: %+v", p.login)
	}
}

func TestSignInFailuresAreClassifiedWithoutLeakingProviderDetail(t *testing.T) {
	for _, test := range []struct {
		name string
		in   Credentials
		err  error
		want error
		code connect.Code
	}{
		{"missing password", Credentials{Email: "e"}, nil, ErrInvalidInput, connect.CodeInvalidArgument},
		{"refused", Credentials{Email: "e", Password: "p"}, connect.NewError(connect.CodeUnauthenticated, errors.New("account 42 locked: internal detail")), ErrInvalidCredentials, connect.CodeUnauthenticated},
		{"throttled", Credentials{Email: "e", Password: "p"}, connect.NewError(connect.CodeResourceExhausted, errors.New("x")), ErrRateLimited, connect.CodeResourceExhausted},
		{"provider down", Credentials{Email: "e", Password: "p"}, connect.NewError(connect.CodeUnavailable, errors.New("dial tcp")), ErrUnavailable, connect.CodeUnavailable},
		{"no token", Credentials{Email: "e", Password: "p"}, nil, ErrUnavailable, connect.CodeUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &provider{loginErr: test.err, loginResp: &accountsv1.LoginResponse{}}
			out := http.Header{}
			_, err := edge(p).SignIn(context.Background(), browser(""), out, test.in)
			if !errors.Is(err, test.want) {
				t.Fatalf("err = %v, want %v", err, test.want)
			}
			mapped := ConnectError(err)
			if mapped.Code() != test.code || mapped.Message() == "" || errors.Is(mapped, errors.New("internal detail")) {
				t.Fatalf("mapped %v", mapped)
			}
			if len(out.Values("Set-Cookie")) != 0 {
				t.Fatal("a failed sign-in sets no cookie")
			}
		})
	}
}

func TestUnreachableProviderIsUnavailable(t *testing.T) {
	down := &Edge{Resource: "money-ledger", Accounts: func(context.Context) (Accounts, error) { return nil, errors.New("not running") }}
	if _, err := down.SignIn(context.Background(), browser(""), http.Header{}, Credentials{Email: "e", Password: "p"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestRenewRotatesTheSessionFromTheRefreshCookie(t *testing.T) {
	p := &provider{refreshFn: func() (*accountsv1.RefreshResponse, error) {
		return &accountsv1.RefreshResponse{Tokens: tokens("access-2", "refresh-2")}, nil
	}}
	out := http.Header{}
	outcome, err := edge(p).Renew(context.Background(), browser(RefreshCookie+"=refresh-1"), out)
	if err != nil || outcome.AccessToken != "access-2" {
		t.Fatalf("outcome %+v err %v", outcome, err)
	}
	if p.refresh.GetRefreshToken() != "refresh-1" || p.refresh.GetResource() != "money-ledger" {
		t.Fatalf("refresh request %+v", p.refresh)
	}
	if set := cookies(out); set[AccessCookie].Value != "access-2" || set[RefreshCookie].Value != "refresh-2" {
		t.Fatalf("cookies %+v", set)
	}
}

func TestRenewWithoutARefreshCookieAsksNothing(t *testing.T) {
	p := &provider{refreshFn: func() (*accountsv1.RefreshResponse, error) { t.Fatal("no refresh without a cookie"); return nil, nil }}
	if _, err := edge(p).Renew(context.Background(), browser(""), http.Header{}); !errors.Is(err, ErrNoSession) {
		t.Fatalf("err = %v", err)
	}
}

// A refused refresh clears the cookies so the browser stops presenting a dead
// session; an outage keeps them for the next attempt.
func TestRenewClearsOnlyARefusedSession(t *testing.T) {
	for _, test := range []struct {
		name    string
		err     error
		cleared bool
	}{
		{"refused", connect.NewError(connect.CodeUnauthenticated, errors.New("revoked")), true},
		{"outage", connect.NewError(connect.CodeUnavailable, errors.New("down")), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &provider{refreshFn: func() (*accountsv1.RefreshResponse, error) { return nil, test.err }}
			out := http.Header{}
			_, err := edge(p).Renew(context.Background(), browser(RefreshCookie+"=r"), out)
			if err == nil {
				t.Fatal("expected an error")
			}
			if cleared := len(out.Values("Set-Cookie")) > 0; cleared != test.cleared {
				t.Fatalf("cleared = %t, want %t", cleared, test.cleared)
			}
		})
	}
}

func TestSignOutAlwaysClearsAndReportsUnconfirmedRevocation(t *testing.T) {
	p := &provider{logoutErr: connect.NewError(connect.CodeUnavailable, errors.New("down"))}
	out := http.Header{}
	err := edge(p).SignOut(context.Background(), browser(AccessCookie+"=access-1"), out)
	if !errors.Is(err, ErrUnavailable) || p.loggedOut != "access-1" {
		t.Fatalf("err %v revoked %q", err, p.loggedOut)
	}
	set := cookies(out)
	if set[AccessCookie] == nil || set[AccessCookie].MaxAge >= 0 || set[RefreshCookie] == nil || set[RefreshCookie].MaxAge >= 0 {
		t.Fatalf("cookies not cleared: %+v", set)
	}
	if err := edge(&provider{}).SignOut(context.Background(), browser(""), http.Header{}); err != nil {
		t.Fatalf("signing out with no session: %v", err)
	}
}

func TestHasSession(t *testing.T) {
	if HasSession(browser("")) || !HasSession(browser(RefreshCookie+"=r")) || !HasSession(browser(AccessCookie+"=a")) {
		t.Fatal("HasSession misreads the cookies")
	}
}
