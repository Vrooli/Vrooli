package composition

import (
	"context"
	"errors"
	"image"
	"image/color"
	"math"
)

// ProjectRasterRect maps a rectangle in the screen raster to the photographed
// surface using the same uniform contain fit and homography as Compose. Both
// rect and quad use pixel-boundary coordinates; rect must lie in rasterBounds.
func ProjectRasterRect(surface, plateSize image.Point, quad [4]Point, rasterBounds, rect image.Rectangle) ([4]Point, error) {
	var projected [4]Point
	if surface.X <= 0 || surface.Y <= 0 || rasterBounds.Empty() || rect.Empty() || rect.Min.X < rasterBounds.Min.X || rect.Min.Y < rasterBounds.Min.Y || rect.Max.X > rasterBounds.Max.X || rect.Max.Y > rasterBounds.Max.Y {
		return projected, errors.New("surface, raster, and contained viewport rectangle are required")
	}
	if err := validateBounds("screen raster", rasterBounds); err != nil {
		return projected, err
	}
	forward, _, err := validatePlateQuad(plateSize.X, plateSize.Y, surface.X, surface.Y, quad)
	if err != nil {
		return projected, err
	}
	scale := math.Min(float64(surface.X)/float64(rasterBounds.Dx()), float64(surface.Y)/float64(rasterBounds.Dy()))
	if !finite(scale) || scale <= 0 {
		return projected, errors.New("screen contain transform is invalid")
	}
	offsetX := (float64(surface.X) - float64(rasterBounds.Dx())*scale) / 2
	offsetY := (float64(surface.Y) - float64(rasterBounds.Dy())*scale) / 2
	points := [4]Point{{float64(rect.Min.X - rasterBounds.Min.X), float64(rect.Min.Y - rasterBounds.Min.Y)}, {float64(rect.Max.X - rasterBounds.Min.X), float64(rect.Min.Y - rasterBounds.Min.Y)}, {float64(rect.Max.X - rasterBounds.Min.X), float64(rect.Max.Y - rasterBounds.Min.Y)}, {float64(rect.Min.X - rasterBounds.Min.X), float64(rect.Max.Y - rasterBounds.Min.Y)}}
	for i, p := range points {
		x, y := p.X*scale+offsetX, p.Y*scale+offsetY
		px, py, ok := project(forward, x, y)
		if !ok {
			return [4]Point{}, errors.New("projected viewport has an unbounded corner")
		}
		projected[i] = Point{X: px, Y: py}
	}
	if err := validateQuad(projected); err != nil {
		return [4]Point{}, err
	}
	return projected, nil
}

// InterpolateQuad linearly interpolates corresponding TL/TR/BR/BL boundary
// corners and rejects any intermediate fold or degenerate geometry.
func InterpolateQuad(from, to [4]Point, progress float64) ([4]Point, error) {
	var out [4]Point
	if !finite(progress) || progress < 0 || progress > 1 {
		return out, errors.New("quad interpolation progress must be within [0,1]")
	}
	for i := range out {
		out[i] = Point{X: from[i].X + (to[i].X-from[i].X)*progress, Y: from[i].Y + (to[i].Y-from[i].Y)*progress}
	}
	if err := validateQuad(out); err != nil {
		return [4]Point{}, err
	}
	return out, nil
}

// RenderScreenEntry warps a composed scene so from maps to to, then blends the
// direct active-viewport raster into the moving quad and fades outside scenery
// to matte. directBlend/outsideFade are bounded [0,1] factors. At frame zero,
// zero factors preserve the scene pixels exactly; at the endpoint use both
// factors as one to expose only the direct viewport and matte.
func RenderScreenEntry(ctx context.Context, scene image.Image, from, to [4]Point, directViewport image.Image, matte color.NRGBA, directBlend, outsideFade float64) (*image.RGBA64, error) {
	if ctx == nil || scene == nil || directViewport == nil || !finite(directBlend) || !finite(outsideFade) || directBlend < 0 || directBlend > 1 || outsideFade < 0 || outsideFade > 1 {
		return nil, errors.New("camera frame inputs or blend factors are invalid")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sb, db := scene.Bounds(), directViewport.Bounds()
	if err := validateBounds("camera scene", sb); err != nil {
		return nil, err
	}
	if err := validateBounds("direct viewport", db); err != nil {
		return nil, err
	}
	warpInverse, err := projectiveWarpInverse(image.Pt(sb.Dx(), sb.Dy()), from, to)
	if err != nil {
		return nil, err
	}
	if directBlend == 0 && outsideFade == 0 && from == to {
		out := image.NewRGBA64(image.Rect(0, 0, sb.Dx(), sb.Dy()))
		for y := 0; y < sb.Dy(); y++ {
			if y&63 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			for x := 0; x < sb.Dx(); x++ {
				out.Set(x, y, scene.At(sb.Min.X+x, sb.Min.Y+y))
			}
		}
		return out, nil
	}
	_, directInverse, err := homographyFromRect(db.Dx(), db.Dy(), to)
	if err != nil {
		return nil, err
	}
	out := image.NewRGBA64(image.Rect(0, 0, sb.Dx(), sb.Dy()))
	const samples = 2
	for y := 0; y < sb.Dy(); y++ {
		if y&15 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		for x := 0; x < sb.Dx(); x++ {
			var sum [4]float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					dx, dy := float64(x)+(float64(sx)+0.5)/samples, float64(y)+(float64(sy)+0.5)/samples
					scx, scy, ok := project(warpInverse, dx, dy)
					if !ok {
						return nil, errors.New("camera scene inverse warp is unbounded")
					}
					var sr, sg, sbc, sa uint16
					if scx < 0 || scy < 0 || scx > float64(sb.Dx()) || scy > float64(sb.Dy()) {
						sr, sg, sbc, sa = uint16(matte.R)*257, uint16(matte.G)*257, uint16(matte.B)*257, 65535
					} else {
						sr, sg, sbc, sa = sampleBilinear(scene, sb, scx-0.5, scy-0.5)
					}
					inside := pointInQuad(to, Point{dx, dy})
					if inside && directBlend > 0 {
						ax, ay, aok := project(directInverse, dx, dy)
						if !aok {
							return nil, errors.New("direct viewport inverse warp is unbounded")
						}
						ar, ag, abc, aa := sampleBilinear(directViewport, db, ax-0.5, ay-0.5)
						sr, sg, sbc, sa = mixRGBA16([4]uint16{sr, sg, sbc, sa}, [4]uint16{ar, ag, abc, aa}, directBlend)
					} else if !inside && outsideFade > 0 {
						matte16 := [4]uint16{uint16(matte.R) * 257, uint16(matte.G) * 257, uint16(matte.B) * 257, 65535}
						sr, sg, sbc, sa = mixRGBA16([4]uint16{sr, sg, sbc, sa}, matte16, outsideFade)
					}
					sum[0] += float64(sr)
					sum[1] += float64(sg)
					sum[2] += float64(sbc)
					sum[3] += float64(sa)
				}
			}
			count := float64(samples * samples)
			out.SetRGBA64(x, y, color.RGBA64{R: uint16(math.Round(sum[0] / count)), G: uint16(math.Round(sum[1] / count)), B: uint16(math.Round(sum[2] / count)), A: uint16(math.Round(sum[3] / count))})
		}
	}
	return out, nil
}

// RenderScreenEntryEndpoint draws only the direct active viewport into its
// contained endpoint quad. It is pixel-equivalent to RenderScreenEntry with
// directBlend and outsideFade both one, without composing a photo whose pixels
// are fully replaced at the endpoint.
func RenderScreenEntryEndpoint(ctx context.Context, output image.Point, directViewport image.Image, target [4]Point, matte color.NRGBA) (*image.RGBA64, error) {
	if ctx == nil || directViewport == nil || output.X <= 0 || output.Y <= 0 || output.X > maxDimension || output.Y > maxDimension {
		return nil, errors.New("camera endpoint raster inputs are invalid")
	}
	if err := checkedPixels("camera endpoint", output.X, output.Y); err != nil {
		return nil, err
	}
	db := directViewport.Bounds()
	if err := validateBounds("direct viewport", db); err != nil {
		return nil, err
	}
	if err := validateQuadInside(output, target); err != nil {
		return nil, err
	}
	_, directInverse, err := homographyFromRect(db.Dx(), db.Dy(), target)
	if err != nil {
		return nil, err
	}
	if err := validateFiniteRasterMap(directInverse, output.X, output.Y); err != nil {
		return nil, err
	}
	out := image.NewRGBA64(image.Rect(0, 0, output.X, output.Y))
	const samples = 2
	matte16 := [4]uint16{uint16(matte.R) * 257, uint16(matte.G) * 257, uint16(matte.B) * 257, 65535}
	for y := 0; y < output.Y; y++ {
		if y&15 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		for x := 0; x < output.X; x++ {
			var sum [4]float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					dx, dy := float64(x)+(float64(sx)+0.5)/samples, float64(y)+(float64(sy)+0.5)/samples
					pixel := matte16
					if pointInQuad(target, Point{dx, dy}) {
						ax, ay, ok := project(directInverse, dx, dy)
						if !ok {
							return nil, errors.New("direct viewport inverse warp is unbounded")
						}
						r, g, b, a := sampleBilinear(directViewport, db, ax-0.5, ay-0.5)
						pixel = [4]uint16{r, g, b, a}
					}
					for channel := range sum {
						sum[channel] += float64(pixel[channel])
					}
				}
			}
			count := float64(samples * samples)
			out.SetRGBA64(x, y, color.RGBA64{R: uint16(math.Round(sum[0] / count)), G: uint16(math.Round(sum[1] / count)), B: uint16(math.Round(sum[2] / count)), A: uint16(math.Round(sum[3] / count))})
		}
	}
	return out, nil
}

// projectiveWarpInverse returns the bounded inverse map used to move from to
// coordinates back into the source raster. It validates raster containment,
// convexity and horizons over the entire raster before native dispatch.
func projectiveWarpInverse(raster image.Point, from, to [4]Point) (matrix3, error) {
	var zero matrix3
	if raster.X <= 0 || raster.Y <= 0 {
		return zero, errors.New("projective warp raster size is invalid")
	}
	if err := validateQuadInside(raster, from); err != nil {
		return zero, err
	}
	if err := validateQuadInside(raster, to); err != nil {
		return zero, err
	}
	_, fromInverse, err := homographyFromRect(raster.X, raster.Y, from)
	if err != nil {
		return zero, err
	}
	toForward, _, err := homographyFromRect(raster.X, raster.Y, to)
	if err != nil {
		return zero, err
	}
	warpInverse, err := invert3(multiply3(toForward, fromInverse))
	if err != nil {
		return zero, err
	}
	if err := validateFiniteRasterMap(warpInverse, raster.X, raster.Y); err != nil {
		return zero, err
	}
	return warpInverse, nil
}

func ValidateProjectiveWarp(raster image.Point, from, to [4]Point) error {
	_, err := projectiveWarpInverse(raster, from, to)
	return err
}

func validateQuadInside(raster image.Point, q [4]Point) error {
	if err := validateQuad(q); err != nil {
		return err
	}
	for _, p := range q {
		if p.X < 0 || p.Y < 0 || p.X > float64(raster.X) || p.Y > float64(raster.Y) {
			return errors.New("projective viewport corner lies outside output raster")
		}
	}
	return nil
}

func multiply3(a, b matrix3) matrix3 {
	var out matrix3
	for y := range 3 {
		for x := range 3 {
			for k := range 3 {
				out[y][x] += a[y][k] * b[k][x]
			}
		}
	}
	return out
}

func pointInQuad(q [4]Point, p Point) bool {
	for i := range q {
		a, b := q[i], q[(i+1)%4]
		if (b.X-a.X)*(p.Y-a.Y)-(b.Y-a.Y)*(p.X-a.X) < -1e-7 {
			return false
		}
	}
	return true
}

func mixRGBA16(a, b [4]uint16, weight float64) (uint16, uint16, uint16, uint16) {
	var out [4]uint16
	for i := range out {
		out[i] = clamp16(float64(a[i])*(1-weight) + float64(b[i])*weight)
	}
	return out[0], out[1], out[2], out[3]
}

func validateFiniteRasterMap(m matrix3, width, height int) error {
	corners := [4]Point{{0, 0}, {float64(width), 0}, {float64(width), float64(height)}, {0, float64(height)}}
	sign := 0.0
	for _, p := range corners {
		w := m[2][0]*p.X + m[2][1]*p.Y + m[2][2]
		if !finite(w) || math.Abs(w) < 1e-10 {
			return errors.New("camera path crosses a projective horizon")
		}
		if sign == 0 {
			sign = math.Copysign(1, w)
		} else if w*sign < 0 {
			return errors.New("camera path crosses a projective horizon")
		}
	}
	return nil
}
