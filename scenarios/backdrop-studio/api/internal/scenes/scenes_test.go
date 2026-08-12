package scenes

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"testing"
)

func decode(t *testing.T, raw []byte) *image.NRGBA {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := image.NewNRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			out.Pix[out.PixOffset(x, y)+0] = uint8(r >> 8)
			out.Pix[out.PixOffset(x, y)+1] = uint8(g >> 8)
			out.Pix[out.PixOffset(x, y)+2] = uint8(b >> 8)
			out.Pix[out.PixOffset(x, y)+3] = 255
		}
	}
	return out
}

func render(t *testing.T, req Request) Result {
	t.Helper()
	r, err := Render(req)
	if err != nil {
		t.Fatalf("render %s: %v", req.Preset, err)
	}
	return r
}

// lum is a plain perceptual-ish luma, adequate for range checks.
func lum(r, g, b uint8) float64 {
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255
}

// TestScenesAreDeterministic pins the contract the whole pipeline rests on: a
// seed reproduces a scene exactly, so provenance means something.
func TestScenesAreDeterministic(t *testing.T) {
	for _, p := range Presets {
		a := render(t, Request{Preset: p, Width: 240, Height: 150, Seed: 7})
		b := render(t, Request{Preset: p, Width: 240, Height: 150, Seed: 7})
		if a.SHA256 != b.SHA256 {
			t.Errorf("%s: same seed produced different output", p)
		}
		c := render(t, Request{Preset: p, Width: 240, Height: 150, Seed: 8})
		if a.SHA256 == c.SHA256 {
			t.Errorf("%s: different seeds produced identical output", p)
		}
	}
}

// TestScenesSpanFullTonalRange is the load-bearing quality assertion. Every
// ink-mapping treatment downstream distributes its inks across the tones a
// scene provides; a scene confined to the midtones yields a flat duotone and a
// uniform halftone no matter how correct the treatment code is. The old
// scaffold-as-scene output failed this — drawArcade emitted near-black shapes
// on a near-black ground with no light anywhere.
func TestScenesSpanFullTonalRange(t *testing.T) {
	for _, p := range Presets {
		img := decode(t, render(t, Request{Preset: p, Width: 480, Height: 300, Seed: 7}).PNG)
		const buckets = 64
		var hist [buckets]int
		total := 0
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := img.NRGBAAt(x, y)
				hist[int(lum(c.R, c.G, c.B)*(buckets-1))]++
				total++
			}
		}
		pct := func(f float64) float64 {
			want, seen := int(float64(total)*f), 0
			for i := 0; i < buckets; i++ {
				if seen += hist[i]; seen >= want {
					return float64(i) / (buckets - 1)
				}
			}
			return 1
		}
		lo, hi := pct(0.02), pct(0.98)
		if lo > 0.18 {
			t.Errorf("%s: p2 luma is %.3f — the scene has no dark values to map ink into", p, lo)
		}
		if hi < 0.80 {
			t.Errorf("%s: p98 luma is %.3f — the scene has no highlight to map paper into", p, hi)
		}
		occupied := 0
		for _, n := range hist {
			if float64(n)/float64(total) > 0.0008 {
				occupied++
			}
		}
		if frac := float64(occupied) / buckets; frac < 0.5 {
			t.Errorf("%s: only %.0f%% of tonal buckets are occupied — the histogram is too sparse to screen", p, frac*100)
		}
	}
}

// TestSceneNoiseIsCoherent distinguishes a real scene from speckle. The
// scaffold generators used a per-pixel xorshift, so adjacent pixels were
// uncorrelated; anything built on coherent noise has neighbours that agree far
// more often than chance.
func TestSceneNoiseIsCoherent(t *testing.T) {
	for _, p := range Presets {
		img := decode(t, render(t, Request{Preset: p, Width: 480, Height: 300, Seed: 7}).PNG)
		var sum float64
		n := 0
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X-1; x++ {
				a := img.NRGBAAt(x, y)
				c := img.NRGBAAt(x+1, y)
				sum += math.Abs(lum(a.R, a.G, a.B) - lum(c.R, c.G, c.B))
				n++
			}
		}
		if mean := sum / float64(n); mean > 0.045 {
			t.Errorf("%s: mean neighbour delta %.4f is speckle-like; scenes need coherent noise", p, mean)
		}
	}
}

// TestFocalParamZeroIsHonoured is the regression test for the clamp bug carried
// over from scaffold.go, where 0 was treated as "unset" and silently replaced
// by the fallback — making the left edge unreachable.
func TestFocalParamZeroIsHonoured(t *testing.T) {
	zero := render(t, Request{Preset: "horizon", Width: 320, Height: 200, Seed: 7, ParamsJSON: `{"focal_x":0}`})
	def := render(t, Request{Preset: "horizon", Width: 320, Height: 200, Seed: 7})
	if zero.SHA256 == def.SHA256 {
		t.Fatal("focal_x=0 produced the default render; an explicit zero is being discarded")
	}
	// and it must differ from the opposite edge too
	one := render(t, Request{Preset: "horizon", Width: 320, Height: 200, Seed: 7, ParamsJSON: `{"focal_x":1}`})
	if zero.SHA256 == one.SHA256 {
		t.Fatal("focal_x=0 and focal_x=1 produced identical output")
	}
}

func TestSceneRequestValidation(t *testing.T) {
	if _, err := Render(Request{Preset: "nope", Width: 64, Height: 64}); err == nil {
		t.Error("unknown preset accepted")
	}
	if _, err := Render(Request{Preset: "horizon", Width: 4, Height: 64}); err == nil {
		t.Error("undersized render accepted")
	}
	if _, err := Render(Request{Preset: "horizon", Width: 64, Height: 64, ParamsJSON: "{oops"}); err == nil {
		t.Error("invalid params_json accepted")
	}
}

// TestScenesScaleWithSize guards against a generator whose features are sized
// in raw pixels: the same seed at two sizes must stay recognisably the same
// composition, which shows up as a similar tonal distribution.
func TestScenesScaleWithSize(t *testing.T) {
	for _, p := range Presets {
		mean := func(w, h int) float64 {
			img := decode(t, render(t, Request{Preset: p, Width: w, Height: h, Seed: 7}).PNG)
			var s float64
			n := 0
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					c := img.NRGBAAt(x, y)
					s += lum(c.R, c.G, c.B)
					n++
				}
			}
			return s / float64(n)
		}
		small, large := mean(320, 200), mean(960, 600)
		if math.Abs(small-large) > 0.08 {
			t.Errorf("%s: mean luma drifts with size (%.3f at 320px vs %.3f at 960px); features are pixel-sized, not relative", p, small, large)
		}
	}
}
