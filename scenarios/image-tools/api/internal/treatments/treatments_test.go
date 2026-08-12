package treatments

import (
	"bytes"
	"image"
	"image/color"
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
