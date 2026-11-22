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
	"image-tools/internal/opparams"
	"image-tools/internal/technique"
)

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
		return luma[min(max(y, 0), h-1)*w+min(max(x, 0), w-1)]
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
func builtinProviderSpecs() []*backends.BuiltinProvider {
	return []*backends.BuiltinProvider{
		backends.NewBuiltinProvider(models.BackendBuiltin, "ai", []string{"naturalize"}, runBuiltin),
		backends.NewBuiltinProvider(models.BackendComputed, "ai", []string{"normal_map"}, runBuiltin),
		backends.NewBuiltinProvider(models.BackendMaskFill, "ai", []string{"object_removal"}, runBuiltin),
	}
}

// runBuiltin owns image I/O once; each operation supplies only its transform.
func runBuiltin(ctx context.Context, req backends.Request) error {
	path, err := technique.Input0(req)
	if err != nil {
		return err
	}
	src, err := readBuiltinImage(path)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var out image.Image
	switch req.Operation {
	case "naturalize":
		out = Naturalize(src, parseNaturalizeParams(req.Params))
	case "normal_map":
		out = NormalMapFromDepth(src, parseNormalMapParams(req.Params))
	case "object_removal":
		path, err := technique.MaskPath(req)
		if err != nil {
			return err
		}
		mask, err := readBuiltinImage(path)
		if err != nil {
			return err
		}
		removal, err := opparams.DecodeObjectRemovalParams(req.Params)
		if err != nil {
			return err
		}
		p := DefaultMaskFillParams()
		if removal.GrowPx != nil && *removal.GrowPx >= 0 {
			p.GrowPx = *removal.GrowPx
		}
		if removal.HaloThreshold != nil && *removal.HaloThreshold >= 0 {
			p.HaloThreshold = *removal.HaloThreshold
		}
		out = MaskFill(src, mask, p)
	default:
		return fmt.Errorf("unsupported builtin operation %q", req.Operation)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := encodePNG(out)
	if err != nil {
		return err
	}
	return os.WriteFile(req.Output.LocalPath, data, 0o644)
}

func readBuiltinImage(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeNaturalizeInput(data)
}
