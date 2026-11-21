package admin

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internaladmin "landing-page-react-vite-api/internal/admin"
)

// Deps wires the AdminAuth Connect handler.
type Deps struct {
	Service *internaladmin.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the AdminAuthService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Login(ctx context.Context, req *connect.Request[landingv1.LoginRequest]) (*connect.Response[landingv1.AdminSessionResponse], error) {
	m := req.Msg
	if err := h.deps.Service.Login(ctx, m.Email, m.Password); err != nil {
		if errors.Is(err, internaladmin.ErrInvalidCredentials) {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		h.deps.Logger.Printf("admin.Login: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := connect.NewResponse(&landingv1.AdminSessionResponse{
		Email:         m.Email,
		Authenticated: true,
		ResetEnabled:  internaladmin.ResetEnabled(),
	})
	resp.Header().Set("Set-Cookie", h.deps.Service.SessionCookie(m.Email).String())
	return resp, nil
}

func (h *connectHandler) Logout(_ context.Context, _ *connect.Request[landingv1.LogoutRequest]) (*connect.Response[landingv1.LogoutResponse], error) {
	resp := connect.NewResponse(&landingv1.LogoutResponse{Success: true})
	resp.Header().Set("Set-Cookie", h.deps.Service.ClearCookie().String())
	return resp, nil
}

func (h *connectHandler) Session(_ context.Context, req *connect.Request[landingv1.SessionRequest]) (*connect.Response[landingv1.AdminSessionResponse], error) {
	email, ok := h.deps.Service.EmailFromHeader(req.Header())
	out := &landingv1.AdminSessionResponse{Authenticated: ok, ResetEnabled: internaladmin.ResetEnabled()}
	if ok {
		out.Email = email
	}
	return connect.NewResponse(out), nil
}
