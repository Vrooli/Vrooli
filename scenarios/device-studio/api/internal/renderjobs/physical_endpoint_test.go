package renderjobs

import (
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestApplyPhysicalCameraEndpointSchedulesOffCenterDisplayHandoff(t *testing.T) {
	identity := identityTRS()
	baseCamera := identity
	baseCamera.Translation.Z = 3
	focal, sensorWidth, sensorHeight := 50.0, 36.0, 24.0
	endpoint := &shotsv1.PhysicalCameraEndpoint{DisplayId: "screen", RasterWidth: 1920, RasterHeight: 1080, X: .36, Y: .2, Width: .56, Height: .56, MaxErrorPixels: 1, ArrivalMotionTick: 2}
	corners := [4]physical.Vec3{
		{X: -.3, Y: .16875}, {X: .3, Y: .16875}, {X: .3, Y: -.16875}, {X: -.3, Y: -.16875},
	}
	samples := []rendering.PhysicalMotionSample{
		{OutputOrdinal: 0, OutputTime: composition.Rational{Num: 0, Den: 1}, RequestedMotionTick: composition.Rational{Num: 0, Den: 1}, CameraTransforms: nil},
		{OutputOrdinal: 1, OutputTime: composition.Rational{Num: 1, Den: 1}, RequestedMotionTick: composition.Rational{Num: 1, Den: 1}, DeviceRoot: &physical.TRS{Translation: physical.Vec3{X: .1}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}},
		{OutputOrdinal: 2, OutputTime: composition.Rational{Num: 2, Den: 1}, RequestedMotionTick: composition.Rational{Num: 2, Den: 1}, DeviceRoot: &physical.TRS{Translation: physical.Vec3{X: .1}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}},
		{OutputOrdinal: 3, OutputTime: composition.Rational{Num: 3, Den: 1}, RequestedMotionTick: composition.Rational{Num: 3, Den: 1}, DeviceRoot: &physical.TRS{Translation: physical.Vec3{X: .1}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}},
	}
	cameras := []scenes.SceneCameraFrame{{ID: "main", Frame: baseCamera, Projection: scenes.ProjectionPerspective, FocalLengthMM: &focal, SensorWidthMM: &sensorWidth, SensorHeightMM: &sensorHeight, SensorFit: scenes.SensorFitHorizontal}}
	if _, err := applyPhysicalCameraEndpoint(endpoint, corners, nil, cameras, "main", identity, physical.Vec3{}, identity, samples, 1920, 1080); err != nil {
		t.Fatal(err)
	}
	first, ok := samples[0].CameraTransforms["main"]
	if !ok || first.Translation != (physical.Vec3{}) || first.Rotation != (physical.QuatXYZW{W: 1}) {
		t.Fatalf("camera should begin at its authored scene frame: %#v", first)
	}
	arrival, ok := samples[2].CameraTransforms["main"]
	if !ok || arrival.Translation == first.Translation {
		t.Fatalf("camera endpoint did not create a moving handoff by tick 2: %#v", arrival)
	}
	last, ok := samples[3].CameraTransforms["main"]
	if !ok || last != arrival {
		t.Fatalf("camera continued moving after endpoint arrival: tick 2=%#v tick 3=%#v", arrival, last)
	}
	finalCorners := corners
	rootMatrix, err := samples[2].DeviceRoot.Matrix()
	if err != nil {
		t.Fatal(err)
	}
	for i := range finalCorners {
		finalCorners[i], err = physical.TransformPoint(rootMatrix, finalCorners[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	wantCamera, residual, err := rendering.CompilePhysicalCameraEndpointBinding(endpoint, finalCorners, focal, sensorWidth, sensorHeight)
	if err != nil {
		t.Fatal(err)
	}
	if residual > 1 {
		t.Fatalf("endpoint projection residual %.3fpx exceeds tolerance", residual)
	}
	gotCamera, err := physicalComposeTRS(baseCamera, last)
	if err != nil {
		t.Fatal(err)
	}
	if delta := gotCamera.Translation.X - wantCamera.Translation.X; delta < -1e-7 || delta > 1e-7 {
		t.Fatalf("compiled camera does not reach target x: got %.9f want %.9f", gotCamera.Translation.X, wantCamera.Translation.X)
	}
	if delta := gotCamera.Translation.Y - wantCamera.Translation.Y; delta < -1e-7 || delta > 1e-7 {
		t.Fatalf("compiled camera does not reach target y: got %.9f want %.9f", gotCamera.Translation.Y, wantCamera.Translation.Y)
	}
	if delta := gotCamera.Translation.Z - wantCamera.Translation.Z; delta < -1e-7 || delta > 1e-7 {
		t.Fatalf("compiled camera does not reach target z: got %.9f want %.9f", gotCamera.Translation.Z, wantCamera.Translation.Z)
	}
}

func TestApplyPhysicalCameraEndpointRejectsUnqualifiedJointMotion(t *testing.T) {
	focal, sensorWidth, sensorHeight := 50.0, 36.0, 24.0
	camera := scenes.SceneCameraFrame{ID: "main", Frame: identityTRS(), Projection: scenes.ProjectionPerspective, FocalLengthMM: &focal, SensorWidthMM: &sensorWidth, SensorHeightMM: &sensorHeight, SensorFit: scenes.SensorFitHorizontal}
	samples := []rendering.PhysicalMotionSample{{JointAngles: map[string]float64{"hinge": .1}}, {JointAngles: map[string]float64{"hinge": .2}}}
	endpoint := &shotsv1.PhysicalCameraEndpoint{DisplayId: "screen", RasterWidth: 1920, RasterHeight: 1080, X: .36, Y: .2, Width: .56, Height: .56}
	corners := [4]physical.Vec3{{X: -.3, Y: .16875}, {X: .3, Y: .16875}, {X: .3, Y: -.16875}, {X: -.3, Y: -.16875}}
	if _, err := applyPhysicalCameraEndpoint(endpoint, corners, nil, []scenes.SceneCameraFrame{camera}, "main", identityTRS(), physical.Vec3{}, identityTRS(), samples, 1920, 1080); err == nil {
		t.Fatal("camera endpoint accepted an unqualified moving hinge")
	}
}
