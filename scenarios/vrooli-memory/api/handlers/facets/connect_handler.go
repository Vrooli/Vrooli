package facets

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets"
	internalfacets "vrooli-memory/internal/facets"
)

type connectHandler struct {
	service *internalfacets.Service
	logger  *log.Logger
}

func NewConnectHandler(s *internalfacets.Service, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{service: s, logger: l}
}

func (h *connectHandler) ListFacets(ctx context.Context, _ *connect.Request[facetsv1.ListFacetsRequest]) (*connect.Response[facetsv1.ListFacetsResponse], error) {
	items, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &facetsv1.ListFacetsResponse{}
	for _, d := range items {
		out.Facets = append(out.Facets, &facetsv1.Facet{Id: d.ID, Label: d.Label, RetentionPolicy: d.RetentionPolicy})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AssignFacet(ctx context.Context, req *connect.Request[facetsv1.AssignFacetRequest]) (*connect.Response[facetsv1.AssignFacetResponse], error) {
	_, err := h.service.ReFacet(ctx, internalfacets.Assignment{EntryID: req.Msg.GetEntryId(), FacetID: req.Msg.GetFacetId()})
	if err != nil {
		var unknown internalfacets.ErrUnknownFacet
		if errors.As(err, &unknown) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&facetsv1.AssignFacetResponse{}), nil
}

func (h *connectHandler) SetPin(ctx context.Context, req *connect.Request[facetsv1.SetPinRequest]) (*connect.Response[facetsv1.SetPinResponse], error) {
	if err := h.service.SetPin(ctx, req.Msg.GetEntryId(), req.Msg.GetPinned()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&facetsv1.SetPinResponse{}), nil
}

func (h *connectHandler) ListPinProposals(context.Context, *connect.Request[facetsv1.ListPinProposalsRequest]) (*connect.Response[facetsv1.ListPinProposalsResponse], error) {
	return connect.NewResponse(&facetsv1.ListPinProposalsResponse{}), nil
}

func (h *connectHandler) ResolvePinProposal(context.Context, *connect.Request[facetsv1.ResolvePinProposalRequest]) (*connect.Response[facetsv1.ResolvePinProposalResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("pin proposal resolution is not implemented"))
}

func (h *connectHandler) MarkSuperseded(ctx context.Context, req *connect.Request[facetsv1.MarkSupersededRequest]) (*connect.Response[facetsv1.MarkSupersededResponse], error) {
	if err := h.service.MarkSuperseded(ctx, req.Msg.GetEntryId(), req.Msg.GetReplacementEntryId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&facetsv1.MarkSupersededResponse{}), nil
}
