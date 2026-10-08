package legibility

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

func solid(c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestSolidWhiteAgainstBlackMatchesWCAG(t *testing.T) {
	v, err := Measure(solid(color.Black), []Region{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "overlay", TextColor: "#ffffff"}}, 4.5, "")
	require.NoError(t, err)
	require.InDelta(t, 21, v.MinimumRatio, .001)
	require.True(t, v.Passes)
}

func TestFailureIncludesAmendment(t *testing.T) {
	v, err := Measure(solid(color.White), []Region{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "overlay", TextColor: "#ffffff"}}, 4.5, "")
	require.NoError(t, err)
	require.False(t, v.Passes)
	require.NotEmpty(t, v.Amendments)
}

// TestAMeasuredRegionIsTheRegionThatWasDeclared pins the geometry of the search
// itself.
//
// Every other test in this package builds a picture, measures it, and checks
// the number. None of them could see this: the crop's top edge was pinned to
// the top of the frame rather than to the region, and a fixture that is uniform
// above the copy scores identically either way. The defect only shows against a
// picture whose bands DIFFER — which is every real backdrop, and none of the
// fixtures.
//
// So this asserts on the search area rather than on a contrast number: dark
// ink is placed strictly outside the declared region and paper strictly inside
// it, and a measurement that reaches beyond its rectangle reports the ink.
func TestAMeasuredRegionIsTheRegionThatWasDeclared(t *testing.T) {
	const w, h = 400, 300
	region := Region{X: 0.10, Y: 0.55, Width: 0.40, Height: 0.30, TextColor: "#000000"}
	x0, y0 := int(region.X*w), int(region.Y*h)
	x1, y1 := int((region.X+region.Width)*w), int((region.Y+region.Height)*h)

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ink := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
			if x >= x0 && x < x1 && y >= y0 && y < y1 {
				ink = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
			img.SetNRGBA(x, y, ink)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	verdict, err := Measure(buf.Bytes(), []Region{region}, 4.5, "")
	require.NoError(t, err)
	// White paper against black type is 21:1, the maximum the ratio can express.
	// Anything less means the search read a pixel that was not in the region.
	require.InDelta(t, 21.0, verdict.MinimumRatio, 0.01,
		"the measurement reached outside the declared rectangle: everything inside it is paper")
}

// TestEveryEdgeOfTheSearchAreaIsHonoured is the same guard from the other side.
//
// The previous test would still pass if the search were too SMALL, and a search
// that is too small is the more dangerous error of the two — it reports a
// headline as legible over ink it never looked at. Here the region is paper
// except for one dark pixel in each corner, so a search that clips any edge
// misses a corner and reports the paper.
func TestEveryEdgeOfTheSearchAreaIsHonoured(t *testing.T) {
	const w, h = 400, 300
	region := Region{X: 0.20, Y: 0.30, Width: 0.50, Height: 0.40, TextColor: "#000000"}
	x0, y0 := int(region.X*w), int(region.Y*h)
	x1, y1 := int((region.X+region.Width)*w)-1, int((region.Y+region.Height)*h)-1

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for _, corner := range [][2]int{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}} {
		img.SetNRGBA(corner[0], corner[1], color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	verdict, err := Measure(buf.Bytes(), []Region{region}, 4.5, "")
	require.NoError(t, err)
	require.InDelta(t, 1.0, verdict.MinimumRatio, 0.01,
		"a corner of the declared rectangle was never searched: each one holds black on black")
}
