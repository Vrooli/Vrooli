package facets

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"

	internalfacets "source-ledger/internal/facets"
	"source-ledger/internal/policy"

	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
)

type connectHandler struct {
	service  *internalfacets.Service
	registry *policy.Registry
	logger   *log.Logger
}

func NewConnectHandler(s *internalfacets.Service, l *log.Logger, registries ...*policy.Registry) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	var registry *policy.Registry
	if len(registries) > 0 {
		registry = registries[0]
	}
	return &connectHandler{service: s, registry: registry, logger: l}
}

func (h *connectHandler) ListFacets(ctx context.Context, req *connect.Request[facetsv1.ListFacetsRequest]) (*connect.Response[facetsv1.ListFacetsResponse], error) {
	if h.registry != nil {
		if _, err := h.registry.Resolve(ctx, req.Msg.GetScope()); err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
	}
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	items, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &facetsv1.ListFacetsResponse{}
	for _, d := range items {
		out.Facets = append(out.Facets, facetProto(d))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CountUnassigned(ctx context.Context, req *connect.Request[facetsv1.CountUnassignedRequest]) (*connect.Response[facetsv1.CountUnassignedResponse], error) {
	if h.registry != nil {
		if _, err := h.registry.Resolve(ctx, req.Msg.GetScope()); err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
	}
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	count, err := h.service.CountUnassigned(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&facetsv1.CountUnassignedResponse{Scope: string(policy.ScopeFromContext(ctx)), Count: int32(count)}), nil
}

func (h *connectHandler) EnsureFacet(ctx context.Context, req *connect.Request[facetsv1.EnsureFacetRequest]) (*connect.Response[facetsv1.EnsureFacetResponse], error) {
	if req.Msg.GetFacet() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("facet is required"))
	}
	if h.registry != nil {
		if _, err := h.registry.Resolve(ctx, req.Msg.GetScope()); err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
	}
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	facet := req.Msg.GetFacet()
	definition, err := h.service.Ensure(ctx, internalfacets.Definition{
		ID: facet.GetId(), Label: facet.GetLabel(), ClassificationGuidance: facet.GetGuidance(),
		RetentionPolicy: facet.GetRetentionPolicy(), CompactionEligible: facet.GetCompactionEligible(), ResidentBudget: int(facet.GetResidentBudget()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&facetsv1.EnsureFacetResponse{Facet: facetProto(definition)}), nil
}

func (h *connectHandler) DeleteFacet(ctx context.Context, req *connect.Request[facetsv1.DeleteFacetRequest]) (*connect.Response[facetsv1.DeleteFacetResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	if err := h.service.Delete(ctx, req.Msg.GetFacetId()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&facetsv1.DeleteFacetResponse{}), nil
}

func (h *connectHandler) SetFacetPolicy(ctx context.Context, req *connect.Request[facetsv1.SetFacetPolicyRequest]) (*connect.Response[facetsv1.SetFacetPolicyResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	definition, err := h.service.SetPolicy(ctx, internalfacets.FacetPolicy{
		ID:                 req.Msg.GetFacetId(),
		RetentionPolicy:    req.Msg.GetRetentionPolicy(),
		CompactionEligible: req.Msg.GetCompactionEligible(),
		ResidentBudget:     int(req.Msg.GetResidentBudget()),
	})
	if err != nil {
		var unknown internalfacets.ErrUnknownFacet
		if errors.As(err, &unknown) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&facetsv1.SetFacetPolicyResponse{Facet: facetProto(definition)}), nil
}

func facetProto(d internalfacets.Definition) *facetsv1.Facet {
	return &facetsv1.Facet{Id: d.ID, Label: d.Label, RetentionPolicy: d.RetentionPolicy, Guidance: d.ClassificationGuidance, CompactionEligible: d.CompactionEligible, ResidentBudget: int32(d.ResidentBudget)}
}

func (h *connectHandler) AssignFacet(ctx context.Context, req *connect.Request[facetsv1.AssignFacetRequest]) (*connect.Response[facetsv1.AssignFacetResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
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
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	if err := h.service.SetPin(ctx, req.Msg.GetEntryId(), req.Msg.GetPinned()); err != nil {
		var budget internalfacets.ErrPinBudgetExceeded
		if errors.As(err, &budget) {
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&facetsv1.SetPinResponse{}), nil
}

func (h *connectHandler) ListPinProposals(ctx context.Context, req *connect.Request[facetsv1.ListPinProposalsRequest]) (*connect.Response[facetsv1.ListPinProposalsResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	items, err := h.service.ListPinProposals(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &facetsv1.ListPinProposalsResponse{}
	for _, item := range items {
		response.Proposals = append(response.Proposals, &facetsv1.PinProposal{Id: item.ID, EntryIds: item.EntryIDs, Rationale: item.Rationale})
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListPinCandidates(ctx context.Context, req *connect.Request[facetsv1.ListPinCandidatesRequest]) (*connect.Response[facetsv1.ListPinCandidatesResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	items, err := h.service.ListPinCandidates(ctx, int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &facetsv1.ListPinCandidatesResponse{}
	for _, item := range items {
		candidate := &facetsv1.PinCandidate{EntryId: item.EntryID, Body: item.Body, RecallCount: int32(item.RecallCount), CreatedAt: item.CreatedAt.Format(time.RFC3339Nano)}
		if item.LastRecalledAt != nil {
			candidate.LastRecalledAt = item.LastRecalledAt.Format(time.RFC3339Nano)
		}
		response.Candidates = append(response.Candidates, candidate)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ResolvePinProposal(ctx context.Context, req *connect.Request[facetsv1.ResolvePinProposalRequest]) (*connect.Response[facetsv1.ResolvePinProposalResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	if err := h.service.ResolvePinProposal(ctx, req.Msg.GetProposalId(), req.Msg.GetAccept()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&facetsv1.ResolvePinProposalResponse{}), nil
}

func (h *connectHandler) MarkSuperseded(ctx context.Context, req *connect.Request[facetsv1.MarkSupersededRequest]) (*connect.Response[facetsv1.MarkSupersededResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	if err := h.service.MarkSuperseded(ctx, req.Msg.GetEntryId(), req.Msg.GetReplacementEntryId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&facetsv1.MarkSupersededResponse{}), nil
}

func (h *connectHandler) ResolveThread(ctx context.Context, req *connect.Request[facetsv1.ResolveThreadRequest]) (*connect.Response[facetsv1.ResolveThreadResponse], error) {
	ctx = policy.WithScope(ctx, req.Msg.GetScope())
	if err := h.service.ResolveThread(ctx, req.Msg.GetEntryId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&facetsv1.ResolveThreadResponse{}), nil
}
