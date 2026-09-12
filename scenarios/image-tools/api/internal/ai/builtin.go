package ai

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strconv"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/technique"
)

// builtinExec runs an op entirely in-process (no external program, no model
// weights). It reads the request's inputs, computes the result, and writes it to
// req.Output.LocalPath by the same contract execProvider follows.
type builtinExec func(ctx context.Context, req backends.Request) error

// builtinProvider is a backends.Provider whose work is deterministic Go code
// shipped in the binary — the always-runnable Local-CPU tier. Unlike
// execProvider it has no PATH dependency: Available() is always true, so an op
// backed by a builtin provider runs on any host with zero provisioning. It is
// CPU-only and never claims the GPU.
type builtinProvider struct {
	name string
	ops  []string
	exec builtinExec
}

func (p *builtinProvider) Name() string                   { return p.name }
func (p *builtinProvider) Operations() []string           { return append([]string(nil), p.ops...) }
func (p *builtinProvider) Standalone() bool               { return true }
func (p *builtinProvider) IsCloud() bool                  { return false }
func (p *builtinProvider) GPUCapable() bool               { return false }
func (p *builtinProvider) Available(context.Context) bool { return true }
func (p *builtinProvider) Availability(context.Context) backends.Availability {
	return backends.Availability{
		Available: true,
		Detail:    fmt.Sprintf("%s provider built into image-tools API binary", p.name),
		Provision: "no host provisioning required",
	}
}

func (p *builtinProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: builtin backend %q requires a local output path", p.name)
	}
	if err := p.exec(ctx, req); err != nil {
		return backends.Result{}, fmt.Errorf("ai: builtin backend %q execution failed: %w", p.name, err)
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name}}, nil
}

// runNaturalize is the builtin executor for the naturalize op: decode the input
// image, run the deterministic realism compositor, and write the PNG result.
func runNaturalize(_ context.Context, req backends.Request) error {
	in, err := technique.Input0(req)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("ai: read naturalize input: %w", err)
	}
	img, err := decodeNaturalizeInput(data)
	if err != nil {
		return err
	}
	out := Naturalize(img, parseNaturalizeParams(req.Params))
	enc, err := encodePNG(out)
	if err != nil {
		return err
	}
	if err := os.WriteFile(req.Output.LocalPath, enc, 0o644); err != nil {
		return fmt.Errorf("ai: write naturalize output: %w", err)
	}
	return nil
}

// runNormalMap derives an RGB normal map from a depth/luma input. It is the
// computed default for normal_map: no weights, no host packages, just local CPU
// gradients over a depth image.
func runNormalMap(_ context.Context, req backends.Request) error {
	in, err := technique.Input0(req)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("ai: read normal-map input: %w", err)
	}
	img, err := decodeNaturalizeInput(data)
	if err != nil {
		return fmt.Errorf("ai: decode normal-map input: %w", err)
	}
	out := NormalMapFromDepth(img, parseNormalMapParams(req.Params))
	enc, err := encodePNG(out)
	if err != nil {
		return err
	}
	if err := os.WriteFile(req.Output.LocalPath, enc, 0o644); err != nil {
		return fmt.Errorf("ai: write normal-map output: %w", err)
	}
	return nil
}

// normalMapParams controls gradient strength. Larger values produce stronger
// X/Y components; default is intentionally conservative for depth maps.
type normalMapParams struct {
	Strength float64
}

func parseNormalMapParams(params map[string]string) normalMapParams {
	p := normalMapParams{Strength: 2}
	if v, err := strconv.ParseFloat(params["strength"], 64); err == nil && v > 0 {
		p.Strength = v
	}
	return p
}

// NormalMapFromDepth converts a depth/luma image into tangent-space normals
// encoded as RGB: X/Y/Z mapped from [-1,1] to [0,255].
func NormalMapFromDepth(src image.Image, p normalMapParams) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	luma := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			luma[y*w+x] = (0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bl>>8)) / 255.0
		}
	}
	at := func(x, y int) float64 {
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		if x >= w {
			x = w - 1
		}
		if y >= h {
			y = h - 1
		}
		return luma[y*w+x]
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := (at(x+1, y) - at(x-1, y)) * p.Strength
			dy := (at(x, y+1) - at(x, y-1)) * p.Strength
			nx, ny, nz := -dx, -dy, 1.0
			invLen := 1.0 / math.Sqrt(nx*nx+ny*ny+nz*nz)
			out.SetNRGBA(x, y, color.NRGBA{
				R: normalByte(nx * invLen),
				G: normalByte(ny * invLen),
				B: normalByte(nz * invLen),
				A: 255,
			})
		}
	}
	return out
}

func normalByte(v float64) uint8 {
	return clampByteF((v + 1.0) * 127.5)
}

// builtinProviderSpecs declares the in-process providers, keyed by the registry
// backend name so the selector's match-by-backend path lines up with the
// weightless seed models that name these backend families.
func builtinProviderSpecs() []*builtinProvider {
	return []*builtinProvider{
		{name: models.BackendBuiltin, ops: []string{"naturalize"}, exec: runNaturalize},
		{name: models.BackendComputed, ops: []string{"normal_map"}, exec: dispatchComputed},
	}
}

func dispatchComputed(ctx context.Context, req backends.Request) error {
	switch req.Operation {
	case "normal_map":
		return runNormalMap(ctx, req)
	default:
		return fmt.Errorf("unsupported computed operation %q", req.Operation)
	}
}
