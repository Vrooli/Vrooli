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
