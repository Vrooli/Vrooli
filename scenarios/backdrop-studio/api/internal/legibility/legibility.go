package legibility

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	// Registers the JPEG decoder. Candidates arrive as PNG, but an operator
	// measuring a supplied comparison image should not get "unknown format"
	// for a JPEG.
	_ "image/jpeg"
	"math"
	"strconv"
	"strings"
)

type Region struct {
	X, Y, Width, Height float64
	Kind, TextColor     string
}
type RegionVerdict struct {
	Index        int
	MinimumRatio float64
	Passes       bool
}
type Amendment struct {
	Kind, Description string
	Value             float64
}
type Verdict struct {
	Passes                  bool
	MinimumRatio, Threshold float64
	Regions                 []RegionVerdict
	Amendments              []Amendment
	Placement               string
}

func Measure(pngBytes []byte, regions []Region, threshold float64, placement string) (Verdict, error) {
	if threshold <= 0 {
		threshold = 4.5
	}
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return Verdict{}, fmt.Errorf("legibility: decode image: %w", err)
	}
	out := Verdict{Passes: true, Threshold: threshold, MinimumRatio: math.Inf(1), Placement: placement}
	for i, region := range regions {
		if region.Kind == "occlusion" {
			continue
		}
		if region.Width <= 0 || region.Height <= 0 {
			return Verdict{}, fmt.Errorf("legibility: region %d has invalid geometry", i)
		}
		text, err := parseHex(region.TextColor)
		if err != nil {
			return Verdict{}, fmt.Errorf("legibility: region %d: %w", i, err)
		}
		min := math.Inf(1)
		x0, y0, x1, y1 := cropBounds(img.Bounds(), region, placement)
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				ratio := contrast(luminance(text), luminance(img.At(x, y)))
				if ratio < min {
					min = ratio
				}
			}
		}
		if math.IsInf(min, 1) {
			return Verdict{}, fmt.Errorf("legibility: region %d contains no pixels", i)
		}
		passes := min >= threshold
		out.Regions = append(out.Regions, RegionVerdict{Index: i, MinimumRatio: min, Passes: passes})
		if min < out.MinimumRatio {
			out.MinimumRatio = min
		}
		if !passes {
			out.Passes = false
		}
	}
	if math.IsInf(out.MinimumRatio, 1) {
		out.MinimumRatio = 0
	}
	if !out.Passes {
		opacity := minimumScrim(out.MinimumRatio, out.Threshold)
		out.Amendments = []Amendment{{Kind: "scrim", Description: fmt.Sprintf("Apply a black scrim at %.3f opacity, or choose another seed/placement.", opacity), Value: opacity}}
	}
	return out, nil
}

func cropBounds(bounds image.Rectangle, r Region, placement string) (int, int, int, int) {
	x, y, w, h := 0.0, 0.0, float64(bounds.Dx()), float64(bounds.Dy())
	if strings.Contains(strings.ToLower(placement), "mobile") {
		x = w * .08
		w = w * .72
	}
	// y + r.Y*h, matching the x term beside it. This read `y * r.Y` — a multiply
	// where every other term adds, against an origin that is always zero — so
	// the top edge was pinned to the top of the FRAME however far down the copy
	// actually sat. Every region was measured as though it began at y=0, which
	// swept the whole band above the copy into the worst-pixel search and
	// returned that band's darkest ink as the headline's contrast.
	//
	// The reading was wrong in the direction that hides work: a region measured
	// taller than it is can only score lower, so a repair could be complete and
	// still report failure. Two styles appeared to respond to a generator quiet
	// zone here while twenty-one did not, and the two were the ones whose copy
	// sits at the top of the frame — the only place the defect was harmless.
	return int(x + r.X*w), int(y + r.Y*h), max(int(x+(r.X+r.Width)*w), int(x)+1), max(int(y+(r.Y+r.Height)*h), int(y)+1)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func parseHex(v string) (color.RGBA, error) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "#")
	if len(v) != 6 {
		return color.RGBA{}, fmt.Errorf("text color %q must be #RRGGBB", v)
	}
	n, err := strconv.ParseUint(v, 16, 32)
	return color.RGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}, err
}

func linear(c uint8) float64 {
	v := float64(c) / 255
	if v <= .04045 {
		return v / 12.92
	}
	return math.Pow((v+.055)/1.055, 2.4)
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return .2126*linear(uint8(r>>8)) + .7152*linear(uint8(g>>8)) + .0722*linear(uint8(b>>8))
}

func contrast(a, b float64) float64 {
	if a < b {
		a, b = b, a
	}
	return (a + .05) / (b + .05)
}

func minimumScrim(current, threshold float64) float64 {
	if current <= 0 {
		return 1
	} // darkening a light background toward black
	if threshold <= 1 {
		return 0
	}
	needed := (current+.05)/threshold - .05
	opacity := 1 - needed/(current)
	if opacity < 0 {
		return 0
	}
	if opacity > 1 {
		return 1
	}
	return opacity
}
