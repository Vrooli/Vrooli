package treatments

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func fixture() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 64; x++ {
			checker := uint8(0)
			if (x/4+y/4)%2 == 1 {
				checker = 32
			}
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x*4) + checker, G: uint8(y*5) + checker, B: uint8((x+y)*2) + checker, A: 255})
		}
	}
	return img
}

var goldenCases = map[string]func(image.Image) (image.Image, error){
	"duotone": func(img image.Image) (image.Image, error) {
		return Duotone(img, Params{Dark: "#15152b", Light: "#f6d58a", Mid: "#bb426b", MidLow: .38, MidHigh: .62})
	},
	"posterize": func(img image.Image) (image.Image, error) {
		return Posterize(img, Params{Levels: 5, Dark: "#111827", Light: "#fef3c7"})
	},
	"halftone": func(img image.Image) (image.Image, error) {
		return Halftone(img, Params{LPI: 6, Angle: 22, Dot: "circle", Dark: "#111827", Light: "#fef3c7"})
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
	for name, fn := range goldenCases {
		name, fn := name, fn
		t.Run(name, func(t *testing.T) {
			got := encode(t, runTreatment(t, fn))
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
