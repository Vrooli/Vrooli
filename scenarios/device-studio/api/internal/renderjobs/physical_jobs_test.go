package renderjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"strconv"
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/presentations"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestPhysicalInputBuilderFingerprintBindsCompiledShotClosure(t *testing.T) {
	digest := func(value string) string { return digestBytes([]byte(value)) }
	deviceGeometry := physical.ArtifactRef{BundleID: "device-bundle", BundleRevision: 2, ManifestSHA256: digest("device-manifest"), ReviewSHA256: digest("device-review"), ArtifactID: "device-glb", Role: library.RoleGeometryGLB, SizeBytes: 100, SHA256: digest("device-bytes")}
	sceneGeometry := physical.ArtifactRef{BundleID: "scene-bundle", BundleRevision: 3, ManifestSHA256: digest("scene-manifest"), ReviewSHA256: digest("scene-review"), ArtifactID: "scene-glb", Role: library.RoleGeometryGLB, SizeBytes: 200, SHA256: digest("scene-bytes")}
	deviceJSON := []byte(`{"geometry_sha256":"` + deviceGeometry.SHA256 + `","display_mask_sha256":"` + digest("mask") + `"}`)
	sceneJSON := []byte(`{"reflection_profile":"` + digest("reflection") + `"}`)
	base := physicalClosure{
		Version: physicalClosureVersion, ProjectID: "project", ShotID: "shot", Revision: 1, RecipeSHA256: digest("shot-recipe"),
		WorkerKey: "workers/" + digest("worker") + "/device-studio-api", WorkerSHA256: digest("worker"), AdapterSHA256: digest("adapter"), ScreenKey: "screen/" + digest("screen"),
		Manifest: rendering.PhysicalRenderManifest{
			Device:    rendering.PhysicalCandidateSnapshot{ID: "device", Revision: 2, CanonicalJSON: deviceJSON, CanonicalSHA: digest(string(deviceJSON))},
			Scene:     rendering.PhysicalCandidateSnapshot{ID: "scene", Revision: 3, CanonicalJSON: sceneJSON, CanonicalSHA: digest(string(sceneJSON))},
			Motion:    rendering.PhysicalCandidateSnapshot{ID: "motion", Revision: 1, CanonicalSHA: digest("motion-candidate"), CanonicalJSON: []byte(`{"motion":"root-track"}`)},
			Assets:    []rendering.PhysicalAssetInput{{Role: "device_geometry", BindingID: "device_geometry", Reference: deviceGeometry, Name: "device.glb"}, {Role: "scene_geometry", BindingID: "scene_geometry", Reference: sceneGeometry, Name: "scene.glb"}},
			Screen:    rendering.PhysicalScreenInput{DisplayID: "front", OriginalMediaID: "screen-media", OriginalMediaSHA256: digest("screen-source"), Raster: rendering.ImageBinding{Name: "screen.png", SHA256: digest("screen-raster"), SizeBytes: 100}, PresentationProfile: "presentation-v3", PresentationVersion: "3", ConfigSHA256: digest("presentation-config"), LayoutSHA256: digest("presentation-layout"), RasterSHA256: digest("screen-raster"), Width: 1206, Height: 2622, UVSet: "TEXCOORD_0", MappingProfile: "physical-screen-mapping-v1"},
			Placement: rendering.PhysicalPlacement{DeviceSupportID: "base", SceneSupportID: "table", SelectedCameraID: "camera-front"},
			Profile:   rendering.PhysicalRenderProfile{ID: "physical-still-v1", DigestSHA256: digest("render-profile"), Backend: "OPTIX", Samples: 256},
			Output:    rendering.PhysicalOutputPlan{ProfileID: rendering.PhysicalStillProfile, Kind: "still_png", Width: 2560, Height: 1440, Codec: "png", OutputName: rendering.PhysicalStillOutputName},
		},
	}
	route := routePin{Root: "/primary/device-studio", TestMode: false}
	inputFor := func(closure physicalClosure) (jobs.Input, string) {
		t.Helper()
		encoded, err := encodePhysicalClosure(closure)
		if err != nil {
			t.Fatal(err)
		}
		encoded = append(encoded, '\n') // same bytes hashed and pinned by physical submit
		closureSHA := digestBytes(encoded)
		outputProfile := OutputProfilePNG
		if closure.Manifest.Output.ProfileID == rendering.PhysicalPreviewProfile {
			outputProfile = OutputProfilePhysicalPreviewPNG
		}
		input := physicalJobInput(route, closure, closureBlobPrefix+closureSHA+".json", closureSHA, closure.WorkerKey, closure.WorkerSHA256, outputProfile)
		_, fingerprint, err := jobs.CanonicalInput(input)
		if err != nil {
			t.Fatal(err)
		}
		return input, fingerprint
	}
	_, baseline := inputFor(base)
	mutations := []struct {
		name string
		edit func(*physicalClosure)
	}{
		{"source raster", func(c *physicalClosure) {
			c.Manifest.Screen.OriginalMediaSHA256 = digest("screen-source-v2")
			c.Manifest.Screen.RasterSHA256 = digest("screen-raster-v2")
			c.Manifest.Screen.Raster.SHA256 = digest("screen-raster-v2")
		}},
		{"device geometry and display mask", func(c *physicalClosure) {
			raw := []byte(`{"geometry_sha256":"` + digest("device-geometry-v2") + `","display_mask_sha256":"` + digest("mask-v2") + `"}`)
			c.Manifest.Device.CanonicalJSON = raw
			c.Manifest.Device.CanonicalSHA = digest(string(raw))
			c.Manifest.Assets[0].Reference.SHA256 = digest("device-geometry-v2")
		}},
		{"presentation configuration and layout", func(c *physicalClosure) {
			c.Manifest.Screen.ConfigSHA256 = digest("config-v2")
			c.Manifest.Screen.LayoutSHA256 = digest("layout-v2")
		}},
		{"scene geometry and reflection", func(c *physicalClosure) {
			raw := []byte(`{"reflection_profile":"` + digest("reflection-v2") + `"}`)
			c.Manifest.Scene.CanonicalJSON = raw
			c.Manifest.Scene.CanonicalSHA = digest(string(raw))
			c.Manifest.Assets[1].Reference.SHA256 = digest("scene-geometry-v2")
		}},
		{"camera and placement", func(c *physicalClosure) {
			c.Manifest.Placement.SelectedCameraID = "camera-side"
			c.Manifest.Placement.SupportPivot.X = 0.125
		}},
		{"motion candidate", func(c *physicalClosure) { c.Manifest.Motion.CanonicalSHA = digest("motion-candidate-v2") }},
		{"render profile", func(c *physicalClosure) {
			c.Manifest.Profile.Backend = "CPU"
			c.Manifest.Profile.DigestSHA256 = digest("render-profile-v2")
		}},
		{"output profile and raster", func(c *physicalClosure) {
			c.Manifest.Output.Width = 1920
			c.Manifest.Output.ProfileID = "device-studio-physical-preview-960x540-v1"
		}},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(base)
			if err != nil {
				t.Fatal(err)
			}
			var changed physicalClosure
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			tc.edit(&changed)
			input, fingerprint := inputFor(changed)
			if fingerprint == baseline {
				t.Fatalf("compiled physical closure dependency did not invalidate input fingerprint: %s", tc.name)
			}
			binding := bindingByRole(input, "physical_closure")
			if binding.SHA256 == "" || binding.ArtifactID != closureBlobPrefix+binding.SHA256+".json" {
				t.Fatalf("closure bytes are not the bound cache dependency: %+v", binding)
			}
		})
	}
}

type qualificationReadabilityFixture struct {
	revision  shots.Revision
	review    shots.ReadabilityReview
	reviewErr error
}

func (f qualificationReadabilityFixture) ResolveForRender(_ context.Context, shotID string, revision int64) (shots.ResolvedShot, error) {
	if shotID != f.revision.ShotID || revision != f.revision.Revision {
		return shots.ResolvedShot{}, errors.New("unexpected saved shot revision lookup")
	}
	return shots.ResolvedShot{ProjectID: "project", Shot: shots.Shot{ID: shotID, ProjectID: "project"}, Revision: f.revision}, nil
}

func (f qualificationReadabilityFixture) GetOutputReadabilityReview(_ context.Context, projectID, shotID string, revision int64, renderID, outputSHA string) (shots.ReadabilityReview, error) {
	if projectID != "project" || shotID != "shot" || revision != 2 || renderID != "render" || outputSHA != strings.Repeat("b", 64) {
		return shots.ReadabilityReview{}, errors.New("unexpected exact readability review lookup")
	}
	return f.review, f.reviewErr
}

type qualificationProofResolverOnly struct {
	revision shots.Revision
}

func (f qualificationProofResolverOnly) ResolveForRender(_ context.Context, shotID string, revision int64) (shots.ResolvedShot, error) {
	return shots.ResolvedShot{ProjectID: "project", Shot: shots.Shot{ID: shotID, ProjectID: "project"}, Revision: f.revision}, nil
}

func TestDeclaredReadabilityProofMustHaveExactAcceptedReviewForQualification(t *testing.T) {
	outputSHA := strings.Repeat("b", 64)
	input := jobs.Input{
		ProjectID: "project", SubjectKind: jobs.SubjectPhysicalRender, SubjectID: "shot",
		ShotRevisionID: shotRevisionID("shot", 2), ShotRevision: 2,
		Bindings: []jobs.ArtifactBinding{{Role: "shot_snapshot", ArtifactID: shotRevisionID("shot", 2), RevisionID: "revision:2", SHA256: strings.Repeat("a", 64)}},
	}
	revision := shots.Revision{ShotID: "shot", Revision: 2, Snapshot: &shotsv1.ShotSnapshot{ReadabilityProof: &shotsv1.ReadabilityProof{RegionName: "title"}}}
	accepted := shots.ReadabilityReview{ProjectID: "project", ShotID: "shot", ShotRevision: 2, RenderID: "render", OutputSHA256: outputSHA, Verdict: int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED)}
	output := jobs.OutputRef{SHA256: outputSHA, Kind: "physical_still_png"}

	for _, tc := range []struct {
		name    string
		fixture qualificationReadabilityFixture
		valid   bool
	}{
		{name: "accepted exact review", fixture: qualificationReadabilityFixture{revision: revision, review: accepted}, valid: true},
		{name: "missing review", fixture: qualificationReadabilityFixture{revision: revision, reviewErr: shots.ErrReadabilityReviewNotFound}},
		{name: "rejected review", fixture: qualificationReadabilityFixture{revision: revision, review: func() shots.ReadabilityReview {
			r := accepted
			r.Verdict = int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_REJECTED)
			return r
		}()}},
		{name: "stale revision review", fixture: qualificationReadabilityFixture{revision: revision, review: func() shots.ReadabilityReview { r := accepted; r.ShotRevision = 1; return r }()}},
		{name: "stale output review", fixture: qualificationReadabilityFixture{revision: revision, review: func() shots.ReadabilityReview { r := accepted; r.OutputSHA256 = strings.Repeat("c", 64); return r }()}},
		{name: "undeclared proof remains independent", fixture: qualificationReadabilityFixture{revision: shots.Revision{ShotID: "shot", Revision: 2, Snapshot: &shotsv1.ShotSnapshot{}}, reviewErr: shots.ErrReadabilityReviewNotFound}, valid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Coordinator{shots: tc.fixture}
			err := c.requireDeclaredReadabilityReview(context.Background(), jobs.Job{ID: "render", Input: input}, output)
			if tc.valid && err != nil {
				t.Fatalf("valid qualification readability binding rejected: %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatal("missing, rejected, or stale declared proof was allowed to qualify")
			}
		})
	}

	legacy := jobs.Job{ID: "legacy-render", Input: jobs.Input{ProjectID: "project", SubjectKind: jobs.SubjectPhysicalRender}}
	if err := (&Coordinator{}).requireDeclaredReadabilityReview(context.Background(), legacy, output); err != nil {
		t.Fatalf("unbound legacy qualification unexpectedly required a shot proof: %v", err)
	}
	withoutReviewOwner := &Coordinator{shots: qualificationProofResolverOnly{revision: revision}}
	if err := withoutReviewOwner.requireDeclaredReadabilityReview(context.Background(), jobs.Job{ID: "render", Input: input}, output); err == nil {
		t.Fatal("declared proof was qualified without a readability review owner")
	}
}

func TestPhysicalStillWallTimeIsBoundedByExplicitBackend(t *testing.T) {
	for backend, want := range map[string]uint32{
		"CPU":   rendering.PhysicalStillCPUWallTimeSeconds,
		"CUDA":  rendering.PhysicalStillWallTimeSeconds,
		"OPTIX": rendering.PhysicalStillWallTimeSeconds,
	} {
		got, err := physicalStillWallTime(backend)
		if err != nil || got != want {
			t.Errorf("physicalStillWallTime(%q) = %d, %v; want %d", backend, got, err, want)
		}
	}
	if _, err := physicalStillWallTime("AUTO"); err == nil {
		t.Fatal("unspecified backend received a still wall-time profile")
	}
}

func TestPhysicalRecordingClosureUsesTheVerifiedLayoutIdentity(t *testing.T) {
	fill := presentations.RGB{R: 24, G: 26, B: 29}
	config := presentations.Config{Target: image.Pt(444, 965), Phone: &presentations.PhoneConfig{TopFill: &fill, BottomFill: &fill}}
	layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 1206, 2400), config)
	if err != nil {
		t.Fatal(err)
	}
	p := &shots.PhysicalResolution{
		PresentationProfile: presentations.SimulatedCompositorProfile,
		PresentationVersion: presentations.SemanticVersionV3,
		RecordingLayout:     &layout,
	}
	profile, version := physicalScreenPresentationIdentity(p, true)
	if profile != layout.ProfileID || version != layout.ProfileVersion {
		t.Fatalf("recording closure identity=(%q,%q), want verified layout (%q,%q)", profile, version, layout.ProfileID, layout.ProfileVersion)
	}
	profile, version = physicalScreenPresentationIdentity(p, false)
	if profile != presentations.SimulatedCompositorProfile || version != presentations.SemanticVersionV3 {
		t.Fatalf("still closure identity changed to (%q,%q)", profile, version)
	}
}

func TestBindPhysicalArticulationProfileSelectsV5OnlyForStaticMotionMP4(t *testing.T) {
	for _, tc := range []struct {
		name, profile string
		kind          string
		wantErr       bool
	}{
		{name: "motion-v1", profile: rendering.PhysicalMotionOutput, kind: "motion_mp4"},
		{name: "motion-v2", profile: rendering.PhysicalMotionOutputV2, kind: "motion_mp4"},
		{name: "still", profile: rendering.PhysicalStillProfile, kind: "still_png", wantErr: true},
		{name: "recording", profile: rendering.PhysicalRecordingMotionOutput, kind: "motion_mp4", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version := rendering.PhysicalMotionManifestVersion
			plan := rendering.PhysicalOutputPlan{ProfileID: tc.profile, Kind: tc.kind}
			err := bindPhysicalArticulationProfile(&plan, &version, true)
			if (err != nil) != tc.wantErr {
				t.Fatalf("articulation profile binding error=%v, wantErr=%t", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if version != rendering.PhysicalArticulatedManifestVersion || plan.ArticulationProfile != rendering.PhysicalArticulationProfile {
				t.Fatalf("articulated plan version/profile=(%d,%q), want (%d,%q)", version, plan.ArticulationProfile, rendering.PhysicalArticulatedManifestVersion, rendering.PhysicalArticulationProfile)
			}
		})
	}
	legacy := rendering.PhysicalOutputPlan{ProfileID: rendering.PhysicalMotionOutputV2, Kind: "motion_mp4"}
	version := rendering.PhysicalMotionManifestVersion
	if err := bindPhysicalArticulationProfile(&legacy, &version, false); err != nil || version != rendering.PhysicalMotionManifestVersion || legacy.ArticulationProfile != "" {
		t.Fatalf("unarticulated v2 output changed legacy manifest contract: version=%d output=%#v err=%v", version, legacy, err)
	}
}

func TestPhysicalClosureUsesDedicatedBoundForTenSecondSchedule(t *testing.T) {
	for _, tc := range []struct {
		name   string
		frames int
	}{{name: "240 frames at 24 fps", frames: 240}, {name: "300 frames at 30 fps", frames: 300}} {
		t.Run(tc.name, func(t *testing.T) {
			closure := physicalClosure{Version: physicalClosureVersion, Manifest: rendering.PhysicalRenderManifest{
				Output: rendering.PhysicalOutputPlan{Kind: "motion_mp4", Rate: composition.Rational{Num: int64(tc.frames / 10), Den: 1}, FrameCount: uint32(tc.frames), Samples: make([]rendering.PhysicalMotionSample, tc.frames)},
			}}
			for i := range closure.Manifest.Output.Samples {
				closure.Manifest.Output.Samples[i] = rendering.PhysicalMotionSample{
					OutputOrdinal: uint32(i), OutputTime: composition.Rational{Num: int64(i), Den: int64(tc.frames / 10)},
					RequestedMotionTick: composition.Rational{Num: int64(i), Den: 1}, SampledMotionTick: composition.Rational{Num: int64(i), Den: 1},
					DeviceRoot: &physical.TRS{Translation: physical.Vec3{X: float64(i) / 1000}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
				}
			}
			encoded, err := encodePhysicalClosure(closure)
			if err != nil {
				t.Fatalf("valid max-length closure exceeded its dedicated bound: %v", err)
			}
			if int64(len(encoded)) <= maxManifestBytes || int64(len(encoded))+1 > maxPhysicalClosureBytes {
				t.Fatalf("test closure size=%d; want > generic %d and newline-inclusive <= physical %d", len(encoded), maxManifestBytes, maxPhysicalClosureBytes)
			}
			if !bytes.Contains(encoded, []byte(`"frame_count":`+strconv.Itoa(tc.frames))) {
				t.Fatal("encoded closure lost its output frame count")
			}
		})
	}

	closure := physicalClosure{Version: physicalClosureVersion}
	closure.WorkerKey = strings.Repeat("x", int(maxPhysicalClosureBytes))
	if _, err := encodePhysicalClosure(closure); err == nil {
		t.Fatal("closure exceeding the explicit physical storage bound was accepted")
	}
}

func TestPhysicalBackendRequiresExplicitClosedSelection(t *testing.T) {
	for _, tc := range []struct {
		value shotsv1.PhysicalRenderBackend
		want  string
	}{{shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU, "CPU"}, {shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CUDA, "CUDA"}, {shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX, "OPTIX"}} {
		got, err := physicalBackendName(tc.value)
		if err != nil || got != tc.want {
			t.Fatalf("backend %v mapped to %q, %v; want %q", tc.value, got, err, tc.want)
		}
	}
	if _, err := physicalBackendName(shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_UNSPECIFIED); err == nil {
		t.Fatal("unspecified backend was silently accepted")
	}
}

func TestPhysicalRecordingAudioPolicyIsClosedAndNeverEnablesPassthrough(t *testing.T) {
	for _, tc := range []struct {
		policy shotsv1.VideoAudioPolicy
		want   string
	}{{shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_UNSPECIFIED, rendering.PhysicalRecordingAudioUnspecified}, {shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_SILENT, rendering.PhysicalRecordingAudioSilent}, {shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_PASSTHROUGH, rendering.PhysicalRecordingAudioPassthrough}} {
		got, err := physicalRecordingAudioPolicy(tc.policy)
		if err != nil || got != tc.want {
			t.Fatalf("audio policy %v mapped to %q, %v; want %q", tc.policy, got, err, tc.want)
		}
	}
	if _, err := physicalRecordingAudioPolicy(shotsv1.VideoAudioPolicy(99)); err == nil {
		t.Fatal("unknown audio policy was accepted")
	}
}

func TestPhysicalRecordingClosureRejectsVideoTimingRateMismatch(t *testing.T) {
	output := &shotsv1.PhysicalMotionOutputBinding{OutputRateNumerator: 30, OutputRateDenominator: 1}
	if !physicalRecordingRateMatches(&shotsv1.VideoTiming{OutputRateNum: 30, OutputRateDen: 1}, output) {
		t.Fatal("matching video/output rates were rejected")
	}
	if physicalRecordingRateMatches(&shotsv1.VideoTiming{OutputRateNum: 24, OutputRateDen: 1}, output) {
		t.Fatal("closure accepted a different timing/output rate authority")
	}
}

func TestPhysicalRecordingBindingsPinOriginalSourceAndInspectionArtifacts(t *testing.T) {
	digest := func(ch byte) string { return strings.Repeat(string(ch), 64) }
	closure := physicalClosure{
		RecordingSourceID:    "media-video",
		RecordingManifestKey: "renderjobs/inputs/manifest",
		RecordingReceiptKey:  "renderjobs/inputs/receipt",
		RecordingClockKey:    "renderjobs/inputs/clock",
		Manifest: rendering.PhysicalRenderManifest{Screen: rendering.PhysicalScreenInput{
			OriginalMediaID: "media-video",
			Recording: &rendering.PhysicalRecordingInput{Inspection: rendering.VideoInspection{
				Source: rendering.VideoBinding{SHA256: digest('a')}, Manifest: rendering.VideoBinding{SHA256: digest('b')},
				Receipt: rendering.VideoBinding{SHA256: digest('c')}, Clock: rendering.VideoBinding{SHA256: digest('d')},
			}},
		}},
	}
	input := jobs.Input{Bindings: []jobs.ArtifactBinding{
		{Role: "physical_recording_source", ArtifactID: closure.RecordingSourceID, RevisionID: "sha256:" + digest('a'), SHA256: digest('a')},
		{Role: "physical_recording_manifest", ArtifactID: closure.RecordingManifestKey, RevisionID: "sha256:" + digest('b'), SHA256: digest('b')},
		{Role: "physical_recording_receipt", ArtifactID: closure.RecordingReceiptKey, RevisionID: "sha256:" + digest('c'), SHA256: digest('c')},
		{Role: "physical_recording_clock", ArtifactID: closure.RecordingClockKey, RevisionID: "sha256:" + digest('d'), SHA256: digest('d')},
	}}
	if err := validatePhysicalRecordingBindings(input, closure); err != nil {
		t.Fatalf("valid original-source/inspection bindings rejected: %v", err)
	}
	for index := range input.Bindings {
		bad := jobs.Input{Bindings: append([]jobs.ArtifactBinding(nil), input.Bindings...)}
		bad.Bindings[index].SHA256 = digest('e')
		if err := validatePhysicalRecordingBindings(bad, closure); err == nil {
			t.Fatalf("tampered %s binding was accepted", input.Bindings[index].Role)
		}
	}
	closure.RecordingSourceID = "different-media"
	if err := validatePhysicalRecordingBindings(input, closure); err == nil {
		t.Fatal("recording source ID detached from original screen media was accepted")
	}
}

func TestPhysicalRecordingAddsBindingsWithoutReorderingLegacyStillInputs(t *testing.T) {
	digest := strings.Repeat("a", 64)
	closure := physicalClosure{ScreenKey: "screen-raster", RecordingSourceID: "video-source", RecordingManifestKey: "manifest-key", RecordingReceiptKey: "receipt-key", RecordingClockKey: "clock-key", Manifest: rendering.PhysicalRenderManifest{
		Screen: rendering.PhysicalScreenInput{Raster: rendering.ImageBinding{SHA256: digest}, Recording: &rendering.PhysicalRecordingInput{Inspection: rendering.VideoInspection{
			Source: rendering.VideoBinding{SHA256: digest}, Manifest: rendering.VideoBinding{SHA256: digest}, Receipt: rendering.VideoBinding{SHA256: digest}, Clock: rendering.VideoBinding{SHA256: digest},
		}}},
	}}
	roles := func(bindings []jobs.ArtifactBinding) []string {
		result := make([]string, len(bindings))
		for i := range bindings {
			result[i] = bindings[i].Role
		}
		return result
	}
	stillRoles := roles(physicalMediaRouteBindings(routePin{}, closure, false))
	if strings.Join(stillRoles, ",") != "physical_screen_raster,_internal_storage_route" {
		t.Fatalf("still role ordering changed: %v", stillRoles)
	}
	videoRoles := roles(physicalMediaRouteBindings(routePin{}, closure, true))
	want := "physical_recording_source,physical_recording_manifest,physical_recording_receipt,physical_recording_clock,_internal_storage_route"
	if strings.Join(videoRoles, ",") != want {
		t.Fatalf("recording roles=%v, want %s", videoRoles, want)
	}
}

func TestPhysicalIdentityBindingsMustMatchExactClosureCandidatesAndWorker(t *testing.T) {
	closure := physicalClosure{
		WorkerKey: "workers/sha/device-studio-api", WorkerSHA256: strings.Repeat("a", 64),
		Manifest: rendering.PhysicalRenderManifest{
			Device: rendering.PhysicalCandidateSnapshot{ID: "device-phone", Revision: 2, CanonicalSHA: strings.Repeat("b", 64)},
			Scene:  rendering.PhysicalCandidateSnapshot{ID: "scene-table", Revision: 3, CanonicalSHA: strings.Repeat("c", 64)},
			Motion: rendering.PhysicalCandidateSnapshot{ID: "motion-still", Revision: 1, CanonicalSHA: strings.Repeat("d", 64)},
		},
	}
	input := jobs.Input{Bindings: []jobs.ArtifactBinding{
		{Role: "worker_executable", ArtifactID: closure.WorkerKey, RevisionID: "sha256:" + closure.WorkerSHA256, SHA256: closure.WorkerSHA256},
		{Role: "device_revision", ArtifactID: "device-phone", RevisionID: "revision:2", SHA256: closure.Manifest.Device.CanonicalSHA},
		{Role: "scene_revision", ArtifactID: "scene-table", RevisionID: "revision:3", SHA256: closure.Manifest.Scene.CanonicalSHA},
		{Role: "motion_revision", ArtifactID: "motion-still", RevisionID: "revision:1", SHA256: closure.Manifest.Motion.CanonicalSHA},
	}}
	if err := validatePhysicalIdentityBindings(input, closure); err != nil {
		t.Fatalf("valid immutable identity bindings rejected: %v", err)
	}
	tests := []struct {
		name   string
		role   string
		mutate func(*jobs.ArtifactBinding)
	}{
		{"worker key", "worker_executable", func(b *jobs.ArtifactBinding) { b.ArtifactID = "workers/other/device-studio-api" }},
		{"device revision", "device_revision", func(b *jobs.ArtifactBinding) { b.RevisionID = "revision:1" }},
		{"scene digest", "scene_revision", func(b *jobs.ArtifactBinding) { b.SHA256 = closure.Manifest.Device.CanonicalSHA }},
		{"motion id", "motion_revision", func(b *jobs.ArtifactBinding) { b.ArtifactID = "motion-other" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bad := input
			bad.Bindings = append([]jobs.ArtifactBinding(nil), input.Bindings...)
			for i := range bad.Bindings {
				if bad.Bindings[i].Role == tc.role {
					tc.mutate(&bad.Bindings[i])
				}
			}
			if err := validatePhysicalIdentityBindings(bad, closure); err == nil {
				t.Fatal("tampered identity binding was accepted")
			}
		})
	}
}

func TestPhysicalMotionScheduleUsesExactRationalTicks(t *testing.T) {
	end := physical.Vec3{X: 36}
	candidate := motions.MotionCandidate{
		Version: motions.CandidateVersion, MotionID: "phone-yaw",
		Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 36,
		Tracks: []motions.MotionTrack{{Target: motions.TrackTarget{DeviceRoot: true}, Interpolation: motions.Linear, Keys: []motions.MotionKey{
			{Tick: 0, Transform: &motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}},
			{Tick: 36, Transform: &motions.TransformKey{Translation: end, Rotation: physical.QuatXYZW{W: 1}}},
		}}},
	}
	output := &shotsv1.PhysicalMotionOutputBinding{
		ProfileId: rendering.PhysicalMotionOutput, ProfileVersion: 1,
		Width: 1920, Height: 1080, OutputRateNumerator: 30, OutputRateDenominator: 1, FrameCount: 45,
	}
	samples, plan, err := physicalMotionSchedule(candidate, output)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != "motion_mp4" || plan.ProfileID != rendering.PhysicalMotionOutput || plan.Rate != (composition.Rational{Num: 30, Den: 1}) || len(samples) != 45 {
		t.Fatalf("motion output plan is not pinned to the requested closed profile: %#v", plan)
	}
	if samples[1].OutputTime != (composition.Rational{Num: 1, Den: 30}) || samples[1].RequestedMotionTick != (composition.Rational{Num: 4, Den: 5}) || samples[1].SampledMotionTick != (composition.Rational{Num: 4, Den: 5}) {
		t.Fatalf("30fps schedule lost exact 24-tick timebase: %#v", samples[1])
	}
	if samples[1].DeviceRoot == nil || samples[1].DeviceRoot.Translation.X < .799999999 || samples[1].DeviceRoot.Translation.X > .800000001 {
		t.Fatalf("scheduled pose does not use motions.SampleAt result: %#v", samples[1].DeviceRoot)
	}
	last := samples[len(samples)-1]
	if last.OutputOrdinal != 44 || last.RequestedMotionTick != (composition.Rational{Num: 176, Den: 5}) {
		t.Fatalf("last output frame has wrong exact time: %#v", last)
	}
}

func TestPhysicalMotionV2SchedulePinsTenSecondOutputProfile(t *testing.T) {
	candidate := motions.MotionCandidate{
		Version: motions.CandidateVersion, MotionID: "motion-v2",
		Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 240,
		Tracks: []motions.MotionTrack{{
			Target: motions.TrackTarget{DeviceRoot: true}, Interpolation: motions.Linear,
			Keys: []motions.MotionKey{
				{Tick: 0, Transform: &motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}},
				{Tick: 240, Transform: &motions.TransformKey{Rotation: physical.QuatXYZW{W: 1}}},
			},
		}},
	}
	output := &shotsv1.PhysicalMotionOutputBinding{ProfileId: rendering.PhysicalMotionOutputV2, ProfileVersion: 2, Width: 1920, Height: 1080, OutputRateNumerator: 24, OutputRateDenominator: 1, FrameCount: 240}
	samples, plan, err := physicalMotionSchedule(candidate, output)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 240 || plan.ProfileID != rendering.PhysicalMotionOutputV2 || plan.ProfileVersion != 2 || plan.MotionProfile != rendering.PhysicalMotionProfileV2 || plan.FrameCount != 240 {
		t.Fatalf("v2 schedule did not retain the ten-second output profile: profile=%+v samples=%d", plan, len(samples))
	}
	output.FrameCount = 241
	if _, _, err := physicalMotionSchedule(candidate, output); err == nil {
		t.Fatal("v2 schedule exceeded ten seconds")
	}
}

func TestPhysicalMotionBindingSelectsDistinctImmutableJobProfile(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		version  uint32
		want     string
	}{
		{"v1", rendering.PhysicalMotionOutput, 1, OutputProfilePhysicalMotionMP4},
		{"v2", rendering.PhysicalMotionOutputV2, 2, OutputProfilePhysicalMotionMP4V2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := physicalMotionJobProfile(&shotsv1.PhysicalMotionOutputBinding{ProfileId: tc.id, ProfileVersion: tc.version})
			if err != nil || got != tc.want {
				t.Fatalf("profile=%q err=%v, want immutable job profile %q", got, err, tc.want)
			}
		})
	}
	if _, err := physicalMotionJobProfile(&shotsv1.PhysicalMotionOutputBinding{ProfileId: rendering.PhysicalMotionOutputV2, ProfileVersion: 1}); err == nil {
		t.Fatal("mismatched v2 profile version was accepted")
	}
}
