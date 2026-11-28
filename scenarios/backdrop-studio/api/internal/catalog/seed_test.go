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

		require.True(t, len(file.Styles) > 0 || len(file.Surfaces) > 0, "seed v%d ships neither styles nor surfaces", file.Version)
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
	firstSurfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.NoError(t, store.Seed(ctx))
	secondSurfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.Equal(t, firstSurfaces, secondSurfaces)
}

func TestSeedIncludesCurrentOfficialStoreSurfacePack(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	listed, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	listedIDs := make(map[string]bool, len(listed))
	storeCount := 0
	for _, surface := range listed {
		listedIDs[surface.ID] = true
		if surface.Kind == "store" {
			storeCount++
		}
	}
	require.Equal(t, 8, storeCount, "surface listing must expose the full declared store pack")

	cases := []struct {
		id, authority string
		width, height int
	}{
		{"play.feature-graphic", "support.google.com/googleplay/android-developer/answer/9866151", 1024, 500},
		{"play.phone-screenshot", "support.google.com/googleplay/android-developer/answer/9866151", 1080, 1920},
		{"play.tablet-screenshot", "support.google.com/googleplay/android-developer/answer/9866151", 1920, 1080},
		{"app-store-iphone-dynamic-island-medium", "developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications", 1206, 2622},
		{"app-store-ipad-13-inch", "developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications", 2064, 2752},
		{"app-store-6.7-screenshot", "developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications", 1290, 2796},
		{"app-store-6.5-screenshot", "developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications", 1284, 2778},
		{"app-store-12.9-screenshot", "developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications", 2048, 2732},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			require.True(t, listedIDs[tc.id], "surface listing omitted seeded store record")
			surface, err := store.GetSurface(ctx, tc.id)
			require.NoError(t, err)
			require.Equal(t, "store", surface.Kind)
			require.Equal(t, tc.width, surface.Width)
			require.Equal(t, tc.height, surface.Height)
			require.Contains(t, surface.Authority, tc.authority)
			require.Equal(t, "2026-11-24", surface.ConfirmedOn)
			require.NotEmpty(t, surface.Placements)
		})
	}
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
	require.Len(t, styles, seededStyleCount(t), "an older install must receive the current catalog")

	upgraded, err := store.GetStyle(ctx, "cyanotype-arcade")
	require.NoError(t, err)
	require.Equal(t, "Cyanotype Arcade", upgraded.Name, "a seed row must be upgraded in place")
	require.NotEmpty(t, upgraded.Inks, "the upgrade must carry the new ink defaults")

	surfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.Len(t, surfaces, seededSurfaceCount(t))

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
	require.Len(t, styles, seededStyleCount(t))
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
	require.Len(t, styles, seededStyleCount(t)+1, "every seeded style plus the operator's own")

	surfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.Len(t, surfaces, seededSurfaceCount(t), "a 5-surface install must reach the full shipped set")

	// Ambiguous legacy rows are operator-owned by default, even when their id
	// collides with a shipped seed. Migration must not infer authorship from ID.
	upgraded, err := store.GetStyle(ctx, "cyanotype-arcade")
	require.NoError(t, err)
	require.Equal(t, "Old cyanotype-arcade", upgraded.Name)
	require.Equal(t, OriginOperator, upgraded.Origin)

	// The operator's row is untouched even though it also defaulted to 'seed':
	// the current seed does not contain that id, and no row is ever deleted.
	survived, err := store.GetStyle(ctx, "my-operator-style")
	require.NoError(t, err)
	require.Equal(t, "My Operator Style", survived.Name)
	require.Equal(t, 3, survived.Version)
}

func TestSeedUpgradesUnmodifiedLegacySeedRow(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(prePlanSchema)
	require.NoError(t, err)

	seeds, err := LoadSeeds()
	require.NoError(t, err)
	var legacy Style
	var operatorSurfaceID string
	var seedSurfaceID string
	for _, seed := range seeds {
		if seed.Version == 1 {
			for _, style := range seed.Styles {
				if style.ID == "cyanotype-arcade" {
					legacy = style
				}
			}
			if len(seed.Surfaces) > 0 {
				surface := seed.Surfaces[0]
				surface.Name = "Operator-authored surface"
				operatorSurfaceID = surface.ID
				_, err = db.Exec(`INSERT INTO backdrop_surfaces(id,name,kind,width,height,placements,authority,confirmed_on) VALUES(?,?,?,?,?,?,?,?)`,
					surface.ID, surface.Name, surface.Kind, surface.Width, surface.Height, mustJSON(surface.Placements), surface.Authority, surface.ConfirmedOn)
				require.NoError(t, err)
			}
			if len(seed.Surfaces) > 1 {
				surface := seed.Surfaces[1]
				seedSurfaceID = surface.ID
				_, err = db.Exec(`INSERT INTO backdrop_surfaces(id,name,kind,width,height,placements,authority,confirmed_on) VALUES(?,?,?,?,?,?,?,?)`,
					surface.ID, surface.Name, surface.Kind, surface.Width, surface.Height, mustJSON(surface.Placements), surface.Authority, surface.ConfirmedOn)
				require.NoError(t, err)
			}
		}
	}
	require.NotEmpty(t, legacy.ID)
	_, err = db.Exec(`INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,parent_id,treatment_params,released,payload) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		legacy.ID, legacy.Name, legacy.Version, legacy.Role, legacy.Subject, legacy.Lineage, legacy.Strategy,
		mustJSON(legacy.Treatments), mustJSON(legacy.Placements), mustJSON(legacy.Regions), legacy.ContrastThreshold,
		mustJSON(legacy.Scaffold), mustJSON(legacy.Generation), legacy.ParentID, mustJSON(legacy.TreatmentParams), 1, `{}`)
	require.NoError(t, err)

	store := NewStore(db)
	require.NoError(t, store.Seed(context.Background()))
	got, err := store.GetStyle(context.Background(), legacy.ID)
	require.NoError(t, err)
	require.Equal(t, OriginSeed, got.Origin)
	require.Equal(t, "Cyanotype Arcade", got.Name)
	require.NotEmpty(t, got.Inks)
	var surfaceName string
	require.NoError(t, db.QueryRow(`SELECT name FROM backdrop_surfaces WHERE id=?`, operatorSurfaceID).Scan(&surfaceName))
	require.Equal(t, "Operator-authored surface", surfaceName)
	var origin string
	require.NoError(t, db.QueryRow(`SELECT origin FROM backdrop_surfaces WHERE id=?`, operatorSurfaceID).Scan(&origin))
	require.Equal(t, OriginOperator, origin)
	require.NoError(t, db.QueryRow(`SELECT origin FROM backdrop_surfaces WHERE id=?`, seedSurfaceID).Scan(&origin))
	require.Equal(t, OriginSeed, origin)
}

func TestSeedMigrationPreservesLegacyOperatorStyleWithSeedID(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(prePlanSchema)
	require.NoError(t, err)

	// Before origin tracking, an operator could edit a shipped style in place.
	// The legacy schema cannot distinguish that row from its shipped seed row.
	_, err = db.Exec(`INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,released,payload) VALUES('cyanotype-arcade','Operator Cyanotype',9,'ambient','statuary_architecture','operator-edited','procedural-treated','["duotone"]','["full_bleed"]','[]',4.1,1,'{}')`)
	require.NoError(t, err)

	store := NewStore(db)
	require.NoError(t, store.Seed(context.Background()))
	got, err := store.GetStyle(context.Background(), "cyanotype-arcade")
	require.NoError(t, err)
	require.Equal(t, "Operator Cyanotype", got.Name, "migration must not replace an operator-authored legacy row")
	require.Equal(t, 9, got.Version)
	require.Equal(t, "operator-edited", got.Lineage)
}

// seededStyleCount is the number of distinct styles the embedded seed versions
// define between them. It is computed rather than written down: a hardcoded
// count has to be edited every time the catalog grows, and an assertion whose
// upkeep is "bump the number until it passes" stops being an assertion.
func seededStyleCount(t *testing.T) int {
	t.Helper()
	seeds, err := LoadSeeds()
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, file := range seeds {
		for _, style := range file.Styles {
			ids[style.ID] = true
		}
	}
	return len(ids)
}

// seededSurfaceCount is derived for the same reason the style count is: a
// literal here is a number that has to be edited in three test files every time
// the catalog grows, and the edit that gets missed turns an unrelated failure
// into a puzzle. The seed data is the authority on how many surfaces ship.
func seededSurfaceCount(t *testing.T) int {
	t.Helper()
	seeds, err := LoadSeeds()
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, file := range seeds {
		for _, surface := range file.Surfaces {
			ids[surface.ID] = true
		}
	}
	return len(ids)
}
