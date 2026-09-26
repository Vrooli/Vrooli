package ops

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// platePNG encodes a flat plate. Alpha is explicit because a compositor's whole
// job is what happens where a plate is transparent.
func platePNG(w, h int, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// halfPlatePNG is opaque on the left half and transparent on the right, which is
// what a real matte looks like and what a flat plate cannot exercise.
func halfPlatePNG(w, h int, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w/2; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func compose(t *testing.T, p *Params) *image.NRGBA {
	t.Helper()
	out, err := Composite(nil, p)
	if err != nil {
		t.Fatalf("composite: %v", err)
	}
	nrgba, ok := out.(*image.NRGBA)
	if !ok {
		t.Fatalf("composite returned %T, want *image.NRGBA", out)
	}
	return nrgba
}

// The three blend modes are asserted on the arithmetic a designer expects, not
// on "it produced something": multiply is ink on paper, screen is light adding,
// normal is placement.
func TestBlendModesDoWhatTheirNamesSay(t *testing.T) {
	const w, h = 8, 8
	grey := color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	half := color.NRGBA{R: 128, G: 128, B: 128, A: 255}

	for _, tc := range []struct {
		mode string
		want uint8
	}{
		// 0.502 * 0.502 ≈ 0.252 → 64
		{BlendMultiply, 64},
		// 1 - (1-0.502)^2 ≈ 0.752 → 192
		{BlendScreen, 192},
		// placement: the top plate wins outright
		{BlendNormal, 128},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			out := compose(t, &Params{Plates: []PlateSpec{
				{Name: "under", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, grey)},
				{Name: "over", Depth: 1, Blend: tc.mode, Opacity: 1, Image: platePNG(w, h, half)},
			}})
			got := out.NRGBAAt(4, 4)
			if diff := int(got.R) - int(tc.want); diff > 1 || diff < -1 {
				t.Fatalf("%s produced R=%d, want ~%d", tc.mode, got.R, tc.want)
			}
			if got.A != 255 {
				t.Fatalf("%s produced A=%d over an opaque base, want 255", tc.mode, got.A)
			}
		})
	}
}

// The failure this guards is the classic one: a multiply plate over an empty
// canvas comes out black, because multiplying by nothing is nothing. Where the
// canvas is transparent there is nothing to blend with, so the plate's own
// colour must show through.
func TestAMultiplyPlateOverAnEmptyCanvasKeepsItsOwnColour(t *testing.T) {
	const w, h = 8, 8
	red := color.NRGBA{R: 220, G: 40, B: 40, A: 255}
	out := compose(t, &Params{Plates: []PlateSpec{
		{Name: "ink", Depth: 0, Blend: BlendMultiply, Opacity: 1, Image: platePNG(w, h, red)},
	}})
	got := out.NRGBAAt(4, 4)
	if got.R != red.R || got.G != red.G || got.B != red.B {
		t.Fatalf("multiply over an empty canvas produced %v, want the plate's own colour %v", got, red)
	}
}

// Transparency is the point of a plate. Where a plate has no alpha, what is
// beneath it must survive untouched.
func TestATransparentRegionLetsTheLayerBeneathThrough(t *testing.T) {
	const w, h = 8, 8
	base := color.NRGBA{R: 10, G: 90, B: 200, A: 255}
	top := color.NRGBA{R: 240, G: 240, B: 60, A: 255}
	out := compose(t, &Params{Plates: []PlateSpec{
		{Name: "sky", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, base)},
		{Name: "figure", Depth: 1, Blend: BlendNormal, Opacity: 1, Image: halfPlatePNG(w, h, top)},
	}})
	if got := out.NRGBAAt(1, 4); got.R != top.R {
		t.Fatalf("covered region is %v, want the top plate %v", got, top)
	}
	if got := out.NRGBAAt(6, 4); got.R != base.R || got.B != base.B {
		t.Fatalf("uncovered region is %v, want the layer beneath %v", got, base)
	}
}

// Depth orders the stack, not list position. A caller reordering a stack should
// not have to rebuild the list, and the same stack must always return the same
// picture whatever order the list arrived in.
func TestDepthOrdersTheStackNotListPosition(t *testing.T) {
	const w, h = 8, 8
	under := color.NRGBA{R: 10, G: 90, B: 200, A: 255}
	over := color.NRGBA{R: 240, G: 240, B: 60, A: 255}

	inOrder := compose(t, &Params{Plates: []PlateSpec{
		{Name: "under", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, under)},
		{Name: "over", Depth: 1, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, over)},
	}})
	shuffled := compose(t, &Params{Plates: []PlateSpec{
		{Name: "over", Depth: 1, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, over)},
		{Name: "under", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, under)},
	}})
	if inOrder.NRGBAAt(4, 4) != shuffled.NRGBAAt(4, 4) {
		t.Fatalf("list order changed the picture: %v vs %v", inOrder.NRGBAAt(4, 4), shuffled.NRGBAAt(4, 4))
	}
	if got := inOrder.NRGBAAt(4, 4); got.R != over.R {
		t.Fatalf("the deeper plate won; got %v, want the higher-depth plate %v", got, over)
	}
}

// Same stack, same bytes, every time. Everything downstream — goldens, a
// reproduced release, a diff a reviewer reads — rests on this.
func TestTheCompositorIsDeterministic(t *testing.T) {
	const w, h = 24, 16
	build := func() *Params {
		return &Params{Plates: []PlateSpec{
			{Name: "a", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, color.NRGBA{R: 30, G: 60, B: 90, A: 255})},
			{Name: "b", Depth: 1, Blend: BlendMultiply, Opacity: 0.7, Image: halfPlatePNG(w, h, color.NRGBA{R: 200, G: 200, B: 40, A: 255})},
			{Name: "c", Depth: 2, Blend: BlendScreen, Opacity: 0.4, Image: platePNG(w, h, color.NRGBA{R: 80, G: 80, B: 80, A: 128})},
		}}
	}
	first, second := compose(t, build()), compose(t, build())
	if !bytes.Equal(first.Pix, second.Pix) {
		t.Fatal("two composites of one stack differ")
	}
}

// A blend mode outside the set is refused by name. Approximating it with the
// nearest one would silently change what a picture depicts, and nothing
// downstream could see it.
func TestAnUnknownBlendModeIsRefusedByName(t *testing.T) {
	_, err := Composite(nil, &Params{Plates: []PlateSpec{
		{Name: "glow", Depth: 0, Blend: "overlay", Opacity: 1, Image: platePNG(4, 4, color.NRGBA{A: 255})},
	}})
	if err == nil {
		t.Fatal("an unsupported blend mode was accepted")
	}
	for _, want := range []string{"overlay", "normal", "multiply", "screen", "glow"} {
		if !bytes.Contains([]byte(err.Error()), []byte(want)) {
			t.Fatalf("refusal %q does not mention %q", err, want)
		}
	}
}

func TestAPlateWithNoImageIsRefused(t *testing.T) {
	_, err := Composite(nil, &Params{Plates: []PlateSpec{{Name: "empty", Depth: 0, Opacity: 1}}})
	if err == nil {
		t.Fatal("a plate with no pixels was accepted")
	}
}

func TestAnEmptyStackIsRefused(t *testing.T) {
	if _, err := Composite(nil, &Params{}); err == nil {
		t.Fatal("an empty stack was accepted")
	}
}

// A single-plate stack must return that plate. This is the property the render
// path's byte-identity guarantee rests on: every existing style becomes a
// one-plate stack, and one plate through the compositor has to be a no-op.
func TestASinglePlateStackReturnsThatPlate(t *testing.T) {
	const w, h = 16, 12
	c := color.NRGBA{R: 143, G: 22, B: 91, A: 255}
	out := compose(t, &Params{Plates: []PlateSpec{
		{Name: "only", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, c)},
	}})
	if out.Bounds().Dx() != w || out.Bounds().Dy() != h {
		t.Fatalf("geometry %v, want %dx%d", out.Bounds(), w, h)
	}
	for _, pt := range [][2]int{{0, 0}, {w - 1, h - 1}, {w / 2, h / 2}} {
		if got := out.NRGBAAt(pt[0], pt[1]); got != c {
			t.Fatalf("at %v got %v, want the plate unchanged %v", pt, got, c)
		}
	}
}

// Plates of different geometry are sampled rather than refused: a model-derived
// matte legitimately comes back at the model's resolution, and refusing would
// push a resize into every caller.
func TestPlatesOfDifferentSizeAreSampledToTheCanvas(t *testing.T) {
	out := compose(t, &Params{
		Width: 32, Height: 16,
		Plates: []PlateSpec{
			{Name: "big", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(64, 32, color.NRGBA{R: 200, A: 255})},
			{Name: "small", Depth: 1, Blend: BlendNormal, Opacity: 1, Image: halfPlatePNG(8, 4, color.NRGBA{G: 200, A: 255})},
		},
	})
	if out.Bounds().Dx() != 32 || out.Bounds().Dy() != 16 {
		t.Fatalf("geometry %v, want the declared 32x16", out.Bounds())
	}
	if got := out.NRGBAAt(2, 8); got.G != 200 {
		t.Fatalf("the small plate did not reach the left half: %v", got)
	}
	if got := out.NRGBAAt(30, 8); got.R != 200 {
		t.Fatalf("the big plate did not show where the small one is transparent: %v", got)
	}
}

// Opacity scales a plate's contribution, and zero really means invisible.
func TestOpacityScalesAPlateAndZeroHidesIt(t *testing.T) {
	const w, h = 8, 8
	base := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	top := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	half := compose(t, &Params{Plates: []PlateSpec{
		{Name: "base", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, base)},
		{Name: "top", Depth: 1, Blend: BlendNormal, Opacity: 0.5, Image: platePNG(w, h, top)},
	}})
	if got := half.NRGBAAt(4, 4); got.R < 120 || got.R > 136 {
		t.Fatalf("half-opacity white over black gave R=%d, want ~128", got.R)
	}
	hidden := compose(t, &Params{Plates: []PlateSpec{
		{Name: "base", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, base)},
		{Name: "top", Depth: 1, Blend: BlendNormal, Opacity: 0, Image: platePNG(w, h, top)},
	}})
	if got := hidden.NRGBAAt(4, 4); got.R != 0 {
		t.Fatalf("a zero-opacity plate was drawn: %v", got)
	}
}

// Compositor goldens.
//
// Byte-stable at two frame sizes, because the compositor is the one step every
// plate-based candidate passes through: a change in its arithmetic changes every
// picture in the catalog at once, and a diff nobody can see is a change nobody
// reviews. Stored as a content hash rather than a PNG — the assertion is "did
// the arithmetic change", and a 60KB image in the repo answers that no better
// than 64 hex characters.
func TestCompositorGoldens(t *testing.T) {
	stack := func(w, h int) *Params {
		return &Params{
			Width: w, Height: h,
			Background: "#EDE6D2",
			Plates: []PlateSpec{
				{Name: "paper", Depth: 0, Blend: BlendNormal, Opacity: 1, Image: platePNG(w, h, color.NRGBA{R: 237, G: 230, B: 210, A: 255})},
				{Name: "ink", Depth: 1, Blend: BlendMultiply, Opacity: 0.85, Image: halfPlatePNG(w, h, color.NRGBA{R: 27, G: 63, B: 216, A: 255})},
				{Name: "glow", Depth: 2, Blend: BlendScreen, Opacity: 0.4, Image: platePNG(w, h, color.NRGBA{R: 90, G: 90, B: 40, A: 160})},
			},
		}
	}
	for _, tc := range []struct {
		name   string
		w, h   int
		digest string
	}{
		{"480x300", 480, 300, "f5f6a5cf81f379e6128fc1747a32330c289d8bdf07de68d88294fa818fef11be"},
		{"1440x720", 1440, 720, "338af87f84abc4b2cdaacf8e525d5c46f91e49911463d414ca541d1b73faba80"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := compose(t, stack(tc.w, tc.h))
			sum := sha256.Sum256(out.Pix)
			got := hex.EncodeToString(sum[:])
			if os.Getenv("UPDATE_COMPOSITOR_GOLDENS") != "" {
				t.Logf("%s digest: %s", tc.name, got)
				return
			}
			if got != tc.digest {
				t.Fatalf("compositor output changed at %s:\n  got  %s\n  want %s\nIf this change is intended, re-run with UPDATE_COMPOSITOR_GOLDENS=1 and paste the printed digests.", tc.name, got, tc.digest)
			}
		})
	}
}
