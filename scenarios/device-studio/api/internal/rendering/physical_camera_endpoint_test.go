package rendering

import (
	"math"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/scenes"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestCompilePhysicalCameraEndpointHitsOffCenterTarget(t *testing.T) {
	// The separate contain-layout test below proves source pixels stay at 16:9
	// while this endpoint targets the enclosing physical display rectangle.
	corners := [4]physical.Vec3{{X: -.15, Y: .084375, Z: 0}, {X: .15, Y: .084375, Z: 0}, {X: .15, Y: -.084375, Z: 0}, {X: -.15, Y: -.084375, Z: 0}}
	in := PhysicalCameraEndpoint{DisplayCorners: corners, RasterWidth: 1920, RasterHeight: 1080, Target: NormalizedRect{X: .36, Y: .2, Width: .5833333333333334, Height: .5833333333333334}, FocalLengthMM: 50, SensorWidthMM: 36, SensorHeightMM: 24}
	camera, residual, err := CompilePhysicalCameraEndpoint(in)
	if err != nil {
		t.Fatal(err)
	}
	if residual > defaultCameraEndpointErrorPixels {
		t.Fatalf("endpoint residual %.5f px", residual)
	}
	if camera.Translation.X >= 0 {
		t.Fatalf("right-side endpoint should offset camera left, got x=%f", camera.Translation.X)
	}
	if camera.Translation.Z <= 0 {
		t.Fatalf("camera must be in front of display: %+v", camera.Translation)
	}
	if !finiteNumber(residual) || math.IsNaN(camera.Rotation.W) {
		t.Fatalf("non-finite compiled endpoint: camera=%+v residual=%v", camera, residual)
	}
}

func TestCompilePhysicalCameraEndpointRejectsAmbiguousOrInvalidTargets(t *testing.T) {
	corners := [4]physical.Vec3{{X: -.15, Y: .084375}, {X: .15, Y: .084375}, {X: .15, Y: -.084375}, {X: -.15, Y: -.084375}}
	base := PhysicalCameraEndpoint{DisplayCorners: corners, RasterWidth: 1920, RasterHeight: 1080, Target: NormalizedRect{X: .36, Y: .2, Width: .5833333333333334, Height: .5833333333333334}, FocalLengthMM: 50, SensorWidthMM: 36, SensorHeightMM: 24}
	bad := base
	bad.Target.X = .8
	if _, _, err := CompilePhysicalCameraEndpoint(bad); err == nil {
		t.Fatal("out-of-raster target accepted")
	}
	bad = base
	bad.Target.Height = .7
	if _, _, err := CompilePhysicalCameraEndpoint(bad); err == nil {
		t.Fatal("aspect mismatch accepted")
	}
	bad = base
	bad.DisplayCorners[2].Z = .01
	if _, _, err := CompilePhysicalCameraEndpoint(bad); err == nil {
		t.Fatal("non-planar display accepted")
	}
}

func TestCompilePhysicalCameraEndpointBoundsMildlyTrapezoidalScreenWithinTolerance(t *testing.T) {
	corners := [4]physical.Vec3{{X: -.17234, Y: .111106}, {X: .17234, Y: .111106}, {X: .17091, Y: -.111106}, {X: -.17091, Y: -.111106}}
	averageAspect := ((.17234 - -.17234) + (.17091 - -.17091)) / 2 / (.111106 * 2)
	targetHeight := (.42 * 1920 / 1080) / averageAspect
	pose, residual, err := CompilePhysicalCameraEndpoint(PhysicalCameraEndpoint{DisplayCorners: corners, RasterWidth: 1920, RasterHeight: 1080, Target: NormalizedRect{X: .48, Y: .25, Width: .42, Height: targetHeight}, FocalLengthMM: 50, SensorWidthMM: 36, SensorHeightMM: 24})
	if err != nil {
		t.Fatal(err)
	}
	if !finiteNumber(residual) || residual <= 0 || residual > 8 || pose.Translation.X == 0 {
		t.Fatalf("mild trapezoid should produce a bounded, non-centered fit: pose=%+v residual=%.3fpx", pose, residual)
	}
}

func TestCompilePhysicalCameraEndpointFitsAuthoredUVScreenSkew(t *testing.T) {
	corners := [4]physical.Vec3{
		{X: -.1423401449, Y: .9698299680, Z: -.2031960206},
		{X: .2023394948, Y: .9695639347, Z: -.2030936987},
		{X: .2023394948, Y: .7873841038, Z: -.1368552135},
		{X: -.1423401449, Y: .7874027670, Z: -.1368623918},
	}
	target := NormalizedRect{X: 700.032 / 1920, Y: 240.03 / 1080, Width: 1119.936 / 1920, Height: 629.964 / 1080}
	pose, residual, err := CompilePhysicalCameraEndpoint(PhysicalCameraEndpoint{DisplayCorners: corners, RasterWidth: 1920, RasterHeight: 1080, Target: target, FocalLengthMM: 50, SensorWidthMM: 36, SensorHeightMM: 24})
	if err != nil {
		t.Fatal(err)
	}
	if residual > .2 || !finiteNumber(pose.Translation.X) {
		t.Fatalf("exact UV-sampled screen quad did not fit tightly: residual=%.6f pose=%+v", residual, pose)
	}
}

func TestContainedScreenContentRetainsAquila16By9ViewportInsidePhysicalDisplay(t *testing.T) {
	const rasterWidth, rasterHeight = 1920, 1080
	desired := NormalizedRect{X: 700.032 / rasterWidth, Y: 240.03 / rasterHeight, Width: 1119.936 / rasterWidth, Height: 629.964 / rasterHeight}
	displayAspect := 343.249 / 222.212
	display, err := DisplayRectForContainedContent(desired, rasterWidth, rasterHeight, displayAspect)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ContainedContentRect(display, rasterWidth, rasterHeight, rasterWidth, rasterHeight)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		name              string
		got, want, pixels float64
	}{{"x", got.X, desired.X, rasterWidth}, {"y", got.Y, desired.Y, rasterHeight}, {"width", got.Width, desired.Width, rasterWidth}, {"height", got.Height, desired.Height, rasterHeight}} {
		if delta := math.Abs(check.got-check.want) * check.pixels; delta > .01 {
			t.Errorf("contained content %s differs by %.4f pixels: got=%g want=%g", check.name, delta, check.got, check.want)
		}
	}
	if display.Height <= desired.Height || math.Abs(display.Width-desired.Width) > 1e-9 {
		t.Fatalf("16:9 source should fill panel width and leave vertical letterboxing: display=%+v content=%+v", display, got)
	}
}

func TestRecordingContentEndpointTargetsContainedAquilaPixelsAndRetainsPhysicalBounds(t *testing.T) {
	const rasterWidth, rasterHeight = 1920, 1080
	desired := NormalizedRect{X: 700.032 / rasterWidth, Y: 240.03 / rasterHeight, Width: 1119.936 / rasterWidth, Height: 629.964 / rasterHeight}
	corners := [4]physical.Vec3{
		{X: -.1709094959661499, Y: .22178300013227126, Z: -.2077869409246938},
		{X: .17090882678642494, Y: .22178300013227126, Z: -.2077869409246938},
		{X: .17233949479956046, Y: .01295123534285129, Z: -.13185775245323655},
		{X: -.1723401448598647, Y: .01295123534285129, Z: -.13185775245323655},
	}
	endpoint := &shotsv1.PhysicalCameraEndpoint{
		DisplayId: "main_display", RasterWidth: rasterWidth, RasterHeight: rasterHeight,
		X: desired.X, Y: desired.Y, Width: desired.Width, Height: desired.Height,
		TargetKind:         shotsv1.PhysicalCameraEndpointTargetKind_PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_RECORDING_CONTENT,
		ContentWidthPixels: rasterWidth, ContentHeightPixels: rasterHeight,
	}
	geometry, err := ResolvePhysicalCameraEndpointGeometry(endpoint, corners)
	if err != nil {
		t.Fatal(err)
	}
	mapped := [4]physical.Vec3{{X: -.77235, Y: .434446875}, {X: .77235, Y: .434446875}, {X: .77235, Y: -.434446875}, {X: -.77235, Y: -.434446875}}
	exactGeometry, err := ResolvePhysicalCameraEndpointGeometryWithContentCorners(endpoint, corners, &mapped)
	if err != nil {
		t.Fatal(err)
	}
	if exactGeometry.ContentCorners != mapped || exactGeometry.ContentTarget != geometry.ContentTarget {
		t.Fatalf("UV-sampled content corners/target were not preserved: %+v", exactGeometry)
	}
	badAspect := mapped
	badAspect[1].X *= .95
	if _, err := ResolvePhysicalCameraEndpointGeometryWithContentCorners(endpoint, corners, &badAspect); err == nil {
		t.Fatal("UV-mapped content with a source-aspect mismatch was accepted")
	}
	if math.Abs(geometry.ContentTarget.X-desired.X) > 1e-9 || math.Abs(geometry.ContentTarget.Y-desired.Y) > 1e-9 || math.Abs(geometry.ContentTarget.Width-desired.Width) > 1e-9 || math.Abs(geometry.ContentTarget.Height-desired.Height) > 1e-9 {
		t.Fatalf("source-content rectangle changed: got=%+v want=%+v", geometry.ContentTarget, desired)
	}
	if geometry.DisplayTarget == desired || geometry.DisplayTarget.Height <= desired.Height || geometry.ContentCorners == corners {
		t.Fatalf("letterboxed content must retain separate outer panel bounds and inner world corners: %+v", geometry)
	}
	pose, residual, err := CompilePhysicalCameraEndpoint(PhysicalCameraEndpoint{DisplayCorners: geometry.ContentCorners, RasterWidth: rasterWidth, RasterHeight: rasterHeight, Target: geometry.ContentTarget, FocalLengthMM: 50, SensorWidthMM: 36, SensorHeightMM: 24})
	if err != nil {
		t.Fatal(err)
	}
	if residual > 2 || pose.Translation.X >= 0 {
		t.Fatalf("off-center content endpoint residual %.4f px or camera position is invalid: %+v", residual, pose)
	}
	endpoint.Height *= .98
	if _, err := ResolvePhysicalCameraEndpointGeometry(endpoint, corners); err == nil {
		t.Fatal("recording-content endpoint with a stretched source aspect was accepted")
	}
}

func TestPhysicalCameraEndpointPlanValidatesAndBindsEveryScheduledFrame(t *testing.T) {
	focal, sensorWidth, sensorHeight := 50.0, 36.0, 24.0
	camera := scenes.SceneCameraFrame{ID: "main", Frame: physical.TRS{Translation: physical.Vec3{Z: 3}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, Projection: scenes.ProjectionPerspective, FocalLengthMM: &focal, SensorWidthMM: &sensorWidth, SensorHeightMM: &sensorHeight, SensorFit: scenes.SensorFitHorizontal}
	endpoint := &PhysicalCameraEndpointPlan{CameraID: "main", DisplayID: "screen", RasterWidth: 1920, RasterHeight: 1080, Target: NormalizedRect{X: .36, Y: .2, Width: .56, Height: .56}, DisplayCorners: [4]physical.Vec3{{X: -.3, Y: .16875}, {X: .3, Y: .16875}, {X: .3, Y: -.16875}, {X: -.3, Y: -.16875}}, CameraFrame: camera.Frame, FocalLengthMM: focal, SensorWidthMM: sensorWidth, SensorHeightMM: sensorHeight, MaxErrorPixels: 1}
	pose, residual, err := CompilePhysicalCameraEndpoint(PhysicalCameraEndpoint{DisplayCorners: endpoint.DisplayCorners, RasterWidth: endpoint.RasterWidth, RasterHeight: endpoint.RasterHeight, Target: endpoint.Target, FocalLengthMM: endpoint.FocalLengthMM, SensorWidthMM: endpoint.SensorWidthMM, SensorHeightMM: endpoint.SensorHeightMM})
	if err != nil {
		t.Fatal(err)
	}
	endpoint.ResidualPixels = residual
	delta, err := CameraEndpointRelativeTransform(camera.Frame, pose)
	if err != nil {
		t.Fatal(err)
	}
	timebase := composition.Rational{Num: 24, Den: 1}
	root := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	output := PhysicalOutputPlan{Kind: "motion_mp4", Width: 1920, Height: 1080, FrameCount: 2, MotionTicks: 24, Timebase: timebase, CameraEndpoint: endpoint, Samples: []PhysicalMotionSample{{OutputOrdinal: 0, RequestedMotionTick: composition.Rational{Num: 0, Den: 1}, SampledMotionTick: composition.Rational{Num: 0, Den: 1}, DeviceRoot: &root, CameraTransforms: map[string]physical.TRS{"main": InterpolateCameraTransform(delta, 0)}}, {OutputOrdinal: 1, RequestedMotionTick: composition.Rational{Num: 24, Den: 1}, SampledMotionTick: composition.Rational{Num: 24, Den: 1}, DeviceRoot: &root, CameraTransforms: map[string]physical.TRS{"main": InterpolateCameraTransform(delta, 1)}}}}
	identity := motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}
	motion := motions.MotionCandidate{Version: motions.CandidateVersion, MotionID: "endpoint-test", Timebase: timebase, DurationTicks: 24, Tracks: []motions.MotionTrack{{Target: motions.TrackTarget{DeviceRoot: true}, Interpolation: motions.Linear, Keys: []motions.MotionKey{{Tick: 0, Transform: &identity}, {Tick: 24, Transform: &identity}}}}}
	placement := PhysicalPlacement{SelectedCameraID: "main"}
	if err := validatePhysicalCameraEndpointPlan(endpoint, []scenes.SceneCameraFrame{camera}, placement, "screen", motion, output); err != nil {
		t.Fatal(err)
	}
	if err := validateMotionOutput(output, motion); err != nil {
		t.Fatal(err)
	}
	endpoint.ResidualPixels += .01
	if err := validatePhysicalCameraEndpointPlan(endpoint, []scenes.SceneCameraFrame{camera}, placement, "screen", motion, output); err == nil {
		t.Fatal("tampered recorded endpoint residual was accepted")
	}
	endpoint.ResidualPixels = residual
	output.Samples[1].CameraTransforms["main"] = physical.TRS{Translation: physical.Vec3{X: .001}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	if err := validatePhysicalCameraEndpointPlan(endpoint, []scenes.SceneCameraFrame{camera}, placement, "screen", motion, output); err == nil {
		t.Fatal("tampered camera endpoint sample was accepted")
	}
}

func TestPhysicalCameraEndpointSampleProgressArrivesThenHolds(t *testing.T) {
	samples := []PhysicalMotionSample{
		{RequestedMotionTick: composition.Rational{Num: 0, Den: 1}},
		{RequestedMotionTick: composition.Rational{Num: 44, Den: 1}},
		{RequestedMotionTick: composition.Rational{Num: 88, Den: 1}},
		{RequestedMotionTick: composition.Rational{Num: 95, Den: 1}},
	}
	progress, err := PhysicalCameraEndpointSampleProgress(88, samples)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0, .5, 1, 1}
	for i := range want {
		if progress[i] != want[i] {
			t.Fatalf("sample %d progress=%v, want %v", i, progress[i], want[i])
		}
	}
	legacy, err := PhysicalCameraEndpointSampleProgress(0, samples)
	if err != nil {
		t.Fatal(err)
	}
	if legacy[2] >= 1 || legacy[3] != 1 {
		t.Fatalf("zero arrival tick should preserve end-of-output arrival: %#v", legacy)
	}
	if _, err := PhysicalCameraEndpointSampleProgress(96, samples); err == nil {
		t.Fatal("arrival tick beyond the last scheduled sample was accepted")
	}
}
