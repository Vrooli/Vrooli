package looks

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"

	"image-tools/internal/ops"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// previewSize is the edge length of the square reference swatch a preview is
// rendered against. Small + fixed so previews are cheap and uniform.
const previewSize = 256

// RenderPreview applies a Look's DETERMINISTIC step chain to a built-in
// reference image in-process and returns the encoded PNG preview plus the list
// of operations that were deferred because they are model-backed (an AI step
// needs a backend + weights, which preview rendering does not require). For a
// pure film/camera Look every step runs and the preview is exact; for a Look
// with AI steps the preview is the deterministic approximation (often the
// unmodified reference) and deferred_steps names what a Workspace run would add.
//
// It is a pure function of the Look (the reference image is synthesized
// deterministically), so previews are reproducible and unit-testable with no
// network, storage, or model dependency.
func RenderPreview(look *looksv1.Look) (pngBytes []byte, deferred []string, err error) {
	cur, err := referencePNG()
	if err != nil {
		return nil, nil, err
	}
	for _, step := range look.GetSteps() {
		op := step.GetOperation()
		if step.GetKind() != looksv1.StepKind_STEP_KIND_DETERMINISTIC || !ops.Has(op) {
			deferred = append(deferred, op)
			continue
		}
		res, runErr := ops.Execute(op, cur, opsParamsFromMap(step.GetParams()))
		if runErr != nil {
			return nil, nil, fmt.Errorf("looks: render step %q: %w", op, runErr)
		}
		cur = res.Bytes
	}
	return cur, deferred, nil
}

// opsParamsFromMap maps a Look step's string params onto the deterministic ops
// Params struct (the subset the seeded color-grade Looks use: the adjust deltas
// and the filter selector). Unparseable/absent fields stay zero (no-op).
func opsParamsFromMap(m map[string]string) *ops.Params {
	p := &ops.Params{}
	p.Brightness = parseF(m["brightness"])
	p.Contrast = parseF(m["contrast"])
	p.Saturation = parseF(m["saturation"])
	p.Gamma = parseF(m["gamma"])
	p.Hue = parseF(m["hue"])
	p.Filter = m["filter"]
	p.Amount = parseF(m["amount"])
	p.Format = m["format"]
	return p
}

func parseF(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// referencePNG synthesizes the deterministic reference swatch: a diagonal
// hue/luma gradient with a centered neutral-grey disc. The gradient exercises
// color-grade adjustments (saturation/hue/contrast) and the disc shows
// tone/contrast shifts, so a film/camera Look's effect is visible in the
// preview without shipping a third-party reference photo.
func referencePNG() ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, previewSize, previewSize))
	cx, cy := float64(previewSize)/2, float64(previewSize)/2
	discR := float64(previewSize) * 0.28
	for y := 0; y < previewSize; y++ {
		for x := 0; x < previewSize; x++ {
			// Diagonal gradient: r rises with x, b rises with y, g is the cross.
			r := uint8(40 + (x*180)/previewSize)
			b := uint8(40 + (y*180)/previewSize)
			g := uint8(40 + ((x + y) * 120 / (2 * previewSize)))
			dx, dy := float64(x)-cx, float64(y)-cy
			if math.Sqrt(dx*dx+dy*dy) <= discR {
				r, g, b = 150, 150, 150 // neutral mid-grey disc
			}
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("looks: encode reference: %w", err)
	}
	return buf.Bytes(), nil
}
