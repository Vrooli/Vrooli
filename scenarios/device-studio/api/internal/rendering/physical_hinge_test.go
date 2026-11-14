package rendering

import (
	"encoding/json"
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
)

func articulatedFixture(t *testing.T) (devices.DeviceCandidate, motions.MotionCandidate, []PhysicalMotionSample, PhysicalGLBInventory) {
	t.Helper()
	manifest := validPhysicalManifest(t)
	device, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		t.Fatal(err)
	}
	device.Parts = append(device.Parts, devices.SemanticPart{
		ID: "lid", Role: "display", ParentID: "body", NodeIndex: 1, NodeName: "Lid", LocalFrame: identityPhysicalTRS(),
	})
	device.DisplayBindings[0].PartID = "lid"
	axis := devices.Vector3{Z: 1}
	pivot := devices.Vector3{X: 0.1, Y: 0.2, Z: 0.3}
	lower, upper := -0.1, 1.4
	device.Definition.Joints = []devices.Joint{{ID: "lid-hinge", ParentPart: "body", ChildPart: "lid", Axis: &axis, PivotMeters: &pivot, LowerRadians: &lower, UpperRadians: &upper, Status: "candidate_only"}}
	motion := motions.MotionCandidate{
		Version: motions.CandidateVersion, MotionID: "motion-a", Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 1,
		Tracks: []motions.MotionTrack{
			{Target: motions.TrackTarget{DeviceRoot: true}, Interpolation: motions.Linear, Keys: []motions.MotionKey{
				{Tick: 0, Transform: &motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}},
				{Tick: 1, Transform: &motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}},
			}},
			{Target: motions.TrackTarget{JointID: "lid-hinge"}, Interpolation: motions.Linear, Keys: []motions.MotionKey{
				{Tick: 0, AngleRad: float64Pointer(-0.1)},
				{Tick: 1, AngleRad: float64Pointer(1.2)},
			}},
		},
	}
	var samples []PhysicalMotionSample
	for i, at := range []composition.Rational{{Num: 0, Den: 1}, {Num: 1, Den: 1}} {
		evaluated, err := motions.SampleAt(motion, at)
		if err != nil {
			t.Fatal(err)
		}
		sample := PhysicalMotionSample{OutputOrdinal: uint32(i), OutputTime: at, RequestedMotionTick: at, SampledMotionTick: at}
		for _, track := range evaluated.Tracks {
			switch {
			case track.Target.DeviceRoot:
				sample.DeviceRoot = &physical.TRS{Translation: track.Transform.Translation, Rotation: track.Transform.Rotation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
			case track.Target.JointID != "":
				sample.JointAngles = map[string]float64{track.Target.JointID: *track.AngleRad}
			}
		}
		samples = append(samples, sample)
	}
	inventory := PhysicalGLBInventory{Nodes: []PhysicalGLBNode{
		{Index: 0, Name: "Body", Mesh: uint32Pointer(0), Frame: identityPhysicalTRS()},
		{Index: 1, Name: "Lid", Parent: uint32Pointer(0), Mesh: uint32Pointer(1), Frame: identityPhysicalTRS()},
	}}
	return device, motion, samples, inventory
}

func TestValidatePhysicalArticulationPinsOneCandidateHingeAndGLBHierarchy(t *testing.T) {
	device, motion, samples, inventory := articulatedFixture(t)
	binding, err := ValidatePhysicalArticulation(device, motion, samples, inventory)
	if err != nil {
		t.Fatal(err)
	}
	if binding.JointID != "lid-hinge" || binding.RenderScope != "local-candidate-render-v1" || binding.ParentNode != 0 || binding.ChildNode != 1 || binding.Axis.Z != 1 || binding.PivotMeters != (devices.Vector3{X: 0.1, Y: 0.2, Z: 0.3}) {
		t.Fatalf("unexpected immutable hinge binding: %+v", binding)
	}

	badInventory := inventory
	badInventory.Nodes = append([]PhysicalGLBNode(nil), inventory.Nodes...)
	badInventory.Nodes[1].Parent = uint32Pointer(7)
	if _, err := ValidatePhysicalArticulation(device, motion, samples, badInventory); err == nil {
		t.Fatal("hinge child with unknown GLB parent was accepted")
	}
}

func TestValidatePhysicalArticulationRejectsUnsupportedJointBindings(t *testing.T) {
	cases := map[string]func(*devices.DeviceCandidate, *motions.MotionCandidate, *[]PhysicalMotionSample, *PhysicalGLBInventory){
		"out of range angle": func(_ *devices.DeviceCandidate, _ *motions.MotionCandidate, samples *[]PhysicalMotionSample, _ *PhysicalGLBInventory) {
			(*samples)[1].JointAngles["lid-hinge"] = 1.5
		},
		"non-unit axis": func(device *devices.DeviceCandidate, _ *motions.MotionCandidate, _ *[]PhysicalMotionSample, _ *PhysicalGLBInventory) {
			device.Definition.Joints[0].Axis = &devices.Vector3{Z: 2}
		},
		"mesh intermediary": func(_ *devices.DeviceCandidate, _ *motions.MotionCandidate, _ *[]PhysicalMotionSample, inventory *PhysicalGLBInventory) {
			inventory.Nodes = []PhysicalGLBNode{
				{Index: 0, Name: "Body"},
				{Index: 1, Name: "Middle", Parent: uint32Pointer(0), Mesh: uint32Pointer(3)},
				{Index: 2, Name: "Lid", Parent: uint32Pointer(1), Mesh: uint32Pointer(1)},
			}
		},
		"nonuniform parent world scale": func(_ *devices.DeviceCandidate, _ *motions.MotionCandidate, _ *[]PhysicalMotionSample, inventory *PhysicalGLBInventory) {
			inventory.Nodes[0].Parent = uint32Pointer(2)
			inventory.Nodes = append(inventory.Nodes, PhysicalGLBNode{Index: 2, Name: "ScaledAncestor", Frame: physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 2, Y: 1, Z: 1}}})
		},
		"nonuniform device source scale": func(device *devices.DeviceCandidate, _ *motions.MotionCandidate, _ *[]PhysicalMotionSample, _ *PhysicalGLBInventory) {
			device.SourceToCanonical.Scale = physical.Vec3{X: 2, Y: 1, Z: 1}
		},
		"negative parent world scale": func(_ *devices.DeviceCandidate, _ *motions.MotionCandidate, _ *[]PhysicalMotionSample, inventory *PhysicalGLBInventory) {
			inventory.Nodes[0].Parent = uint32Pointer(2)
			inventory.Nodes = append(inventory.Nodes, PhysicalGLBNode{Index: 2, Name: "ReflectedAncestor", Frame: physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: -1, Y: -1, Z: -1}}})
		},
		"moving semantic child outside subtree": func(device *devices.DeviceCandidate, _ *motions.MotionCandidate, _ *[]PhysicalMotionSample, inventory *PhysicalGLBInventory) {
			device.Parts = append(device.Parts, devices.SemanticPart{ID: "screen", Role: "screen", ParentID: "lid", NodeIndex: 2, NodeName: "Screen", LocalFrame: identityPhysicalTRS()})
			inventory.Nodes = append(inventory.Nodes, PhysicalGLBNode{Index: 2, Name: "Screen", Parent: uint32Pointer(0)})
		},
		"two hinge tracks": func(_ *devices.DeviceCandidate, motion *motions.MotionCandidate, _ *[]PhysicalMotionSample, _ *PhysicalGLBInventory) {
			track := motion.Tracks[1]
			track.Target.JointID = "other-hinge"
			motion.Tracks = append(motion.Tracks, track)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			device, motion, samples, inventory := articulatedFixture(t)
			mutate(&device, &motion, &samples, &inventory)
			if _, err := ValidatePhysicalArticulation(device, motion, samples, inventory); err == nil {
				t.Fatal("unsupported hinge binding was accepted")
			}
		})
	}
}

func TestPhysicalHingeAllowsPositiveUniformNonunitScale(t *testing.T) {
	device, motion, samples, inventory := articulatedFixture(t)
	device.SourceToCanonical.Scale = physical.Vec3{X: 2, Y: 2, Z: 2}
	inventory.Nodes[0].Parent = uint32Pointer(2)
	inventory.Nodes = append(inventory.Nodes, PhysicalGLBNode{Index: 2, Name: "UniformAncestor", Frame: physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 3, Y: 3, Z: 3}}})
	if _, err := ValidatePhysicalArticulation(device, motion, samples, inventory); err != nil {
		t.Fatalf("positive uniform nonunit scale should preserve a revolute hinge: %v", err)
	}
}

func uint32Pointer(value uint32) *uint32    { return &value }
func float64Pointer(value float64) *float64 { return &value }

func TestArticulatedManifestVersionFiveKeepsExistingProfileAndLegacyJSONShape(t *testing.T) {
	manifest := validPhysicalManifest(t)
	legacyBytes, _, err := CanonicalPhysicalRenderManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(legacyBytes), "articulation_profile") {
		t.Fatal("legacy v2 manifest gained an articulated field")
	}
	legacyReceipt, err := json.Marshal(PhysicalRenderReceipt{Version: manifest.Version, Status: "succeeded"})
	if err != nil || strings.Contains(string(legacyReceipt), "articulation") {
		t.Fatalf("legacy receipt shape changed: %s (%v)", legacyReceipt, err)
	}

	device, motion, samples, _ := articulatedFixture(t)
	_, deviceBytes, deviceSHA, err := devices.CanonicalizeDeviceCandidate(device)
	if err != nil {
		t.Fatal(err)
	}
	_, motionBytes, motionSHA, err := motions.CanonicalizeCandidate(motion)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Version = PhysicalArticulatedManifestVersion
	manifest.Device.CanonicalJSON, manifest.Device.CanonicalSHA = deviceBytes, deviceSHA
	manifest.Motion.CanonicalJSON, manifest.Motion.CanonicalSHA = motionBytes, motionSHA
	manifest.Output.ArticulationProfile = PhysicalArticulationProfile
	samples[1].OutputTime = composition.Rational{Num: 1, Den: 24}
	samples[1].RequestedMotionTick = composition.Rational{Num: 1, Den: 1}
	samples[1].SampledMotionTick = composition.Rational{Num: 1, Den: 1}
	manifest.Output.Samples = samples
	if err := ValidatePhysicalRenderManifest(manifest); err != nil {
		t.Fatalf("valid articulated manifest rejected: %v", err)
	}
	encoded, _, err := CanonicalPhysicalRenderManifest(manifest)
	if err != nil || !strings.Contains(string(encoded), `"articulation_profile":"`+PhysicalArticulationProfile+`"`) {
		t.Fatalf("v5 manifest omitted its explicit profile: %v", err)
	}
	uniformPlacement := manifest
	uniformPlacement.Placement.SceneSupportWorld.Scale = physical.Vec3{X: 2, Y: 2, Z: 2}
	uniformPlacement.Placement.DeviceRootRest.Scale = physical.Vec3{X: 0.5, Y: 0.5, Z: 0.5}
	if err := ValidatePhysicalRenderManifest(uniformPlacement); err != nil {
		t.Fatalf("positive uniform placement scales should remain valid for the hinge profile: %v", err)
	}

	mutations := map[string]func(*PhysicalRenderManifest){
		"legacy version cannot claim articulation":   func(m *PhysicalRenderManifest) { m.Version = PhysicalMotionManifestVersion },
		"wrong articulation profile":                 func(m *PhysicalRenderManifest) { m.Output.ArticulationProfile = "other" },
		"recording input forbidden on still":         func(m *PhysicalRenderManifest) { m.Screen.Recording = &PhysicalRecordingInput{} },
		"nonuniform articulated scene support scale": func(m *PhysicalRenderManifest) { m.Placement.SceneSupportWorld.Scale = physical.Vec3{X: 1, Y: 2, Z: 1} },
		"nonuniform articulated device rest scale":   func(m *PhysicalRenderManifest) { m.Placement.DeviceRootRest.Scale = physical.Vec3{X: 1, Y: 1, Z: 2} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := manifest
			candidate.Output.Samples = append([]PhysicalMotionSample(nil), manifest.Output.Samples...)
			mutate(&candidate)
			if err := ValidatePhysicalRenderManifest(candidate); err == nil {
				t.Fatal("unsupported articulated protocol mutation was accepted")
			}
		})
	}

	recording := validPhysicalRecordingManifest(t)
	recordingDevice, recordingMotion, recordingSamples, _ := articulatedFixture(t)
	_, recordingDeviceBytes, recordingDeviceSHA, err := devices.CanonicalizeDeviceCandidate(recordingDevice)
	if err != nil {
		t.Fatal(err)
	}
	_, recordingMotionBytes, recordingMotionSHA, err := motions.CanonicalizeCandidate(recordingMotion)
	if err != nil {
		t.Fatal(err)
	}
	recording.Version = PhysicalArticulatedManifestVersion
	recording.Device.CanonicalJSON, recording.Device.CanonicalSHA = recordingDeviceBytes, recordingDeviceSHA
	recording.Motion.CanonicalJSON, recording.Motion.CanonicalSHA = recordingMotionBytes, recordingMotionSHA
	recording.Output.ArticulationProfile = PhysicalArticulationProfile
	recording.Output.Samples = recordingSamples
	recording.Output.Samples[1].OutputTime = composition.Rational{Num: 1, Den: 24}
	if err := ValidatePhysicalRenderManifest(recording); err != nil {
		t.Fatalf("articulated recording profile rejected: %v", err)
	}
}
