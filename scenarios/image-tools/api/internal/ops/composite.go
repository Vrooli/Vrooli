package ops

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"
)

// The multi-plate compositor.
//
// A plate is one depth layer of a picture — a sky, a sea, a colonnade, a figure
// — kept separate so a consumer can move them against each other and so a
// treatment can touch one without touching the others. Composite merges an
// ordered stack into the flat raster that stays every consumer's deliverable.
//
// Three blend modes, and the set is not arbitrary. Normal is placement.
// Multiply is how ink sits on paper, which is what every screen and duotone in
// this system models. Screen is how light adds, which is what a glow or a star
// field needs. A mode outside the set is refused by name rather than
// approximated with the nearest one — substituting a blend silently changes
// what a picture depicts, and nothing downstream could see it.

// Blend modes.
const (
	BlendNormal   = "normal"
	BlendMultiply = "multiply"
	BlendScreen   = "screen"
)

// PlateSpec is one layer in a stack.
type PlateSpec struct {
	Name  string `json:"name,omitempty"`
	Depth int    `json:"depth,omitempty"`
	Blend string `json:"blend,omitempty"`
	// Opacity is 0..1. Zero is fully transparent, so a caller wanting the
	// default has to send 1 — the alternative, treating zero as "unset", makes
	// a deliberately hidden plate unexpressible.
	Opacity float64 `json:"opacity"`
	// Image is the plate's pixels, set by the handler from a multipart part
	// rather than carried in JSON.
	Image []byte `json:"-"`
}

// Composite merges the plate stack in p onto a canvas.
//
// The base image is the first plate when the caller sent one through the normal
// single-image edge; it is otherwise the background. Either way the result is
// one raster at the declared geometry.
func Composite(base image.Image, p *Params) (image.Image, error) {
	plates := append([]PlateSpec(nil), p.Plates...)
	if len(plates) == 0 {
		return nil, fmt.Errorf("ops: composite requires at least one plate")
	}
	// Sorted by declared depth, stably, so two plates that claim the same layer
	// keep their list order rather than swapping between runs. Depth is
	// explicit rather than implied by position because a caller reordering a
	// stack should not have to rebuild the list, and because two plates
	// silently claiming one layer is a mistake worth keeping visible.
	sort.SliceStable(plates, func(i, j int) bool { return plates[i].Depth < plates[j].Depth })

	width, height := p.Width, p.Height
	decoded := make([]image.Image, 0, len(plates))
	for i, plate := range plates {
		if len(plate.Image) == 0 {
			return nil, fmt.Errorf("ops: composite plate %d (%q) carries no image", i, plate.Name)
		}
		img, _, err := Decode(plate.Image)
		if err != nil {
			return nil, fmt.Errorf("ops: decode composite plate %d (%q): %w", i, plate.Name, err)
		}
		if err := validateBlend(plate.Blend, i, plate.Name); err != nil {
			return nil, err
		}
		decoded = append(decoded, img)
	}
	if width <= 0 || height <= 0 {
		b := decoded[0].Bounds()
		width, height = b.Dx(), b.Dy()
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("ops: composite needs positive geometry (got %dx%d)", width, height)
	}
	_ = base

	canvas := image.NewNRGBA(image.Rect(0, 0, width, height))
	if strings.TrimSpace(p.Background) != "" {
		bg, err := parseHexColor(p.Background)
		if err != nil {
			return nil, fmt.Errorf("ops: composite background: %w", err)
		}
		fill(canvas, bg)
	}
	for i, plate := range plates {
		opacity := plate.Opacity
		if opacity < 0 {
			opacity = 0
		}
		if opacity > 1 {
			opacity = 1
		}
		blendPlate(canvas, decoded[i], blendModeOf(plate.Blend), opacity)
	}
	return canvas, nil
}

func validateBlend(mode string, index int, name string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", BlendNormal, BlendMultiply, BlendScreen:
		return nil
	default:
		return fmt.Errorf(
			"ops: composite plate %d (%q) declares blend %q; supported modes are %q, %q and %q",
			index, name, mode, BlendNormal, BlendMultiply, BlendScreen)
	}
}

func blendModeOf(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return BlendNormal
	}
	return normalized
}

func fill(dst *image.NRGBA, c color.NRGBA) {
	b := dst.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.SetNRGBA(x, y, c)
		}
	}
}

// blendPlate merges one plate onto the canvas.
//
// Plates are sampled by relative position rather than requiring identical
// pixel geometry: a plate produced at a different size than the canvas is a
// real case — a model-derived matte comes back at the model's resolution — and
// refusing it would push a resize into every caller. Nearest sampling, because
// a plate that needs resampling has already lost the argument about fidelity
// and a caller that cares resizes deliberately through the resize op.
func blendPlate(dst *image.NRGBA, src image.Image, mode string, opacity float64) {
	if opacity <= 0 {
		return
	}
	db := dst.Bounds()
	sb := src.Bounds()
	dw, dh := db.Dx(), db.Dy()
	sw, sh := sb.Dx(), sb.Dy()
	if dw <= 0 || dh <= 0 || sw <= 0 || sh <= 0 {
		return
	}
	for y := 0; y < dh; y++ {
		sy := sb.Min.Y + y*sh/dh
		for x := 0; x < dw; x++ {
			sx := sb.Min.X + x*sw/dw
			sr, sg, sbv, sa := src.At(sx, sy).RGBA()
			alpha := float64(sa) / 65535 * opacity
			if alpha <= 0 {
				continue
			}
			base := dst.NRGBAAt(db.Min.X+x, db.Min.Y+y)
			// Un-premultiply the source into 0..1 straight colour so the blend
			// functions operate on the colour a designer sees rather than on a
			// value already scaled by its own alpha.
			var srcR, srcG, srcB float64
			if sa > 0 {
				srcR = float64(sr) / float64(sa)
				srcG = float64(sg) / float64(sa)
				srcB = float64(sbv) / float64(sa)
			}
			dstR, dstG, dstB := float64(base.R)/255, float64(base.G)/255, float64(base.B)/255
			dstA := float64(base.A) / 255

			blendedR := blendChannel(mode, dstR, srcR)
			blendedG := blendChannel(mode, dstG, srcG)
			blendedB := blendChannel(mode, dstB, srcB)

			// Source-over of the blended colour. Where the canvas is
			// transparent there is nothing to blend with, so the plate's own
			// colour shows through unchanged — otherwise a multiply plate over
			// an empty canvas would come out black, which is the classic way a
			// compositor produces a picture nobody drew.
			outR := mix(srcR, blendedR, dstA)
			outG := mix(srcG, blendedG, dstA)
			outB := mix(srcB, blendedB, dstA)

			outA := alpha + dstA*(1-alpha)
			if outA <= 0 {
				dst.SetNRGBA(db.Min.X+x, db.Min.Y+y, color.NRGBA{})
				continue
			}
			finalR := (outR*alpha + dstR*dstA*(1-alpha)) / outA
			finalG := (outG*alpha + dstG*dstA*(1-alpha)) / outA
			finalB := (outB*alpha + dstB*dstA*(1-alpha)) / outA
			dst.SetNRGBA(db.Min.X+x, db.Min.Y+y, color.NRGBA{
				R: unitTo8(finalR), G: unitTo8(finalG), B: unitTo8(finalB), A: unitTo8(outA),
			})
		}
	}
}

func blendChannel(mode string, dst, src float64) float64 {
	switch mode {
	case BlendMultiply:
		return dst * src
	case BlendScreen:
		return 1 - (1-dst)*(1-src)
	default:
		return src
	}
}

// mix interpolates between the source colour and the blended colour by how
// opaque the canvas already is.
func mix(plain, blended, weight float64) float64 {
	return plain*(1-weight) + blended*weight
}

// unitTo8 maps a 0..1 channel to a byte. Named apart from adjust.go's clamp8,
// which takes an already-scaled 0..255 value: two clamps with the same name and
// different domains is exactly how an off-by-255 lands.
func unitTo8(v float64) uint8 {
	if math.IsNaN(v) || v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(math.Round(v * 255))
}
