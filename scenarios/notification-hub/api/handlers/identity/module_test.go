package identity

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/identity"

	internalidentity "notification-hub/internal/identity"
)

type fakeAuth struct{ refreshed int }

func (f *fakeAuth) Login(_ context.Context, email, password string) (internalidentity.Session, error) {
	if password != "right" {
		return internalidentity.Session{}, internalidentity.ErrInvalidCredentials
	}
	return internalidentity.Session{AccessToken: "access-1", RefreshToken: "refresh-1", Email: email, UserID: "owner-1"}, nil
}

func (f *fakeAuth) Refresh(_ context.Context, token string) (internalidentity.Session, error) {
	if token != "refresh-1" {
		return internalidentity.Session{}, internalidentity.ErrInvalidCredentials
	}
	f.refreshed++
	return internalidentity.Session{AccessToken: "access-2", RefreshToken: "refresh-2"}, nil
}

type fakeVerifier struct{}

func (fakeVerifier) Validate(_ context.Context, token string) (owneridentity.Identity, error) {
	if token == "access-2" || token == "access-1" {
		return owneridentity.Identity{Subject: "owner-1", Email: "owner@example.test"}, nil
	}
	return owneridentity.Identity{}, errors.New("expired")
}

func TestLogin_BrowserSessionGetsHttpOnlyCookiesAndNoTokenInTheBody(t *testing.T) {
	h := &handler{auth: &fakeAuth{}, verifier: fakeVerifier{}}
	req := connect.NewRequest(&v1.LoginRequest{Email: "owner@example.test", Password: "right"})
	req.Header().Set(browserSessionHeader, "1")
	req.Header().Set("X-Forwarded-Proto", "https")
	resp, err := h.Login(context.Background(), req)
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetToken())
	require.Empty(t, resp.Msg.GetRefreshToken())
	cookies := strings.Join(resp.Header().Values("Set-Cookie"), "\n")
	require.Contains(t, cookies, internalidentity.OwnerCookieName+"=access-1")
	require.Contains(t, cookies, refreshCookieName+"=refresh-1")
	require.Contains(t, cookies, "HttpOnly")
	require.Contains(t, cookies, "Secure")

	_, err = h.Login(context.Background(), connect.NewRequest(&v1.LoginRequest{Email: "owner@example.test", Password: "wrong"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestGetSession_RenewsAnExpiredSessionFromTheRefreshCookie(t *testing.T) {
	auth := &fakeAuth{}
	h := &handler{auth: auth, verifier: fakeVerifier{}}
	req := connect.NewRequest(&v1.GetSessionRequest{})
	req.Header().Add("Cookie", (&http.Cookie{Name: internalidentity.OwnerCookieName, Value: "stale"}).String())
	req.Header().Add("Cookie", (&http.Cookie{Name: refreshCookieName, Value: "refresh-1"}).String())
	resp, err := h.GetSession(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.GetSignedIn())
	require.Equal(t, "owner-1", resp.Msg.GetSubject())
	require.Equal(t, 1, auth.refreshed)
	require.Contains(t, strings.Join(resp.Header().Values("Set-Cookie"), "\n"), internalidentity.OwnerCookieName+"=access-2")

	signedOut, err := h.GetSession(context.Background(), connect.NewRequest(&v1.GetSessionRequest{}))
	require.NoError(t, err)
	require.False(t, signedOut.Msg.GetSignedIn())
}

type stubProvider struct {
	principal apiidentity.Principal
	err       error
}

func (stubProvider) Source() apiidentity.AuthSource { return apiidentity.SourceCloudflareAccess }

func (p stubProvider) VerifyRequest(context.Context, *http.Request) (apiidentity.Principal, error) {
	return p.principal, p.err
}

// getSessionThrough calls GetSession behind the owner middleware, as mounted.
func getSessionThrough(t *testing.T, h *handler, provider authn.Provider, cookies ...*http.Cookie) *v1.GetSessionResponse {
	t.Helper()
	auth := internalidentity.OwnerAuthenticator{Providers: []authn.Provider{provider}, OwnerSubject: func(context.Context) string { return "operator" }}
	var resp *connect.Response[v1.GetSessionResponse]
	auth.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		req := connect.NewRequest(&v1.GetSessionRequest{})
		for _, cookie := range cookies {
			req.Header().Add("Cookie", cookie.String())
		}
		var err error
		resp, err = h.GetSession(r.Context(), req)
		require.NoError(t, err)
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	return resp.Msg
}

func TestGetSession_ReportsTheCloudflareAccessOwnerWithoutAPassword(t *testing.T) {
	h := &handler{auth: &fakeAuth{}, verifier: fakeVerifier{}}
	access := stubProvider{principal: apiidentity.Principal{Kind: apiidentity.ActorHuman, Subject: "cf-user-1", Email: "operator@example.test", Verified: true, Source: apiidentity.SourceCloudflareAccess}}
	session := getSessionThrough(t, h, access)
	require.True(t, session.GetSignedIn())
	require.Equal(t, "operator", session.GetSubject())
	require.Equal(t, "operator@example.test", session.GetEmail())
}

func TestGetSession_ARefusedAccessAssertionIsNotRenewedFromTheRefreshCookie(t *testing.T) {
	auth := &fakeAuth{}
	h := &handler{auth: auth, verifier: fakeVerifier{}}
	expired := stubProvider{err: apiidentity.NewFailure(apiidentity.FailureExpired, apiidentity.SourceCloudflareAccess)}
	session := getSessionThrough(t, h, expired, &http.Cookie{Name: refreshCookieName, Value: "refresh-1"})
	require.False(t, session.GetSignedIn())
	require.Zero(t, auth.refreshed)
}
