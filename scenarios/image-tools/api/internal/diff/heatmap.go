package diff

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// defaultHighlight is the heat-map tint applied to changed pixels (magenta).
var defaultHighlight = color.NRGBA{R: 255, G: 0, B: 200, A: 255}

// heatmap renders a visual diff overlay: the base image is dimmed to grayscale,
// and each pixel is tinted toward the highlight colour in proportion to how much
// it changed (max per-channel delta, gated by tolerance). The result makes the
// changed regions pop while keeping the original as faint context — exactly what
// a reviewer (or test-genie) wants to see. Always PNG.
func heatmap(base, cmp *image.NRGBA, tolerance float64, highlightHex string) ([]byte, error) {
	highlight := defaultHighlight
	if c, ok := parseHex(highlightHex); ok {
		highlight = c
	}
	tolByte := tolerance * 255

	b := base.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		bi := base.PixOffset(base.Rect.Min.X, base.Rect.Min.Y+y)
		ci := cmp.PixOffset(cmp.Rect.Min.X, cmp.Rect.Min.Y+y)
		oi := out.PixOffset(0, y)
		brow := base.Pix[bi : bi+w*4]
		crow := cmp.Pix[ci : ci+w*4]
		orow := out.Pix[oi : oi+w*4]
		for x := 0; x < w; x++ {
			o := x * 4
			// Dimmed grayscale context from the base.
			g := uint8(luma(brow[o], brow[o+1], brow[o+2]) * 0.45)

			var maxDelta float64
			for ch := 0; ch < 4; ch++ {
				d := math.Abs(float64(brow[o+ch]) - float64(crow[o+ch]))
				if d > maxDelta {
					maxDelta = d
				}
			}
			// Intensity 0..1 of the change beyond tolerance.
			var t float64
			if maxDelta > tolByte {
				t = clamp01((maxDelta - tolByte) / (255 - tolByte + 1e-9))
				// Floor the visible tint so any change is perceptible.
				if t < 0.35 {
					t = 0.35
				}
			}
			orow[o] = lerp(g, highlight.R, t)
			orow[o+1] = lerp(g, highlight.G, t)
			orow[o+2] = lerp(g, highlight.B, t)
			orow[o+3] = 255
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func lerp(a, b uint8, t float64) uint8 {
	v := float64(a)*(1-t) + float64(b)*t
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return uint8(v + 0.5)
	}
}

// parseHex parses #rrggbb (with or without the leading #) into an opaque colour.
func parseHex(s string) (color.NRGBA, bool) {
	if s == "" {
		return color.NRGBA{}, false
	}
	if s[0] == '#' {
		s = s[1:]
	}
	if len(s) != 6 {
		return color.NRGBA{}, false
	}
	var rgb [3]uint8
	for i := 0; i < 3; i++ {
		hi, ok1 := hexNibble(s[i*2])
		lo, ok2 := hexNibble(s[i*2+1])
		if !ok1 || !ok2 {
			return color.NRGBA{}, false
		}
		rgb[i] = hi<<4 | lo
	}
	return color.NRGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}, true
}

func hexNibble(c byte) (uint8, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}
