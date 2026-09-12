package campaigns

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	campaignsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/campaigns"
	campaignsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/campaigns/campaigns_v1connect"
)

const GroupName = "campaigns"

type handlers struct {
	client campaignsconnect.CampaignsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: campaignsconnect.NewCampaignsServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*campaignsv1.ListCampaignsResponse, error) {
	response, err := h.client.ListCampaigns(context.Background(), connect.NewRequest(&campaignsv1.ListCampaignsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list campaigns", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no campaigns response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *campaignsv1.ListCampaignsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Campaigns))
	for _, campaign := range message.Campaigns {
		results = append(results, fmt.Sprintf("%s — %s (%s)", campaign.Id, campaign.Name, campaign.Status))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d campaign(s).", len(message.Campaigns))}, ResultsHeading: "Campaigns", Results: results}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*campaignsv1.CreateCampaignResponse, error) {
	slotSpecs := splitCSV(ctx.Flag("slot"))
	slots := make([]*campaignsv1.CampaignSlot, 0, len(slotSpecs))
	for _, value := range slotSpecs {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("slot must be channel:format:capacity, got %q", value)
		}
		capacity, err := strconv.ParseInt(parts[2], 10, 32)
		if err != nil || capacity <= 0 {
			return nil, fmt.Errorf("slot capacity must be a positive integer, got %q", parts[2])
		}
		slots = append(slots, &campaignsv1.CampaignSlot{Channel: parts[0], Format: parts[1], Capacity: int32(capacity)})
	}
	response, err := h.client.CreateCampaign(context.Background(), connect.NewRequest(&campaignsv1.CreateCampaignRequest{Name: ctx.Flag("name"), EvidenceRefs: splitCSV(ctx.Flag("evidence-ref")), Slots: slots, ScenarioNames: splitCSV(ctx.Flag("scenario"))}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create campaign", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Campaign == nil {
		return nil, fmt.Errorf("server returned no campaign")
	}
	return response.Msg, nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	values := strings.Split(value, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}

func (h *handlers) createReport(_ cliapp.OperationContext, message *campaignsv1.CreateCampaignResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created campaign %s.", message.Campaign.Id)}, Changes: []string{fmt.Sprintf("status=%s", message.Campaign.Status)}}
}

func (h *handlers) activateCall(ctx cliapp.OperationContext) (*campaignsv1.ActivateCampaignResponse, error) {
	response, err := h.client.ActivateCampaign(context.Background(), connect.NewRequest(&campaignsv1.ActivateCampaignRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("activate campaign", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Campaign == nil {
		return nil, fmt.Errorf("server returned no activated campaign")
	}
	return response.Msg, nil
}

func (h *handlers) activateReport(_ cliapp.OperationContext, message *campaignsv1.ActivateCampaignResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Campaign %s activated.", message.Campaign.Id)}}
}

func (h *handlers) launchAssetsCall(ctx cliapp.OperationContext) (*campaignsv1.GetLaunchAssetsResponse, error) {
	response, err := h.client.GetLaunchAssets(context.Background(), connect.NewRequest(&campaignsv1.GetLaunchAssetsRequest{ScenarioName: ctx.Flag("scenario")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get launch assets", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no launch-assets response")
	}
	return response.Msg, nil
}

func (h *handlers) launchAssetsReport(_ cliapp.OperationContext, message *campaignsv1.GetLaunchAssetsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Slots))
	for _, slot := range message.Slots {
		results = append(results, fmt.Sprintf("%s / %s — %s:%s reserved=%d/%d drafts=%d", slot.CampaignName, slot.CampaignId, slot.Channel, slot.Format, slot.Reserved, slot.Capacity, slot.DraftCount))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Launch assets for %s: %d slot(s).", message.ScenarioName, len(results))}, ResultsHeading: "Launch assets", Results: results}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"CampaignsService.ListCampaigns":    cliapp.ProtoList(h.listCall, h.listReport),
		"CampaignsService.CreateCampaign":   cliapp.ProtoMutation(h.createCall, h.createReport),
		"CampaignsService.ActivateCampaign": cliapp.ProtoMutation(h.activateCall, h.activateReport),
		"CampaignsService.GetLaunchAssets":  cliapp.ProtoList(h.launchAssetsCall, h.launchAssetsReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("campaigns: load from manifest: %w", err)
	}
	return group, nil
}
