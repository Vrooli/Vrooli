package render

import (
	"bytes"
	"context"
	"database/sql"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/vector"
)

// The guarantee the whole plate refactor rests on: an existing style, which
// declares no plate spec, renders exactly the bytes it rendered before plates
// existed.
//
// Asserted by construction rather than against a stored golden. The single-plate
// path applies the chain to the source and returns those exact bytes as both the
// composite and the plate — no compositor call, no re-encode — so the composite
// IS the treated source. A PNG that round-trips through a second encoder is not
// guaranteed to be the same bytes even when it is the same picture, which is
// exactly the drift a golden would catch a release too late.
func TestASinglePlateCandidateIsItsOwnComposite(t *testing.T) {
	for _, style := range []catalog.Style{
		{ID: "proc", Strategy: "procedural", Subject: "horizon", Placements: []string{"full_bleed"}},
		{ID: "treated", Strategy: "procedural-treated", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}},
	} {
		t.Run(style.ID, func(t *testing.T) {
			job, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{
				Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
			})
			require.NoError(t, err)
			candidate := job.Candidates[0]

			require.Len(t, candidate.Plates, 1, "a style declaring no spec renders one plate")
			require.True(t, bytes.Equal(candidate.PNG, candidate.Plates[0].PNG),
				"the single plate must BE the composite, not a re-encode of it")
			require.Equal(t, style.Treatments, candidate.Plates[0].Treatments,
				"the style's chain becomes the one plate's chain")
			require.Equal(t, catalog.BlendNormal, candidate.Plates[0].Blend)
			require.Equal(t, 1.0, candidate.Plates[0].Opacity)
		})
	}
}

// Every strategy produces a flat composite. No consumer loses its single image
// because a style grew a stack.
func TestEveryStrategyStillProducesACompositePNG(t *testing.T) {
	cases := []struct {
		name  string
		style catalog.Style
		gen   bool
	}{
		{"procedural", catalog.Style{ID: "p", Strategy: "procedural", Subject: "horizon", Placements: []string{"full_bleed"}}, false},
		{"procedural-treated", catalog.Style{ID: "pt", Strategy: "procedural-treated", Subject: "horizon", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"}}, false},
		{"vector", catalog.Style{
			ID: "v", Strategy: "vector", Subject: "cartographic", Placements: []string{"full_bleed"},
			Inks: map[string]string{"$brand.primary": "#12327a", "$brand.background": "#efe7d3", "$brand.accent": "#c9432f"},
		}, false},
		{"synthesized", catalog.Style{
			ID: "s", Strategy: "synthesized", QualityTier: catalog.TierLocalModel, Subject: "figure",
			Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
			Generation: &catalog.GenerationBlock{PromptTemplate: "a quiet figure"},
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore(&fakeExecutor{})
			if tc.gen {
				store = NewStoreWithGenerator(&fakeExecutor{}, &fakeGenerator{})
			}
			job, err := store.SubmitWithContext(context.Background(), Request{
				Style: tc.style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
			})
			require.NoError(t, err)
			candidate := job.Candidates[0]
			require.NotEmpty(t, candidate.PNG, "every strategy must deliver a flat composite")
			require.NotEmpty(t, candidate.Plates, "every candidate carries at least one plate")

			width, height, geoErr := pngGeometry(candidate.PNG)
			require.NoError(t, geoErr, "the composite must decode as a PNG")
			require.Equal(t, testSurface.Width, width)
			require.Equal(t, testSurface.Height, height)
		})
	}
}

// A style whose SOURCE has no layers at all is refused by name.
//
// This was written when no generator separated anything; both raster and vector
// generators do now, so the remaining case is the model-backed lane: a
// diffusion model returns one finished picture with no depth information in it,
// and deriving plates from that is a different capability with different
// honesty properties — an estimated matte, not the generator's own layer.
//
// Until that exists, a model-backed style declaring a stack is an error rather
// than a silently flattened picture. A style that says it draws a sky behind a
// colonnade and delivers one flat picture is the substitution this plan exists
// to remove.
func TestAModelBackedStyleDeclaringAStackIsRefusedByName(t *testing.T) {
	style := catalog.Style{
		ID: "layered-synth", Strategy: "synthesized", QualityTier: catalog.TierLocalModel,
		Subject: "figure", Placements: []string{"full_bleed"}, Treatments: []string{"duotone"},
		Generation: &catalog.GenerationBlock{PromptTemplate: "a quiet figure"},
		PlateSpec: []catalog.PlateSpec{
			{Name: "ground", Depth: 0, Blend: catalog.BlendNormal, Opacity: 1},
			{Name: "figure", Depth: 1, Blend: catalog.BlendNormal, Opacity: 1},
		},
	}
	_, err := NewStoreWithGenerator(&fakeExecutor{}, &fakeGenerator{}).SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.Error(t, err)
	var unavailable *MultiPlateUnavailableError
	require.ErrorAs(t, err, &unavailable)
	require.Equal(t, "layered-synth", unavailable.StyleID)
	require.Equal(t, 2, unavailable.Plates)
}

// The default spec a style gets when it declares none: one plate, normal blend,
// full opacity, carrying the style's own chain. Asserted directly because every
// byte-identity claim above depends on this being what it says.
func TestTheDefaultPlateSpecIsTheWholePicture(t *testing.T) {
	style := catalog.Style{ID: "x", Treatments: []string{"halftone", "grain"}}
	spec := style.EffectivePlateSpec()
	require.Len(t, spec, 1)
	require.Equal(t, 0, spec[0].Depth)
	require.Equal(t, catalog.BlendNormal, spec[0].Blend)
	require.Equal(t, 1.0, spec[0].Opacity)
	require.Equal(t, []string{"halftone", "grain"}, spec[0].Treatments)
}

// The vector lane fills a real stack: a style declaring three plates gets three
// plates, each carrying its own layer, and a composite assembled from them.
//
// This is the capability Phase 5 refused by name and this phase supplies. The
// fake executor's compositor mirrors the real one closely enough to prove the
// wiring; what the real one produces is proven in image-tools' own suite.
func TestAVectorStyleShipsItsGeneratorsDepthPlanes(t *testing.T) {
	style := catalog.Style{
		ID: "layered-colonnade", Strategy: "vector", Subject: "statuary_architecture",
		Placements: []string{"full_bleed"},
		Scaffold:   &catalog.ScaffoldBinding{Preset: "colonnade"},
		Inks: map[string]string{
			"$brand.primary": "#12327a", "$brand.background": "#efe7d3", "$brand.accent": "#c9432f",
		},
		PlateSpec: []catalog.PlateSpec{
			{Name: "distance", Depth: 0, Blend: catalog.BlendNormal, Opacity: 1},
			{Name: "arcade", Depth: 2, Blend: catalog.BlendNormal, Opacity: 1},
			{Name: "canopy", Depth: 3, Blend: catalog.BlendMultiply, Opacity: 0.9},
		},
	}
	job, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err)
	candidate := job.Candidates[0]

	require.Len(t, candidate.Plates, 3, "the declared stack is what ships")
	for i, want := range []struct {
		name  string
		depth int
		blend string
	}{
		{"distance", 0, catalog.BlendNormal},
		{"arcade", 2, catalog.BlendNormal},
		{"canopy", 3, catalog.BlendMultiply},
	} {
		require.Equal(t, want.name, candidate.Plates[i].Name)
		require.Equal(t, want.depth, candidate.Plates[i].Depth)
		require.Equal(t, want.blend, candidate.Plates[i].Blend)
		require.NotEmptyf(t, candidate.Plates[i].PNG, "plate %q carries no pixels", want.name)
	}

	// Each plate is a DIFFERENT picture. A stack whose layers were all the same
	// bytes would pass every structural assertion above and be worthless.
	require.NotEqual(t, candidate.Plates[0].PNG, candidate.Plates[1].PNG)
	require.NotEqual(t, candidate.Plates[1].PNG, candidate.Plates[2].PNG)

	require.NotEmpty(t, candidate.PNG, "the flat composite is never optional")
	width, height, geoErr := pngGeometry(candidate.PNG)
	require.NoError(t, geoErr)
	require.Equal(t, testSurface.Width, width)
	require.Equal(t, testSurface.Height, height)
}

// A style naming a plate its generator does not draw is refused by name, with
// the layers that generator does draw in the message — because the fix is a
// catalog edit and the operator needs to know what to rename it to.
func TestAStyleNamingAPlaneItsGeneratorDoesNotDrawIsRefused(t *testing.T) {
	style := catalog.Style{
		ID: "bad-plate", Strategy: "vector", Subject: "statuary_architecture",
		Placements: []string{"full_bleed"},
		Scaffold:   &catalog.ScaffoldBinding{Preset: "colonnade"},
		Inks: map[string]string{
			"$brand.primary": "#12327a", "$brand.background": "#efe7d3", "$brand.accent": "#c9432f",
		},
		PlateSpec: []catalog.PlateSpec{
			{Name: "distance", Depth: 0, Blend: catalog.BlendNormal, Opacity: 1},
			{Name: "stratosphere", Depth: 1, Blend: catalog.BlendNormal, Opacity: 1},
		},
	}
	_, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.Error(t, err)
	var unknown *UnknownPlaneError
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "stratosphere", unknown.Plane)
	require.Contains(t, unknown.Available, "arcade", "the refusal must name what the generator does draw")
}

// A generator that draws ONE layer cannot fill a stack, and says which layer it
// has rather than refusing anonymously.
//
// This replaces an earlier assertion that raster styles are refused outright.
// That was true before the raster generators were separated and is not true
// now: a horizon draws four layers and a terrain seven. What is still true —
// and what matters — is that a style cannot declare a stack its generator does
// not draw, whichever family it belongs to.
func TestAStyleCannotStackAGeneratorThatDrawsOneLayer(t *testing.T) {
	style := catalog.Style{
		ID: "flat-source-stack", Strategy: "procedural", Subject: "non_representational",
		Placements: []string{"full_bleed"},
		Scaffold:   &catalog.ScaffoldBinding{Preset: "mesh"},
		PlateSpec: []catalog.PlateSpec{
			{Name: "back", Depth: 0, Blend: catalog.BlendNormal, Opacity: 1},
			{Name: "front", Depth: 1, Blend: catalog.BlendNormal, Opacity: 1},
		},
	}
	_, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.Error(t, err)
	var unknown *UnknownPlaneError
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "back", unknown.Plane)
	require.Contains(t, unknown.Available, "scene",
		"a generator that names no planes draws one implicit layer, and the refusal must say so")
}

// A raster style can now declare a stack, and gets its generator's own layers.
//
// This is what Phase 5 refused by name and the raster plane separation
// supplies. The horizon is the case that matters most: a sky that can move
// against its sea is the parallax the flat lane could only suggest.
func TestARasterHorizonStyleShipsItsGeneratorsLayers(t *testing.T) {
	style := catalog.Style{
		ID: "layered-horizon", Strategy: "procedural", Subject: "horizon",
		Placements: []string{"full_bleed"},
		PlateSpec: []catalog.PlateSpec{
			{Name: "sky", Depth: 0, Blend: catalog.BlendNormal, Opacity: 1, Planes: []string{"sky", "sea"}},
			{Name: "headlands", Depth: 1, Blend: catalog.BlendNormal, Opacity: 1},
			{Name: "bank", Depth: 2, Blend: catalog.BlendNormal, Opacity: 1},
		},
	}
	job, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err)
	candidate := job.Candidates[0]

	require.Len(t, candidate.Plates, 3)
	require.Equal(t, "sky", candidate.Plates[0].Name)
	require.Equal(t, "headlands", candidate.Plates[1].Name)
	require.Equal(t, "bank", candidate.Plates[2].Name)
	for _, plate := range candidate.Plates {
		require.NotEmptyf(t, plate.PNG, "plate %q carries no pixels", plate.Name)
	}
	require.NotEqual(t, candidate.Plates[0].PNG, candidate.Plates[1].PNG)
	require.NotEmpty(t, candidate.PNG)
}

// A per-plate chain is not the same picture as the same chain applied once.
//
// This pins a measured defect rather than a desired behaviour, and it is here
// so nobody "fixes" it by lowering a floor. `normalize: true` maps the input's
// own tonal range onto the full ink ramp: over a whole scene that range spans
// sky to foreground, over a plate it spans only that layer. Each plate is
// therefore stretched against its own narrow range and the composite is three
// independently re-stretched bands — a different picture, legitimately
// different, but not the one the style declared.
//
// Flat scores subject_survival 0.990; the same chain declared per plate scores
// 0.640 against a 0.800 floor. The gate catching that is the system working.
//
// The repair is a normalization range that spans the stack, which needs an
// explicit range parameter on the image-tools operations. Until then, this test
// documents the constraint in the place someone will actually hit it.
func TestAPerPlateNormalizeChainIsNotTheSamePictureAsAFlatOne(t *testing.T) {
	base := catalog.Style{
		ID: "normalize-per-plate", Strategy: "procedural-treated", Subject: "horizon",
		Placements: []string{"full_bleed"}, Treatments: []string{"posterize"},
		TreatmentParams: map[string]string{
			"posterize": `{"levels":7,"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
		},
		Inks: map[string]string{"$brand.primary": "#14204d", "$brand.background": "#ffd166"},
	}
	store := NewStore(&fakeExecutor{})

	flat, err := store.SubmitWithContext(context.Background(), Request{
		Style: base, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err, "the flat style must still render; it is the control")
	require.Len(t, flat.Candidates[0].Plates, 1)

	plated := base
	plated.PlateSpec = []catalog.PlateSpec{
		{Name: "sky", Depth: 0, Blend: catalog.BlendNormal, Opacity: 1, Planes: []string{"sky", "sea"}},
		{Name: "headlands", Depth: 1, Blend: catalog.BlendNormal, Opacity: 1},
		{Name: "bank", Depth: 2, Blend: catalog.BlendNormal, Opacity: 1},
	}
	_, err = store.SubmitWithContext(context.Background(), Request{
		Style: plated, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.Error(t, err,
		"if this now passes, either the normalization range was fixed to span the stack — in which case delete this test and say so — "+
			"or a floor was lowered, which hides the distortion rather than removing it")
	require.Contains(t, err.Error(), "subject_survival",
		"the composite-level gate is what sees the distortion; a different failure means something else changed")
}

// The sweep gate's rule, tested where it lives: motion must not make a picture
// worse than it was standing still.
//
// It refuses a MOTION-INDUCED failure and deliberately does not refuse a style
// already illegible at rest. That is scope rather than leniency — measured over
// the settled catalog, zero of the twenty styles declaring an overlay region
// pass rest legibility, so enforcing rest here would refuse the whole catalog
// under the banner of a motion feature. The catalog-wide failure is recorded in
// PROBLEMS.md as its own defect.
func TestTheSweepGateRefusesAMotionInducedFailureAndOnlyThat(t *testing.T) {
	const w, h = 320, 320
	dark := color.NRGBA{R: 10, G: 12, B: 18, A: 255}
	bright := color.NRGBA{R: 250, G: 248, B: 240, A: 255}

	style := func(textColor string) catalog.Style {
		return catalog.Style{
			ID: "swept", ContrastThreshold: 4.5,
			Regions: []catalog.Region{
				{X: 0.05, Y: 0.05, Width: 0.5, Height: 0.25, Kind: "overlay", TextColor: textColor},
			},
		}
	}
	// A dark mass low in the frame, travelling far enough to reach the region.
	plates := []Plate{
		{
			Name: "ground", PNG: testPlatePNG(w, h, 0, 0, color.NRGBA{A: 0}), Opacity: 1,
			Motion: &catalog.MotionProfile{Parallax: 0.01},
		},
		{
			Name: "mass", PNG: testPlatePNG(w, h, 0.55, 1.0, dark), Opacity: 1,
			Motion: &catalog.MotionProfile{Parallax: 0.60},
		},
	}
	composite := testPlatePNG(w, h, 0, 1, bright)
	store := NewStore(&fakeExecutor{})

	// Dark type on bright paper: legible at rest, and the dark mass ruins it in
	// motion. That is the case this gate exists for.
	err := store.gateParallaxSweep(style("#111111"), composite, plates)
	require.Error(t, err)
	var swept *SweepRejectedError
	require.ErrorAs(t, err, &swept)
	require.Contains(t, swept.Error(), "scroll offset", "the refusal must name where in the sweep it failed")
	require.Contains(t, swept.Error(), "passes at rest")

	// Type that is already illegible standing still is NOT this gate's refusal.
	// It is a real defect and it is recorded as one; refusing it here would
	// blame motion for a picture that was never legible.
	require.NoError(t, store.gateParallaxSweep(style("#fbfbfb"), composite, plates),
		"a style illegible at rest is a catalog defect, not a motion-induced one")

	// And a stack with no depth has nothing to sweep.
	still := []Plate{
		{Name: "a", PNG: plates[0].PNG, Opacity: 1, Motion: &catalog.MotionProfile{Parallax: 0.2}},
		{Name: "b", PNG: plates[1].PNG, Opacity: 1, Motion: &catalog.MotionProfile{Parallax: 0.2}},
	}
	require.NoError(t, store.gateParallaxSweep(style("#111111"), composite, still))

	// A style with no overlay region has nothing to keep legible.
	require.NoError(t, store.gateParallaxSweep(catalog.Style{ID: "no-region"}, composite, plates))
}

// testPlatePNG builds a plate opaque between two vertical fractions.
func testPlatePNG(w, h int, top, bottom float64, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := int(top * float64(h)); y < int(bottom*float64(h)) && y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// A plated style with a legible reserved region passes the sweep and renders.
// Without this the test above would pass for a gate that refused everything.
func TestALegiblePlatedStylePassesTheSweep(t *testing.T) {
	style := catalog.Style{
		ID: "legible-colonnade", Strategy: "vector", Subject: "statuary_architecture",
		Placements: []string{"full_bleed"}, ContrastThreshold: 4.5,
		Scaffold: &catalog.ScaffoldBinding{Preset: "colonnade"},
		Inks: map[string]string{
			"$brand.primary": "#12327a", "$brand.background": "#efe7d3", "$brand.accent": "#c9432f",
		},
		PlateSpec: []catalog.PlateSpec{
			{
				Name: "distance", Depth: 0, Blend: catalog.BlendNormal, Opacity: 1, Planes: []string{"distance", "headland"},
				Motion: &catalog.MotionProfile{Parallax: 0.04},
			},
			{
				Name: "arcade", Depth: 1, Blend: catalog.BlendNormal, Opacity: 1,
				Motion: &catalog.MotionProfile{Parallax: 0.22},
			},
			{
				Name: "canopy", Depth: 2, Blend: catalog.BlendNormal, Opacity: 1,
				Motion: &catalog.MotionProfile{Parallax: 0.46},
			},
		},
	}
	job, err := NewStore(&fakeExecutor{}).SubmitWithContext(context.Background(), Request{
		Style: style, Surface: testSurface, Placement: "full_bleed", Seed: 7, Count: 1,
	})
	require.NoError(t, err, "a style declaring no overlay region has nothing to keep legible and must not be refused")
	require.Len(t, job.Candidates[0].Plates, 3)
	for _, plate := range job.Candidates[0].Plates {
		require.NotNilf(t, plate.Motion, "plate %q lost its motion profile on the way to the candidate", plate.Name)
	}
}

// A scrim's region is derived from the style's own reserved region rather than
// declared twice.
//
// An author repeating the rectangle in `regions` and again in
// `treatment_params.scrim` would eventually change one and not the other, and
// the failure — a scrim shading somewhere the copy no longer is — is invisible
// to every test that does not measure contrast where the text actually sits.
func TestAScrimTakesItsRegionFromTheStyle(t *testing.T) {
	style := catalog.Style{
		ID: "scrimmed", Treatments: []string{"scrim"},
		Regions: []catalog.Region{
			{X: 0.05, Y: 0.10, Width: 0.20, Height: 0.10, Kind: "overlay", TextColor: "#ffffff"},
			// The larger overlay region is the one a headline goes in.
			{X: 0.40, Y: 0.20, Width: 0.50, Height: 0.30, Kind: "overlay", TextColor: "#ffffff"},
			// A decorative region is not where copy sits and must not win.
			{X: 0.00, Y: 0.00, Width: 1.00, Height: 1.00, Kind: "decorative"},
		},
	}
	got := scrimParams(style, map[string]string{})
	require.Contains(t, got["scrim"], `"region_width":0.5`, "the largest OVERLAY region is the one shaded")
	require.Contains(t, got["scrim"], `"region_x":0.4`)

	// An explicit region wins: a pool deliberately wider than the copy is a
	// real art-direction choice.
	explicit := scrimParams(style, map[string]string{"scrim": `{"region_x":0,"region_y":0,"region_width":1,"region_height":1}`})
	require.Contains(t, explicit["scrim"], `"region_width":1`)

	// A style that does not run a scrim gets its parameters back untouched.
	quiet := catalog.Style{ID: "quiet", Regions: style.Regions}
	require.Empty(t, scrimParams(quiet, map[string]string{})["scrim"])
}

// TestEveryDeclaredPlaneReachesAPlate refuses a plate spec that drops part of
// the picture.
//
// A generator declares its depth planes and a style declares how those planes
// group into plates. Nothing connected the two: a style could name three plates
// for a generator that draws four, and the fourth was simply never composited.
// The flat document still contained it — the flatten-equivalence proof compares
// the generator's own planes against the generator's own flat render, and both
// of those are complete — so the defect was invisible to every existing test
// while the DELIVERED image was missing a whole layer.
//
// It shipped that way. `pale-moon` was delivered without its ground and
// `tidal-halftone` without its sea, which is the subject of the picture. It was
// found by a legibility measurement reading a reserved region as uniform blank
// paper, because the sea that should have filled it was not there.
//
// Merging is the accommodation, not omission: `maxPlates` is 3 and `colonnade`
// draws four planes by carrying two of them on one plate. This asserts coverage,
// never plate count.
func TestEveryDeclaredPlaneReachesAPlate(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(catalog.Schema())
	require.NoError(t, err)
	store := catalog.NewStore(db)
	ctx := context.Background()
	require.NoError(t, store.Seed(ctx))
	styles, err := store.ListStyles(ctx, "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	for _, style := range styles {
		if style.Strategy != "vector" {
			continue
		}
		spec := style.EffectivePlateSpec()
		if len(spec) <= 1 {
			continue
		}
		t.Run(style.ID, func(t *testing.T) {
			declared := scaffoldPreset(style)
			if _, builtin := vector.SubjectOf(declared); declared != "" && !builtin {
				// An authored generator draws one plane by construction and has
				// no stack to cover.
				t.Skipf("style %q uses an authored generator", style.ID)
			}
			preset, err := vector.ResolvePreset(style.Subject, declared)
			require.NoError(t, err)
			drawn, err := vector.Render(vector.Request{
				Preset: preset, Width: 320, Height: 200, Seed: 3,
				ParamsJSON: scaffoldParams(style), Inks: style.EffectivePalette(nil),
			})
			require.NoError(t, err)

			carried := map[string]bool{}
			for _, plate := range spec {
				for _, plane := range plate.SourcePlanes() {
					carried[plane] = true
				}
			}
			var dropped []string
			for _, plane := range drawn.Planes {
				if !carried[plane] {
					dropped = append(dropped, plane)
				}
			}
			require.Emptyf(t, dropped,
				"style %q draws %v but its plates carry %v: plane(s) %v never reach the delivered image",
				style.ID, drawn.Planes, carried, dropped)
		})
	}
}
