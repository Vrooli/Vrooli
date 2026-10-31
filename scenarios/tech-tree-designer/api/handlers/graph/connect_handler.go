package graph

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	graphdomain "tech-tree-designer/internal/graph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph/graph_v1connect"
)

type Handler struct {
	graphconnect.UnimplementedGraphServiceHandler
	service *graphdomain.Service
}

func NewHandler(service *graphdomain.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) DescribeTechTree(ctx context.Context, req *connect.Request[graphv1.DescribeTechTreeRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	msg := req.Msg
	graph, err := h.service.Describe(ctx, graphdomain.SourceRequest{
		ScenarioFilter:  msg.GetScenarioFilter(),
		Limit:           msg.GetLimit(),
		StabilityFilter: msg.GetStabilityFilter(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: graph}), nil
}

func (h *Handler) GetNeighborhood(ctx context.Context, req *connect.Request[graphv1.GetNeighborhoodRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	graph, err := h.service.Neighborhood(ctx, req.Msg.GetScenario(), req.Msg.GetDepth(), req.Msg.GetScenarioFilter())
	if err != nil {
		return nil, requestError(err)
	}
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: graph}), nil
}

func (h *Handler) FindPath(ctx context.Context, req *connect.Request[graphv1.FindPathRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	graph, err := h.service.Path(ctx, req.Msg.GetFromScenario(), req.Msg.GetToScenario(), req.Msg.GetScenarioFilter())
	if err != nil {
		return nil, requestError(err)
	}
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: graph}), nil
}

func (h *Handler) ListAncestors(ctx context.Context, req *connect.Request[graphv1.ListAncestorsRequest]) (*connect.Response[graphv1.DescribeTechTreeResponse], error) {
	graph, err := h.service.Ancestors(ctx, req.Msg.GetScenario(), req.Msg.GetScenarioFilter())
	if err != nil {
		return nil, requestError(err)
	}
	return connect.NewResponse(&graphv1.DescribeTechTreeResponse{Graph: graph}), nil
}

func (h *Handler) ExportTechTree(ctx context.Context, req *connect.Request[graphv1.ExportTechTreeRequest]) (*connect.Response[graphv1.ExportTechTreeResponse], error) {
	resp, err := h.service.Export(ctx, graphdomain.SourceRequest{
		ScenarioFilter:  req.Msg.GetScenarioFilter(),
		StabilityFilter: req.Msg.GetStabilityFilter(),
	}, req.Msg.GetFormat())
	if err != nil {
		return nil, requestError(err)
	}
	return connect.NewResponse(resp), nil
}

func requestError(err error) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w", err))
}
