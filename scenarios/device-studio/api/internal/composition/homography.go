package composition

import (
	"errors"
	"fmt"
	"math"
)

// StillProfile is the versioned profile identity consumed by both the shot
// compiler and the isolated worker. Change it only when the executable
// compositor contract changes.
const StillProfile = "go-still-homography-v1"

// Point is expressed in top-left-origin raster boundary coordinates: the
// boundary around pixel (0, 0) starts at (0, 0), and the far edge is (W, H).
// ScreenQuad corners must be ordered top-left, top-right, bottom-right,
// bottom-left. The compositor samples pixels at their centers (+0.5,+0.5).
type Point struct {
	X float64
	Y float64
}

type matrix3 [3][3]float64

// ValidatePlateQuad applies the same raster limits and homography checks used
// by Compose. Coordinates are plate raster boundaries, not pixel centers.
func ValidatePlateQuad(plateWidth, plateHeight, surfaceWidth, surfaceHeight int, quad [4]Point) error {
	_, _, err := validatePlateQuad(plateWidth, plateHeight, surfaceWidth, surfaceHeight, quad)
	return err
}

func validatePlateQuad(plateWidth, plateHeight, surfaceWidth, surfaceHeight int, quad [4]Point) (matrix3, matrix3, error) {
	if plateWidth <= 0 || plateHeight <= 0 || plateWidth > maxDimension || plateHeight > maxDimension {
		return matrix3{}, matrix3{}, fmt.Errorf("plate dimensions must be between 1 and %d", maxDimension)
	}
	if err := checkedPixels("plate", plateWidth, plateHeight); err != nil {
		return matrix3{}, matrix3{}, err
	}
	if surfaceWidth <= 0 || surfaceHeight <= 0 || surfaceWidth > maxDimension || surfaceHeight > maxDimension {
		return matrix3{}, matrix3{}, fmt.Errorf("surface dimensions must be between 1 and %d", maxDimension)
	}
	if err := checkedPixels("surface", surfaceWidth, surfaceHeight); err != nil {
		return matrix3{}, matrix3{}, err
	}
	for i, point := range quad {
		if point.X < 0 || point.Y < 0 || point.X > float64(plateWidth) || point.Y > float64(plateHeight) {
			return matrix3{}, matrix3{}, fmt.Errorf("quad corner %d lies outside plate raster boundaries", i)
		}
	}
	forward, inverse, err := homographyFromRect(surfaceWidth, surfaceHeight, quad)
	if err != nil {
		return matrix3{}, matrix3{}, err
	}
	for _, point := range [4]Point{{0, 0}, {float64(surfaceWidth), 0}, {float64(surfaceWidth), float64(surfaceHeight)}, {0, float64(surfaceHeight)}} {
		if _, _, ok := project(forward, point.X, point.Y); !ok {
			return matrix3{}, matrix3{}, errors.New("quad produces an unbounded projective transform")
		}
	}
	return forward, inverse, nil
}

func homographyFromRect(width, height int, quad [4]Point) (matrix3, matrix3, error) {
	if width <= 0 || height <= 0 {
		return matrix3{}, matrix3{}, errors.New("surface dimensions must be positive")
	}
	if err := validateQuad(quad); err != nil {
		return matrix3{}, matrix3{}, err
	}
	src := [4]Point{{0, 0}, {float64(width), 0}, {float64(width), float64(height)}, {0, float64(height)}}
	var a [8][9]float64
	for i := 0; i < 4; i++ {
		x, y := src[i].X, src[i].Y
		u, v := quad[i].X, quad[i].Y
		a[2*i] = [9]float64{x, y, 1, 0, 0, 0, -u * x, -u * y, u}
		a[2*i+1] = [9]float64{0, 0, 0, x, y, 1, -v * x, -v * y, v}
	}
	coeff, err := solve8(a)
	if err != nil {
		return matrix3{}, matrix3{}, fmt.Errorf("solve screen homography: %w", err)
	}
	forward := matrix3{{coeff[0], coeff[1], coeff[2]}, {coeff[3], coeff[4], coeff[5]}, {coeff[6], coeff[7], 1}}
	inverse, err := invert3(forward)
	if err != nil {
		return matrix3{}, matrix3{}, fmt.Errorf("invert screen homography: %w", err)
	}
	return forward, inverse, nil
}

func validateQuad(q [4]Point) error {
	for i, p := range q {
		if !finite(p.X) || !finite(p.Y) {
			return fmt.Errorf("quad point %d is not finite", i)
		}
	}
	// Strict convexity rejects bow-ties, collapsed edges, and near-collinear
	// corners before solving. Coordinates are plate pixels, so the threshold is
	// intentionally a small fraction of one pixel squared.
	const minCross = 1e-4
	sign := 0.0
	area2 := 0.0
	for i := range q {
		a, b, c := q[i], q[(i+1)%4], q[(i+2)%4]
		cross := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
		if math.Abs(cross) < minCross {
			return fmt.Errorf("quad has a degenerate corner at index %d", i)
		}
		if sign == 0 {
			sign = math.Copysign(1, cross)
		} else if cross*sign < 0 {
			return errors.New("quad must be convex and ordered top-left, top-right, bottom-right, bottom-left")
		}
		area2 += a.X*b.Y - b.X*a.Y
	}
	if math.Abs(area2) < 1e-2 {
		return errors.New("quad area is too small")
	}
	if sign < 0 {
		return errors.New("quad corners must use top-left, top-right, bottom-right, bottom-left winding")
	}
	return nil
}

func solve8(a [8][9]float64) ([8]float64, error) {
	for col := 0; col < 8; col++ {
		pivot := col
		for row := col + 1; row < 8; row++ {
			if math.Abs(a[row][col]) > math.Abs(a[pivot][col]) {
				pivot = row
			}
		}
		if math.Abs(a[pivot][col]) < 1e-12 {
			return [8]float64{}, errors.New("singular transform")
		}
		a[col], a[pivot] = a[pivot], a[col]
		factor := a[col][col]
		for j := col; j < 9; j++ {
			a[col][j] /= factor
		}
		for row := 0; row < 8; row++ {
			if row == col {
				continue
			}
			factor = a[row][col]
			for j := col; j < 9; j++ {
				a[row][j] -= factor * a[col][j]
			}
		}
	}
	var result [8]float64
	for i := range result {
		result[i] = a[i][8]
		if !finite(result[i]) {
			return [8]float64{}, errors.New("transform coefficients are not finite")
		}
	}
	return result, nil
}

func invert3(m matrix3) (matrix3, error) {
	det := m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
	if !finite(det) || math.Abs(det) < 1e-12 {
		return matrix3{}, errors.New("singular transform")
	}
	d := 1 / det
	out := matrix3{
		{(m[1][1]*m[2][2] - m[1][2]*m[2][1]) * d, (m[0][2]*m[2][1] - m[0][1]*m[2][2]) * d, (m[0][1]*m[1][2] - m[0][2]*m[1][1]) * d},
		{(m[1][2]*m[2][0] - m[1][0]*m[2][2]) * d, (m[0][0]*m[2][2] - m[0][2]*m[2][0]) * d, (m[0][2]*m[1][0] - m[0][0]*m[1][2]) * d},
		{(m[1][0]*m[2][1] - m[1][1]*m[2][0]) * d, (m[0][1]*m[2][0] - m[0][0]*m[2][1]) * d, (m[0][0]*m[1][1] - m[0][1]*m[1][0]) * d},
	}
	for _, row := range out {
		for _, value := range row {
			if !finite(value) {
				return matrix3{}, errors.New("inverse transform is not finite")
			}
		}
	}
	return out, nil
}

func project(m matrix3, x, y float64) (float64, float64, bool) {
	w := m[2][0]*x + m[2][1]*y + m[2][2]
	if !finite(w) || math.Abs(w) < 1e-12 {
		return 0, 0, false
	}
	u := (m[0][0]*x + m[0][1]*y + m[0][2]) / w
	v := (m[1][0]*x + m[1][1]*y + m[1][2]) / w
	return u, v, finite(u) && finite(v)
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
