package looks

import (
	"bytes"
	"encoding/json"
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

// opsParamsFromMap maps a Look step's string parameters onto the deterministic
// ops Params struct. Look parameters are a string map because a workflow edge is
// text; each value is coerced to the type its field expects and decoded through
// the struct's own JSON tags, so every operation's scalar parameters work here
// without a per-field list to drift from internal/ops.
//
// Values that name a list (vectorize's keep_colors, icon_container's sizes) are
// not expressible in a string map and are ignored in a preview; previews are a
// visual approximation, not the run.
func opsParamsFromMap(m map[string]string) *ops.Params {
	p := &ops.Params{}
	if len(m) == 0 {
		return p
	}
	values := make(map[string]any, len(m))
	for k, v := range m {
		switch {
		case v == "":
			continue
		case v == "true":
			values[k] = true
		case v == "false":
			values[k] = false
		default:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				values[k] = f
			} else {
				values[k] = v
			}
		}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return p
	}
	// Errors here mean a Look carries a parameter the operation does not have, or
	// a value of the wrong type. The run path rejects that with an actionable
	// message; a preview keeps whatever decoded and shows the rest as a no-op.
	_ = json.Unmarshal(raw, p)
	return p
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
