// Package motions owns pure, canonical physical-motion candidate values.
// Candidate validation does not resolve or qualify external device/scene assets.
package motions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"sort"

	"device-studio/internal/composition"
	"device-studio/internal/library"
	"device-studio/internal/physical"
)

const (
	CandidateVersion      = "motion-candidate/v1"
	MaxTracks             = 64
	MaxKeys               = 4096
	MaxRequiredCapability = 128
	MaxCandidateBytes     = 1 << 20
)

type MotionCandidate struct {
	Version              string               `json:"version"`
	MotionID             string               `json:"motion_id"`
	Timebase             composition.Rational `json:"timebase_ticks_per_second"`
	DurationTicks        int64                `json:"duration_ticks"`
	Tracks               []MotionTrack        `json:"tracks"`
	RequiredCapabilities []string             `json:"required_capabilities"`
}

type MotionTrack struct {
	Target        TrackTarget   `json:"target"`
	Interpolation Interpolation `json:"interpolation"`
	Keys          []MotionKey   `json:"keys"`
}

// Exactly one target field is set. DeviceRoot addresses the selected device's
// root without embedding a device-specific branch in a motion.
type TrackTarget struct {
	DeviceRoot    bool   `json:"device_root,omitempty"`
	JointID       string `json:"joint_id,omitempty"`
	SceneCameraID string `json:"scene_camera_id,omitempty"`
}

type Interpolation string

const (
	Step       Interpolation = "step"
	Linear     Interpolation = "linear"
	Smoothstep Interpolation = "smoothstep"
)

type MotionKey struct {
	Tick      int64         `json:"tick"`
	Transform *TransformKey `json:"transform,omitempty"`
	AngleRad  *float64      `json:"angle_rad,omitempty"`
}

// TransformKey is a local translation and rotation. Scale is fixed at the
// target's rest value for this candidate version.
type TransformKey struct {
	Translation physical.Vec3     `json:"translation_m"`
	Rotation    physical.QuatXYZW `json:"rotation_xyzw"`
}

// CanonicalizeCandidate validates and canonicalizes a private copy. It never
// reorders or normalizes slices, pointers, or values owned by the caller.
func CanonicalizeCandidate(input MotionCandidate) (MotionCandidate, []byte, string, error) {
	if input.Version != CandidateVersion {
		return MotionCandidate{}, nil, "", fmt.Errorf("unsupported motion candidate version %q", input.Version)
	}
	if !library.IsValidStableID(input.MotionID) {
		return MotionCandidate{}, nil, "", errors.New("motion ID is invalid")
	}
	if input.DurationTicks <= 0 {
		return MotionCandidate{}, nil, "", errors.New("motion duration must be positive")
	}
	if input.Timebase.Num <= 0 || input.Timebase.Den <= 0 {
		return MotionCandidate{}, nil, "", errors.New("motion timebase must be a positive rational")
	}
	input.Timebase = reduceRational(input.Timebase)
	if len(input.Tracks) == 0 || len(input.Tracks) > MaxTracks {
		return MotionCandidate{}, nil, "", fmt.Errorf("motion tracks must contain between 1 and %d entries", MaxTracks)
	}
	if len(input.RequiredCapabilities) > MaxRequiredCapability {
		return MotionCandidate{}, nil, "", fmt.Errorf("required capabilities exceed %d entries", MaxRequiredCapability)
	}

	input.Tracks = append([]MotionTrack(nil), input.Tracks...)
	input.RequiredCapabilities = append([]string{}, input.RequiredCapabilities...)
	capabilities := make(map[string]struct{}, len(input.RequiredCapabilities))
	for _, capability := range input.RequiredCapabilities {
		if !library.IsValidStableID(capability) {
			return MotionCandidate{}, nil, "", fmt.Errorf("required capability %q is invalid", capability)
		}
		if _, exists := capabilities[capability]; exists {
			return MotionCandidate{}, nil, "", fmt.Errorf("duplicate required capability %q", capability)
		}
		capabilities[capability] = struct{}{}
	}
	sort.Strings(input.RequiredCapabilities)

	seenTargets := make(map[string]struct{}, len(input.Tracks))
	keyCount := 0
	for i := range input.Tracks {
		track := &input.Tracks[i]
		targetKey, targetKind, err := canonicalTarget(track.Target)
		if err != nil {
			return MotionCandidate{}, nil, "", fmt.Errorf("track %d target: %w", i, err)
		}
		if _, exists := seenTargets[targetKey]; exists {
			return MotionCandidate{}, nil, "", fmt.Errorf("duplicate motion track target %q", targetKey)
		}
		seenTargets[targetKey] = struct{}{}
		if track.Interpolation != Step && track.Interpolation != Linear && track.Interpolation != Smoothstep {
			return MotionCandidate{}, nil, "", fmt.Errorf("track %q has unsupported interpolation %q", targetKey, track.Interpolation)
		}
		if len(track.Keys) < 2 {
			return MotionCandidate{}, nil, "", fmt.Errorf("track %q needs at least its start and end key", targetKey)
		}
		keyCount += len(track.Keys)
		if keyCount > MaxKeys {
			return MotionCandidate{}, nil, "", fmt.Errorf("motion keys exceed %d entries", MaxKeys)
		}
		track.Keys = append([]MotionKey(nil), track.Keys...)
		if track.Keys[0].Tick != 0 || track.Keys[len(track.Keys)-1].Tick != input.DurationTicks {
			return MotionCandidate{}, nil, "", fmt.Errorf("track %q must have keys at tick 0 and duration %d", targetKey, input.DurationTicks)
		}
		for j := range track.Keys {
			key := &track.Keys[j]
			if key.Tick < 0 || key.Tick > input.DurationTicks || (j > 0 && key.Tick <= track.Keys[j-1].Tick) {
				return MotionCandidate{}, nil, "", fmt.Errorf("track %q keys must be unique, increasing, and within the duration", targetKey)
			}
			if (key.Transform == nil) == (key.AngleRad == nil) {
				return MotionCandidate{}, nil, "", fmt.Errorf("track %q key %d must set exactly one value", targetKey, j)
			}
			if targetKind == "joint" {
				if key.AngleRad == nil || !finite(*key.AngleRad) {
					return MotionCandidate{}, nil, "", fmt.Errorf("joint track %q key %d needs a finite angle", targetKey, j)
				}
				angle := canonicalZero(*key.AngleRad)
				key.AngleRad = &angle
			} else {
				if key.Transform == nil {
					return MotionCandidate{}, nil, "", fmt.Errorf("transform track %q key %d needs a transform", targetKey, j)
				}
				value, err := canonicalTransform(*key.Transform)
				if err != nil {
					return MotionCandidate{}, nil, "", fmt.Errorf("track %q key %d: %w", targetKey, j, err)
				}
				key.Transform = &value
			}
		}
	}
	sort.Slice(input.Tracks, func(i, j int) bool {
		left, _, _ := canonicalTarget(input.Tracks[i].Target)
		right, _, _ := canonicalTarget(input.Tracks[j].Target)
		return left < right
	})

	data, err := json.Marshal(input)
	if err != nil {
		return MotionCandidate{}, nil, "", fmt.Errorf("encode motion candidate: %w", err)
	}
	if len(data) > MaxCandidateBytes {
		return MotionCandidate{}, nil, "", errors.New("motion candidate exceeds 1 MiB canonical snapshot limit")
	}
	digest := sha256.Sum256(data)
	return input, data, hex.EncodeToString(digest[:]), nil
}

// DecodeCandidate accepts only the exact canonical JSON representation emitted
// by CanonicalizeCandidate; unknown fields and trailing values are rejected.
func DecodeCandidate(data []byte) (MotionCandidate, string, error) {
	if len(data) == 0 || len(data) > MaxCandidateBytes {
		return MotionCandidate{}, "", errors.New("motion candidate byte length is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var candidate MotionCandidate
	if err := decoder.Decode(&candidate); err != nil {
		return MotionCandidate{}, "", fmt.Errorf("decode motion candidate: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return MotionCandidate{}, "", errors.New("motion candidate has trailing JSON")
	}
	canonical, encoded, digest, err := CanonicalizeCandidate(candidate)
	if err != nil {
		return MotionCandidate{}, "", err
	}
	if !bytes.Equal(encoded, data) {
		return MotionCandidate{}, "", errors.New("motion candidate JSON is not canonical")
	}
	return canonical, digest, nil
}

func canonicalTarget(target TrackTarget) (string, string, error) {
	count := 0
	if target.DeviceRoot {
		count++
	}
	if target.JointID != "" {
		count++
	}
	if target.SceneCameraID != "" {
		count++
	}
	if count != 1 {
		return "", "", errors.New("exactly one device_root, joint_id, or scene_camera_id is required")
	}
	switch {
	case target.DeviceRoot:
		return "device_root", "transform", nil
	case target.JointID != "":
		if !library.IsValidStableID(target.JointID) {
			return "", "", errors.New("joint ID is invalid")
		}
		return "joint:" + target.JointID, "joint", nil
	default:
		if !library.IsValidStableID(target.SceneCameraID) {
			return "", "", errors.New("scene camera ID is invalid")
		}
		return "camera:" + target.SceneCameraID, "transform", nil
	}
}

func canonicalTransform(input TransformKey) (TransformKey, error) {
	trs, err := physical.CanonicalizeTRS(physical.TRS{
		Translation: input.Translation,
		Rotation:    input.Rotation,
		Scale:       physical.Vec3{X: 1, Y: 1, Z: 1},
	})
	if err != nil {
		return TransformKey{}, err
	}
	return TransformKey{Translation: trs.Translation, Rotation: trs.Rotation}, nil
}

func reduceRational(value composition.Rational) composition.Rational {
	n := big.NewInt(value.Num)
	d := big.NewInt(value.Den)
	gcd := new(big.Int).GCD(nil, nil, n, d)
	n.Quo(n, gcd)
	d.Quo(d, gcd)
	return composition.Rational{Num: n.Int64(), Den: d.Int64()}
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func canonicalZero(value float64) float64 {
	if value == 0 {
		return 0
	}
	return value
}
