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

	// UpsertFinding inserts or updates a tracked finding keyed by
	// (campaign_id, stable_id). first_seen_at is preserved on update.
	UpsertFinding(ctx context.Context, campaignID string, f Finding) error
	GetFinding(ctx context.Context, campaignID, stableID string) (Finding, error)
	ListFindings(ctx context.Context, campaignID string) ([]Finding, error)
}
