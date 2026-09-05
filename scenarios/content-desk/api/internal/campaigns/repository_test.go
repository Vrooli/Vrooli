package campaigns_test

import (
	"context"
	"errors"
	"testing"

	"content-desk/internal/campaigns"

	db "github.com/vrooli/api-core/databasetest"

	localdb "content-desk/internal/database"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func campaignRepo(t *testing.T) campaigns.Repository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(campaigns.Schema)))
	// Launch-asset reporting joins the artifact-owned draft tables. The
	// production module mounts both schemas; this fixture supplies the narrow
	// cross-domain tables needed by the repository contract test.
	_, err := d.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS drafts (id TEXT PRIMARY KEY, status TEXT NOT NULL); CREATE TABLE IF NOT EXISTS draft_slots (draft_id TEXT PRIMARY KEY, campaign_id TEXT NOT NULL, channel TEXT NOT NULL, format TEXT NOT NULL);`)
	require.NoError(t, err)
	return campaigns.NewSQLiteRepository(d)
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
	require.Empty(t, report[0].Reserved)
}
