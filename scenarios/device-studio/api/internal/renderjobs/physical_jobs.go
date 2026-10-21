package renderjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

const (
	physicalClosureVersion    = 1
	physicalWorkspaceInputMax = 512 << 20
)

type physicalClosure struct {
	Version              int                              `json:"version"`
	ProjectID            string                           `json:"project_id"`
	ShotID               string                           `json:"shot_id"`
	Revision             int64                            `json:"revision"`
	RecipeSHA256         string                           `json:"recipe_sha256"`
	WorkerKey            string                           `json:"worker_key"`
	WorkerSHA256         string                           `json:"worker_sha256"`
	AdapterSHA256        string                           `json:"adapter_sha256"`
	ScreenKey            string                           `json:"screen_key"`
	RecordingSourceID    string                           `json:"recording_source_id,omitempty"`
	RecordingManifestKey string                           `json:"recording_manifest_key,omitempty"`
	RecordingReceiptKey  string                           `json:"recording_receipt_key,omitempty"`
	RecordingClockKey    string                           `json:"recording_clock_key,omitempty"`
	Manifest             rendering.PhysicalRenderManifest `json:"manifest"`
}

func (c *Coordinator) submitPhysical(ctx context.Context, route routePin, req SubmitRequest, resolved shots.ResolvedShot) (jobs.Job, bool, error) {
	if c.physicalArtifacts == nil || c.physicalProfile == nil || c.physicalAdapter == nil || resolved.Revision.Physical == nil {
		return jobs.Job{}, false, errors.New("renderjobs: physical still owners are unavailable")
	}
	outputProfile := OutputProfilePNG
	recording := resolved.Revision.Snapshot.GetVideoTiming() != nil
	articulated := physicalMotionHasJointTracks(resolved.Revision.Physical.MotionCandidate)
	if articulated && req.OutputProfile != OutputProfileMP4 {
		return jobs.Job{}, false, fmt.Errorf("%w: articulated physical motion currently requires the final static-screen MP4 output", ErrInvalidRequest)
	}
	if recording {
		if articulated {
			return jobs.Job{}, false, fmt.Errorf("%w: articulated recording textures are not supported yet", ErrInvalidRequest)
		}
		if req.OutputProfile == OutputProfilePhysicalPreviewPNG {
			return jobs.Job{}, false, fmt.Errorf("%w: recording sources need a selected-frame preview profile; submit final MP4 only", ErrInvalidRequest)
		}
		if req.OutputProfile != OutputProfileMP4 || resolved.Revision.Physical.Binding.GetMotionOutput() == nil {
			return jobs.Job{}, false, fmt.Errorf("%w: physical recording sources require the pinned final MP4 schedule", ErrInvalidRequest)
		}
		outputProfile = OutputProfilePhysicalRecordingMP4
	} else if req.OutputProfile == OutputProfilePhysicalPreviewPNG {
		outputProfile = OutputProfilePhysicalPreviewPNG
	} else if resolved.Revision.Physical.Binding.GetMotionOutput() != nil {
		if req.OutputProfile != OutputProfileMP4 {
			return jobs.Job{}, false, fmt.Errorf("%w: physical motion shots require the pinned MP4 output profile", ErrInvalidRequest)
		}
		selectedProfile, profileErr := physicalMotionJobProfile(resolved.Revision.Physical.Binding.GetMotionOutput())
		if profileErr != nil {
			return jobs.Job{}, false, profileErr
		}
		outputProfile = selectedProfile
	} else if req.OutputProfile != OutputProfilePNG {
		return jobs.Job{}, false, fmt.Errorf("%w: this physical shot has no motion output binding", ErrInvalidRequest)
	}
	spanStarted := time.Now()
	workerBytes, _, workerSHA, err := c.readCurrentExecutable()
	if err != nil {
		return jobs.Job{}, false, err
	}
	workerKey := workerBlobPrefix + workerSHA + "/device-studio-api"
	if err := c.putVerified(ctx, route, workerKey, workerBytes, "application/octet-stream", workerSHA); err != nil {
		return jobs.Job{}, false, fmt.Errorf("pin physical worker: %w", err)
	}
	submitTimingSpan(ctx, "worker_pin", spanStarted)
	spanStarted = time.Now()
	_, toolProfile, err := c.physicalProfile(ctx)
	if err != nil {
		return jobs.Job{}, false, fmt.Errorf("resolve pinned Blender profile: %w", err)
	}
	submitTimingSpan(ctx, "blender_profile", spanStarted)
	if toolProfile == nil {
		return jobs.Job{}, false, errors.New("renderjobs: Blender profile response is empty")
	}
	adapterBytes := c.physicalAdapter()
	if len(adapterBytes) == 0 || len(adapterBytes) > 1<<20 {
		return jobs.Job{}, false, errors.New("renderjobs: embedded Blender adapter is empty or outside bounds")
	}
	spanStarted = time.Now()
	closure, err := c.makePhysicalClosure(ctx, route, req, resolved, workerKey, workerSHA, toolProfile, adapterBytes)
	if err != nil {
		return jobs.Job{}, false, err
	}
	closureBytes, err := encodePhysicalClosure(closure)
	if err != nil {
		return jobs.Job{}, false, fmt.Errorf("renderjobs: physical closure exceeds its bounded storage size: %w", err)
	}
	closureBytes = append(closureBytes, '\n')
	closureSHA := digestBytes(closureBytes)
	closureKey := closureBlobPrefix + closureSHA + ".json"
	if err := c.putVerified(ctx, route, closureKey, closureBytes, "application/json", closureSHA); err != nil {
		return jobs.Job{}, false, fmt.Errorf("pin physical closure: %w", err)
	}
	submitTimingSpan(ctx, "closure", spanStarted)
	input := jobs.Input{
		ProjectID: req.ProjectID, ShotRevisionID: shotRevisionID(req.ShotID, req.Revision), ShotRevision: req.Revision,
		ToolProfileDigest: closure.Manifest.Profile.DigestSHA256, RendererVersion: closure.Manifest.Profile.ID,
		OutputProfile: outputProfile, SubjectKind: jobs.SubjectPhysicalRender, SubjectID: req.ShotID,
		Bindings: []jobs.ArtifactBinding{
			{Role: "shot_snapshot", ArtifactID: "shot:" + req.ShotID + ":revision:" + strconv.FormatInt(req.Revision, 10), RevisionID: "revision:" + strconv.FormatInt(req.Revision, 10), SHA256: resolved.Revision.RecipeSHA256},
			{Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerSHA, SHA256: workerSHA},
			{Role: "physical_closure", ArtifactID: closureKey, RevisionID: "sha256:" + closureSHA, SHA256: closureSHA},
			{Role: "physical_adapter", ArtifactID: "embedded-adapter:sha256:" + closure.AdapterSHA256, RevisionID: "sha256:" + closure.AdapterSHA256, SHA256: closure.AdapterSHA256},
		},
	}
	input.Bindings = append(input.Bindings, physicalMediaRouteBindings(route, closure, recording)...)
	for _, candidate := range []struct {
		role, id string
		revision uint32
		sha      string
	}{
		{"device_revision", closure.Manifest.Device.ID, closure.Manifest.Device.Revision, closure.Manifest.Device.CanonicalSHA},
		{"scene_revision", closure.Manifest.Scene.ID, closure.Manifest.Scene.Revision, closure.Manifest.Scene.CanonicalSHA},
		{"motion_revision", closure.Manifest.Motion.ID, closure.Manifest.Motion.Revision, closure.Manifest.Motion.CanonicalSHA},
	} {
		input.Bindings = append(input.Bindings, jobs.ArtifactBinding{Role: candidate.role, ArtifactID: candidate.id, RevisionID: "revision:" + strconv.FormatUint(uint64(candidate.revision), 10), SHA256: candidate.sha})
	}
	for _, asset := range closure.Manifest.Assets {
		id := physicalArtifactIdentity(asset.Reference)
		input.Bindings = append(input.Bindings, jobs.ArtifactBinding{Role: "physical_asset:" + asset.BindingID, ArtifactID: id, RevisionID: "sha256:" + asset.Reference.SHA256, SHA256: asset.Reference.SHA256})
	}
	spanStarted = time.Now()
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: req.IdempotencyKey, Input: input})
	submitTimingSpan(ctx, "jobs_submit", spanStarted)
	if err != nil {
		if errors.Is(err, jobs.ErrIdempotencyConflict) {
			if existing, found, lookupErr := c.findIdempotentSubmit(ctx, route, req); lookupErr != nil || found {
				return existing, false, lookupErr
			}
		}
		return jobs.Job{}, false, err
	}
	return job, created, nil
}

func physicalMediaRouteBindings(route routePin, closure physicalClosure, recording bool) []jobs.ArtifactBinding {
	if recording {
		i := closure.Manifest.Screen.Recording.Inspection
		return []jobs.ArtifactBinding{
			{Role: "physical_recording_source", ArtifactID: closure.RecordingSourceID, RevisionID: "sha256:" + i.Source.SHA256, SHA256: i.Source.SHA256},
			{Role: "physical_recording_manifest", ArtifactID: closure.RecordingManifestKey, RevisionID: "sha256:" + i.Manifest.SHA256, SHA256: i.Manifest.SHA256},
			{Role: "physical_recording_receipt", ArtifactID: closure.RecordingReceiptKey, RevisionID: "sha256:" + i.Receipt.SHA256, SHA256: i.Receipt.SHA256},
			{Role: "physical_recording_clock", ArtifactID: closure.RecordingClockKey, RevisionID: "sha256:" + i.Clock.SHA256, SHA256: i.Clock.SHA256},
			routeBinding(route),
		}
	}
	// The raster precedes the route fence to preserve the historical still
	// input fingerprint and idempotency key across this additive recording path.
	return []jobs.ArtifactBinding{
		{Role: "physical_screen_raster", ArtifactID: closure.ScreenKey, RevisionID: "sha256:" + closure.Manifest.Screen.Raster.SHA256, SHA256: closure.Manifest.Screen.Raster.SHA256},
		routeBinding(route),
	}
}

func encodePhysicalClosure(closure physicalClosure) ([]byte, error) {
	data, err := json.Marshal(closure)
	if err != nil {
		return nil, err
	}
	// submitPhysical appends one canonical newline before hashing and storing.
	if len(data) == 0 || int64(len(data))+1 > maxPhysicalClosureBytes {
		return nil, errors.New("physical closure exceeds its bounded storage size")
	}
	return data, nil
}

func (c *Coordinator) makePhysicalClosure(ctx context.Context, route routePin, req SubmitRequest, resolved shots.ResolvedShot, workerKey, workerSHA string, toolProfile *cliv1.BlenderToolProfile, adapterBytes []byte) (physicalClosure, error) {
	p := resolved.Revision.Physical
	if p == nil || p.Binding == nil || p.ScreenMedia.ID == "" || p.ScreenMedia.SHA256 == "" {
		return physicalClosure{}, errors.New("renderjobs: physical shot lacks pinned candidate or screen raster facts")
	}
	recording := resolved.Revision.Snapshot.GetVideoTiming() != nil
	if recording {
		if len(p.ScreenRasterPNG) != 0 || p.RecordingConfig == nil || p.RecordingLayout == nil || p.Binding.GetMotionOutput() == nil || req.OutputProfile != OutputProfileMP4 {
			return physicalClosure{}, fmt.Errorf("%w: physical recording closure requires a pinned final MP4 schedule and verified layout", ErrInvalidRequest)
		}
		if !physicalRecordingRateMatches(resolved.Revision.Snapshot.GetVideoTiming(), p.Binding.GetMotionOutput()) {
			return physicalClosure{}, fmt.Errorf("%w: recording output rate does not match the pinned physical motion output rate", ErrInvalidRequest)
		}
	} else if len(p.ScreenRasterPNG) == 0 || digestBytes(p.ScreenRasterPNG) != p.RasterSHA256 {
		return physicalClosure{}, errors.New("renderjobs: physical shot lacks pinned screen raster facts")
	}
	motionOutput := p.Binding.GetMotionOutput()
	articulated := physicalMotionHasJointTracks(p.MotionCandidate)
	if articulated && (recording || motionOutput == nil || req.OutputProfile != OutputProfileMP4) {
		return physicalClosure{}, fmt.Errorf("%w: articulated physical rendering requires a static-screen MP4 motion; recording, still, and preview modes are unsupported", ErrInvalidRequest)
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return physicalClosure{}, err
	}
	adapterSum := digestBytes(adapterBytes)
	screenKey := ""
	if !recording {
		screenKey = inputBlobPrefix + p.RasterSHA256 + "/physical-screen.png"
		if err := c.putVerified(ctx, route, screenKey, p.ScreenRasterPNG, "image/png", p.RasterSHA256); err != nil {
			return physicalClosure{}, err
		}
	}
	blender := toolProfile.GetBlender()
	if blender == nil {
		return physicalClosure{}, errors.New("renderjobs: managed Blender profile has no executable facts")
	}
	backend, err := physicalBackendName(p.Binding.GetBackend())
	if err != nil {
		return physicalClosure{}, err
	}
	wallTime := rendering.PhysicalStillWallTimeSeconds
	sampleCount := rendering.PhysicalStillSamples
	preview := req.OutputProfile == OutputProfilePhysicalPreviewPNG
	if recording {
		wallTime = rendering.PhysicalRecordingWallTimeSeconds
		sampleCount = rendering.PhysicalMotionSamples
	} else if preview {
		wallTime = rendering.PhysicalPreviewWallTimeSeconds
		sampleCount = rendering.PhysicalPreviewSamples
	} else if motionOutput != nil {
		wallTime = rendering.PhysicalMotionWallTimeSeconds
		sampleCount = rendering.PhysicalMotionSamples
		if motionOutput.GetProfileId() == rendering.PhysicalMotionOutputV2 {
			wallTime = rendering.PhysicalMotionV2WallTimeSeconds
		}
	}
	renderProfile := rendering.PhysicalRenderProfile{
		ID: rendering.PhysicalRendererProfile, Version: 1, BlenderArtifactID: toolProfile.GetArtifactIdentity(),
		BlenderTreeSHA256: toolProfile.GetExpectedTreeSha256(), BlenderBinarySHA256: blender.GetSha256(), BlenderVersion: blender.GetVersion(), AdapterSHA256: adapterSum,
		Backend: backend, Engine: "CYCLES", Samples: sampleCount, Seed: 1307, ColorManagement: rendering.PhysicalColorProfile,
		ColorView: "Standard", ColorLook: "None", Exposure: "0", Gamma: "1",
		WallTimeSeconds: wallTime,
		MemoryMaxBytes:  rendering.PhysicalOperationMemoryMaxBytes,
		TasksMax:        rendering.PhysicalOperationTasksMax,
		CPUWeight:       rendering.PhysicalOperationCPUWeight,
		Threads:         rendering.PhysicalOperationThreads,
	}
	profileDigest, err := rendering.PhysicalRenderProfileDigest(renderProfile)
	if err != nil {
		return physicalClosure{}, err
	}
	renderProfile.DigestSHA256 = profileDigest
	assets := make([]rendering.PhysicalAssetInput, 0, len(p.Assets))
	deviceGLB := []byte(nil)
	geometryByBinding := make(map[string][]byte)
	for i, asset := range p.Assets {
		name := physicalAssetName(i, asset.Reference.Role)
		assets = append(assets, rendering.PhysicalAssetInput{Role: asset.Role, BindingID: asset.BindingID, Reference: asset.Reference, Name: name})
		if asset.Role == "device_geometry" || asset.Role == "scene_geometry" {
			data, readErr := c.readPhysicalArtifact(ctx, asset.Reference)
			if readErr != nil {
				return physicalClosure{}, fmt.Errorf("verify %s before render submit: %w", asset.Role, readErr)
			}
			if asset.Role == "device_geometry" {
				// The device candidate has one geometry binding. Keep its bytes
				// separately because scene geometry entries can follow it.
				deviceGLB = data
			} else {
				geometryByBinding[asset.BindingID] = data
			}
		}
	}
	if len(deviceGLB) == 0 {
		return physicalClosure{}, errors.New("renderjobs: physical device geometry binding is missing")
	}
	mapping, err := rendering.ResolvePhysicalScreenGeometry(deviceGLB, p.DeviceCandidate, p.Binding.GetDisplayId())
	if err != nil {
		return physicalClosure{}, fmt.Errorf("resolve physical screen geometry: %w", err)
	}
	presentationProfile, presentationVersion := physicalScreenPresentationIdentity(p, recording)
	screenInput := rendering.PhysicalScreenInput{
		DisplayID: p.Binding.GetDisplayId(), OriginalMediaID: p.ScreenMedia.ID, OriginalMediaSHA256: p.ScreenMedia.SHA256,
		PresentationProfile: presentationProfile, PresentationVersion: presentationVersion,
		ConfigSHA256: p.ConfigSHA256, LayoutSHA256: p.LayoutSHA256, RasterSHA256: p.RasterSHA256,
		Width: int(p.Binding.GetScreenWidthPixels()), Height: int(p.Binding.GetScreenHeightPixels()), UVSet: "TEXCOORD_0",
		MappingProfile: rendering.PhysicalScreenMappingProfile, PrimitiveIndex: mapping.PrimitiveIndex, MaterialSlot: mapping.MaterialIndex,
	}
	if !recording {
		screenInput.Raster = rendering.ImageBinding{Name: "screen.png", SHA256: p.RasterSHA256, SizeBytes: int64(len(p.ScreenRasterPNG))}
	}
	deviceInventory, err := rendering.ValidateDevicePhysicalGLB(deviceGLB, p.DeviceCandidate, screenInput)
	if err != nil {
		return physicalClosure{}, fmt.Errorf("validate complete device geometry before render submit: %w", err)
	}
	for _, binding := range p.SceneCandidate.Artifacts {
		if binding.Ref.Role != library.RoleGeometryGLB {
			continue
		}
		data := geometryByBinding[binding.ID]
		if len(data) == 0 {
			return physicalClosure{}, fmt.Errorf("scene geometry binding %q was not read before render submit", binding.ID)
		}
		candidate := p.SceneCandidate
		candidate.Artifacts = []scenes.SceneArtifactBinding{binding}
		if _, err := rendering.ValidateScenePhysicalGLB(data, candidate); err != nil {
			return physicalClosure{}, fmt.Errorf("validate scene geometry binding %q before render submit: %w", binding.ID, err)
		}
	}
	sceneWorld, err := physicalTRSFromProto(p.Binding.GetSceneSupportWorld())
	if err != nil {
		return physicalClosure{}, err
	}
	deviceRest, err := physicalTRSFromProto(p.Binding.GetDeviceRootRest())
	if err != nil {
		return physicalClosure{}, err
	}
	pivot := p.Binding.GetSupportPivotM()
	requested := p.Binding.GetMotionSampleTick()
	manifestVersion := rendering.PhysicalManifestVersion
	output := rendering.PhysicalOutputPlan{ProfileID: rendering.PhysicalStillProfile, Kind: "still_png", Width: p.OutputWidth, Height: p.OutputHeight, Rate: composition.Rational{Num: 1, Den: 1}, FrameCount: 1, MotionTicks: 0, Timebase: p.MotionCandidate.Timebase, FinalPoseHold: "none", OutputName: rendering.PhysicalStillOutputName, Codec: "png", PixelFormat: "rgba8"}
	var scheduled []rendering.PhysicalMotionSample
	var recordingArtifacts acceptedInspectionArtifacts
	var recordingSource projects.SourceMedia
	var recordingManifestKey, recordingReceiptKey, recordingClockKey string
	if motionOutput == nil {
		if requested == nil {
			return physicalClosure{}, errors.New("renderjobs: physical still has no exact motion sample tick")
		}
		sample, sampleErr := motions.SampleAt(p.MotionCandidate, composition.Rational{Num: requested.GetNumerator(), Den: requested.GetDenominator()})
		if sampleErr != nil {
			return physicalClosure{}, fmt.Errorf("sample physical still pose: %w", sampleErr)
		}
		scheduled = []rendering.PhysicalMotionSample{physicalRenderSample(requested, sample)}
	} else {
		var scheduleErr error
		scheduled, output, scheduleErr = physicalMotionSchedule(p.MotionCandidate, motionOutput)
		if scheduleErr != nil {
			return physicalClosure{}, scheduleErr
		}
		manifestVersion = rendering.PhysicalMotionManifestVersion
	}
	if err := bindPhysicalArticulationProfile(&output, &manifestVersion, articulated); err != nil {
		return physicalClosure{}, err
	}
	if recording {
		source, sourceErr := c.media.Get(ctx, req.ProjectID, p.ScreenMedia.ID)
		if sourceErr != nil {
			return physicalClosure{}, sourceErr
		}
		if source.Kind != projects.MediaKindVideo || source.SHA256 != p.ScreenMedia.SHA256 || source.SizeBytes != p.ScreenMedia.SizeBytes {
			return physicalClosure{}, errors.New("renderjobs: physical recording source differs from the compiled shot revision")
		}
		recordingSource = source
		recordingArtifacts, err = c.reconstructAcceptedInspection(ctx, route, req.ProjectID, source)
		if err != nil {
			return physicalClosure{}, fmt.Errorf("reconstruct original accepted inspection: %w", err)
		}
		timing := resolved.Revision.Snapshot.GetVideoTiming()
		audioPolicy, policyErr := physicalRecordingAudioPolicy(timing.GetAudioPolicy())
		if policyErr != nil {
			return physicalClosure{}, policyErr
		}
		receipt := recordingArtifacts.Receipt
		if receipt.VideoStreamIndex == nil || receipt.StreamStartPTS == nil || receipt.SourceEndPTS == nil {
			return physicalClosure{}, errors.New("renderjobs: accepted inspection lacks required video clock facts")
		}
		inspection := rendering.VideoInspection{
			Source:    rendering.VideoBinding{Name: rendering.InspectionSourceName, SHA256: source.SHA256, SizeBytes: source.SizeBytes},
			Manifest:  rendering.VideoBinding{Name: rendering.AcceptedInspectionManifestName, SHA256: digestBytes(recordingArtifacts.ManifestBytes), SizeBytes: int64(len(recordingArtifacts.ManifestBytes))},
			Receipt:   rendering.VideoBinding{Name: rendering.InspectionReceiptName, SHA256: digestBytes(recordingArtifacts.ReceiptBytes), SizeBytes: int64(len(recordingArtifacts.ReceiptBytes))},
			Clock:     rendering.VideoBinding{Name: rendering.InspectionClockName, SHA256: digestBytes(recordingArtifacts.ClockBytes), SizeBytes: int64(len(recordingArtifacts.ClockBytes))},
			ProfileID: receipt.ProfileID, ProfileDigest: receipt.ProfileDigest, ProtocolVersion: receipt.ProtocolVersion,
			SelectedStream: uint32(*receipt.VideoStreamIndex), TimeBase: composition.Rational{Num: receipt.TimeBaseNum, Den: receipt.TimeBaseDen},
			StreamStartPTS: *receipt.StreamStartPTS, SourceEndPTS: *receipt.SourceEndPTS, LastFrameDurationTicks: receipt.LastFrameDurationTicks,
			EOFPolicy: receipt.EOFPolicy, FrameCount: receipt.FrameCount, Width: int(receipt.DisplayWidth), Height: int(receipt.DisplayHeight),
			ColorPrimaries: receipt.ColorPrimaries, ColorTransfer: receipt.ColorTransfer, ColorMatrix: receipt.ColorMatrix, ColorRange: receipt.ColorRange, ColorPolicy: receipt.ColorPolicy,
		}
		recordingInput := rendering.PhysicalRecordingInput{
			Inspection: inspection, TrimStartPTS: timing.GetTrimStartPts(), TrimEndPTS: timing.GetTrimEndPts(),
			PlaybackRate: composition.Rational{Num: int64(timing.GetPlaybackRateNum()), Den: int64(timing.GetPlaybackRateDen())},
			AudioPolicy:  audioPolicy, Presentation: *p.RecordingConfig, PresentationLayout: *p.RecordingLayout,
		}
		facts, deriveErr := rendering.DerivePhysicalRecordingSchedule(ctx, recordingInput, output.Rate, output.FrameCount, recordingArtifacts.ManifestBytes, recordingArtifacts.ReceiptBytes, recordingArtifacts.ClockBytes)
		if deriveErr != nil {
			return physicalClosure{}, fmt.Errorf("derive physical recording schedule before claim: %w", deriveErr)
		}
		recordingInput.SourcePrefixFrameCount = facts.RequiredSourcePrefixFrameCount
		screenInput.Recording = &recordingInput
		output.ProfileID, output.ProfileVersion, output.MotionProfile = rendering.PhysicalRecordingMotionOutput, 1, rendering.PhysicalRecordingMotionProfile
		output.VideoToolProfileID, output.VideoToolArtifactID = rendering.PhysicalMotionVideoProfileID, rendering.PhysicalMotionVideoArtifactID
		manifestVersion = rendering.PhysicalRecordingManifestVersion
		pinSidecar := func(name string, data []byte) (string, error) {
			sha := digestBytes(data)
			key := inputBlobPrefix + sha + "/" + name
			if err := c.putVerified(ctx, route, key, data, "application/octet-stream", sha); err != nil {
				return "", err
			}
			return key, nil
		}
		recordingManifestKey, err = pinSidecar(rendering.AcceptedInspectionManifestName, recordingArtifacts.ManifestBytes)
		if err == nil {
			recordingReceiptKey, err = pinSidecar(rendering.InspectionReceiptName, recordingArtifacts.ReceiptBytes)
		}
		if err == nil {
			recordingClockKey, err = pinSidecar(rendering.InspectionClockName, recordingArtifacts.ClockBytes)
		}
		if err != nil {
			return physicalClosure{}, fmt.Errorf("pin accepted recording inspection artifacts: %w", err)
		}
	}
	if preview {
		if len(scheduled) == 0 {
			return physicalClosure{}, errors.New("renderjobs: physical preview has no exact saved pose sample")
		}
		output = rendering.NewPhysicalPreviewOutputPlan(scheduled[0], p.MotionCandidate.Timebase)
		scheduled = output.Samples
		manifestVersion = rendering.PhysicalPreviewManifestVersion
	}
	if articulated {
		if _, err := rendering.ValidatePhysicalArticulation(p.DeviceCandidate, p.MotionCandidate, scheduled, deviceInventory); err != nil {
			return physicalClosure{}, fmt.Errorf("validate physical articulation before render submit: %w", err)
		}
	}
	manifest := rendering.PhysicalRenderManifest{
		Version: manifestVersion, WorkerSHA256: workerSHA,
		Device:    rendering.PhysicalCandidateSnapshot{ID: p.DeviceID, Revision: p.DeviceRevision, CanonicalSHA: p.DeviceSHA256, CanonicalJSON: bytes.Clone(p.DeviceCanonicalJSON)},
		Scene:     rendering.PhysicalCandidateSnapshot{ID: p.SceneID, Revision: p.SceneRevision, CanonicalSHA: p.SceneSHA256, CanonicalJSON: bytes.Clone(p.SceneCanonicalJSON)},
		Motion:    rendering.PhysicalCandidateSnapshot{ID: p.MotionID, Revision: p.MotionRevision, CanonicalSHA: p.MotionSHA256, CanonicalJSON: bytes.Clone(p.MotionCanonicalJSON)},
		Assets:    assets,
		Screen:    screenInput,
		Placement: rendering.PhysicalPlacement{DeviceSupportID: p.Binding.GetDeviceSupportId(), SceneSupportID: p.Binding.GetSceneSupportId(), SelectedCameraID: p.Binding.GetCameraId(), SceneSupportWorld: sceneWorld, SupportPivot: physical.Vec3{X: pivot.GetX(), Y: pivot.GetY(), Z: pivot.GetZ()}, DeviceRootRest: deviceRest},
		Profile:   renderProfile,
		Output:    output,
	}
	manifest.Output.Samples = scheduled
	preflight := manifest
	preflight.JobID, preflight.AttemptID = "preflight-job", "preflight-attempt"
	preflight.InputFingerprint = strings.Repeat("0", 64)
	if err := rendering.ValidatePhysicalRenderManifest(preflight); err != nil {
		return physicalClosure{}, fmt.Errorf("physical render closure is not worker-valid: %w", err)
	}
	return physicalClosure{Version: physicalClosureVersion, ProjectID: req.ProjectID, ShotID: req.ShotID, Revision: req.Revision, RecipeSHA256: resolved.Revision.RecipeSHA256, WorkerKey: workerKey, WorkerSHA256: workerSHA, AdapterSHA256: adapterSum, ScreenKey: screenKey, RecordingSourceID: recordingSource.ID, RecordingManifestKey: recordingManifestKey, RecordingReceiptKey: recordingReceiptKey, RecordingClockKey: recordingClockKey, Manifest: manifest}, nil
}

func physicalScreenPresentationIdentity(p *shots.PhysicalResolution, recording bool) (string, string) {
	if recording && p.RecordingLayout != nil {
		return p.RecordingLayout.ProfileID, p.RecordingLayout.ProfileVersion
	}
	return p.PresentationProfile, p.PresentationVersion
}

func physicalMotionHasJointTracks(motion motions.MotionCandidate) bool {
	for _, track := range motion.Tracks {
		if track.Target.JointID != "" {
			return true
		}
	}
	return false
}

func bindPhysicalArticulationProfile(output *rendering.PhysicalOutputPlan, manifestVersion *int, articulated bool) error {
	if !articulated {
		return nil
	}
	if output == nil || manifestVersion == nil || output.Kind != "motion_mp4" ||
		(output.ProfileID != rendering.PhysicalMotionOutput && output.ProfileID != rendering.PhysicalMotionOutputV2) {
		return fmt.Errorf("%w: articulation is supported only for static-screen physical motion MP4 profiles", ErrInvalidRequest)
	}
	output.ArticulationProfile = rendering.PhysicalArticulationProfile
	*manifestVersion = rendering.PhysicalArticulatedManifestVersion
	return nil
}

func physicalRecordingRateMatches(timing *shotsv1.VideoTiming, output *shotsv1.PhysicalMotionOutputBinding) bool {
	return timing != nil && output != nil && timing.GetOutputRateNum() == output.GetOutputRateNumerator() && timing.GetOutputRateDen() == output.GetOutputRateDenominator()
}

func physicalRecordingAudioPolicy(policy shotsv1.VideoAudioPolicy) (string, error) {
	switch policy {
	case shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_UNSPECIFIED:
		return rendering.PhysicalRecordingAudioUnspecified, nil
	case shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_SILENT:
		return rendering.PhysicalRecordingAudioSilent, nil
	case shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_PASSTHROUGH:
		return rendering.PhysicalRecordingAudioPassthrough, nil
	default:
		return "", fmt.Errorf("%w: physical recording audio policy is unknown", ErrInvalidRequest)
	}
}

func (c *Coordinator) readPhysicalArtifact(ctx context.Context, ref physical.ArtifactRef) ([]byte, error) {
	if c.physicalArtifacts == nil {
		return nil, errors.New("reviewed physical asset owner is unavailable")
	}
	if ref.SizeBytes <= 0 || ref.SizeBytes > library.MaxGeometryBytes {
		return nil, errors.New("physical geometry input is outside the library byte bound")
	}
	resolved, err := c.physicalArtifacts.Resolve(ctx, ref, library.UseRender)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(resolved.Reader, ref.SizeBytes+1))
	closeErr := resolved.Reader.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(data)) != ref.SizeBytes || digestBytes(data) != ref.SHA256 {
		return nil, errors.New("physical geometry bytes differ from reviewed artifact digest")
	}
	return data, nil
}

func physicalAssetName(index int, role library.ArtifactRole) string {
	ext := ".bin"
	if role == library.RoleGeometryGLB {
		ext = ".glb"
	}
	if role == library.RoleTextureImage {
		ext = ".png"
	}
	return fmt.Sprintf("physical-asset-%02d%s", index, ext)
}

func physicalArtifactIdentity(ref physical.ArtifactRef) string {
	return fmt.Sprintf("bundle:%s:revision:%d:artifact:%s", ref.BundleID, ref.BundleRevision, ref.ArtifactID)
}

func physicalBackendName(value shotsv1.PhysicalRenderBackend) (string, error) {
	switch value {
	case shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU:
		return "CPU", nil
	case shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CUDA:
		return "CUDA", nil
	case shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX:
		return "OPTIX", nil
	default:
		return "", errors.New("renderjobs: physical render backend must be selected explicitly")
	}
}

func physicalTRSFromProto(value *physicalv1.TRS) (physical.TRS, error) {
	if value == nil || value.TranslationM == nil || value.RotationXyzw == nil || value.Scale == nil {
		return physical.TRS{}, errors.New("physical placement transform is incomplete")
	}
	return physical.CanonicalizeTRS(physical.TRS{Translation: physical.Vec3{X: value.TranslationM.X, Y: value.TranslationM.Y, Z: value.TranslationM.Z}, Rotation: physical.QuatXYZW{X: value.RotationXyzw.X, Y: value.RotationXyzw.Y, Z: value.RotationXyzw.Z, W: value.RotationXyzw.W}, Scale: physical.Vec3{X: value.Scale.X, Y: value.Scale.Y, Z: value.Scale.Z}})
}

func physicalRenderSample(requested *shotsv1.RationalTime, sample motions.MotionSample) rendering.PhysicalMotionSample {
	requestedTick := composition.Rational{Num: requested.GetNumerator(), Den: requested.GetDenominator()}
	result := rendering.PhysicalMotionSample{OutputOrdinal: 0, OutputTime: composition.Rational{}, RequestedMotionTick: requestedTick, SampledMotionTick: sample.Tick, JointAngles: map[string]float64{}, CameraTransforms: map[string]physical.TRS{}}
	for _, track := range sample.Tracks {
		if track.Target.DeviceRoot && track.Transform != nil {
			result.DeviceRoot = transformKeyTRS(track.Transform)
		}
		if track.Target.JointID != "" && track.AngleRad != nil {
			result.JointAngles[track.Target.JointID] = *track.AngleRad
		}
		if track.Target.SceneCameraID != "" && track.Transform != nil {
			result.CameraTransforms[track.Target.SceneCameraID] = *transformKeyTRS(track.Transform)
		}
	}
	if len(result.JointAngles) == 0 {
		result.JointAngles = nil
	}
	if len(result.CameraTransforms) == 0 {
		result.CameraTransforms = nil
	}
	return result
}

func physicalMotionSchedule(candidate motions.MotionCandidate, output *shotsv1.PhysicalMotionOutputBinding) ([]rendering.PhysicalMotionSample, rendering.PhysicalOutputPlan, error) {
	if output == nil || candidate.DurationTicks <= 0 || uint64(candidate.DurationTicks) > uint64(^uint32(0)) {
		return nil, rendering.PhysicalOutputPlan{}, errors.New("renderjobs: physical motion duration is outside the worker profile")
	}
	rate := composition.Rational{Num: int64(output.GetOutputRateNumerator()), Den: int64(output.GetOutputRateDenominator())}
	if rate.Num <= 0 || rate.Den <= 0 || candidate.Timebase.Num <= 0 || candidate.Timebase.Den <= 0 || output.GetFrameCount() == 0 {
		return nil, rendering.PhysicalOutputPlan{}, errors.New("renderjobs: physical motion schedule is invalid")
	}
	if rate.Den != 1 || rate.Num != 24 && rate.Num != 30 {
		return nil, rendering.PhysicalOutputPlan{}, errors.New("renderjobs: physical motion output rate must be 24 or 30 fps")
	}
	profileID, profileVersion, motionProfile, maxFrames := rendering.PhysicalMotionOutput, uint32(1), rendering.PhysicalMotionProfile, uint32(48)
	if output.GetProfileId() == rendering.PhysicalMotionOutputV2 && output.GetProfileVersion() == 2 {
		profileID, profileVersion, motionProfile, maxFrames = rendering.PhysicalMotionOutputV2, 2, rendering.PhysicalMotionProfileV2, rateNumLimit(rate.Num)
	}
	if output.GetProfileId() != profileID || output.GetProfileVersion() != profileVersion || output.GetFrameCount() > maxFrames {
		return nil, rendering.PhysicalOutputPlan{}, errors.New("renderjobs: physical motion profile or frame count is unsupported")
	}
	plan := rendering.PhysicalOutputPlan{
		ProfileID: profileID, ProfileVersion: profileVersion, Kind: "motion_mp4",
		MotionProfile: motionProfile, Width: output.GetWidth(), Height: output.GetHeight(),
		Codec: "h264", PixelFormat: "yuv420p", ColorPolicy: "sdr-unmarked-default709-v1",
		Rate: rate, FrameCount: output.GetFrameCount(), MotionTicks: uint32(candidate.DurationTicks),
		Timebase: candidate.Timebase, FinalPoseHold: "last-key-through-output-end-v1", OutputName: rendering.PhysicalMotionOutputName,
		VideoToolProfileID: rendering.PhysicalMotionVideoProfileID, VideoToolArtifactID: rendering.PhysicalMotionVideoArtifactID,
	}
	samples := make([]rendering.PhysicalMotionSample, 0, output.GetFrameCount())
	for ordinal := uint32(0); ordinal < output.GetFrameCount(); ordinal++ {
		outputTime := new(big.Rat).SetFrac(new(big.Int).Mul(new(big.Int).SetUint64(uint64(ordinal)), big.NewInt(rate.Den)), big.NewInt(rate.Num))
		requestedTick := new(big.Rat).Mul(outputTime, new(big.Rat).SetFrac(big.NewInt(candidate.Timebase.Num), big.NewInt(candidate.Timebase.Den)))
		sampledTick := new(big.Rat).Set(requestedTick)
		if sampledTick.Cmp(new(big.Rat).SetInt64(candidate.DurationTicks)) > 0 {
			sampledTick.SetInt64(candidate.DurationTicks)
		}
		requested, err := physicalRationalFromBig(requestedTick)
		if err != nil {
			return nil, rendering.PhysicalOutputPlan{}, err
		}
		sampled, err := physicalRationalFromBig(sampledTick)
		if err != nil {
			return nil, rendering.PhysicalOutputPlan{}, err
		}
		timeValue, err := physicalRationalFromBig(outputTime)
		if err != nil {
			return nil, rendering.PhysicalOutputPlan{}, err
		}
		pose, err := motions.SampleAt(candidate, sampled)
		if err != nil {
			return nil, rendering.PhysicalOutputPlan{}, fmt.Errorf("sample physical motion at output frame %d: %w", ordinal, err)
		}
		sample := physicalRenderSample(&shotsv1.RationalTime{Numerator: sampled.Num, Denominator: sampled.Den}, pose)
		sample.OutputOrdinal = ordinal
		sample.OutputTime = timeValue
		sample.RequestedMotionTick = requested
		sample.SampledMotionTick = sampled
		samples = append(samples, sample)
	}
	plan.Samples = samples
	return samples, plan, nil
}

func physicalMotionJobProfile(output *shotsv1.PhysicalMotionOutputBinding) (string, error) {
	if output == nil {
		return "", errors.New("renderjobs: physical motion output is missing")
	}
	switch {
	case output.GetProfileId() == rendering.PhysicalMotionOutput && output.GetProfileVersion() == 1:
		return OutputProfilePhysicalMotionMP4, nil
	case output.GetProfileId() == rendering.PhysicalMotionOutputV2 && output.GetProfileVersion() == 2:
		return OutputProfilePhysicalMotionMP4V2, nil
	default:
		return "", fmt.Errorf("%w: physical motion output profile is unsupported", ErrInvalidRequest)
	}
}

func rateNumLimit(rate int64) uint32 {
	if rate <= 0 || rate > int64(rendering.PhysicalMotionMaxFramesV2)/10 {
		return 0
	}
	return uint32(rate * 10)
}

func physicalRationalFromBig(value *big.Rat) (composition.Rational, error) {
	if value == nil || !value.Num().IsInt64() || !value.Denom().IsInt64() || value.Denom().Sign() <= 0 {
		return composition.Rational{}, errors.New("renderjobs: exact motion schedule exceeds rational representation")
	}
	return composition.Rational{Num: value.Num().Int64(), Den: value.Denom().Int64()}, nil
}

func transformKeyTRS(key *motions.TransformKey) *physical.TRS {
	if key == nil {
		return nil
	}
	return &physical.TRS{Translation: key.Translation, Rotation: key.Rotation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}

type physicalSubjectAdapter struct{}

func (physicalSubjectAdapter) prepare(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt) (preparedOperation, error) {
	closure, err := c.loadPhysicalClosure(ctx, route, job)
	if err != nil {
		return preparedOperation{}, err
	}
	manifest := closure.Manifest
	manifest.JobID, manifest.AttemptID, manifest.InputFingerprint = job.ID, attempt.ID, job.InputFingerprint
	manifest.WorkerSHA256 = closure.WorkerSHA256
	if err := rendering.ValidatePhysicalRenderManifest(manifest); err != nil {
		return preparedOperation{}, errors.Join(errDeterministicPreparation, err)
	}
	manifestBytes, _, err := rendering.CanonicalPhysicalRenderManifest(manifest)
	if err != nil {
		return preparedOperation{}, errors.Join(errDeterministicPreparation, err)
	}
	manifestBytes = append(manifestBytes, '\n')
	files := make([]stagedFile, 0, len(manifest.Assets)+4)
	if manifest.Screen.Recording == nil {
		files = append(files, stagedFile{Name: "screen.png", BlobKey: closure.ScreenKey, SHA256: manifest.Screen.Raster.SHA256, Size: manifest.Screen.Raster.SizeBytes, Mode: manifestMode, MaxBytes: 64 << 20})
	} else {
		recording := manifest.Screen.Recording
		sourceBinding := bindingByRole(job.Input, "physical_recording_source")
		if closure.RecordingSourceID == "" || sourceBinding.ArtifactID != closure.RecordingSourceID || sourceBinding.SHA256 != recording.Inspection.Source.SHA256 || sourceBinding.RevisionID != "sha256:"+recording.Inspection.Source.SHA256 {
			return preparedOperation{}, errors.New("renderjobs: physical recording source differs from its immutable job binding")
		}
		files = append(files, stagedFile{Name: recording.Inspection.Source.Name, SHA256: recording.Inspection.Source.SHA256, Size: recording.Inspection.Source.SizeBytes, Mode: manifestMode, MaxBytes: maxInspectionInputBytes, Open: func(openCtx context.Context) (io.ReadCloser, error) {
			if err := c.requireRoute(openCtx, route); err != nil {
				return nil, err
			}
			opened, reader, openErr := c.media.Open(openCtx, job.Input.ProjectID, closure.RecordingSourceID)
			if openErr != nil {
				return nil, openErr
			}
			if opened.ID != closure.RecordingSourceID || opened.ProjectID != job.Input.ProjectID || opened.Kind != projects.MediaKindVideo || opened.SHA256 != recording.Inspection.Source.SHA256 || opened.SizeBytes != recording.Inspection.Source.SizeBytes {
				_ = reader.Close()
				return nil, errors.New("renderjobs: source owner returned changed physical recording bytes")
			}
			return reader, nil
		}})
		for _, sidecar := range []struct {
			name, role, key, digest string
			max                     int64
			size                    int64
		}{
			{recording.Inspection.Manifest.Name, "physical_recording_manifest", closure.RecordingManifestKey, recording.Inspection.Manifest.SHA256, maxManifestBytes, recording.Inspection.Manifest.SizeBytes},
			{recording.Inspection.Receipt.Name, "physical_recording_receipt", closure.RecordingReceiptKey, recording.Inspection.Receipt.SHA256, maxReceiptBytes, recording.Inspection.Receipt.SizeBytes},
			{recording.Inspection.Clock.Name, "physical_recording_clock", closure.RecordingClockKey, recording.Inspection.Clock.SHA256, maxInspectionClockBytes, recording.Inspection.Clock.SizeBytes},
		} {
			binding := bindingByRole(job.Input, sidecar.role)
			if sidecar.key == "" || binding.ArtifactID != sidecar.key || binding.SHA256 != sidecar.digest || binding.RevisionID != "sha256:"+sidecar.digest {
				return preparedOperation{}, errors.New("renderjobs: physical recording inspection sidecar differs from its immutable job binding")
			}
			data, readErr := c.readBlob(ctx, sidecar.key, sidecar.max, sidecar.digest)
			if readErr != nil || int64(len(data)) != sidecar.size {
				return preparedOperation{}, errors.Join(errors.New("renderjobs: physical recording inspection sidecar failed bounded readback"), readErr)
			}
			files = append(files, stagedFile{Name: sidecar.name, Data: data, SHA256: sidecar.digest, Size: sidecar.size, Mode: manifestMode, MaxBytes: sidecar.max})
		}
	}
	for _, asset := range manifest.Assets {
		ref := asset.Reference
		maxBytes := library.MaxTextureBytes
		if ref.Role == library.RoleGeometryGLB {
			maxBytes = library.MaxGeometryBytes
		}
		assetRef := ref
		files = append(files, stagedFile{Name: asset.Name, SHA256: ref.SHA256, Size: ref.SizeBytes, Mode: manifestMode, MaxBytes: maxBytes, Open: func(openCtx context.Context) (io.ReadCloser, error) {
			if err := c.requireRoute(openCtx, route); err != nil {
				return nil, err
			}
			resolved, resolveErr := c.physicalArtifacts.Resolve(openCtx, assetRef, library.UseRender)
			if resolveErr != nil {
				return nil, resolveErr
			}
			return resolved.Reader, nil
		}})
	}
	mode, err := operationModeForJob(job.Input)
	if err != nil || !isPhysicalOperation(mode) {
		return preparedOperation{}, errors.Join(errors.New("renderjobs: physical job output mode is invalid"), err)
	}
	return preparedOperation{Mode: mode, ManifestBytes: manifestBytes, Files: files, MaxInputBytes: physicalWorkspaceInputMax}, nil
}

func (physicalSubjectAdapter) validateCompletion(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt, intent persistedIntent, nativeKey, nativeSHA string) (jobs.CompletionEvidence, error) {
	mode, modeErr := operationModeForJob(job.Input)
	if modeErr != nil || !isPhysicalOperation(mode) {
		return jobs.CompletionEvidence{}, errors.Join(errors.New("renderjobs: physical completion output mode is invalid"), modeErr)
	}
	if err := validateNativeEvidence(ctx, c, route, job, attempt, intent, mode, nativeKey, nativeSHA); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	closure, err := c.loadPhysicalClosure(ctx, route, job)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	manifest := closure.Manifest
	manifest.JobID, manifest.AttemptID, manifest.InputFingerprint = job.ID, attempt.ID, job.InputFingerprint
	manifestBytes, _, err := rendering.CanonicalPhysicalRenderManifest(manifest)
	manifestBytes = append(manifestBytes, '\n')
	if err != nil || digestBytes(manifestBytes) != intent.ManifestSHA256 {
		return jobs.CompletionEvidence{}, errors.Join(errors.New("renderjobs: physical manifest differs from persisted intent"), err)
	}
	workspace, err := attemptWorkspace(route.Root, job.ID, attempt.ID)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	verified, err := rendering.VerifyPhysicalRenderReceipt(workspace, manifest)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if verified.JobID != job.ID || verified.AttemptID != attempt.ID || verified.InputFingerprint != job.InputFingerprint || verified.WorkerSHA256 != intent.WorkerSHA256 || verified.ManifestSHA256 != intent.ManifestSHA256 {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: verified physical receipt does not match claimed job attempt")
	}
	receiptBytes, err := readWorkspaceFile(workspace, rendering.PhysicalReceiptName, 2<<20)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	outputKind, outputMIME, outputExt := physicalOutputKind(job.Input.OutputProfile), "image/png", ".png"
	if manifest.Output.Kind == "motion_mp4" {
		outputKind, outputMIME, outputExt = "physical_motion_mp4", "video/mp4", ".mp4"
	}
	outputKey := outputBlobPrefix + job.ID + "/" + attempt.ID + "/" + verified.Output.SHA256 + outputExt
	if err := c.publishWorkspaceOutput(ctx, route, workspace, verified.Output.Name, outputKey, verified.Output.SizeBytes, verified.Output.SHA256, outputMIME); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	manifestKey := receiptBlobPrefix + job.ID + "/" + attempt.ID + "/physical-manifest-" + intent.ManifestSHA256 + ".json"
	if err := c.putVerified(ctx, route, manifestKey, manifestBytes, "application/json", intent.ManifestSHA256); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	receiptSHA := digestBytes(receiptBytes)
	receiptKey := receiptBlobPrefix + job.ID + "/" + attempt.ID + "/physical-receipt-" + receiptSHA + ".json"
	if err := c.putVerified(ctx, route, receiptKey, receiptBytes, "application/json", receiptSHA); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	return jobs.CompletionEvidence{AttemptID: attempt.ID, ExecutionClaimID: attempt.Execution.ID, InputFingerprint: job.InputFingerprint, Outputs: []jobs.OutputRef{
		{Kind: outputKind, ArtifactID: outputKey, SHA256: verified.Output.SHA256, MIMEType: outputMIME, SizeBytes: verified.Output.SizeBytes, Width: int(verified.Output.Width), Height: int(verified.Output.Height)},
		{Kind: "physical_receipt", ArtifactID: receiptKey, SHA256: receiptSHA, MIMEType: "application/json", SizeBytes: int64(len(receiptBytes))},
		{Kind: "physical_manifest", ArtifactID: manifestKey, SHA256: intent.ManifestSHA256, MIMEType: "application/json", SizeBytes: int64(len(manifestBytes))},
	}, ReceiptArtifactID: receiptKey, ReceiptSHA256: receiptSHA, CompletedAt: c.now().UTC()}, nil
}

func (physicalSubjectAdapter) finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error {
	return nil
}

func (c *Coordinator) loadPhysicalClosure(ctx context.Context, route routePin, job jobs.Job) (physicalClosure, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return physicalClosure{}, err
	}
	binding := bindingByRole(job.Input, "physical_closure")
	if binding.ArtifactID == "" || !strings.HasPrefix(binding.ArtifactID, closureBlobPrefix) || !validDigest(binding.SHA256) || binding.RevisionID != "sha256:"+binding.SHA256 {
		return physicalClosure{}, errors.New("renderjobs: job does not bind an immutable physical closure")
	}
	data, err := c.readBlob(ctx, binding.ArtifactID, maxPhysicalClosureBytes, binding.SHA256)
	if err != nil {
		return physicalClosure{}, err
	}
	var closure physicalClosure
	if err := decodeJSON(data, &closure); err != nil {
		return physicalClosure{}, err
	}
	canonical, err := json.Marshal(closure)
	if err != nil || !bytes.Equal(data, append(canonical, '\n')) {
		return physicalClosure{}, errors.New("renderjobs: physical closure is not canonical")
	}
	if closure.Version != physicalClosureVersion || closure.ProjectID != job.Input.ProjectID || closure.ShotID != job.Input.SubjectID || closure.Revision != job.Input.ShotRevision || job.Input.ShotRevisionID != shotRevisionID(closure.ShotID, closure.Revision) || closure.RecipeSHA256 != bindingDigest(job.Input, "shot_snapshot") || closure.WorkerSHA256 != bindingDigest(job.Input, "worker_executable") || closure.Manifest.Profile.DigestSHA256 != job.Input.ToolProfileDigest || closure.Manifest.Profile.ID != job.Input.RendererVersion || job.Input.SubjectKind != jobs.SubjectPhysicalRender {
		return physicalClosure{}, errors.New("renderjobs: physical closure does not match its immutable job input")
	}
	wantOutputProfile := OutputProfilePNG
	if closure.Manifest.Output.ProfileID == rendering.PhysicalPreviewProfile && closure.Manifest.Output.ProfileVersion == rendering.PhysicalPreviewProfileVersion {
		wantOutputProfile = OutputProfilePhysicalPreviewPNG
	}
	if closure.Manifest.Output.Kind == "motion_mp4" {
		if closure.Manifest.Output.ProfileID == rendering.PhysicalMotionOutput && closure.Manifest.Output.ProfileVersion == 1 {
			wantOutputProfile = OutputProfilePhysicalMotionMP4
		} else if closure.Manifest.Output.ProfileID == rendering.PhysicalMotionOutputV2 && closure.Manifest.Output.ProfileVersion == 2 {
			wantOutputProfile = OutputProfilePhysicalMotionMP4V2
		} else if closure.Manifest.Output.ProfileID == rendering.PhysicalRecordingMotionOutput && closure.Manifest.Output.ProfileVersion == 1 && closure.Manifest.Version == rendering.PhysicalRecordingManifestVersion && closure.Manifest.Screen.Recording != nil {
			wantOutputProfile = OutputProfilePhysicalRecordingMP4
		} else {
			return physicalClosure{}, errors.New("renderjobs: physical closure uses an unsupported motion output profile")
		}
	}
	if job.Input.OutputProfile != wantOutputProfile {
		return physicalClosure{}, errors.New("renderjobs: physical closure output profile differs from immutable job input")
	}
	if err := validatePhysicalIdentityBindings(job.Input, closure); err != nil {
		return physicalClosure{}, err
	}
	if closure.Manifest.Screen.Recording == nil {
		got := bindingByRole(job.Input, "physical_screen_raster")
		if closure.ScreenKey == "" || got.ArtifactID != closure.ScreenKey || got.SHA256 != closure.Manifest.Screen.Raster.SHA256 || got.RevisionID != "sha256:"+closure.Manifest.Screen.Raster.SHA256 {
			return physicalClosure{}, errors.New("renderjobs: physical screen raster binding differs from closure")
		}
	} else if err := validatePhysicalRecordingBindings(job.Input, closure); err != nil {
		return physicalClosure{}, err
	}
	if closure.Manifest.Screen.Recording == nil {
		for _, role := range []string{"physical_recording_source", "physical_recording_manifest", "physical_recording_receipt", "physical_recording_clock"} {
			if bindingByRole(job.Input, role).ArtifactID != "" {
				return physicalClosure{}, errors.New("renderjobs: still job carries unexpected recording bindings")
			}
		}
	}
	for _, asset := range closure.Manifest.Assets {
		got := bindingByRole(job.Input, "physical_asset:"+asset.BindingID)
		if got.ArtifactID != physicalArtifactIdentity(asset.Reference) || got.SHA256 != asset.Reference.SHA256 || got.RevisionID != "sha256:"+asset.Reference.SHA256 {
			return physicalClosure{}, errors.New("renderjobs: physical asset binding differs from closure")
		}
	}
	adapter := bindingByRole(job.Input, "physical_adapter")
	if adapter.ArtifactID != "embedded-adapter:sha256:"+closure.AdapterSHA256 || adapter.SHA256 != closure.AdapterSHA256 || adapter.RevisionID != "sha256:"+closure.AdapterSHA256 || closure.Manifest.Profile.AdapterSHA256 != closure.AdapterSHA256 {
		return physicalClosure{}, errors.New("renderjobs: physical adapter binding differs from immutable profile")
	}
	return closure, nil
}

func validatePhysicalRecordingBindings(input jobs.Input, closure physicalClosure) error {
	recording := closure.Manifest.Screen.Recording
	if recording == nil {
		return errors.New("renderjobs: recording bindings have no recording closure")
	}
	if closure.RecordingSourceID != closure.Manifest.Screen.OriginalMediaID {
		return errors.New("renderjobs: physical recording media ID differs from original screen input")
	}
	i := recording.Inspection
	for _, expected := range []struct{ role, key, sha string }{
		{"physical_recording_source", closure.RecordingSourceID, i.Source.SHA256},
		{"physical_recording_manifest", closure.RecordingManifestKey, i.Manifest.SHA256},
		{"physical_recording_receipt", closure.RecordingReceiptKey, i.Receipt.SHA256},
		{"physical_recording_clock", closure.RecordingClockKey, i.Clock.SHA256},
	} {
		got := bindingByRole(input, expected.role)
		if expected.key == "" || !validDigest(expected.sha) || got.ArtifactID != expected.key || got.SHA256 != expected.sha || got.RevisionID != "sha256:"+expected.sha {
			return fmt.Errorf("renderjobs: physical %s binding differs from closure", expected.role)
		}
	}
	return nil
}

func validatePhysicalIdentityBindings(input jobs.Input, closure physicalClosure) error {
	worker := bindingByRole(input, "worker_executable")
	if worker.ArtifactID != closure.WorkerKey || worker.SHA256 != closure.WorkerSHA256 || worker.RevisionID != "sha256:"+closure.WorkerSHA256 {
		return errors.New("renderjobs: physical worker binding differs from immutable closure")
	}
	for _, candidate := range []struct {
		role, id string
		revision uint32
		sha      string
	}{
		{"device_revision", closure.Manifest.Device.ID, closure.Manifest.Device.Revision, closure.Manifest.Device.CanonicalSHA},
		{"scene_revision", closure.Manifest.Scene.ID, closure.Manifest.Scene.Revision, closure.Manifest.Scene.CanonicalSHA},
		{"motion_revision", closure.Manifest.Motion.ID, closure.Manifest.Motion.Revision, closure.Manifest.Motion.CanonicalSHA},
	} {
		binding := bindingByRole(input, candidate.role)
		if binding.ArtifactID != candidate.id || binding.RevisionID != "revision:"+strconv.FormatUint(uint64(candidate.revision), 10) || binding.SHA256 != candidate.sha {
			return fmt.Errorf("renderjobs: physical %s binding differs from immutable closure", candidate.role)
		}
	}
	return nil
}

var _ = physicalv1.TRS{}
