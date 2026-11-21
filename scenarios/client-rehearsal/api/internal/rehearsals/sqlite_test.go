package rehearsals_test

import (
	"context"
	"errors"
	"testing"
	"time"

	localdb "client-rehearsal/internal/database"
	internal "client-rehearsal/internal/rehearsals"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	databasetest "github.com/vrooli/api-core/databasetest"
)

func newRepository(t *testing.T) internal.Repository {
	t.Helper()
	db := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internal.Schema)))
	return internal.NewSQLiteRepository(db)
}

func TestSQLiteRepositoryRoundTripAndExplicitReview(t *testing.T) {
	repo := newRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, internal.Rehearsal{ID: "run-1", Title: "Synthetic intake", SourceFormat: "n8n-workflow-json/v1", SourceSHA256: "workflow-hash", CredentialReplacements: 2, SimulatorProfile: "openapi", SimulatorSourceSHA256: "simulator-hash", State: internal.StateNeedsReview, Trace: []internal.TraceEvent{{Sequence: 1, Operation: "POST /contacts", Status: "succeeded", OccurredAt: now}}, FaultSummary: "injected fault", RepairSummary: "retry passed", HandoffMarkdown: "review me", CreatedAt: now})
	require.NoError(t, err)
	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "workflow-hash", got.SourceSHA256)
	require.Equal(t, "simulator-hash", got.SimulatorSourceSHA256)
	require.Len(t, got.Trace, 1)
	require.Nil(t, got.ReviewedAt)
	list, err := repo.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	reviewed, err := repo.RecordReview(ctx, "run-1", internal.DecisionApprove, "Evidence reviewed", now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, internal.StateApproved, reviewed.State)
	require.NotNil(t, reviewed.ReviewedAt)
	_, err = repo.RecordReview(ctx, "run-1", internal.DecisionRequestChanges, "second choice", now.Add(2*time.Minute))
	var conflict internal.ErrConflict
	require.True(t, errors.As(err, &conflict), "second review must not overwrite the first: %v", err)
}
