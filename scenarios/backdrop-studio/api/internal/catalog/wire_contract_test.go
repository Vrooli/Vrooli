package catalog_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"

	"github.com/stretchr/testify/require"
	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
)

func seededStore(t *testing.T) *catalog.Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(catalog.Schema())
	require.NoError(t, err)
	store := catalog.NewStore(db)
	require.NoError(t, store.Seed(t.Context()))
	return store
}

// TestSeededCatalogParamsParseAsImageToolsOpParams is the cross-scenario
// contract gate.
//
// On 2026-08-12 the catalog was seeded with styles requesting "normalize" and
// brand inks on the Tier-2 screens, neither of which existed on image-tools'
// proto messages. protojson rejects unknown fields, so eleven of sixteen styles
// would have failed their render with a 400 — and every unit suite stayed green,
// because backdrop-studio tests against a fake executor that never touches the
// REST edge and image-tools tests its treatments below the wire.
//
// It is resolved BOTH ways on purpose. The earlier version of this test bound a
// real brand, which is the one path a CLI caller never takes — so it watched
// ten styles ship broken. The unbound case is the one that matters most.
func TestSeededCatalogParamsParseAsImageToolsOpParams(t *testing.T) {
	store := seededStore(t)
	styles, err := store.ListStyles(t.Context(), "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	cases := []struct {
		name  string
		brand map[string]string
	}{
		{name: "no brand bound", brand: nil},
		{name: "brand bound", brand: map[string]string{
			"$brand.primary":    "#1B3FD8",
			"$brand.secondary":  "#0F2A6B",
			"$brand.accent":     "#F5A623",
			"$brand.background": "#EDE6D2",
			"$brand.surface":    "#FFFFFF",
			"$brand.text":       "#102A43",
			"$brand.error":      "#B00020",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checked := 0
			for _, style := range styles {
				for _, op := range style.Treatments {
					// The effective palette is what the render path actually
					// sends: the style's declared ink defaults, overlaid by the
					// bound brand.
					raw, resolveErr := imageengine.ResolveParams(op, style.TreatmentParams[op], style.EffectivePalette(tc.brand), nil)
					require.NoErrorf(t, resolveErr, "style %q op %q failed to resolve", style.ID, op)

					pb := &opsv1.OpParams{}
					require.NoErrorf(t, protojson.Unmarshal([]byte(raw), pb),
						"style %q op %q emits params image-tools will reject:\n%s", style.ID, op, raw)

					// An unresolved slot means the ink never bound and the
					// render would carry a literal "$brand.primary" onto the
					// wire, which image-tools answers with 422.
					require.NotContainsf(t, raw, "$brand.",
						"style %q op %q left an unresolved brand slot: %s", style.ID, op, raw)
					checked++
				}
			}
			require.Greater(t, checked, 20, "expected the seeded catalog to exercise a real spread of operations")
		})
	}
}

// absoluteSpatialFields are the pixel-denominated parameters image-tools still
// accepts for compatibility. A seeded style must not send one: a value in
// pixels ties the style to a single delivery surface, and this catalog renders
// the same style at geometries from 390px to 2732px on the short edge.
var absoluteSpatialFields = map[string][]string{
	"line_screen":  {"spacing"},
	"stipple":      {"spacing"},
	"engraving":    {"spacing"},
	"ascii_mosaic": {"block_size"},
	"displacement": {"spacing", "amplitude"},
	"aberration":   {"distance", "amplitude"},
	"bloom":        {"radius"},
	"defocus":      {"radius"},
	"motion_blur":  {"distance"},
}

// TestSeededStylesSendNoAbsoluteSpatialParameter enforces the Phase 4 rule at
// the catalog boundary, where it is cheap to hold.
//
// Every value here reaches image-tools, which happily accepts both forms — so
// nothing downstream can tell an intentional pixel value from an un-migrated
// one. The catalog is the only place that knows a seeded style is meant to work
// at every surface, so the catalog is where the rule lives.
func TestSeededStylesSendNoAbsoluteSpatialParameter(t *testing.T) {
	store := seededStore(t)
	styles, err := store.ListStyles(t.Context(), "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	checked := 0
	for _, style := range styles {
		for _, op := range style.Treatments {
			fields, spatial := absoluteSpatialFields[op]
			if !spatial {
				continue
			}
			raw, resolveErr := imageengine.ResolveParams(op, style.TreatmentParams[op], style.EffectivePalette(nil), nil)
			require.NoError(t, resolveErr)

			var params map[string]map[string]any
			require.NoError(t, json.Unmarshal([]byte(raw), &params))
			for _, field := range fields {
				require.NotContainsf(t, params[op], field,
					"style %q op %q sends %q in pixels; use the relative form so the style holds its look at every surface\n%s",
					style.ID, op, field, raw)
			}
			require.NotEmptyf(t, params[op], "style %q op %q resolved to no parameters at all", style.ID, op)
			checked++
		}
	}
	require.Positive(t, checked, "no seeded style exercises a spatial treatment; this test would pass vacuously")
}

// TestSeededSpatialParametersResolveProportionally is the end of the wire: it
// takes each seeded style's declared relative value through the same conversion
// image-tools performs, at the smallest and largest surfaces this catalog
// delivers, and asserts the pixel result tracks the frame.
func TestSeededSpatialParametersResolveProportionally(t *testing.T) {
	store := seededStore(t)
	styles, err := store.ListStyles(t.Context(), "", "", "", "", "")
	require.NoError(t, err)
	surfaces, err := store.ListSurfaces(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, surfaces)

	// The range a style must hold across is the range of surfaces it can
	// actually be delivered to, which is decided by placement.
	//
	// This used to take the smallest and largest short edge over every surface
	// in the catalog. That was right while the surface set was narrow and
	// became wrong the moment an email header at 600x240 was seeded: it held
	// `stipple-massif` — a style that permits only `framed_inset` and
	// `corner_bleed`, neither of which an email header accepts — to a size it
	// can never be asked to render at, and would have forced a coarser screen
	// on every surface it does serve to satisfy one it does not.
	deliverySpan := func(style catalog.Style) (shortest, longest int) {
		shortest, longest = math.MaxInt, 0
		for _, s := range surfaces {
			if !anyShared(style.Placements, s.Placements) {
				continue
			}
			shortEdge := min(s.Width, s.Height)
			shortest = min(shortest, shortEdge)
			longest = max(longest, shortEdge)
		}
		return shortest, longest
	}

	// The catalog as a whole still has to span a real range, or the whole
	// assertion is vacuous however carefully it is scoped.
	widest, narrowest := 0, math.MaxInt
	for _, s := range surfaces {
		narrowest = min(narrowest, min(s.Width, s.Height))
		widest = max(widest, min(s.Width, s.Height))
	}
	require.Greater(t, widest, narrowest*2, "the catalog must span a real range of geometries for this to mean anything")

	checked := 0
	for _, style := range styles {
		shortest, longest := deliverySpan(style)
		require.NotEqualf(t, math.MaxInt, shortest,
			"style %q declares placements no seeded surface permits, so it can never be delivered", style.ID)
		for _, op := range style.Treatments {
			if _, spatial := absoluteSpatialFields[op]; !spatial {
				continue
			}
			raw, resolveErr := imageengine.ResolveParams(op, style.TreatmentParams[op], style.EffectivePalette(nil), nil)
			require.NoError(t, resolveErr)
			var params map[string]map[string]any
			require.NoError(t, json.Unmarshal([]byte(raw), &params))

			for field, value := range params[op] {
				if !strings.HasSuffix(field, "_rel") {
					continue
				}
				rel, ok := value.(float64)
				require.Truef(t, ok, "style %q op %q: %s is not a number", style.ID, op, field)
				require.Positivef(t, rel, "style %q op %q: %s must be a positive fraction", style.ID, op, field)
				small, large := rel*float64(shortest), rel*float64(longest)
				require.GreaterOrEqualf(t, small, 3.0,
					"style %q op %q: %s resolves to %.1fpx at the %dpx short edge, under the floor where the treatment discards it",
					style.ID, op, field, small, shortest)
				require.InDeltaf(t, float64(longest)/float64(shortest), large/small, 1e-9,
					"style %q op %q: %s must scale with the frame", style.ID, op, field)
				checked++
			}
		}
	}
	require.Positive(t, checked)
}

// TestBrandBindingChangesRenderedInks proves the palette is load-bearing rather
// than decorative: a style must render differently for two brands, or the
// "$brand.*" indirection is costing complexity and buying nothing.
func TestBrandBindingChangesRenderedInks(t *testing.T) {
	store := seededStore(t)
	style, err := store.GetStyle(t.Context(), "cyanotype-arcade")
	require.NoError(t, err)

	unbound, err := imageengine.ResolveParams("duotone", style.TreatmentParams["duotone"], style.EffectivePalette(nil), nil)
	require.NoError(t, err)
	acme, err := imageengine.ResolveParams("duotone", style.TreatmentParams["duotone"], style.EffectivePalette(map[string]string{"$brand.primary": "#B00020", "$brand.background": "#FFF8E7"}), nil)
	require.NoError(t, err)

	require.NotEqual(t, unbound, acme, "a bound brand must change the inks that reach the wire")
	require.Contains(t, acme, "#B00020")
	require.Contains(t, unbound, style.Inks["$brand.primary"])
}

// TestEverySeededSlotHasADeclaredDefault is the write-side half of the
// fail-closed contract. Phase 1 step 8 asks for the slot-to-token mapping to be
// written down and enforced; this is the enforcement. A style that references a
// slot it declares no default for cannot render on a cold install, which is
// exactly the failure that made ten styles unrenderable.
func TestEverySeededSlotHasADeclaredDefault(t *testing.T) {
	store := seededStore(t)
	styles, err := store.ListStyles(t.Context(), "", "", "", "", "")
	require.NoError(t, err)

	known := map[string]bool{}
	for _, slot := range catalog.BrandSlots {
		known[slot] = true
	}
	for _, style := range styles {
		for slot := range style.Inks {
			require.Truef(t, known[slot], "style %q declares ink for slot %q, which brand-manager does not emit", style.ID, slot)
		}
		for op, params := range style.TreatmentParams {
			for _, slot := range catalog.BrandSlots {
				if !strings.Contains(params, slot) {
					continue
				}
				require.NotEmptyf(t, style.Inks[slot],
					"style %q op %q references %s with no declared default; it could not render without a brand", style.ID, op, slot)
			}
		}
	}
}

// anyShared reports whether two placement lists intersect, which is the test
// for "can this style be delivered to this surface".
func anyShared(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

// perceptualGridColumns is the width of the grid the perceptual gate samples a
// composition on. It is duplicated from internal/perceptual deliberately: the
// point of this test is that a *catalog* value must respect a *gate* property,
// and importing the gate here would let a change to one silently satisfy the
// other.
const perceptualGridColumns = 64

// TestSeededScreensResolveFinerThanTheGateSamples is a measured constraint, not
// a style preference.
//
// A halftone's `lpi` is a count of screen lines across the image width, so its
// cell is width/lpi. The perceptual gate reduces a composition to a 64-column
// field before correlating it, so its cell is width/64. Once the screen cell is
// the larger of the two, the gate is measuring the screen instead of the
// picture — and so is a viewer, because the composition really has been
// replaced by a pattern at that scale.
//
// Measured on one caustic source across rulings, subject survival runs:
//
//	lpi  34 → 0.386
//	lpi  48 → 0.638
//	lpi  64 → 0.924
//	lpi  96 → 0.891
//
// The knee sits exactly where the two cells cross. Two styles were retuned
// below it on the theory that a coarser screen would read more calmly over a
// busy source; the gate refused both, correctly.
func TestSeededScreensResolveFinerThanTheGateSamples(t *testing.T) {
	store := seededStore(t)
	styles, err := store.ListStyles(t.Context(), "", "", "", "", "")
	require.NoError(t, err)

	checked := 0
	for _, style := range styles {
		raw, ok := style.TreatmentParams["halftone"]
		if !ok {
			continue
		}
		var params struct {
			LPI int `json:"lpi"`
		}
		require.NoErrorf(t, json.Unmarshal([]byte(raw), &params), "style %q halftone params are not an object: %s", style.ID, raw)
		if params.LPI == 0 {
			continue // the operation's own default, which is fine enough
		}
		require.GreaterOrEqualf(t, params.LPI, perceptualGridColumns,
			"style %q screens at %d lines across the width, which is coarser than the %d-column grid the perceptual gate samples on. "+
				"At that ruling the screen replaces the composition rather than reproducing it. Raise the ruling, or give the source stronger large-scale structure and raise it anyway.",
			style.ID, params.LPI, perceptualGridColumns)
		checked++
	}
	require.Positive(t, checked, "no seeded style declares a halftone ruling; this test would pass vacuously")
}

// The same rule, applied per plate.
//
// A plate declaring its own halftone is a screen over one depth layer, and the
// gate now scores each plate against its own source — so a ruling coarser than
// the gate's grid replaces that layer's composition exactly as it would replace
// a whole frame's. The style-level rule above would not see it: a plate chain
// lives in the plate spec, not in TreatmentParams.
//
// It runs over the settled catalog rather than a fixture so it covers whatever
// the seed actually declares, and it says so when nothing does — a rule that
// passes vacuously is worse than no rule, because it reads as coverage.
func TestSeededPlateScreensResolveFinerThanTheGateSamples(t *testing.T) {
	store := seededStore(t)
	styles, err := store.ListStyles(t.Context(), "", "", "", "", "")
	require.NoError(t, err)

	checked, plated := 0, 0
	for _, style := range styles {
		if len(style.PlateSpec) < 2 {
			continue
		}
		plated++
		for _, plate := range style.EffectivePlateSpec() {
			screens := false
			for _, treatment := range plate.Treatments {
				if treatment == "halftone" {
					screens = true
				}
			}
			if !screens {
				continue
			}
			raw, ok := style.TreatmentParams["halftone"]
			if !ok {
				continue // the operation's own default, which is fine enough
			}
			var params struct {
				LPI int `json:"lpi"`
			}
			require.NoErrorf(t, json.Unmarshal([]byte(raw), &params), "style %q halftone params are not an object: %s", style.ID, raw)
			if params.LPI == 0 {
				continue
			}
			require.GreaterOrEqualf(t, params.LPI, perceptualGridColumns,
				"style %q plate %q screens at %d lines across the width, coarser than the %d-column grid the gate samples on. "+
					"At that ruling the screen replaces that layer's composition rather than reproducing it.",
				style.ID, plate.Name, params.LPI, perceptualGridColumns)
			checked++
		}
	}
	require.Positive(t, plated, "no seeded style declares a plate stack; this rule would pass vacuously")
	if checked == 0 {
		t.Logf("%d plated style(s) and no plate declares a halftone ruling: the vector styles carry tone as line density and are deliberately unscreened", plated)
	}
}

// A plate that declares no chain inherits the style's.
//
// Without the default, adding a plate spec to a treated style would silently
// strip the treatment from every plate: the style's own chain would apply to
// nothing, and a screened style would ship as an untreated one. An explicit
// empty list still means "no treatment on this plate", which is the case a
// nil-means-inherit rule has to keep expressible.
func TestAPlateInheritsTheStylesChainUnlessItDeclaresOne(t *testing.T) {
	style := catalog.Style{
		ID: "x", Treatments: []string{"halftone", "grain"},
		PlateSpec: []catalog.PlateSpec{
			{Name: "sky", Depth: 0, Opacity: 1},
			{Name: "sea", Depth: 1, Opacity: 1, Treatments: []string{"stipple"}},
			{Name: "shore", Depth: 2, Opacity: 1, Treatments: []string{}},
		},
	}
	spec := style.EffectivePlateSpec()
	require.Equal(t, []string{"halftone", "grain"}, spec[0].Treatments, "a plate with no chain inherits the style's")
	require.Equal(t, []string{"stipple"}, spec[1].Treatments, "a declared chain wins")
	require.Empty(t, spec[2].Treatments, "an explicit empty chain means no treatment, not inherit")
}
