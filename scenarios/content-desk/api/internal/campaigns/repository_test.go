package campaigns_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"content-desk/internal/campaigns"

	db "github.com/vrooli/api-core/databasetest"

	localdb "content-desk/internal/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func campaignRepoAndDB(t *testing.T) (*sql.DB, campaigns.Repository) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(campaigns.Schema)))
	// Launch-asset reporting joins the artifact-owned draft tables. The
	// production module mounts both schemas; this fixture supplies the narrow
	// cross-domain tables needed by the repository contract test.
	_, err := d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS drafts (id TEXT PRIMARY KEY, status TEXT NOT NULL); CREATE TABLE IF NOT EXISTS draft_slots (draft_id TEXT PRIMARY KEY, campaign_id TEXT NOT NULL, channel TEXT NOT NULL, format TEXT NOT NULL);`)
	require.NoError(t, err)
	return d, campaigns.NewSQLiteRepository(d)
}

func campaignRepo(t *testing.T) campaigns.Repository {
	t.Helper()
	_, repo := campaignRepoAndDB(t)
	return repo
}

// [REQ:CONTENTD-P0-001]
func TestCampaignActivationRequiresEvidence(t *testing.T) {
	t.Run("[CONTENTD-P0-001] campaign activation requires evidence", func(t *testing.T) {
		repo := campaignRepo(t)
		campaign, err := repo.Create(context.Background(), campaigns.Campaign{Name: "Launch"}, nil, nil)
		require.NoError(t, err)
		require.ErrorIs(t, repo.Activate(context.Background(), campaign.ID), campaigns.ErrEvidenceRequired)

		active, err := repo.Create(context.Background(), campaigns.Campaign{Name: "Evidence-backed", Status: campaigns.StatusActive}, []string{"research:audience-scan-1"}, nil)
		require.NoError(t, err)
		require.Equal(t, campaigns.StatusActive, active.Status)
	})
}

// [REQ:CONTENTD-P0-002]
func TestSlotBudgetIsHardCapAndReleaseReopensCapacity(t *testing.T) {
	t.Run("[CONTENTD-P0-002] artifact slots are a hard cap", func(t *testing.T) {
		repo := campaignRepo(t)
		campaign, err := repo.Create(context.Background(), campaigns.Campaign{Name: "Bounded"}, []string{"research:1"}, []campaigns.Slot{{Channel: "x-twitter", Format: "thread", Capacity: 2}})
		require.NoError(t, err)
		require.NoError(t, repo.ReserveSlot(context.Background(), campaign.ID, "x-twitter", "thread"))
		require.NoError(t, repo.ReserveSlot(context.Background(), campaign.ID, "x-twitter", "thread"))
		require.ErrorIs(t, repo.ReserveSlot(context.Background(), campaign.ID, "x-twitter", "thread"), campaigns.ErrSlotExhausted)
		require.NoError(t, repo.ReleaseSlot(context.Background(), campaign.ID, "x-twitter", "thread"))
		require.NoError(t, repo.ReserveSlot(context.Background(), campaign.ID, "x-twitter", "thread"))

		slots, err := repo.Slots(context.Background(), campaign.ID)
		require.NoError(t, err)
		require.Equal(t, []campaigns.Slot{{Channel: "x-twitter", Format: "thread", Capacity: 2, Reserved: 2}}, slots)
	})
}

func TestActiveCampaignWithoutEvidenceFailsAtCreate(t *testing.T) {
	repo := campaignRepo(t)
	_, err := repo.Create(context.Background(), campaigns.Campaign{Name: "Invalid", Status: campaigns.StatusActive}, nil, nil)
	require.True(t, errors.Is(err, campaigns.ErrEvidenceRequired))
}

func TestLaunchAssetsReportsScenarioSlotsAndDraftCounts(t *testing.T) {
	repo := campaignRepo(t)
	ctx := context.Background()
	campaign, err := repo.Create(ctx, campaigns.Campaign{Name: "Console launch", ScenarioNames: []string{"web-console"}}, []string{"research:launch"}, []campaigns.Slot{{Channel: "linkedin", Format: "post", Capacity: 2}})
	require.NoError(t, err)
	report, err := repo.LaunchAssets(ctx, "web-console")
	require.NoError(t, err)
	require.Len(t, report, 1)
	require.Equal(t, campaign.ID, report[0].CampaignID)
	require.Equal(t, 2, report[0].Capacity)
	require.Equal(t, 0, report[0].DraftCount)
	require.Equal(t, campaigns.LaunchAssetReadinessEmpty, report[0].Readiness)
	require.Empty(t, report[0].Reserved)
}

// [REQ:CONTENTD-P0-001]
func TestLaunchAssetsDistinguishesReadinessTiers(t *testing.T) {
	d, repo := campaignRepoAndDB(t)
	ctx := context.Background()
	createCampaign := func(name, channel, format string) string {
		campaign, err := repo.Create(ctx, campaigns.Campaign{Name: name, ScenarioNames: []string{"web-console"}}, []string{"research:launch"}, []campaigns.Slot{{Channel: channel, Format: format, Capacity: 2}})
		require.NoError(t, err)
		return campaign.ID
	}
	attachDraft := func(campaignID, channel, format, status string) {
		draftID := uuid.NewString()
		_, err := d.ExecContext(ctx, `INSERT INTO drafts (id, status) VALUES (?, ?)`, draftID, status)
		require.NoError(t, err)
		_, err = d.ExecContext(ctx, `INSERT INTO draft_slots (draft_id, campaign_id, channel, format) VALUES (?, ?, ?, ?)`, draftID, campaignID, channel, format)
		require.NoError(t, err)
	}

	approved := createCampaign("Approved", "linkedin", "post")
	attachDraft(approved, "linkedin", "post", "approved")
	attachDraft(approved, "linkedin", "post", "published")
	attachDraft(approved, "linkedin", "post", "drafted")
	attachDraft(approved, "linkedin", "post", "abandoned")

	review := createCampaign("Review", "x-twitter", "thread")
	attachDraft(review, "x-twitter", "thread", "drafted")

	inFlight := createCampaign("In flight", "blog", "post")
	attachDraft(inFlight, "blog", "post", "drafting")

	createCampaign("Empty", "youtube", "script")

	report, err := repo.LaunchAssets(ctx, "web-console")
	require.NoError(t, err)
	bySlot := map[string]campaigns.LaunchAssetSlot{}
	for _, slot := range report {
		bySlot[slot.Channel+"/"+slot.Format] = slot
	}

	require.Equal(t, campaigns.LaunchAssetReadinessApproved, bySlot["linkedin/post"].Readiness)
	require.Equal(t, 2, bySlot["linkedin/post"].ApprovedCount)
	require.Equal(t, 1, bySlot["linkedin/post"].ReadyForReviewCount)
	require.Equal(t, 0, bySlot["linkedin/post"].InProgressCount)
	require.Equal(t, 2, bySlot["linkedin/post"].DraftCount)

	require.Equal(t, campaigns.LaunchAssetReadinessReadyForReview, bySlot["x-twitter/thread"].Readiness)
	require.Equal(t, 0, bySlot["x-twitter/thread"].ApprovedCount)
	require.Equal(t, 1, bySlot["x-twitter/thread"].ReadyForReviewCount)

	require.Equal(t, campaigns.LaunchAssetReadinessInProgress, bySlot["blog/post"].Readiness)
	require.Equal(t, 1, bySlot["blog/post"].InProgressCount)

	require.Equal(t, campaigns.LaunchAssetReadinessEmpty, bySlot["youtube/script"].Readiness)
	require.Equal(t, 0, bySlot["youtube/script"].ApprovedCount+bySlot["youtube/script"].ReadyForReviewCount+bySlot["youtube/script"].InProgressCount)
}
