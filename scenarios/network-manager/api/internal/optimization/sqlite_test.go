package optimization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"network-manager/internal/adapters"

	db "github.com/vrooli/api-core/databasetest"
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

func TestSQLiteRepositoryAllowsRepeatedOptimizationRuns(t *testing.T) {
	// [REQ:NM-P0-005] Candidate IDs are scoped to each run so repeated live optimization runs do not collide.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)
	store := newSnapshotStore(baselineSnapshot())
	svc := NewService(Config{
		Repo:         repo,
		Capabilities: fakeCapabilities{caps: []adapters.Capability{capability("read_network_status", true, true)}},
		Snapshots:    store,
		Runner:       store,
		Applier:      fakeApplier{},
		Now:          fixedNow,
	})

	first, err := svc.CreateRun(context.Background(), "", false)
	require.NoError(t, err)
	second, err := svc.CreateRun(context.Background(), "", false)
	require.NoError(t, err)

	require.NotEqual(t, first.ID, second.ID)
	require.Len(t, first.Candidates, 1)
	require.Len(t, second.Candidates, 1)
	require.NotEqual(t, first.Candidates[0].ID, second.Candidates[0].ID)
	require.Contains(t, first.Candidates[0].ID, "read-only-baseline-compare")
	require.Contains(t, second.Candidates[0].ID, "read-only-baseline-compare")
}
