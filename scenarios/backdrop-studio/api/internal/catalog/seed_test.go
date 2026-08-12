package catalog

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func freshDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	return db
}

// TestSeedContentIsValid runs at build time over the shipped data files. Seed
// content is data now, so nothing but this test stands between a typo in a JSON
// file and an install that cannot start.
func TestSeedContentIsValid(t *testing.T) {
	seeds, err := LoadSeeds()
	require.NoError(t, err)
	require.NotEmpty(t, seeds)

	for _, file := range seeds {
		version, ok := parseSeedVersionFromName(seedFileName(file.Version))
		require.True(t, ok)
		require.Equal(t, file.Version, version)

		require.NotEmpty(t, file.Styles, "seed v%d ships no styles", file.Version)
		for i := range file.Styles {
			style := file.Styles[i]
			require.NoErrorf(t, validateStyle(&style), "seed v%d style %q is invalid", file.Version, style.ID)
		}
		for _, surface := range file.Surfaces {
			require.NoErrorf(t, validateSurface(surface), "seed v%d surface %q is invalid", file.Version, surface.ID)
		}
	}
}

func seedFileName(version int) string {
	return "v" + strconv.Itoa(version) + ".json"
}

// TestSeedIsIdempotent proves a restart is free. Seed runs on every start, so a
// second application that duplicated or reset rows would corrupt a live install
// on an ordinary restart.
func TestSeedIsIdempotent(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	first, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)

	require.NoError(t, store.Seed(ctx))
	second, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, first, second)
}

// TestSeedUpgradesAnOlderInstall is the defect this phase exists to close.
// `Seed` used to guard on "is the styles table empty?", so an install
// bootstrapped with four styles could never receive a fifth — the catalog, which
// this scenario's own documentation calls "the product", was frozen at whatever
// version first created the database.
func TestSeedUpgradesAnOlderInstall(t *testing.T) {
	db := freshDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// An install from before seed versioning: some seed rows, no recorded
	// version, and no idea that more exist.
	require.NoError(t, store.migrate(ctx))
	require.NoError(t, store.insertStyle(ctx, Style{
		ID: "cyanotype-arcade", Name: "Old Cyanotype Arcade", Version: 1, Role: "ambient",
		Subject: "statuary_architecture", Lineage: "cyanotype", Strategy: "procedural-treated",
		Treatments: []string{"duotone"}, Placements: []string{"full_bleed"}, ContrastThreshold: 4.5,
	}, OriginSeed, 0))

	require.NoError(t, store.Seed(ctx))

	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.Len(t, styles, 16, "an older install must receive the current catalog")

	upgraded, err := store.GetStyle(ctx, "cyanotype-arcade")
	require.NoError(t, err)
	require.Equal(t, "Cyanotype Arcade", upgraded.Name, "a seed row must be upgraded in place")
	require.NotEmpty(t, upgraded.Inks, "the upgrade must carry the new ink defaults")

	surfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.Len(t, surfaces, 9)

	applied, err := store.AppliedSeedVersion(ctx)
	require.NoError(t, err)
	shipped, err := SeedVersion()
	require.NoError(t, err)
	require.Equal(t, shipped, applied)
}

// TestSeedNeverOverwritesOperatorRows is the other half of upgradeability. An
// upgrade that silently discarded an operator's authored art direction would be
// a worse failure than never upgrading at all, so origin is load-bearing.
func TestSeedNeverOverwritesOperatorRows(t *testing.T) {
	db := freshDB(t)
	store := NewStore(db)
	ctx := context.Background()
	require.NoError(t, store.migrate(ctx))

	// The operator claims an id the shipped seed also uses.
	operatorStyle := Style{
		ID: "cyanotype-arcade", Name: "My Own Arcade", Version: 4, Role: "ambient",
		Subject: "statuary_architecture", Lineage: "bauhaus", Strategy: "procedural-treated",
		Treatments: []string{"posterize"}, Placements: []string{"split_panel"}, ContrastThreshold: 3.2,
		Inks:            map[string]string{"$brand.primary": "#123456"},
		TreatmentParams: map[string]string{"posterize": `{"levels":3,"dark":"$brand.primary"}`},
	}
	require.NoError(t, store.CreateStyle(ctx, operatorStyle))
	require.NoError(t, store.PutSurface(ctx, Surface{
		ID: "web.hero", Name: "My Own Hero", Kind: "product", Width: 1600, Height: 900,
		Placements: []string{"full_bleed"}, Authority: "operator", ConfirmedOn: "2026-08-12",
	}))

	require.NoError(t, store.Seed(ctx))

	got, err := store.GetStyle(ctx, "cyanotype-arcade")
	require.NoError(t, err)
	require.Equal(t, "My Own Arcade", got.Name, "seeding must not overwrite an operator-authored style")
	require.Equal(t, "bauhaus", got.Lineage)
	require.Equal(t, OriginOperator, got.Origin)

	surface, err := store.GetSurface(ctx, "web.hero")
	require.NoError(t, err)
	require.Equal(t, "My Own Hero", surface.Name, "seeding must not overwrite an operator-authored surface")

	// Everything else still arrives.
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.Len(t, styles, 16)
}

// TestEverySeededStyleRendersWithoutABrand is the cold-install contract stated
// as a test. It is deliberately separate from the wire-contract test: that one
// asks "will image-tools accept these bytes", this one asks "can this style
// produce bytes at all with nothing bound".
func TestEverySeededStyleRendersWithoutABrand(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)

	for _, style := range styles {
		palette := style.EffectivePalette(nil)
		for _, op := range style.Treatments {
			params := style.TreatmentParams[op]
			for _, slot := range BrandSlots {
				if !strings.Contains(params, slot) {
					continue
				}
				require.NotEmptyf(t, palette[slot],
					"style %q op %q references %s but the effective palette cannot bind it without a brand", style.ID, op, slot)
			}
		}
	}
}

// prePlanSchema is the schema an install created before this plan actually has.
// It is reproduced verbatim rather than derived from Schema(), because the whole
// point is to migrate from what old installs hold, not from what new ones get.
//
// The `payload` column is the load-bearing detail: NOT NULL with no default. A
// migration test that starts from the current schema cannot see it, and one that
// starts from here fails immediately without the drop.
const prePlanSchema = `
CREATE TABLE IF NOT EXISTS backdrop_surfaces (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, kind TEXT NOT NULL,
  width INTEGER NOT NULL, height INTEGER NOT NULL, placements TEXT NOT NULL,
  authority TEXT NOT NULL, confirmed_on TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS backdrop_styles (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, version INTEGER NOT NULL,
  role TEXT NOT NULL, subject TEXT NOT NULL, lineage TEXT NOT NULL,
  strategy TEXT NOT NULL, treatments TEXT NOT NULL, placements TEXT NOT NULL,
  regions TEXT NOT NULL, contrast_threshold REAL NOT NULL,
  scaffold TEXT NOT NULL DEFAULT 'null', generation TEXT NOT NULL DEFAULT 'null',
  parent_id TEXT NOT NULL DEFAULT '', treatment_params TEXT NOT NULL DEFAULT '{}',
  released INTEGER NOT NULL, payload TEXT NOT NULL
);`

// TestSeedMigratesAPrePlanInstall is the test the unit suite was missing. The
// first live upgrade attempt failed with
// `NOT NULL constraint failed: backdrop_styles.payload` while every unit test
// passed, because the tests built their database from the *current* schema —
// the one shape no real upgrade starts from.
func TestSeedMigratesAPrePlanInstall(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(prePlanSchema)
	require.NoError(t, err)

	// Four styles and five surfaces: the shape the 2026-08-12 audit found on a
	// real install, frozen there because Seed() guarded on an empty table.
	for i, id := range []string{"cyanotype-arcade", "riso-horizon", "stipple-massif", "technical-field"} {
		_, err = db.Exec(`INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,released,payload) VALUES(?,?,1,'ambient','non_representational','bauhaus','procedural-treated','["grain"]','["full_bleed"]','[]',4.5,0,'{}')`, id, "Old "+id)
		require.NoErrorf(t, err, "seeding pre-plan style %d", i)
	}
	_, err = db.Exec(`INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,released,payload) VALUES('my-operator-style','My Operator Style',3,'ambient','non_representational','bauhaus','procedural-treated','["grain"]','["full_bleed"]','[]',4.5,0,'{}')`)
	require.NoError(t, err)

	store := NewStore(db)
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx), "a pre-plan install must upgrade")

	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.Len(t, styles, 17, "16 seeded styles plus the operator's own")

	surfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.Len(t, surfaces, 9, "a 5-surface install must reach the current 9")

	// Rows that predate origin tracking default to 'seed' and are upgraded.
	upgraded, err := store.GetStyle(ctx, "cyanotype-arcade")
	require.NoError(t, err)
	require.Equal(t, "Cyanotype Arcade", upgraded.Name)
	require.NotEmpty(t, upgraded.Inks)

	// The operator's row is untouched even though it also defaulted to 'seed':
	// the current seed does not contain that id, and no row is ever deleted.
	survived, err := store.GetStyle(ctx, "my-operator-style")
	require.NoError(t, err)
	require.Equal(t, "My Operator Style", survived.Name)
	require.Equal(t, 3, survived.Version)
}
