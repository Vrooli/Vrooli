package families

import (
	"context"
	"errors"

	"plan-manager/internal/families"

	"connectrpc.com/connect"
	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

type connectHandler struct{ service *families.Service }

func NewConnectHandler(service *families.Service) *connectHandler {
	return &connectHandler{service: service}
}

func connectError(err error) error {
	code := connect.CodeInvalidArgument
	if errors.Is(err, families.ErrNotFound) {
		code = connect.CodeNotFound
	}
	if errors.Is(err, families.ErrConflict) {
		code = connect.CodeAborted
	}
	return connect.NewError(code, err)
}

func (h *connectHandler) CreateFamily(ctx context.Context, req *connect.Request[familiesv1.CreateFamilyRequest]) (*connect.Response[familiesv1.CreateFamilyResponse], error) {
	f, err := h.service.Create(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.CreateFamilyResponse{Family: f}), nil
}
func (h *connectHandler) GetFamily(ctx context.Context, req *connect.Request[familiesv1.GetFamilyRequest]) (*connect.Response[familiesv1.GetFamilyResponse], error) {
	f, err := h.service.Get(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.GetFamilyResponse{Family: f}), nil
}
func (h *connectHandler) ListFamilies(ctx context.Context, req *connect.Request[familiesv1.ListFamiliesRequest]) (*connect.Response[familiesv1.ListFamiliesResponse], error) {
	items, next, err := h.service.List(ctx, req.Msg.GetPageSize(), req.Msg.GetPageToken())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.ListFamiliesResponse{Families: items, NextPageToken: next}), nil
}
func (h *connectHandler) UpdateFamily(ctx context.Context, req *connect.Request[familiesv1.UpdateFamilyRequest]) (*connect.Response[familiesv1.UpdateFamilyResponse], error) {
	f, err := h.service.Update(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.UpdateFamilyResponse{Family: f}), nil
}
func (h *connectHandler) PutMember(ctx context.Context, req *connect.Request[familiesv1.PutMemberRequest]) (*connect.Response[familiesv1.PutMemberResponse], error) {
	f, err := h.service.PutMember(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.PutMemberResponse{Family: f}), nil
}
func (h *connectHandler) RemoveMember(ctx context.Context, req *connect.Request[familiesv1.RemoveMemberRequest]) (*connect.Response[familiesv1.RemoveMemberResponse], error) {
	f, err := h.service.RemoveMember(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.RemoveMemberResponse{Family: f}), nil
}
func (h *connectHandler) PutClaim(ctx context.Context, req *connect.Request[familiesv1.PutClaimRequest]) (*connect.Response[familiesv1.PutClaimResponse], error) {
	f, err := h.service.PutClaim(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.PutClaimResponse{Family: f}), nil
}
func (h *connectHandler) RemoveClaim(ctx context.Context, req *connect.Request[familiesv1.RemoveClaimRequest]) (*connect.Response[familiesv1.RemoveClaimResponse], error) {
	f, err := h.service.RemoveClaim(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.RemoveClaimResponse{Family: f}), nil
}
func (h *connectHandler) ProposeGraph(ctx context.Context, req *connect.Request[familiesv1.ProposeGraphRequest]) (*connect.Response[familiesv1.ProposeGraphResponse], error) {
	f, err := h.service.ProposeGraph(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.ProposeGraphResponse{Family: f}), nil
}
func (h *connectHandler) ReviewGraph(ctx context.Context, req *connect.Request[familiesv1.ReviewGraphRequest]) (*connect.Response[familiesv1.ReviewGraphResponse], error) {
	f, err := h.service.ReviewGraph(ctx, req.Msg)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.ReviewGraphResponse{Family: f}), nil
}
func (h *connectHandler) GetFrontier(ctx context.Context, req *connect.Request[familiesv1.GetFrontierRequest]) (*connect.Response[familiesv1.GetFrontierResponse], error) {
	frontier, err := h.service.Frontier(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(frontier), nil
}

func (h *connectHandler) RenderFamily(ctx context.Context, req *connect.Request[familiesv1.RenderFamilyRequest]) (*connect.Response[familiesv1.RenderFamilyResponse], error) {
	family, markdown, err := h.service.Render(ctx, req.Msg.GetFamilyId())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&familiesv1.RenderFamilyResponse{Family: family, Markdown: markdown}), nil
}
