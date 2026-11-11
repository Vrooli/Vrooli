package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"lighting-control/internal/control"
	"lighting-control/internal/policy"
)

func TestFreshRoutedLeaseSupportsRealLightingRepositoriesWithoutPrimaryWrites(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(dir, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, initializeDatabase(ctx, db))
	require.NoError(t, db.InstallTestPool(ctx, filepath.Join(dir, "lease.db"), "lighting-browser", 0))
	t.Cleanup(func() { require.NoError(t, db.ClearTestPool("lighting-browser")) })
	testCtx := database.WithTestMode(ctx)
	repository := control.NewSQLiteRepository(db, nil)
	states, err := repository.ListDesired(testCtx)
	require.NoError(t, err, "fresh routed control schema must exist before rendering Presets")
	require.Empty(t, states)
	store := policy.NewSQLiteStore(db)
	_, _, err = policy.LoadOrInitialize(testCtx, store)
	require.NoError(t, err, "fresh routed policy schema must support default initialization")
	require.NoError(t, repository.SavePolicyDesired(testCtx, &controlv1.ZoneState{ControllerId: "fixture-controller", ZoneId: "fixture-zone"}, nil))
	states, err = repository.ListDesired(testCtx)
	require.NoError(t, err)
	require.Len(t, states, 1)
	states, err = repository.ListDesired(ctx)
	require.NoError(t, err)
	require.Empty(t, states, "leased desired state must never reach primary")
	_, _, err = store.Load(ctx)
	require.ErrorIs(t, err, policy.ErrNotFound, "leased default policy must never reach primary")
}
