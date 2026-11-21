package scenes

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
)

// Flattening a generator's planes reproduces its composite exactly.
//
// This is the assertion that makes the plane refactor safe. Thirteen generators
// draw into one buffer, and separating them is exactly the kind of change that
// silently alters what a style looks like. The property is asserted on the
// FLATTENED result rather than on each plane, because overlapping planes are
// allowed to look like anything as long as they compose back — a generator that
// blends haze over a headland is drawing two layers whose individual contents
// are not separately meaningful.
//
// It is exact rather than approximate: the flat buffer takes the blended result
// and each plane takes the source colour at its own coverage, and alpha
// compositing is associative, so the two agree to the byte.
func TestFlattenedPlanesReproduceEveryGeneratorsComposite(t *testing.T) {
	const w, h = 480, 300
	for _, preset := range Presets {
		t.Run(preset, func(t *testing.T) {
			res, err := Render(Request{Preset: preset, Width: w, Height: h, Seed: 7})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if len(res.Planes) == 0 {
				t.Fatal("a generator declared no planes; every generator draws at least one")
			}
			if len(res.PlaneImages) != len(res.Planes) {
				t.Fatalf("%d planes named and %d images returned", len(res.Planes), len(res.PlaneImages))
			}

			composite, err := png.Decode(bytes.NewReader(res.PNG))
			if err != nil {
				t.Fatalf("decode composite: %v", err)
			}
			flattened := image.NewNRGBA(image.Rect(0, 0, w, h))
			for i := range res.Planes {
				plane, decodeErr := png.Decode(bytes.NewReader(res.PlaneImages[i]))
				if decodeErr != nil {
					t.Fatalf("decode plane %q: %v", res.Planes[i], decodeErr)
				}
				sourceOver(flattened, plane)
			}

			worst, at := 0.0, image.Point{}
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					cr, cg, cb, _ := composite.At(x, y).RGBA()
					f := flattened.NRGBAAt(x, y)
					for _, d := range []float64{
						math.Abs(float64(cr>>8) - float64(f.R)),
						math.Abs(float64(cg>>8) - float64(f.G)),
						math.Abs(float64(cb>>8) - float64(f.B)),
					} {
						if d > worst {
							worst, at = d, image.Point{X: x, Y: y}
						}
					}
				}
			}
			// One 8-bit step of slack, and no more: the flat buffer rounds once
			// per write while the plane path rounds once per write and once per
			// flatten, so a single least-significant bit can differ. Anything
			// larger is a real difference in what the generator drew.
			if worst > 1 {
				t.Fatalf("%s: flattening its %d planes differs from its composite by %.0f/255 at %v; the plane separation changed the picture",
					preset, len(res.Planes), worst, at)
			}
		})
	}
}

// The generator the plan names: a horizon really does ship four layers, not one.
func TestTheHorizonSeparatesItsFourLayers(t *testing.T) {
	res, err := Render(Request{Preset: "horizon", Width: 480, Height: 300, Seed: 7})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := []string{"sky", "sea", "headlands", "bank"}
	if len(res.Planes) != len(want) {
		t.Fatalf("horizon drew planes %v, want %v", res.Planes, want)
	}
	for i, name := range want {
		if res.Planes[i] != name {
			t.Fatalf("plane %d is %q, want %q (depth order matters: %v)", i, res.Planes[i], name, res.Planes)
		}
	}
	// Each layer must actually carry something, and they must differ. Four
	// identical planes would flatten correctly and be worthless.
	seen := map[string]bool{}
	for i, name := range res.Planes {
		if len(res.PlaneImages[i]) == 0 {
			t.Fatalf("plane %q carries no image", name)
		}
		key := string(res.PlaneImages[i][:min(512, len(res.PlaneImages[i]))])
		if seen[key] {
			t.Fatalf("plane %q duplicates an earlier layer", name)
		}
		seen[key] = true
	}
}

// A plane must be transparent where its layer does not draw. Without that the
// stack is opaque rectangles and the layer beneath can never show through,
// which is the whole point of separating them.
func TestAPlaneIsTransparentWhereItDoesNotDraw(t *testing.T) {
	res, err := Render(Request{Preset: "horizon", Width: 480, Height: 300, Seed: 7})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The bank is the last layer and sits along the bottom edge, so the top of
	// its plane must be empty.
	bank, err := png.Decode(bytes.NewReader(res.PlaneImages[len(res.PlaneImages)-1]))
	if err != nil {
		t.Fatalf("decode bank: %v", err)
	}
	if _, _, _, a := bank.At(240, 20).RGBA(); a != 0 {
		t.Fatalf("the foreground bank plane is opaque at the top of the frame (alpha %d); it would hide the sky", a>>8)
	}
	if _, _, _, a := bank.At(240, 290).RGBA(); a == 0 {
		t.Fatal("the foreground bank plane is transparent along the bottom edge, where it draws")
	}
}

// sourceOver composites src over dst, straight alpha.
func sourceOver(dst *image.NRGBA, src image.Image) {
	b := dst.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			sr, sg, sb, sa := src.At(x, y).RGBA()
			a := float64(sa) / 65535
			if a <= 0 {
				continue
			}
			var srcR, srcG, srcB float64
			if sa > 0 {
				srcR = float64(sr) / float64(sa) * 255
				srcG = float64(sg) / float64(sa) * 255
				srcB = float64(sb) / float64(sa) * 255
			}
			d := dst.NRGBAAt(x, y)
			da := float64(d.A) / 255
			outA := a + da*(1-a)
			if outA <= 0 {
				continue
			}
			mix := func(s, e float64) uint8 {
				return clamp8((s*a + e*da*(1-a)) / outA)
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: mix(srcR, float64(d.R)), G: mix(srcG, float64(d.G)), B: mix(srcB, float64(d.B)),
				A: clamp8(outA * 255),
			})
		}
	}
}

// The terrain's ridges are separate layers, and their haze runs light-to-dark
// back-to-front. That gradient IS the depth cue the flat lane could only
// suggest — a consumer moving these against each other gets real parallax.
func TestTheTerrainSeparatesItsRidges(t *testing.T) {
	res, err := Render(Request{Preset: "terrain", Width: 480, Height: 300, Seed: 7})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := []string{"sky", "ridge-1", "ridge-2", "ridge-3", "ridge-4", "ridge-5", "canopy"}
	if len(res.Planes) != len(want) {
		t.Fatalf("terrain drew %d planes %v, want %d %v", len(res.Planes), res.Planes, len(want), want)
	}
	for i, name := range want {
		if res.Planes[i] != name {
			t.Fatalf("plane %d is %q, want %q", i, res.Planes[i], name)
		}
	}

	// Ridges darken toward the viewer. Measured on each ridge's own opaque
	// pixels, because a plane is mostly transparent and averaging that in would
	// measure coverage rather than tone.
	previous := math.MaxFloat64
	for i := 1; i <= 5; i++ {
		lightness := meanOpaqueLightness(t, res.PlaneImages[i])
		if lightness > previous {
			t.Fatalf("ridge-%d is lighter (%.1f) than the ridge behind it (%.1f); the atmospheric haze runs backwards", i, lightness, previous)
		}
		previous = lightness
	}
}

// meanOpaqueLightness averages a plane's tone over the pixels it actually draws.
func meanOpaqueLightness(t *testing.T, encoded []byte) float64 {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode plane: %v", err)
	}
	b := img.Bounds()
	total, samples := 0.0, 0.0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a < 0xC000 {
				continue
			}
			total += (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 257
			samples++
		}
	}
	if samples == 0 {
		t.Fatal("a ridge plane draws no opaque pixels")
	}
	return total / samples
}
