// Package identity is the same-origin owner sign-in facade for the installed
// web app. It forwards credentials to scenario-authenticator and keeps the
// session in HttpOnly cookies; browser calls never receive a token in a body.
package identity

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
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/identity"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/identity/identity_v1connect"

	internalidentity "notification-hub/internal/identity"
	"notification-hub/internal/module"
)

const (
	refreshCookieName    = "notification_hub_refresh_token"
	browserSessionHeader = "X-Vrooli-Browser-Session"
	// refreshCookieMaxAge matches scenario-authenticator's 7-day rotating
	// refresh window; each refresh renews it.
	refreshCookieMaxAge = 7 * 24 * time.Hour
	accessCookieMaxAge  = time.Hour
)

// Authenticator is the forwarding seam (internal/identity.Forwarder in
// production; a fake in tests).
type Authenticator interface {
	Login(ctx context.Context, email, password string) (internalidentity.Session, error)
	Refresh(ctx context.Context, refreshToken string) (internalidentity.Session, error)
}

type handler struct {
	auth     Authenticator
	verifier internalidentity.Verifier
}

func Module(auth Authenticator, verifier internalidentity.Verifier) module.Module {
	h := &handler{auth: auth, verifier: verifier}
	return module.Module{Name: "identity", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewIdentityServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	session, err := h.auth.Login(ctx, req.Msg.GetEmail(), req.Msg.GetPassword())
	if err != nil {
		return nil, authError(err)
	}
	resp := connect.NewResponse(&v1.LoginResponse{Email: session.Email, UserId: session.UserID})
	if req.Header().Get(browserSessionHeader) != "1" {
		resp.Msg.Token = session.AccessToken
		resp.Msg.RefreshToken = session.RefreshToken
	}
	setSessionCookies(resp.Header(), req.Header(), session)
	return resp, nil
}

// GetSession reports the verified owner: the Cloudflare Access login when the
// request carries one, else the cookie session. When the access cookie has expired
// but the refresh cookie is still valid, it rotates both, so a tap on a push
// days later still lands signed in.
func (h *handler) GetSession(ctx context.Context, req *connect.Request[v1.GetSessionRequest]) (*connect.Response[v1.GetSessionResponse], error) {
	owner, err := internalidentity.Owner(ctx, req.Header(), h.verifier)
	if err == nil {
		return connect.NewResponse(&v1.GetSessionResponse{SignedIn: true, Subject: owner.Subject, Email: owner.Email}), nil
	}
	refresh := cookieValue(req.Header(), refreshCookieName)
	if refresh == "" || h.verifier == nil || internalidentity.Refused(ctx) {
		return connect.NewResponse(&v1.GetSessionResponse{}), nil
	}
	session, err := h.auth.Refresh(ctx, refresh)
	if err != nil {
		resp := connect.NewResponse(&v1.GetSessionResponse{})
		if errors.Is(err, internalidentity.ErrInvalidCredentials) {
			clearSessionCookies(resp.Header(), req.Header())
		}
		return resp, nil
	}
	owner, err = h.verifier.Validate(ctx, session.AccessToken)
	if err != nil {
		return connect.NewResponse(&v1.GetSessionResponse{}), nil
	}
	resp := connect.NewResponse(&v1.GetSessionResponse{SignedIn: true, Subject: owner.Subject, Email: owner.Email})
	setSessionCookies(resp.Header(), req.Header(), session)
	return resp, nil
}

func (h *handler) Logout(_ context.Context, req *connect.Request[v1.LogoutRequest]) (*connect.Response[v1.LogoutResponse], error) {
	resp := connect.NewResponse(&v1.LogoutResponse{})
	clearSessionCookies(resp.Header(), req.Header())
	return resp, nil
}

func setSessionCookies(out, in http.Header, session internalidentity.Session) {
	secure := cookieSecure(in)
	out.Add("Set-Cookie", (&http.Cookie{Name: internalidentity.OwnerCookieName, Value: session.AccessToken, Path: "/", MaxAge: int(accessCookieMaxAge.Seconds()), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
	if session.RefreshToken != "" {
		out.Add("Set-Cookie", (&http.Cookie{Name: refreshCookieName, Value: session.RefreshToken, Path: "/", MaxAge: int(refreshCookieMaxAge.Seconds()), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
	}
}

func clearSessionCookies(out, in http.Header) {
	secure := cookieSecure(in)
	for _, name := range []string{internalidentity.OwnerCookieName, refreshCookieName} {
		out.Add("Set-Cookie", (&http.Cookie{Name: name, Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
	}
}

func cookieValue(headers http.Header, name string) string {
	cookie, err := (&http.Request{Header: headers}).Cookie(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

// cookieSecure marks cookies Secure behind HTTPS (the public origin sets
// X-Forwarded-Proto) and leaves them usable on plain-HTTP localhost.
func cookieSecure(headers http.Header) bool {
	if configured := strings.TrimSpace(os.Getenv("VROOLI_AUTH_COOKIE_SECURE")); configured != "" {
		secure, _ := strconv.ParseBool(configured)
		return secure
	}
	return strings.EqualFold(strings.TrimSpace(headers.Get("X-Forwarded-Proto")), "https")
}

func authError(err error) error {
	switch {
	case errors.Is(err, internalidentity.ErrInvalidCredentials):
		return connect.NewError(connect.CodeUnauthenticated, internalidentity.ErrInvalidCredentials)
	case errors.Is(err, internalidentity.ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeUnavailable, internalidentity.ErrAuthUnavailable)
	}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "identity_login", Path: connectv1.IdentityServiceLoginProcedure, Method: http.MethodPost, Summary: "Sign in through scenario-authenticator and set the owner session cookie", Category: "identity"},
	{ID: "identity_get_session", Path: connectv1.IdentityServiceGetSessionProcedure, Method: http.MethodPost, Summary: "Report the signed-in owner, renewing an expired session", Category: "identity"},
	{ID: "identity_logout", Path: connectv1.IdentityServiceLogoutProcedure, Method: http.MethodPost, Summary: "Clear the owner session cookies", Category: "identity"},
}
