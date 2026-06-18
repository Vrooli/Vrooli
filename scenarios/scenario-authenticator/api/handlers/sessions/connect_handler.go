// Package sessions implements the SessionsService Connect-RPC surface (list,
// revoke, revoke-all) over the accounts.Service facade. RevokeSession preserves
// the idempotent semantics device-sync-hub relies on when un-pairing a device.
package sessions

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions"

	"scenario-authenticator/internal/accounts"
	intsessions "scenario-authenticator/internal/sessions"
)

// Deps wires the SessionsService handler.
type Deps struct {
	Service *accounts.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the SessionsService handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListSessions(ctx context.Context, req *connect.Request[sessionsv1.ListSessionsRequest]) (*connect.Response[sessionsv1.ListSessionsResponse], error) {
	list, err := h.deps.Service.ListSessions(ctx, req.Msg.GetAccessToken())
	if err != nil {
		if errors.Is(err, accounts.ErrInvalidCredentials) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired token"))
		}
		h.deps.Logger.Printf("sessions.ListSessions: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	resp := &sessionsv1.ListSessionsResponse{Sessions: make([]*sessionsv1.Session, 0, len(list))}
	for _, s := range list {
		resp.Sessions = append(resp.Sessions, sessionToProto(s))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) RevokeSession(ctx context.Context, req *connect.Request[sessionsv1.RevokeSessionRequest]) (*connect.Response[sessionsv1.RevokeSessionResponse], error) {
	if err := h.deps.Service.RevokeSession(ctx, req.Msg.GetSessionId()); err != nil {
		h.deps.Logger.Printf("sessions.RevokeSession: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	return connect.NewResponse(&sessionsv1.RevokeSessionResponse{}), nil
}

func (h *connectHandler) RevokeAllSessions(ctx context.Context, req *connect.Request[sessionsv1.RevokeAllSessionsRequest]) (*connect.Response[sessionsv1.RevokeAllSessionsResponse], error) {
	n, err := h.deps.Service.RevokeAllSessions(ctx, req.Msg.GetAccessToken())
	if err != nil {
		if errors.Is(err, accounts.ErrInvalidCredentials) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired token"))
		}
		h.deps.Logger.Printf("sessions.RevokeAllSessions: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	return connect.NewResponse(&sessionsv1.RevokeAllSessionsResponse{RevokedCount: int64(n)}), nil
}

func sessionToProto(s intsessions.Session) *sessionsv1.Session {
	ps := &sessionsv1.Session{
		Id: s.ID, UserId: s.UserID, IpAddress: s.IPAddress, UserAgent: s.UserAgent,
	}
	if !s.CreatedAt.IsZero() {
		ps.CreatedAt = timestamppb.New(s.CreatedAt)
	}
	if !s.ExpiresAt.IsZero() {
		ps.ExpiresAt = timestamppb.New(s.ExpiresAt)
	}
	return ps
}
