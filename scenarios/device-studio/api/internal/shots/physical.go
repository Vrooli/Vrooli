package shots

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"math/big"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/presentations"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

// PhysicalResolver resolves exact candidate revisions and current library
// render rights. It performs no qualification or GLB-node-existence claim.
type PhysicalResolver interface {
	Resolve(context.Context, string, *shotsv1.PhysicalShotBinding, *shotsv1.ScreenPresentation, *shotsv1.VideoTiming, string) (*PhysicalResolution, error)
}

// ValidatePhysicalSetup is the no-source admission seam used by immutable
// setup presets. It checks component compatibility and current render rights;
// it does not claim GLB node resolution or renderer qualification.
func (r *ownedPhysicalResolver) ValidatePhysicalSetup(ctx context.Context, setup *shotsv1.PhysicalShotSetup) error {
	if setup == nil || setup.GetDevice() == nil || setup.GetScene() == nil || setup.GetMotion() == nil {
		return ErrInvalidPhysicalBinding{Reason: "device, scene, and motion revision references are required"}
	}
	deviceRevision, err := r.devices.GetRevision(ctx, setup.Device.Id, setup.Device.Revision)
	if err != nil {
		return ErrInvalidPhysicalBinding{Reason: "selected device revision is unavailable"}
	}
	sceneRevision, err := r.scenes.GetRevision(ctx, setup.Scene.Id, setup.Scene.Revision)
	if err != nil {
		return ErrInvalidPhysicalBinding{Reason: "selected scene revision is unavailable"}
	}
	motionRevision, err := r.motions.GetRevision(ctx, setup.Motion.Id, setup.Motion.Revision)
	if err != nil {
		return ErrInvalidPhysicalBinding{Reason: "selected motion revision is unavailable"}
	}
	if deviceRevision.SHA256 != setup.Device.Sha256 || sceneRevision.SHA256 != setup.Scene.Sha256 || motionRevision.SHA256 != setup.Motion.Sha256 {
		return ErrInvalidPhysicalBinding{Reason: "selected component digest does not match its immutable revision"}
	}
	deviceCandidate, _, deviceSHA, err := devices.CanonicalizeDeviceCandidate(deviceRevision.Candidate)
	if err != nil || deviceSHA != deviceRevision.SHA256 {
		return ErrInvalidPhysicalBinding{Reason: "selected device candidate is not canonical"}
	}
	sceneCandidate, _, sceneSHA, err := scenes.CanonicalizeCandidate(sceneRevision.Candidate)
	if err != nil || sceneSHA != sceneRevision.SHA256 {
		return ErrInvalidPhysicalBinding{Reason: "selected scene candidate is not canonical"}
	}
	motionCandidate, _, motionSHA, err := motions.CanonicalizeCandidate(motionRevision.Candidate)
	if err != nil || motionSHA != motionRevision.SHA256 {
		return ErrInvalidPhysicalBinding{Reason: "selected motion candidate is not canonical"}
	}
	binding := &shotsv1.PhysicalShotBinding{
		Device: proto.Clone(setup.Device).(*shotsv1.ComponentRevisionRef), Scene: proto.Clone(setup.Scene).(*shotsv1.ComponentRevisionRef), Motion: proto.Clone(setup.Motion).(*shotsv1.ComponentRevisionRef),
		DisplayId: setup.DisplayId, DeviceSupportId: setup.DeviceSupportId, SceneSupportId: setup.SceneSupportId, CameraId: setup.CameraId,
		SceneSupportWorld: proto.Clone(setup.SceneSupportWorld).(*physicalv1.TRS), SupportPivotM: proto.Clone(setup.SupportPivotM).(*physicalv1.Vec3), DeviceRootRest: proto.Clone(setup.DeviceRootRest).(*physicalv1.TRS),
	}
	if err := validatePhysicalSelection(binding, deviceCandidate, sceneCandidate, motionCandidate); err != nil {
		return ErrInvalidPhysicalBinding{Reason: err.Error()}
	}
	assets := []physical.ArtifactRef{deviceCandidate.Geometry}
	for _, asset := range sceneCandidate.Artifacts {
		assets = append(assets, asset.Ref)
	}
	for _, ref := range assets {
		resolved, resolveErr := r.library.Resolve(ctx, ref, library.UseRender)
		if resolveErr != nil {
			return ErrInvalidPhysicalBinding{Reason: "a selected asset lacks current render rights or exact reviewed bytes"}
		}
		if _, readErr := io.Copy(io.Discard, resolved.Reader); readErr != nil {
			_ = resolved.Reader.Close()
			return fmt.Errorf("verify physical preset artifact bytes: %w", readErr)
		}
		if closeErr := resolved.Reader.Close(); closeErr != nil {
			return fmt.Errorf("close verified preset artifact: %w", closeErr)
		}
	}
	return nil
}

type PhysicalResolution struct {
	DeviceID, SceneID, MotionID                                  string
	DeviceRevision, SceneRevision, MotionRevision                uint32
	DeviceSHA256, SceneSHA256, MotionSHA256                      string
	DeviceCandidate                                              devices.DeviceCandidate
	SceneCandidate                                               scenes.SceneCandidate
	MotionCandidate                                              motions.MotionCandidate
	DeviceCanonicalJSON, SceneCanonicalJSON, MotionCanonicalJSON []byte
	Assets                                                       []PhysicalAssetBinding
	Binding                                                      *shotsv1.PhysicalShotBinding
	ScreenMedia                                                  projects.SourceMedia
	ScreenRasterPNG                                              []byte
	PresentationProfile                                          string
	PresentationVersion                                          string
	ConfigSHA256, LayoutSHA256, RasterSHA256                     string
	MotionSample                                                 motions.MotionSample
	RecordingConfig                                              *presentations.Config
	RecordingLayout                                              *presentations.Layout
	OutputWidth, OutputHeight                                    uint32
}

type PhysicalAssetBinding struct {
	BindingID string
	Role      string
	Reference physical.ArtifactRef
}

type ownedPhysicalResolver struct {
	devices *devices.DeviceRevisionService
	scenes  *scenes.RevisionService
	motions *motions.RevisionService
	library physical.LibraryArtifactResolver
	media   projects.SourceMediaService
}

func (s *service) buildPhysicalRevision(ctx context.Context, projectID, shotID string, revision int64, name string, snapshot *shotsv1.ShotSnapshot, createdAt time.Time, presentationVersion string) (Revision, error) {
	if snapshot.GetScene() != nil {
		return Revision{}, ErrInvalidPhysicalBinding{Reason: "physical shots cannot also contain a photographic scene"}
	}
	if snapshot.GetCameraMotion() != nil || snapshot.GetMotionMode() == shotsv1.MotionMode_MOTION_MODE_CROP_PUSH {
		return Revision{}, ErrInvalidPhysicalBinding{Reason: "physical shots do not accept photographic camera motion"}
	}
	if snapshot.GetMotionMode() != shotsv1.MotionMode_MOTION_MODE_UNSPECIFIED && snapshot.GetMotionMode() != shotsv1.MotionMode_MOTION_MODE_FIXED {
		return Revision{}, ErrInvalidPhysicalBinding{Reason: "physical still shots require fixed photographic motion mode"}
	}
	if snapshot.GetVideoTiming() != nil && snapshot.GetPhysicalBinding().GetMotionOutput() == nil {
		return Revision{}, ErrInvalidPhysicalBinding{Reason: "physical recording shots require a pinned final MP4 output schedule"}
	}
	if timing := snapshot.GetVideoTiming(); timing != nil && !physicalRecordingRateMatches(timing, snapshot.GetPhysicalBinding().GetMotionOutput()) {
		return Revision{}, ErrInvalidPhysicalBinding{Reason: "recording output rate must match the pinned physical motion output rate"}
	}
	if snapshot.GetPresentation() == nil {
		return Revision{}, ErrInvalidPhysicalBinding{Reason: "a screen presentation is required"}
	}
	if s.physical == nil {
		return Revision{}, errors.New("physical shot resolver is unavailable")
	}
	resolved, err := s.physical.Resolve(ctx, projectID, snapshot.GetPhysicalBinding(), snapshot.GetPresentation(), snapshot.GetVideoTiming(), presentationVersion)
	if err != nil {
		return Revision{}, err
	}
	canonicalBytes, err := json.Marshal(snapshot)
	if err != nil {
		return Revision{}, fmt.Errorf("canonicalize physical shot snapshot: %w", err)
	}
	snapshotHash := sha256.Sum256(canonicalBytes)
	presentation := proto.Clone(snapshot.GetPresentation()).(*shotsv1.ScreenPresentation)
	presentation.SourceMediaId = ""
	presentationIdentity, err := json.Marshal(struct {
		Version      string
		Presentation *shotsv1.ScreenPresentation
		Screen       struct {
			SHA256        string
			SizeBytes     int64
			Width, Height uint32
		}
	}{presentationVersion, presentation, struct {
		SHA256        string
		SizeBytes     int64
		Width, Height uint32
	}{resolved.ScreenMedia.SHA256, resolved.ScreenMedia.SizeBytes, uint32(resolved.ScreenMedia.Width), uint32(resolved.ScreenMedia.Height)}})
	if err != nil {
		return Revision{}, fmt.Errorf("canonicalize physical presentation identity: %w", err)
	}
	presentationHash := sha256.Sum256(presentationIdentity)
	identityBytes, err := json.Marshal(struct {
		Version             string
		Name                string
		PresentationVersion string
		SnapshotSHA         string
		SourceSHA256        string
		SourceSizeBytes     int64
		ConfigSHA256        string
		LayoutSHA256        string
		RasterSHA256        string
		OutputWidth         uint32
		OutputHeight        uint32
	}{"device-studio-physical-shot-v1", name, presentationVersion, hex.EncodeToString(snapshotHash[:]), resolved.ScreenMedia.SHA256, resolved.ScreenMedia.SizeBytes, resolved.ConfigSHA256, resolved.LayoutSHA256, resolved.RasterSHA256, resolved.OutputWidth, resolved.OutputHeight})
	if err != nil {
		return Revision{}, err
	}
	recipeHash := sha256.Sum256(identityBytes)
	var issues []*shotsv1.ShotIssue
	if timing := snapshot.GetVideoTiming(); timing != nil && timing.GetAudioPolicy() == shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_PASSTHROUGH {
		issues = append(issues, &shotsv1.ShotIssue{Code: shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_AUDIO_POLICY, FieldPath: "video_timing.audio_policy", Summary: "Source audio passthrough is not supported by this renderer.", Repair: "Select silent output. The original recording retains its audio.", BlocksRender: true})
	}
	mediaType := resolved.ScreenMedia.MediaType
	if mediaType == "" {
		mediaType = "image/png"
		if snapshot.GetVideoTiming() != nil {
			mediaType = "video/mp4"
		}
	}
	return Revision{
		ShotID: shotID, Revision: revision, CreatedAt: createdAt.UTC(), Name: name, Snapshot: snapshot,
		RecipeSHA256: hex.EncodeToString(recipeHash[:]), SceneSHA256: resolved.SceneSHA256,
		PresentationSHA256: hex.EncodeToString(presentationHash[:]), SceneVersion: "device-studio-physical-scene-v1", PresentationVersion: presentationVersion,
		CompositorProfile: resolved.PresentationProfile, Issues: issues, Screen: &shotsv1.ResolvedSourceMedia{
			Id: resolved.ScreenMedia.ID, Sha256: resolved.ScreenMedia.SHA256,
			SizeBytes: resolved.ScreenMedia.SizeBytes, MediaType: mediaType,
			WidthPixels: resolved.Binding.GetScreenWidthPixels(), HeightPixels: resolved.Binding.GetScreenHeightPixels(),
		}, RenderEligible: len(issues) == 0, Physical: resolved,
	}, nil
}

func physicalRecordingRateMatches(timing *shotsv1.VideoTiming, output *shotsv1.PhysicalMotionOutputBinding) bool {
	return timing != nil && output != nil && timing.GetOutputRateNum() == output.GetOutputRateNumerator() && timing.GetOutputRateDen() == output.GetOutputRateDenominator()
}

func physicalOutputDimensions(profile shotsv1.PhysicalStillOutputProfile) (uint32, uint32) {
	switch profile {
	case shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1:
		return 2560, 1440
	case shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1:
		return 3840, 2160
	default:
		return 0, 0
	}
}

func NewPhysicalResolver(deviceService *devices.DeviceRevisionService, sceneService *scenes.RevisionService, motionService *motions.RevisionService, libraryResolver physical.LibraryArtifactResolver, media projects.SourceMediaService) (PhysicalResolver, error) {
	if deviceService == nil || sceneService == nil || motionService == nil || media == nil {
		return nil, errors.New("physical shot resolution requires device, scene, motion, and project media owners")
	}
	return &ownedPhysicalResolver{devices: deviceService, scenes: sceneService, motions: motionService, library: libraryResolver, media: media}, nil
}

func (r *ownedPhysicalResolver) Resolve(ctx context.Context, projectID string, binding *shotsv1.PhysicalShotBinding, presentation *shotsv1.ScreenPresentation, timing *shotsv1.VideoTiming, presentationVersion string) (*PhysicalResolution, error) {
	if binding == nil || binding.GetDevice() == nil || binding.GetScene() == nil || binding.GetMotion() == nil {
		return nil, ErrInvalidPhysicalBinding{Reason: "device, scene, and motion revision references are required"}
	}
	if !validComponentRef(binding.GetDevice()) || !validComponentRef(binding.GetScene()) || !validComponentRef(binding.GetMotion()) {
		return nil, ErrInvalidPhysicalBinding{Reason: "component revision references require stable IDs, positive revisions, and SHA-256"}
	}
	deviceRevision, err := r.devices.GetRevision(ctx, binding.GetDevice().GetId(), binding.GetDevice().GetRevision())
	if err != nil {
		return nil, fmt.Errorf("resolve device revision: %w", err)
	}
	sceneRevision, err := r.scenes.GetRevision(ctx, binding.GetScene().GetId(), binding.GetScene().GetRevision())
	if err != nil {
		return nil, fmt.Errorf("resolve scene revision: %w", err)
	}
	motionRevision, err := r.motions.GetRevision(ctx, binding.GetMotion().GetId(), binding.GetMotion().GetRevision())
	if err != nil {
		return nil, fmt.Errorf("resolve motion revision: %w", err)
	}
	if deviceRevision.SHA256 != binding.GetDevice().GetSha256() || sceneRevision.SHA256 != binding.GetScene().GetSha256() || motionRevision.SHA256 != binding.GetMotion().GetSha256() {
		return nil, ErrInvalidPhysicalBinding{Reason: "component digest no longer matches the selected immutable revision"}
	}
	deviceCandidate, deviceJSON, deviceSHA, err := devices.CanonicalizeDeviceCandidate(deviceRevision.Candidate)
	if err != nil {
		return nil, fmt.Errorf("verify device revision canonical bytes: %w", err)
	}
	if deviceSHA != deviceRevision.SHA256 {
		return nil, errors.New("device revision canonical digest mismatch")
	}
	sceneCandidate, sceneJSON, sceneSHA, err := scenes.CanonicalizeCandidate(sceneRevision.Candidate)
	if err != nil {
		return nil, fmt.Errorf("verify scene revision canonical bytes: %w", err)
	}
	if sceneSHA != sceneRevision.SHA256 {
		return nil, errors.New("scene revision canonical digest mismatch")
	}
	motionCandidate, motionJSON, motionSHA, err := motions.CanonicalizeCandidate(motionRevision.Candidate)
	if err != nil {
		return nil, fmt.Errorf("verify motion revision canonical bytes: %w", err)
	}
	if motionSHA != motionRevision.SHA256 {
		return nil, errors.New("motion revision canonical digest mismatch")
	}
	if err := validatePhysicalSelection(binding, deviceCandidate, sceneCandidate, motionCandidate); err != nil {
		return nil, ErrInvalidPhysicalBinding{Reason: err.Error()}
	}
	if err := validatePhysicalArticulationMode(binding, timing, motionCandidate); err != nil {
		return nil, ErrInvalidPhysicalBinding{Reason: err.Error()}
	}
	if err := validatePhysicalProfileSelection(binding); err != nil {
		return nil, ErrInvalidPhysicalBinding{Reason: err.Error()}
	}
	if binding.GetScreenWidthPixels() == 0 || binding.GetScreenHeightPixels() == 0 {
		return nil, ErrInvalidPhysicalBinding{Reason: "selected display raster dimensions are required"}
	}
	if presentation == nil || presentation.GetSourceMediaId() == "" {
		return nil, ErrInvalidPhysicalBinding{Reason: "a project-owned screen source is required"}
	}
	media, err := r.media.Get(ctx, projectID, presentation.GetSourceMediaId())
	if err != nil {
		return nil, fmt.Errorf("resolve physical screen source: %w", err)
	}
	var screenPNG []byte
	var configSHA, layoutSHA, rasterSHA string
	var recordingConfig *presentations.Config
	var recordingLayout *presentations.Layout
	if timing == nil {
		if media.Kind != projects.MediaKindImage || media.Readiness != projects.ReadinessReady || !media.RasterKnown || media.Width == 0 || media.Height == 0 || media.SizeBytes <= 0 || media.SizeBytes > projects.MaxSourceImageBytes {
			return nil, ErrInvalidPhysicalBinding{Reason: "physical screen source must be a ready bounded still image"}
		}
		if err := validatePhysicalPresentation(presentation, binding, presentationVersion); err != nil {
			return nil, ErrInvalidPhysicalBinding{Reason: err.Error()}
		}
		screenPNG, configSHA, layoutSHA, rasterSHA, err = r.composePhysicalScreen(ctx, projectID, media, presentation, presentationVersion, int(binding.GetScreenWidthPixels()), int(binding.GetScreenHeightPixels()))
		if err != nil {
			return nil, err
		}
	} else {
		if media.Kind != projects.MediaKindVideo || media.Readiness != projects.ReadinessReady || media.SizeBytes <= 0 || media.SizeBytes > projects.MaxSourceVideoBytes || media.SHA256 != timing.GetSourceSha256() || media.SizeBytes != timing.GetSourceSizeBytes() {
			return nil, ErrInvalidPhysicalBinding{Reason: "physical recording source must match a ready bounded video and its saved digest"}
		}
		if issue := validateVideoTiming(timing, media, false); issue != nil {
			return nil, ErrInvalidPhysicalBinding{Reason: issue.GetSummary()}
		}
		if timing.GetAudioPolicy() != shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_UNSPECIFIED && timing.GetAudioPolicy() != shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_SILENT && timing.GetAudioPolicy() != shotsv1.VideoAudioPolicy_VIDEO_AUDIO_POLICY_PASSTHROUGH {
			return nil, ErrInvalidPhysicalBinding{Reason: "unknown source audio policy"}
		}
		if err := validatePhysicalRecordingPresentation(presentation, binding, presentationVersion, media.InspectionReceipt); err != nil {
			return nil, ErrInvalidPhysicalBinding{Reason: err.Error()}
		}
		cfg, configErr := presentations.ConfigFromSavedVersion(image.Pt(int(binding.GetScreenWidthPixels()), int(binding.GetScreenHeightPixels())), presentation, presentationVersion)
		if configErr != nil {
			return nil, ErrInvalidPhysicalBinding{Reason: configErr.Error()}
		}
		layout, layoutErr := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, int(media.InspectionReceipt.VideoWidth), int(media.InspectionReceipt.VideoHeight)), cfg)
		if layoutErr != nil {
			return nil, ErrInvalidPhysicalBinding{Reason: layoutErr.Error()}
		}
		configBytes, _ := json.Marshal(cfg)
		layoutBytes, _ := json.Marshal(layout)
		configSum, layoutSum := sha256.Sum256(configBytes), sha256.Sum256(layoutBytes)
		configSHA, layoutSHA = hex.EncodeToString(configSum[:]), hex.EncodeToString(layoutSum[:])
		recordingConfig, recordingLayout = &cfg, &layout
	}
	sampleTime := binding.GetMotionSampleTick()
	var sample motions.MotionSample
	if binding.GetMotionOutput() == nil {
		if sampleTime == nil || sampleTime.GetDenominator() <= 0 || sampleTime.GetNumerator() < 0 {
			return nil, ErrInvalidPhysicalBinding{Reason: "motion sample time must be a nonnegative rational tick"}
		}
		sample, err = motions.SampleAt(motionCandidate, composition.Rational{Num: sampleTime.GetNumerator(), Den: sampleTime.GetDenominator()})
		if err != nil {
			return nil, ErrInvalidPhysicalBinding{Reason: "motion sample time is outside the selected motion revision"}
		}
	}
	assets := make([]PhysicalAssetBinding, 0, 1+len(sceneCandidate.Artifacts))
	assets = append(assets, PhysicalAssetBinding{BindingID: "device_geometry", Role: "device_geometry", Reference: deviceCandidate.Geometry})
	for _, asset := range sceneCandidate.Artifacts {
		role := "scene_texture"
		if asset.Ref.Role == library.RoleGeometryGLB {
			role = "scene_geometry"
		}
		assets = append(assets, PhysicalAssetBinding{BindingID: asset.ID, Role: role, Reference: asset.Ref})
	}
	for _, asset := range assets {
		resolved, resolveErr := r.library.Resolve(ctx, asset.Reference, library.UseRender)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve %s artifact %q for render: %w", asset.Role, asset.BindingID, resolveErr)
		}
		_ = resolved.Reader.Close()
	}
	return &PhysicalResolution{
		DeviceID: deviceRevision.DeviceID, DeviceRevision: deviceRevision.Revision, DeviceSHA256: deviceSHA,
		SceneID: sceneRevision.SceneID, SceneRevision: sceneRevision.Revision, SceneSHA256: sceneSHA,
		MotionID: motionRevision.MotionID, MotionRevision: motionRevision.Revision, MotionSHA256: motionSHA,
		DeviceCandidate: deviceCandidate, SceneCandidate: sceneCandidate, MotionCandidate: motionCandidate,
		DeviceCanonicalJSON: deviceJSON, SceneCanonicalJSON: sceneJSON, MotionCanonicalJSON: motionJSON,
		Assets: assets, Binding: binding, ScreenMedia: media, ScreenRasterPNG: screenPNG,
		PresentationProfile: presentations.SimulatedCompositorProfile, PresentationVersion: presentationVersion,
		ConfigSHA256: configSHA, LayoutSHA256: layoutSHA, RasterSHA256: rasterSHA, MotionSample: sample,
		RecordingConfig: recordingConfig, RecordingLayout: recordingLayout,
		OutputWidth: physicalOutputWidth(binding), OutputHeight: physicalOutputHeight(binding),
	}, nil
}

// validatePhysicalArticulationMode keeps the public path on the single-hinge
// renderer profile. Recording texture input and still/preview output have
// separate source/output contracts and are rejected before a job can be
// claimed.
func validatePhysicalArticulationMode(binding *shotsv1.PhysicalShotBinding, timing *shotsv1.VideoTiming, motion motions.MotionCandidate) error {
	jointTracks := 0
	for _, track := range motion.Tracks {
		if track.Target.JointID != "" {
			jointTracks++
		}
	}
	if jointTracks == 0 {
		return nil
	}
	if jointTracks != 1 {
		return errors.New("physical articulation currently supports exactly one joint track")
	}
	if timing != nil {
		return errors.New("physical articulation with recording textures is not supported yet; use a still screen source")
	}
	if binding == nil || binding.GetMotionOutput() == nil {
		return errors.New("physical articulation currently requires a static-screen MP4 motion output")
	}
	return nil
}

func validatePhysicalRecordingPresentation(saved *shotsv1.ScreenPresentation, binding *shotsv1.PhysicalShotBinding, version string, receipt *projects.InspectionReceipt) error {
	if receipt == nil || receipt.Classification != string(rendering.InspectionAccepted) || receipt.VideoWidth == 0 || receipt.VideoHeight == 0 || receipt.VideoWidth > 3840 || receipt.VideoHeight > 3840 || uint64(receipt.VideoWidth)*uint64(receipt.VideoHeight) > 3840*2160 {
		return errors.New("physical recording requires an accepted inspected video raster")
	}
	if saved == nil || saved.GetCaptureArea() != shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY || (saved.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM && saved.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP) {
		return errors.New("physical recording requires a supported app-only phone or browser presentation")
	}
	_, err := presentations.ConfigFromSavedVersion(image.Pt(int(binding.GetScreenWidthPixels()), int(binding.GetScreenHeightPixels())), saved, version)
	return err
}

func outputWidth(profile shotsv1.PhysicalStillOutputProfile) uint32 {
	width, _ := physicalOutputDimensions(profile)
	return width
}

func outputHeight(profile shotsv1.PhysicalStillOutputProfile) uint32 {
	_, height := physicalOutputDimensions(profile)
	return height
}

func (r *ownedPhysicalResolver) composePhysicalScreen(ctx context.Context, projectID string, media projects.SourceMedia, saved *shotsv1.ScreenPresentation, version string, width, height int) ([]byte, string, string, string, error) {
	if width <= 0 || height <= 0 || width > 4096 || height > 4096 || int64(width)*int64(height) > 16_000_000 {
		return nil, "", "", "", ErrInvalidPhysicalBinding{Reason: "screen raster is outside supported bounds"}
	}
	source, err := (&service{media: r.media}).openPresentationSource(ctx, projectID, media.ID, media)
	if err != nil {
		return nil, "", "", "", err
	}
	var raster *image.NRGBA
	var configBytes, layoutBytes []byte
	if saved.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN {
		if saved.GetCaptureArea() != shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED || source.Bounds().Dx() != width || source.Bounds().Dy() != height {
			return nil, "", "", "", ErrInvalidPhysicalBinding{Reason: "plain physical screen input must be an already-composed image at the declared display raster"}
		}
		cfg := presentations.Config{Target: image.Pt(width, height)}
		configBytes, _ = json.Marshal(cfg)
		layoutBytes, _ = json.Marshal(struct{ Width, Height int }{width, height})
		if rgba, ok := source.(*image.NRGBA); ok {
			raster = rgba
		} else {
			raster = image.NewNRGBA(image.Rect(0, 0, width, height))
			draw.Draw(raster, raster.Bounds(), source, source.Bounds().Min, draw.Src)
		}
	} else {
		cfg, configErr := presentations.ConfigFromSavedVersion(image.Pt(width, height), saved, version)
		if configErr != nil {
			return nil, "", "", "", configErr
		}
		result, composeErr := presentations.Compose(ctx, source, cfg)
		if composeErr != nil {
			return nil, "", "", "", composeErr
		}
		raster = result.Raster
		configBytes, _ = json.Marshal(cfg)
		layoutBytes, _ = json.Marshal(result.Layout)
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, raster); err != nil {
		return nil, "", "", "", fmt.Errorf("encode physical screen raster: %w", err)
	}
	configSum, layoutSum, rasterSum := sha256.Sum256(configBytes), sha256.Sum256(layoutBytes), sha256.Sum256(buffer.Bytes())
	return buffer.Bytes(), hex.EncodeToString(configSum[:]), hex.EncodeToString(layoutSum[:]), hex.EncodeToString(rasterSum[:]), nil
}

func validComponentRef(ref *shotsv1.ComponentRevisionRef) bool {
	return ref != nil && ref.GetId() != "" && ref.GetRevision() > 0 && physicalBindingDigest(ref)
}

func validatePhysicalProfileSelection(binding *shotsv1.PhysicalShotBinding) error {
	if binding.GetRendererProfileId() != "device-studio-physical-cycles-v1" || binding.GetRendererProfileVersion() != 1 {
		return errors.New("unsupported physical renderer profile")
	}
	switch binding.GetBackend() {
	case shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CPU,
		shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_CUDA,
		shotsv1.PhysicalRenderBackend_PHYSICAL_RENDER_BACKEND_OPTIX:
	default:
		return errors.New("an explicit supported physical render backend is required; fallback is not automatic")
	}
	if motion := binding.GetMotionOutput(); motion != nil {
		if binding.GetOutputProfile() != shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_UNSPECIFIED || binding.GetMotionSampleTick() != nil {
			return errors.New("physical motion output cannot be combined with the still profile or still sample tick")
		}
		profileMaxFrames := uint32(48)
		validProfile := motion.GetProfileId() == rendering.PhysicalMotionOutput && motion.GetProfileVersion() == 1
		if motion.GetProfileId() == rendering.PhysicalMotionOutputV2 && motion.GetProfileVersion() == 2 {
			profileMaxFrames = rendering.PhysicalMotionMaxFramesV2
			validProfile = true
		}
		if !validProfile {
			return errors.New("unsupported physical motion output profile")
		}
		if (motion.GetWidth() != 1920 || motion.GetHeight() != 1080) && (motion.GetWidth() != 1080 || motion.GetHeight() != 1920) {
			return errors.New("physical motion output must use the pinned 1080p raster")
		}
		if (motion.GetOutputRateNumerator() != 24 && motion.GetOutputRateNumerator() != 30) || motion.GetOutputRateDenominator() != 1 {
			return errors.New("physical motion output rate must be 24 or 30 fps")
		}
		maxFrames := profileMaxFrames
		if profileMaxFrames == rendering.PhysicalMotionMaxFramesV2 {
			maxFrames = motion.GetOutputRateNumerator() * 10
		}
		if motion.GetFrameCount() == 0 || motion.GetFrameCount() > maxFrames {
			return fmt.Errorf("physical motion output must contain between 1 and %d frames for its pinned profile", maxFrames)
		}
		return nil
	}
	switch binding.GetOutputProfile() {
	case shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1, shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1:
		if binding.GetMotionSampleTick() == nil {
			return errors.New("physical still output requires one exact motion sample tick")
		}
		return nil
	default:
		return errors.New("a supported physical still output profile is required")
	}
}

func physicalOutputWidth(binding *shotsv1.PhysicalShotBinding) uint32 {
	if binding.GetMotionOutput() != nil {
		return binding.GetMotionOutput().GetWidth()
	}
	return outputWidth(binding.GetOutputProfile())
}

func physicalOutputHeight(binding *shotsv1.PhysicalShotBinding) uint32 {
	if binding.GetMotionOutput() != nil {
		return binding.GetMotionOutput().GetHeight()
	}
	return outputHeight(binding.GetOutputProfile())
}

func validatePhysicalPresentation(saved *shotsv1.ScreenPresentation, binding *shotsv1.PhysicalShotBinding, version string) error {
	if saved.GetKind() == shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN {
		if saved.GetCaptureArea() != shotsv1.CaptureArea_CAPTURE_AREA_ALREADY_COMPOSED {
			return errors.New("plain physical presentation requires an already-composed screen image")
		}
		return nil
	}
	if saved.GetCaptureArea() != shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY {
		return errors.New("physical simulated presentation requires an app-only source")
	}
	if saved.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM && saved.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP {
		return errors.New("physical presentation kind is unsupported")
	}
	if _, err := presentations.ConfigFromSavedVersion(image.Pt(int(binding.GetScreenWidthPixels()), int(binding.GetScreenHeightPixels())), saved, version); err != nil {
		return fmt.Errorf("physical screen presentation: %w", err)
	}
	return nil
}

func physicalBindingDigest(ref *shotsv1.ComponentRevisionRef) bool {
	if ref == nil || len(ref.GetSha256()) != 64 {
		return false
	}
	_, err := hex.DecodeString(ref.GetSha256())
	return err == nil
}

func validatePhysicalSelection(binding *shotsv1.PhysicalShotBinding, device devices.DeviceCandidate, scene scenes.SceneCandidate, motion motions.MotionCandidate) error {
	if !physicalBindingDigest(binding.GetDevice()) || !physicalBindingDigest(binding.GetScene()) || !physicalBindingDigest(binding.GetMotion()) {
		return errors.New("component revision digest must be lowercase SHA-256")
	}
	if binding.GetDisplayId() == "" || binding.GetDeviceSupportId() == "" || binding.GetSceneSupportId() == "" || binding.GetCameraId() == "" {
		return errors.New("display, support, and camera selections are required")
	}
	deviceSupport, sceneSupport, camera, display := false, false, false, false
	var deviceInterface string
	for _, support := range device.SupportFrames {
		if support.ID == binding.GetDeviceSupportId() {
			deviceSupport, deviceInterface = true, support.InterfaceID
		}
	}
	for _, support := range scene.SupportFrames {
		if support.ID == binding.GetSceneSupportId() && support.InterfaceID == deviceInterface {
			sceneSupport = true
		}
	}
	for _, frame := range scene.CameraFrames {
		camera = camera || frame.ID == binding.GetCameraId()
	}
	for _, item := range device.DisplayBindings {
		if item.DisplayID == binding.GetDisplayId() {
			for _, displaySurface := range device.Definition.Displays {
				if displaySurface.ID == binding.GetDisplayId() {
					display = true
					if metric := displaySurface.CoordinateMetric; metric != nil && metric.WidthMM > 0 && metric.HeightMM > 0 && binding.GetScreenWidthPixels() > 0 && binding.GetScreenHeightPixels() > 0 {
						want := metric.WidthMM / metric.HeightMM
						got := float64(binding.GetScreenWidthPixels()) / float64(binding.GetScreenHeightPixels())
						if math.Abs(got-want) > want*0.005 {
							return errors.New("screen raster aspect differs from the candidate display metric")
						}
					}
				}
			}
		}
	}
	if !deviceSupport || !sceneSupport || !camera || !display {
		return errors.New("selected display, matching support interface, or scene camera is not present in the pinned candidates")
	}
	if output := binding.GetMotionOutput(); output != nil {
		if err := validatePhysicalMotionFrameCount(output, motion); err != nil {
			return err
		}
	}
	if binding.GetSceneSupportWorld() == nil || binding.GetDeviceRootRest() == nil || binding.GetSupportPivotM() == nil {
		return errors.New("explicit support world, pivot, and device rest transform are required")
	}
	if _, err := fromProtoTRS(binding.GetSceneSupportWorld()); err != nil {
		return err
	}
	if _, err := fromProtoTRS(binding.GetDeviceRootRest()); err != nil {
		return err
	}
	pivot := physical.Vec3{X: binding.GetSupportPivotM().GetX(), Y: binding.GetSupportPivotM().GetY(), Z: binding.GetSupportPivotM().GetZ()}
	if err := pivot.Validate(); err != nil {
		return errors.New("support pivot must be finite")
	}
	for _, track := range motion.Tracks {
		if track.Target.DeviceRoot {
			continue
		}
		if track.Target.JointID != "" {
			found := false
			for _, joint := range device.Definition.Joints {
				found = found || joint.ID == track.Target.JointID
			}
			if !found {
				return fmt.Errorf("motion target joint %q is not part of the device candidate", track.Target.JointID)
			}
		}
		if track.Target.SceneCameraID != "" {
			found := false
			for _, frame := range scene.CameraFrames {
				found = found || frame.ID == track.Target.SceneCameraID
			}
			if !found {
				return fmt.Errorf("motion target camera %q is not part of the scene candidate", track.Target.SceneCameraID)
			}
		}
	}
	return nil
}

func validatePhysicalMotionFrameCount(output *shotsv1.PhysicalMotionOutputBinding, motion motions.MotionCandidate) error {
	if output == nil || motion.DurationTicks <= 0 || uint64(motion.DurationTicks) > uint64(^uint32(0)) {
		return errors.New("physical motion duration exceeds the supported exact schedule")
	}
	if motion.Timebase.Num <= 0 || motion.Timebase.Den <= 0 || output.GetOutputRateNumerator() == 0 || output.GetOutputRateDenominator() == 0 {
		return errors.New("physical motion timebase or output rate is invalid")
	}
	numerator := new(big.Int).SetInt64(motion.DurationTicks)
	numerator.Mul(numerator, big.NewInt(motion.Timebase.Den))
	numerator.Mul(numerator, new(big.Int).SetUint64(uint64(output.GetOutputRateNumerator())))
	denominator := new(big.Int).Mul(big.NewInt(motion.Timebase.Num), new(big.Int).SetUint64(uint64(output.GetOutputRateDenominator())))
	frames := new(big.Int).Quo(numerator, denominator)
	if new(big.Int).Rem(numerator, denominator).Sign() != 0 {
		frames.Add(frames, big.NewInt(1))
	}
	if !frames.IsUint64() || frames.Uint64() != uint64(output.GetFrameCount()) {
		return errors.New("physical motion frame count must cover the exact motion duration at the selected output rate")
	}
	return nil
}

func fromProtoTRS(value *physicalv1.TRS) (physical.TRS, error) {
	if value == nil || value.GetTranslationM() == nil || value.GetRotationXyzw() == nil || value.GetScale() == nil {
		return physical.TRS{}, errors.New("physical transform requires translation, rotation, and scale")
	}
	result := physical.TRS{
		Translation: physical.Vec3{X: value.GetTranslationM().GetX(), Y: value.GetTranslationM().GetY(), Z: value.GetTranslationM().GetZ()},
		Rotation:    physical.QuatXYZW{X: value.GetRotationXyzw().GetX(), Y: value.GetRotationXyzw().GetY(), Z: value.GetRotationXyzw().GetZ(), W: value.GetRotationXyzw().GetW()},
		Scale:       physical.Vec3{X: value.GetScale().GetX(), Y: value.GetScale().GetY(), Z: value.GetScale().GetZ()},
	}
	return physical.CanonicalizeTRS(result)
}
