// Package physical holds small, renderer-independent physical values and
// reviewed artifact bindings shared by Device Studio domains.
package physical

import (
	"errors"
	"math"
)

const CoordinateConvention = "device-studio-right-handed-meters-x-right-y-up-z-front-v1"

// Vec3 is a finite position/direction in canonical meters. Its JSON shape is
// kept byte-compatible with the existing devices.Vector3 value.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// QuatXYZW is a rotation quaternion. CanonicalizeQuaternion normalizes it and
// chooses one deterministic sign for the equivalent q and -q values.
type QuatXYZW struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

// TRS is an active column-vector local transform composed as T·R·S.
type TRS struct {
	Translation Vec3     `json:"translation_m"`
	Rotation    QuatXYZW `json:"rotation_xyzw"`
	Scale       Vec3     `json:"scale"`
}

// Matrix4 stores a column-major 4x4 matrix acting on column vectors.
type Matrix4 [16]float64

// DisplayUV is normalized display space with top-left origin, +u right, +v
// down. It is not a physical coordinate or raster pixel coordinate.
type DisplayUV struct {
	U float64 `json:"u"`
	V float64 `json:"v"`
}

func (v Vec3) Validate() error {
	if !finite(v.X) || !finite(v.Y) || !finite(v.Z) {
		return errors.New("vector components must be finite")
	}
	return nil
}

func (uv DisplayUV) Validate() error {
	if !finite(uv.U) || !finite(uv.V) || uv.U < 0 || uv.U > 1 || uv.V < 0 || uv.V > 1 {
		return errors.New("display UV must be finite and within [0,1]")
	}
	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
