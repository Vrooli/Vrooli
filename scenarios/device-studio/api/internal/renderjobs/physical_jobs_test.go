package renderjobs

import (
	"bytes"
	"image"
	"strconv"
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/jobs"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/presentations"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

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
