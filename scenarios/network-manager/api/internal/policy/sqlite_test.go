package policy

import (
	"context"
	"testing"

	"network-manager/internal/testutil/db"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestSQLiteRepositoryStoresPolicyChangeApprovalAndRollback(t *testing.T) {
	// [REQ:NM-P0-003] Policy plans, approval records, and rollback records use domain-owned storage.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)

	change, err := repo.SaveChange(context.Background(), Change{
		ID:                "change-1",
		Target:            "device:phone",
		Action:            "blocklist",
		Status:            "previewed",
		Values:            []string{"ads.example"},
		Effects:           []string{"preview ok"},
		RollbackSupported: true,
	})
	require.NoError(t, err)
	require.NoError(t, err)
	require.NotZero(t, change.CreatedAt)

	_, err = repo.SaveApproval(context.Background(), ApprovalRecord{ID: "approval-1", ChangeID: "change-1", Approved: true, Note: "operator approved"})
	require.NoError(t, err)
	_, err = repo.SaveRollback(context.Background(), RollbackRecord{ID: "rollback-1", ChangeID: "change-1", Status: "rolled_back", Details: []string{"restored"}})
	require.NoError(t, err)

	stored, err := repo.GetChange(context.Background(), "change-1")
	require.NoError(t, err)
	require.Equal(t, []string{"ads.example"}, stored.Values)
	require.Equal(t, []string{"preview ok"}, stored.Effects)
	require.True(t, stored.RollbackSupported)

	stored.Status = "applied"
	stored.RollbackHandle = "rollback://policy/1"
	updated, err := repo.UpdateChange(context.Background(), stored)
	require.NoError(t, err)
	require.Equal(t, "applied", updated.Status)
	require.Equal(t, "rollback://policy/1", updated.RollbackHandle)
}
