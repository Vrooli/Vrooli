package workspace

import (
	"context"
	"errors"
	"log"

	internal "nutrition-planner/internal/workspace"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct {
	service internal.Service
	logger  *log.Logger
}

func NewConnectHandler(service internal.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service, logger}
}

func owner(ctx context.Context) (string, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	return p.Subject, nil
}

func (h *connectHandler) ListWorkspaces(ctx context.Context, _ *connect.Request[v1.ListWorkspacesRequest]) (*connect.Response[v1.ListWorkspacesResponse], error) {
	o, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	items, err := h.service.List(ctx, o)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListWorkspacesResponse{Workspaces: make([]*v1.Workspace, 0, len(items))}
	for _, w := range items {
		out.Workspaces = append(out.Workspaces, toProto(w))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateWorkspace(ctx context.Context, req *connect.Request[v1.CreateWorkspaceRequest]) (*connect.Response[v1.CreateWorkspaceResponse], error) {
	o, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	w, err := h.service.Create(ctx, internal.CreateInput{Name: req.Msg.Name, OwnerSubject: o, IdempotencyKey: req.Msg.IdempotencyKey})
	if err != nil {
		var inv internal.ErrInvalid
		if errors.As(err, &inv) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		h.logger.Printf("workspace create: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateWorkspaceResponse{Workspace: toProto(w)}), nil
}

func (h *connectHandler) GetWorkspace(ctx context.Context, req *connect.Request[v1.GetWorkspaceRequest]) (*connect.Response[v1.GetWorkspaceResponse], error) {
	o, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	w, err := h.service.Get(ctx, req.Msg.Id, o)
	if err != nil {
		var nf internal.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden internal.ErrForbidden
		if errors.As(err, &forbidden) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetWorkspaceResponse{Workspace: toProto(w)}), nil
}

func toProto(w internal.Workspace) *v1.Workspace {
	return &v1.Workspace{Id: w.ID, Name: w.Name, OwnerSubject: w.OwnerSubject, Revision: w.Revision, CreatedAt: timestamppb.New(w.CreatedAt), UpdatedAt: timestamppb.New(w.UpdatedAt)}
}
