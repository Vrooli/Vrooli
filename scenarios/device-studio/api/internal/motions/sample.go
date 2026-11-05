package motions

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"

	"device-studio/internal/composition"
	"device-studio/internal/physical"
)

type TrackSample struct {
	Target    TrackTarget   `json:"target"`
	Transform *TransformKey `json:"transform,omitempty"`
	AngleRad  *float64      `json:"angle_rad,omitempty"`
}

type MotionSample struct {
	Tick   composition.Rational `json:"tick"`
	Tracks []TrackSample        `json:"tracks"`
}

// SampleAt evaluates all tracks at an exact rational tick position. It only
// interpolates the candidate's keyed values; it does not apply them to a GLB,
// compose a root pivot, or apply scene placement/rest transforms.
func SampleAt(input MotionCandidate, at composition.Rational) (MotionSample, error) {
	candidate, _, _, err := CanonicalizeCandidate(input)
	if err != nil {
		return MotionSample{}, err
	}
	if at.Den <= 0 || at.Num < 0 {
		return MotionSample{}, errors.New("sample tick must be a nonnegative rational")
	}
	tick := new(big.Rat).SetFrac(big.NewInt(at.Num), big.NewInt(at.Den))
	duration := new(big.Rat).SetInt64(candidate.DurationTicks)
	if tick.Cmp(duration) > 0 {
		return MotionSample{}, errors.New("sample tick exceeds motion duration")
	}
	canonicalTick, err := rationalValue(tick)
	if err != nil {
		return MotionSample{}, fmt.Errorf("canonical sample tick: %w", err)
	}

	result := MotionSample{Tick: canonicalTick, Tracks: make([]TrackSample, 0, len(candidate.Tracks))}
	for _, track := range candidate.Tracks {
		value, err := sampleTrack(track, tick)
		if err != nil {
			return MotionSample{}, fmt.Errorf("sample target %s: %w", targetName(track.Target), err)
		}
		result.Tracks = append(result.Tracks, value)
	}
	return result, nil
}

func sampleTrack(track MotionTrack, tick *big.Rat) (TrackSample, error) {
	upper := sort.Search(len(track.Keys), func(i int) bool {
		return new(big.Rat).SetInt64(track.Keys[i].Tick).Cmp(tick) >= 0
	})
	if upper == len(track.Keys) {
		return TrackSample{}, errors.New("no key at or after sample tick")
	}
	if new(big.Rat).SetInt64(track.Keys[upper].Tick).Cmp(tick) == 0 {
		return sampleKey(track.Target, track.Keys[upper])
	}
	if upper == 0 {
		return TrackSample{}, errors.New("sample precedes first key")
	}
	left, right := track.Keys[upper-1], track.Keys[upper]
	if track.Interpolation == Step {
		return sampleKey(track.Target, left)
	}
	start := new(big.Rat).SetInt64(left.Tick)
	span := new(big.Rat).SetInt64(right.Tick - left.Tick)
	progress := new(big.Rat).Quo(new(big.Rat).Sub(tick, start), span)
	u, _ := progress.Float64()
	if math.IsNaN(u) || math.IsInf(u, 0) || u < 0 || u > 1 {
		return TrackSample{}, errors.New("interpolation progress is outside [0,1]")
	}
	if track.Interpolation == Smoothstep {
		u = u * u * (3 - 2*u)
	}
	result := TrackSample{Target: track.Target}
	if left.AngleRad != nil {
		angle := interpolateFinite(*left.AngleRad, *right.AngleRad, u)
		if !finite(angle) {
			return TrackSample{}, errors.New("interpolated angle is not finite")
		}
		result.AngleRad = ptr(canonicalZero(angle))
		return result, nil
	}
	translation := physical.Vec3{
		X: interpolateFinite(left.Transform.Translation.X, right.Transform.Translation.X, u),
		Y: interpolateFinite(left.Transform.Translation.Y, right.Transform.Translation.Y, u),
		Z: interpolateFinite(left.Transform.Translation.Z, right.Transform.Translation.Z, u),
	}
	if err := translation.Validate(); err != nil {
		return TrackSample{}, errors.New("interpolated translation is not finite")
	}
	rotation, err := slerp(left.Transform.Rotation, right.Transform.Rotation, u)
	if err != nil {
		return TrackSample{}, err
	}
	result.Transform = &TransformKey{Translation: canonicalVec(translation), Rotation: rotation}
	return result, nil
}

func sampleKey(target TrackTarget, key MotionKey) (TrackSample, error) {
	result := TrackSample{Target: target}
	if key.AngleRad != nil {
		result.AngleRad = ptr(*key.AngleRad)
		return result, nil
	}
	if key.Transform == nil {
		return TrackSample{}, errors.New("canonical key has no value")
	}
	transform := *key.Transform
	result.Transform = &transform
	return result, nil
}

func slerp(start, end physical.QuatXYZW, u float64) (physical.QuatXYZW, error) {
	a, err := physical.CanonicalizeQuaternion(start)
	if err != nil {
		return physical.QuatXYZW{}, err
	}
	b, err := physical.CanonicalizeQuaternion(end)
	if err != nil {
		return physical.QuatXYZW{}, err
	}
	dot := quaternionDot(a, b)
	if dot < 0 {
		b = negateQuaternion(b)
		dot = -dot
	}
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	// Derive the sine from the orthogonal residual instead of sqrt(1-dot²):
	// the latter loses the small angle when dot rounds to exactly one.
	residual := physical.QuatXYZW{
		X: b.X - dot*a.X,
		Y: b.Y - dot*a.Y,
		Z: b.Z - dot*a.Z,
		W: b.W - dot*a.W,
	}
	sinTheta := math.Sqrt(quaternionDot(residual, residual))
	if !finite(sinTheta) {
		return physical.QuatXYZW{}, errors.New("quaternion interpolation residual is invalid")
	}
	if sinTheta == 0 {
		return a, nil
	}
	theta := math.Atan2(sinTheta, dot)
	var weightA, weightB float64
	if theta < 1e-4 {
		theta2 := theta * theta
		oneMinusU := 1 - u
		weightA = oneMinusU * (1 + (1-oneMinusU*oneMinusU)*theta2/6)
		weightB = u * (1 + (1-u*u)*theta2/6)
	} else {
		denominator := math.Sin(theta)
		weightA = math.Sin((1-u)*theta) / denominator
		weightB = math.Sin(u*theta) / denominator
	}
	value := physical.QuatXYZW{
		X: weightA*a.X + weightB*b.X,
		Y: weightA*a.Y + weightB*b.Y,
		Z: weightA*a.Z + weightB*b.Z,
		W: weightA*a.W + weightB*b.W,
	}
	return physical.CanonicalizeQuaternion(value)
}

func rationalValue(value *big.Rat) (composition.Rational, error) {
	if !value.Num().IsInt64() || !value.Denom().IsInt64() || value.Denom().Sign() <= 0 {
		return composition.Rational{}, errors.New("sample tick exceeds rational representation")
	}
	return composition.Rational{Num: value.Num().Int64(), Den: value.Denom().Int64()}, nil
}

func targetName(target TrackTarget) string {
	key, _, _ := canonicalTarget(target)
	return key
}

func quaternionDot(a, b physical.QuatXYZW) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
}

func negateQuaternion(value physical.QuatXYZW) physical.QuatXYZW {
	return physical.QuatXYZW{X: -value.X, Y: -value.Y, Z: -value.Z, W: -value.W}
}

func interpolateFinite(a, b, u float64) float64 {
	if u == 0 {
		return a
	}
	if u == 1 {
		return b
	}
	if math.Signbit(a) != math.Signbit(b) {
		return a*(1-u) + b*u
	}
	return a + (b-a)*u
}

func canonicalVec(value physical.Vec3) physical.Vec3 {
	if value.X == 0 {
		value.X = 0
	}
	if value.Y == 0 {
		value.Y = 0
	}
	if value.Z == 0 {
		value.Z = 0
	}
	return value
}

func ptr[T any](value T) *T { return &value }
