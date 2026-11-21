package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
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
			// `procedural` and `vector` ship what the generator drew, so they
			// have no treatment to parameterise. Their art direction lives
			// entirely in the generator parameters, which the distinctness lint
			// checks. Both are legal here for the same reason: a mesh gradient
			// is finished when the generator finishes, and a burin cut is
			// finished when the burin stops — putting a screen over either adds
			// a mechanical texture the look exists to avoid.
			require.Containsf(t, []string{"procedural", "vector"}, s.Strategy,
				"style %q declares no treatments but is not a `procedural` or `vector` style", s.ID)
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
	// Import normalises the same defaults a write does, so the round trip
	// returns the style as the catalog would actually store it rather than
	// exactly as it was handed over. An unset tier reads back as `procedural`
	// for the same reason an unset contrast threshold reads back as 4.5: the
	// default states what the style already is.
	expected := style
	expected.QualityTier = TierProcedural
	require.Equal(t, []Style{expected}, styles)
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

// A plate spec is refused for the relational mistakes, not just the obvious
// ones. Each of these would otherwise be discovered by the compositor at render
// time, when the only honest response is to fail a picture someone asked for.
func TestThePlateSpecRefusesAStackThatCouldNotComposite(t *testing.T) {
	base := func(spec []PlateSpec) Style {
		return Style{
			ID: "layered", Name: "Layered", Role: "ambient", Subject: "horizon",
			Lineage: "bauhaus", Strategy: "procedural", Placements: []string{"full_bleed"},
			PlateSpec: spec,
		}
	}
	for name, tc := range map[string]struct {
		spec []PlateSpec
		want string
	}{
		"two plates at one depth": {
			spec: []PlateSpec{{Name: "sky", Depth: 0, Opacity: 1}, {Name: "sea", Depth: 0, Opacity: 1}},
			want: "same depth",
		},
		"two plates with one name": {
			spec: []PlateSpec{{Name: "sky", Depth: 0, Opacity: 1}, {Name: "sky", Depth: 1, Opacity: 1}},
			want: "two plates named",
		},
		"a plate with no name": {
			spec: []PlateSpec{{Name: "  ", Depth: 0, Opacity: 1}},
			want: "no name",
		},
		"a blend the compositor cannot run": {
			spec: []PlateSpec{{Name: "sky", Depth: 0, Blend: "overlay", Opacity: 1}},
			want: "the compositor runs",
		},
		"an opacity outside 0..1": {
			spec: []PlateSpec{{Name: "sky", Depth: 0, Opacity: 1.4}},
			want: "between 0 and 1",
		},
		"a treatment no engine implements": {
			spec: []PlateSpec{{Name: "sky", Depth: 0, Opacity: 1, Treatments: []string{"caustics"}}},
			want: "no engine operation implements",
		},
		"more plates than the plan permits": {
			spec: []PlateSpec{
				{Name: "a", Depth: 0, Opacity: 1},
				{Name: "b", Depth: 1, Opacity: 1},
				{Name: "c", Depth: 2, Opacity: 1},
				{Name: "d", Depth: 3, Opacity: 1},
			},
			want: "at most",
		},
	} {
		t.Run(name, func(t *testing.T) {
			style := base(tc.spec)
			require.ErrorContains(t, validateStyle(&style), tc.want)
		})
	}
}

// And a legal stack is accepted, so the refusals above are discriminating
// rather than a blanket rejection of the field.
func TestALegalPlateSpecIsAccepted(t *testing.T) {
	style := Style{
		ID: "layered", Name: "Layered", Role: "ambient", Subject: "horizon",
		Lineage: "bauhaus", Strategy: "procedural", Placements: []string{"full_bleed"},
		PlateSpec: []PlateSpec{
			{Name: "sky", Depth: 0, Blend: BlendNormal, Opacity: 1},
			{Name: "sea", Depth: 1, Blend: BlendMultiply, Opacity: 0.9, Treatments: []string{"halftone"}},
			{Name: "shore", Depth: 2, Blend: BlendScreen, Opacity: 0.6},
		},
	}
	require.NoError(t, validateStyle(&style))
}

// A plate spec has to survive storage, or a style that declared a stack reads
// back as a flat one and renders a different picture than it says.
func TestAPlateSpecRoundTripsThroughTheStore(t *testing.T) {
	store := NewStore(freshDB(t))
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))

	style := Style{
		ID: "operator-layered", Name: "Operator Layered", Role: "ambient", Subject: "horizon",
		Lineage: "bauhaus", Strategy: "procedural", Placements: []string{"full_bleed"},
		PlateSpec: []PlateSpec{
			{Name: "sky", Depth: 0, Blend: BlendNormal, Opacity: 1},
			{Name: "sea", Depth: 1, Blend: BlendMultiply, Opacity: 0.9, Treatments: []string{"halftone"}},
		},
	}
	require.NoError(t, store.CreateStyle(ctx, style))

	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	var read Style
	for _, s := range styles {
		if s.ID == style.ID {
			read = s
		}
	}
	require.Equal(t, style.PlateSpec, read.PlateSpec)
}

// A plate may merge several generator planes, and no plane may belong to two
// plates. The first is what lets a style ship fewer plates than its generator
// separates; the second is what stops a plane being drawn twice, which
// composites to a different picture than the generator drew.
func TestAPlateMayMergePlanesButAPlaneBelongsToOnePlate(t *testing.T) {
	base := func(spec []PlateSpec) Style {
		return Style{
			ID: "layered", Name: "Layered", Role: "ambient", Subject: "horizon",
			Lineage: "bauhaus", Strategy: "procedural", Placements: []string{"full_bleed"},
			PlateSpec: spec,
		}
	}
	merged := base([]PlateSpec{
		{Name: "distance", Depth: 0, Blend: BlendNormal, Opacity: 1, Planes: []string{"sea", "headland"}},
		{Name: "arcade", Depth: 1, Blend: BlendNormal, Opacity: 1},
	})
	require.NoError(t, validateStyle(&merged))

	doubled := base([]PlateSpec{
		{Name: "far", Depth: 0, Blend: BlendNormal, Opacity: 1, Planes: []string{"sea", "headland"}},
		{Name: "near", Depth: 1, Blend: BlendNormal, Opacity: 1, Planes: []string{"headland"}},
	})
	require.ErrorContains(t, validateStyle(&doubled), "belongs to one plate")

	empty := base([]PlateSpec{{Name: "far", Depth: 0, Blend: BlendNormal, Opacity: 1, Planes: []string{" "}}})
	require.ErrorContains(t, validateStyle(&empty), "empty source plane")
}

// A plate that names no planes sources the one plane matching its own name.
// Every plate spec in the catalog relies on this default.
func TestAPlateWithNoPlaneListSourcesItsOwnName(t *testing.T) {
	require.Equal(t, []string{"canopy"}, PlateSpec{Name: "canopy"}.SourcePlanes())
	require.Equal(t, []string{"sea", "headland"}, PlateSpec{Name: "distance", Planes: []string{"sea", "headland"}}.SourcePlanes())
}

// Per-plate parameters are what makes depth-grading expressible: the same
// operation at two rulings, one per layer. Per-plate chains alone can say
// "screen this layer and not that one" but never "screen both, differently" —
// and the second is the depth cue.
func TestPerPlateParametersOverlayTheStyles(t *testing.T) {
	plate := PlateSpec{
		Name: "sea", Treatments: []string{"halftone", "grain"},
		TreatmentParams: map[string]string{"halftone": `{"lpi":96}`},
	}
	style := map[string]string{"halftone": `{"lpi":180}`, "grain": `{"amount":0.05}`}

	merged := plate.EffectiveTreatmentParams(style)
	require.Equal(t, `{"lpi":96}`, merged["halftone"], "the plate's parameters win for the operation it tunes")
	require.Equal(t, `{"amount":0.05}`, merged["grain"], "the style's parameters survive for operations the plate does not tune")
	require.Equal(t, `{"lpi":180}`, style["halftone"], "the style's own map must not be mutated")

	// A plate that tunes nothing gets the style's map unchanged.
	require.Equal(t, style, PlateSpec{Name: "sky"}.EffectiveTreatmentParams(style))
}

// Parameters for an operation a plate does not run are dead weight that reads
// as intent. An author who tuned a ruling and then removed the screen should
// learn it here, not by wondering why the picture never changed.
func TestAPlateMayNotParameteriseATreatmentItDoesNotRun(t *testing.T) {
	style := Style{
		ID: "layered", Name: "Layered", Role: "ambient", Subject: "horizon",
		Lineage: "bauhaus", Strategy: "procedural", Placements: []string{"full_bleed"},
		PlateSpec: []PlateSpec{
			{
				Name: "sky", Depth: 0, Opacity: 1, Treatments: []string{"grain"},
				TreatmentParams: map[string]string{"halftone": `{"lpi":96}`},
			},
			{Name: "sea", Depth: 1, Opacity: 1},
		},
	}
	require.ErrorContains(t, validateStyle(&style), "does not run it")

	unknown := style
	unknown.PlateSpec = []PlateSpec{
		{Name: "sky", Depth: 0, Opacity: 1, Treatments: []string{"caustics"}},
	}
	require.ErrorContains(t, validateStyle(&unknown), "no engine operation implements")
}
