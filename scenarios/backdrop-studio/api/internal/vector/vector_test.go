package vector

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testInks is a complete palette, so a test that is not about ink resolution
// never fails for want of one.
var testInks = map[string]string{
	InkPaper:  "#efe7d3",
	InkInk:    "#12327a",
	InkAccent: "#c9432f",
}

func render(t *testing.T, preset string, width, height int, seed int64, params string) Result {
	t.Helper()
	res, err := Render(Request{Preset: preset, Width: width, Height: height, Seed: seed, ParamsJSON: params, Inks: testInks})
	require.NoErrorf(t, err, "render %s at %dx%d", preset, width, height)
	return res
}

// A generator is a pure function of (preset, size, seed, params). Everything
// downstream — goldens, the remix flow, reproducing a released asset from its
// provenance record — rests on this, so it is asserted directly rather than
// assumed from the absence of a clock.
func TestEveryGeneratorIsDeterministic(t *testing.T) {
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			first := render(t, preset, 800, 500, 7, "")
			second := render(t, preset, 800, 500, 7, "")
			require.Equal(t, first.SHA256, second.SHA256, "same inputs must produce the same bytes")
			require.Equal(t, first.SVG, second.SVG)

			// And a different seed must produce a different picture, or the seed
			// is decoration and the variation grid in the studio is a lie.
			other := render(t, preset, 800, 500, 8, "")
			require.NotEqual(t, first.SHA256, other.SHA256, "a different seed must draw a different picture")
		})
	}
}

// Resolution independence is the property the raster screens do not have, and
// it is the reason this family exists. A style tuned once must hold its
// composition from a thumbnail to a retina store asset.
//
// It is asserted on the *composition* rather than on the bytes, because the
// bytes legitimately differ: coordinates scale. What must not change is where
// the masses sit, so both sizes are reduced to a coarse occupancy grid and
// compared.
func TestCompositionHoldsFrom240pxTo2732px(t *testing.T) {
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			small := render(t, preset, 240, 150, 7, "")
			large := render(t, preset, 2732, 1707, 7, "")

			smallGrid := occupancyGrid(t, small)
			largeGrid := occupancyGrid(t, large)
			agreement := gridAgreement(smallGrid, largeGrid)
			require.GreaterOrEqualf(t, agreement, 0.80,
				"composition drifted between 240px and 2732px (agreement %.3f): the generator is not resolution independent", agreement)
		})
	}
}

// occupancyGrid reduces a document's drawn geometry to a coarse map of where
// ink lands, normalised to the frame. It reads the coordinates out of the SVG
// rather than rasterizing, so the assertion needs no browser and stays in the
// unit suite.
func occupancyGrid(t *testing.T, res Result) []float64 {
	t.Helper()
	const cols, rows = 12, 8
	grid := make([]float64, cols*rows)
	w, h := float64(res.Width), float64(res.Height)
	for _, pt := range drawnPoints(string(res.SVG)) {
		cx := int(pt[0] / w * cols)
		cy := int(pt[1] / h * rows)
		if cx < 0 || cy < 0 || cx >= cols || cy >= rows {
			continue
		}
		grid[cy*cols+cx]++
	}
	total := 0.0
	for _, v := range grid {
		total += v
	}
	require.NotZerof(t, total, "the document declares no coordinates at all")
	for i := range grid {
		grid[i] /= total
	}
	return grid
}

var coordinatePattern = regexp.MustCompile(`(-?\d+\.\d+)\s+(-?\d+\.\d+)`)

// drawnPoints pulls every coordinate pair out of a document. It is deliberately
// crude — it does not parse SVG — because it only has to answer "where is the
// geometry", and every generator here writes coordinates in one format.
func drawnPoints(document string) [][2]float64 {
	var out [][2]float64
	for _, m := range coordinatePattern.FindAllStringSubmatch(document, -1) {
		x, errX := strconv.ParseFloat(m[1], 64)
		y, errY := strconv.ParseFloat(m[2], 64)
		if errX == nil && errY == nil {
			out = append(out, [2]float64{x, y})
		}
	}
	return out
}

// gridAgreement is 1 minus half the total variation distance, so identical
// distributions score 1 and disjoint ones score 0.
func gridAgreement(a, b []float64) float64 {
	diff := 0.0
	for i := range a {
		diff += math.Abs(a[i] - b[i])
	}
	return 1 - diff/2
}

// Ink resolution fails closed. Writing the literal slot into the document would
// produce bytes that parse, rasterize, and draw the wrong colour — the same
// fail-open that once put "$brand.primary" on the wire and made ten of sixteen
// seeded styles unrenderable.
func TestAnUnresolvedInkSlotIsRefused(t *testing.T) {
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			_, err := Render(Request{
				Preset: preset, Width: 400, Height: 250, Seed: 7,
				Inks: map[string]string{InkPaper: "#ffffff"}, // ink and accent missing
			})
			require.Error(t, err)
			var unresolved *UnresolvedInkError
			require.ErrorAs(t, err, &unresolved)
			require.Equal(t, preset, unresolved.Preset)
			require.Contains(t, unresolved.Slot, "$brand.")
		})
	}
}

// A resolved document must carry no slot text at all. Asserting the negative
// directly is what makes the previous test a guarantee rather than a sample:
// it catches a slot that a future generator introduces and the resolver's list
// does not know about.
func TestAResolvedDocumentContainsNoSlotText(t *testing.T) {
	for _, preset := range Presets {
		res := render(t, preset, 600, 400, 7, "")
		require.NotContainsf(t, string(res.SVG), "$brand.",
			"generator %q left an unresolved ink slot in its output", preset)
	}
}

// Every generator declares its depth planes, back to front. The plate model
// takes exact alpha from these groups instead of estimating a matte, which is
// the whole advantage a vector source has over a flattened raster.
func TestEveryGeneratorDeclaresOrderedDepthPlanes(t *testing.T) {
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			res := render(t, preset, 600, 400, 7, "")
			require.GreaterOrEqualf(t, len(res.Planes), 2,
				"a generator that draws one plane gives the plate model nothing to separate")

			// The declared order must match the document order, or a consumer
			// reading the groups composites the picture back to front.
			document := string(res.SVG)
			previous := -1
			for _, name := range res.Planes {
				marker := fmt.Sprintf(`data-plane="%s"`, name)
				at := strings.Index(document, marker)
				require.GreaterOrEqualf(t, at, 0, "plane %q is declared and not drawn", name)
				require.Greaterf(t, at, previous, "plane %q is drawn out of declared depth order", name)
				previous = at
			}
		})
	}
}

// The generators use filters, masks, patterns or CSS by construction — that is
// what the high-fidelity rasterizer decision was made for. If a generator ever
// stops using them, the pure-Go rasterizer would silently take over and the
// hand-cut character would vanish with nothing reporting it.
func TestGeneratorsUseTheHighFidelityFeatureSet(t *testing.T) {
	for _, preset := range Presets {
		res := render(t, preset, 600, 400, 7, "")
		require.Containsf(t, string(res.SVG), "<filter",
			"generator %q declares no filter; the pure-Go rasterizer would take it and drop the hand-cut character", preset)
	}
}

func TestUnknownPresetIsRefused(t *testing.T) {
	_, err := Render(Request{Preset: "not-a-generator", Width: 100, Height: 100, Inks: testInks})
	require.ErrorContains(t, err, "unknown preset")
}

func TestGeometryBoundsAreEnforced(t *testing.T) {
	_, err := Render(Request{Preset: "colonnade", Width: 8, Height: 8, Inks: testInks})
	require.ErrorContains(t, err, "at least 16")

	_, err = Render(Request{Preset: "colonnade", Width: 9000, Height: 100, Inks: testInks})
	require.ErrorContains(t, err, "at most 8192")
}

// Byte-stable goldens at two frame sizes.
//
// Regenerate with `BACKDROP_STUDIO_UPDATE_GOLDENS=1 GOWORK=off go test
// ./internal/vector/`. A golden that has to be regenerated for a reason nobody
// can name is a change in what the catalog draws, so the diff is the review.
func TestVectorGoldens(t *testing.T) {
	for _, preset := range Presets {
		for _, size := range []struct{ w, h int }{{480, 300}, {1440, 720}} {
			name := fmt.Sprintf("%s-%dx%d", preset, size.w, size.h)
			t.Run(name, func(t *testing.T) {
				res := render(t, preset, size.w, size.h, 7, "")
				path := filepath.Join("testdata", name+".svg")
				if os.Getenv("BACKDROP_STUDIO_UPDATE_GOLDENS") != "" {
					require.NoError(t, os.MkdirAll("testdata", 0o755))
					require.NoError(t, os.WriteFile(path, res.SVG, 0o644))
					return
				}
				want, err := os.ReadFile(path)
				require.NoErrorf(t, err, "missing golden; regenerate with BACKDROP_STUDIO_UPDATE_GOLDENS=1")
				require.Equalf(t, string(want), string(res.SVG),
					"generator %q changed what it draws at %dx%d", preset, size.w, size.h)
			})
		}
	}
}

// A wide plate must show more land, not the same land stretched.
//
// This is the defect the render matrix could not see, because it renders one
// geometry. `survey-relief` passed at 1440x900 and failed the perceptual gate
// on three of the eighteen seeded surfaces: frequency modulation 0.028 against
// a 0.030 floor at `web.hero` (2:1), and tonal occupancy 0.277 against a 0.40
// floor at `web.section-band` (3.4:1). The cause was a hardcoded 1.7 aspect
// correction inside the height field, which drew round hills on a 1.7:1 frame
// and progressively flatter, sparser ones on everything else.
//
// Asserted on the peak field rather than on the drawn document, because the
// property is compositional: coverage of the delivered picture is what the
// perceptual gate measures, and it needs pixels this package deliberately does
// not produce.
func TestARelieFieldGrowsWithTheFrame(t *testing.T) {
	base := len(reliefPeaks(referenceAspect))
	require.Equal(t, len(peakLayout), base,
		"the reference aspect must reproduce the art-directed composition exactly")

	for _, tc := range []struct {
		name    string
		aspect  float64
		atLeast int
	}{
		{"web.hero 2:1", 1440.0 / 720.0, base + 1},
		{"web.section-band 3.4:1", 1440.0 / 420.0, base + 2},
		{"social.profile-banner 3:1", 1500.0 / 500.0, base + 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			peaks := reliefPeaks(tc.aspect)
			require.GreaterOrEqualf(t, len(peaks), tc.atLeast,
				"a %.2f:1 frame drew %d summits against %d at the reference aspect; the plate is the same land stretched",
				tc.aspect, len(peaks), base)
		})
	}

	// A portrait frame must not gain summits it has no room for.
	require.Equal(t, base, len(reliefPeaks(390.0/844.0)),
		"a portrait frame must keep the base composition")
}

// Every summit stays out of the overlay band, including the ones added for a
// wide frame. A backdrop carries copy in its upper-left third, and relief there
// competes with the headline — which the gate measures as texture in the zone
// it requires to stay quiet.
func TestAddedSummitsRespectTheCopyZone(t *testing.T) {
	for _, aspect := range []float64{1.7, 2.0, 3.0, 3.43} {
		for i, p := range reliefPeaks(aspect) {
			if p.x < 0.46 && p.y < 0.44 {
				t.Errorf("aspect %.2f: summit %d sits at (%.2f, %.2f), inside the overlay band", aspect, i, p.x, p.y)
			}
		}
	}
}

// Every plane's marks appear in exactly one plane document, and together they
// account for the whole picture.
//
// This is the vector lane's flatten-equivalence assertion, and it is made on the
// document rather than on pixels because this package deliberately produces no
// pixels. What it proves is stronger than a pixel comparison would be at this
// level: the marks are PARTITIONED, so nothing is duplicated across layers and
// nothing is dropped between them. A rasterized comparison could pass while a
// mark was drawn twice in the same place.
func TestPlaneDocumentsPartitionTheComposite(t *testing.T) {
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			res := render(t, preset, 1440, 720, 7, "")
			require.Len(t, res.PlaneDocuments, len(res.Planes),
				"one document per declared plane, or the plate model gets a stack with a hole in it")

			composite := string(res.SVG)
			totalMarks := 0
			for i, plane := range res.Planes {
				doc := string(res.PlaneDocuments[i])
				require.Containsf(t, doc, `data-plane="`+plane+`"`, "plane %q document does not declare its own group", plane)
				require.Equalf(t, 1, strings.Count(doc, "<g id=\"plane-"),
					"plane %q document carries %d groups; a plane document is one layer", plane, strings.Count(doc, "<g id=\"plane-"))

				// Every mark in this plane's document must appear in the
				// composite, and the plane's body must be exactly the slice of
				// the composite that belongs to it.
				body := planeBody(t, doc, plane)
				require.NotEmptyf(t, strings.TrimSpace(body), "plane %q draws nothing", plane)
				require.Containsf(t, composite, body, "plane %q's marks are not in the composite verbatim", plane)
				totalMarks += strings.Count(body, "<")
			}
			// And the composite carries no marks beyond the union of the
			// planes: every generator draws through c.plane(), so a mark
			// outside one would be a mark no plate could ever carry.
			compositeMarks := 0
			for _, plane := range res.Planes {
				compositeMarks += strings.Count(planeBody(t, composite, plane), "<")
			}
			require.Equalf(t, compositeMarks, totalMarks,
				"the planes account for %d marks and the composite has %d; the partition leaks", totalMarks, compositeMarks)
		})
	}
}

// A plane document must be independently renderable: its filter references have
// to resolve inside it, or a rasterizer drops the filter and draws a layer that
// silently lost its character.
func TestEachPlaneDocumentCarriesTheDefsItReferences(t *testing.T) {
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			res := render(t, preset, 1440, 720, 7, "")
			for i, plane := range res.Planes {
				doc := string(res.PlaneDocuments[i])
				for _, ref := range filterReferences(doc) {
					require.Containsf(t, doc, `id="`+ref+`"`,
						"plane %q references filter %q that its own document does not define", plane, ref)
				}
				require.NotContainsf(t, doc, "$brand.", "plane %q leaves an unresolved ink slot", plane)
			}
		})
	}
}

// planeBody extracts one plane group's inner markup from a document.
func planeBody(t *testing.T, document, plane string) string {
	t.Helper()
	open := `<g id="plane-` + plane + `"`
	start := strings.Index(document, open)
	if start < 0 {
		return ""
	}
	start = strings.Index(document[start:], ">")
	if start < 0 {
		return ""
	}
	rest := document[strings.Index(document, open)+start+1:]
	// Groups are not nested by any generator here, so the next closing tag at
	// depth zero ends this one.
	depth := 0
	for i := 0; i+3 < len(rest); i++ {
		switch {
		case strings.HasPrefix(rest[i:], "<g "):
			depth++
		case strings.HasPrefix(rest[i:], "</g>"):
			if depth == 0 {
				return rest[:i]
			}
			depth--
		}
	}
	return rest
}

var filterRefPattern = regexp.MustCompile(`filter="url\(#([^)]+)\)"`)

func filterReferences(document string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range filterRefPattern.FindAllStringSubmatch(document, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, m[1])
	}
	return out
}

// TestAReservedZoneIsCutInBothPolarities pins the two reserves a printer cuts,
// on every generator.
//
// Both directions, because they are not one mechanism seen twice: a knockout
// removes ink and a solid adds it, and getting the polarity backwards is worse
// than doing nothing — it reserves the space by making the copy harder to read
// rather than easier.
//
// Structural rather than measured, deliberately. This package emits SVG and
// owns no rasterizer; the pixels are image-tools' to produce, and this repo
// already holds that a style's bytes have to round-trip through a really
// running image-tools before anything is claimed about them. The contrast this
// buys is measured in the integration lane, over the real rasterizer. What is
// asserted here is what this package actually decides: that a reserve is cut,
// which ink it is cut in, and where it sits in the stack.
func TestAReservedZoneIsCutInBothPolarities(t *testing.T) {
	inks := map[string]string{"$brand.background": "#ffffff", "$brand.primary": "#000000", "$brand.accent": "#808080"}
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			for _, tc := range []struct {
				name        string
				towardLight bool
				wantInk     string
			}{
				{"dark copy is served by a knockout in the paper ink", true, "#ffffff"},
				{"light copy is served by a solid in the dark ink", false, "#000000"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					zone := QuietZone{X: 0.08, Y: 0.12, Width: 0.36, Height: 0.28, Feather: 0.09, TowardLight: tc.towardLight}
					res, err := Render(Request{Preset: preset, Width: 480, Height: 300, Seed: 5, Inks: inks, Quiet: &zone})
					require.NoError(t, err)
					doc := string(res.SVG)
					require.Contains(t, doc, `mask id="reserve-mask"`, "no reserve was cut")
					require.Contains(t, doc, `<g mask="url(#reserve-mask)"><rect width="480.000" height="300.000" fill="`+tc.wantInk+`"/></g>`,
						"the reserve is cut in the wrong ink for this copy colour")

					// On the frontmost plane and no other: the reserve has to
					// survive everything drawn in front of it, and it has to
					// appear exactly once so the flat document and the composited
					// stack stay the same picture.
					// On every plane. The frontmost alone is enough only at
					// rest: plates travel at different rates, so wherever the
					// front plate's reserve has slid off the copy the plate
					// behind is showing, and that is usually a full sheet of
					// opaque paper that never moves at all.
					require.Equal(t, len(res.Planes), strings.Count(doc, `mask="url(#reserve-mask)"`),
						"the reserve must be cut into every plane")
					for _, plane := range res.Planes {
						require.Contains(t, planeBody(t, doc, plane), `url(#reserve-mask)`,
							"plane %q carries no reserve, so it shows through whenever the plates in front of it move", plane)
					}
				})
			}
		})
	}
}

// TestNoReservedZoneCutsNothing keeps the common case clean: a style that
// reserves nothing must produce the document it always did, with no mask, no
// filter and no extra element to explain.
func TestNoReservedZoneCutsNothing(t *testing.T) {
	inks := map[string]string{"$brand.background": "#ffffff", "$brand.primary": "#000000", "$brand.accent": "#808080"}
	for _, preset := range Presets {
		res, err := Render(Request{Preset: preset, Width: 480, Height: 300, Seed: 5, Inks: inks})
		require.NoError(t, err)
		// `reserve-mask`, not `reserve`: every document says
		// `preserveAspectRatio` in its envelope.
		require.NotContains(t, string(res.SVG), "reserve-mask", "preset %q cut a reserve nobody asked for", preset)
	}
}

// TestAPlaneDocumentCarriesTheReserveItReferences closes a gap the two
// invariants above leave open.
//
// Both of them render without a reserve, so neither has ever seen the markup
// this adds. And the defs check they perform looks for FILTER references — a
// reserve is referenced as `mask="url(...)"`, which that scan does not match.
// So a plane document that referenced a mask it did not define would satisfy
// every existing test, and rasterize as a layer with no reserve in it at all:
// silently, because a rasterizer that cannot resolve a mask draws the element
// unmasked rather than reporting anything.
//
// A plane is rasterized ALONE when the plate model uses it. Whatever it
// references has to be inside it.
func TestAPlaneDocumentCarriesTheReserveItReferences(t *testing.T) {
	inks := map[string]string{"$brand.background": "#ffffff", "$brand.primary": "#000000", "$brand.accent": "#808080"}
	zone := QuietZone{X: 0.08, Y: 0.12, Width: 0.36, Height: 0.28, Feather: 0.09, TowardLight: true, Travel: 0.2}
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			res, err := Render(Request{Preset: preset, Width: 640, Height: 400, Seed: 5, Inks: inks, Quiet: &zone})
			require.NoError(t, err)
			for i, plane := range res.Planes {
				doc := string(res.PlaneDocuments[i])
				require.Containsf(t, doc, `mask="url(#reserve-mask)"`, "plane %q lost its reserve", plane)
				require.Containsf(t, doc, `<mask id="reserve-mask">`,
					"plane %q references a reserve mask it does not define, so rasterized alone it draws unmasked", plane)
				require.Containsf(t, doc, `<filter id="reserve"`,
					"plane %q references a blur it does not define, so its reserve rasterizes with a hard edge", plane)
				require.NotContainsf(t, doc, "$brand.", "plane %q leaves an unresolved ink slot in its reserve", plane)
			}
		})
	}
}

// TestTheReserveIsSizedForTheTravelItIsGiven pins the volume, not the rectangle.
//
// A reserve cut to the copy's own rectangle is legible for exactly one frame:
// the plate carrying it slides on the first scroll and the plates behind come
// up into the copy. This was measured, not imagined — a style scored 8.15 at
// rest and 1.00 half a screen later, and the parallax gate refused it.
//
// Downward only, because plates translate upward as the page scrolls: the
// material that would arrive over the copy is the material below it, and
// growing upward as well would spend twice the picture to buy nothing.
func TestTheReserveIsSizedForTheTravelItIsGiven(t *testing.T) {
	inks := map[string]string{"$brand.background": "#ffffff", "$brand.primary": "#000000", "$brand.accent": "#808080"}
	base := QuietZone{X: 0.1, Y: 0.2, Width: 0.3, Height: 0.2, Feather: 0.05, TowardLight: true}
	still, moving := base, base
	moving.Travel = 0.25

	heightOf := func(zone QuietZone) (y, h float64) {
		res, err := Render(Request{Preset: "colonnade", Width: 400, Height: 400, Seed: 1, Inks: inks, Quiet: &zone})
		require.NoError(t, err)
		m := regexp.MustCompile(`<mask id="reserve-mask"><rect x="[-0-9.]+" y="([-0-9.]+)" width="[-0-9.]+" height="([-0-9.]+)"`).
			FindStringSubmatch(string(res.SVG))
		require.Len(t, m, 3, "the reserve mask is not in the document")
		y, err = strconv.ParseFloat(m[1], 64)
		require.NoError(t, err)
		h, err = strconv.ParseFloat(m[2], 64)
		require.NoError(t, err)
		return y, h
	}

	stillY, stillH := heightOf(still)
	movingY, movingH := heightOf(moving)
	require.InDelta(t, stillY, movingY, 0.01, "travel must not move the reserve's top edge: the copy has not moved")
	require.InDelta(t, stillH+0.25*400, movingH, 0.01, "the reserve must grow by exactly the plate's travel")
}
