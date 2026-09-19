package legibility

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
)

// The parallax sweep.
//
// A style that passes at rest can fail in motion. Reserved regions are declared
// against the picture as it sits, but a plated style moves: at 60% scroll a dark
// headland has slid up under white type that cleared a bright sky at rest. The
// rest-only measurement cannot see it, and neither can a reader — until they
// scroll.
//
// So the reserved rectangle becomes a reserved VOLUME: the same region swept
// across the offsets the plates travel through. The gate measures worst-pixel
// contrast at each sampled offset and takes the worst across all of them, which
// is the same rule OT-P0-011 already states for a single frame — the least
// legible pixel, never a mean or a median — applied along one more axis.
//
// The composite is sampled rather than each plate. That is a cost decision with
// a correctness argument behind it: the composite is what a reader sees, and a
// plate measured alone would report contrast against transparency.

// SweepSample is one offset's verdict.
type SweepSample struct {
	// Offset is the scroll position as a fraction of the sweep, 0 at rest.
	Offset  float64
	Verdict Verdict
}

// SweepVerdict is the whole sweep's result.
type SweepVerdict struct {
	Passes bool
	// Worst is the least legible sample. A caller that wants one number wants
	// this one.
	Worst SweepSample
	// Samples are every offset measured, in order, so an operator can see
	// whether a failure is a cliff or a slope.
	Samples []SweepSample
	// ReducedMotion is the composite measured on its own. A reduced-motion
	// viewer never sees any offset but zero, so their picture has to pass
	// independently — a style that only cleared the bar because motion moved a
	// dark mass out of the way would fail them and pass everyone else.
	ReducedMotion Verdict
}

// Layer is one plate as the sweep needs it: pixels and how far they travel.
type Layer struct {
	Name     string
	PNG      []byte
	Parallax float64
	Opacity  float64
}

// SweepSamples is how many offsets are measured, including both endpoints.
//
// Nine, chosen by measurement rather than by taste. The failure this gate exists
// to catch is a plate edge crossing a reserved region, and an edge crosses a
// region of height h over a scroll span of h/travel. With the catalog's widest
// parallax spread (0.46 for the colonnade's canopy against 0.04 for its
// distance) and its shallowest reserved region (0.28 of the frame height), the
// narrowest window in which a crossing is visible is about 0.13 of the sweep —
// wider than the 0.125 spacing nine samples give.
//
// Sampling at 17 was measured against the same catalog and found no failure
// that nine missed, which is the record the plan asks for. It is a floor rather
// than a constant to tune: a future style with a deeper spread or a shallower
// region narrows that window, and the right response is to raise this with a
// new measurement rather than to hope.
const SweepSamples = 9

// Sweep measures a plated candidate through its motion range.
//
// composite is the flat picture at rest, and layers are the plates that move
// over it. A candidate with fewer than two distinct parallax factors does not
// move, so its sweep is one sample and the result is the rest verdict — the
// same answer as before, reached by the same path.
func Sweep(composite []byte, layers []Layer, regions []Region, threshold float64, placement string) (SweepVerdict, error) {
	return sweepAt(composite, layers, regions, threshold, placement, SweepSamples)
}

// sweepAt is Sweep with an explicit sample count, so the count itself can be
// measured against a finer one rather than asserted in a comment.
func sweepAt(composite []byte, layers []Layer, regions []Region, threshold float64, placement string, samples int) (SweepVerdict, error) {
	if samples < 2 {
		samples = 2
	}
	rest, err := Measure(composite, regions, threshold, placement)
	if err != nil {
		return SweepVerdict{}, err
	}
	out := SweepVerdict{
		Passes:        rest.Passes,
		Worst:         SweepSample{Offset: 0, Verdict: rest},
		Samples:       []SweepSample{{Offset: 0, Verdict: rest}},
		ReducedMotion: rest,
	}
	if !moves(layers) {
		return out, nil
	}

	base, _, err := image.Decode(bytes.NewReader(composite))
	if err != nil {
		return SweepVerdict{}, fmt.Errorf("legibility: decode composite: %w", err)
	}
	decoded := make([]image.Image, 0, len(layers))
	for _, layer := range layers {
		img, _, decodeErr := image.Decode(bytes.NewReader(layer.PNG))
		if decodeErr != nil {
			return SweepVerdict{}, fmt.Errorf("legibility: decode plate %q: %w", layer.Name, decodeErr)
		}
		decoded = append(decoded, img)
	}

	for i := 1; i < samples; i++ {
		offset := float64(i) / float64(samples-1)
		frame, frameErr := encodePNG(compose(base, decoded, layers, offset))
		if frameErr != nil {
			return SweepVerdict{}, frameErr
		}
		verdict, measureErr := Measure(frame, regions, threshold, placement)
		if measureErr != nil {
			return SweepVerdict{}, fmt.Errorf("legibility: measure offset %.3f: %w", offset, measureErr)
		}
		sample := SweepSample{Offset: offset, Verdict: verdict}
		out.Samples = append(out.Samples, sample)
		if verdict.MinimumRatio < out.Worst.Verdict.MinimumRatio {
			out.Worst = sample
		}
		if !verdict.Passes {
			out.Passes = false
		}
	}
	// The amendment is recomputed against the WORST offset, so a scrim that a
	// caller applies actually passes everywhere rather than only at rest.
	if !out.Passes {
		opacity := minimumScrim(out.Worst.Verdict.MinimumRatio, out.Worst.Verdict.Threshold)
		out.Worst.Verdict.Amendments = []Amendment{{
			Kind: "scrim",
			Description: fmt.Sprintf(
				"Apply a black scrim at %.3f opacity — sized for scroll offset %.3f, which is the least legible point in the sweep, not for the picture at rest.",
				opacity, out.Worst.Offset),
			Value: opacity,
		}}
	}
	return out, nil
}

// Error renders the sweep refusal, naming the offset.
//
// The offset is the actionable half: "this style fails" sends an author to
// re-tune a picture that looks fine in front of them, while "this style fails at
// 0.625 of its scroll" tells them where to look.
func (v SweepVerdict) Error() string {
	if v.Passes {
		return ""
	}
	if v.Worst.Offset == 0 {
		return fmt.Sprintf("legibility: contrast %.2f is below the %.2f threshold at rest",
			v.Worst.Verdict.MinimumRatio, v.Worst.Verdict.Threshold)
	}
	return fmt.Sprintf(
		"legibility: contrast %.2f is below the %.2f threshold at scroll offset %.3f (it passes at rest at %.2f); "+
			"a plate slides under the reserved region in motion",
		v.Worst.Verdict.MinimumRatio, v.Worst.Verdict.Threshold, v.Worst.Offset, v.Samples[0].Verdict.MinimumRatio)
}

// moves reports whether the layers have any depth to sweep through.
func moves(layers []Layer) bool {
	if len(layers) < 2 {
		return false
	}
	first := layers[0].Parallax
	for _, layer := range layers[1:] {
		if layer.Parallax != first {
			return true
		}
	}
	return false
}

// compose renders the stack at one scroll offset.
//
// Each plate is translated by its parallax factor against a travel of the
// frame's own height, which is the span a full-viewport backdrop scrolls
// through. The composite underneath is NOT translated: it is the ground the
// layers move over, exactly as the emitted CSS paints it.
func compose(base image.Image, plates []image.Image, layers []Layer, offset float64) image.Image {
	b := base.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), base, b.Min, draw.Src)
	travel := float64(b.Dy())
	for i, plate := range plates {
		dy := int(math.Round(-layers[i].Parallax * offset * travel))
		pb := plate.Bounds()
		target := image.Rect(0, dy, b.Dx(), dy+b.Dy())
		draw.Draw(out, target, plate, pb.Min, draw.Over)
	}
	return out
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("legibility: encode swept frame: %w", err)
	}
	return buf.Bytes(), nil
}
