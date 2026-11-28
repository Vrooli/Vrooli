package ai

import (
	"bytes"
	"fmt"
	"math"
	"os"

	"github.com/disintegration/imaging"
)

// Frame policies recorded in a job's meta (frame_policy) so a result says what
// happened to its size.
const (
	framePolicySource   = "source-size"         // the provider already returned the source frame
	framePolicyResample = "resampled-to-source" // same aspect, different size: resampled back to the source frame
	framePolicyModel    = "model-size"          // kept as produced: no source, an explicit size, or a different aspect
)

// frameAspectTolerance is how far a result's aspect ratio may drift from the
// source's before it is treated as a deliberate crop or recomposition rather
// than a model working at its own resolution.
const frameAspectTolerance = 0.03

// frameKeepingOps are the operations that edit a picture in place, so the result
// belongs in the source's frame: input size in, input size out.
var frameKeepingOps = map[string]bool{
	"edit_instruct":   true,
	"garment_replace": true,
	"inpaint":         true,
	"image_to_image":  true,
}

// matchSourceFrame gives an in-place edit the source's size. A model that works
// at its own resolution (FLUX.2 klein scales anything over about 1 MP down, Kontext
// snaps to a bucket, a cloud model returns 1K) hands back a different size; the
// result is resampled to the source only when the aspect survived, so an explicit
// width/height or a deliberate crop is never stretched. It runs once, after any
// provider, so every edit route (local or cloud, every model, every future one)
// inherits it. It returns the policy that applied and whether it rewrote outPath
// (as PNG); callers refresh the output's media type when it did.
func matchSourceFrame(op, sourcePath, outPath string, params map[string]string) (policy string, rewritten bool, err error) {
	if !frameKeepingOps[op] || sourcePath == "" || params["width"] != "" || params["height"] != "" {
		return framePolicyModel, false, nil
	}
	source, err := imaging.Open(sourcePath, imaging.AutoOrientation(true))
	if err != nil {
		// A source this package cannot decode has no frame to match; the provider
		// already accepted it, so the result is kept as produced.
		return framePolicyModel, false, nil
	}
	result, err := imaging.Open(outPath)
	if err != nil {
		// Not a raster this package can read (an SVG, say): nothing to match.
		return framePolicyModel, false, nil
	}
	sw, sh, rw, rh := source.Bounds().Dx(), source.Bounds().Dy(), result.Bounds().Dx(), result.Bounds().Dy()
	if sw == rw && sh == rh {
		return framePolicySource, false, nil
	}
	if sw == 0 || sh == 0 || rw == 0 || rh == 0 || math.Abs((float64(rw)/float64(rh))/(float64(sw)/float64(sh))-1) > frameAspectTolerance {
		return framePolicyModel, false, nil
	}
	// The resampled pixels are a new PNG; the receipt the sidecar stamped into the
	// original (placement, measured GPU peak) is carried across, not lost.
	original, err := os.ReadFile(outPath)
	if err != nil {
		return "", false, fmt.Errorf("ai: read output for frame match: %w", err)
	}
	var encoded bytes.Buffer
	if err := imaging.Encode(&encoded, imaging.Resize(result, sw, sh, imaging.Lanczos), imaging.PNG); err != nil {
		return "", false, fmt.Errorf("ai: encode frame-matched output: %w", err)
	}
	if err := os.WriteFile(outPath, withPNGTextChunks(encoded.Bytes(), pngTextChunks(original)), 0o600); err != nil {
		return "", false, fmt.Errorf("ai: write frame-matched output: %w", err)
	}
	return framePolicyResample, true, nil
}
