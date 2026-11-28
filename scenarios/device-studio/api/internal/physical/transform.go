package physical

import (
	"errors"
	"math"
)

func CanonicalizeQuaternion(q QuatXYZW) (QuatXYZW, error) {
	if !finite(q.X) || !finite(q.Y) || !finite(q.Z) || !finite(q.W) {
		return QuatXYZW{}, errors.New("quaternion components must be finite")
	}
	norm := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)
	if norm == 0 || !finite(norm) {
		return QuatXYZW{}, errors.New("quaternion must have nonzero finite length")
	}
	// A normalized quaternion can have a computed norm one or two ULPs away
	// from 1. Preserve that representation so canonicalization is idempotent
	// across JSON persistence and subsequent validation. Values outside this
	// floating-point tolerance are still fully normalized.
	const unitNormTolerance = 4 * 2.220446049250313e-16
	if math.Abs(norm-1) > unitNormTolerance {
		q.X, q.Y, q.Z, q.W = q.X/norm, q.Y/norm, q.Z/norm, q.W/norm
	}
	negate := q.W < 0 || (q.W == 0 && (q.X < 0 || (q.X == 0 && (q.Y < 0 || (q.Y == 0 && q.Z < 0)))))
	if negate {
		q.X, q.Y, q.Z, q.W = -q.X, -q.Y, -q.Z, -q.W
	}
	if q.X == 0 {
		q.X = 0
	}
	if q.Y == 0 {
		q.Y = 0
	}
	if q.Z == 0 {
		q.Z = 0
	}
	if q.W == 0 {
		q.W = 0
	}
	return q, nil
}

func CanonicalizeTRS(t TRS) (TRS, error) {
	if err := t.Translation.Validate(); err != nil {
		return TRS{}, err
	}
	if err := t.Scale.Validate(); err != nil || t.Scale.X <= 0 || t.Scale.Y <= 0 || t.Scale.Z <= 0 {
		return TRS{}, errors.New("TRS scale must be finite and positive")
	}
	q, err := CanonicalizeQuaternion(t.Rotation)
	if err != nil {
		return TRS{}, err
	}
	t.Translation.X = canonicalZero(t.Translation.X)
	t.Translation.Y = canonicalZero(t.Translation.Y)
	t.Translation.Z = canonicalZero(t.Translation.Z)
	t.Rotation = q
	return t, nil
}

func (t TRS) Matrix() (Matrix4, error) {
	t, err := CanonicalizeTRS(t)
	if err != nil {
		return Matrix4{}, err
	}
	q := t.Rotation
	x, y, z, w := q.X, q.Y, q.Z, q.W
	// R is column-major. Scale multiplies its basis columns (R·S).
	m := Matrix4{
		(1 - 2*(y*y+z*z)) * t.Scale.X, (2 * (x*y + z*w)) * t.Scale.X, (2 * (x*z - y*w)) * t.Scale.X, 0,
		(2 * (x*y - z*w)) * t.Scale.Y, (1 - 2*(x*x+z*z)) * t.Scale.Y, (2 * (y*z + x*w)) * t.Scale.Y, 0,
		(2 * (x*z + y*w)) * t.Scale.Z, (2 * (y*z - x*w)) * t.Scale.Z, (1 - 2*(x*x+y*y)) * t.Scale.Z, 0,
		t.Translation.X, t.Translation.Y, t.Translation.Z, 1,
	}
	if !validMatrix(m) {
		return Matrix4{}, errors.New("TRS values overflowed the transform matrix")
	}
	return m, nil
}

// Compose returns parent·local, preserving the active column-vector order.
func Compose(parent, local Matrix4) (Matrix4, error) {
	if !validMatrix(parent) || !validMatrix(local) {
		return Matrix4{}, errors.New("transform matrix must contain finite values")
	}
	composed := multiply(parent, local)
	if !validMatrix(composed) {
		return Matrix4{}, errors.New("composed transform overflowed")
	}
	return composed, nil
}

func TransformPoint(m Matrix4, point Vec3) (Vec3, error) {
	if !validMatrix(m) {
		return Vec3{}, errors.New("transform matrix must contain finite values")
	}
	if err := point.Validate(); err != nil {
		return Vec3{}, err
	}
	x := m[0]*point.X + m[4]*point.Y + m[8]*point.Z + m[12]
	y := m[1]*point.X + m[5]*point.Y + m[9]*point.Z + m[13]
	z := m[2]*point.X + m[6]*point.Y + m[10]*point.Z + m[14]
	w := m[3]*point.X + m[7]*point.Y + m[11]*point.Z + m[15]
	if !finite(x) || !finite(y) || !finite(z) || !finite(w) || w == 0 {
		return Vec3{}, errors.New("transformed point is not finite")
	}
	transformed := Vec3{X: x / w, Y: y / w, Z: z / w}
	if err := transformed.Validate(); err != nil {
		return Vec3{}, errors.New("transformed point is not finite")
	}
	return transformed, nil
}

func canonicalZero(value float64) float64 {
	if value == 0 {
		return 0
	}
	return value
}

// CanonicalToBlenderVector applies the fixed proper basis rotation
// (x,y,z) -> (x,-z,y); it does not convert units.
func CanonicalToBlenderVector(v Vec3) (Vec3, error) {
	if err := v.Validate(); err != nil {
		return Vec3{}, err
	}
	return Vec3{X: v.X, Y: -v.Z, Z: v.Y}, nil
}

// CanonicalToBlenderMatrix conjugates a canonical transform by the explicit
// canonical-to-Blender basis matrix. Translation and rotation change basis;
// scale remains expressed in the same positive meter units.
func CanonicalToBlenderMatrix(m Matrix4) (Matrix4, error) {
	if !validMatrix(m) {
		return Matrix4{}, errors.New("transform matrix must contain finite values")
	}
	c := Matrix4{1, 0, 0, 0, 0, 0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1}
	cInverse := Matrix4{1, 0, 0, 0, 0, 0, -1, 0, 0, 1, 0, 0, 0, 0, 0, 1}
	converted := multiply(multiply(c, m), cInverse)
	if !validMatrix(converted) {
		return Matrix4{}, errors.New("basis conversion overflowed")
	}
	return converted, nil
}

func validMatrix(m Matrix4) bool {
	for _, value := range m {
		if !finite(value) {
			return false
		}
	}
	return true
}

func multiply(a, b Matrix4) Matrix4 {
	var out Matrix4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			for k := 0; k < 4; k++ {
				out[col*4+row] += a[k*4+row] * b[col*4+k]
			}
		}
	}
	return out
}
