package shots

import (
	"os"
	"path/filepath"
	"testing"

	"device-studio/internal/devices"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/scenes"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestPublicPhysicalCompatibilityFamiliesFailClosed(t *testing.T) {
	definitions := loadDeviceDefinitionsForCompatibilityTest(t)
	byID := make(map[string]devices.Definition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	identity := physical.QuatXYZW{W: 1}
	quarterTurn := physical.QuatXYZW{Z: 0.7071067811865476, W: 0.7071067811865476}
	tests := []struct {
		name       string
		deviceID   string
		kind       string
		iface      string
		panel      string
		pose       string
		rotation   physical.QuatXYZW
		motion     motions.MotionCandidate
		mutate     func(*devices.Definition)
		wantStatus devices.CompatibilityStatus
	}{
		{name: "monitor bracket interface mismatch", deviceID: "dell-ultrasharp-u2723qe", kind: "wall_mount", iface: "vesa-75", panel: "panel", pose: "landscape", rotation: identity, mutate: func(d *devices.Definition) { setSupport(d, "wall_mount", "vesa-100", "qualified") }, wantStatus: devices.Incompatible},
		{name: "monitor unknown bracket needs review", deviceID: "dell-ultrasharp-u2723qe", kind: "wall_mount", iface: "vesa-100", panel: "panel", pose: "landscape", rotation: identity, wantStatus: devices.NeedsReview},
		{name: "tablet unsupported wall mount", deviceID: "ipad-pro-13-m4-2024", kind: "wall_mount", iface: "vesa-100", panel: "display", pose: "portrait_bare", rotation: identity, wantStatus: devices.Incompatible},
		{name: "smart display unsupported orientation", deviceID: "google-nest-hub-2nd-gen", kind: "tabletop", iface: "base", panel: "display", pose: "tabletop-facing", rotation: quarterTurn, mutate: func(d *devices.Definition) {
			setSupport(d, "tabletop", "base", "qualified")
			setPoseRotation(d, "tabletop-facing", identity)
		}, wantStatus: devices.Incompatible},
		{name: "vehicle panel must belong to pinned host", deviceID: "bmw-i4-2022-curved-display-host", kind: "vehicle_integrated", iface: "oem-integrated", panel: "unlisted-panel", pose: "driver-view", rotation: identity, wantStatus: devices.Incompatible},
		{name: "fold joint motion outside limit", deviceID: "galaxy-z-fold6", kind: "tabletop", iface: "base", panel: "inner-display", pose: "open-flat", rotation: identity, motion: motions.MotionCandidate{MotionID: "fold-over-limit", Tracks: []motions.MotionTrack{{Target: motions.TrackTarget{JointID: "hinge"}, Keys: []motions.MotionKey{{AngleRad: floatPointerCompatibility(2)}}}}}, mutate: func(d *devices.Definition) {
			d.Joints = []devices.Joint{{ID: "hinge", ParentPart: "body", ChildPart: "cover", Status: "qualified", Axis: &physical.Vec3{X: 1}, PivotMeters: &physical.Vec3{}, LowerRadians: floatPointerCompatibility(-1), UpperRadians: floatPointerCompatibility(1)}}
		}, wantStatus: devices.Incompatible},
		{name: "unknown phone cutout needs review", deviceID: "iphone-16-pro", kind: "tabletop", iface: "base", panel: "front", pose: "front-facing", rotation: identity, mutate: func(d *devices.Definition) {
			if len(d.Displays) > 0 {
				d.Displays[0].HardwareOcclusions = append(d.Displays[0].HardwareOcclusions, devices.HardwareOcclusion{ID: "test-cutout", Kind: "camera_cutout", Region: devices.Region{Status: "unknown", Shape: "rect"}})
			}
		}, wantStatus: devices.NeedsReview},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			definition := byID[tc.deviceID]
			if tc.mutate != nil {
				tc.mutate(&definition)
			}
			if tc.panel == "unlisted-panel" {
				// CheckDisplay's public compatibility outcome is independent of a
				// scene slot; the host owns the panel identity.
				if got := devices.CheckDisplay(definition, tc.panel); got.Status != tc.wantStatus {
					t.Fatalf("CheckDisplay = %+v, want %s", got, tc.wantStatus)
				}
				return
			}
			binding := &shotsv1.PhysicalShotBinding{DisplayId: tc.panel, DeviceSupportId: "device-support", SceneSupportId: "support-frame", SceneSupportSlotId: "support-slot", DeviceRootRest: &physicalv1.TRS{TranslationM: &physicalv1.Vec3{}, RotationXyzw: &physicalv1.QuatXYZW{X: tc.rotation.X, Y: tc.rotation.Y, Z: tc.rotation.Z, W: tc.rotation.W}, Scale: &physicalv1.Vec3{X: 1, Y: 1, Z: 1}}}
			scene := scenes.SceneCandidate{Version: scenes.CandidateVersionSlots, SupportFrames: []scenes.SceneSupportFrame{{ID: "support-frame", InterfaceID: tc.iface}}, SupportSlots: []scenes.SceneSupportSlot{{ID: "support-slot", FrameID: "support-frame", Kind: tc.kind, RequiredInterface: tc.iface, AllowedGroups: []string{string(definition.Group)}, DimensionState: firstDimensionState(definition), RequiredPose: tc.pose, MaxWidthM: 2, MaxHeightM: 2, MaxDepthM: 2}}}
			motion := tc.motion
			got := evaluatePhysicalCompatibility(binding, devices.DeviceCandidate{Definition: definition}, scene, motion)
			if got.Status != tc.wantStatus {
				t.Fatalf("compatibility = %+v, want %s", got, tc.wantStatus)
			}
		})
	}
}

func loadDeviceDefinitionsForCompatibilityTest(t *testing.T) []devices.Definition {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(dir, "../../../assets/devices"))
	definitions, err := devices.LoadDefinitions(root)
	if err != nil {
		t.Fatal(err)
	}
	return definitions
}

func setSupport(definition *devices.Definition, kind, iface, status string) {
	for i := range definition.Supports {
		if definition.Supports[i].Kind == kind {
			definition.Supports[i].InterfaceID = iface
			definition.Supports[i].Status = status
			return
		}
	}
	definition.Supports = append(definition.Supports, devices.SupportCapability{Kind: kind, InterfaceID: iface, Status: status})
}

func setPoseRotation(definition *devices.Definition, name string, rotation physical.QuatXYZW) {
	for i := range definition.SupportedPoses {
		if definition.SupportedPoses[i].Name == name {
			definition.SupportedPoses[i].Status = "qualified"
			definition.SupportedPoses[i].RootRotation = &rotation
			return
		}
	}
	definition.SupportedPoses = append(definition.SupportedPoses, devices.Pose{Name: name, Status: "qualified", RootRotation: &rotation})
}

func firstDimensionState(definition devices.Definition) string {
	for state := range definition.Dimensions {
		return state
	}
	return "unknown"
}

func floatPointerCompatibility(value float64) *float64 { return &value }
