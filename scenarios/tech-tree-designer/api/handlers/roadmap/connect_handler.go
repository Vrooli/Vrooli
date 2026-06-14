package roadmap

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	roadmapdomain "tech-tree-designer/internal/roadmap"

	roadmapv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap"
	roadmapconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap/roadmap_v1connect"
)

type Handler struct {
	roadmapconnect.UnimplementedRoadmapServiceHandler
	service *roadmapdomain.Service
}

func NewHandler(service *roadmapdomain.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListSectors(ctx context.Context, _ *connect.Request[roadmapv1.ListSectorsRequest]) (*connect.Response[roadmapv1.ListSectorsResponse], error) {
	sectors, err := h.service.ListSectors(ctx)
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &roadmapv1.ListSectorsResponse{Sectors: make([]*roadmapv1.Sector, 0, len(sectors))}
	for _, sector := range sectors {
		resp.Sectors = append(resp.Sectors, sectorToProto(sector))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) UpsertSector(ctx context.Context, req *connect.Request[roadmapv1.UpsertSectorRequest]) (*connect.Response[roadmapv1.Sector], error) {
	sector := req.Msg.GetSector()
	out, err := h.service.UpsertSector(ctx, roadmapdomain.Sector{
		Slug:        sector.GetSlug(),
		Name:        sector.GetName(),
		Description: sector.GetDescription(),
	})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(sectorToProto(out)), nil
}

func (h *Handler) ListMilestones(ctx context.Context, _ *connect.Request[roadmapv1.ListMilestonesRequest]) (*connect.Response[roadmapv1.ListMilestonesResponse], error) {
	milestones, err := h.service.ListMilestones(ctx)
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &roadmapv1.ListMilestonesResponse{Milestones: make([]*roadmapv1.Milestone, 0, len(milestones))}
	for _, milestone := range milestones {
		resp.Milestones = append(resp.Milestones, milestoneToProto(milestone))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) UpsertMilestone(ctx context.Context, req *connect.Request[roadmapv1.UpsertMilestoneRequest]) (*connect.Response[roadmapv1.Milestone], error) {
	milestone := req.Msg.GetMilestone()
	out, err := h.service.UpsertMilestone(ctx, roadmapdomain.Milestone{
		ID:                milestone.GetId(),
		Name:              milestone.GetName(),
		Description:       milestone.GetDescription(),
		RequiredScenarios: milestone.GetRequiredScenarios(),
	})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(milestoneToProto(out)), nil
}

func (h *Handler) GetProgress(ctx context.Context, req *connect.Request[roadmapv1.GetProgressRequest]) (*connect.Response[roadmapv1.ProgressRollup], error) {
	progress, err := h.service.GetProgress(ctx, roadmapdomain.ProgressFilter{Sector: req.Msg.GetSector(), Tier: req.Msg.GetTier()})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(progress), nil
}

func toConnectError(err error) error {
	var invalid roadmapdomain.ErrInvalidArgument
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w", err))
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%w", err))
}

func sectorToProto(sector roadmapdomain.Sector) *roadmapv1.Sector {
	return &roadmapv1.Sector{
		Slug:        sector.Slug,
		Name:        sector.Name,
		Description: sector.Description,
	}
}

func milestoneToProto(milestone roadmapdomain.Milestone) *roadmapv1.Milestone {
	return &roadmapv1.Milestone{
		Id:                milestone.ID,
		Name:              milestone.Name,
		Description:       milestone.Description,
		RequiredScenarios: milestone.RequiredScenarios,
	}
}
