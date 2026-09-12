package treatments

import (
	"image"
	"image/color"
	"math"
	"sort"
	"testing"
)

// Perceptual assertions.
//
// Golden-image tests compare a treatment against a fixture produced by that
// same treatment, so they prove determinism and nothing else. Every defect
// found in the 2026-08-11 output-quality audit — ink ramps that never reached
// their light ink, halftone dots that never changed size, a line screen that
// saturated to black, operations that were silently no-ops — passed the golden
// suite on the day it was written and would have passed forever.
//
// These tests assert measurable properties of the *output* instead: that a ramp
// is actually traversed, that a screen actually modulates, that an operation
// actually does something. They are the gate a golden cannot be.

func toNRGBA(t *testing.T, img image.Image) *image.NRGBA {
	t.Helper()
	out, ok := img.(*image.NRGBA)
	if !ok {
		out = clone(img)
	}
	return out
}

// must returns a checker bound to t. It takes exactly (image.Image, error) so a
// treatment call can be passed straight through: m(Duotone(src, params)).
func must(t *testing.T) func(image.Image, error) *image.NRGBA {
	return func(img image.Image, err error) *image.NRGBA {
		t.Helper()
		if err != nil {
			t.Fatalf("treatment returned an error: %v", err)
		}
		return toNRGBA(t, img)
	}
}

// rampCoverage projects every output pixel onto the dark→light ink axis and
// reports how much of that axis the bulk of the image occupies, measured as the
// p10–p90 span.
//
// The percentile span matters, not min–max. A crushed tone scale still touches
// both ends of the ramp wherever the source happens to contain a pure white and
// a pure black pixel, so a min–max measure reports near-full coverage on an
// image that reads as one flat tone. What the linear-luminance defect actually
// did was pile the *distribution* into the dark end, and only a percentile
// measure sees that.
func rampCoverage(img *image.NRGBA, dark, light color.NRGBA) float64 {
	dx := float64(light.R) - float64(dark.R)
	dy := float64(light.G) - float64(dark.G)
	dz := float64(light.B) - float64(dark.B)
	den := dx*dx + dy*dy + dz*dz
	if den == 0 {
		return 0
	}
	const buckets = 1024
	var hist [buckets]int
	total := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			tproj := ((float64(c.R)-float64(dark.R))*dx +
				(float64(c.G)-float64(dark.G))*dy +
				(float64(c.B)-float64(dark.B))*dz) / den
			idx := int(math.Round(tproj * (buckets - 1)))
			if idx < 0 {
				idx = 0
			}
			if idx >= buckets {
				idx = buckets - 1
			}
			hist[idx]++
			total++
		}
	}
	if total == 0 {
		return 0
	}
	pct := func(frac float64) float64 {
		want, seen := int(float64(total)*frac), 0
		for i := 0; i < buckets; i++ {
			seen += hist[i]
			if seen >= want {
				return float64(i) / (buckets - 1)
			}
		}
		return 1
	}
	return pct(0.90) - pct(0.10)
}

func uniqueColors(img *image.NRGBA) map[color.NRGBA]int {
	seen := map[color.NRGBA]int{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			seen[img.NRGBAAt(x, y)]++
		}
	}
	return seen
}

// TestInkRampIsTraversed is the direct regression test for the linear-vs-
// perceptual luminance defect. Driving an ink ramp with linear luminance put
// the median pixel of a natural scene at 0.055, so the output never left the
// dark end and every duotone came out a flat single hue.
func TestInkRampIsTraversed(t *testing.T) {
	const dark, light = "#15152b", "#f6d58a"
	darkC, _ := parseColor(dark, color.NRGBA{})
	lightC, _ := parseColor(light, color.NRGBA{})
	src := sceneFixture(assertW, assertH)

	// Thresholds sit between measured defective and measured correct behaviour
	// on this fixture, so they fail the bug and pass the fix with margin rather
	// than encoding an aspiration. Linear-luminance mapping scores .406 / .427 /
	// .501; perceptual mapping scores .662 / .703 / .70+. Re-measure and
	// re-state these numbers if the fixture changes.
	for _, tc := range []struct {
		name string
		min  float64
		run  func() (image.Image, error)
	}{
		{"duotone", 0.55, func() (image.Image, error) {
			return Duotone(src, Params{Dark: dark, Light: light})
		}},
		{"duotone_normalized", 0.60, func() (image.Image, error) {
			return Duotone(src, Params{Dark: dark, Light: light, Normalize: true})
		}},
		{"posterize", 0.65, func() (image.Image, error) {
			return Posterize(src, Params{Levels: 5, Dark: dark, Light: light})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := must(t)
			out := m(tc.run())
			if got := rampCoverage(out, darkC, lightC); got < tc.min {
				t.Fatalf("ink ramp coverage %.3f < %.3f — the ramp is not being traversed, "+
					"which is what a non-perceptual tone scale looks like", got, tc.min)
			}
		})
	}
}

// TestPosterizeReachesEveryLevel guards the same defect from the other side: a
// crushed tone scale silently collapses the requested levels. Before the fix a
// 5-level posterize emitted 3 distinct colours.
func TestPosterizeReachesEveryLevel(t *testing.T) {
	m := must(t)
	src := sceneFixture(assertW, assertH)
	for _, levels := range []int{3, 5, 8} {
		out := m(Posterize(src, Params{
			Levels: levels, Dark: "#111827", Light: "#fef3c7", Normalize: true,
		}))
		if got := len(uniqueColors(out)); got != levels {
			t.Errorf("posterize(levels=%d) produced %d distinct colours; every requested level must be reachable", levels, got)
		}
	}
}

// TestScreenedTreatmentsModulate asserts that a screen carries tone. A halftone
// whose dots are all the same size is a texture, not an image: the audit found
// the dot-radius factor collapsed into a narrow band, so the picture vanished
// behind a uniform field of dots.
//
// It measures ink coverage per tile across the frame and requires real
// variation between the lightest and darkest tiles.
func TestScreenedTreatmentsModulate(t *testing.T) {
	src := sceneFixture(assertW, assertH)
	const dark, light = "#111827", "#fef3c7"
	darkC, _ := parseColor(dark, color.NRGBA{})

	cases := map[string]func() (image.Image, error){
		"halftone": func() (image.Image, error) {
			return Halftone(src, Params{LPI: 48, Angle: 15, Dot: "circle", Dark: dark, Light: light})
		},
		"line_screen": func() (image.Image, error) {
			return Tier2(src, "line_screen", Params{Spacing: 8, Angle: 45, Dark: dark, Light: light})
		},
		"stipple": func() (image.Image, error) {
			return Tier2(src, "stipple", Params{Spacing: 7, Seed: 19, Dark: dark, Light: light})
		},
		"engraving": func() (image.Image, error) {
			return Tier2(src, "engraving", Params{Spacing: 8, Dark: dark, Light: light})
		},
		"dither_ordered": func() (image.Image, error) {
			return DitherOrdered(src, Params{Dark: dark, Light: light})
		},
		"dither_diffusion": func() (image.Image, error) {
			return DitherDiffusion(src, Params{Dark: dark, Light: light})
		},
	}

	for name, run := range cases {
		name, run := name, run
		t.Run(name, func(t *testing.T) {
			m := must(t)
			out := m(run())
			const tiles = 8
			tw, th := assertW/tiles, assertH/tiles
			cov := make([]float64, 0, tiles*tiles)
			for ty := 0; ty < tiles; ty++ {
				for tx := 0; tx < tiles; tx++ {
					inked, total := 0, 0
					for y := ty * th; y < (ty+1)*th; y++ {
						for x := tx * tw; x < (tx+1)*tw; x++ {
							if out.NRGBAAt(x, y) == darkC {
								inked++
							}
							total++
						}
					}
					cov = append(cov, float64(inked)/float64(total))
				}
			}
			sort.Float64s(cov)
			// p20–p80 rather than min–max. The fixture deliberately contains a
			// near-black hill and a near-white sun, so min and max stay far
			// apart even when every midtone tile has collapsed to the same
			// coverage — which is exactly what the crushed tone scale did. The
			// interior spread is what proves the screen carries tone across the
			// range rather than only at the extremes.
			lo, hi := cov[len(cov)*20/100], cov[len(cov)*80/100]
			if hi-lo < 0.25 {
				t.Fatalf("interior ink coverage varies only %.3f across the frame (p20=%.3f, p80=%.3f); "+
					"the screen modulates at the extremes but not through the midtones", hi-lo, lo, hi)
			}
		})
	}
}

// TestQuantizingTreatmentsAreBinary pins the ops that must reduce to exactly
// two inks. A "dither" that emits continuous tone has not quantised anything.
func TestQuantizingTreatmentsAreBinary(t *testing.T) {
	src := sceneFixture(assertW, assertH)
	const dark, light = "#111827", "#fef3c7"
	cases := map[string]func() (image.Image, error){
		"dither_ordered":   func() (image.Image, error) { return DitherOrdered(src, Params{Dark: dark, Light: light}) },
		"dither_diffusion": func() (image.Image, error) { return DitherDiffusion(src, Params{Dark: dark, Light: light}) },
		"halftone": func() (image.Image, error) {
			return Halftone(src, Params{LPI: 48, Angle: 15, Dark: dark, Light: light})
		},
	}
	for name, run := range cases {
		name, run := name, run
		t.Run(name, func(t *testing.T) {
			m := must(t)
			if got := len(uniqueColors(m(run()))); got != 2 {
				t.Fatalf("%s produced %d colours, want exactly 2", name, got)
			}
		})
	}
}

// TestEveryTreatmentChangesItsInput catches silent no-ops. The audit found
// pixel_sort returning output with byte-identical statistics to its input, and
// backdrop-studio's guided lane filing its own input as its output. An
// operation that cannot be shown to have done anything must not ship.
func TestEveryTreatmentChangesItsInput(t *testing.T) {
	src := toNRGBA(t, sceneFixture(assertW, assertH))
	for name, fn := range goldenCases {
		name, fn := name, fn
		t.Run(name, func(t *testing.T) {
			m := must(t)
			out := m(fn(src))
			if out.Bounds() != src.Bounds() {
				return // a resizing treatment has self-evidently done something
			}
			changed := 0
			b := src.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					if out.NRGBAAt(x, y) != src.NRGBAAt(x, y) {
						changed++
					}
				}
			}
			frac := float64(changed) / float64(b.Dx()*b.Dy())
			if frac < 0.01 {
				t.Fatalf("%s changed only %.4f%% of pixels — it is effectively a no-op at its golden parameters", name, frac*100)
			}
		})
	}
}

// TestHalftoneIsResolutionIndependent pins the LPI contract. LPI is lines
// across the image width, so the same LPI must yield the same visual coarseness
// at any render size — otherwise a style that looks right in a preview is wrong
// at delivery size. Previously LPI was used as a raw pixel pitch, so a screen
// got finer as the frame grew.
func TestHalftoneIsResolutionIndependent(t *testing.T) {
	coverage := func(w, h int) float64 {
		m := must(t)
		out := m(Halftone(sceneFixture(w, h), Params{
			LPI: 40, Angle: 15, Dark: "#111827", Light: "#fef3c7",
		}))
		darkC, _ := parseColor("#111827", color.NRGBA{})
		inked := 0
		b := out.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if out.NRGBAAt(x, y) == darkC {
					inked++
				}
			}
		}
		return float64(inked) / float64(b.Dx()*b.Dy())
	}
	small, large := coverage(600, 375), coverage(1800, 1125)
	if math.Abs(small-large) > 0.06 {
		t.Fatalf("halftone ink coverage differs across render sizes (%.3f at 600px vs %.3f at 1800px); "+
			"LPI is not resolution-independent", small, large)
	}
}

// TestLightnessIsPerceptual pins the scale itself, so nobody re-points the tonal
// ops at linear luminance later. luminance must stay linear — the legibility
// gate and WCAG depend on it — and lightness must stay perceptual.
func TestLightnessIsPerceptual(t *testing.T) {
	mid := color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	if y := luminance(mid); math.Abs(y-0.216) > 0.01 {
		t.Fatalf("luminance(mid grey) = %.4f, want ~0.216 — luminance must stay linear-light for WCAG contrast", y)
	}
	// sRGB 128 lands at L* ≈ 0.536, not exactly 0.5: sRGB's ~2.2 gamma and the
	// L* curve are close but not identical (L* 0.5 is sRGB ≈119). The property
	// that matters is that it sits near the middle of the range rather than at
	// the 0.216 linear luminance reports.
	if l := lightness(mid); l < 0.45 || l > 0.60 {
		t.Fatalf("lightness(mid grey) = %.4f, want ~0.536 — tonal mapping must run on a perceptual scale", l)
	}
	// Monotonic, and spanning the full range at the endpoints.
	if lightness(color.NRGBA{A: 255}) != 0 {
		t.Fatal("lightness(black) must be 0")
	}
	if l := lightness(color.NRGBA{R: 255, G: 255, B: 255, A: 255}); math.Abs(l-1) > 1e-9 {
		t.Fatalf("lightness(white) = %.6f, want 1", l)
	}
	prev := -1.0
	for v := 0; v <= 255; v++ {
		l := lightness(color.NRGBA{R: uint8(v), G: uint8(v), B: uint8(v), A: 255})
		if l < prev {
			t.Fatalf("lightness is not monotonic at v=%d", v)
		}
		prev = l
	}
}

// TestNormalizeStretchesLowContrastSources proves the auto-level actually
// rescues a source the plain mapping would waste, and that it is a no-op on a
// flat image rather than amplifying nothing into noise.
func TestNormalizeStretchesLowContrastSources(t *testing.T) {
	m := must(t)
	// A deliberately low-contrast source: all mid greys.
	low := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			v := uint8(110 + x/25)
			low.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	darkC, _ := parseColor("#000000", color.NRGBA{})
	lightC, _ := parseColor("#ffffff", color.NRGBA{})
	plain := m(Duotone(low, Params{Dark: "#000000", Light: "#ffffff"}))
	norm := m(Duotone(low, Params{Dark: "#000000", Light: "#ffffff", Normalize: true}))
	pc, nc := rampCoverage(plain, darkC, lightC), rampCoverage(norm, darkC, lightC)
	if nc <= pc*2 {
		t.Fatalf("normalize did not stretch a low-contrast source: coverage %.3f -> %.3f", pc, nc)
	}

	// A perfectly flat source has no range to stretch; normalize must leave it
	// alone rather than divide by ~zero.
	flat := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			flat.SetNRGBA(x, y, color.NRGBA{R: 90, G: 90, B: 90, A: 255})
		}
	}
	out := m(Duotone(flat, Params{Dark: "#000000", Light: "#ffffff", Normalize: true}))
	if got := len(uniqueColors(out)); got != 1 {
		t.Fatalf("normalize on a flat source produced %d colours, want 1", got)
	}
}

// TestAsciiMosaicRendersGlyphs pins that ascii_mosaic actually substitutes
// characters. The pre-fix implementation averaged each block to one of five
// colours and filled the block — a pixelate with no glyph in the output, which
// is not the operation it is named for.
//
// A glyph mosaic has a signature a block fill cannot fake: within a single
// cell, ink and paper both appear (the character has interior structure), and
// the ink pattern differs between cells of different tone.
func TestAsciiMosaicRendersGlyphs(t *testing.T) {
	m := must(t)
	const dark, light = "#111827", "#fef3c7"
	darkC, _ := parseColor(dark, color.NRGBA{})
	const cell = 8
	out := m(Tier2(sceneFixture(assertW, assertH), "ascii_mosaic", Params{
		BlockSize: cell, Dark: dark, Light: light,
	}))

	if got := len(uniqueColors(out)); got != 2 {
		t.Fatalf("ascii_mosaic produced %d colours, want exactly 2 (ink on paper)", got)
	}

	// Count cells that contain BOTH ink and paper. A block fill yields zero.
	ch := cell * 13 / 7
	mixed, inspected := 0, 0
	for by := 0; by+ch <= assertH; by += ch {
		for bx := 0; bx+cell <= assertW; bx += cell {
			ink, paper := 0, 0
			for y := by; y < by+ch; y++ {
				for x := bx; x < bx+cell; x++ {
					if out.NRGBAAt(x, y) == darkC {
						ink++
					} else {
						paper++
					}
				}
			}
			if ink > 0 && paper > 0 {
				mixed++
			}
			inspected++
		}
	}
	if frac := float64(mixed) / float64(inspected); frac < 0.25 {
		t.Fatalf("only %.1f%% of cells contain both ink and paper — the output is filled blocks, not glyphs", frac*100)
	}
}

// TestPixelSortReordersPixels pins that pixel_sort does something and that what
// it does is a permutation. At its golden threshold against linear luminance it
// selected no runs at all and returned its input unchanged.
func TestPixelSortReordersPixels(t *testing.T) {
	m := must(t)
	src := toNRGBA(t, sceneFixture(assertW, assertH))
	out := m(Tier2(src, "pixel_sort", Params{Threshold: .55, Axis: "horizontal"}))

	moved := 0
	for y := 0; y < assertH; y++ {
		for x := 0; x < assertW; x++ {
			if out.NRGBAAt(x, y) != src.NRGBAAt(x, y) {
				moved++
			}
		}
	}
	if frac := float64(moved) / float64(assertW*assertH); frac < 0.05 {
		t.Fatalf("pixel_sort moved %.2f%% of pixels — it is selecting no runs", frac*100)
	}

	// Sorting rearranges; it must not invent or destroy colour.
	inHist, outHist := uniqueColors(src), uniqueColors(out)
	if len(inHist) != len(outHist) {
		t.Fatalf("pixel_sort changed the colour set (%d -> %d); a sort must be a permutation", len(inHist), len(outHist))
	}
	for c, n := range inHist {
		if outHist[c] != n {
			t.Fatalf("pixel_sort changed the count of %v (%d -> %d); a sort must be a permutation", c, n, outHist[c])
		}
	}

	// Each sorted run must be non-decreasing in lightness.
	for _, y := range []int{assertH / 4, assertH / 2, 3 * assertH / 4} {
		runStart := -1
		for x := 0; x <= assertW; x++ {
			active := x < assertW && lightness(src.NRGBAAt(x, y)) > .55
			if active && runStart < 0 {
				runStart = x
			}
			if !active && runStart >= 0 {
				for i := runStart + 1; i < x; i++ {
					if lightness(out.NRGBAAt(i, y)) < lightness(out.NRGBAAt(i-1, y))-1e-9 {
						t.Fatalf("run at y=%d is not sorted at x=%d", y, i)
					}
				}
				runStart = -1
			}
		}
	}
}
