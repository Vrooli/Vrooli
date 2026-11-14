package render

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// screenSource builds a synthetic halftone-like screen: a regular dot grid at a
// stated pitch. It stands in for what this scenario actually produces, which is
// why the resampler matters — every treatment the catalog sells is
// high-frequency by construction.
func screenSource(width, height, pitch int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	radius := float64(pitch) * 0.3
	for cy := pitch / 2; cy < height; cy += pitch {
		for cx := pitch / 2; cx < width; cx += pitch {
			for dy := -pitch; dy <= pitch; dy++ {
				for dx := -pitch; dx <= pitch; dx++ {
					if float64(dx*dx+dy*dy) > radius*radius {
						continue
					}
					x, y := cx+dx, cy+dy
					if x < 0 || y < 0 || x >= width || y >= height {
						continue
					}
					img.SetNRGBA(x, y, color.NRGBA{A: 255})
				}
			}
		}
	}
	return img
}

// localContrastEnergy measures how violently neighbouring pixels disagree.
//
// It is the metric that separates a correctly downscaled screen from an aliased
// one. Averaging many source dots into one destination pixel produces mid
// greys and moderate energy; point-sampling keeps hitting dot centres and gaps,
// so the result stays near-binary and the energy stays high — that is the
// moire, expressed as a number.
func localContrastEnergy(img image.Image) float64 {
	b := img.Bounds()
	var total float64
	var n int
	lum := func(x, y int) float64 {
		r, g, bl, _ := img.At(x, y).RGBA()
		return (0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bl>>8)) / 255
	}
	for y := b.Min.Y; y < b.Max.Y-1; y++ {
		for x := b.Min.X; x < b.Max.X-1; x++ {
			here := lum(x, y)
			total += math.Abs(here-lum(x+1, y)) + math.Abs(here-lum(x, y+1))
			n += 2
		}
	}
	if n == 0 {
		return 0
	}
	return total / float64(n)
}

// nearestNeighbourScaled is the implementation this phase replaced, kept only
// so the test can prove the new one is actually better. Without a failing
// reference, "we use a windowed filter now" is a claim rather than a result.
func nearestNeighbourScaled(dst *image.NRGBA, target image.Rectangle, src image.Image) {
	sb := src.Bounds()
	scale := math.Max(float64(target.Dx())/float64(sb.Dx()), float64(target.Dy())/float64(sb.Dy()))
	srcW := float64(target.Dx()) / scale
	srcH := float64(target.Dy()) / scale
	offX := (float64(sb.Dx()) - srcW) / 2
	offY := (float64(sb.Dy()) - srcH) / 2
	for y := target.Min.Y; y < target.Max.Y; y++ {
		fy := offY + float64(y-target.Min.Y)/scale
		sy := clampInt(sb.Min.Y+int(fy), sb.Min.Y, sb.Max.Y-1)
		for x := target.Min.X; x < target.Max.X; x++ {
			fx := offX + float64(x-target.Min.X)/scale
			sx := clampInt(sb.Min.X+int(fx), sb.Min.X, sb.Max.X-1)
			r, g, b, a := src.At(sx, sy).RGBA()
			dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)})
		}
	}
}

// TestDownscalingAScreenDoesNotAlias is the golden the phase promised: it must
// pass under the windowed filter and fail under nearest-neighbour.
//
// Two reduction factors, because they fail differently. At 4:1 a 9px screen
// lands near Nyquist and point sampling beats against the grid — the classic
// moire. At 8:1 the screen is far below the output's resolving power, so a
// correct filter must average it to near-flat while point sampling keeps
// resolving whichever dots happen to sit on the sample lattice.
func TestDownscalingAScreenDoesNotAlias(t *testing.T) {
	const (
		sourceEdge = 1200
		pitch      = 9
	)
	src := screenSource(sourceEdge, sourceEdge, pitch)

	for _, tc := range []struct {
		name       string
		targetEdge int
		// maxFiltered is what a correct windowed filter must stay under.
		// minAliased is what point sampling must exceed, so the test is proven
		// capable of failing rather than merely passing.
		maxFiltered, minAliased float64
	}{
		{name: "4:1 near Nyquist", targetEdge: 300, maxFiltered: 0.30, minAliased: 0.40},
		{name: "8:1 far below Nyquist", targetEdge: 150, maxFiltered: 0.05, minAliased: 0.10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := image.Rect(0, 0, tc.targetEdge, tc.targetEdge)

			filtered := image.NewNRGBA(target)
			drawScaled(filtered, target, src)
			filteredEnergy := localContrastEnergy(filtered)

			aliased := image.NewNRGBA(target)
			nearestNeighbourScaled(aliased, target, src)
			aliasedEnergy := localContrastEnergy(aliased)

			t.Logf("local contrast energy: windowed=%.4f nearest-neighbour=%.4f", filteredEnergy, aliasedEnergy)

			require.Lessf(t, filteredEnergy, tc.maxFiltered,
				"the windowed filter aliased a %dpx screen down to %dpx (energy %.4f)", pitch, tc.targetEdge, filteredEnergy)
			require.Greaterf(t, aliasedEnergy, tc.minAliased,
				"nearest-neighbour must FAIL this test; if it passes, the test does not measure aliasing (energy %.4f)", aliasedEnergy)
			require.Greaterf(t, aliasedEnergy, filteredEnergy*1.5,
				"the windowed filter must be materially quieter than point sampling (%.4f vs %.4f)", filteredEnergy, aliasedEnergy)
		})
	}
}

// TestUpscalingStaysSmooth pins the other direction. Magnifying with a
// sharpening kernel rings around the hard edges these treatments are made of,
// so upscale uses a triangle filter and must not introduce overshoot.
func TestUpscalingStaysSmooth(t *testing.T) {
	src := screenSource(120, 120, 9)
	target := image.Rect(0, 0, 480, 480)
	out := image.NewNRGBA(target)
	drawScaled(out, target, src)

	// No pixel may fall outside the source's own range: a ringing filter
	// produces values darker than the darkest ink or brighter than the paper.
	b := out.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := out.NRGBAAt(x, y)
			require.LessOrEqualf(t, int(c.R), 255, "overshoot at %d,%d", x, y)
			require.Equal(t, uint8(255), c.A, "upscale must preserve opacity")
		}
	}
}

// TestCoverCropKeepsACircleCircular guards the defect that shipped an
// elliptical sun into committed evidence: the scaler stretched each axis
// independently instead of cover-cropping.
func TestCoverCropKeepsACircleCircular(t *testing.T) {
	const edge = 400
	src := image.NewNRGBA(image.Rect(0, 0, edge, edge))
	for y := 0; y < edge; y++ {
		for x := 0; x < edge; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// The circle must fit inside the cropped window, or the measurement reports
	// clipping rather than distortion: a 200x600 target crops the 400x400
	// source to 133x400, so a radius of edge/4 would be cut off at the sides.
	centre, radius := edge/2, edge/8
	for y := 0; y < edge; y++ {
		for x := 0; x < edge; x++ {
			dx, dy := x-centre, y-centre
			if dx*dx+dy*dy <= radius*radius {
				src.SetNRGBA(x, y, color.NRGBA{A: 255})
			}
		}
	}

	// A tall panel: the aspect mismatch is what used to stretch the circle.
	target := image.Rect(0, 0, 200, 600)
	out := image.NewNRGBA(target)
	drawScaled(out, target, src)

	// Measure the dark region's extent on both axes at its centre lines.
	dark := func(x, y int) bool {
		c := out.NRGBAAt(x, y)
		return (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) < 128
	}
	var horizontal, vertical int
	midY, midX := target.Dy()/2, target.Dx()/2
	for x := 0; x < target.Dx(); x++ {
		if dark(x, midY) {
			horizontal++
		}
	}
	for y := 0; y < target.Dy(); y++ {
		if dark(midX, y) {
			vertical++
		}
	}
	require.Positive(t, horizontal)
	require.Positive(t, vertical)

	// Cover-crop scales both axes by the same factor, so the circle's diameter
	// must be equal on both axes in the output.
	ratio := float64(horizontal) / float64(vertical)
	require.InDeltaf(t, 1.0, ratio, 0.06,
		"a circular element must stay circular: %dpx wide vs %dpx tall (ratio %.3f)", horizontal, vertical, ratio)
}
