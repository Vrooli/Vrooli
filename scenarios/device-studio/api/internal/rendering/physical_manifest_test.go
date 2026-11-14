package rendering

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/scenes"
)

func TestPhysicalManifestPinsOwnerCanonicalBytesAndMotionSamples(t *testing.T) {
	manifest := validPhysicalManifest(t)
	first, digest, err := CanonicalPhysicalRenderManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	second, secondDigest, err := CanonicalPhysicalRenderManifest(manifest)
	if err != nil || digest != secondDigest || string(first) != string(second) {
		t.Fatalf("manifest canonical replay changed: digest=%s err=%v", secondDigest, err)
	}
	manifest.Output.Samples[0].DeviceRoot.Translation.X = 0.01
	if err := ValidatePhysicalRenderManifest(manifest); err == nil {
		t.Fatal("tampered pre-sampled root transform passed without matching motions.SampleAt")
	}
}

func TestPhysicalManifestRejectsHistoricalSourcePathWithoutRewritingCandidateHash(t *testing.T) {
	manifest := validPhysicalManifest(t)
	device, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		t.Fatal(err)
	}
	device.Definition.ModelCandidates = []devices.ModelCandidate{{ID: "historical", Status: "candidate_only", SourcePath: "/old/machine/model.glb"}}
	canonical, encoded, digest, err := devices.CanonicalizeDeviceCandidate(device)
	if err != nil || canonical.Definition.ModelCandidates[0].SourcePath == "" {
		t.Fatalf("fixture canonicalization lost path: %v", err)
	}
	manifest.Device.CanonicalJSON, manifest.Device.CanonicalSHA = encoded, digest
	if err := ValidatePhysicalRenderManifest(manifest); err == nil || !strings.Contains(err.Error(), "source_path") {
		t.Fatalf("historical host path should be an explicit compile failure, got %v", err)
	}
}

func TestPhysicalManifestRejectsChangedProfileAndArtifactFacts(t *testing.T) {
	tests := []struct {
		name string
		edit func(*PhysicalRenderManifest)
	}{
		{"quality changed without profile digest", func(m *PhysicalRenderManifest) { m.Profile.Samples++ }},
		{"asset reference differs from candidate", func(m *PhysicalRenderManifest) { m.Assets[0].Reference.SHA256 = strings.Repeat("a", 64) }},
		{"asset name escapes attempt root", func(m *PhysicalRenderManifest) { m.Assets[0].Name = "../geometry.glb" }},
		{"unbound display", func(m *PhysicalRenderManifest) { m.Screen.DisplayID = "rear" }},
		{"unselected camera", func(m *PhysicalRenderManifest) { m.Placement.SelectedCameraID = "missing" }},
		{"schedule has skipped ordinal", func(m *PhysicalRenderManifest) { m.Output.Samples[1].OutputOrdinal = 3 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := validPhysicalManifest(t)
			tt.edit(&manifest)
			if err := ValidatePhysicalRenderManifest(manifest); err == nil {
				t.Fatal("invalid physical manifest passed validation")
			}
		})
	}
}

func TestPhysicalStillProfilePins256SamplesAndMotionStays128(t *testing.T) {
	manifest := validPhysicalManifest(t)
	manifest.Output = PhysicalOutputPlan{
		ProfileID: PhysicalStillProfile, Kind: "still_png", Width: 2560, Height: 1440,
		Rate: composition.Rational{Num: 1, Den: 1}, FrameCount: 1, Timebase: composition.Rational{Num: 24, Den: 1},
		FinalPoseHold: "none", Samples: []PhysicalMotionSample{physicalScheduleSample(t, 0, composition.Rational{Num: 0, Den: 1}, composition.Rational{Num: 24, Den: 1})},
		Codec: "png", PixelFormat: "rgba8", OutputName: PhysicalStillOutputName,
	}
	manifest.Output.Samples[0].OutputTime = composition.Rational{}
	manifest.Version = PhysicalManifestVersion
	manifest.Profile.Samples = PhysicalStillSamples
	manifest.Profile.WallTimeSeconds = PhysicalStillWallTimeSeconds
	manifest.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(manifest.Profile)
	if err := ValidatePhysicalRenderManifest(manifest); err != nil {
		t.Fatalf("valid closed still profile rejected: %v", err)
	}
	for _, tc := range []struct {
		name    string
		backend string
		wall    uint32
		valid   bool
	}{
		{name: "legacy CPU still remains replayable", backend: "CPU", wall: PhysicalStillWallTimeSeconds, valid: true},
		{name: "bounded extended CPU still", backend: "CPU", wall: PhysicalStillCPUWallTimeSeconds, valid: true},
		{name: "arbitrary CPU duration rejected", backend: "CPU", wall: 181},
		{name: "OptiX retains original deadline", backend: "OPTIX", wall: PhysicalStillWallTimeSeconds, valid: true},
		{name: "CUDA retains original deadline", backend: "CUDA", wall: PhysicalStillWallTimeSeconds, valid: true},
		{name: "GPU cannot use extended CPU duration", backend: "OPTIX", wall: PhysicalStillCPUWallTimeSeconds},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := manifest
			candidate.Profile.Backend = tc.backend
			candidate.Profile.WallTimeSeconds = tc.wall
			candidate.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(candidate.Profile)
			err := ValidatePhysicalRenderManifest(candidate)
			if tc.valid && err != nil {
				t.Fatalf("valid backend-bound still limit rejected: %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatal("arbitrary or GPU-extended still wall time was accepted")
			}
		})
	}
	manifest.Profile.Samples = PhysicalMotionSamples
	manifest.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(manifest.Profile)
	if err := ValidatePhysicalRenderManifest(manifest); err == nil {
		t.Fatal("still render silently accepted the lower motion quality profile")
	}
}

func TestPhysicalPreviewProfileIsClosedAndReusesOnlyThePinnedPose(t *testing.T) {
	manifest := validPhysicalManifest(t)
	sample := manifest.Output.Samples[0]
	// A scheduled motion sample uses 0/1 for its first frame. The preview
	// still must normalize that to the still profile's canonical no-time value.
	sample.OutputTime = composition.Rational{Num: 0, Den: 1}
	manifest.Version = PhysicalPreviewManifestVersion
	manifest.Output = NewPhysicalPreviewOutputPlan(sample, manifest.Output.Timebase)
	manifest.Profile.Samples = PhysicalPreviewSamples
	manifest.Profile.WallTimeSeconds = PhysicalPreviewWallTimeSeconds
	manifest.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(manifest.Profile)
	if manifest.Output.Samples[0].RequestedMotionTick != sample.RequestedMotionTick || manifest.Output.Samples[0].SampledMotionTick != sample.SampledMotionTick {
		t.Fatal("preview constructor changed the exact saved pose sample")
	}
	if manifest.Output.Samples[0].OutputOrdinal != 0 || manifest.Output.Samples[0].OutputTime != (composition.Rational{}) {
		t.Fatalf("preview constructor did not normalize still output coordinates: %+v", manifest.Output.Samples[0])
	}
	if err := ValidatePhysicalRenderManifest(manifest); err != nil {
		t.Fatalf("closed physical preview profile rejected: %v", err)
	}
	if _, _, err := CanonicalPhysicalRenderManifest(manifest); err != nil {
		t.Fatalf("canonical preview manifest rejected: %v", err)
	}

	for name, mutate := range map[string]func(*PhysicalRenderManifest){
		"wrong profile version":  func(m *PhysicalRenderManifest) { m.Output.ProfileVersion++ },
		"wrong raster":           func(m *PhysicalRenderManifest) { m.Output.Width = 1280 },
		"wrong sample count":     func(m *PhysicalRenderManifest) { m.Profile.Samples = 65 },
		"wrong wall limit":       func(m *PhysicalRenderManifest) { m.Profile.WallTimeSeconds++ },
		"wrong manifest version": func(m *PhysicalRenderManifest) { m.Version = PhysicalManifestVersion },
		"wrong output path":      func(m *PhysicalRenderManifest) { m.Output.OutputName = PhysicalStillOutputName },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := manifest
			candidate.Output = manifest.Output
			candidate.Output.Samples = append([]PhysicalMotionSample(nil), manifest.Output.Samples...)
			candidate.Profile.DigestSHA256 = manifest.Profile.DigestSHA256
			mutate(&candidate)
			candidate.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(candidate.Profile)
			if err := ValidatePhysicalRenderManifest(candidate); err == nil {
				t.Fatal("invalid preview profile mutation was accepted")
			}
		})
	}
}

func TestPhysicalDiagnosticProfileIsDistinctLowSampleAndNotPreviewQuality(t *testing.T) {
	manifest := validPhysicalManifest(t)
	sample := manifest.Output.Samples[0]
	sample.OutputTime = composition.Rational{Num: 0, Den: 1}
	manifest.Version = PhysicalPreviewManifestVersion
	manifest.Output = NewPhysicalDiagnosticOutputPlan(sample, manifest.Output.Timebase)
	manifest.Profile.Samples = PhysicalDiagnosticSamples
	manifest.Profile.WallTimeSeconds = PhysicalDiagnosticWallTimeSeconds
	manifest.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(manifest.Profile)
	if manifest.Output.ProfileID == PhysicalPreviewProfile || manifest.Profile.Samples == PhysicalPreviewSamples {
		t.Fatal("diagnostic output reused the delivery preview profile")
	}
	if err := ValidatePhysicalRenderManifest(manifest); err != nil {
		t.Fatalf("closed diagnostic profile rejected: %v", err)
	}
	manifest.Profile.Samples = PhysicalPreviewSamples
	manifest.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(manifest.Profile)
	if err := ValidatePhysicalRenderManifest(manifest); err == nil {
		t.Fatal("diagnostic output silently accepted delivery-preview sample count")
	}
}

func TestPhysicalMotionManifestV2PinsToolPairAndInitialBound(t *testing.T) {
	manifest := validPhysicalManifest(t)
	if manifest.Version != PhysicalMotionManifestVersion {
		t.Fatalf("motion fixture protocol = %d, want v2", manifest.Version)
	}
	if err := ValidatePhysicalRenderManifest(manifest); err != nil {
		t.Fatalf("valid v2 motion manifest rejected: %v", err)
	}
	for name, mutate := range map[string]func(*PhysicalRenderManifest){
		"v1 cannot carry video": func(m *PhysicalRenderManifest) { m.Version = PhysicalManifestVersion },
		"wrong tool profile":    func(m *PhysicalRenderManifest) { m.Output.VideoToolProfileID = "other" },
		"wrong tool artifact":   func(m *PhysicalRenderManifest) { m.Output.VideoToolArtifactID = testDigest('a') },
		"over initial frame ceiling": func(m *PhysicalRenderManifest) {
			m.Output.FrameCount = 49
			m.Output.Samples = make([]PhysicalMotionSample, 49)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := validPhysicalManifest(t)
			mutate(&candidate)
			if err := ValidatePhysicalRenderManifest(candidate); err == nil {
				t.Fatal("unsupported or unpinned motion manifest passed")
			}
		})
	}
}

func TestPhysicalMotionOutputV2AllowsTenSecondsWithoutChangingV1Bounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		id     string
		ver    uint32
		motion string
		wall   uint32
		fps    int64
		count  uint32
	}{
		{name: "v1 24fps ceiling", id: PhysicalMotionOutput, ver: 1, motion: PhysicalMotionProfile, wall: PhysicalMotionWallTimeSeconds, fps: 24, count: 48},
		{name: "v2 24fps ten seconds", id: PhysicalMotionOutputV2, ver: 2, motion: PhysicalMotionProfileV2, wall: PhysicalMotionV2WallTimeSeconds, fps: 24, count: 240},
		{name: "v2 30fps ten seconds", id: PhysicalMotionOutputV2, ver: 2, motion: PhysicalMotionProfileV2, wall: PhysicalMotionV2WallTimeSeconds, fps: 30, count: 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest := validPhysicalManifest(t)
			manifest.Output.ProfileID = tc.id
			manifest.Output.ProfileVersion = tc.ver
			manifest.Output.MotionProfile = tc.motion
			manifest.Output.Rate = composition.Rational{Num: tc.fps, Den: 1}
			manifest.Output.FrameCount = tc.count
			manifest.Output.Samples = make([]PhysicalMotionSample, tc.count)
			for i := uint32(0); i < tc.count; i++ {
				manifest.Output.Samples[i] = physicalScheduleSample(t, i, composition.Rational{Num: int64(i), Den: tc.fps}, manifest.Output.Timebase)
			}
			manifest.Profile.WallTimeSeconds = tc.wall
			manifest.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(manifest.Profile)
			if err := ValidatePhysicalRenderManifest(manifest); err != nil {
				t.Fatalf("closed profile rejected at its maximum frame count: %v", err)
			}

			manifest.Output.FrameCount++
			manifest.Output.Samples = append(manifest.Output.Samples, physicalScheduleSample(t, tc.count, composition.Rational{Num: int64(tc.count), Den: tc.fps}, manifest.Output.Timebase))
			if err := ValidatePhysicalRenderManifest(manifest); err == nil {
				t.Fatal("frame beyond the profile duration was accepted")
			}
		})
	}
}

func TestPhysicalMotionProfilesRequireTheirOwnWallLimits(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      string
		version uint32
		motion  string
		wall    uint32
	}{
		{name: "v1 cannot borrow v2 wall", id: PhysicalMotionOutput, version: 1, motion: PhysicalMotionProfile, wall: PhysicalMotionV2WallTimeSeconds},
		{name: "v2 cannot use v1 wall", id: PhysicalMotionOutputV2, version: 2, motion: PhysicalMotionProfileV2, wall: PhysicalMotionWallTimeSeconds},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest := validPhysicalManifest(t)
			manifest.Output.ProfileID, manifest.Output.ProfileVersion, manifest.Output.MotionProfile = tc.id, tc.version, tc.motion
			manifest.Profile.WallTimeSeconds = tc.wall
			manifest.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(manifest.Profile)
			if err := ValidatePhysicalRenderManifest(manifest); err == nil {
				t.Fatal("motion profile accepted another version's wall limit")
			}
		})
	}
}

func TestPhysicalManifestRejectsAffineNormalMappingThatWouldMisalignTangentBasis(t *testing.T) {
	identity := [6]float64{1, 0, 0, 0, 1, 0}
	normal := scenes.SceneMaterialInput{Socket: scenes.SocketNormal, Coordinates: scenes.CoordinatesUVSet, AffineUV: &identity}
	candidate := scenes.SceneCandidate{Materials: []scenes.SceneMaterial{{ID: "mat", Inputs: []scenes.SceneMaterialInput{normal}}}}
	if err := validatePhysicalMaterialRuntimeSupport(candidate); err != nil {
		t.Fatalf("identity UV mapping should preserve the tangent basis: %v", err)
	}
	transformed := [6]float64{2, 0, 0, 0, 2, 0}
	candidate.Materials[0].Inputs[0].AffineUV = &transformed
	if err := validatePhysicalMaterialRuntimeSupport(candidate); err == nil {
		t.Fatal("nonidentity affine UV was accepted for a tangent normal map")
	}
}

func TestPhysicalManifestSupportsFractionalOutputTicksAndExplicitFinalHold(t *testing.T) {
	manifest := validPhysicalManifest(t)
	manifest.Output.Rate = composition.Rational{Num: 30, Den: 1}
	manifest.Output.FrameCount = 3
	manifest.Output.Samples = []PhysicalMotionSample{
		physicalScheduleSample(t, 0, composition.Rational{Num: 0, Den: 1}, composition.Rational{Num: 24, Den: 1}),
		physicalScheduleSample(t, 1, composition.Rational{Num: 1, Den: 30}, composition.Rational{Num: 24, Den: 1}),
		physicalScheduleSample(t, 2, composition.Rational{Num: 2, Den: 30}, composition.Rational{Num: 24, Den: 1}),
	}
	manifest.Output.Samples[0].RequestedMotionTick = composition.Rational{Num: 0, Den: 1}
	manifest.Output.Samples[0].SampledMotionTick = composition.Rational{Num: 0, Den: 1}
	manifest.Output.Samples[1].RequestedMotionTick = composition.Rational{Num: 4, Den: 5}
	manifest.Output.Samples[1].SampledMotionTick = composition.Rational{Num: 4, Den: 5}
	manifest.Output.Samples[2].RequestedMotionTick = composition.Rational{Num: 8, Den: 5}
	manifest.Output.Samples[2].SampledMotionTick = composition.Rational{Num: 1, Den: 1}
	if err := ValidatePhysicalRenderManifest(manifest); err != nil {
		t.Fatalf("valid 24-ticks/sec to 30fps schedule with explicit final hold was rejected: %v", err)
	}
	manifest.Output.Samples[1].RequestedMotionTick = composition.Rational{Num: 1, Den: 1}
	if err := ValidatePhysicalRenderManifest(manifest); err == nil {
		t.Fatal("non-clock-aligned requested tick was accepted")
	}
}

func physicalScheduleSample(t *testing.T, ordinal uint32, outputTime, motionTimebase composition.Rational) PhysicalMotionSample {
	t.Helper()
	seconds := new(big.Rat).SetFrac(big.NewInt(outputTime.Num), big.NewInt(outputTime.Den))
	tick := new(big.Rat).Mul(seconds, new(big.Rat).SetFrac(big.NewInt(motionTimebase.Num), big.NewInt(motionTimebase.Den)))
	requested, err := rationalFromBig(tick)
	if err != nil {
		t.Fatal(err)
	}
	sampled := requested
	if tick.Cmp(big.NewRat(1, 1)) > 0 {
		sampled = composition.Rational{Num: 1, Den: 1}
	}
	return PhysicalMotionSample{OutputOrdinal: ordinal, OutputTime: outputTime, RequestedMotionTick: requested, SampledMotionTick: sampled, DeviceRoot: &physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}}
}

func validPhysicalManifest(t *testing.T) PhysicalRenderManifest {
	t.Helper()
	deviceCandidate := devices.DeviceCandidate{
		Version: devices.DeviceCandidateVersion,
		Definition: devices.Definition{
			SchemaVersion: "device-definition/v1", ID: "phone-a", Group: devices.GroupPhone,
			Kind: "phone", IdentityStatus: devices.IdentityGeneric, Model: "Fixture phone", Qualification: "unqualified",
			Dimensions: map[string]devices.Dimensions{"default": {
				Width: knownDeviceMeasurement(70), Height: knownDeviceMeasurement(145), Depth: knownDeviceMeasurement(8),
			}},
			Displays: []devices.DisplaySurface{{ID: "front", Topology: "planar", PhysicalBoundary: devices.Region{Status: "unknown"}, ActiveArea: devices.Region{Status: "unknown"}, AppSafeArea: devices.Region{Status: "unknown"}}},
			Joints:   []devices.Joint{}, SupportedPoses: []devices.Pose{}, SupportedMotion: []devices.Capability{},
			Supports:        []devices.SupportCapability{{Kind: "tabletop", InterfaceID: "horizontal_surface", Status: "candidate_only"}},
			ModelCandidates: []devices.ModelCandidate{}, Provenance: []devices.Evidence{{Source: "fixture", Kind: "test"}},
		},
		Geometry:     physical.ArtifactRef{BundleID: "phone-bundle", BundleRevision: 1, ManifestSHA256: testDigest('a'), ReviewSHA256: testDigest('b'), ArtifactID: "phone-geometry", Role: library.RoleGeometryGLB, SizeBytes: 1024, SHA256: testDigest('c')},
		GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention, SourceToCanonical: identityPhysicalTRS(),
		Parts:           []devices.SemanticPart{{ID: "body", Role: "assembly", NodeIndex: 0, NodeName: "Body", LocalFrame: identityPhysicalTRS()}},
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: "front", PartID: "body", LocalFrame: identityPhysicalTRS()}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: "device-base", InterfaceID: "horizontal_surface", PartID: "body", LocalFrame: identityPhysicalTRS()}},
	}
	deviceCanonical, deviceBytes, deviceSHA, err := devices.CanonicalizeDeviceCandidate(deviceCandidate)
	if err != nil {
		t.Fatalf("canonical device fixture: %v", err)
	}
	_ = deviceCanonical
	sceneRef := physical.ArtifactRef{BundleID: "scene-bundle", BundleRevision: 1, ManifestSHA256: testDigest('d'), ReviewSHA256: testDigest('e'), ArtifactID: "scene-geometry", Role: library.RoleGeometryGLB, SizeBytes: 2048, SHA256: testDigest('f')}
	sceneCandidate := scenes.SceneCandidate{
		Version: scenes.CandidateVersion, SceneID: "scene-a", CoordinateConvention: physical.CoordinateConvention,
		Artifacts:     []scenes.SceneArtifactBinding{{ID: "scene-geometry", Ref: sceneRef, SourceUnit: "m", SourceToCanonical: trsPtr(identityPhysicalTRS())}},
		Parts:         []scenes.ScenePart{{ID: "table", Role: "support", ArtifactBindingID: "scene-geometry", NodeIndex: 0, NodeName: "Table", LocalFrame: identityPhysicalTRS()}},
		SupportFrames: []scenes.SceneSupportFrame{{ID: "table-top", InterfaceID: "horizontal_surface", FramePartID: "table", LocalFrame: identityPhysicalTRS()}},
		CameraFrames:  []scenes.SceneCameraFrame{{ID: "camera-main", Frame: identityPhysicalTRS(), Projection: scenes.ProjectionPerspective, FocalLengthMM: floatPtr(85), SensorWidthMM: floatPtr(36), SensorHeightMM: floatPtr(24), SensorFit: scenes.SensorFitHorizontal, NearClipM: 0.01, FarClipM: 100}},
		Environment:   scenes.SceneEnvironment{Model: "CONSTANT_V1", LinearRGB: [3]float64{0.5, 0.5, 0.5}, Strength: 0.2},
	}
	_, sceneBytes, sceneSHA, err := scenes.CanonicalizeCandidate(sceneCandidate)
	if err != nil {
		t.Fatalf("canonical scene fixture: %v", err)
	}
	motionCandidate := motions.MotionCandidate{
		Version: motions.CandidateVersion, MotionID: "motion-a", Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 1,
		Tracks: []motions.MotionTrack{{Target: motions.TrackTarget{DeviceRoot: true}, Interpolation: motions.Linear, Keys: []motions.MotionKey{
			{Tick: 0, Transform: &motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}},
			{Tick: 1, Transform: &motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}},
		}}},
	}
	_, motionBytes, motionSHA, err := motions.CanonicalizeCandidate(motionCandidate)
	if err != nil {
		t.Fatalf("canonical motion fixture: %v", err)
	}
	first, err := motions.SampleAt(motionCandidate, composition.Rational{Num: 0, Den: 1})
	if err != nil {
		t.Fatal(err)
	}
	root := physical.TRS{Translation: first.Tracks[0].Transform.Translation, Rotation: first.Tracks[0].Transform.Rotation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	second, err := motions.SampleAt(motionCandidate, composition.Rational{Num: 1, Den: 1})
	if err != nil {
		t.Fatal(err)
	}
	rootEnd := physical.TRS{Translation: second.Tracks[0].Transform.Translation, Rotation: second.Tracks[0].Transform.Rotation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	assets := []PhysicalAssetInput{
		{Role: "device_geometry", BindingID: "device_geometry", Reference: deviceCandidate.Geometry, Name: "asset-device.glb"},
		{Role: "scene_geometry", BindingID: "scene-geometry", Reference: sceneRef, Name: "asset-scene.glb"},
	}
	profile := PhysicalRenderProfile{
		ID: PhysicalRendererProfile, Version: 1, BlenderArtifactID: PhysicalBlenderArtifact, BlenderTreeSHA256: PhysicalBlenderTreeSHA,
		BlenderBinarySHA256: PhysicalBlenderBinarySHA, BlenderVersion: PhysicalBlenderVersion, AdapterSHA256: testDigest('3'), Backend: "OPTIX", Engine: "CYCLES", Samples: PhysicalMotionSamples, Seed: 1307,
		AdaptiveSampling: false, Denoise: false, ColorManagement: PhysicalColorProfile, ColorView: "Standard", ColorLook: "None", Exposure: "0", Gamma: "1",
		WallTimeSeconds: PhysicalMotionWallTimeSeconds, MemoryMaxBytes: PhysicalOperationMemoryMaxBytes, TasksMax: PhysicalOperationTasksMax,
		CPUWeight: PhysicalOperationCPUWeight, Threads: PhysicalOperationThreads,
	}
	profile.DigestSHA256, err = PhysicalRenderProfileDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	return PhysicalRenderManifest{
		Version: PhysicalMotionManifestVersion, JobID: "job-1", AttemptID: "attempt-1", InputFingerprint: testDigest('4'), WorkerSHA256: testDigest('5'),
		Device:    PhysicalCandidateSnapshot{ID: "phone-a", Revision: 1, CanonicalSHA: deviceSHA, CanonicalJSON: deviceBytes},
		Scene:     PhysicalCandidateSnapshot{ID: "scene-a", Revision: 1, CanonicalSHA: sceneSHA, CanonicalJSON: sceneBytes},
		Motion:    PhysicalCandidateSnapshot{ID: "motion-a", Revision: 1, CanonicalSHA: motionSHA, CanonicalJSON: motionBytes},
		Assets:    assets,
		Screen:    PhysicalScreenInput{DisplayID: "front", OriginalMediaID: "media-1", OriginalMediaSHA256: testDigest('6'), Raster: ImageBinding{Name: "screen.png", SHA256: testDigest('7'), SizeBytes: 500}, PresentationProfile: "device-studio-presentation-v3", PresentationVersion: "3", ConfigSHA256: testDigest('8'), LayoutSHA256: testDigest('9'), RasterSHA256: testDigest('7'), Width: 1206, Height: 2622, UVSet: "TEXCOORD_0", MappingProfile: PhysicalScreenMappingProfile, PrimitiveIndex: 0, MaterialSlot: 0},
		Placement: PhysicalPlacement{DeviceSupportID: "device-base", SceneSupportID: "table-top", SelectedCameraID: "camera-main", SceneSupportWorld: identityPhysicalTRS(), SupportPivot: physical.Vec3{}, DeviceRootRest: identityPhysicalTRS()},
		Profile:   profile,
		Output:    PhysicalOutputPlan{ProfileID: PhysicalMotionOutput, ProfileVersion: 1, MotionProfile: PhysicalMotionProfile, Kind: "motion_mp4", Width: 1920, Height: 1080, Codec: "h264", PixelFormat: "yuv420p", ColorPolicy: "sdr-unmarked-default709-v1", Rate: composition.Rational{Num: 24, Den: 1}, FrameCount: 2, MotionTicks: 1, Timebase: composition.Rational{Num: 24, Den: 1}, FinalPoseHold: "last-key-through-output-end-v1", Samples: []PhysicalMotionSample{{OutputOrdinal: 0, OutputTime: composition.Rational{Num: 0, Den: 1}, RequestedMotionTick: composition.Rational{Num: 0, Den: 1}, SampledMotionTick: composition.Rational{Num: 0, Den: 1}, DeviceRoot: &root}, {OutputOrdinal: 1, OutputTime: composition.Rational{Num: 1, Den: 24}, RequestedMotionTick: composition.Rational{Num: 1, Den: 1}, SampledMotionTick: composition.Rational{Num: 1, Den: 1}, DeviceRoot: &rootEnd}}, OutputName: PhysicalMotionOutputName, VideoToolProfileID: PhysicalMotionVideoProfileID, VideoToolArtifactID: PhysicalMotionVideoArtifactID},
	}
}

func identityPhysicalTRS() physical.TRS {
	return physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}
func trsPtr(value physical.TRS) *physical.TRS { return &value }
func floatPtr(value float64) *float64         { return &value }
func testDigest(ch byte) string               { sum := sha256.Sum256([]byte{ch}); return hex.EncodeToString(sum[:]) }
func knownDeviceMeasurement(value float64) devices.Measurement {
	return devices.Measurement{Value: &value, Unit: "mm", Meaning: "test dimension", Configuration: "default", Method: "measured", Uncertainty: devices.Uncertainty{Status: "source_unspecified"}, Evidence: []devices.Evidence{{Source: "test", Kind: "fixture"}}}
}
