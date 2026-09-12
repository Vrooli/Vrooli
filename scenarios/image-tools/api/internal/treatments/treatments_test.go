package treatments

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Golden and assertion sizes. A treatment fixture has to be big enough that a
// screen, a dot or a hatch is structurally present — at the previous 64×48 a
// halftone cell was a large fraction of the frame, so no tonal defect could
// show up. Goldens stay modest so the repo does not carry megabytes of noisy
// PNG; the perceptual assertions and the on-demand evidence render use the
// larger size, where defects are actually measurable.
const (
	goldenW, goldenH     = 480, 300
	assertW, assertH     = 1200, 750
	evidenceW, evidenceH = 1600, 1000
)

// sceneFixture renders a deterministic landscape spanning the full tonal range,
// from a near-black hill to a near-white sun. The previous fixture was a
// synthetic colour chart: correct for proving determinism, useless for proving
// a tonal mapping is right, because its luminance histogram looks nothing like
// a real image's. Every treatment defect found in the 2026-08-11 audit was
// invisible against a chart and obvious against a scene.
func sceneFixture(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fw, fh := float64(w), float64(h)
	hz := fh * 0.58
	set := func(x, y int, r, g, b float64) {
		cl := func(v float64) uint8 {
			if v < 0 {
				return 0
			}
			if v > 255 {
				return 255
			}
			return uint8(v + 0.5)
		}
		img.SetNRGBA(x, y, color.NRGBA{R: cl(r), G: cl(g), B: cl(b), A: 255})
	}
	sunX, sunY, sunR := fw*0.74, fh*0.21, fh*0.11
	for y := 0; y < h; y++ {
		fy := float64(y)
		for x := 0; x < w; x++ {
			fx := float64(x)
			switch {
			case fy < hz: // sky: deep at the zenith, pale at the horizon
				t := fy / hz
				set(x, y, 14+186*t*t, 26+200*t*t, 78+165*t*t)
			default: // sea: darkening with depth, banded
				t := (fy - hz) / (fh - hz)
				band := 10 * math.Sin(fy*0.7)
				set(x, y, 16-8*t+band, 62-40*t+band, 150-96*t+band)
			}
			// sun disc plus a soft halo — the fixture's white point
			d := math.Hypot(fx-sunX, fy-sunY)
			if d < sunR {
				set(x, y, 255, 250, 236)
			} else if d < sunR*2.4 {
				k := 1 - (d-sunR)/(sunR*1.4)
				c := img.NRGBAAt(x, y)
				set(x, y, float64(c.R)+150*k, float64(c.G)+140*k, float64(c.B)+110*k)
			}
		}
	}
	// two hills: the far one mid-tone, the near one the fixture's black point
	for x := 0; x < w; x++ {
		fx := float64(x)
		far := hz - 0.30*fh*math.Max(0, math.Sin(math.Pi*fx/(fw*0.78)))
		near := hz + 0.06*fh - 0.19*fh*math.Max(0, math.Sin(math.Pi*fx/(fw*0.52)))
		for y := int(far); y < h; y++ {
			if float64(y) >= far {
				set(x, y, 26, 74, 52)
			}
		}
		for y := int(near); y < h; y++ {
			set(x, y, 8, 16, 13)
		}
	}
	// a bright sand strip so the low midtones are populated too
	for x := 0; x < w; x++ {
		fx := float64(x)
		top := hz + 0.13*fh - 0.05*fh*math.Max(0, math.Sin(math.Pi*fx/(fw*0.46)))
		for y := int(top); y < int(top+fh*0.03) && y < h; y++ {
			set(x, y, 236, 224, 190)
		}
	}
	return img
}

func fixture() image.Image { return sceneFixture(goldenW, goldenH) }

var goldenCases = map[string]func(image.Image) (image.Image, error){
	"duotone": func(img image.Image) (image.Image, error) {
		return Duotone(img, Params{Dark: "#15152b", Light: "#f6d58a", Mid: "#bb426b", MidLow: .38, MidHigh: .62})
	},
	"posterize": func(img image.Image) (image.Image, error) {
		return Posterize(img, Params{Levels: 5, Dark: "#111827", Light: "#fef3c7"})
	},
	"halftone": func(img image.Image) (image.Image, error) {
		return Halftone(img, Params{LPI: 48, Angle: 22, Dot: "circle", Dark: "#111827", Light: "#fef3c7"})
	},
	"dither_ordered": func(img image.Image) (image.Image, error) {
		return DitherOrdered(img, Params{Dark: "#111827", Light: "#fef3c7"})
	},
	"dither_diffusion": func(img image.Image) (image.Image, error) {
		return DitherDiffusion(img, Params{Dark: "#111827", Light: "#fef3c7"})
	},
	"grain": func(img image.Image) (image.Image, error) {
		return Grain(img, Params{Seed: 42, Amount: .12, Contrast: 1.08})
	},
	"scrim": func(img image.Image) (image.Image, error) {
		return Scrim(img, Params{ScrimColor: "#13294b", Opacity: .65, Direction: "left"})
	},
	"line_screen": func(img image.Image) (image.Image, error) {
		return Tier2(img, "line_screen", Params{Spacing: 7, Angle: 18, Dark: "#111827", Light: "#fef3c7"})
	},
	"stipple": func(img image.Image) (image.Image, error) {
		return Tier2(img, "stipple", Params{Spacing: 6, Seed: 19, Dark: "#111827", Light: "#fef3c7"})
	},
	"engraving": func(img image.Image) (image.Image, error) {
		return Tier2(img, "engraving", Params{Spacing: 8, Dark: "#111827", Light: "#fef3c7"})
	},
	"aberration": func(img image.Image) (image.Image, error) { return Tier2(img, "aberration", Params{Amplitude: 3}) },
	"bloom": func(img image.Image) (image.Image, error) {
		return Tier2(img, "bloom", Params{Radius: 3, Threshold: .68})
	},
	"curve": func(img image.Image) (image.Image, error) { return Tier2(img, "curve", Params{Curve: .78}) },
	"defocus": func(img image.Image) (image.Image, error) {
		return Tier2(img, "defocus", Params{Radius: 2, BladeCount: 6})
	},
	"motion_blur": func(img image.Image) (image.Image, error) {
		return Tier2(img, "motion_blur", Params{Distance: 5, Angle: 24})
	},
	"ascii_mosaic": func(img image.Image) (image.Image, error) {
		return Tier2(img, "ascii_mosaic", Params{BlockSize: 7, Dark: "#111827", Light: "#fef3c7"})
	},
	"pixel_sort": func(img image.Image) (image.Image, error) {
		return Tier2(img, "pixel_sort", Params{Threshold: .64, Axis: "horizontal"})
	},
	"displacement": func(img image.Image) (image.Image, error) { return Tier2(img, "displacement", Params{Amplitude: 4}) },
}

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestGoldenTreatments(t *testing.T) {
	root := filepath.Join("testdata", "golden")
	evidenceRoot := os.Getenv("EVIDENCE_DIR")
	for name, fn := range goldenCases {
		name, fn := name, fn
		t.Run(name, func(t *testing.T) {
			got := encode(t, runTreatment(t, fn))
			if evidenceRoot != "" {
				// Evidence renders at delivery resolution from a full-size
				// scene. A golden is a drift tripwire; evidence is what a human
				// judges, and it cannot be judged at golden size.
				big, err := fn(sceneFixture(evidenceW, evidenceH))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(evidenceRoot, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(evidenceRoot, name+".png"), encode(t, big), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(root, name+".png")
			if os.Getenv("UPDATE_GOLDENS") == "1" {
				if err := os.MkdirAll(root, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden %s: %v (run UPDATE_GOLDENS=1)", path, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("golden %s changed: got %d bytes, want %d", name, len(got), len(want))
			}
			second := encode(t, runTreatment(t, fn))
			if !bytes.Equal(got, second) {
				t.Fatal("treatment is not deterministic")
			}
		})
	}
}

func runTreatment(t *testing.T, fn func(image.Image) (image.Image, error)) image.Image {
	t.Helper()
	img, err := fn(fixture())
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestTreatmentsRejectInvalidParameters(t *testing.T) {
	if _, err := Posterize(fixture(), Params{Levels: 1}); err == nil {
		t.Fatal("posterize accepted invalid levels")
	}
	if _, err := Halftone(fixture(), Params{LPI: 1}); err == nil {
		t.Fatal("halftone accepted invalid lpi")
	}
	if _, err := Duotone(fixture(), Params{Dark: "not-a-color"}); err == nil {
		t.Fatal("duotone accepted invalid color")
	}
}

// A region-scoped scrim shades where copy sits and leaves the picture alone.
//
// The whole-frame gradient is the right tool for setting a mood and the wrong
// one for making a headline readable: it dims a picture chosen for its beauty
// everywhere in order to fix one corner. These assertions are what separate the
// two — without them a "region" that quietly covered the frame would pass every
// structural check and defeat its own purpose.
func TestARegionScopedScrimShadesTheRegionAndSparesTheRest(t *testing.T) {
	const w, h = 200, 200
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 240, G: 236, B: 220, A: 255})
		}
	}
	out, err := Scrim(src, Params{
		ScrimColor: "#000000", Opacity: 0.6,
		RegionX: 0.05, RegionY: 0.05, RegionWidth: 0.35, RegionHeight: 0.25, RegionFeather: 0.06,
	})
	if err != nil {
		t.Fatalf("scrim: %v", err)
	}
	nrgba, ok := out.(*image.NRGBA)
	if !ok {
		t.Fatalf("scrim returned %T", out)
	}

	// Inside the region: shaded at full strength.
	inside := nrgba.NRGBAAt(40, 40)
	if inside.R > 110 {
		t.Fatalf("inside the region the scrim barely applied (R=%d); copy there is no more legible than before", inside.R)
	}
	// Far outside: untouched. This is the whole point.
	far := nrgba.NRGBAAt(190, 190)
	if far.R != 240 {
		t.Fatalf("the far corner was dimmed to R=%d; a region-scoped scrim must leave the picture alone", far.R)
	}
	// And the edge is soft: a pixel just past the region is neither fully
	// shaded nor fully clear, or the reader sees a box before the headline.
	edge := nrgba.NRGBAAt(int(0.42*w), 40)
	if edge.R <= inside.R || edge.R >= far.R {
		t.Fatalf("the pool has a hard edge (inside %d, edge %d, outside %d)", inside.R, edge.R, far.R)
	}
}

// With no region declared a scrim is exactly what it always was. Every seeded
// style using one depends on that.
func TestAScrimWithNoRegionIsStillTheWholeFrameGradient(t *testing.T) {
	const w, h = 100, 100
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
		}
	}
	out, err := Scrim(src, Params{ScrimColor: "#000000", Opacity: 0.8, Direction: "left"})
	if err != nil {
		t.Fatalf("scrim: %v", err)
	}
	nrgba := out.(*image.NRGBA)
	// A left-directional gradient is clear at the left edge and darkest at the
	// right, and it touches the whole frame.
	if left, right := nrgba.NRGBAAt(0, 50), nrgba.NRGBAAt(w-1, 50); left.R <= right.R {
		t.Fatalf("the directional gradient did not run left-to-right (left %d, right %d)", left.R, right.R)
	}
	if nrgba.NRGBAAt(w-1, 0).R == 240 {
		t.Fatal("the whole-frame gradient left a corner untouched; it is not region-scoped and must cover the frame")
	}
}

// knockoutOps is every colour treatment the ops registry can dispatch, with
// parameters that make each one actually draw. Listed explicitly rather than
// derived, so adding a treatment without deciding what it does to reserved
// space is a compile-time gap a reader will notice, not a silent omission.
var knockoutOps = []struct {
	name string
	run  func(image.Image, Params) (image.Image, error)
}{
	{"duotone", Duotone},
	{"posterize", func(i image.Image, p Params) (image.Image, error) { p.Levels = 5; return Posterize(i, p) }},
	{"halftone", func(i image.Image, p Params) (image.Image, error) { p.LPI = 40; return Halftone(i, p) }},
	{"dither_ordered", DitherOrdered},
	{"dither_diffusion", DitherDiffusion},
	{"grain", Grain},
	{"line_screen", tier2Op("line_screen")},
	{"stipple", tier2Op("stipple")},
	{"engraving", tier2Op("engraving")},
	{"aberration", tier2Op("aberration")},
	{"bloom", tier2Op("bloom")},
	{"curve", tier2Op("curve")},
	{"defocus", tier2Op("defocus")},
	{"motion_blur", tier2Op("motion_blur")},
	{"ascii_mosaic", tier2Op("ascii_mosaic")},
	{"pixel_sort", tier2Op("pixel_sort")},
	{"displacement", tier2Op("displacement")},
}

func tier2Op(name string) func(image.Image, Params) (image.Image, error) {
	return func(i image.Image, p Params) (image.Image, error) { return Tier2(i, name, p) }
}

// TestNoTreatmentPrintsIntoAKnockout is the contract that makes reserved space
// mean anything.
//
// It asserts on the DARKEST pixel in the reserve, not the average. A treatment
// that clears ninety-nine percent of the area and leaves one dot has not
// reserved anything: worst-pixel contrast is what decides whether a headline
// can be read, and one dot sets it. An averaged assertion passes happily on
// exactly the failure this exists to prevent.
//
// The source is a real gradient rather than a flat field, because a flat field
// hides two whole classes of defect: an operation that ignores tone entirely
// looks correct on it, and an operation that fetches from a neighbour has
// nothing dark nearby to fetch.
func TestNoTreatmentPrintsIntoAKnockout(t *testing.T) {
	reserve := Params{KnockoutX: 0.10, KnockoutY: 0.16, KnockoutWidth: 0.40, KnockoutHeight: 0.50, KnockoutFeather: 0.10}
	for _, op := range knockoutOps {
		t.Run(op.name, func(t *testing.T) {
			bare, err := op.run(knockoutSource(), Params{})
			if err != nil {
				t.Fatalf("without a knockout: %v", err)
			}
			held, err := op.run(ReserveBefore(knockoutSource(), reserve), reserve)
			if err != nil {
				t.Fatalf("with a knockout: %v", err)
			}
			before, after := worstInReserve(bare, reserve), worstInReserve(held, reserve)
			// 0.86 is duotone's own paper: the lightest ink in a two-ink ramp is
			// a cream, not white, and a knockout that came out whiter than the
			// paper the picture is printed on would be a hole rather than a
			// reserve. Every operation must reach its own paper; none has to
			// exceed it.
			if after < 0.86 {
				t.Errorf("reserved space holds ink at %.3f (was %.3f without the knockout)", after, before)
			}
		})
	}
}

// knockoutSource is a picture with a full tonal sweep and hard dark structure,
// so an operation that drags ink inward has something to drag.
func knockoutSource() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 200, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 200; x++ {
			v := uint8(x * 255 / 199)
			if (x/9+y/9)%2 == 0 {
				v = uint8(int(v) * 2 / 5)
			}
			img.Set(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func worstInReserve(img image.Image, p Params) float64 {
	b := img.Bounds()
	fw, fh := float64(b.Dx()), float64(b.Dy())
	x0, y0 := int(p.KnockoutX*fw), int(p.KnockoutY*fh)
	x1, y1 := int((p.KnockoutX+p.KnockoutWidth)*fw), int((p.KnockoutY+p.KnockoutHeight)*fh)
	worst := 1.0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if v := lightness(color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bb >> 8), A: 255}); v < worst {
				worst = v
			}
		}
	}
	return worst
}

// TestAKnockoutDoesNotRescaleTheRestOfThePicture pins the reason the tone
// mapper excludes reserved space from its histogram.
//
// A knockout is white this package added, not a highlight the picture has. Let
// it into the auto-level statistics and it becomes the new p99, and every real
// tone gets compressed downward to make room for it — the picture outside the
// reserve darkens because of a decision about the space inside it. The larger
// the area an author reserves for their copy, the worse the distortion, so the
// failure would be at its worst on precisely the layouts that need this most.
func TestAKnockoutDoesNotRescaleTheRestOfThePicture(t *testing.T) {
	reserve := Params{
		Normalize: true, Levels: 6,
		KnockoutX: 0.05, KnockoutY: 0.05, KnockoutWidth: 0.45, KnockoutHeight: 0.85, KnockoutFeather: 0.04,
	}
	plain, err := Posterize(knockoutSource(), Params{Normalize: true, Levels: 6})
	if err != nil {
		t.Fatalf("posterize: %v", err)
	}
	held, err := Posterize(ReserveBefore(knockoutSource(), reserve), reserve)
	if err != nil {
		t.Fatalf("posterize with a knockout: %v", err)
	}
	// Sampled well clear of the reserve and its feather, where the two pictures
	// are the same picture and any difference is the tone mapper's doing.
	var diff, n float64
	for y := 10; y < 110; y++ {
		for x := 140; x < 195; x++ {
			pr, pg, pb, _ := plain.At(x, y).RGBA()
			hr, hg, hb, _ := held.At(x, y).RGBA()
			diff += math.Abs(lightness(color.NRGBA{R: uint8(pr >> 8), G: uint8(pg >> 8), B: uint8(pb >> 8), A: 255}) -
				lightness(color.NRGBA{R: uint8(hr >> 8), G: uint8(hg >> 8), B: uint8(hb >> 8), A: 255}))
			n++
		}
	}
	if mean := diff / n; mean > 0.01 {
		t.Errorf("reserving space shifted the untouched part of the picture by %.4f in mean lightness", mean)
	}
}

// TestAStippleLeavesPaperAlone pins the rule engraving already followed and
// stipple did not: a dot too small to draw is not drawn.
//
// Without it the dot bounds always covered their own centre, so every cell in
// the grid deposited one full-strength pixel however light the tone above it —
// a stippled white field came out as a regular lattice of ink rather than as
// paper. This is a tonal bug in its own right, not only a legibility one: it
// put ink into the top of the ramp on every stippled picture in the catalog.
func TestAStippleLeavesPaperAlone(t *testing.T) {
	white := image.NewNRGBA(image.Rect(0, 0, 60, 60))
	draw.Draw(white, white.Bounds(), &image.Uniform{C: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	out, err := Tier2(white, "stipple", Params{})
	if err != nil {
		t.Fatalf("stipple: %v", err)
	}
	for y := 0; y < 60; y++ {
		for x := 0; x < 60; x++ {
			r, g, b, _ := out.At(x, y).RGBA()
			if v := lightness(color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}); v < 0.95 {
				t.Fatalf("stipple put ink at (%d,%d) on a white field: lightness %.3f", x, y, v)
			}
		}
	}
}

// TestASolidReservesGroundForLightCopy is the mirror of the knockout contract.
//
// It asserts on the BRIGHTEST pixel, because the copy this serves is light: one
// bright mark inside the reserve decides worst-pixel contrast for white type
// exactly as one dark mark decides it for black type.
//
// The continuous-tone operations are listed and the discrete SCREENS are not,
// and the split is the medium's rather than this code's. Paper is achieved by
// not printing, so feeding a screen white is enough and it lays nothing. Ink is
// achieved by printing, and a screen cannot print a solid: its marks are
// discrete, so its coverage is bounded below one however hard it is driven. Fed
// full tone, line screen still showed 0.932 of paper between its lines, stipple
// 0.949, ASCII mosaic 0.940, engraving 0.932.
//
// The gap is real and is recorded rather than papered over. Painting a flat
// solid after the screen closes it and costs more than it buys: the perceptual
// gate scored that at 0.116 subject survival against a 0.600 floor and refused
// four styles that had been rendering correctly. A style whose chain screens and
// whose copy is light has to change one of the two.
func TestASolidReservesGroundForLightCopy(t *testing.T) {
	screens := map[string]bool{"halftone": true, "line_screen": true, "stipple": true, "engraving": true, "ascii_mosaic": true}
	reserve := Params{
		KnockoutX: 0.10, KnockoutY: 0.16, KnockoutWidth: 0.40, KnockoutHeight: 0.50,
		KnockoutFeather: 0.10, KnockoutSolid: true, Dark: "#101828",
	}
	for _, op := range knockoutOps {
		if screens[op.name] {
			continue
		}
		t.Run(op.name, func(t *testing.T) {
			held, err := op.run(ReserveBefore(knockoutSource(), reserve), reserve)
			if err != nil {
				t.Fatalf("with a solid: %v", err)
			}
			if bright := brightestInReserve(held, reserve); bright > 0.14 {
				t.Errorf("reserved space shows paper at %.3f; light copy has nothing to sit against", bright)
			}
		})
	}
}

// TestADiscreteScreenCannotLayASolid pins the limitation above as a measured
// fact rather than a remark, so that a later reader who assumes every operation
// can serve light copy is corrected by a failing test rather than by a backdrop.
//
// `halftone` is in this set and `dither_ordered` is not, which is the line
// itself: a halftone's mark is a dot INSIDE a cell and can never fill the cell's
// corners, while a dither's mark is the pixel, so at full tone every pixel turns
// and the area goes genuinely solid. Sub-cell marks cap; per-pixel marks do not.
func TestADiscreteScreenCannotLayASolid(t *testing.T) {
	reserve := Params{
		KnockoutX: 0.10, KnockoutY: 0.16, KnockoutWidth: 0.40, KnockoutHeight: 0.50,
		KnockoutFeather: 0.10, KnockoutSolid: true, Dark: "#101828", LPI: 40,
	}
	for _, name := range []string{"line_screen", "stipple", "engraving", "ascii_mosaic"} {
		t.Run(name, func(t *testing.T) {
			held, err := Tier2(ReserveBefore(knockoutSource(), reserve), name, reserve)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if bright := brightestInReserve(held, reserve); bright < 0.5 {
				t.Errorf("%s reached %.3f in a solid reserve; if a screen can now cover, "+
					"light copy over a screened style is supportable and the catalog rule should be revisited", name, bright)
			}
		})
	}
	t.Run("halftone", func(t *testing.T) {
		held, err := Halftone(ReserveBefore(knockoutSource(), reserve), reserve)
		if err != nil {
			t.Fatalf("halftone: %v", err)
		}
		if bright := brightestInReserve(held, reserve); bright < 0.5 {
			t.Errorf("halftone reached %.3f in a solid reserve; see above", bright)
		}
	})
}

// brightestInReserve is worstInReserve for light copy: the lightest pixel is the
// one a white headline has least to work with.
func brightestInReserve(img image.Image, p Params) float64 {
	b := img.Bounds()
	fw, fh := float64(b.Dx()), float64(b.Dy())
	x0, y0 := int(p.KnockoutX*fw), int(p.KnockoutY*fh)
	x1, y1 := int((p.KnockoutX+p.KnockoutWidth)*fw), int((p.KnockoutY+p.KnockoutHeight)*fh)
	bright := 0.0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if v := lightness(color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bb >> 8), A: 255}); v > bright {
				bright = v
			}
		}
	}
	return bright
}
