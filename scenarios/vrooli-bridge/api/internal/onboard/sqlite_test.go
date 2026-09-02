package onboard_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboarding"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "vrooli-bridge/internal/database"
)

func newSchemaDB(t *testing.T) (*sql.DB, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC))
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(onboard.Schema),
	))
	return d, clk
}

func TestSQLiteRepository_CreateGetRoundTrip(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := onboard.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	created, err := repo.Create(ctx, onboard.Op{
		Host:           "web-01",
		Port:           2222,
		User:           "deploy",
		NodeName:       "web-01",
		TargetRevision: "a1b2c3d",
		RepoURL:        "https://example.com/repo.git",
		SourceMode:     onboard.SourceModeWorkingTree,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, onboard.StatePending, created.State)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "web-01", got.Host)
	require.Equal(t, 2222, got.Port)
	require.Equal(t, "deploy", got.User)
	require.Equal(t, "a1b2c3d", got.TargetRevision)
	require.Equal(t, onboard.SourceModeWorkingTree, got.SourceMode)
	require.Equal(t, clk.Now().UTC(), got.CreatedAt)
}

func TestSQLiteRepository_UpdateLifecycle(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := onboard.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	created, err := repo.Create(ctx, onboard.Op{Host: "h"})
	require.NoError(t, err)

	created.State = onboard.StateFailed
	created.NodeID = "node-xyz"
	created.ExitCode = 1
	created.FailureReason = onboard.FailureBootstrap
	created.FailureDetail = "make[1]: *** [setup] Error 2\nvrooli setup failed"
	created.StartedAt = clk.Now().UTC()
	created.FinishedAt = clk.Now().UTC()
	updated, err := repo.Update(ctx, created)
	require.NoError(t, err)
	require.Equal(t, onboard.StateFailed, updated.State)
	require.Equal(t, "node-xyz", updated.NodeID)
	require.Equal(t, created.FailureDetail, updated.FailureDetail)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, onboard.StateFailed, got.State)
	require.Equal(t, "node-xyz", got.NodeID)
	require.Equal(t, created.FailureDetail, got.FailureDetail, "failure_detail must round-trip through the durable store")
	require.False(t, got.FinishedAt.IsZero())
}

func TestSQLiteRepository_ConfigurationDispositionsRoundTrip(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := onboard.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	op, err := repo.Create(ctx, onboard.Op{Host: "h", ConfigurationDispositions: []onboarding.Disposition{{ID: "scenario-x", Kind: "scenario", Name: "scenario-x", Disposition: "not_applicable", Reason: "disabled by profile", Remediation: "enable scenario-x"}}})
	require.NoError(t, err)

	got, err := repo.Get(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, []onboarding.Disposition{{ID: "scenario-x", Kind: "scenario", Name: "scenario-x", Disposition: "not_applicable", Reason: "disabled by profile", Remediation: "enable scenario-x"}}, got.ConfigurationDispositions)
}

func TestSQLiteRepository_EventsAppendOnlyOrderedDeduped(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := onboard.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	op, err := repo.Create(ctx, onboard.Op{Host: "h"})
	require.NoError(t, err)

	require.NoError(t, repo.AppendEvent(ctx, onboard.StepEvent{OpID: op.ID, Sequence: 2, StepID: "clone", Status: onboard.StepStatusOK, EmittedAt: clk.Now()}))
	require.NoError(t, repo.AppendEvent(ctx, onboard.StepEvent{OpID: op.ID, Sequence: 1, StepID: "detect-os", Status: onboard.StepStatusStarted, EmittedAt: clk.Now()}))
	// Replay of sequence 1 is ignored (idempotent at-least-once).
	require.NoError(t, repo.AppendEvent(ctx, onboard.StepEvent{OpID: op.ID, Sequence: 1, StepID: "detect-os", Status: onboard.StepStatusStarted, EmittedAt: clk.Now()}))

	events, err := repo.ListEvents(ctx, op.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, uint64(1), events[0].Sequence)
	require.Equal(t, "detect-os", events[0].StepID)
	require.Equal(t, uint64(2), events[1].Sequence)
}

func TestSQLiteRepository_ListNewestFirstAndNonTerminal(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(onboard.Schema),
	))
	ctx := context.Background()

	base := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	mk := func(id string, offset time.Duration, state onboard.State) {
		clk := scheduletest.New(base.Add(offset))
		repo := onboard.NewSQLiteRepository(d, clk)
		op, err := repo.Create(ctx, onboard.Op{ID: id, Host: "h"})
		require.NoError(t, err)
		op.State = state
		_, err = repo.Update(ctx, op)
		require.NoError(t, err)
	}
	mk("old", 0, onboard.StateSucceeded)
	mk("mid", time.Minute, onboard.StateBootstrapping)
	mk("new", 2*time.Minute, onboard.StateFailed)

	repo := onboard.NewSQLiteRepository(d, scheduletest.New(base))
	all, err := repo.List(ctx, onboard.ListFilter{})
	require.NoError(t, err)
	require.Equal(t, []string{"new", "mid", "old"}, []string{all[0].ID, all[1].ID, all[2].ID})

	nonTerminal, err := repo.ListNonTerminal(ctx)
	require.NoError(t, err)
	require.Len(t, nonTerminal, 1)
	require.Equal(t, "mid", nonTerminal[0].ID)
}

// TestSQLiteSchema_ReappliesOverExistingData proves the schema is a forward-only
// migration: applying it a second time over a DB that already holds rows is a
// no-op that preserves the data (migrate, never recreate).
func TestSQLiteSchema_ReappliesOverExistingData(t *testing.T) {
	d, clk := newSchemaDB(t)
	repo := onboard.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	created, err := repo.Create(ctx, onboard.Op{Host: "survives", TargetRevision: "rev1"})
	require.NoError(t, err)
	require.NoError(t, repo.AppendEvent(ctx, onboard.StepEvent{OpID: created.ID, Sequence: 1, StepID: "clone", Status: onboard.StepStatusOK, EmittedAt: clk.Now()}))

	// Re-apply the schema over the populated DB (what a restart does).
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(onboard.Schema),
	))

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "survives", got.Host)
	events, err := repo.ListEvents(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
}
