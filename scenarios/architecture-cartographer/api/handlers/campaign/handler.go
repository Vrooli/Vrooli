// Package campaign is the Connect-RPC surface for the campaign domain —
// the stateful scenario-improvement tracker that ingests the shared
// ArchitectureFinding photograph, tracks each finding through a lifecycle,
// hands the agent a profile-ranked worklist, and reconciles re-audits by
// stable id.
package campaign

import (
	"context"
	"errors"

	"architecture-cartographer/internal/campaign"

	"connectrpc.com/connect"
	campaignv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign/campaign_v1connect"
)

// Handler implements campaign_v1connect.CampaignServiceHandler.
type Handler struct {
	campaign_v1connect.UnimplementedCampaignServiceHandler
	svc campaign.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc campaign.Service) *Handler { return &Handler{svc: svc} }

var _ campaign_v1connect.CampaignServiceHandler = (*Handler)(nil)

// errorToConnectCode maps domain errors to Connect codes.
func errorToConnectCode(err error) connect.Code {
	var notFoundC campaign.ErrCampaignNotFound
	var notFoundF campaign.ErrFindingNotFound
	var invalid campaign.ErrInvalidInput
	switch {
	case errors.As(err, &notFoundC), errors.As(err, &notFoundF):
		return connect.CodeNotFound
	case errors.As(err, &invalid):
		return connect.CodeInvalidArgument
	default:
		return connect.CodeInternal
	}
}

func (h *Handler) CreateCampaign(ctx context.Context, req *connect.Request[campaignv1.CreateCampaignRequest]) (*connect.Response[campaignv1.CreateCampaignResponse], error) {
	st, err := h.svc.Create(ctx, req.Msg.GetScenario(), req.Msg.GetName(), req.Msg.GetFindings())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&campaignv1.CreateCampaignResponse{Status: statusProjectionToProto(st)}), nil
}

func (h *Handler) ListCampaigns(ctx context.Context, req *connect.Request[campaignv1.ListCampaignsRequest]) (*connect.Response[campaignv1.ListCampaignsResponse], error) {
	cs, err := h.svc.List(ctx, req.Msg.GetScenario())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	out := make([]*campaignv1.Campaign, 0, len(cs))
	for _, c := range cs {
		out = append(out, campaignToProto(c))
	}
	return connect.NewResponse(&campaignv1.ListCampaignsResponse{Campaigns: out}), nil
}

func (h *Handler) GetCampaignStatus(ctx context.Context, req *connect.Request[campaignv1.GetCampaignStatusRequest]) (*connect.Response[campaignv1.GetCampaignStatusResponse], error) {
	st, err := h.svc.Status(ctx, req.Msg.GetCampaignId())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&campaignv1.GetCampaignStatusResponse{Status: statusProjectionToProto(st)}), nil
}

func (h *Handler) NextCampaignStep(ctx context.Context, req *connect.Request[campaignv1.NextCampaignStepRequest]) (*connect.Response[campaignv1.NextCampaignStepResponse], error) {
	findings, err := h.svc.Next(ctx, req.Msg.GetCampaignId(), req.Msg.GetProfile())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&campaignv1.NextCampaignStepResponse{Items: findingsToProto(findings)}), nil
}

func (h *Handler) ResolveItem(ctx context.Context, req *connect.Request[campaignv1.ResolveItemRequest]) (*connect.Response[campaignv1.ResolveItemResponse], error) {
	f, err := h.svc.Resolve(ctx, req.Msg.GetCampaignId(), req.Msg.GetStableId(), req.Msg.GetNote())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&campaignv1.ResolveItemResponse{Item: findingToProto(f)}), nil
}

func (h *Handler) ApplyItem(ctx context.Context, req *connect.Request[campaignv1.ApplyItemRequest]) (*connect.Response[campaignv1.ApplyItemResponse], error) {
	// v1: apply records a hand-fix as a status-only transition (no file
	// write). Auto-execution of file-op fixes stays deferred to the
	// apply-execution plan.
	f, err := h.svc.Resolve(ctx, req.Msg.GetCampaignId(), req.Msg.GetStableId(), "applied (manual)")
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&campaignv1.ApplyItemResponse{Item: findingToProto(f)}), nil
}

func (h *Handler) ReauditCampaign(ctx context.Context, req *connect.Request[campaignv1.ReauditCampaignRequest]) (*connect.Response[campaignv1.ReauditCampaignResponse], error) {
	res, err := h.svc.Reaudit(ctx, req.Msg.GetCampaignId(), req.Msg.GetFindings())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&campaignv1.ReauditCampaignResponse{
		Validated:   findingsToProto(res.Validated),
		StillOpen:   findingsToProto(res.StillOpen),
		Regressions: findingsToProto(res.Regressions),
		Status:      statusProjectionToProto(res.Status),
	}), nil
}

func (h *Handler) CloseCampaign(ctx context.Context, req *connect.Request[campaignv1.CloseCampaignRequest]) (*connect.Response[campaignv1.CloseCampaignResponse], error) {
	st, err := h.svc.Close(ctx, req.Msg.GetCampaignId())
	if err != nil {
		return nil, connect.NewError(errorToConnectCode(err), err)
	}
	return connect.NewResponse(&campaignv1.CloseCampaignResponse{Status: statusProjectionToProto(st)}), nil
}
