package shots

import (
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/motions"
	"device-studio/internal/rendering"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

func TestPhysicalMotionOutputPinsClosedProfileAndMutualExclusion(t *testing.T) {
	valid := &shotsv1.PhysicalShotBinding{
		RendererProfileId: "device-studio-physical-cycles-v1", RendererProfileVersion: 1,
		Backend: shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX,
		MotionOutput: &shotsv1.PhysicalMotionOutputBinding{
			ProfileId: "device-studio-physical-motion-1080p-v1", ProfileVersion: 1,
			Width: 1920, Height: 1080, OutputRateNumerator: 24, OutputRateDenominator: 1, FrameCount: 48,
		},
	}
	if err := validatePhysicalProfileSelection(valid); err != nil {
		t.Fatalf("valid closed motion profile rejected: %v", err)
	}

	bad := proto.Clone(valid).(*shotsv1.PhysicalShotBinding)
	bad.OutputProfile = shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1
	if err := validatePhysicalProfileSelection(bad); err == nil {
		t.Fatal("simultaneous still and motion output profiles were accepted")
	}
	bad = proto.Clone(valid).(*shotsv1.PhysicalShotBinding)
	bad.MotionSampleTick = &shotsv1.RationalTime{Numerator: 0, Denominator: 1}
	if err := validatePhysicalProfileSelection(bad); err == nil {
		t.Fatal("motion output accepted the still-only single sample tick")
	}
	bad = proto.Clone(valid).(*shotsv1.PhysicalShotBinding)
	bad.MotionOutput.FrameCount = 49
	if err := validatePhysicalProfileSelection(bad); err == nil {
		t.Fatal("motion profile exceeded its 48-frame qualification window")
	}
	bad = proto.Clone(valid).(*shotsv1.PhysicalShotBinding)
	bad.MotionOutput.OutputRateNumerator = 25
	if err := validatePhysicalProfileSelection(bad); err == nil {
		t.Fatal("unsupported motion output rate was accepted")
	}
}

func TestPhysicalArticulationRequiresOneStaticRasterMotionTrack(t *testing.T) {
	output := &shotsv1.PhysicalMotionOutputBinding{ProfileId: rendering.PhysicalMotionOutput, ProfileVersion: 1}
	joint := motions.MotionTrack{Target: motions.TrackTarget{JointID: "lid-hinge"}}
	root := motions.MotionTrack{Target: motions.TrackTarget{DeviceRoot: true}}
	for _, tc := range []struct {
		name     string
		binding  *shotsv1.PhysicalShotBinding
		timing   *shotsv1.VideoTiming
		tracks   []motions.MotionTrack
		wantErr  bool
		contains string
	}{
		{name: "one hinge on static raster motion", binding: &shotsv1.PhysicalShotBinding{MotionOutput: output}, tracks: []motions.MotionTrack{joint}},
		{name: "unarticulated legacy motion", binding: &shotsv1.PhysicalShotBinding{}, tracks: []motions.MotionTrack{root}},
		{name: "hinge still is unsupported", binding: &shotsv1.PhysicalShotBinding{}, tracks: []motions.MotionTrack{joint}, wantErr: true, contains: "static-screen MP4"},
		{name: "recording hinge is unsupported", binding: &shotsv1.PhysicalShotBinding{MotionOutput: output}, timing: &shotsv1.VideoTiming{}, tracks: []motions.MotionTrack{joint}, wantErr: true, contains: "recording textures"},
		{name: "multiple hinges are unsupported", binding: &shotsv1.PhysicalShotBinding{MotionOutput: output}, tracks: []motions.MotionTrack{joint, {Target: motions.TrackTarget{JointID: "keyboard-hinge"}}}, wantErr: true, contains: "exactly one joint track"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePhysicalArticulationMode(tc.binding, tc.timing, motions.MotionCandidate{Tracks: tc.tracks})
			if (err != nil) != tc.wantErr || tc.contains != "" && (err == nil || !strings.Contains(err.Error(), tc.contains)) {
				t.Fatalf("articulation mode error=%v, wantErr=%t containing %q", err, tc.wantErr, tc.contains)
			}
		})
	}
}

func TestPhysicalMotionOutputV2PinsTenSecondProfileBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		rate   uint32
		frames uint32
		valid  bool
	}{{"24fps-ten-seconds", 24, 240, true}, {"30fps-ten-seconds", 30, 300, true}, {"24fps-over-ten-seconds", 24, 241, false}, {"30fps-over-ten-seconds", 30, 301, false}} {
		t.Run(tc.name, func(t *testing.T) {
			binding := &shotsv1.PhysicalShotBinding{RendererProfileId: "device-studio-physical-cycles-v1", RendererProfileVersion: 1,
				Backend: shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX,
				MotionOutput: &shotsv1.PhysicalMotionOutputBinding{ProfileId: rendering.PhysicalMotionOutputV2, ProfileVersion: 2,
					Width: 1920, Height: 1080, OutputRateNumerator: tc.rate, OutputRateDenominator: 1, FrameCount: tc.frames}}
			err := validatePhysicalProfileSelection(binding)
			if (err == nil) != tc.valid {
				t.Fatalf("v2 profile validation err=%v, valid=%t", err, tc.valid)
			}
		})
	}
}

func TestPhysicalMotionFrameCountMustExactlyCoverRationalMotionDuration(t *testing.T) {
	candidate := motions.MotionCandidate{DurationTicks: 48, Timebase: composition.Rational{Num: 24, Den: 1}}
	output := &shotsv1.PhysicalMotionOutputBinding{OutputRateNumerator: 24, OutputRateDenominator: 1, FrameCount: 48}
	if err := validatePhysicalMotionFrameCount(output, candidate); err != nil {
		t.Fatalf("exact 2-second 24fps schedule rejected: %v", err)
	}

	output = &shotsv1.PhysicalMotionOutputBinding{OutputRateNumerator: 30, OutputRateDenominator: 1, FrameCount: 48}
	if err := validatePhysicalMotionFrameCount(output, candidate); err == nil {
		t.Fatal("48 frames at 30fps incorrectly accepted for a 2-second motion")
	}

	candidate.DurationTicks = 25
	output = &shotsv1.PhysicalMotionOutputBinding{OutputRateNumerator: 24, OutputRateDenominator: 1, FrameCount: 25}
	if err := validatePhysicalMotionFrameCount(output, candidate); err != nil {
		t.Fatalf("nonintegral frame boundary should ceil to 25 frames: %v", err)
	}
	output.FrameCount = 24
	if err := validatePhysicalMotionFrameCount(output, candidate); err == nil {
		t.Fatal("truncated frame schedule was accepted")
	}

	candidate.DurationTicks = int64(^uint64(0) >> 1)
	if err := validatePhysicalMotionFrameCount(output, candidate); err == nil {
		t.Fatal("duration outside the renderer's exact uint32 tick range was accepted")
	}
}

func TestPhysicalRecordingRateMustMatchMotionOutputRate(t *testing.T) {
	output := &shotsv1.PhysicalMotionOutputBinding{OutputRateNumerator: 30, OutputRateDenominator: 1}
	for _, tc := range []struct {
		name   string
		timing *shotsv1.VideoTiming
		want   bool
	}{{"matching", &shotsv1.VideoTiming{OutputRateNum: 30, OutputRateDen: 1}, true}, {"mismatched-rate", &shotsv1.VideoTiming{OutputRateNum: 24, OutputRateDen: 1}, false}, {"mismatched-rational", &shotsv1.VideoTiming{OutputRateNum: 30, OutputRateDen: 2}, false}, {"missing-timing", nil, false}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := physicalRecordingRateMatches(tc.timing, output); got != tc.want {
				t.Fatalf("rate match=%t, want %t", got, tc.want)
			}
		})
	}
}
