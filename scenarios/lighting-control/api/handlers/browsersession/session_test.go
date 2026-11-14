package browsersession

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
)

type fakeAccounts struct {
	login   func(*connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error)
	refresh func(*connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error)
	logout  func(*connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error)
}

func (f fakeAccounts) Login(_ context.Context, r *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	return f.login(r)
}
func (f fakeAccounts) Refresh(_ context.Context, r *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
	return f.refresh(r)
}
func (f fakeAccounts) Logout(_ context.Context, r *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error) {
	return f.logout(r)
}
func sessionRequest(path, body string) *http.Request {
	r := httptest.NewRequest("POST", "http://desk.test"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Vrooli-Session", "1")
	r.Header.Set("Origin", "http://desk.test")
	return r
}
func runRequest(f fakeAccounts, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h := handler{client: func(context.Context) (accountsClient, error) { return f, nil }}
	h.ServeHTTP(w, r)
	return w
}
func authCookies(h http.Header) {
	h.Add("Set-Cookie", (&http.Cookie{Name: accessCookie, Value: "secret-access", Path: "/", MaxAge: 300, HttpOnly: true, SameSite: http.SameSiteLaxMode}).String())
	h.Add("Set-Cookie", (&http.Cookie{Name: refreshCookie, Value: "secret-refresh", Path: "/", MaxAge: 600, HttpOnly: true, SameSite: http.SameSiteLaxMode}).String())
}
func TestLoginUsesResourceAndRedactsTokens(t *testing.T) {
	f := fakeAccounts{login: func(r *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
		if r.Msg.GetResource() != "lighting-control" || r.Msg.GetEmail() != "a@b.test" {
			t.Fatal("incorrect provider request")
		}
		res := connect.NewResponse(&accountsv1.LoginResponse{Tokens: &accountsv1.TokenPair{AccessToken: "secret-access", RefreshToken: "secret-refresh"}})
		authCookies(res.Header())
		return res, nil
	}}
	r := sessionRequest("/api/session/login", `{"email":"a@b.test","password":"secret-password"}`)
	w := runRequest(f, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), "secret-") || strings.Contains(w.Body.String(), `"authenticated":true`) {
		t.Fatalf("unsafe response %d %s", w.Code, w.Body)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatal("missing canonical cookies")
	}
	for _, c := range cookies {
		if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Domain != "" || c.Path != "/" {
			t.Fatalf("unsafe cookie %+v", c)
		}
	}
}
func TestMFAChallengeDoesNotEstablishSession(t *testing.T) {
	f := fakeAccounts{login: func(r *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
		res := connect.NewResponse(&accountsv1.LoginResponse{MfaRequired: true, MfaChallenge: "challenge"})
		authCookies(res.Header())
		return res, nil
	}}
	w := runRequest(f, sessionRequest("/api/session/login", `{"email":"a@b.test","password":"password"}`))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"mfaRequired":true`) || len(w.Result().Cookies()) != 0 {
		t.Fatalf("MFA not preserved %d %s", w.Code, w.Body)
	}
}
func TestCrossSiteAndSpoofedForwardingRejected(t *testing.T) {
	for _, tc := range []struct{ name, origin, site, forwarded, peer string }{
		{"cross origin", "http://evil.test", "", "", "192.0.2.1:44"},
		{"null origin", "null", "", "", "192.0.2.1:44"},
		{"cross site metadata", "http://desk.test", "cross-site", "", "192.0.2.1:44"},
		{"untrusted forwarded", "http://evil.test", "", "evil.test", "192.0.2.1:44"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := sessionRequest("/api/session/logout", "")
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Sec-Fetch-Site", tc.site)
			r.Header.Set("X-Forwarded-Host", tc.forwarded)
			r.RemoteAddr = tc.peer
			w := runRequest(fakeAccounts{}, r)
			if w.Code != 403 {
				t.Fatalf("got %d", w.Code)
			}
		})
	}
	r := sessionRequest("/api/session/logout", "")
	r.Header.Del("X-Vrooli-Session")
	if w := runRequest(fakeAccounts{}, r); w.Code != 403 {
		t.Fatal("missing header admitted")
	}
	r = sessionRequest("/api/session/logout", "")
	r.Header.Set("Origin", "http://ui.test:4000")
	r.Header.Set("X-Forwarded-Host", "ui.test:4000")
	r.RemoteAddr = "127.0.0.1:1234"
	if !sameOrigin(r) {
		t.Fatal("same-origin UI proxy refused")
	}
}
func TestStatusOnlyUsesVerifiedUnexpiredPrincipal(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    identity.Principal
		ok   bool
	}{
		{"verified", identity.Principal{Subject: "operator", Verified: true, ExpiresAt: time.Now().Add(time.Hour)}, true},
		{"unverified", identity.Principal{Subject: "operator", ExpiresAt: time.Now().Add(time.Hour)}, false},
		{"expired", identity.Principal{Subject: "operator", Verified: true, ExpiresAt: time.Now().Add(-time.Hour)}, false},
		{"no expiry", identity.Principal{Subject: "operator", Verified: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://desk.test/api/session", nil)
			r = r.WithContext(identity.WithPrincipal(r.Context(), tc.p))
			w := runRequest(fakeAccounts{}, r)
			if strings.Contains(w.Body.String(), `"authenticated":true`) != tc.ok {
				t.Fatal(w.Body.String())
			}
		})
	}
}
func TestRefreshAndLogoutUseCookiesOnly(t *testing.T) {
	f := fakeAccounts{
		refresh: func(r *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
			if r.Msg.GetRefreshToken() != "cookie-refresh" {
				t.Fatal("wrong refresh")
			}
			res := connect.NewResponse(&accountsv1.RefreshResponse{})
			authCookies(res.Header())
			return res, nil
		},
		logout: func(r *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error) {
			if r.Msg.GetAccessToken() != "cookie-access" {
				t.Fatal("wrong access")
			}
			return connect.NewResponse(&accountsv1.LogoutResponse{}), nil
		},
	}
	r := sessionRequest("/api/session/refresh", `{"refreshToken":"attacker"}`)
	r.AddCookie(&http.Cookie{Name: refreshCookie, Value: "cookie-refresh"})
	w := runRequest(f, r)
	if w.Code != 200 || len(w.Result().Cookies()) != 2 {
		t.Fatalf("refresh %d", w.Code)
	}
	r = sessionRequest("/api/session/logout", "")
	r.AddCookie(&http.Cookie{Name: accessCookie, Value: "cookie-access"})
	w = runRequest(f, r)
	if w.Code != 200 {
		t.Fatalf("logout %d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.MaxAge != -1 {
			t.Fatal("logout did not expire cookie")
		}
	}
}
func TestUnavailableProviderAndErrorsDoNotLeak(t *testing.T) {
	h := handler{client: func(context.Context) (accountsClient, error) { return nil, errors.New("secret") }}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, sessionRequest("/api/session/logout", ""))
	if w.Code != 503 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal(w.Body.String())
	}
	f := fakeAccounts{login: func(*connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("password secret"))
	}}
	w = runRequest(f, sessionRequest("/api/session/login", `{"email":"a@b.test","password":"password"}`))
	if w.Code != 401 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal(w.Body.String())
	}
}

func TestHTTPSProxyCookiesAreSecure(t *testing.T) {
	f := fakeAccounts{
		login: func(r *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
			if r.Header().Get("X-Forwarded-Proto") != "https" {
				t.Fatal("lost verified HTTPS context")
			}
			res := connect.NewResponse(&accountsv1.LoginResponse{Tokens: &accountsv1.TokenPair{AccessToken: "secret-access"}})
			authCookies(res.Header())
			return res, nil
		},
		refresh: func(r *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
			if r.Header().Get("X-Forwarded-Proto") != "https" {
				t.Fatal("lost HTTPS on refresh")
			}
			res := connect.NewResponse(&accountsv1.RefreshResponse{})
			authCookies(res.Header())
			return res, nil
		},
	}
	for _, action := range []string{"login", "refresh", "logout"} {
		r := sessionRequest("/api/session/"+action, `{"email":"owner@test.example","password":"password"}`)
		r.Header.Set("Origin", "https://desk.example")
		r.Header.Set("X-Forwarded-Host", "desk.example")
		r.RemoteAddr = "127.0.0.1:1234"
		r.AddCookie(&http.Cookie{Name: refreshCookie, Value: "refresh"})
		w := runRequest(f, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", action, w.Code, w.Body)
		}
		if len(w.Result().Cookies()) != 2 {
			t.Fatal("missing cookies")
		}
		for _, c := range w.Result().Cookies() {
			if !c.Secure {
				t.Fatalf("%s insecure cookie", action)
			}
		}
	}
	r := sessionRequest("/api/session/logout", "")
	r.Header.Set("X-Forwarded-Proto", "https")
	if secureRequest(r) {
		t.Fatal("untrusted scheme header changed cookie security")
	}
}
func TestLogoutClearsLocalCookiesDuringProviderOutage(t *testing.T) {
	h := handler{client: func(context.Context) (accountsClient, error) { return nil, errors.New("offline") }}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, sessionRequest("/api/session/logout", ""))
	if w.Code != 503 || !strings.Contains(w.Body.String(), "Signed out locally") || len(w.Result().Cookies()) != 2 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	for _, c := range w.Result().Cookies() {
		if c.MaxAge != -1 {
			t.Fatal("cookie retained")
		}
	}
}
