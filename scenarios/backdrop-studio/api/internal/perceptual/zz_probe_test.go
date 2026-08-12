package perceptual

import (
	"bytes"
	"image"
	_ "image/png"
	"math"
	"os"
	"testing"
)

const probeDir = "/tmp/claude-1000/-home-matthalloran8-Vrooli/0d5a5dda-d98a-448e-903f-f55f3e5f2f8a/scratchpad/"

func load(t *testing.T, name string) image.Image {
	raw, err := os.ReadFile(probeDir + name)
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// edgeField: gradient magnitude reduced to a w*h grid.
func edgeField(img image.Image, w, h int) []float64 {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	f := make([]float64, w*h)
	n := make([]int, w*h)
	for y := 1; y < ih-1; y++ {
		cy := y * h / ih
		for x := 1; x < iw-1; x++ {
			cx := x * w / iw
			gx := lightnessAt(img, b.Min.X+x+1, b.Min.Y+y) - lightnessAt(img, b.Min.X+x-1, b.Min.Y+y)
			gy := lightnessAt(img, b.Min.X+x, b.Min.Y+y+1) - lightnessAt(img, b.Min.X+x, b.Min.Y+y-1)
			f[cy*w+cx] += math.Hypot(gx, gy)
			n[cy*w+cx]++
		}
	}
	for i := range f {
		if n[i] > 0 {
			f[i] /= float64(n[i])
		}
	}
	return f
}

func TestProbeMetrics(t *testing.T) {
	src := load(t, "scene-arcade.png")
	cases := []struct {
		name, file string
		good       bool
	}{
		{"op-art (GOOD)", "opart-now.png", true},
		{"cyanotype (GOOD)", "cyanotype-now.png", true},
		{"engraved (BAD)", "engraved-now.png", false},
	}
	for _, tc := range cases {
		out := load(t, tc.file)
		t.Logf("--- %s", tc.name)
		for _, g := range []struct{ w, h int }{{16, 9}, {32, 18}, {64, 36}, {128, 72}, {256, 144}} {
			c := math.Abs(correlation(lightnessField(src, g.w, g.h), lightnessField(out, g.w, g.h)))
			e := math.Abs(correlation(edgeField(src, g.w, g.h), edgeField(out, g.w, g.h)))
			t.Logf("    grid %3dx%-3d  tone-corr %.4f   edge-corr %.4f", g.w, g.h, c, e)
		}
		t.Logf("    ink modulation (16x9) %.4f", standardDeviation(inkCoverage(out, 16, 9)))
	}
}
