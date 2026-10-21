// Package composition implements the portable, still-image part of Device
// Studio's fixed-plate presentation route. It owns no file, process, or tool IO.
package composition

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"image"
	"image/color"
	"math"
	"math/bits"
)

const (
	StillMaxDimension        = 16_384
	StillMaxPixels     int64 = 16_000_000
	StillMaxImageBytes int64 = 32 << 20
	StillMaxInputBytes int64 = 64 << 20
	maxDimension             = StillMaxDimension
	maxPixels                = StillMaxPixels
)

// Inputs are already-resolved immutable images. Quad coordinates are plate
// raster boundaries (not pixel centers), ordered TL, TR, BR, BL. Masks must be
// image.Gray/Gray16 coverage images or image.Alpha/Alpha16 masks; other image
// encodings are rejected. Their dimensions must match the plate. Screen is the
// authentic, already-precomposed screen raster. This package does not invent
// operating-system chrome.
type Inputs struct {
	Plate        image.Image
	Screen       image.Image
	SurfaceSize  image.Point
	Quad         [4]Point
	ActiveMask   image.Image
	HardwareMask image.Image // nil means no additional physical occlusion mask.
	Samples      int         // Samples per output-pixel axis; 1..4. Zero selects 2.
}

type Receipt struct {
	Algorithm                  string  `json:"algorithm"`
	PlateWidth                 int     `json:"plate_width"`
	PlateHeight                int     `json:"plate_height"`
	SourceWidth                int     `json:"source_width"`
	SourceHeight               int     `json:"source_height"`
	SurfaceWidth               int     `json:"surface_width"`
	SurfaceHeight              int     `json:"surface_height"`
	ContainScale               float64 `json:"contain_scale"`
	ContainOffsetX             float64 `json:"contain_offset_x"`
	ContainOffsetY             float64 `json:"contain_offset_y"`
	SamplesPerAxis             int     `json:"samples_per_axis"`
	ProtectedPixelCount        int64   `json:"protected_pixel_count"`
	ProtectedChangedPixelCount int64   `json:"protected_changed_pixel_count"`
	ProtectedPixelsSHA256      string  `json:"protected_pixels_sha256"`
	ReplayIdentityBound        bool    `json:"replay_identity_bound"`
}

type Result struct {
	Image   *image.RGBA64
	Receipt Receipt
}

// Compose maps Screen onto Quad while retaining the plate exactly wherever the
// combined active-screen coverage is zero. HardwareMask always wins over the
// screen. Screen is uniformly contained in SurfaceSize before perspective
// mapping, so this operation never silently stretches or crops app content.
func Compose(ctx context.Context, in Inputs) (Result, error) {
	if ctx == nil {
		return Result{}, errors.New("composition context is required")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if in.Plate == nil || in.Screen == nil || in.ActiveMask == nil {
		return Result{}, errors.New("plate, screen, and active mask are required")
	}
	plateBounds, sourceBounds := in.Plate.Bounds(), in.Screen.Bounds()
	if err := validateBounds("plate", plateBounds); err != nil {
		return Result{}, err
	}
	if err := validateBounds("screen", sourceBounds); err != nil {
		return Result{}, err
	}
	if err := validateBounds("active mask", in.ActiveMask.Bounds()); err != nil {
		return Result{}, err
	}
	if _, err := maskEncoding(in.ActiveMask); err != nil {
		return Result{}, fmt.Errorf("active mask: %w", err)
	}
	if in.ActiveMask.Bounds().Dx() != plateBounds.Dx() || in.ActiveMask.Bounds().Dy() != plateBounds.Dy() {
		return Result{}, errors.New("active mask dimensions must match the plate")
	}
	if in.HardwareMask != nil {
		if err := validateBounds("hardware mask", in.HardwareMask.Bounds()); err != nil {
			return Result{}, err
		}
		if _, err := maskEncoding(in.HardwareMask); err != nil {
			return Result{}, fmt.Errorf("hardware mask: %w", err)
		}
		if in.HardwareMask.Bounds().Dx() != plateBounds.Dx() || in.HardwareMask.Bounds().Dy() != plateBounds.Dy() {
			return Result{}, errors.New("hardware mask dimensions must match the plate")
		}
	}
	quad := in.Quad
	_, inverse, err := validatePlateQuad(plateBounds.Dx(), plateBounds.Dy(), in.SurfaceSize.X, in.SurfaceSize.Y, quad)
	if err != nil {
		return Result{}, err
	}
	samples := in.Samples
	if samples == 0 {
		samples = 2
	}
	if samples < 1 || samples > 4 {
		return Result{}, errors.New("samples per axis must be in [1,4]")
	}
	activeScreen := false
	for y := 0; y < plateBounds.Dy(); y++ {
		if y&63 == 0 {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
		}
		for x := 0; x < plateBounds.Dx(); x++ {
			active := maskCoverageAt(in.ActiveMask, x, y)
			hardware := 0.0
			if in.HardwareMask != nil {
				hardware = maskCoverageAt(in.HardwareMask, x, y)
			}
			if active > 0 && hardware < 1 {
				activeScreen = true
				break
			}
		}
		if activeScreen {
			break
		}
	}
	if !activeScreen {
		return Result{}, errors.New("active mask has no unoccluded screen pixels")
	}
	fitScale := math.Min(float64(in.SurfaceSize.X)/float64(sourceBounds.Dx()), float64(in.SurfaceSize.Y)/float64(sourceBounds.Dy()))
	if !finite(fitScale) || fitScale <= 0 {
		return Result{}, errors.New("invalid contain fit scale")
	}
	fitX := (float64(in.SurfaceSize.X) - float64(sourceBounds.Dx())*fitScale) / 2
	fitY := (float64(in.SurfaceSize.Y) - float64(sourceBounds.Dy())*fitScale) / 2
	out := image.NewRGBA64(plateBounds)
	protectedHash := sha256.New()
	var protectedCount, changedCount int64
	for y := 0; y < plateBounds.Dy(); y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
		}
		for x := 0; x < plateBounds.Dx(); x++ {
			base := in.Plate.At(plateBounds.Min.X+x, plateBounds.Min.Y+y)
			br, bg, bb, ba := base.RGBA()
			activeCenter := maskCoverageAt(in.ActiveMask, x, y)
			hardwareCenter := 0.0
			if in.HardwareMask != nil {
				hardwareCenter = maskCoverageAt(in.HardwareMask, x, y)
			}
			// The literal center samples establish the protected set before
			// antialiasing can diffuse neighboring coverage across its boundary.
			if activeCenter == 0 || hardwareCenter == 1 {
				out.SetRGBA64(plateBounds.Min.X+x, plateBounds.Min.Y+y, color.RGBA64{R: uint16(br), G: uint16(bg), B: uint16(bb), A: uint16(ba)})
				protectedCount++
				if !sameRGBA16(out.At(plateBounds.Min.X+x, plateBounds.Min.Y+y), base) {
					changedCount++
				}
				writeRGBA16(protectedHash, base)
				continue
			}
			var accumR, accumG, accumB, accumA uint64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					px := float64(x) + (float64(sx)+0.5)/float64(samples)
					py := float64(y) + (float64(sy)+0.5)/float64(samples)
					active := sampleCoverage(in.ActiveMask, px, py)
					hardware := 0.0
					if in.HardwareMask != nil {
						hardware = sampleCoverage(in.HardwareMask, px, py)
					}
					coverage := active * (1 - hardware)
					if coverage <= 0 {
						accumR += uint64(br)
						accumG += uint64(bg)
						accumB += uint64(bb)
						accumA += uint64(ba)
						continue
					}
					u, v, ok := project(inverse, px, py)
					if !ok || u < 0 || v < 0 || u > float64(in.SurfaceSize.X) || v > float64(in.SurfaceSize.Y) {
						accumR += uint64(br)
						accumG += uint64(bg)
						accumB += uint64(bb)
						accumA += uint64(ba)
						continue
					}
					r, g, b, a := sampleContained(in.Screen, sourceBounds, fitScale, fitX, fitY, u, v)
					alpha := float64(a) / 65535 * coverage
					rr := float64(r)*coverage + float64(br)*(1-alpha)
					gg := float64(g)*coverage + float64(bg)*(1-alpha)
					bbv := float64(b)*coverage + float64(bb)*(1-alpha)
					aa := float64(a)*coverage + float64(ba)*(1-alpha)
					accumR += uint64(clamp16(rr))
					accumG += uint64(clamp16(gg))
					accumB += uint64(clamp16(bbv))
					accumA += uint64(clamp16(aa))
				}
			}
			count := uint64(samples * samples)
			pixel := color.RGBA64{
				R: uint16((accumR + count/2) / count),
				G: uint16((accumG + count/2) / count),
				B: uint16((accumB + count/2) / count),
				A: uint16((accumA + count/2) / count),
			}
			out.SetRGBA64(plateBounds.Min.X+x, plateBounds.Min.Y+y, pixel)
		}
	}
	return Result{
		Image: out,
		Receipt: Receipt{
			Algorithm:  "inverse-homography-premultiplied-bilinear-ssaa-v1",
			PlateWidth: plateBounds.Dx(), PlateHeight: plateBounds.Dy(),
			SourceWidth: sourceBounds.Dx(), SourceHeight: sourceBounds.Dy(),
			SurfaceWidth: in.SurfaceSize.X, SurfaceHeight: in.SurfaceSize.Y,
			ContainScale: fitScale, ContainOffsetX: fitX, ContainOffsetY: fitY,
			SamplesPerAxis:      samples,
			ProtectedPixelCount: protectedCount, ProtectedChangedPixelCount: changedCount,
			ProtectedPixelsSHA256: hex.EncodeToString(protectedHash.Sum(nil)),
			ReplayIdentityBound:   false,
		},
	}, nil
}

func validateBounds(name string, b image.Rectangle) error {
	if b.Max.X <= b.Min.X || b.Max.Y <= b.Min.Y {
		return fmt.Errorf("%s dimensions must be positive", name)
	}
	width := uint64(b.Max.X) - uint64(b.Min.X)
	height := uint64(b.Max.Y) - uint64(b.Min.Y)
	if width > maxDimension || height > maxDimension {
		return fmt.Errorf("%s dimensions must be between 1 and %d", name, maxDimension)
	}
	return checkedPixels(name, int(width), int(height))
}

func checkedPixels(name string, width, height int) error {
	if width <= 0 || height <= 0 || width > maxDimension || height > maxDimension {
		return fmt.Errorf("%s dimensions must be between 1 and %d", name, maxDimension)
	}
	high, low := bits.Mul64(uint64(width), uint64(height))
	if high != 0 || low > uint64(maxPixels) {
		return fmt.Errorf("%s exceeds %d pixels", name, maxPixels)
	}
	return nil
}

func sampleContained(src image.Image, bounds image.Rectangle, scale, offsetX, offsetY, u, v float64) (uint16, uint16, uint16, uint16) {
	if u < offsetX || v < offsetY || u > offsetX+float64(bounds.Dx())*scale || v > offsetY+float64(bounds.Dy())*scale {
		return 0, 0, 0, 0
	}
	x := (u-offsetX)/scale - 0.5
	y := (v-offsetY)/scale - 0.5
	return sampleBilinear(src, bounds, x, y)
}

func sampleBilinear(src image.Image, bounds image.Rectangle, x, y float64) (uint16, uint16, uint16, uint16) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > float64(bounds.Dx()-1) {
		x = float64(bounds.Dx() - 1)
	}
	if y > float64(bounds.Dy()-1) {
		y = float64(bounds.Dy() - 1)
	}
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	x1, y1 := min(x0+1, bounds.Dx()-1), min(y0+1, bounds.Dy()-1)
	fx, fy := x-float64(x0), y-float64(y0)
	c00 := rgba16(src.At(bounds.Min.X+x0, bounds.Min.Y+y0))
	c10 := rgba16(src.At(bounds.Min.X+x1, bounds.Min.Y+y0))
	c01 := rgba16(src.At(bounds.Min.X+x0, bounds.Min.Y+y1))
	c11 := rgba16(src.At(bounds.Min.X+x1, bounds.Min.Y+y1))
	var out [4]uint16
	for i := range out {
		top := float64(c00[i])*(1-fx) + float64(c10[i])*fx
		bottom := float64(c01[i])*(1-fx) + float64(c11[i])*fx
		out[i] = clamp16(top*(1-fy) + bottom*fy)
	}
	return out[0], out[1], out[2], out[3]
}

func rgba16(c interface {
	RGBA() (uint32, uint32, uint32, uint32)
},
) [4]uint16 {
	r, g, b, a := c.RGBA()
	return [4]uint16{uint16(r), uint16(g), uint16(b), uint16(a)}
}

func maskEncoding(mask image.Image) (string, error) {
	switch mask.(type) {
	case *image.Gray:
		return "gray8", nil
	case *image.Gray16:
		return "gray16", nil
	case *image.Alpha:
		return "alpha8", nil
	case *image.Alpha16:
		return "alpha16", nil
	default:
		return "", fmt.Errorf("mask image type %T is unsupported; use grayscale coverage or alpha-only", mask)
	}
}

func maskCoverageAt(mask image.Image, x, y int) float64 {
	bounds := mask.Bounds()
	switch m := mask.(type) {
	case *image.Gray:
		return float64(m.GrayAt(bounds.Min.X+x, bounds.Min.Y+y).Y) / 255
	case *image.Gray16:
		return float64(m.Gray16At(bounds.Min.X+x, bounds.Min.Y+y).Y) / 65535
	case *image.Alpha:
		return float64(m.AlphaAt(bounds.Min.X+x, bounds.Min.Y+y).A) / 255
	case *image.Alpha16:
		return float64(m.Alpha16At(bounds.Min.X+x, bounds.Min.Y+y).A) / 65535
	default:
		return 0 // Rejected during input validation.
	}
}

func sampleCoverage(mask image.Image, x, y float64) float64 {
	bounds := mask.Bounds()
	coverageAt := func(ix, iy int) float64 {
		if ix < 0 {
			ix = 0
		}
		if iy < 0 {
			iy = 0
		}
		if ix >= bounds.Dx() {
			ix = bounds.Dx() - 1
		}
		if iy >= bounds.Dy() {
			iy = bounds.Dy() - 1
		}
		return maskCoverageAt(mask, ix, iy)
	}
	x -= 0.5
	y -= 0.5
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	top := coverageAt(x0, y0)*(1-fx) + coverageAt(x0+1, y0)*fx
	bottom := coverageAt(x0, y0+1)*(1-fx) + coverageAt(x0+1, y0+1)*fx
	return math.Max(0, math.Min(1, top*(1-fy)+bottom*fy))
}

func clamp16(v float64) uint16 {
	if v <= 0 {
		return 0
	}
	if v >= 65535 {
		return 65535
	}
	return uint16(math.Round(v))
}

func sameRGBA16(a, b interface {
	RGBA() (uint32, uint32, uint32, uint32)
},
) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func writeRGBA16(h hash.Hash, c interface {
	RGBA() (uint32, uint32, uint32, uint32)
},
) {
	r, g, b, a := c.RGBA()
	_, _ = h.Write([]byte{byte(r >> 8), byte(r), byte(g >> 8), byte(g), byte(b >> 8), byte(b), byte(a >> 8), byte(a)})
}
