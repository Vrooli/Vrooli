package motions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/physical"
)

func TestCanonicalizeCandidateCopiesCanonicalizesAndRoundTrips(t *testing.T) {
	input := baseCandidate()
	input.Timebase = composition.Rational{Num: 48, Den: 2}
	input.RequiredCapabilities = []string{"z-cap", "a-cap"}
	input.Tracks = []MotionTrack{
		jointTrack("hinge", 0, 10, 0, 1),
		rootTrack(0, 10, TransformKey{Rotation: physical.QuatXYZW{X: 0.1, Y: 0.2, Z: 0.3, W: 0.4}}, TransformKey{Translation: physical.Vec3{X: 2}, Rotation: physical.QuatXYZW{W: 1}}),
	}
	before, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	canonical, encoded, digest, err := CanonicalizeCandidate(input)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(input)
	if !bytes.Equal(before, after) {
		t.Fatal("canonicalization mutated caller-owned candidate")
	}
	if canonical.Timebase != (composition.Rational{Num: 24, Den: 1}) {
		t.Fatalf("reduced timebase = %+v", canonical.Timebase)
	}
	if canonical.RequiredCapabilities[0] != "a-cap" || canonical.Tracks[0].Target.DeviceRoot != true {
		t.Fatalf("canonical order = %+v", canonical)
	}
	decoded, decodedDigest, err := DecodeCandidate(encoded)
	if err != nil {
		t.Fatal(err)
	}
	_, secondEncoding, secondDigest, err := CanonicalizeCandidate(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if decodedDigest != digest || secondDigest != digest || !bytes.Equal(encoded, secondEncoding) {
		t.Fatal("canonical JSON/hash changed on decode and second canonicalization")
	}
}

func TestCanonicalizeRejectsMalformedUnionsAndDuplicateTargets(t *testing.T) {
	tests := []struct {
		name string
		edit func(*MotionCandidate)
	}{
		{"target-none", func(c *MotionCandidate) { c.Tracks[0].Target = TrackTarget{} }},
		{"target-many", func(c *MotionCandidate) { c.Tracks[0].Target.JointID = "joint" }},
		{"value-none", func(c *MotionCandidate) { c.Tracks[0].Keys[0].Transform = nil }},
		{"value-many", func(c *MotionCandidate) { c.Tracks[0].Keys[0].AngleRad = ptr(0.5) }},
		{"wrong-value-for-target", func(c *MotionCandidate) { c.Tracks[0].Keys[0].AngleRad = ptr(0.5); c.Tracks[0].Keys[0].Transform = nil }},
		{"joint-transform", func(c *MotionCandidate) {
			c.Tracks[0] = jointTrack("hinge", 0, 10, 0, 1)
			c.Tracks[0].Keys[0].Transform = ptr(identityTransform())
			c.Tracks[0].Keys[0].AngleRad = nil
		}},
		{"duplicate-target", func(c *MotionCandidate) { c.Tracks = append(c.Tracks, c.Tracks[0]) }},
		{"missing-start", func(c *MotionCandidate) { c.Tracks[0].Keys[0].Tick = 1 }},
		{"bad-timebase", func(c *MotionCandidate) { c.Timebase.Den = 0 }},
		{"capability-duplicate", func(c *MotionCandidate) { c.RequiredCapabilities = []string{"same", "same"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := baseCandidate()
			test.edit(&candidate)
			if _, _, _, err := CanonicalizeCandidate(candidate); err == nil {
				t.Fatal("expected candidate validation error")
			}
		})
	}
}

func TestDecodeCandidateRejectsUnknownAndTrailingJSON(t *testing.T) {
	_, encoded, _, err := CanonicalizeCandidate(baseCandidate())
	if err != nil {
		t.Fatal(err)
	}
	unknown := bytes.Replace(encoded, []byte(`"version"`), []byte(`"unknown":true,"version"`), 1)
	if _, _, err := DecodeCandidate(unknown); err == nil {
		t.Fatal("unknown field was accepted")
	}
	if _, _, err := DecodeCandidate(append(append([]byte(nil), encoded...), []byte(` {}`)...)); err == nil {
		t.Fatal("trailing JSON value was accepted")
	}
}

func TestSampleAtUsesExactRationalTicksBeyondFloatIntegerPrecision(t *testing.T) {
	const large = int64(1 << 53)
	candidate := baseCandidate()
	candidate.DurationTicks = large + 2
	candidate.Tracks[0].Keys = []MotionKey{
		transformKey(0, identityTransform()),
		transformKey(large, identityTransform()),
		transformKey(large+2, TransformKey{Translation: physical.Vec3{X: 10}, Rotation: physical.QuatXYZW{W: 1}}),
	}
	sample, err := SampleAt(candidate, composition.Rational{Num: large + 1, Den: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := sample.Tracks[0].Transform.Translation.X; got != 5 {
		t.Fatalf("sample at exact midpoint = %g, want 5", got)
	}
}

func TestSampleInterpolationRulesAndExactBoundaries(t *testing.T) {
	linear := baseCandidate()
	linear.Tracks[0].Keys = []MotionKey{
		transformKey(0, identityTransform()),
		transformKey(10, TransformKey{Translation: physical.Vec3{X: 10}, Rotation: physical.QuatXYZW{W: 1}}),
	}
	if got, err := SampleAt(linear, composition.Rational{Num: 5, Den: 1}); err != nil || got.Tracks[0].Transform.Translation.X != 5 {
		t.Fatalf("linear midpoint = %+v, %v", got, err)
	}
	if got, err := SampleAt(linear, composition.Rational{Num: 10, Den: 1}); err != nil || got.Tracks[0].Transform.Translation.X != 10 {
		t.Fatalf("exact endpoint = %+v, %v", got, err)
	}
	if _, err := SampleAt(linear, composition.Rational{Num: 11, Den: 1}); err == nil {
		t.Fatal("sample beyond duration was accepted")
	}

	step := linear
	step.Tracks = append([]MotionTrack(nil), linear.Tracks...)
	step.Tracks[0].Interpolation = Step
	if got, err := SampleAt(step, composition.Rational{Num: 9, Den: 1}); err != nil || got.Tracks[0].Transform.Translation.X != 0 {
		t.Fatalf("step value before endpoint = %+v, %v", got, err)
	}
	smooth := linear
	smooth.Tracks = append([]MotionTrack(nil), linear.Tracks...)
	smooth.Tracks[0].Interpolation = Smoothstep
	if got, err := SampleAt(smooth, composition.Rational{Num: 5, Den: 1}); err != nil || got.Tracks[0].Transform.Translation.X != 5 {
		t.Fatalf("smoothstep midpoint = %+v, %v", got, err)
	}
	if got, err := SampleAt(smooth, composition.Rational{Num: 5, Den: 2}); err != nil || math.Abs(got.Tracks[0].Transform.Translation.X-1.5625) > 1e-12 {
		t.Fatalf("smoothstep quarter = %+v, %v", got, err)
	}
}

func TestInterpolationHandlesFiniteOppositeSignExtremes(t *testing.T) {
	const extreme = 1e308
	candidate := baseCandidate()
	candidate.Tracks = []MotionTrack{
		rootTrack(0, 10,
			TransformKey{Translation: physical.Vec3{X: -extreme}, Rotation: physical.QuatXYZW{W: 1}},
			TransformKey{Translation: physical.Vec3{X: extreme}, Rotation: physical.QuatXYZW{W: 1}}),
		jointTrack("hinge", 0, 10, -extreme, extreme),
	}
	for _, tick := range []int64{0, 5, 10} {
		sample, err := SampleAt(candidate, composition.Rational{Num: tick, Den: 1})
		if err != nil {
			t.Fatalf("sample at %d: %v", tick, err)
		}
		want := -extreme
		if tick == 5 {
			want = 0
		} else if tick == 10 {
			want = extreme
		}
		if got := sample.Tracks[0].Transform.Translation.X; got != want {
			t.Errorf("translation at %d = %g, want %g", tick, got, want)
		}
		if got := *sample.Tracks[1].AngleRad; got != want {
			t.Errorf("angle at %d = %g, want %g", tick, got, want)
		}
	}
}

func TestSampleJointAngleAndSceneCameraTransformTargets(t *testing.T) {
	candidate := baseCandidate()
	candidate.Tracks = []MotionTrack{
		jointTrack("hinge", 0, 10, -1, 1),
		{Target: TrackTarget{SceneCameraID: "main-camera"}, Interpolation: Linear, Keys: []MotionKey{
			transformKey(0, identityTransform()),
			transformKey(10, TransformKey{Translation: physical.Vec3{Y: 4}, Rotation: physical.QuatXYZW{W: 1}}),
		}},
	}
	sample, err := SampleAt(candidate, composition.Rational{Num: 10, Den: 2})
	if err != nil {
		t.Fatal(err)
	}
	if sample.Tick != (composition.Rational{Num: 5, Den: 1}) {
		t.Fatalf("sample tick was not reduced exactly: %+v", sample.Tick)
	}
	if len(sample.Tracks) != 2 || sample.Tracks[0].Target.SceneCameraID != "main-camera" || sample.Tracks[1].Target.JointID != "hinge" || *sample.Tracks[1].AngleRad != 0 {
		t.Fatalf("joint sample = %+v", sample)
	}
	if sample.Tracks[0].Transform.Translation.Y != 2 {
		t.Fatalf("camera sample = %+v", sample.Tracks[0])
	}
}

func TestCanonicalizeEnforcesTrackKeyAndCapabilityBounds(t *testing.T) {
	tooManyTracks := baseCandidate()
	tooManyTracks.Tracks = make([]MotionTrack, MaxTracks+1)
	for i := range tooManyTracks.Tracks {
		tooManyTracks.Tracks[i] = jointTrack(fmt.Sprintf("joint-%02d", i), 0, 10, 0, 1)
	}
	if _, _, _, err := CanonicalizeCandidate(tooManyTracks); err == nil {
		t.Fatal("track limit was not enforced")
	}

	tooManyKeys := baseCandidate()
	tooManyKeys.DurationTicks = MaxKeys
	keys := make([]MotionKey, MaxKeys+1)
	for i := range keys {
		keys[i] = angleKey(int64(i), 0)
	}
	tooManyKeys.Tracks[0] = jointTrack("hinge", 0, MaxKeys, 0, 0)
	tooManyKeys.Tracks[0].Keys = keys
	if _, _, _, err := CanonicalizeCandidate(tooManyKeys); err == nil {
		t.Fatal("aggregate key limit was not enforced")
	}

	tooManyCapabilities := baseCandidate()
	tooManyCapabilities.RequiredCapabilities = make([]string, MaxRequiredCapability+1)
	for i := range tooManyCapabilities.RequiredCapabilities {
		tooManyCapabilities.RequiredCapabilities[i] = fmt.Sprintf("cap-%03d", i)
	}
	if _, _, _, err := CanonicalizeCandidate(tooManyCapabilities); err == nil {
		t.Fatal("capability limit was not enforced")
	}
}

func TestSLERPAntipodalNearEqualAndHalfTurnDeterminism(t *testing.T) {
	identity := physical.QuatXYZW{W: 1}
	antipodal := candidateWithRotations(identity, physical.QuatXYZW{W: -1})
	first, err := SampleAt(antipodal, composition.Rational{Num: 5, Den: 1})
	if err != nil {
		t.Fatal(err)
	}
	if first.Tracks[0].Transform.Rotation != identity {
		t.Fatalf("antipodal equivalent representation sampled as %+v", first.Tracks[0].Transform.Rotation)
	}

	halfTurn := candidateWithRotations(identity, physical.QuatXYZW{X: 1})
	midpoint, err := SampleAt(halfTurn, composition.Rational{Num: 5, Den: 1})
	if err != nil {
		t.Fatal(err)
	}
	want := math.Sqrt(0.5)
	if math.Abs(midpoint.Tracks[0].Transform.Rotation.X-want) > 1e-12 || math.Abs(midpoint.Tracks[0].Transform.Rotation.W-want) > 1e-12 {
		t.Fatalf("180-degree deterministic midpoint = %+v", midpoint.Tracks[0].Transform.Rotation)
	}

	near := candidateWithRotations(identity, physical.QuatXYZW{Z: 1e-12, W: 1})
	nearMid, err := SampleAt(near, composition.Rational{Num: 5, Den: 1})
	if err != nil {
		t.Fatal(err)
	}
	rotation := nearMid.Tracks[0].Transform.Rotation
	if !finite(rotation.Z) || rotation.Z <= 0 || rotation.Z >= 1e-12 {
		t.Fatalf("near-equal SLERP did not retain a finite intermediate rotation: %+v", rotation)
	}
}

func TestSampleAtDoesNotMutateCandidate(t *testing.T) {
	candidate := baseCandidate()
	before, _ := json.Marshal(candidate)
	if _, err := SampleAt(candidate, composition.Rational{Num: 4, Den: 1}); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(candidate)
	if !bytes.Equal(before, after) {
		t.Fatal("sampling mutated caller-owned candidate")
	}
}

func baseCandidate() MotionCandidate {
	return MotionCandidate{
		Version: CandidateVersion, MotionID: "motion-test", Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 10,
		Tracks: []MotionTrack{rootTrack(0, 10, identityTransform(), identityTransform())},
	}
}

func rootTrack(start, end int64, left, right TransformKey) MotionTrack {
	return MotionTrack{Target: TrackTarget{DeviceRoot: true}, Interpolation: Linear, Keys: []MotionKey{transformKey(start, left), transformKey(end, right)}}
}

func jointTrack(id string, start, end int64, left, right float64) MotionTrack {
	return MotionTrack{Target: TrackTarget{JointID: id}, Interpolation: Linear, Keys: []MotionKey{angleKey(start, left), angleKey(end, right)}}
}

func candidateWithRotations(left, right physical.QuatXYZW) MotionCandidate {
	candidate := baseCandidate()
	candidate.Tracks[0].Keys = []MotionKey{
		transformKey(0, TransformKey{Rotation: left}),
		transformKey(candidate.DurationTicks, TransformKey{Rotation: right}),
	}
	return candidate
}

func transformKey(tick int64, value TransformKey) MotionKey {
	return MotionKey{Tick: tick, Transform: &value}
}

func identityTransform() TransformKey { return TransformKey{Rotation: physical.QuatXYZW{W: 1}} }

func angleKey(tick int64, value float64) MotionKey { return MotionKey{Tick: tick, AngleRad: &value} }
