package ai

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"strconv"

	"github.com/disintegration/imaging"
)

// =============================================================================
// Naturalize — the deterministic realism compositor (IMG-P1-011).
//
// Restoration/upscale models (and over-aggressive denoise) leave humans looking
// "plastic": smooth gradients, no pore-level micro-texture, flattened midtone
// contrast. Naturalize reintroduces that realism WITHOUT a generative model —
// it is pure, deterministic image processing, so it is the one AI-catalog op
// that is guaranteed runnable on every host (the headless-completeness tenet),
// independent of Phase 1's weight provisioning.
//
// It composes two effects, both scaled by the fidelity↔realism knob:
//  1. Local-contrast (clarity): an unsharp high-pass add that restores the
//     midtone micro-contrast smoothing destroys.
//  2. Film grain: a deterministic, seed-reproducible luminance noise field that
//     breaks up the too-smooth gradients that read as "plastic". Optional
//     face-aware weighting biases the grain toward midtone regions (where skin
//     sits) so faces de-plasticize more than flat backgrounds.
//
// A heavier generative detail/instruction-edit pass (Phase 3) can refine
// further on a capable GPU/BYOK; this deterministic floor always works.
// =============================================================================

// Tunable ceilings for the realism knob. At realism=1 the clarity add is
// clarityMax and the grain amplitude is grainMax 8-bit levels (±). These are
// deliberately moderate: naturalize should look like skin, not like sandpaper.
const (
	naturalizeClarityMax = 0.65
	naturalizeGrainMax   = 14.0
	naturalizeBlurSigma  = 1.4
	naturalizeFaceFloor  = 0.35 // min grain weight in shadows/highlights when face-aware
)

// NaturalizeParams are the typed inputs to the compositor, parsed from the AI
// request's string params.
type NaturalizeParams struct {
	// Realism is the fidelity↔realism knob in (0,1]. <=0 means "use the gentle
	// default"; values >1 clamp to 1.
	Realism float64
	// FaceAware biases texture/grain toward midtone (skin) regions. A luminance
	// heuristic, not face detection.
	FaceAware bool
	// Seed pins the grain RNG for reproducibility.
	Seed int64
}

// parseNaturalizeParams reads the compositor inputs from the engine's string
// param map (realism / face_aware / seed). Unset/invalid fields fall back to
// zero values (Naturalize then applies the default realism).
func parseNaturalizeParams(params map[string]string) NaturalizeParams {
	np := NaturalizeParams{}
	if v, err := strconv.ParseFloat(params["realism"], 64); err == nil {
		np.Realism = v
	}
	if params["face_aware"] == "true" {
		np.FaceAware = true
	}
	if v, err := strconv.ParseInt(params["seed"], 10, 64); err == nil {
		np.Seed = v
	}
	return np
}

// effectiveRealism clamps the knob into (0,1], defaulting an unset/zero value to
// 0.5 (a visible-but-gentle amount).
func (p NaturalizeParams) effectiveRealism() float64 {
	r := p.Realism
	if r <= 0 {
		r = 0.5
	}
	if r > 1 {
		r = 1
	}
	return r
}

// Naturalize returns a de-plasticized copy of src: local-contrast clarity plus a
// deterministic grain field, both scaled by the realism knob. Alpha is
// preserved. It is a pure function of (src, params) — same inputs always yield
// byte-identical output (the grain is hash-derived from pixel coordinates + the
// seed), which is what makes it unit-testable and reproducible.
func Naturalize(src image.Image, p NaturalizeParams) *image.NRGBA {
	r := p.effectiveRealism()
	clarityAmt := naturalizeClarityMax * r
	grainAmp := naturalizeGrainMax * r

	base := imaging.Clone(src)
	blurred := imaging.Blur(base, naturalizeBlurSigma)
	out := imaging.Clone(base)

	b := base.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := out.PixOffset(x, y)
			or, og, ob := float64(base.Pix[i]), float64(base.Pix[i+1]), float64(base.Pix[i+2])
			br, bg, bb := float64(blurred.Pix[i]), float64(blurred.Pix[i+1]), float64(blurred.Pix[i+2])

			// Local-contrast (clarity): re-add the high-frequency detail the
			// blur removed, scaled by the knob.
			nr := or + clarityAmt*(or-br)
			ng := og + clarityAmt*(og-bg)
			nb := ob + clarityAmt*(ob-bb)

			// Grain: a luminance delta added equally to all channels so it reads
			// as monochrome film grain, not chroma noise.
			weight := 1.0
			if p.FaceAware {
				luma := (0.299*or + 0.587*og + 0.114*ob) / 255.0
				// Triangular midtone emphasis: peak at luma=0.5, floor in the
				// extremes so backgrounds still get a touch of grain.
				weight = naturalizeFaceFloor + (1.0-naturalizeFaceFloor)*(1.0-math.Abs(2.0*luma-1.0))
			}
			delta := grainAmp * grainNoise(x, y, p.Seed) * weight
			nr += delta
			ng += delta
			nb += delta

			out.Pix[i] = clampByteF(nr)
			out.Pix[i+1] = clampByteF(ng)
			out.Pix[i+2] = clampByteF(nb)
			// out.Pix[i+3] (alpha) is left as the cloned original.
		}
	}
	return out
}

// grainNoise returns a deterministic pseudo-random value in [-1,1) for a pixel
// coordinate + seed. It is a finalized-hash mixer (decorrelated neighbors, no
// visible tiling), pure and reproducible — no math/rand global state.
func grainNoise(x, y int, seed int64) float64 {
	h := uint64(seed) * 0x9E3779B97F4A7C15
	h ^= uint64(uint32(x)) * 0x85EBCA6B
	h ^= uint64(uint32(y)) * 0xC2B2AE35
	h ^= h >> 13
	h *= 0xFF51AFD7ED558CCD
	h ^= h >> 33
	// Top 53 bits → [0,1) → [-1,1).
	return float64(h>>11)/float64(uint64(1)<<53)*2.0 - 1.0
}

// clampByteF rounds and clamps a float channel value into a uint8.
func clampByteF(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// decodeNaturalizeInput decodes an input image (any format imaging supports),
// honoring EXIF orientation so the result matches what the user sees.
func decodeNaturalizeInput(data []byte) (image.Image, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("ai: decode naturalize input: %w", err)
	}
	return img, nil
}

// encodePNG encodes an image as PNG (the engine's canonical output container).
func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("ai: encode naturalize output: %w", err)
	}
	return buf.Bytes(), nil
}
