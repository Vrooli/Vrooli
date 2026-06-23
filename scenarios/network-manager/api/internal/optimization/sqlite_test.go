package optimization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"network-manager/internal/testutil/db"
)

func TestSQLiteRepositoryPersistsRunAndCandidates(t *testing.T) {
	// [REQ:NM-P0-005] Optimization runs and candidates are persisted in the domain-owned ledger.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema), apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)

	run := Run{ID: "run-1", Status: "draft", ScoringProfile: "reliability", BaselineSnapshotID: "baseline-1", Recommendation: "run candidates", CreatedAt: fixedNow(), UpdatedAt: fixedNow()}
	_, err := repo.SaveRun(context.Background(), run)
	require.NoError(t, err)
	_, err = repo.SaveCandidate(context.Background(), Candidate{ID: "candidate-1", RunID: run.ID, Description: "test candidate", Status: "not_run", Evidence: []string{"baseline baseline-1"}, ApprovalRequired: true, RollbackSupported: true, BaselineSnapshotID: "baseline-1", CreatedAt: fixedNow(), UpdatedAt: fixedNow()})
	require.NoError(t, err)

	stored, err := repo.GetRun(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, "draft", stored.Status)
	require.Len(t, stored.Candidates, 1)
	require.Equal(t, "candidate-1", stored.Candidates[0].ID)
	require.True(t, stored.Candidates[0].ApprovalRequired)
}
