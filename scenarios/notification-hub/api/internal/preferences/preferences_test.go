package preferences_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"

	"notification-hub/internal/preferences"
)

func newStore(t *testing.T) *preferences.Storage {
	t.Helper()
	db := databasetest.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), preferences.Schema())
	require.NoError(t, err)
	return preferences.New(db)
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func TestUrgencyAtLeast(t *testing.T) {
	cases := []struct {
		have, min string
		want      bool
	}{
		{"low", "", true}, // empty min always passes
		{"", "low", true}, // empty have == normal >= low
		{"low", "normal", false},
		{"normal", "normal", true},
		{"high", "normal", true},
		{"critical", "high", true},
		{"normal", "critical", false},
		{"low", "critical", false},
		{"garbage", "high", false}, // unknown ranks as normal
	}
	for _, c := range cases {
		require.Equalf(t, c.want, preferences.UrgencyAtLeast(c.have, c.min), "UrgencyAtLeast(%q,%q)", c.have, c.min)
	}
}

func TestObserveCatalogIsIdempotentAndKeepsDeclaredPrecedence(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	require.NoError(t, store.ObserveCatalog(ctx, "planner", "reminder", now()))
	require.NoError(t, store.ObserveCatalog(ctx, "planner", "reminder", now()))

	// Unfiltered listing (empty repo root disables the installed-scenario
	// filter) so this test exercises observe/declare mechanics, not catalog
	// filtering, which has its own coverage in manifest_test.go.
	sources, err := store.ListCatalogForRepo(ctx, "")
	require.NoError(t, err)
	require.Len(t, sources, 1)
	require.Equal(t, "planner", sources[0].SourceScenario)
	require.False(t, sources[0].Declared, "observed source is not declared")
	require.Len(t, sources[0].Categories, 1, "re-observation must not duplicate the category")
	require.False(t, sources[0].Categories[0].Declared)

	// A declared source upgrades the baseline branding and declared flag.
	require.NoError(t, store.UpsertDeclaredSource(ctx, "planner", "Personal Planner", "calendar", "/planner", now()))
	displayName, icon, deepLink, ok := store.ResolveBrand(ctx, "planner")
	require.True(t, ok)
	require.Equal(t, "Personal Planner", displayName)
	require.Equal(t, "calendar", icon)
	require.Equal(t, "/planner", deepLink)

	// A later observation must not clobber the declared branding.
	require.NoError(t, store.ObserveCatalog(ctx, "planner", "reminder", now()))
	displayName, _, _, ok = store.ResolveBrand(ctx, "planner")
	require.True(t, ok)
	require.Equal(t, "Personal Planner", displayName)

	_, _, _, ok = store.ResolveBrand(ctx, "unknown")
	require.False(t, ok)
}

func TestGetPreferenceSpecificityResolution(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	// No preference, no category default: source default is enabled true.
	eff, err := store.GetPreference(ctx, "alice", "planner", "reminder", "")
	require.NoError(t, err)
	require.True(t, eff.Enabled)
	require.Equal(t, "", eff.MinUrgency)

	// Category default from a declared category is consulted (tier 3).
	require.NoError(t, store.UpsertDeclaredCategory(ctx, "planner", "reminder", "Reminders", "", false, "high", now()))
	eff, err = store.GetPreference(ctx, "alice", "planner", "reminder", "")
	require.NoError(t, err)
	require.False(t, eff.Enabled, "category default_enabled=false applies")
	require.Equal(t, "high", eff.MinUrgency)

	// A source-wide preference (tier 2) overrides the category default.
	require.NoError(t, store.SetPreference(ctx, "alice", "planner", "", true, "normal", "", "", now()))
	eff, err = store.GetPreference(ctx, "alice", "planner", "reminder", "")
	require.NoError(t, err)
	require.True(t, eff.Enabled)
	require.Equal(t, "normal", eff.MinUrgency)

	// An exact category preference (tier 1) is the most specific.
	require.NoError(t, store.SetPreference(ctx, "alice", "planner", "reminder", false, "", "", "", now()))
	eff, err = store.GetPreference(ctx, "alice", "planner", "reminder", "")
	require.NoError(t, err)
	require.False(t, eff.Enabled, "muted at the category level")

	// Another recipient is unaffected by alice's preferences.
	eff, err = store.GetPreference(ctx, "bob", "planner", "reminder", "")
	require.NoError(t, err)
	require.False(t, eff.Enabled, "bob still sees the declared category default")
	require.Equal(t, "high", eff.MinUrgency)
}

func TestListPreferences(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	require.NoError(t, store.SetPreference(ctx, "alice", "planner", "reminder", false, "high", "email", "", now()))
	require.NoError(t, store.SetPreference(ctx, "alice", "chat", "", true, "", "", "", now()))
	require.NoError(t, store.SetPreference(ctx, "bob", "planner", "", true, "", "", "", now()))

	prefs, err := store.ListPreferences(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, prefs, 2)
	require.Equal(t, "chat", prefs[0].SourceScenario)
	require.Equal(t, "planner", prefs[1].SourceScenario)
	require.False(t, prefs[1].Enabled)
	require.Equal(t, "high", prefs[1].MinUrgency)
	require.Equal(t, "email", prefs[1].ChannelOverride)
}
