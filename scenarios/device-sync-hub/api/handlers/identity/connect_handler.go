// Package identity is the HTTP/Connect transport edge for the same-origin owner
// sign-in / registration facade. It is intentionally thin: decode the request,
// call internal/identity.Forwarder (which forwards to scenario-authenticator via
// api-core/discovery), and translate the result or typed error to Connect. No
// credential logic lives here or in the forwarder — only relay.
package identity

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	internalauth "device-sync-hub/internal/auth"
	internalidentity "device-sync-hub/internal/identity"

	"connectrpc.com/connect"

	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity"
)

// Forwarder is the seam the handler depends on (internal/identity.Forwarder in
// production; a fake in tests).
type Forwarder interface {
	Login(ctx context.Context, c internalidentity.Credentials) (internalidentity.Owner, error)
	Register(ctx context.Context, r internalidentity.Registration) (internalidentity.Owner, error)
}

// Deps wires the seams the Connect identity handler needs.
type Deps struct {
	Forwarder Forwarder
	Logger    *log.Logger
}

type connectHandler struct {
	deps Deps
}

const browserSessionHeader = "X-Vrooli-Browser-Session"

// NewConnectHandler constructs the Connect handler for the identity service.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Login(ctx context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
	owner, err := h.deps.Forwarder.Login(ctx, internalidentity.Credentials{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
	})
	if err != nil {
		// Invalid credentials is an expected, non-noteworthy outcome; only log
		// the unexpected (unavailable/internal) cases to avoid log spam from
		// failed sign-in attempts.
		if !errors.Is(err, internalidentity.ErrInvalidCredentials) {
			h.deps.Logger.Printf("identity.Login: %v", err)
		}
		return nil, toConnectError(err)
	}
	resp := connect.NewResponse(&identityv1.LoginResponse{
		Email:  owner.Email,
		UserId: owner.UserID,
	})
	if req.Header().Get(browserSessionHeader) != "1" {
		resp.Msg.Token = owner.Token
		resp.Msg.RefreshToken = owner.RefreshToken
	}
	setOwnerCookie(resp, req.Header(), owner.Token)
	return resp, nil
}

func (h *connectHandler) Register(ctx context.Context, req *connect.Request[identityv1.RegisterRequest]) (*connect.Response[identityv1.RegisterResponse], error) {
	owner, err := h.deps.Forwarder.Register(ctx, internalidentity.Registration{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
		Username: req.Msg.GetUsername(),
	})
	if err != nil {
		if !errors.Is(err, internalidentity.ErrEmailTaken) && !errors.Is(err, internalidentity.ErrInvalidInput) {
			h.deps.Logger.Printf("identity.Register: %v", err)
		}
		return nil, toConnectError(err)
	}
	resp := connect.NewResponse(&identityv1.RegisterResponse{
		Email:  owner.Email,
		UserId: owner.UserID,
	})
	if req.Header().Get(browserSessionHeader) != "1" {
		resp.Msg.Token = owner.Token
		resp.Msg.RefreshToken = owner.RefreshToken
	}
	setOwnerCookie(resp, req.Header(), owner.Token)
	return resp, nil
}

func setOwnerCookie[T any](resp *connect.Response[T], headers http.Header, token string) {
	secure := ownerCookieSecure(headers)
	resp.Header().Add("Set-Cookie", (&http.Cookie{
		Name: internalauth.OwnerCookieName, Value: token, Path: "/", MaxAge: int((time.Hour).Seconds()),
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	}).String())
}

// Logout clears the browser-only owner cookie. CLI callers continue to manage
// their bearer token lifecycle themselves.
func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: internalauth.OwnerCookieName, Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: ownerCookieSecure(r.Header), SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func ownerCookieSecure(headers http.Header) bool {
	if configured := strings.TrimSpace(os.Getenv("VROOLI_AUTH_COOKIE_SECURE")); configured != "" {
		secure, _ := strconv.ParseBool(configured)
		return secure
	}
	return strings.EqualFold(strings.TrimSpace(headers.Get("X-Forwarded-Proto")), "https")
}

// toConnectError maps forwarder errors to Connect codes the UI/CLI can branch
// on. The message is preserved so an input-validation failure (weak password,
// bad email) surfaces verbatim.
func toConnectError(err error) error {
	switch {
	case errors.Is(err, internalidentity.ErrInvalidCredentials):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, internalidentity.ErrEmailTaken):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, internalidentity.ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalidentity.ErrAuthUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
