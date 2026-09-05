package campaign

import (
	"architecture-cartographer/internal/campaign"

	campaignv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// statusToProto maps the domain FindingStatus to the wire enum.
func statusToProto(s campaign.FindingStatus) campaignv1.CampaignItemStatus {
	switch s {
	case campaign.StatusDetected:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_DETECTED
	case campaign.StatusAssigned:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_ASSIGNED
	case campaign.StatusSplit:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_SPLIT
	case campaign.StatusResolved:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_RESOLVED
	case campaign.StatusValidated:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_VALIDATED
	case campaign.StatusCommitted:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_COMMITTED
	case campaign.StatusForceResolved:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_FORCE_RESOLVED
	default:
		return campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_UNSPECIFIED
	}
}

func lifecycleToProto(s campaign.CampaignLifecycle) campaignv1.CampaignLifecycle {
	switch s {
	case campaign.CampaignOpen:
		return campaignv1.CampaignLifecycle_CAMPAIGN_LIFECYCLE_OPEN
	case campaign.CampaignClosed:
		return campaignv1.CampaignLifecycle_CAMPAIGN_LIFECYCLE_CLOSED
	default:
		return campaignv1.CampaignLifecycle_CAMPAIGN_LIFECYCLE_UNSPECIFIED
	}
}

func findingToProto(f campaign.Finding) *campaignv1.CampaignItem {
	return &campaignv1.CampaignItem{
		StableId:       f.StableID,
		Scenario:       f.Scenario,
		Source:         f.Source,
		Code:           f.Code,
		Severity:       f.Severity,
		Locations:      f.Locations,
		Domains:        f.Domains,
		Message:        f.Message,
		Suggestion:     f.Suggestion,
		Status:         statusToProto(f.Status),
		ResolutionNote: f.ResolutionNote,
		Regressed:      f.Regressed,
		Effort:         f.Effort,
		FirstSeenAt:    timestamppb.New(f.FirstSeenAt),
		UpdatedAt:      timestamppb.New(f.UpdatedAt),
	}
}

func findingsToProto(in []campaign.Finding) []*campaignv1.CampaignItem {
	out := make([]*campaignv1.CampaignItem, 0, len(in))
	for _, f := range in {
		out = append(out, findingToProto(f))
	}
	return out
}

func campaignToProto(c campaign.Campaign) *campaignv1.Campaign {
	return &campaignv1.Campaign{
		Id:        c.ID,
		Scenario:  c.Scenario,
		Name:      c.Name,
		Status:    lifecycleToProto(c.Status),
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func statusProjectionToProto(st campaign.Status) *campaignv1.CampaignStatus {
	return &campaignv1.CampaignStatus{
		Campaign:    campaignToProto(st.Campaign),
		Items:       findingsToProto(st.Findings),
		Total:       int32(st.Total),
		Open:        int32(st.Open),
		Resolved:    int32(st.Resolved),
		Validated:   int32(st.Validated),
		Regressions: int32(st.Regressions),
		BySeverity:  intMap(st.BySeverity),
		ByStatus:    intMap(st.ByStatus),
	}
}

func intMap(in map[string]int) map[string]int32 {
	out := make(map[string]int32, len(in))
	for k, v := range in {
		out[k] = int32(v)
	}
	return out
}
