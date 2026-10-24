package rendering

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"math/big"
	"path/filepath"
	"regexp"
	"strings"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/presentations"
	"device-studio/internal/scenes"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

const (
	PhysicalManifestName                    = "physical-render-manifest.json"
	PhysicalAdapterInputName                = "physical-worker-input.json"
	PhysicalReceiptName                     = "physical-render-receipt.json"
	PhysicalAdapterName                     = "physical_blender.py"
	PhysicalManifestVersion                 = 1
	PhysicalMotionManifestVersion           = 2
	PhysicalPreviewManifestVersion          = 3
	PhysicalRecordingManifestVersion        = 4
	PhysicalArticulatedManifestVersion      = 5
	PhysicalArticulationProfile             = "device-studio-single-revolute-hinge-v1"
	PhysicalRendererProfile                 = "device-studio-physical-cycles-v1"
	PhysicalMotionProfile                   = "device-studio-rigid-root-motion-v1"
	PhysicalMotionProfileV2                 = "device-studio-rigid-root-motion-v2"
	PhysicalStillProfile                    = "device-studio-physical-still-2560-v1"
	PhysicalPreviewProfile                  = "device-studio-physical-preview-960x540-v1"
	PhysicalPreviewProfileVersion           = uint32(1)
	PhysicalDiagnosticProfile               = "device-studio-physical-diagnostic-960x540-v1"
	PhysicalDiagnosticProfileVersion        = uint32(1)
	PhysicalDiagnosticMotionOutput          = "device-studio-physical-recording-diagnostic-1080p-v1"
	PhysicalDiagnosticMotionProfile         = "device-studio-physical-recording-diagnostic-motion-v1"
	PhysicalMotionOutput                    = "device-studio-physical-motion-1080p-v1"
	PhysicalMotionOutputV2                  = "device-studio-physical-motion-1080p-v2"
	PhysicalRecordingMotionOutput           = "device-studio-physical-recording-1080p-v1"
	PhysicalRecordingMotionProfile          = "device-studio-physical-recording-motion-v1"
	PhysicalRecordingAudioUnspecified       = "unspecified"
	PhysicalRecordingAudioSilent            = "silent"
	PhysicalRecordingAudioPassthrough       = "passthrough"
	PhysicalColorProfile                    = "device-studio-blender-standard-v1"
	PhysicalBlenderProfileID                = "blender-5.2.2-linux-x64"
	PhysicalBlenderVersion                  = "Blender 5.2.2 LTS"
	PhysicalBlenderArtifact                 = "a1660c40fcc34e79aaa339f846b95d5973981a23b9e4b87e50ee2c633159762e"
	PhysicalBlenderTreeSHA                  = "5fca420feab408276778511a84d7a2f5659bbc09aa5cb43b6d24ae59b220cb55"
	PhysicalBlenderBinarySHA                = "bafec183e960819c76fe213c000cdb2284c53a87ef6791812f8728b51f07145b"
	PhysicalBlenderArchive                  = "blender-5.2.2-linux-x64"
	PhysicalStillWallTimeSeconds            = uint32(180)
	PhysicalStillCPUWallTimeSeconds         = uint32(600)
	PhysicalPreviewWallTimeSeconds          = uint32(30)
	PhysicalDiagnosticWallTimeSeconds       = uint32(30)
	PhysicalMotionWallTimeSeconds           = uint32(300)
	PhysicalMotionV2WallTimeSeconds         = uint32(1200)
	PhysicalRecordingWallTimeSeconds        = uint32(1200)
	PhysicalDiagnosticMotionWallTimeSeconds = uint32(600)
	PhysicalOperationMemoryMaxBytes         = uint64(8 << 30)
	PhysicalOperationTasksMax               = uint32(128)
	PhysicalOperationCPUWeight              = uint32(100)
	PhysicalOperationThreads                = uint32(2)
	PhysicalStillSamples                    = uint32(256)
	PhysicalPreviewSamples                  = uint32(64)
	PhysicalDiagnosticSamples               = uint32(4)
	PhysicalDiagnosticMotionSamples         = uint32(4)
	PhysicalMotionSamples                   = uint32(128)
	PhysicalMotionMaxFramesV1               = uint32(48)
	PhysicalMotionMaxFramesV2               = uint32(300)
	PhysicalRecordingMaxFrames              = uint32(300)
	PhysicalDiagnosticMotionMaxFrames       = uint32(48)
	PhysicalRecordingMaxSourcePrefix        = uint32(3600)
	PhysicalStillOutputName                 = "physical-output.png"
	PhysicalPreviewOutputName               = "physical-preview.png"
	PhysicalDiagnosticOutputName            = "physical-diagnostic.png"
	PhysicalMotionOutputName                = "physical-output.mp4"
	PhysicalMotionVideoProfileID            = "btbn-ffmpeg-video-n9.0.2-28-g669f5ab083-linux64-gpl"
	PhysicalMotionVideoArtifactID           = "2ad074849ce8635ce03d623865a26fdb1636a9a44346d21d4af08c1b191a9789"
	physicalManifestMax                     = 4 << 20
	physicalSnapshotMax                     = 1 << 20
	physicalAssetsMax                       = 16
	physicalFrameMax                        = 360
	physicalInputBytesMax                   = 512 << 20
	physicalScreenBytesMax                  = 64 << 20
)

var physicalDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type PhysicalCandidateSnapshot struct {
	ID            string `json:"id"`
	Revision      uint32 `json:"revision"`
	CanonicalSHA  string `json:"canonical_sha256"`
	CanonicalJSON []byte `json:"canonical_json"`
}

type PhysicalAssetInput struct {
	Role      string               `json:"role"`
	BindingID string               `json:"binding_id"`
	Reference physical.ArtifactRef `json:"reference"`
	Name      string               `json:"name"`
}

// PhysicalScreenInput binds the project-owned screen raster separately from
// reviewed library assets. The renderer never opens OriginalMediaID or any
// provenance SourcePath; it opens only the named, digest-checked ImageBinding.
type PhysicalScreenInput struct {
	DisplayID           string                  `json:"display_id"`
	OriginalMediaID     string                  `json:"original_media_id"`
	OriginalMediaSHA256 string                  `json:"original_media_sha256"`
	Raster              ImageBinding            `json:"raster"`
	PresentationProfile string                  `json:"presentation_profile"`
	PresentationVersion string                  `json:"presentation_version"`
	ConfigSHA256        string                  `json:"config_sha256"`
	LayoutSHA256        string                  `json:"layout_sha256"`
	RasterSHA256        string                  `json:"raster_sha256"`
	Width               int                     `json:"width"`
	Height              int                     `json:"height"`
	UVSet               string                  `json:"uv_set"`
	MappingProfile      string                  `json:"mapping_profile"`
	PrimitiveIndex      uint32                  `json:"primitive_index"`
	MaterialSlot        uint32                  `json:"material_slot"`
	Recording           *PhysicalRecordingInput `json:"recording,omitempty"`
}

// PhysicalRecordingInput binds an already-inspected source clock and the
// immutable presentation applied to each selected source frame. The source
// and clock bytes remain private attempt inputs; this value contains no paths.
type PhysicalRecordingInput struct {
	Inspection             VideoInspection      `json:"inspection"`
	TrimStartPTS           int64                `json:"trim_start_pts"`
	TrimEndPTS             int64                `json:"trim_end_pts"`
	PlaybackRate           composition.Rational `json:"playback_rate"`
	AudioPolicy            string               `json:"audio_policy"`
	SourcePrefixFrameCount uint32               `json:"source_prefix_frame_count"`
	Presentation           presentations.Config `json:"presentation"`
	PresentationLayout     presentations.Layout `json:"presentation_layout"`
}

// PhysicalPlacement is explicit scene-local data. Matrices use column-major
// column-vector convention and the physical package's canonical meter basis.
// DevicePartLocalFrame remains a parsed-node assertion and is not multiplied
// a second time by the renderer.
type PhysicalPlacement struct {
	DeviceSupportID   string        `json:"device_support_id"`
	SceneSupportID    string        `json:"scene_support_id"`
	SelectedCameraID  string        `json:"selected_camera_id"`
	SceneSupportWorld physical.TRS  `json:"scene_support_world"`
	SupportPivot      physical.Vec3 `json:"support_pivot_m"`
	DeviceRootRest    physical.TRS  `json:"device_root_rest"`
}

type PhysicalRenderProfile struct {
	ID                  string `json:"id"`
	Version             uint32 `json:"version"`
	DigestSHA256        string `json:"digest_sha256"`
	BlenderArtifactID   string `json:"blender_artifact_identity"`
	BlenderTreeSHA256   string `json:"blender_tree_sha256"`
	BlenderBinarySHA256 string `json:"blender_binary_sha256"`
	BlenderVersion      string `json:"blender_version"`
	AdapterSHA256       string `json:"adapter_sha256"`
	Backend             string `json:"backend"`
	Engine              string `json:"engine"`
	Samples             uint32 `json:"samples"`
	Seed                uint32 `json:"seed"`
	AdaptiveSampling    bool   `json:"adaptive_sampling"`
	Denoise             bool   `json:"denoise"`
	ColorManagement     string `json:"color_management"`
	ColorView           string `json:"color_view"`
	ColorLook           string `json:"color_look"`
	Exposure            string `json:"exposure"`
	Gamma               string `json:"gamma"`
	WallTimeSeconds     uint32 `json:"wall_time_seconds"`
	MemoryMaxBytes      uint64 `json:"memory_max_bytes"`
	TasksMax            uint32 `json:"tasks_max"`
	CPUWeight           uint32 `json:"cpu_weight"`
	Threads             uint32 `json:"threads"`
}

type PhysicalOutputPlan struct {
	ProfileID           string                      `json:"profile_id"`
	ProfileVersion      uint32                      `json:"profile_version,omitempty"`
	ArticulationProfile string                      `json:"articulation_profile,omitempty"`
	Kind                string                      `json:"kind"` // still_png or motion_mp4
	MotionProfile       string                      `json:"motion_profile,omitempty"`
	Width               uint32                      `json:"width"`
	Height              uint32                      `json:"height"`
	Codec               string                      `json:"codec,omitempty"`
	PixelFormat         string                      `json:"pixel_format,omitempty"`
	ColorPolicy         string                      `json:"color_policy,omitempty"`
	Rate                composition.Rational        `json:"rate"`
	FrameCount          uint32                      `json:"frame_count"`
	MotionTicks         uint32                      `json:"motion_ticks"`
	Timebase            composition.Rational        `json:"timebase_ticks_per_second"`
	FinalPoseHold       string                      `json:"final_pose_hold"`
	Samples             []PhysicalMotionSample      `json:"samples,omitempty"`
	CameraEndpoint      *PhysicalCameraEndpointPlan `json:"camera_endpoint,omitempty"`
	OutputName          string                      `json:"output_name"`
	VideoToolProfileID  string                      `json:"video_tool_profile_id,omitempty"`
	VideoToolArtifactID string                      `json:"video_tool_artifact_identity,omitempty"`
}

// PhysicalCameraEndpointPlan captures the reviewed geometric input for a
// compiled camera handoff. Display corners are already transformed into the
// final device pose and canonical world basis; the immutable shot/asset
// revision remains the source of those facts.
type PhysicalCameraEndpointPlan struct {
	CameraID              string           `json:"camera_id"`
	DisplayID             string           `json:"display_id"`
	TargetKind            string           `json:"target_kind"`
	RasterWidth           uint32           `json:"raster_width"`
	RasterHeight          uint32           `json:"raster_height"`
	Target                NormalizedRect   `json:"target"`
	DisplayTarget         NormalizedRect   `json:"display_target"`
	ContentTarget         NormalizedRect   `json:"content_target"`
	ContentWidth          uint32           `json:"content_width_pixels,omitempty"`
	ContentHeight         uint32           `json:"content_height_pixels,omitempty"`
	DisplayCorners        [4]physical.Vec3 `json:"display_corners_world_m"`
	ContentCorners        [4]physical.Vec3 `json:"content_corners_world_m"`
	ContentMappingProfile string           `json:"content_mapping_profile,omitempty"`
	CameraFrame           physical.TRS     `json:"camera_frame"`
	FocalLengthMM         float64          `json:"focal_length_mm"`
	SensorWidthMM         float64          `json:"sensor_width_mm"`
	SensorHeightMM        float64          `json:"sensor_height_mm"`
	MaxErrorPixels        float64          `json:"max_error_pixels"`
	ResidualPixels        float64          `json:"residual_pixels,omitempty"`
	ArrivalMotionTick     uint32           `json:"arrival_motion_tick,omitempty"`
}

// PhysicalMotionSample is computed by motions.SampleAt before a native claim.
// Blender applies these values; it does not implement a second sampler.
type PhysicalMotionSample struct {
	OutputOrdinal       uint32                  `json:"output_ordinal"`
	OutputTime          composition.Rational    `json:"output_time_seconds"`
	RequestedMotionTick composition.Rational    `json:"requested_motion_tick"`
	SampledMotionTick   composition.Rational    `json:"sampled_motion_tick"`
	DeviceRoot          *physical.TRS           `json:"device_root,omitempty"`
	JointAngles         map[string]float64      `json:"joint_angles,omitempty"`
	CameraTransforms    map[string]physical.TRS `json:"camera_transforms,omitempty"`
}

// PhysicalRenderManifest is the immutable, path-free adapter input. Candidate
// snapshots are byte strings so nested canonical bytes and hashes remain exact.
type PhysicalRenderManifest struct {
	Version          int                       `json:"version"`
	JobID            string                    `json:"job_id"`
	AttemptID        string                    `json:"attempt_id"`
	InputFingerprint string                    `json:"input_fingerprint"`
	WorkerSHA256     string                    `json:"worker_sha256"`
	Device           PhysicalCandidateSnapshot `json:"device"`
	Scene            PhysicalCandidateSnapshot `json:"scene"`
	Motion           PhysicalCandidateSnapshot `json:"motion"`
	Assets           []PhysicalAssetInput      `json:"assets"`
	Screen           PhysicalScreenInput       `json:"screen"`
	Placement        PhysicalPlacement         `json:"placement"`
	Profile          PhysicalRenderProfile     `json:"profile"`
	Output           PhysicalOutputPlan        `json:"output"`
}

// CanonicalPhysicalRenderManifest validates all pinned owner snapshots and
// returns canonical manifest bytes and their digest.
func CanonicalPhysicalRenderManifest(input PhysicalRenderManifest) ([]byte, string, error) {
	if err := ValidatePhysicalRenderManifest(input); err != nil {
		return nil, "", err
	}
	data, err := json.Marshal(input)
	if err != nil {
		return nil, "", fmt.Errorf("encode physical render manifest: %w", err)
	}
	if len(data) > physicalManifestMax {
		return nil, "", errors.New("physical render manifest exceeds 4 MiB")
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), nil
}

func ValidatePhysicalRenderManifest(m PhysicalRenderManifest) error {
	if !((m.Version == PhysicalManifestVersion && m.Output.Kind == "still_png" && m.Output.ProfileID == PhysicalStillProfile) ||
		(m.Version == PhysicalPreviewManifestVersion && m.Output.Kind == "still_png" && (m.Output.ProfileID == PhysicalPreviewProfile || m.Output.ProfileID == PhysicalDiagnosticProfile)) ||
		(m.Version == PhysicalMotionManifestVersion && m.Output.Kind == "motion_mp4" && m.Screen.Recording == nil) ||
		(m.Version == PhysicalRecordingManifestVersion && m.Output.Kind == "motion_mp4" && (m.Output.ProfileID == PhysicalRecordingMotionOutput || m.Output.ProfileID == PhysicalDiagnosticMotionOutput) && m.Screen.Recording != nil) ||
		(m.Version == PhysicalArticulatedManifestVersion && m.Output.ArticulationProfile == PhysicalArticulationProfile && validPhysicalArticulatedOutput(m.Output, m.Screen.Recording != nil))) || !identifierPattern.MatchString(m.JobID) || !identifierPattern.MatchString(m.AttemptID) ||
		!physicalDigestPattern.MatchString(m.InputFingerprint) || !physicalDigestPattern.MatchString(m.WorkerSHA256) {
		return errors.New("physical render identity is invalid")
	}
	if m.Version != PhysicalRecordingManifestVersion && m.Version != PhysicalArticulatedManifestVersion && m.Screen.Recording != nil {
		return errors.New("physical recording input requires the recording manifest version")
	}
	if m.Version != PhysicalArticulatedManifestVersion && m.Output.ArticulationProfile != "" {
		return errors.New("articulation profile requires its versioned manifest")
	}
	if err := validatePhysicalCandidate(m.Device, "device"); err != nil {
		return err
	}
	if err := validatePhysicalCandidate(m.Scene, "scene"); err != nil {
		return err
	}
	if err := validatePhysicalCandidate(m.Motion, "motion"); err != nil {
		return err
	}
	device, _, err := decodeCanonicalDeviceSnapshot(m.Device.CanonicalJSON)
	if err != nil {
		return fmt.Errorf("decode device snapshot: %w", err)
	}
	for _, model := range device.Definition.ModelCandidates {
		if strings.TrimSpace(model.SourcePath) != "" {
			return errors.New("device candidate contains a historical source_path; save a portable candidate revision before rendering")
		}
	}
	if m.Device.ID != device.Definition.ID || m.Scene.ID != decodeSceneID(m.Scene.CanonicalJSON) || m.Motion.ID != decodeMotionID(m.Motion.CanonicalJSON) {
		return errors.New("candidate snapshot IDs do not match the canonical owner snapshots")
	}
	scene, _, err := decodeCanonicalSceneSnapshot(m.Scene.CanonicalJSON)
	if err != nil {
		return fmt.Errorf("decode scene snapshot: %w", err)
	}
	motion, _, err := decodeCanonicalMotionSnapshot(m.Motion.CanonicalJSON)
	if err != nil {
		return fmt.Errorf("decode motion snapshot: %w", err)
	}
	if err := validateCandidateAssetBindings(device, scene, m.Assets); err != nil {
		return err
	}
	if err := validatePhysicalMaterialRuntimeSupport(scene); err != nil {
		return err
	}
	if err := validateSelectedFrames(device, scene, m.Screen.DisplayID, m.Placement); err != nil {
		return err
	}
	if err := validatePhysicalAssets(m.Assets); err != nil {
		return err
	}
	var stagedBytes int64
	for _, asset := range m.Assets {
		if asset.Reference.SizeBytes > physicalInputBytesMax-stagedBytes {
			return errors.New("physical render inputs exceed the aggregate staging limit")
		}
		stagedBytes += asset.Reference.SizeBytes
	}
	if m.Screen.Recording != nil {
		if m.Screen.Recording.Inspection.Source.SizeBytes > physicalInputBytesMax-stagedBytes {
			return errors.New("physical recording source exceeds the aggregate staging limit")
		}
		stagedBytes += m.Screen.Recording.Inspection.Source.SizeBytes
	} else if m.Screen.Raster.SizeBytes > physicalScreenBytesMax || m.Screen.Raster.SizeBytes > physicalInputBytesMax-stagedBytes {
		return errors.New("physical screen raster exceeds its per-file or aggregate staging limit")
	}
	if err := validatePhysicalScreen(m.Screen); err != nil {
		return err
	}
	if err := validatePhysicalPlacement(m.Placement); err != nil {
		return err
	}
	if err := validatePhysicalProfile(m.Profile, m.Output); err != nil {
		return err
	}
	if err := validatePhysicalOutput(m.Output); err != nil {
		return err
	}
	wantSamples := PhysicalStillSamples
	switch m.Output.ProfileID {
	case PhysicalPreviewProfile:
		wantSamples = PhysicalPreviewSamples
	case PhysicalDiagnosticProfile:
		wantSamples = PhysicalDiagnosticSamples
	case PhysicalMotionOutput, PhysicalMotionOutputV2, PhysicalRecordingMotionOutput:
		wantSamples = PhysicalMotionSamples
	case PhysicalDiagnosticMotionOutput:
		wantSamples = PhysicalDiagnosticMotionSamples
	}
	if m.Profile.Samples != wantSamples {
		return errors.New("physical render quality does not match the closed output profile")
	}
	if err := validatePhysicalCameraEndpointPlan(m.Output.CameraEndpoint, scene.CameraFrames, m.Placement, m.Screen.DisplayID, motion, m.Output); err != nil {
		return err
	}
	for _, sample := range m.Output.Samples {
		if m.Version != PhysicalArticulatedManifestVersion && len(sample.JointAngles) != 0 {
			return errors.New("physical renderer profile does not support articulated joint poses")
		}
		for cameraID := range sample.CameraTransforms {
			if cameraID != m.Placement.SelectedCameraID {
				return errors.New("physical renderer profile only supports motion on its selected camera")
			}
		}
	}
	if m.Version == PhysicalArticulatedManifestVersion {
		if err := validateUniformPositiveScale(m.Placement.SceneSupportWorld.Scale); err != nil {
			return fmt.Errorf("articulated scene support transform: %w", err)
		}
		if err := validateUniformPositiveScale(m.Placement.DeviceRootRest.Scale); err != nil {
			return fmt.Errorf("articulated device rest transform: %w", err)
		}
		if _, err := ValidatePhysicalArticulation(device, motion, m.Output.Samples, PhysicalGLBInventory{}); err != nil {
			return fmt.Errorf("validate selected physical hinge: %w", err)
		}
	}
	return validateMotionOutput(m.Output, motion)
}

func validatePhysicalCameraEndpointPlan(endpoint *PhysicalCameraEndpointPlan, cameras []scenes.SceneCameraFrame, placement PhysicalPlacement, displayID string, motion motions.MotionCandidate, output PhysicalOutputPlan) error {
	if endpoint == nil {
		return nil
	}
	if output.Kind != "motion_mp4" || len(output.Samples) < 2 || endpoint.CameraID != placement.SelectedCameraID || endpoint.DisplayID == "" || endpoint.DisplayID != displayID || endpoint.RasterWidth != output.Width || endpoint.RasterHeight != output.Height {
		return errors.New("physical camera endpoint plan does not match the selected display, camera, and motion output")
	}
	var camera *scenes.SceneCameraFrame
	for i := range cameras {
		if cameras[i].ID == endpoint.CameraID {
			camera = &cameras[i]
			break
		}
	}
	if camera == nil || camera.Projection != scenes.ProjectionPerspective || camera.SensorFit != scenes.SensorFitHorizontal || camera.FocalLengthMM == nil || camera.SensorWidthMM == nil || camera.SensorHeightMM == nil || camera.Frame != endpoint.CameraFrame || *camera.FocalLengthMM != endpoint.FocalLengthMM || *camera.SensorWidthMM != endpoint.SensorWidthMM || *camera.SensorHeightMM != endpoint.SensorHeightMM {
		return errors.New("physical camera endpoint plan intrinsics differ from the selected immutable scene camera")
	}
	legacyActiveDisplay := endpoint.TargetKind == "" && endpoint.DisplayTarget == (NormalizedRect{}) && endpoint.ContentTarget == (NormalizedRect{}) && endpoint.ContentCorners == ([4]physical.Vec3{})
	targetKind := shotsv1.PhysicalCameraEndpointTargetKind_PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_ACTIVE_DISPLAY
	switch endpoint.TargetKind {
	case targetKind.String():
	case shotsv1.PhysicalCameraEndpointTargetKind_PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_RECORDING_CONTENT.String():
		targetKind = shotsv1.PhysicalCameraEndpointTargetKind_PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_RECORDING_CONTENT
	case "":
	default:
		return errors.New("physical camera endpoint target kind is unknown")
	}
	protoEndpoint := &shotsv1.PhysicalCameraEndpoint{DisplayId: endpoint.DisplayID, RasterWidth: endpoint.RasterWidth, RasterHeight: endpoint.RasterHeight, X: endpoint.Target.X, Y: endpoint.Target.Y, Width: endpoint.Target.Width, Height: endpoint.Target.Height, TargetKind: targetKind, ContentWidthPixels: endpoint.ContentWidth, ContentHeightPixels: endpoint.ContentHeight, ArrivalMotionTick: endpoint.ArrivalMotionTick}
	var mappedContentCorners *[4]physical.Vec3
	if endpoint.ContentMappingProfile != "" {
		if endpoint.ContentMappingProfile != PhysicalScreenTriangleUVMappingProfile || targetKind != shotsv1.PhysicalCameraEndpointTargetKind_PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_RECORDING_CONTENT {
			return errors.New("physical camera endpoint has an unsupported content mapping profile")
		}
		mappedContentCorners = &endpoint.ContentCorners
	}
	geometry, err := ResolvePhysicalCameraEndpointGeometryWithContentCorners(protoEndpoint, endpoint.DisplayCorners, mappedContentCorners)
	if err != nil {
		return fmt.Errorf("validate physical camera endpoint content layout: %w", err)
	}
	if !legacyActiveDisplay && (geometry.DisplayTarget != endpoint.DisplayTarget || geometry.ContentTarget != endpoint.ContentTarget || geometry.ContentCorners != endpoint.ContentCorners || geometry.ContentWidth != endpoint.ContentWidth || geometry.ContentHeight != endpoint.ContentHeight) {
		return errors.New("physical camera endpoint recorded contain geometry differs from its source, display, or target")
	}
	pose, residual, err := CompilePhysicalCameraEndpoint(PhysicalCameraEndpoint{DisplayCorners: geometry.ContentCorners, RasterWidth: endpoint.RasterWidth, RasterHeight: endpoint.RasterHeight, Target: geometry.ContentTarget, FocalLengthMM: endpoint.FocalLengthMM, SensorWidthMM: endpoint.SensorWidthMM, SensorHeightMM: endpoint.SensorHeightMM})
	if err != nil {
		return fmt.Errorf("validate physical camera endpoint geometry: %w", err)
	}
	tolerance := endpoint.MaxErrorPixels
	if tolerance == 0 {
		tolerance = defaultCameraEndpointErrorPixels
	}
	if !positiveCameraNumber(tolerance) || tolerance > 8 || residual > tolerance {
		return errors.New("physical camera endpoint residual exceeds its bounded pixel tolerance")
	}
	if endpoint.ResidualPixels != 0 && (!finiteNumber(endpoint.ResidualPixels) || math.Abs(endpoint.ResidualPixels-residual) > 1e-6) {
		return errors.New("physical camera endpoint recorded residual differs from its recomputed residual")
	}
	delta, err := CameraEndpointRelativeTransform(endpoint.CameraFrame, pose)
	if err != nil {
		return fmt.Errorf("validate physical camera endpoint local transform: %w", err)
	}
	for _, track := range motion.Tracks {
		if track.Target.SceneCameraID == endpoint.CameraID {
			return errors.New("physical camera endpoint duplicates an authored selected-camera motion track")
		}
	}
	progress, err := PhysicalCameraEndpointSampleProgress(endpoint.ArrivalMotionTick, output.Samples)
	if err != nil {
		return fmt.Errorf("validate physical camera endpoint arrival schedule: %w", err)
	}
	for i, sample := range output.Samples {
		got, ok := sample.CameraTransforms[endpoint.CameraID]
		if !ok || got != InterpolateCameraTransform(delta, progress[i]) {
			return fmt.Errorf("physical camera endpoint sample %d differs from the compiled deterministic handoff", i)
		}
	}
	return nil
}

func validPhysicalArticulatedOutput(output PhysicalOutputPlan, hasRecording bool) bool {
	if output.ArticulationProfile != PhysicalArticulationProfile {
		return false
	}
	if output.Kind == "still_png" {
		return !hasRecording && (output.ProfileID == PhysicalStillProfile || output.ProfileID == PhysicalPreviewProfile || output.ProfileID == PhysicalDiagnosticProfile)
	}
	if output.Kind != "motion_mp4" {
		return false
	}
	if output.ProfileID == PhysicalRecordingMotionOutput {
		return hasRecording
	}
	return !hasRecording && (output.ProfileID == PhysicalMotionOutput || output.ProfileID == PhysicalMotionOutputV2)
}

func validatePhysicalMaterialRuntimeSupport(candidate scenes.SceneCandidate) error {
	identityUV := [6]float64{1, 0, 0, 0, 1, 0}
	for _, material := range candidate.Materials {
		for _, input := range material.Inputs {
			if input.Socket == scenes.SocketNormal && input.Coordinates == scenes.CoordinatesUVSet &&
				(input.AffineUV == nil || *input.AffineUV != identityUV) {
				return fmt.Errorf("normal material %q requires identity UV_SET_AFFINE_V1 so texture sampling and tangent basis agree", material.ID)
			}
		}
	}
	return nil
}

func validatePhysicalCandidate(s PhysicalCandidateSnapshot, kind string) error {
	if !identifierPattern.MatchString(s.ID) || s.Revision == 0 || len(s.CanonicalJSON) == 0 || len(s.CanonicalJSON) > physicalSnapshotMax || !physicalDigestPattern.MatchString(s.CanonicalSHA) {
		return fmt.Errorf("%s candidate snapshot identity is invalid", kind)
	}
	sum := sha256.Sum256(s.CanonicalJSON)
	if hex.EncodeToString(sum[:]) != s.CanonicalSHA {
		return fmt.Errorf("%s candidate snapshot digest does not match exact bytes", kind)
	}
	switch kind {
	case "device":
		value, err := decodeDevice(s.CanonicalJSON)
		if err != nil {
			return fmt.Errorf("decode device candidate: %w", err)
		}
		_, canonical, digest, err := devices.CanonicalizeDeviceCandidate(value)
		if err != nil || digest != s.CanonicalSHA || !bytes.Equal(canonical, s.CanonicalJSON) {
			return errors.New("device candidate bytes are not the exact canonical owner snapshot")
		}
	case "scene":
		value, err := decodeScene(s.CanonicalJSON)
		if err != nil {
			return fmt.Errorf("decode scene candidate: %w", err)
		}
		_, canonical, digest, err := scenes.CanonicalizeCandidate(value)
		if err != nil || digest != s.CanonicalSHA || !bytes.Equal(canonical, s.CanonicalJSON) {
			return errors.New("scene candidate bytes are not the exact canonical owner snapshot")
		}
	case "motion":
		value, err := decodeMotion(s.CanonicalJSON)
		if err != nil {
			return fmt.Errorf("decode motion candidate: %w", err)
		}
		_, canonical, digest, err := motions.CanonicalizeCandidate(value)
		if err != nil || digest != s.CanonicalSHA || !bytes.Equal(canonical, s.CanonicalJSON) {
			return errors.New("motion candidate bytes are not the exact canonical owner snapshot")
		}
	default:
		return errors.New("unknown physical candidate kind")
	}
	return nil
}

func decodeDevice(data []byte) (devices.DeviceCandidate, error) {
	var value devices.DeviceCandidate
	err := json.Unmarshal(data, &value)
	return value, err
}
func decodeScene(data []byte) (scenes.SceneCandidate, error) {
	var value scenes.SceneCandidate
	err := json.Unmarshal(data, &value)
	return value, err
}
func decodeMotion(data []byte) (motions.MotionCandidate, error) {
	var value motions.MotionCandidate
	err := json.Unmarshal(data, &value)
	return value, err
}

func decodeCanonicalDeviceSnapshot(data []byte) (devices.DeviceCandidate, string, error) {
	if len(data) == 0 || len(data) > physicalSnapshotMax {
		return devices.DeviceCandidate{}, "", errors.New("device candidate byte length is invalid")
	}
	value, err := decodeDevice(data)
	if err != nil {
		return devices.DeviceCandidate{}, "", err
	}
	canonical, encoded, digest, err := devices.CanonicalizeDeviceCandidate(value)
	if err != nil {
		return devices.DeviceCandidate{}, "", err
	}
	if !bytes.Equal(encoded, data) {
		return devices.DeviceCandidate{}, "", errors.New("device candidate JSON is not canonical")
	}
	return canonical, digest, nil
}

func decodeCanonicalSceneSnapshot(data []byte) (scenes.SceneCandidate, string, error) {
	if len(data) == 0 || len(data) > physicalSnapshotMax {
		return scenes.SceneCandidate{}, "", errors.New("scene candidate byte length is invalid")
	}
	value, err := decodeScene(data)
	if err != nil {
		return scenes.SceneCandidate{}, "", err
	}
	canonical, encoded, digest, err := scenes.CanonicalizeCandidate(value)
	if err != nil || !bytes.Equal(encoded, data) {
		return scenes.SceneCandidate{}, "", errors.New("scene candidate JSON is not exact canonical owner data")
	}
	return canonical, digest, nil
}

func decodeCanonicalMotionSnapshot(data []byte) (motions.MotionCandidate, string, error) {
	if len(data) == 0 || len(data) > physicalSnapshotMax {
		return motions.MotionCandidate{}, "", errors.New("motion candidate byte length is invalid")
	}
	value, err := decodeMotion(data)
	if err != nil {
		return motions.MotionCandidate{}, "", err
	}
	canonical, encoded, digest, err := motions.CanonicalizeCandidate(value)
	if err != nil || !bytes.Equal(encoded, data) {
		return motions.MotionCandidate{}, "", errors.New("motion candidate JSON is not exact canonical owner data")
	}
	return canonical, digest, nil
}

func validatePhysicalAssets(assets []PhysicalAssetInput) error {
	if len(assets) == 0 || len(assets) > physicalAssetsMax {
		return errors.New("physical render asset count is outside bounds")
	}
	seenName, seenBinding := map[string]bool{}, map[string]bool{}
	for _, a := range assets {
		if !identifierPattern.MatchString(a.BindingID) || seenBinding[a.BindingID] || !isSafePhysicalName(a.Name) || seenName[a.Name] ||
			!identifierPattern.MatchString(a.Reference.BundleID) || a.Reference.BundleRevision == 0 || !identifierPattern.MatchString(a.Reference.ArtifactID) ||
			a.Reference.SizeBytes <= 0 || !physicalDigestPattern.MatchString(a.Reference.ManifestSHA256) || !physicalDigestPattern.MatchString(a.Reference.ReviewSHA256) || !physicalDigestPattern.MatchString(a.Reference.SHA256) {
			return errors.New("physical asset binding is invalid or duplicated")
		}
		if (a.Role == "device_geometry" || a.Role == "scene_geometry") && a.Reference.Role != library.RoleGeometryGLB {
			return fmt.Errorf("physical asset %q must bind geometry_glb", a.BindingID)
		}
		if (a.Role == "device_geometry" || a.Role == "scene_geometry") && a.Reference.SizeBytes > library.MaxGeometryBytes {
			return fmt.Errorf("physical geometry asset %q exceeds the library geometry size limit", a.BindingID)
		}
		if a.Role == "scene_texture" && a.Reference.Role != library.RoleTextureImage {
			return fmt.Errorf("physical asset %q must bind texture_image", a.BindingID)
		}
		if a.Role == "scene_texture" && a.Reference.SizeBytes > library.MaxTextureBytes {
			return fmt.Errorf("physical texture asset %q exceeds the library texture size limit", a.BindingID)
		}
		if a.Name == PhysicalManifestName || a.Name == PhysicalReceiptName || a.Name == PhysicalAdapterName || a.Name == PhysicalStillOutputName || a.Name == PhysicalMotionOutputName || a.Name == "blender-profile.json" {
			return fmt.Errorf("physical asset name %q is reserved", a.Name)
		}
		if a.Role != "device_geometry" && a.Role != "scene_geometry" && a.Role != "scene_texture" {
			return fmt.Errorf("physical asset %q has unsupported role", a.BindingID)
		}
		seenName[a.Name], seenBinding[a.BindingID] = true, true
	}
	return nil
}

func validatePhysicalScreen(s PhysicalScreenInput) error {
	if !identifierPattern.MatchString(s.DisplayID) || !identifierPattern.MatchString(s.OriginalMediaID) || !physicalDigestPattern.MatchString(s.OriginalMediaSHA256) ||
		s.Width < 1 || s.Height < 1 || s.Width > 4096 || s.Height > 4096 ||
		!physicalDigestPattern.MatchString(s.ConfigSHA256) || !physicalDigestPattern.MatchString(s.LayoutSHA256) ||
		strings.TrimSpace(s.PresentationProfile) == "" || strings.TrimSpace(s.PresentationVersion) == "" || s.UVSet != "TEXCOORD_0" || s.MappingProfile != PhysicalScreenMappingProfile {
		return errors.New("physical screen binding is incomplete or invalid")
	}
	if s.Recording == nil {
		if !isSafePhysicalName(s.Raster.Name) || s.Raster.Name != "screen.png" || !physicalDigestPattern.MatchString(s.Raster.SHA256) || s.Raster.SizeBytes <= 0 || s.Raster.SizeBytes > physicalScreenBytesMax || s.Raster.SHA256 != s.RasterSHA256 {
			return errors.New("physical screen raster binding is incomplete or invalid")
		}
		return nil
	}
	if s.Raster != (ImageBinding{}) || s.RasterSHA256 != "" {
		return errors.New("physical recording screen must not bind a static raster")
	}
	if err := validatePhysicalRecording(s); err != nil {
		return err
	}
	return nil
}

func validatePhysicalRecording(screen PhysicalScreenInput) error {
	r := screen.Recording
	if r == nil {
		return errors.New("physical recording timing is missing")
	}
	i := r.Inspection
	if err := validateVideoInspection(i); err != nil {
		return err
	}
	if i.Manifest.SizeBytes > videoMaxManifestBytes || i.Receipt.SizeBytes > videoMaxReceiptBytes || i.Clock.SizeBytes > videoMaxClockBytes || i.Source.SizeBytes > physicalInputBytesMax {
		return errors.New("physical recording inspection artifacts exceed their fixed size limits")
	}
	if i.Source.SHA256 != screen.OriginalMediaSHA256 || i.Source.SizeBytes <= 0 ||
		r.TrimStartPTS < i.StreamStartPTS || r.TrimEndPTS <= r.TrimStartPTS || r.TrimEndPTS > i.SourceEndPTS ||
		r.PlaybackRate.Num <= 0 || r.PlaybackRate.Den <= 0 ||
		(r.AudioPolicy != PhysicalRecordingAudioUnspecified && r.AudioPolicy != PhysicalRecordingAudioSilent && r.AudioPolicy != PhysicalRecordingAudioPassthrough) {
		return errors.New("physical recording source clock, trim, rate, or audio policy is invalid")
	}
	if r.AudioPolicy == PhysicalRecordingAudioPassthrough {
		return errors.New("physical recording profile does not support source audio passthrough")
	}
	if r.SourcePrefixFrameCount == 0 || r.SourcePrefixFrameCount > PhysicalRecordingMaxSourcePrefix {
		return errors.New("physical recording source decode prefix exceeds its bounded profile")
	}
	if r.SourcePrefixFrameCount > i.FrameCount {
		return errors.New("physical recording source decode prefix exceeds the accepted inspection clock")
	}
	if r.Presentation.Target.X != screen.Width || r.Presentation.Target.Y != screen.Height {
		return errors.New("physical recording presentation target differs from the pinned screen raster")
	}
	configBytes, err := json.Marshal(r.Presentation)
	if err != nil || digest(configBytes) != screen.ConfigSHA256 {
		return errors.New("physical recording presentation config digest differs from the screen binding")
	}
	expectedLayout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, i.Width, i.Height), r.Presentation)
	if err != nil || !jsonValuesEqual(expectedLayout, r.PresentationLayout) {
		return errors.New("physical recording presentation layout differs from its pinned source raster")
	}
	layoutBytes, err := json.Marshal(r.PresentationLayout)
	if err != nil || digest(layoutBytes) != screen.LayoutSHA256 {
		return errors.New("physical recording presentation layout digest differs from the screen binding")
	}
	if r.PresentationLayout.ProfileID != screen.PresentationProfile || r.PresentationLayout.ProfileVersion != screen.PresentationVersion {
		return errors.New("physical recording presentation profile differs from the screen binding")
	}
	return nil
}

func validatePhysicalPlacement(p PhysicalPlacement) error {
	if !identifierPattern.MatchString(p.DeviceSupportID) || !identifierPattern.MatchString(p.SceneSupportID) || !identifierPattern.MatchString(p.SelectedCameraID) {
		return errors.New("physical placement must select exact device support, scene support, and camera IDs")
	}
	if _, err := p.SceneSupportWorld.Matrix(); err != nil {
		return fmt.Errorf("scene support transform: %w", err)
	}
	if _, err := p.DeviceRootRest.Matrix(); err != nil {
		return fmt.Errorf("device rest transform: %w", err)
	}
	if err := p.SupportPivot.Validate(); err != nil {
		return fmt.Errorf("support pivot: %w", err)
	}
	return nil
}

func validatePhysicalProfile(p PhysicalRenderProfile, output PhysicalOutputPlan) error {
	wallTime := PhysicalStillWallTimeSeconds
	wantSamples := PhysicalStillSamples
	if output.ProfileID == PhysicalStillProfile && output.Kind == "still_png" && p.Backend == "CPU" && p.WallTimeSeconds == PhysicalStillCPUWallTimeSeconds {
		wallTime = PhysicalStillCPUWallTimeSeconds
	}
	switch output.ProfileID {
	case PhysicalPreviewProfile:
		wallTime = PhysicalPreviewWallTimeSeconds
		wantSamples = PhysicalPreviewSamples
	case PhysicalDiagnosticProfile:
		wallTime = PhysicalDiagnosticWallTimeSeconds
		wantSamples = PhysicalDiagnosticSamples
	case PhysicalMotionOutput, PhysicalMotionOutputV2, PhysicalRecordingMotionOutput:
		wantSamples = PhysicalMotionSamples
	case PhysicalDiagnosticMotionOutput:
		wallTime = PhysicalDiagnosticMotionWallTimeSeconds
		wantSamples = PhysicalDiagnosticMotionSamples
	}
	if output.Kind == "motion_mp4" {
		switch output.ProfileID {
		case PhysicalMotionOutput:
			wallTime = PhysicalMotionWallTimeSeconds
		case PhysicalMotionOutputV2:
			wallTime = PhysicalMotionV2WallTimeSeconds
		case PhysicalRecordingMotionOutput:
			wallTime = PhysicalRecordingWallTimeSeconds
		case PhysicalDiagnosticMotionOutput:
			wallTime = PhysicalDiagnosticMotionWallTimeSeconds
		default:
			return errors.New("physical motion output profile is unsupported")
		}
	}
	if p.ID != PhysicalRendererProfile || p.Version != 1 || !physicalDigestPattern.MatchString(p.DigestSHA256) ||
		p.BlenderArtifactID != PhysicalBlenderArtifact || p.BlenderTreeSHA256 != PhysicalBlenderTreeSHA ||
		p.BlenderBinarySHA256 != PhysicalBlenderBinarySHA || p.BlenderVersion != PhysicalBlenderVersion || !physicalDigestPattern.MatchString(p.AdapterSHA256) ||
		(p.Backend != "CPU" && p.Backend != "CUDA" && p.Backend != "OPTIX") || p.Engine != "CYCLES" ||
		p.Samples != wantSamples || p.Seed != 1307 || p.AdaptiveSampling || p.Denoise || p.ColorManagement != PhysicalColorProfile ||
		p.ColorView != "Standard" || p.ColorLook != "None" || p.Exposure != "0" || p.Gamma != "1" ||
		p.WallTimeSeconds != wallTime || p.MemoryMaxBytes != PhysicalOperationMemoryMaxBytes ||
		p.TasksMax != PhysicalOperationTasksMax || p.CPUWeight != PhysicalOperationCPUWeight || p.Threads != PhysicalOperationThreads {
		return errors.New("physical renderer profile is incomplete or outside the pinned profile")
	}
	actual, err := PhysicalRenderProfileDigest(p)
	if err != nil || actual != p.DigestSHA256 {
		return errors.New("physical renderer profile digest does not bind its exact quality and tool settings")
	}
	return nil
}

// PhysicalRenderProfileDigest hashes all profile fields except DigestSHA256.
func PhysicalRenderProfileDigest(profile PhysicalRenderProfile) (string, error) {
	profile.DigestSHA256 = ""
	data, err := json.Marshal(profile)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// NewPhysicalPreviewOutputPlan returns the only supported low-resolution
// physical preview plan. It reuses the saved still's exact pose sample; callers
// cannot select an arbitrary raster or sampling quality.
func NewPhysicalPreviewOutputPlan(sample PhysicalMotionSample, timebase composition.Rational) PhysicalOutputPlan {
	// A preview is a still image at the selected pose. It carries motion ticks
	// for reproducibility, but no output-time coordinate; preserving the source
	// motion frame's 0/1 output time here makes it fail the still manifest's
	// canonical zero-value invariant (0/0).
	sample.OutputOrdinal = 0
	sample.OutputTime = composition.Rational{}
	return PhysicalOutputPlan{
		ProfileID: PhysicalPreviewProfile, ProfileVersion: PhysicalPreviewProfileVersion, Kind: "still_png",
		Width: 960, Height: 540, Rate: composition.Rational{Num: 1, Den: 1}, FrameCount: 1,
		Timebase: timebase, FinalPoseHold: "none", Samples: []PhysicalMotionSample{sample},
		OutputName: PhysicalPreviewOutputName, Codec: "png", PixelFormat: "rgba8",
	}
}

// NewPhysicalDiagnosticOutputPlan returns a low-sample diagnostic still. It is
// intended for wiring/backend checks and carries no delivery-quality claim.
func NewPhysicalDiagnosticOutputPlan(sample PhysicalMotionSample, timebase composition.Rational) PhysicalOutputPlan {
	sample.OutputOrdinal = 0
	sample.OutputTime = composition.Rational{}
	return PhysicalOutputPlan{
		ProfileID: PhysicalDiagnosticProfile, ProfileVersion: PhysicalDiagnosticProfileVersion, Kind: "still_png",
		Width: 960, Height: 540, Rate: composition.Rational{Num: 1, Den: 1}, FrameCount: 1,
		Timebase: timebase, FinalPoseHold: "none", Samples: []PhysicalMotionSample{sample},
		OutputName: PhysicalDiagnosticOutputName, Codec: "png", PixelFormat: "rgba8",
	}
}

func validatePhysicalOutput(o PhysicalOutputPlan) error {
	if o.Rate.Num <= 0 || o.Rate.Den <= 0 || o.Timebase.Num <= 0 || o.Timebase.Den <= 0 || o.FrameCount == 0 || o.FrameCount > physicalFrameMax {
		return errors.New("physical output schedule is outside bounds")
	}
	if o.ArticulationProfile != "" && o.ArticulationProfile != PhysicalArticulationProfile {
		return errors.New("physical articulation profile is unsupported")
	}
	if !isSafePhysicalName(o.OutputName) {
		return errors.New("physical output name is invalid")
	}
	switch o.Kind {
	case "still_png":
		fullStill := o.ProfileID == PhysicalStillProfile && o.ProfileVersion == 0 &&
			((o.Width == 2560 && o.Height == 1440) || (o.Width == 3840 && o.Height == 2160)) && o.OutputName == PhysicalStillOutputName
		preview := o.ProfileID == PhysicalPreviewProfile && o.ProfileVersion == PhysicalPreviewProfileVersion &&
			o.Width == 960 && o.Height == 540 && o.OutputName == PhysicalPreviewOutputName
		diagnostic := o.ProfileID == PhysicalDiagnosticProfile && o.ProfileVersion == PhysicalDiagnosticProfileVersion &&
			o.Width == 960 && o.Height == 540 && o.OutputName == PhysicalDiagnosticOutputName
		if (!fullStill && !preview && !diagnostic) || o.VideoToolProfileID != "" || o.VideoToolArtifactID != "" || o.MotionProfile != "" || o.MotionTicks != 0 || len(o.Samples) != 1 || o.FinalPoseHold != "none" || o.FrameCount != 1 || o.Codec != "png" || o.PixelFormat != "rgba8" {
			return errors.New("physical still output does not match the still profile")
		}
		if o.Samples[0].OutputOrdinal != 0 || o.Samples[0].OutputTime != (composition.Rational{}) || o.Samples[0].RequestedMotionTick.Den <= 0 || o.Samples[0].RequestedMotionTick.Num < 0 || !rationalEqual(o.Samples[0].RequestedMotionTick, o.Samples[0].SampledMotionTick) {
			return errors.New("physical still pose sample is missing or inconsistent")
		}
	case "motion_mp4":
		recording := o.ProfileID == PhysicalRecordingMotionOutput && o.ProfileVersion == 1 && o.MotionProfile == PhysicalRecordingMotionProfile
		diagnosticRecording := o.ProfileID == PhysicalDiagnosticMotionOutput && o.ProfileVersion == 1 && o.MotionProfile == PhysicalDiagnosticMotionProfile
		motionV1 := o.ProfileID == PhysicalMotionOutput && o.ProfileVersion == 1 && o.MotionProfile == PhysicalMotionProfile
		motionV2 := o.ProfileID == PhysicalMotionOutputV2 && o.ProfileVersion == 2 && o.MotionProfile == PhysicalMotionProfileV2
		if (!motionV1 && !motionV2 && !recording && !diagnosticRecording) ||
			o.VideoToolProfileID != PhysicalMotionVideoProfileID || o.VideoToolArtifactID != PhysicalMotionVideoArtifactID ||
			(o.Width != 1920 || o.Height != 1080) && (o.Width != 1080 || o.Height != 1920) || o.OutputName != PhysicalMotionOutputName {
			return errors.New("physical motion output must use a pinned 1080p raster")
		}
		maxFrames := PhysicalMotionMaxFramesV1
		if motionV2 || recording {
			maxFrames = PhysicalMotionMaxFramesV2
		}
		if diagnosticRecording {
			maxFrames = PhysicalDiagnosticMotionMaxFrames
		}
		if o.Rate.Num != 24 && o.Rate.Num != 30 || o.Rate.Den != 1 || o.MotionTicks == 0 || o.FrameCount > maxFrames ||
			o.FrameCount > uint32(o.Rate.Num)*10 || o.FrameCount != uint32(len(o.Samples)) || o.FinalPoseHold != "last-key-through-output-end-v1" || o.Codec != "h264" || o.PixelFormat != "yuv420p" || o.ColorPolicy != "sdr-unmarked-default709-v1" {
			return errors.New("physical motion schedule is not supported by the current profile")
		}
		for i, sample := range o.Samples {
			outputNumerator := new(big.Int).Mul(big.NewInt(int64(i)), big.NewInt(o.Rate.Den))
			outputTime := new(big.Rat).SetFrac(outputNumerator, big.NewInt(o.Rate.Num))
			requestedTick := new(big.Rat).Mul(outputTime, new(big.Rat).SetFrac(big.NewInt(o.Timebase.Num), big.NewInt(o.Timebase.Den)))
			sampledTick := new(big.Rat).Set(requestedTick)
			if sampledTick.Cmp(new(big.Rat).SetInt64(int64(o.MotionTicks))) > 0 {
				sampledTick.SetInt64(int64(o.MotionTicks))
			}
			expectedOutput, err := rationalFromBig(outputTime)
			if err != nil || sample.OutputOrdinal != uint32(i) || !rationalEqual(sample.OutputTime, expectedOutput) {
				return errors.New("physical output ordinal or exact output timestamp is inconsistent")
			}
			expectedRequest, err := rationalFromBig(requestedTick)
			if err != nil || !rationalEqual(sample.RequestedMotionTick, expectedRequest) {
				return errors.New("physical requested motion tick differs from the exact output clock")
			}
			expectedSample, err := rationalFromBig(sampledTick)
			if err != nil || !rationalEqual(sample.SampledMotionTick, expectedSample) {
				return errors.New("physical sampled motion tick differs from the explicit final-pose hold")
			}
		}
	default:
		return errors.New("physical output kind is unsupported")
	}
	if o.Width > 4096 || o.Height > 4096 {
		return errors.New("physical raster exceeds profile dimensions")
	}
	return nil
}

func validateCandidateAssetBindings(device devices.DeviceCandidate, scene scenes.SceneCandidate, assets []PhysicalAssetInput) error {
	byBinding := make(map[string]PhysicalAssetInput, len(assets))
	for _, asset := range assets {
		byBinding[asset.BindingID] = asset
	}
	deviceAsset, ok := byBinding["device_geometry"]
	if !ok || deviceAsset.Role != "device_geometry" || deviceAsset.Reference != device.Geometry {
		return errors.New("device geometry input does not match the immutable device candidate reference")
	}
	for _, artifact := range scene.Artifacts {
		asset, exists := byBinding[artifact.ID]
		if !exists || asset.Reference != artifact.Ref {
			return fmt.Errorf("scene artifact input %q does not match the immutable candidate reference", artifact.ID)
		}
		if artifact.Ref.Role == library.RoleGeometryGLB && asset.Role != "scene_geometry" || artifact.Ref.Role == library.RoleTextureImage && asset.Role != "scene_texture" {
			return fmt.Errorf("scene artifact input %q has the wrong renderer role", artifact.ID)
		}
	}
	if len(byBinding) != len(scene.Artifacts)+1 {
		return errors.New("physical assets contain an unreferenced or missing binding")
	}
	return nil
}

func validateSelectedFrames(device devices.DeviceCandidate, scene scenes.SceneCandidate, displayID string, placement PhysicalPlacement) error {
	displayExists, displayBound := false, false
	for _, display := range device.Definition.Displays {
		if display.ID == displayID {
			displayExists = true
		}
	}
	for _, binding := range device.DisplayBindings {
		if binding.DisplayID == displayID {
			displayBound = true
		}
	}
	if !displayExists || !displayBound {
		return errors.New("screen display ID is not bound by the immutable device candidate")
	}
	deviceSupport, sceneSupport, camera := false, false, false
	for _, support := range device.SupportFrames {
		deviceSupport = deviceSupport || support.ID == placement.DeviceSupportID
	}
	for _, support := range scene.SupportFrames {
		sceneSupport = sceneSupport || support.ID == placement.SceneSupportID
	}
	for _, frame := range scene.CameraFrames {
		camera = camera || frame.ID == placement.SelectedCameraID
	}
	if !deviceSupport || !sceneSupport || !camera {
		return errors.New("selected device support, scene support, or camera ID is not present in the candidate snapshots")
	}
	return nil
}

func validateMotionOutput(plan PhysicalOutputPlan, motion motions.MotionCandidate) error {
	if plan.Timebase != motion.Timebase || motion.DurationTicks <= 0 || uint64(motion.DurationTicks) > uint64(^uint32(0)) {
		return errors.New("physical output timing does not match the immutable motion candidate timebase and duration")
	}
	if plan.Kind == "motion_mp4" && plan.MotionTicks != uint32(motion.DurationTicks) {
		return errors.New("physical output timing does not match the immutable motion candidate timebase and duration")
	}
	targets := make(map[string]bool, len(motion.Tracks))
	for _, track := range motion.Tracks {
		switch {
		case track.Target.DeviceRoot:
			targets["device_root"] = true
		case track.Target.JointID != "":
			targets["joint:"+track.Target.JointID] = true
		case track.Target.SceneCameraID != "":
			targets["camera:"+track.Target.SceneCameraID] = true
		}
	}
	endpointCameraID := ""
	if plan.CameraEndpoint != nil {
		endpointCameraID = plan.CameraEndpoint.CameraID
		if targets["camera:"+endpointCameraID] {
			return errors.New("physical camera endpoint duplicates an authored selected-camera motion track")
		}
	}
	for _, sample := range plan.Samples {
		if sample.DeviceRoot != nil {
			if !targets["device_root"] {
				return errors.New("motion sample supplies an unrequested device-root transform")
			}
			if _, err := sample.DeviceRoot.Matrix(); err != nil {
				return fmt.Errorf("sampled device-root transform: %w", err)
			}
		}
		for id, value := range sample.JointAngles {
			if !targets["joint:"+id] || math.IsNaN(value) || math.IsInf(value, 0) {
				return errors.New("motion sample supplies an invalid or unrequested joint angle")
			}
		}
		for id, transform := range sample.CameraTransforms {
			if id != endpointCameraID && !targets["camera:"+id] {
				return errors.New("motion sample supplies an unrequested camera transform")
			}
			if _, err := transform.Matrix(); err != nil {
				return fmt.Errorf("sampled camera transform: %w", err)
			}
		}
		maxCameraTargets := len(targets)
		if endpointCameraID != "" {
			maxCameraTargets++
		}
		if len(sample.JointAngles) > len(targets) || len(sample.CameraTransforms) > maxCameraTargets {
			return errors.New("motion sample contains unexpected target data")
		}
		if err := comparePhysicalSample(motion, sample, targets); err != nil {
			return err
		}
	}
	if len(plan.Samples) > 0 {
		last := plan.Samples[len(plan.Samples)-1]
		for target := range targets {
			switch {
			case target == "device_root" && last.DeviceRoot == nil:
				return errors.New("motion sample omits the final device-root pose")
			case strings.HasPrefix(target, "joint:"):
				if _, ok := last.JointAngles[strings.TrimPrefix(target, "joint:")]; !ok {
					return errors.New("motion sample omits a final joint pose")
				}
			case strings.HasPrefix(target, "camera:"):
				if _, ok := last.CameraTransforms[strings.TrimPrefix(target, "camera:")]; !ok {
					return errors.New("motion sample omits a final camera pose")
				}
			}
		}
	}
	return nil
}

func comparePhysicalSample(motion motions.MotionCandidate, sample PhysicalMotionSample, targets map[string]bool) error {
	actual, err := motions.SampleAt(motion, sample.SampledMotionTick)
	if err != nil {
		return fmt.Errorf("sample motion candidate at tick %v: %w", sample.SampledMotionTick, err)
	}
	if len(actual.Tracks) != len(targets) {
		return errors.New("sampled motion track count differs from candidate")
	}
	for _, track := range actual.Tracks {
		switch {
		case track.Target.DeviceRoot:
			want := physical.TRS{Rotation: track.Transform.Rotation, Translation: track.Transform.Translation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
			if sample.DeviceRoot == nil || *sample.DeviceRoot != want {
				return errors.New("manifest device-root sample differs from motions.SampleAt")
			}
		case track.Target.JointID != "":
			value, ok := sample.JointAngles[track.Target.JointID]
			if !ok || track.AngleRad == nil || value != *track.AngleRad {
				return errors.New("manifest joint sample differs from motions.SampleAt")
			}
		case track.Target.SceneCameraID != "":
			value, ok := sample.CameraTransforms[track.Target.SceneCameraID]
			if !ok || track.Transform == nil || value != (physical.TRS{Rotation: track.Transform.Rotation, Translation: track.Transform.Translation, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}) {
				return errors.New("manifest camera sample differs from motions.SampleAt")
			}
		}
	}
	return nil
}

func rationalFromBig(value *big.Rat) (composition.Rational, error) {
	if value == nil || !value.Num().IsInt64() || !value.Denom().IsInt64() || value.Denom().Sign() <= 0 {
		return composition.Rational{}, errors.New("physical clock value exceeds rational representation")
	}
	return composition.Rational{Num: value.Num().Int64(), Den: value.Denom().Int64()}, nil
}

func rationalEqual(left, right composition.Rational) bool {
	if left.Den <= 0 || right.Den <= 0 {
		return false
	}
	return new(big.Rat).SetFrac(big.NewInt(left.Num), big.NewInt(left.Den)).Cmp(new(big.Rat).SetFrac(big.NewInt(right.Num), big.NewInt(right.Den))) == 0
}

func decodeSceneID(data []byte) string {
	value, err := decodeScene(data)
	if err != nil {
		return ""
	}
	return value.SceneID
}

func decodeMotionID(data []byte) string {
	value, err := decodeMotion(data)
	if err != nil {
		return ""
	}
	return value.MotionID
}

func isSafePhysicalName(name string) bool {
	return name != "" && filepath.Base(name) == name && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}
