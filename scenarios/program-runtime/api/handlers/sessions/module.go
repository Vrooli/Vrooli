package sessions

import (
	"context"
	"errors"
	"sort"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/sessions/sessions_v1connect"
	"program-runtime/internal/module"
	internalsessions "program-runtime/internal/sessions"
)

type handler struct {
	sessionsconnect.UnimplementedSessionServiceHandler
	manager *internalsessions.Manager
}

func Module(manager *internalsessions.Manager) module.Module {
	return module.Module{Name: "sessions", Mount: func(r *mux.Router) {
		path, h := sessionsconnect.NewSessionServiceHandler(&handler{manager: manager})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func (h *handler) CreateSession(ctx context.Context, req *connect.Request[sessionsv1.CreateSessionRequest]) (*connect.Response[sessionsv1.CreateSessionResponse], error) {
	s, err := h.manager.CreateWithBudgets(ctx, req.Msg.Name, req.Msg.SandboxWorkspace, req.Msg.Grants, req.Msg.GetInferenceCeilingMicros(), req.Msg.GetDelegationCeilingMicros())
	if err != nil {
		if errors.Is(err, internalsessions.ErrInvalidWorkspace) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&sessionsv1.CreateSessionResponse{Session: toProto(s)}), nil
}

func (h *handler) GetSession(ctx context.Context, req *connect.Request[sessionsv1.GetSessionRequest]) (*connect.Response[sessionsv1.GetSessionResponse], error) {
	s, err := h.manager.Get(ctx, req.Msg.Id)
	if errors.Is(err, internalsessions.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sessionsv1.GetSessionResponse{Session: toProto(s)}), nil
}

func (h *handler) ListSessions(ctx context.Context, _ *connect.Request[sessionsv1.ListSessionsRequest]) (*connect.Response[sessionsv1.ListSessionsResponse], error) {
	out := &sessionsv1.ListSessionsResponse{}
	for _, s := range h.manager.List(ctx) {
		out.Sessions = append(out.Sessions, toProto(s))
	}
	out.Count = int64(len(out.Sessions))
	return connect.NewResponse(out), nil
}

func (h *handler) DeleteSession(ctx context.Context, req *connect.Request[sessionsv1.DeleteSessionRequest]) (*connect.Response[sessionsv1.DeleteSessionResponse], error) {
	s, err := h.manager.Delete(ctx, req.Msg.Id, req.Msg.Reason)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&sessionsv1.DeleteSessionResponse{Session: toProto(s)}), nil
}

func (h *handler) GrantSession(ctx context.Context, req *connect.Request[sessionsv1.GrantSessionRequest]) (*connect.Response[sessionsv1.GrantSessionResponse], error) {
	s, err := h.manager.Grant(ctx, req.Msg.Id, req.Msg.Grants)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&sessionsv1.GrantSessionResponse{Session: toProto(s)}), nil
}

func toProto(s *internalsessions.Session) *sessionsv1.Session {
	out := &sessionsv1.Session{Id: s.ID, Name: s.Name, State: s.State, CreatedAt: s.CreatedAt.Format(time.RFC3339Nano), LastActivityAt: s.LastActivityAt.Format(time.RFC3339Nano), SandboxWorkspace: s.SandboxWorkspace, ReclaimedReason: s.ReclaimedReason, InferenceCostMicros: s.InferenceCostMicros, InferenceTokens: s.InferenceTokens, DelegationCostMicros: s.DelegationCostMicros, InferenceCeilingMicros: s.InferenceCeilingMicros, DelegationCeilingMicros: s.DelegationCeilingMicros, DelegationSpendMeasured: s.DelegationSpendMeasured, DelegationSpendNote: s.DelegationSpendNote}
	for grant := range s.Grants {
		out.Grants = append(out.Grants, grant)
	}
	sort.Strings(out.Grants)
	return out
}
