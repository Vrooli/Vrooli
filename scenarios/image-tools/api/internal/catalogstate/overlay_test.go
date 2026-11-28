package catalogstate_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"image-tools/internal/adapters"
	"image-tools/internal/catalogstate"
	"image-tools/internal/models"
)

var t0 = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

func newModelOverlay(t *testing.T) (*catalogstate.Overlay, *sql.DB) {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(models.Schema)); err != nil {
		t.Fatal(err)
	}
	return catalogstate.NewOverlay(d, catalogstate.Models), d
}

func seed(flags map[string]bool) catalogstate.SeedState {
	return func(id string) (bool, bool) { v, ok := flags[id]; return v, ok }
}

func set(t *testing.T, st *catalogstate.Overlay, id string, enabled, pinned bool, seedNow bool) {
	t.Helper()
	o := catalogstate.Override{ID: id, Enabled: enabled, SetBy: "agent-a", Reason: "because", Pinned: pinned}
	if err := st.SetOverride(context.Background(), o, seedNow, t0); err != nil {
		t.Fatal(err)
	}
}

// TestOverrideRecordsProvenance proves who/when/why/pinned and the seed baseline
// are stored with an override, and that a missing author is refused.
func TestOverrideRecordsProvenance(t *testing.T) {
	ctx := context.Background()
	st, _ := newModelOverlay(t)
	set(t, st, "rf", false, true, true)
	o, ok, err := st.GetOverride(ctx, "rf")
	if err != nil || !ok {
		t.Fatalf("get: %v %v", ok, err)
	}
	if o.SetBy != "agent-a" || o.Reason != "because" || !o.Pinned || !o.SetAt.Equal(t0) {
		t.Fatalf("provenance = %+v", o)
	}
	if o.SeedEnabled == nil || !*o.SeedEnabled {
		t.Fatalf("seed baseline = %v, want true", o.SeedEnabled)
	}
	if err := st.SetOverride(ctx, catalogstate.Override{ID: "x", Enabled: true}, true, t0); err == nil || !strings.Contains(err.Error(), "who set it") {
		t.Fatalf("an override with no author must be refused, got %v", err)
	}
	if err := st.SetOverride(ctx, catalogstate.Override{SetBy: "a"}, true, t0); err == nil {
		t.Fatal("an override with no id must be refused")
	}
}

// TestSeedChangeSupersedesAnUnpinnedOverride is the bug from 2026-11-21: an
// agent disabled RestoreFormer++ on 2026-11-10, the seed later enabled it, and
// the stale override kept it off with no trace. The rule: a seed change newer
// than the override wins unless the override is pinned.
func TestSeedChangeSupersedesAnUnpinnedOverride(t *testing.T) {
	ctx := context.Background()
	st, _ := newModelOverlay(t)
	set(t, st, "restoreformer", false, false, false) // seed said disabled; agent also disabled it
	set(t, st, "pinned-model", true, true, true)     // deliberate standing choice, enabled over a seed that said true
	set(t, st, "unchanged", false, false, true)      // seed has not moved

	// A new binary ships a seed that enables restoreformer (and still enables the others).
	// The pinned model's seed changed too (it was true when pinned to false, and
	// now says false): the pin must win regardless, so it is not superseded.
	flags := map[string]bool{"restoreformer": true, "pinned-model": false, "unchanged": true}
	now := t0.Add(11 * 24 * time.Hour)
	changes, err := st.Reconcile(ctx, seed(flags), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].ID != "restoreformer" || changes[0].Action != "superseded" {
		t.Fatalf("changes = %+v", changes)
	}
	for _, want := range []string{"changed false to true", "agent-a", "2026-11-10"} {
		if !strings.Contains(changes[0].Detail, want) {
			t.Fatalf("detail %q lacks %q", changes[0].Detail, want)
		}
	}

	overlay, err := st.LoadOverlay(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, applies := overlay["restoreformer"]; applies {
		t.Fatal("the superseded override must no longer apply")
	}
	if v, ok := overlay["pinned-model"]; !ok || !v {
		t.Fatalf("a pinned override must keep applying, got %v present=%t", v, ok)
	}
	if v, ok := overlay["unchanged"]; !ok || v {
		t.Fatalf("an override over an unchanged seed must keep winning, got %v present=%t", v, ok)
	}
	all, _ := st.LoadOverrides(ctx)
	if o := all["restoreformer"]; o.Active() || o.SupersededNote == "" || !o.SupersededAt.Equal(now) {
		t.Fatalf("a superseded override is kept for the record: %+v", o)
	}

	// Idempotent, and a later seed flip must not resurrect the superseded override.
	if changes, err = st.Reconcile(ctx, seed(map[string]bool{"restoreformer": false, "pinned-model": false, "unchanged": true}), now.Add(time.Hour)); err != nil || len(changes) != 0 {
		t.Fatalf("second reconcile changed %+v (err %v)", changes, err)
	}
	if overlay, _ = st.LoadOverlay(ctx); len(overlay) != 2 {
		t.Fatalf("overlay = %v", overlay)
	}
	// Setting it again replaces the superseded row with a fresh, active one.
	set(t, st, "restoreformer", false, false, false)
	if overlay, _ = st.LoadOverlay(ctx); overlay["restoreformer"] {
		t.Fatalf("a fresh override must apply: %v", overlay)
	}
}

// TestLegacyOverrideKeepsItsEffectUntilTheSeedMoves proves a pre-provenance row
// is never silently flipped, is adopted with the current seed as its baseline,
// and is superseded by the next seed change like any other.
func TestLegacyOverrideKeepsItsEffectUntilTheSeedMoves(t *testing.T) {
	ctx := context.Background()
	st, d := newModelOverlay(t)
	if _, err := d.ExecContext(ctx, `INSERT INTO model_override (id, enabled, set_by) VALUES ('legacy', 0, ?)`, catalogstate.UnknownActor); err != nil {
		t.Fatal(err)
	}
	changes, err := st.Reconcile(ctx, seed(map[string]bool{"legacy": true}), t0)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Action != "adopted" || !strings.Contains(changes[0].Detail, "contradicts the seed") {
		t.Fatalf("changes = %+v", changes)
	}
	if overlay, _ := st.LoadOverlay(ctx); overlay["legacy"] {
		t.Fatal("an adopted legacy override must keep its effect")
	}
	o, _, _ := st.GetOverride(ctx, "legacy")
	if o.SeedEnabled == nil || !*o.SeedEnabled || o.SetBy != catalogstate.UnknownActor {
		t.Fatalf("adopted override = %+v", o)
	}
	// The next seed change supersedes it.
	changes, err = st.Reconcile(ctx, seed(map[string]bool{"legacy": false}), t0.Add(time.Hour))
	if err != nil || len(changes) != 1 || changes[0].Action != "superseded" {
		t.Fatalf("changes = %+v err %v", changes, err)
	}
}

func TestReconcileLeavesUnknownIDsAlone(t *testing.T) {
	ctx := context.Background()
	st, _ := newModelOverlay(t)
	set(t, st, "removed-from-seed", false, false, true)
	if changes, err := st.Reconcile(ctx, seed(nil), t0); err != nil || len(changes) != 0 {
		t.Fatalf("changes = %+v err %v", changes, err)
	}
	if overlay, _ := st.LoadOverlay(ctx); len(overlay) != 1 {
		t.Fatalf("overlay = %v", overlay)
	}
}

func TestClearOverride(t *testing.T) {
	ctx := context.Background()
	st, _ := newModelOverlay(t)
	set(t, st, "m", false, false, true)
	if existed, err := st.ClearOverride(ctx, "m"); err != nil || !existed {
		t.Fatalf("clear = %v %v", existed, err)
	}
	if existed, _ := st.ClearOverride(ctx, "m"); existed {
		t.Fatal("clearing twice must report no row")
	}
}

// TestAdapterOverridesFollowTheSameRule proves the mechanism is shared: the
// adapter catalog's overlay has the same columns and the same supersession.
func TestAdapterOverridesFollowTheSameRule(t *testing.T) {
	ctx := context.Background()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(adapters.Schema)); err != nil {
		t.Fatal(err)
	}
	st := adapters.NewStore(d)
	if err := st.SetOverride(ctx, catalogstate.Override{ID: "lora", Enabled: false, SetBy: "api"}, false, t0); err != nil {
		t.Fatal(err)
	}
	changes, err := st.Reconcile(ctx, seed(map[string]bool{"lora": true}), t0.Add(time.Hour))
	if err != nil || len(changes) != 1 || changes[0].Action != "superseded" {
		t.Fatalf("changes = %+v err %v", changes, err)
	}
}
