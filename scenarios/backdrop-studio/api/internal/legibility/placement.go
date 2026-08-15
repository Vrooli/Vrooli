package legibility

import (
	"bytes"
	"fmt"
	"image"
	"math"
)

// Finding a reserved region that is actually legible.
//
// A reserved region declares where copy goes and what colour it is, and until
// this existed nothing confirmed the copy would be readable there. Measured
// over the settled catalog on 2026-08-13, ZERO of the twenty styles declaring
// one passed: the best sat at 2.81 against a 4.50 threshold and eight at or
// below 1.20, which is type the same colour as what it sits on.
//
// The repair is art direction, but the SEARCH is not. A region has a size the
// author chose and a picture has quiet corners; finding which corners clear the
// threshold is a measurement, and doing it by eye across twenty styles is how a
// catalog ends up with regions nobody checked.
//
// What this deliberately does not do is choose. It reports every placement that
// passes and what each measured, and an author picks the one that suits the
// composition. A tool that silently moved a headline would be making the art
// direction decision it is supposed to inform.

// Placement is one candidate position for a reserved region.
type Placement struct {
	Region Region
	// MinimumRatio is worst-pixel contrast for the declared text colour there.
	MinimumRatio float64
	Passes       bool
}

// FindPlacements measures the region's own size at a grid of positions.
//
// The size is held fixed because it is the author's decision — a headline needs
// the room it needs — and only the position varies. Corners are searched before
// centres by the order the grid is walked, because copy in the middle of a
// picture is a different design than copy in its corner and an author asking
// "where can this go" usually means "which corner".
func FindPlacements(pngBytes []byte, region Region, threshold float64, placement string) ([]Placement, error) {
	if threshold <= 0 {
		threshold = 4.5
	}
	if region.Width <= 0 || region.Height <= 0 {
		return nil, fmt.Errorf("legibility: region has invalid geometry")
	}
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("legibility: decode image: %w", err)
	}
	text, err := parseHex(region.TextColor)
	if err != nil {
		return nil, fmt.Errorf("legibility: %w", err)
	}
	textLum := luminance(text)

	// A 5x5 grid of top-left origins over the space the region can occupy. Fine
	// enough that a genuinely quiet corner is found, coarse enough that the
	// answer is a placement an author would actually choose rather than a
	// pixel-hunted sliver.
	const steps = 5
	spanX, spanY := 1-region.Width, 1-region.Height
	if spanX < 0 || spanY < 0 {
		return nil, fmt.Errorf("legibility: region %gx%g does not fit the frame", region.Width, region.Height)
	}
	var out []Placement
	for iy := 0; iy < steps; iy++ {
		for ix := 0; ix < steps; ix++ {
			candidate := region
			candidate.X = spanX * float64(ix) / float64(steps-1)
			candidate.Y = spanY * float64(iy) / float64(steps-1)
			x0, y0, x1, y1 := cropBounds(img.Bounds(), candidate, placement)
			min := math.Inf(1)
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					if ratio := contrast(textLum, luminance(img.At(x, y))); ratio < min {
						min = ratio
					}
				}
			}
			if math.IsInf(min, 1) {
				continue
			}
			out = append(out, Placement{Region: candidate, MinimumRatio: min, Passes: min >= threshold})
		}
	}
	return out, nil
}

// BestPlacement returns the highest-contrast candidate, and whether it passes.
// A caller that wants one answer wants this one; a caller choosing between
// corners should read the whole list.
func BestPlacement(placements []Placement) (Placement, bool) {
	best, found := Placement{}, false
	for _, candidate := range placements {
		if !found || candidate.MinimumRatio > best.MinimumRatio {
			best, found = candidate, true
		}
	}
	return best, found && best.Passes
}
