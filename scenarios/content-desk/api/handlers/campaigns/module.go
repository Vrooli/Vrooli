// Package campaigns mounts the Campaigns Connect surface. Domain persistence
// arrives in the campaigns implementation phase; the contract is live now.
package campaigns

import (
	"context"
	"fmt"

	internalcampaigns "content-desk/internal/campaigns"
	"content-desk/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	campaignsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/campaigns"
	campaignsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/campaigns/campaigns_v1connect"
)

type handler struct{ repo internalcampaigns.Repository }

var _ campaignsconnect.CampaignsServiceHandler = handler{}

func (h handler) ListCampaigns(ctx context.Context, _ *connect.Request[campaignsv1.ListCampaignsRequest]) (*connect.Response[campaignsv1.ListCampaignsResponse], error) {
	campaigns, err := h.repo.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &campaignsv1.ListCampaignsResponse{}
	for _, campaign := range campaigns {
		response.Campaigns = append(response.Campaigns, campaignMessage(campaign))
	}
	return connect.NewResponse(response), nil
}

func (h handler) CreateCampaign(ctx context.Context, request *connect.Request[campaignsv1.CreateCampaignRequest]) (*connect.Response[campaignsv1.CreateCampaignResponse], error) {
	slots := make([]internalcampaigns.Slot, 0, len(request.Msg.Slots))
	for _, slot := range request.Msg.Slots {
		if slot == nil || slot.Capacity <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("campaign slots require a positive capacity"))
		}
		slots = append(slots, internalcampaigns.Slot{Channel: slot.Channel, Format: slot.Format, Capacity: int(slot.Capacity)})
	}
	campaign, err := h.repo.Create(ctx, internalcampaigns.Campaign{Name: request.Msg.Name, ScenarioNames: request.Msg.ScenarioNames}, request.Msg.EvidenceRefs, slots)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&campaignsv1.CreateCampaignResponse{Campaign: campaignMessage(campaign)}), nil
}

func (h handler) ActivateCampaign(ctx context.Context, request *connect.Request[campaignsv1.ActivateCampaignRequest]) (*connect.Response[campaignsv1.ActivateCampaignResponse], error) {
	if err := h.repo.Activate(ctx, request.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	campaigns, err := h.repo.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, campaign := range campaigns {
		if campaign.ID == request.Msg.Id {
			return connect.NewResponse(&campaignsv1.ActivateCampaignResponse{Campaign: campaignMessage(campaign)}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("campaign %q was not found after activation", request.Msg.Id))
}

func (h handler) GetLaunchAssets(ctx context.Context, request *connect.Request[campaignsv1.GetLaunchAssetsRequest]) (*connect.Response[campaignsv1.GetLaunchAssetsResponse], error) {
	if request.Msg.GetScenarioName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_name is required"))
	}
	slots, err := h.repo.LaunchAssets(ctx, request.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &campaignsv1.GetLaunchAssetsResponse{ScenarioName: request.Msg.GetScenarioName()}
	for _, slot := range slots {
		response.Slots = append(response.Slots, &campaignsv1.LaunchAssetSlot{CampaignId: slot.CampaignID, CampaignName: slot.CampaignName, Channel: slot.Channel, Format: slot.Format, Capacity: int32(slot.Capacity), Reserved: int32(slot.Reserved), DraftCount: int32(slot.DraftCount)})
	}
	return connect.NewResponse(response), nil
}

func campaignMessage(campaign internalcampaigns.Campaign) *campaignsv1.Campaign {
	return &campaignsv1.Campaign{Id: campaign.ID, Name: campaign.Name, Status: campaign.Status, ScenarioNames: campaign.ScenarioNames}
}

func Module(db *database.RoutedDB) module.Module {
	path, h := campaignsconnect.NewCampaignsServiceHandler(handler{repo: internalcampaigns.NewSQLiteRepository(db)})
	return module.Module{Name: "campaigns", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalcampaigns.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "campaigns_list", Path: campaignsconnect.CampaignsServiceListCampaignsProcedure, Method: "POST", Summary: "List campaigns", Category: "campaigns"},
	{ID: "campaigns_create", Path: campaignsconnect.CampaignsServiceCreateCampaignProcedure, Method: "POST", Summary: "Create proposed campaign with evidence and slot budget", Category: "campaigns"},
	{ID: "campaigns_activate", Path: campaignsconnect.CampaignsServiceActivateCampaignProcedure, Method: "POST", Summary: "Activate evidence-backed campaign", Category: "campaigns"},
	{ID: "launch_assets", Path: campaignsconnect.CampaignsServiceGetLaunchAssetsProcedure, Method: "POST", Summary: "Report launch assets and open slots for a scenario", Category: "campaigns"},
}
