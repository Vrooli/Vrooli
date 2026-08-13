package facts

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internalfacts "code-facts/internal/facts"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type Deps struct {
	Service *internalfacts.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Service == nil {
		d.Service = internalfacts.NewService()
	}
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) DescribeCodeFacts(ctx context.Context, req *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error) {
	report, err := h.deps.Service.Describe(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[factsv1.SearchRequest]) (*connect.Response[factsv1.SearchResponse], error) {
	response, err := h.deps.Service.Search(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) DescribeFleetImports(ctx context.Context, req *connect.Request[factsv1.DescribeFleetImportsRequest]) (*connect.Response[factsv1.DescribeFleetImportsResponse], error) {
	report, err := h.deps.Service.DescribeFleetImports(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) ListSurfaces(ctx context.Context, req *connect.Request[factsv1.ListSurfacesRequest]) (*connect.Response[factsv1.ListSurfacesResponse], error) {
	report, err := h.deps.Service.Surfaces(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) CheckProtoAdoption(ctx context.Context, req *connect.Request[factsv1.CheckProtoAdoptionRequest]) (*connect.Response[factsv1.ProofReport], error) {
	report, err := h.deps.Service.ProtoAdoption(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) CheckEndpointProof(ctx context.Context, req *connect.Request[factsv1.CheckEndpointProofRequest]) (*connect.Response[factsv1.ProofReport], error) {
	report, err := h.deps.Service.EndpointProof(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) GetCacheStatus(ctx context.Context, req *connect.Request[factsv1.GetCacheStatusRequest]) (*connect.Response[factsv1.CacheStatus], error) {
	report, err := h.deps.Service.CacheStatus(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) InspectCache(ctx context.Context, req *connect.Request[factsv1.InspectCacheRequest]) (*connect.Response[factsv1.CacheStatus], error) {
	report, err := h.deps.Service.InspectCache(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}

func (h *connectHandler) ClearCache(ctx context.Context, req *connect.Request[factsv1.ClearCacheRequest]) (*connect.Response[factsv1.ClearCacheResponse], error) {
	report, err := h.deps.Service.ClearCache(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(report), nil
}
