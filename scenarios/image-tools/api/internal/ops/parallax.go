package ops

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
)

// LayeredCamera is a bounded pinhole view of prepared RGBA planes. Relative
// distances are authored, not measured. Hidden scenery must already be filled.
type LayeredCamera struct {
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Z                 float64 `json:"z"`
	YawDegrees        float64 `json:"yaw_degrees"`
	PitchDegrees      float64 `json:"pitch_degrees"`
	FocalLengthPixels float64 `json:"focal_length_pixels"`
	RequireOpaque     bool    `json:"require_opaque"`
}

func validateLayeredCamera(c LayeredCamera, width, height int, plates []PlateSpec) error {
	if width <= 0 || height <= 0 || width > 8192 || height > 8192 || int64(width)*int64(height) > 32_000_000 {
		return fmt.Errorf("ops: parallax canvas must be positive, at most 8192 per axis and 32 million pixels")
	}
	for _, v := range []float64{c.X, c.Y, c.Z, c.YawDegrees, c.PitchDegrees, c.FocalLengthPixels} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("ops: parallax camera values must be finite")
		}
	}
	if math.Abs(c.X) > .08 || math.Abs(c.Y) > .08 || math.Abs(c.Z) > .08 || math.Abs(c.YawDegrees) > 5 || math.Abs(c.PitchDegrees) > 5 {
		return fmt.Errorf("ops: parallax nearby view permits translations up to 0.08 and yaw/pitch up to 5 degrees")
	}
	if c.FocalLengthPixels != 0 && (c.FocalLengthPixels < float64(width)*.5 || c.FocalLengthPixels > float64(width)*4) {
		return fmt.Errorf("ops: parallax focal length must be zero (canvas width) or 0.5..4 times canvas width")
	}
	last := math.Inf(1)
	for i, p := range plates {
		if p.Distance == nil || math.IsNaN(*p.Distance) || math.IsInf(*p.Distance, 0) || *p.Distance < .25 || *p.Distance > 16 {
			return fmt.Errorf("ops: parallax plate %d (%q) needs a finite distance in 0.25..16", i, p.Name)
		}
		if *p.Distance > last {
			return fmt.Errorf("ops: parallax plates must be ordered far-to-near by stack depth")
		}
		if math.IsNaN(p.Opacity) || math.IsInf(p.Opacity, 0) || p.Opacity < 0 || p.Opacity > 1 {
			return fmt.Errorf("ops: parallax plate %d opacity must be finite and in 0..1", i)
		}
		last = *p.Distance
	}
	return nil
}

// Inverse ray/plane intersection avoids forward-warp cracks. A depth boundary
// never gets stretched into a newly visible region: outside-layer samples stay
// transparent. Bilinear sampling is premultiplied to avoid colored alpha halos.
func renderLayeredView(src image.Image, output image.Point, distance float64, c LayeredCamera) (*image.NRGBA, error) {
	return renderLayeredViewContext(context.Background(), src, output, distance, c)
}

func renderLayeredViewContext(ctx context.Context, src image.Image, output image.Point, distance float64, c LayeredCamera) (*image.NRGBA, error) {
	out := image.NewNRGBA(image.Rect(0, 0, output.X, output.Y))
	f := c.FocalLengthPixels
	if f == 0 {
		f = float64(output.X)
	}
	yaw, pitch := c.YawDegrees*math.Pi/180, c.PitchDegrees*math.Pi/180
	cy, sy, cp, sp := math.Cos(yaw), math.Sin(yaw), math.Cos(pitch), math.Sin(pitch)
	ox, oy, oz := c.X*float64(output.X)/f, c.Y*float64(output.Y)/f, c.Z
	b := src.Bounds()
	for y := 0; y < output.Y; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := 0; x < output.X; x++ {
			rx, ry := (float64(x)+.5-float64(output.X)/2)/f, (float64(y)+.5-float64(output.Y)/2)/f
			// Camera-to-world rotation, yaw after pitch. Positive pitch looks down.
			px, py, pz := rx, cp*ry+sp, cp-sp*ry
			wx, wy, wz := cy*px+sy*pz, py, cy*pz-sy*px
			if wz <= 1e-8 {
				return nil, fmt.Errorf("ops: parallax camera crosses the plane horizon")
			}
			t := (distance - oz) / wz
			u := .5 + f*(ox+t*wx)/distance/float64(output.X)
			v := .5 + f*(oy+t*wy)/distance/float64(output.Y)
			// Pixel boundary coordinates. The edge is transparent beyond the plane.
			if u < 0 || u >= 1 || v < 0 || v >= 1 {
				continue
			}
			sx, sy := u*float64(b.Dx())-.5, v*float64(b.Dy())-.5
			out.SetNRGBA(x, y, layeredSample(src, sx, sy))
		}
	}
	return out, nil
}

func layeredSample(src image.Image, x, y float64) color.NRGBA {
	b := src.Bounds()
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	var rgba [4]float64
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			xx, yy := max(0, min(b.Dx()-1, x0+i)), max(0, min(b.Dy()-1, y0+j))
			w := []float64{1 - fx, fx}[i] * []float64{1 - fy, fy}[j]
			r, g, bl, a := src.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
			for k, value := range []uint32{r, g, bl, a} {
				rgba[k] += float64(value) * w
			}
		}
	}
	if rgba[3] <= 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{R: unitTo8(rgba[0] / rgba[3]), G: unitTo8(rgba[1] / rgba[3]), B: unitTo8(rgba[2] / rgba[3]), A: unitTo8(rgba[3] / 65535)}
}
