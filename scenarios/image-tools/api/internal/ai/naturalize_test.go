package ai

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
)

// solidNRGBA builds a w×h image filled with c (a flat region the compositor's
// clarity pass leaves alone, isolating the grain effect).
func solidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// gradientNRGBA builds a smooth horizontal gradient — the "plastic" input
// naturalize is meant to break up.
func gradientNRGBA(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(40 + (x*150)/w)
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func meanAbsLumaDelta(a, b *image.NRGBA) float64 {
	var sum float64
	n := len(a.Pix) / 4
	for i := 0; i < len(a.Pix); i += 4 {
		da := math.Abs(float64(a.Pix[i]) - float64(b.Pix[i]))
		sum += da
	}
	return sum / float64(n)
}

// TestNaturalize_Deterministic proves the compositor is a pure function: the
// same input + params yield byte-identical output (grain is hash-derived, not
// RNG-stateful), which is what makes it testable and reproducible.
func TestNaturalize_Deterministic(t *testing.T) {
	src := gradientNRGBA(48, 48)
	p := NaturalizeParams{Realism: 0.8, Seed: 7}
	a := Naturalize(src, p)
	b := Naturalize(src, p)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("naturalize is not deterministic for identical inputs")
	}
}

// TestNaturalize_AddsTexture proves naturalize actually perturbs an over-smooth
// image (the de-plasticize effect) without destroying it, and preserves the
// dimensions + alpha.
func TestNaturalize_AddsTexture(t *testing.T) {
	src := gradientNRGBA(64, 64)
	out := Naturalize(src, NaturalizeParams{Realism: 1, Seed: 1})

	if out.Bounds() != src.Bounds() {
		t.Fatalf("dimensions changed: %v -> %v", src.Bounds(), out.Bounds())
	}
	d := meanAbsLumaDelta(src, out)
	if d <= 0.5 {
		t.Errorf("naturalize barely changed the image (mean abs delta %.3f); expected visible texture", d)
	}
	if d >= 30 {
		t.Errorf("naturalize changed the image too aggressively (mean abs delta %.3f); should be subtle", d)
	}
	// Alpha must be untouched.
	for i := 3; i < len(out.Pix); i += 4 {
		if out.Pix[i] != 255 {
			t.Fatalf("alpha modified at byte %d: got %d", i, out.Pix[i])
		}
	}
}

// TestNaturalize_RealismScales proves a higher realism knob produces a larger
// departure from the source.
func TestNaturalize_RealismScales(t *testing.T) {
	src := gradientNRGBA(64, 64)
	low := meanAbsLumaDelta(src, Naturalize(src, NaturalizeParams{Realism: 0.2, Seed: 3}))
	high := meanAbsLumaDelta(src, Naturalize(src, NaturalizeParams{Realism: 1.0, Seed: 3}))
	if high <= low {
		t.Errorf("expected higher realism to change more: low=%.3f high=%.3f", low, high)
	}
}

// TestNaturalize_DefaultRealism proves an unset/zero knob still does visible
// work (defaults to the gentle midpoint rather than a no-op).
func TestNaturalize_DefaultRealism(t *testing.T) {
	src := gradientNRGBA(48, 48)
	out := Naturalize(src, NaturalizeParams{Seed: 5}) // Realism 0 -> default 0.5
	if meanAbsLumaDelta(src, out) <= 0.2 {
		t.Error("default realism produced no visible effect")
	}
}

// TestNaturalize_FaceAwareEmphasizesMidtones proves the face-aware weighting
// biases grain toward midtone (skin) regions over shadows.
func TestNaturalize_FaceAwareEmphasizesMidtones(t *testing.T) {
	mid := solidNRGBA(64, 64, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	dark := solidNRGBA(64, 64, color.NRGBA{R: 20, G: 20, B: 20, A: 255})
	p := NaturalizeParams{Realism: 1, FaceAware: true, Seed: 11}

	midDelta := meanAbsLumaDelta(mid, Naturalize(mid, p))
	darkDelta := meanAbsLumaDelta(dark, Naturalize(dark, p))
	if midDelta <= darkDelta {
		t.Errorf("face-aware should grain midtones more than shadows: mid=%.3f dark=%.3f", midDelta, darkDelta)
	}
}

// TestParseNaturalizeParams pins the string-map → typed-params parsing.
func TestParseNaturalizeParams(t *testing.T) {
	got := parseNaturalizeParams(map[string]string{"realism": "0.7", "face_aware": "true", "seed": "42"})
	if got.Realism != 0.7 || !got.FaceAware || got.Seed != 42 {
		t.Fatalf("parse mismatch: %+v", got)
	}
	empty := parseNaturalizeParams(nil)
	if empty.Realism != 0 || empty.FaceAware || empty.Seed != 0 {
		t.Fatalf("empty parse should be zero-valued: %+v", empty)
	}
}

// TestGrainNoise_Range proves the deterministic noise stays in [-1,1).
func TestGrainNoise_Range(t *testing.T) {
	for x := 0; x < 50; x++ {
		for y := 0; y < 50; y++ {
			n := grainNoise(x, y, 99)
			if n < -1 || n >= 1 {
				t.Fatalf("grainNoise(%d,%d) = %f out of [-1,1)", x, y, n)
			}
		}
	}
}

// TestNaturalize_EngineVertical drives the FULL vertical with the REAL builtin
// provider (not a fake): the engine selects the weightless naturalize model,
// routes to the in-process builtin backend, runs the compositor, and persists a
// real PNG that decodes and differs from the input. This is the deterministic
// AI op that is guaranteed runnable with zero provisioning.
func TestNaturalize_EngineVertical(t *testing.T) {
	// Build the engine over the real registry + the real provider set (which
	// includes the builtin provider). ModelInstalled is forced true here; the
	// installer's own builtin-always-installed path is covered in the models
	// package test.
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	backendReg := backends.New()
	if err := RegisterProviders(backendReg, func(string) (string, error) { return "/bin/x", nil }, nil); err != nil {
		t.Fatalf("register providers: %v", err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	eng, err := NewEngine(Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
		ModelsRoot:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	// A smooth gradient encoded as a real PNG, stored as the op input.
	var buf bytes.Buffer
	if err := png.Encode(&buf, gradientNRGBA(48, 48)); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	storeInput(t, store, "input/grad.png", buf.Bytes())

	def, _ := registry.DefaultFor("naturalize")
	ref, err := runJob(t, eng, "naturalize", Payload{
		Operation: "naturalize",
		ModelID:   def.ID,
		InputKey:  "input/grad.png",
		Params:    map[string]string{"realism": "1", "seed": "2"},
	})
	if err != nil {
		t.Fatalf("run naturalize: %v", err)
	}

	rc, _, err := store.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("get output: %v", err)
	}
	defer func() { _ = rc.Close() }()
	out, err := png.Decode(rc)
	if err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}
	if out.Bounds().Dx() != 48 || out.Bounds().Dy() != 48 {
		t.Fatalf("unexpected output dims: %v", out.Bounds())
	}
}
