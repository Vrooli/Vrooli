package policy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/proto"
)

func newPolicyStore(t *testing.T) Store {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)))
	return NewSQLiteStore(db)
}

func TestLoadOrInitializePersistsSafeOffDefaults(t *testing.T) {
	ctx := context.Background()
	store := newPolicyStore(t)
	config, state, err := LoadOrInitialize(ctx, store)
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_OFF, config.GetDefaultPreset())
	require.Equal(t, State{}, state)

	loaded, loadedState, err := store.Load(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(config, loaded))
	require.Equal(t, state, loadedState)
}

func TestLoadOrInitializeAddsHealthAndActivityDefaultsToOlderPolicy(t *testing.T) {
	ctx := context.Background()
	store := newPolicyStore(t)
	older := DefaultConfig()
	older.HealthPalette = nil
	older.ActivityEffects = nil
	require.NoError(t, store.Save(ctx, older, State{ManualPreset: policyv1.PresetKind_PRESET_KIND_OFF}))

	loaded, state, err := LoadOrInitialize(ctx, store)
	require.NoError(t, err)
	require.NotNil(t, loaded.HealthPalette)
	require.NotNil(t, loaded.ActivityEffects)
	require.Equal(t, uint32(5), loaded.ActivityEffects.BusyAgentThreshold)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_OFF, state.ManualPreset)
}

func TestStoreRoundTripsPolicyAndReplayStateAcrossRestart(t *testing.T) {
	ctx := context.Background()
	store := newPolicyStore(t)
	config := DefaultConfig()
	state := State{
		ActivePreset:           policyv1.PresetKind_PRESET_KIND_LOAD,
		LoadActive:             true,
		Candidate:              policyv1.PresetKind_PRESET_KIND_WORK,
		CandidateAt:            time.Date(2026, 11, 10, 17, 0, 0, 0, time.UTC),
		ManualPreset:           policyv1.PresetKind_PRESET_KIND_OFF,
		ManualExpiry:           time.Date(2026, 11, 10, 18, 0, 0, 0, time.UTC),
		ManualRequestedAt:      time.Date(2026, 11, 10, 17, 30, 0, 0, time.UTC),
		ManualSelectionPending: true,
	}
	require.NoError(t, store.Save(ctx, config, state))

	loadedConfig, loadedState, err := store.Load(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(config, loadedConfig))
	require.Equal(t, state, loadedState)
	config.Presets[1].Color.Red = 1
	require.False(t, proto.Equal(config, loadedConfig), "loaded configuration must not alias the caller's mutable config")
}

func TestStoreRejectsInvalidPolicyConfig(t *testing.T) {
	store := newPolicyStore(t)
	config := DefaultConfig()
	config.LoadExitPercent = 100
	config.LoadEnterPercent = 50
	require.Error(t, store.Save(context.Background(), config, State{}))
	_, _, err := store.Load(context.Background())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestStoreUsesInjectedClockForUpdatedAt(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	fixed := time.Date(2026, 11, 10, 23, 59, 0, 0, time.UTC)
	store := &sqliteStore{db: db, now: func() time.Time { return fixed }}
	require.NoError(t, store.Save(ctx, DefaultConfig(), State{}))

	var updatedAt string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT updated_at FROM lighting_policy_state WHERE id = 'default'`).Scan(&updatedAt))
	require.Equal(t, fixed.Format(time.RFC3339Nano), updatedAt)
}
