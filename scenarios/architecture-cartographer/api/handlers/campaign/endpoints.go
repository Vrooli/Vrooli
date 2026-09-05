package campaign

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign/campaign_v1connect"
)

// Endpoints describes the campaign domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "campaign.create",
		Path:        campaign_v1connect.CampaignServiceCreateCampaignProcedure,
		Method:      "POST",
		Summary:     "Open a campaign and ingest findings",
		Description: "Opens a tracked scenario-improvement campaign and ingests the initial ArchitectureFinding set (all start detected).",
		Category:    "campaign",
	},
	{
		ID:          "campaign.list",
		Path:        campaign_v1connect.CampaignServiceListCampaignsProcedure,
		Method:      "POST",
		Summary:     "List a scenario's campaigns",
		Description: "Returns the campaign headers for a scenario (newest first), or every campaign when scenario is empty.",
		Category:    "campaign",
	},
	{
		ID:          "campaign.status",
		Path:        campaign_v1connect.CampaignServiceGetCampaignStatusProcedure,
		Method:      "POST",
		Summary:     "Get campaign status",
		Description: "Returns the campaign plus every tracked item and rollup counts.",
		Category:    "campaign",
	},
	{
		ID:          "campaign.next",
		Path:        campaign_v1connect.CampaignServiceNextCampaignStepProcedure,
		Method:      "POST",
		Summary:     "Get the next worklist chunk",
		Description: "Returns the profile-ranked worklist of open items (fast / balanced / long-term).",
		Category:    "campaign",
	},
	{
		ID:          "campaign.resolve",
		Path:        campaign_v1connect.CampaignServiceResolveItemProcedure,
		Method:      "POST",
		Summary:     "Mark an item resolved",
		Description: "Records that the agent fixed an item by hand, with an operator note.",
		Category:    "campaign",
	},
	{
		ID:          "campaign.apply",
		Path:        campaign_v1connect.CampaignServiceApplyItemProcedure,
		Method:      "POST",
		Summary:     "Apply an item fix (status-only)",
		Description: "Records a hand-fix as a status-only transition. Auto-execution of file-op fixes stays deferred to the apply-execution plan.",
		Category:    "campaign",
	},
	{
		ID:          "campaign.reaudit",
		Path:        campaign_v1connect.CampaignServiceReauditCampaignProcedure,
		Method:      "POST",
		Summary:     "Reconcile a re-audit",
		Description: "Reconciles a fresh findings set against the tracked set by stable id: absent→validated, present→stay, (re)appeared→regression.",
		Category:    "campaign",
	},
	{
		ID:          "campaign.close",
		Path:        campaign_v1connect.CampaignServiceCloseCampaignProcedure,
		Method:      "POST",
		Summary:     "Close a campaign",
		Description: "Marks the campaign closed.",
		Category:    "campaign",
	},
}
