// Package auth implements the AccountsService Connect-RPC surface (register,
// login, refresh, logout, validate) over the accounts.Service orchestrator. It
// is the typed transport edge: it maps the service's domain errors to Connect
// codes and translates between the proto wire types and the domain types.
package auth

import (
	"context"
	"errors"
	"log"
	"net"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"

	"scenario-authenticator/internal/accounts"
)

// Deps wires the AccountsService handler.
type Deps struct {
	Service *accounts.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the AccountsService handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Register(ctx context.Context, req *connect.Request[accountsv1.RegisterRequest]) (*connect.Response[accountsv1.RegisterResponse], error) {
	res, err := h.deps.Service.Register(ctx, accounts.RegisterParams{
		Email: req.Msg.GetEmail(), Password: req.Msg.GetPassword(),
		Username: req.Msg.GetUsername(), Realm: req.Msg.GetRealm(),
	}, metaFrom(req))
	if err != nil {
		return nil, h.toConnectErr("Register", err)
	}
	return connect.NewResponse(&accountsv1.RegisterResponse{
		Account: accountToProto(res.Account),
		Tokens:  tokensToProto(res),
	}), nil
}

func (h *connectHandler) Login(ctx context.Context, req *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	res, err := h.deps.Service.Login(ctx, accounts.LoginParams{
		Email: req.Msg.GetEmail(), Password: req.Msg.GetPassword(), Realm: req.Msg.GetRealm(),
	}, metaFrom(req))
	if err != nil {
		return nil, h.toConnectErr("Login", err)
	}
	return connect.NewResponse(&accountsv1.LoginResponse{
		Account: accountToProto(res.Account),
		Tokens:  tokensToProto(res),
	}), nil
}

func (h *connectHandler) Refresh(ctx context.Context, req *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
	res, err := h.deps.Service.Refresh(ctx, req.Msg.GetRefreshToken(), metaFrom(req))
	if err != nil {
		return nil, h.toConnectErr("Refresh", err)
	}
	return connect.NewResponse(&accountsv1.RefreshResponse{Tokens: tokensToProto(res)}), nil
}

func (h *connectHandler) Logout(ctx context.Context, req *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error) {
	if err := h.deps.Service.Logout(ctx, req.Msg.GetAccessToken(), metaFrom(req)); err != nil {
		return nil, h.toConnectErr("Logout", err)
	}
	return connect.NewResponse(&accountsv1.LogoutResponse{}), nil
}

func (h *connectHandler) Validate(ctx context.Context, req *connect.Request[accountsv1.ValidateRequest]) (*connect.Response[accountsv1.ValidateResponse], error) {
	vt, ok, err := h.deps.Service.Validate(ctx, req.Msg.GetAccessToken())
	if err != nil {
		return nil, h.toConnectErr("Validate", err)
	}
	resp := &accountsv1.ValidateResponse{Valid: ok}
	if ok {
		resp.UserId = vt.UserID
		resp.Email = vt.Email
		resp.Roles = vt.Roles
		resp.Realm = vt.Realm
		if !vt.ExpiresAt.IsZero() {
			resp.ExpiresAt = timestamppb.New(vt.ExpiresAt)
		}
	}
	return connect.NewResponse(resp), nil
}

// toConnectErr maps the service's domain errors to Connect codes. Login/refresh
// failures collapse to a single code+message (anti-enumeration); unexpected
// errors are logged and surfaced as Internal without leaking detail.
func (h *connectHandler) toConnectErr(op string, err error) error {
	switch {
	case errors.Is(err, accounts.ErrEmailTaken):
		return connect.NewError(connect.CodeAlreadyExists, errors.New("email already registered"))
	case errors.Is(err, accounts.ErrInvalidCredentials):
		return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid email or password"))
	case errors.Is(err, accounts.ErrRefreshRejected):
		return connect.NewError(connect.CodeUnauthenticated, errors.New("refresh token rejected"))
	case errors.Is(err, accounts.ErrAccountLocked):
		return connect.NewError(connect.CodePermissionDenied, accounts.ErrAccountLocked)
	}
	var inputErr accounts.InvalidInputError
	if errors.As(err, &inputErr) {
		return connect.NewError(connect.CodeInvalidArgument, errors.New(inputErr.Msg))
	}
	h.deps.Logger.Printf("auth.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

func accountToProto(a accounts.Account) *accountsv1.Account {
	pa := &accountsv1.Account{
		Id: a.ID, Email: a.Email, Username: a.Username, Roles: a.Roles,
		Realm: a.RealmID, EmailVerified: a.EmailVerified,
	}
	if !a.CreatedAt.IsZero() {
		pa.CreatedAt = timestamppb.New(a.CreatedAt)
	}
	return pa
}

func tokensToProto(r accounts.AuthResult) *accountsv1.TokenPair {
	tp := &accountsv1.TokenPair{AccessToken: r.AccessToken, RefreshToken: r.RefreshToken}
	if !r.AccessExpiresAt.IsZero() {
		tp.AccessTokenExpiresAt = timestamppb.New(r.AccessExpiresAt)
	}
	return tp
}

// metaFrom extracts the request-scoped IP + user agent for sessions/audit. IP
// honors the standard forwarding headers then falls back to the peer address.
func metaFrom[T any](req *connect.Request[T]) accounts.RequestMeta {
	hdr := req.Header()
	ip := clientIP(hdr.Get("X-Forwarded-For"), hdr.Get("X-Real-IP"), req.Peer().Addr)
	return accounts.RequestMeta{IP: ip, UserAgent: hdr.Get("User-Agent")}
}

func clientIP(forwardedFor, realIP, peerAddr string) string {
	if forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	}
	if realIP != "" {
		return realIP
	}
	if host, _, err := net.SplitHostPort(peerAddr); err == nil {
		return host
	}
	return peerAddr
}
