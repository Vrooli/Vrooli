package catalog

import (
	"context"
	"testing"

	"backdrop-studio/internal/testutil/db"
	"github.com/stretchr/testify/require"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.Exec(Schema())
	require.NoError(t, err)
	return NewStore(sqlDB)
}

func TestSeedIsIdempotentAndCatalogIsQueryable(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	require.NoError(t, store.Seed(ctx))
	require.NoError(t, store.Seed(ctx))

	surfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.Len(t, surfaces, seededSurfaceCount(t))
	// Every surface kind the catalog claims to serve must actually be present.
	// This replaced an assertion on surfaces[3].ID, which passed only because
	// the alphabetical position of one store surface happened to be third —
	// adding an `email` row moved it and failed a test about nothing.
	kinds := map[string]bool{}
	for _, s := range surfaces {
		kinds[s.Kind] = true
	}
	for _, kind := range []string{"product", "store", "social", "email"} {
		require.Truef(t, kinds[kind], "no seeded surface has kind %q, so that delivery lane has nothing to render into", kind)
	}

	styles, err := store.ListStyles(ctx, "ambient", "horizon", "dither_diffusion", "riso_zine", "full_bleed")
	require.NoError(t, err)
	require.Len(t, styles, 1)
	require.Equal(t, "riso-horizon", styles[0].ID)
}

// TestSeededStylesCarryTheirOwnParameters guards the reason TreatmentParams
// exists. Before it, every style naming an op rendered with one hardcoded
// parameter set, so a catalog of sixteen entries produced about four distinct
// looks. A style that names an op without saying how it wants that op run is
// not an art direction, it is a label.
func TestSeededStylesCarryTheirOwnParameters(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	for _, s := range styles {
		if len(s.Treatments) == 0 {
			// A `procedural` style ships the scene as drawn, so it has no
			// treatment to parameterise. Its art direction lives entirely in
			// the generator parameters, which the distinctness lint checks.
			require.Equal(t, "procedural", s.Strategy,
				"style %q declares no treatments but is not a `procedural` style", s.ID)
			require.NotEmptyf(t, s.Scaffold, "style %q has neither treatments nor a generator binding, so nothing decides what it looks like", s.ID)
			continue
		}
		require.NotEmpty(t, s.TreatmentParams, "style %q names treatments but no parameters", s.ID)
		for _, op := range s.Treatments {
			require.Contains(t, s.TreatmentParams, op,
				"style %q names treatment %q without parameters for it", s.ID, op)
		}
	}

	// Distinctness itself is checked by TestNoTwoSettledStylesRenderTheSamePicture
	// in distinctness_test.go, not here. This check used to key on subject plus
	// chain plus parameters, which was right when every style ran a treatment
	// and wrong once styles could ship a scene untreated: five `procedural`
	// styles with empty chains all hashed to the same key and were reported as
	// duplicates when the generator, its palette and its framing were what made
	// them different. The replacement resolves the generator and canonicalises
	// the parameter JSON, so it compares what actually decides the pixels.

	// The ops whose whole character is a parameter must vary across the
	// catalog. Uniformity here was the original defect: every halftone ran at
	// the same line frequency, so every screened style looked the same.
	for _, op := range []string{"halftone", "posterize", "line_screen"} {
		configs := map[string]bool{}
		users := 0
		for _, s := range styles {
			if p, ok := s.TreatmentParams[op]; ok {
				configs[p] = true
				users++
			}
		}
		if users >= 2 && len(configs) < 2 {
			t.Errorf("op %q is used by %d styles but configured only one way — the catalog names variety it does not have", op, users)
		}
	}
}

func TestCreateStyleRejectsUnknownClosedAxesAndEmptyOpenAxes(t *testing.T) {
	store := testStore(t)
	base := Style{
		ID: "test", Name: "Test", Role: "ambient", Subject: "horizon", Lineage: "wpa_poster",
		Strategy: "procedural", Treatments: []string{"posterize"}, Placements: []string{"full_bleed"},
	}

	badRole := base
	badRole.Role = "decorative"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badRole), "invalid role")

	badStrategy := base
	badStrategy.Strategy = "model-only"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badStrategy), "invalid strategy")

	badSubjectEmpty := base
	badSubjectEmpty.Subject = ""
	require.ErrorContains(t, store.CreateStyle(context.Background(), badSubjectEmpty), "invalid subject")

	badPlacement := base
	badPlacement.Placements = []string{""}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badPlacement), "invalid placement")

	badTreatment := base
	badTreatment.Treatments = []string{"unknown-treatment"}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badTreatment), "invalid treatment")

	badSubject := base
	badSubject.Subject = "unknown-subject"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badSubject), "invalid subject")

	badLineage := base
	badLineage.Lineage = "unknown-lineage"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badLineage), "invalid lineage")

	badGeneration := base
	badGeneration.Generation = &GenerationBlock{PromptTemplate: "bad shape"}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badGeneration), "cannot carry a generation block")

	badRegion := base
	badRegion.Regions = []Region{{X: .5, Y: .5, Width: .6, Height: .2, Kind: "overlay"}}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badRegion), "invalid region")

	require.NoError(t, store.CreateStyle(context.Background(), base))
}

func TestReleasedStyleCannotBeTouched(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	v := Style{ID: "released", Name: "Released", Role: "ambient", Subject: "horizon", Lineage: "wpa_poster", Strategy: "procedural", Treatments: []string{"grain"}, Placements: []string{"full_bleed"}}
	require.NoError(t, store.CreateStyle(ctx, v))
	_, err := store.db.ExecContext(ctx, "UPDATE backdrop_styles SET released=1 WHERE id=?", v.ID)
	require.NoError(t, err)
	require.ErrorContains(t, store.TouchStyle(ctx, v.ID), "immutable")
}

func TestForkMutatesExactlyOneAxisAndPreservesLineage(t *testing.T) {
	store := testStore(t)
	parent := Style{ID: "parent", Name: "Parent", Version: 1, Role: "ambient", Subject: "horizon", Lineage: "wpa_poster", Strategy: "procedural", Treatments: []string{"posterize"}, Placements: []string{"full_bleed"}, Regions: []Region{{X: .1, Y: .1, Width: .4, Height: .2, Kind: "overlay"}}, ContrastThreshold: 4.5}
	require.NoError(t, store.CreateStyle(context.Background(), parent))
	child, err := store.ForkStyle(context.Background(), "parent", "child", map[string]string{"lineage": "bauhaus"})
	require.NoError(t, err)
	require.Equal(t, "parent", child.ParentID)
	require.Equal(t, "bauhaus", child.Lineage)
	_, err = store.ForkStyle(context.Background(), "parent", "bad", map[string]string{})
	require.ErrorContains(t, err, "exactly one axis")
}

func TestStylePackRoundTrips(t *testing.T) {
	style := Style{ID: "pack-style", Name: "Pack", Version: 1, Role: "ambient", Subject: "horizon", Lineage: "wpa_poster", Strategy: "procedural", Treatments: []string{"posterize"}, Placements: []string{"full_bleed"}, Regions: []Region{{X: .1, Y: .1, Width: .4, Height: .2, Kind: "overlay"}}, ContrastThreshold: 4.5}
	raw, err := ExportStylePack([]Style{style})
	require.NoError(t, err)
	styles, err := ImportStylePack(raw)
	require.NoError(t, err)
	require.Equal(t, []Style{style}, styles)
}

// TestCreateStyleRejectsParametersTheEngineWillNotAccept pins validation at the
// write path, which is the point of doing it at all. Before this, a style whose
// parameters image-tools would reject stored cleanly, passed every unit suite,
// and failed at its first real render with a 400 — by which time the author was
// long gone from the decision.
func TestCreateStyleRejectsParametersTheEngineWillNotAccept(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	base := Style{
		ID: "probe", Name: "Probe", Role: "ambient", Subject: "horizon", Lineage: "wpa_poster",
		Strategy: "procedural-treated", Treatments: []string{"duotone"},
		Placements: []string{"full_bleed"}, ContrastThreshold: 4.5,
	}

	t.Run("unknown field", func(t *testing.T) {
		v := base
		v.TreatmentParams = map[string]string{"duotone": `{"dark":"#111827","nonexistent_knob":true}`}
		err := store.CreateStyle(ctx, v)
		require.ErrorContains(t, err, "not accepted by image-tools")
		require.ErrorContains(t, err, "probe")
	})

	t.Run("malformed json", func(t *testing.T) {
		v := base
		v.TreatmentParams = map[string]string{"duotone": `{"dark":`}
		require.ErrorContains(t, store.CreateStyle(ctx, v), "must be a JSON object")
	})

	t.Run("parameters for an operation the style does not run", func(t *testing.T) {
		v := base
		v.TreatmentParams = map[string]string{"duotone": `{}`, "halftone": `{"lpi":72}`}
		require.ErrorContains(t, store.CreateStyle(ctx, v), "does not run")
	})

	t.Run("a brand slot with no declared default is refused", func(t *testing.T) {
		v := base
		v.ID = "probe-unbound"
		v.TreatmentParams = map[string]string{"duotone": `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`}
		// No Inks: this style could not render on a cold install, which is the
		// exact shape that made ten seeded styles unrenderable.
		require.ErrorContains(t, store.CreateStyle(ctx, v), "no declared ink default")
	})

	t.Run("an unknown brand slot is refused", func(t *testing.T) {
		v := base
		v.ID = "probe-typo"
		v.TreatmentParams = map[string]string{"duotone": `{"dark":"$brand.primry","light":"#ffffff"}`}
		v.Inks = map[string]string{"$brand.primary": "#111827"}
		require.ErrorContains(t, store.CreateStyle(ctx, v), "unknown brand slot")
	})

	t.Run("valid parameters are accepted", func(t *testing.T) {
		v := base
		v.ID = "probe-ok"
		v.TreatmentParams = map[string]string{"duotone": `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`}
		v.Inks = map[string]string{"$brand.primary": "#111827", "$brand.background": "#f5efdc"}
		require.NoError(t, store.CreateStyle(ctx, v))
		got, err := store.GetStyle(ctx, "probe-ok")
		require.NoError(t, err)
		require.Equal(t, v.TreatmentParams, got.TreatmentParams, "parameters must round-trip through storage")
		require.Equal(t, v.Inks, got.Inks, "ink defaults must round-trip through storage")
	})
}

// TestImportStylePackRejectsBadParameters pins the other write path. A style
// pack is the route by which styles arrive from outside this scenario, so it is
// the one most likely to carry parameters authored against a different engine.
func TestImportStylePackRejectsBadParameters(t *testing.T) {
	pack := []byte(`{"version":1,"styles":[{
		"id":"imported","name":"Imported","version":1,"role":"ambient","subject":"horizon",
		"lineage":"wpa_poster","strategy":"procedural-treated","treatments":["halftone"],
		"placements":["full_bleed"],"contrast_threshold":4.5,
		"treatment_params":{"halftone":"{\"lpi\":72,\"screen_angle\":15}"}
	}]}`)
	_, err := ImportStylePack(pack)
	require.ErrorContains(t, err, "not accepted by image-tools")
}
