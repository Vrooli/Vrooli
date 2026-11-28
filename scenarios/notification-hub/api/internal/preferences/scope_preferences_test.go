package preferences_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// A per-instance scope override is the most-specific tier: it wins over the
// recipient's global category and source-wide preferences, in both directions.
func TestGetPreferenceScopeOverride(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	const (
		rk      = "alice"
		src     = "web-console"
		cat     = "session.turn_completed"
		session = "sess-123"
	)

	// Global: alice has muted the category entirely.
	require.NoError(t, store.SetPreference(ctx, rk, src, cat, false, "", "", "", now()))

	// Without a scope key, the global mute applies.
	eff, err := store.GetPreference(ctx, rk, src, cat, "")
	require.NoError(t, err)
	require.False(t, eff.Enabled)

	// A different session with no scope override still inherits the global mute.
	eff, err = store.GetPreference(ctx, rk, src, cat, "sess-other")
	require.NoError(t, err)
	require.False(t, eff.Enabled, "an unrelated scope falls through to the global preference")

	// Force this one session ON — scoped-on beats global-off.
	require.NoError(t, store.SetPreference(ctx, rk, src, cat, true, "", "", session, now()))
	eff, err = store.GetPreference(ctx, rk, src, cat, session)
	require.NoError(t, err)
	require.True(t, eff.Enabled, "scoped-on overrides the global mute for this session")

	// The global preference is untouched for other sessions.
	eff, err = store.GetPreference(ctx, rk, src, cat, "sess-other")
	require.NoError(t, err)
	require.False(t, eff.Enabled)
}

// Scoped-off beats a global-on, and a scoped min-urgency is honored.
func TestGetPreferenceScopeMuteAndUrgency(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	const (
		rk      = "bob"
		src     = "web-console"
		cat     = "session.input_needed"
		session = "sess-abc"
	)

	// Global default (no row) is enabled. Mute just this session.
	require.NoError(t, store.SetPreference(ctx, rk, src, cat, false, "", "", session, now()))
	eff, err := store.GetPreference(ctx, rk, src, cat, session)
	require.NoError(t, err)
	require.False(t, eff.Enabled, "scoped-off overrides the enabled default for this session")

	// A source-wide scope override (empty category) applies when no scoped
	// category row exists.
	require.NoError(t, store.SetPreference(ctx, rk, src, "", true, "high", "", session, now()))
	eff, err = store.GetPreference(ctx, rk, src, "session.turn_completed", session)
	require.NoError(t, err)
	require.True(t, eff.Enabled)
	require.Equal(t, "high", eff.MinUrgency, "scoped source-wide min-urgency applies to a category with no scoped row")

	// The scoped category row stays more specific than the scoped source-wide row.
	eff, err = store.GetPreference(ctx, rk, src, cat, session)
	require.NoError(t, err)
	require.False(t, eff.Enabled, "scoped category row still wins over the scoped source-wide row")
}

// ListPreferences returns global rows and scope rows, with ScopeKey populated
// only on the latter.
func TestListPreferencesIncludesScopeRows(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	require.NoError(t, store.SetPreference(ctx, "carol", "web-console", "session.message", false, "", "", "", now()))
	require.NoError(t, store.SetPreference(ctx, "carol", "web-console", "session.message", true, "", "", "sess-9", now()))

	prefs, err := store.ListPreferences(ctx, "carol")
	require.NoError(t, err)
	require.Len(t, prefs, 2)

	var global, scoped int
	for _, p := range prefs {
		if p.ScopeKey == "" {
			global++
			require.False(t, p.Enabled)
		} else {
			scoped++
			require.Equal(t, "sess-9", p.ScopeKey)
			require.True(t, p.Enabled)
		}
	}
	require.Equal(t, 1, global)
	require.Equal(t, 1, scoped)
}
