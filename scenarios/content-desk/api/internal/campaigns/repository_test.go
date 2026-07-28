package campaigns_test

import (
	"context"
	"errors"
	"testing"

	"content-desk/internal/campaigns"
	"content-desk/internal/testutil/db"

	localdb "content-desk/internal/database"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func campaignRepo(t *testing.T) campaigns.Repository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(campaigns.Schema)))
	return campaigns.NewSQLiteRepository(d)
}

func TestCampaignActivationRequiresEvidence(t *testing.T) {
	repo := campaignRepo(t)
	campaign, err := repo.Create(context.Background(), campaigns.Campaign{Name: "Launch"}, nil, nil)
	require.NoError(t, err)
	require.ErrorIs(t, repo.Activate(context.Background(), campaign.ID), campaigns.ErrEvidenceRequired)

	active, err := repo.Create(context.Background(), campaigns.Campaign{Name: "Evidence-backed", Status: campaigns.StatusActive}, []string{"research:audience-scan-1"}, nil)
	require.NoError(t, err)
	require.Equal(t, campaigns.StatusActive, active.Status)
}

func TestSlotBudgetIsHardCapAndReleaseReopensCapacity(t *testing.T) {
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
}

func TestActiveCampaignWithoutEvidenceFailsAtCreate(t *testing.T) {
	repo := campaignRepo(t)
	_, err := repo.Create(context.Background(), campaigns.Campaign{Name: "Invalid", Status: campaigns.StatusActive}, nil, nil)
	require.True(t, errors.Is(err, campaigns.ErrEvidenceRequired))
}
