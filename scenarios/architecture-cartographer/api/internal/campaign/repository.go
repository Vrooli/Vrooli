package campaign

import "context"

// Repository is the persistence seam for the campaign tracker. Production
// wires the SQLite implementation; tests substitute a fake or a real
// in-temp-dir SQLite handle.
type Repository interface {
	CreateCampaign(ctx context.Context, c Campaign) error
	GetCampaign(ctx context.Context, id string) (Campaign, error)
	// ListCampaigns returns campaign headers newest-first. An empty
	// scenario returns every campaign.
	ListCampaigns(ctx context.Context, scenario string) ([]Campaign, error)
	UpdateCampaignStatus(ctx context.Context, id string, status CampaignLifecycle) error

	// CreateFinding inserts a new lifecycle record; existing rows are immutable here.
	CreateFinding(ctx context.Context, campaignID string, f Finding) error
	// TransitionFinding updates finding metadata and status only if the persisted
	// state still equals expected and the transition is allowed.
	TransitionFinding(ctx context.Context, campaignID string, expected FindingStatus, f Finding) error
	GetFinding(ctx context.Context, campaignID, stableID string) (Finding, error)
	ListFindings(ctx context.Context, campaignID string) ([]Finding, error)
}
