package renderjobs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"image"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"device-studio/internal/composition"
	"device-studio/internal/jobs"
	"device-studio/internal/presentations"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

const (
	videoSampleOutputProfile                = "video-sample-png-v1"
	videoFinalOutputProfile                 = "video-final-mp4-h264-sdr-v1"
	videoBaseRecipeVersionForManifest       = "device-studio-video-base-recipe-v1"
	maxVideoInputBytes                int64 = 512 << 20
	maxVideoWorkspaceInputBytes       int64 = 600 << 20
	maxInspectionClockBytes           int64 = 1 << 20
	maxVideoOutputBytes               int64 = 160 << 20
)

type videoClosure struct {
	Version            int                     `json:"version"`
	ProjectID          string                  `json:"project_id"`
	ShotID             string                  `json:"shot_id"`
	ShotRevision       int64                   `json:"shot_revision"`
	RecipeSHA256       string                  `json:"recipe_sha256"`
	Snapshot           *shotsv1.ShotSnapshot   `json:"snapshot"`
	Manifest           rendering.VideoManifest `json:"manifest"`
	WorkerKey          string                  `json:"worker_key"`
	WorkerSHA256       string                  `json:"worker_sha256"`
	ToolProfileKey     string                  `json:"tool_profile_key"`
	ToolProfileSHA256  string                  `json:"tool_profile_sha256"`
	InspectionManifest string                  `json:"inspection_manifest_key"`
	InspectionSHA256   string                  `json:"inspection_manifest_sha256"`
	InspectionReceipt  string                  `json:"inspection_receipt_key"`
	ReceiptSHA256      string                  `json:"inspection_receipt_sha256"`
	ClockKey           string                  `json:"clock_key"`
	ClockSHA256        string                  `json:"clock_sha256"`
	SourceMediaID      string                  `json:"source_media_id"`
	PlateMediaID       string                  `json:"plate_media_id"`
	ActiveMaskMediaID  string                  `json:"active_mask_media_id"`
	HardwareMaskID     string                  `json:"hardware_mask_media_id,omitempty"`
	SampleReceiptKey   string                  `json:"sample_receipt_key,omitempty"`
	SampleReceiptSHA   string                  `json:"sample_receipt_sha256,omitempty"`
	SampleReceiptSize  int64                   `json:"sample_receipt_size_bytes,omitempty"`
}

func (c *Coordinator) submitVideo(ctx context.Context, req SubmitRequest) (jobs.Job, bool, error) {
	if err := validateVideoSubmitRequest(req); err != nil {
		return jobs.Job{}, false, err
	}
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, false, err
	}
	if existing, found, lookupErr := c.findIdempotentSubmit(ctx, route, req); lookupErr != nil || found {
		return existing, false, lookupErr
	}
	resolved, err := c.shots.ResolveForRender(ctx, req.ShotID, req.Revision)
	if err != nil {
		return jobs.Job{}, false, fmt.Errorf("resolve immutable video shot: %w", err)
	}
	if err := validateResolved(req.ProjectID, req.ShotID, req.Revision, resolved); err != nil {
		return jobs.Job{}, false, err
	}
	if !resolved.Revision.RenderEligible {
		return jobs.Job{}, false, fmt.Errorf("%w: %v", ErrNotEligible, resolved.Revision.Issues)
	}
	workerBytes, _, workerSHA, err := c.readCurrentExecutable()
	if err != nil {
		return jobs.Job{}, false, err
	}
	workerKey := workerBlobPrefix + workerSHA + "/device-studio-api"
	if err := c.putVerified(ctx, route, workerKey, workerBytes, "application/octet-stream", workerSHA); err != nil {
		return jobs.Job{}, false, err
	}
	video, err := c.makeVideoClosure(ctx, route, req, resolved, workerKey, workerSHA)
	if err != nil {
		return jobs.Job{}, false, err
	}
	closureBytes, err := json.Marshal(video)
	if err != nil {
		return jobs.Job{}, false, err
	}
	closureBytes = append(closureBytes, '\n')
	if len(closureBytes) > maxManifestBytes {
		return jobs.Job{}, false, errors.New("renderjobs: video closure exceeds limit")
	}
	closureSHA := digestBytes(closureBytes)
	closureKey := closureBlobPrefix + closureSHA + ".json"
	if err := c.putVerified(ctx, route, closureKey, closureBytes, "application/json", closureSHA); err != nil {
		return jobs.Job{}, false, err
	}
	input := jobs.Input{
		ProjectID: req.ProjectID, ShotRevisionID: shotRevisionID(req.ShotID, req.Revision), ShotRevision: req.Revision,
		Bindings: []jobs.ArtifactBinding{
			{Role: "shot_snapshot", ArtifactID: "shot:" + req.ShotID + ":revision:" + strconv.FormatInt(req.Revision, 10), RevisionID: "revision:" + strconv.FormatInt(req.Revision, 10), SHA256: resolved.Revision.RecipeSHA256},
			{Role: "video_closure", ArtifactID: closureKey, RevisionID: "sha256:" + closureSHA, SHA256: closureSHA},
			{Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerSHA, SHA256: workerSHA},
			routeBinding(route),
		},
		SourceClockDigest: video.Manifest.Inspection.Clock.SHA256,
		ToolProfileDigest: video.Manifest.Inspection.ProfileDigest,
		RendererVersion:   video.Manifest.VideoProfile, OutputProfile: req.OutputProfile,
		SubjectKind: jobs.SubjectVideoRender, SubjectID: req.ShotID,
	}
	for _, b := range videoInputBindings(video) {
		input.Bindings = append(input.Bindings, b)
	}
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: req.IdempotencyKey, Input: input})
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

func validateVideoSubmitRequest(req SubmitRequest) error {
	if req.ProjectID == "" || req.ShotID == "" || req.Revision <= 0 || req.IdempotencyKey == "" || len(req.IdempotencyKey) > 200 {
		return ErrInvalidRequest
	}
	switch req.VideoMode {
	case rendering.VideoModeSample:
		if req.OutputProfile != "" && req.OutputProfile != videoSampleOutputProfile {
			return ErrInvalidRequest
		}
	case rendering.VideoModeFinal:
		if req.OutputProfile != "" && req.OutputProfile != videoFinalOutputProfile {
			return ErrInvalidRequest
		}
	default:
		return ErrInvalidRequest
	}
	return nil
}

func videoInputBindings(v videoClosure) []jobs.ArtifactBinding {
	bindings := []jobs.ArtifactBinding{
		{Role: "source_media", ArtifactID: v.SourceMediaID, RevisionID: "sha256:" + v.Manifest.Inspection.Source.SHA256, SHA256: v.Manifest.Inspection.Source.SHA256},
		{Role: "video_plate", ArtifactID: v.PlateMediaID, RevisionID: "sha256:" + v.Manifest.Scene.Plate.SHA256, SHA256: v.Manifest.Scene.Plate.SHA256},
		{Role: "video_active_mask", ArtifactID: v.ActiveMaskMediaID, RevisionID: "sha256:" + v.Manifest.Scene.ActiveMask.SHA256, SHA256: v.Manifest.Scene.ActiveMask.SHA256},
		{Role: "inspection_manifest", ArtifactID: v.InspectionManifest, RevisionID: "sha256:" + v.InspectionSHA256, SHA256: v.InspectionSHA256},
		{Role: "inspection_receipt", ArtifactID: v.InspectionReceipt, RevisionID: "sha256:" + v.ReceiptSHA256, SHA256: v.ReceiptSHA256},
		{Role: "inspection_clock", ArtifactID: v.ClockKey, RevisionID: "sha256:" + v.ClockSHA256, SHA256: v.ClockSHA256},
		{Role: "renderer_tool_profile", ArtifactID: v.ToolProfileKey, RevisionID: "sha256:" + v.ToolProfileSHA256, SHA256: v.ToolProfileSHA256},
	}
	if v.HardwareMaskID != "" && v.Manifest.Scene.HardwareMask != nil {
		b := v.Manifest.Scene.HardwareMask
		bindings = append(bindings, jobs.ArtifactBinding{Role: "video_hardware_mask", ArtifactID: v.HardwareMaskID, RevisionID: "sha256:" + b.SHA256, SHA256: b.SHA256})
	}
	if v.SampleReceiptKey != "" {
		bindings = append(bindings, jobs.ArtifactBinding{Role: "video_sample_receipt", ArtifactID: v.SampleReceiptKey, RevisionID: "sha256:" + v.SampleReceiptSHA, SHA256: v.SampleReceiptSHA})
	}
	return bindings
}

func (c *Coordinator) makeVideoClosure(ctx context.Context, route routePin, req SubmitRequest, resolved shots.ResolvedShot, workerKey, workerSHA string) (videoClosure, error) {
	rev := resolved.Revision
	snapshot := rev.Snapshot
	timing := snapshot.GetVideoTiming()
	if timing == nil || rev.Screen == nil || rev.Plate == nil || rev.ActiveMask == nil || !validDigest(rev.RecipeSHA256) {
		return videoClosure{}, ErrNotEligible
	}
	source, err := c.media.Get(ctx, req.ProjectID, rev.Screen.GetId())
	if err != nil || source.Kind != projects.MediaKindVideo || source.Readiness != projects.ReadinessReady || source.InspectionReceipt == nil || source.InspectionReceipt.Classification != "accepted" || source.SHA256 != timing.GetSourceSha256() || source.SizeBytes != timing.GetSourceSizeBytes() {
		return videoClosure{}, errors.Join(ErrNotEligible, err)
	}
	accepted, err := c.reconstructAcceptedInspection(ctx, route, req.ProjectID, source)
	if err != nil {
		return videoClosure{}, err
	}
	inspectionReceipt, profileBytes := accepted.Receipt, accepted.ProfileBytes
	manifestBytes, receiptBytes, clockBytes := accepted.ManifestBytes, accepted.ReceiptBytes, accepted.ClockBytes
	pin := func(role, suffix string, data []byte, mime string) (string, string, error) {
		sha := digestBytes(data)
		key := inputBlobPrefix + sha + "/" + suffix
		if err := c.putVerified(ctx, route, key, data, mime, sha); err != nil {
			return "", "", err
		}
		return key, sha, nil
	}
	profileKey, profileSHA, err := pin("profile", "renderer-tool-profile.json", profileBytes, "application/json")
	if err != nil {
		return videoClosure{}, err
	}
	manifestKey, manifestSHA, err := pin("inspection-manifest", "inspection-manifest.json", manifestBytes, "application/json")
	if err != nil {
		return videoClosure{}, err
	}
	receiptKey, receiptSHA, err := pin("inspection-receipt", "inspection-receipt.json", receiptBytes, "application/json")
	if err != nil {
		return videoClosure{}, err
	}
	clockKey, clockSHA, err := pin("inspection-clock", "source-clock.v1.bin", clockBytes, "application/octet-stream")
	if err != nil {
		return videoClosure{}, err
	}
	scene := snapshot.GetScene()
	quad := scene.GetPlateQuad()
	if quad == nil || rev.Plate.GetWidthPixels() == 0 || rev.Plate.GetHeightPixels() == 0 {
		return videoClosure{}, ErrNotEligible
	}
	w, h := 1920, 1080
	if rev.Plate.GetWidthPixels() < rev.Plate.GetHeightPixels() {
		w, h = 1080, 1920
	}
	cfg, err := presentations.ConfigFromSavedVersion(image.Pt(int(scene.GetDisplayWidthPixels()), int(scene.GetDisplayHeightPixels())), snapshot.GetPresentation(), rev.PresentationVersion)
	if err != nil {
		return videoClosure{}, err
	}
	baseSHA, err := shots.VideoBaseRecipeSHA(snapshot)
	if err != nil {
		return videoClosure{}, err
	}
	videoProfile := rendering.VideoProfileID
	if snapshot.GetCameraMotion() != nil {
		videoProfile = rendering.VideoScreenEntryProfileID
	}
	manifest := rendering.VideoManifest{
		Version: 1, ProtocolVersion: 1, Mode: req.VideoMode, WorkerSHA256: workerSHA,
		VideoProfile: videoProfile, CompositorProfile: rendering.VideoCompositorID,
		TrimStartPTS: timing.GetTrimStartPts(), TrimEndPTS: timing.GetTrimEndPts(),
		PlaybackRate: composition.Rational{Num: int64(timing.GetPlaybackRateNum()), Den: int64(timing.GetPlaybackRateDen())},
		OutputRate:   composition.Rational{Num: int64(timing.GetOutputRateNum()), Den: int64(timing.GetOutputRateDen())},
		Width:        w, Height: h, ShotRevision: req.Revision, BaseShotRevision: req.Revision,
		BaseRecipeVersion: videoBaseRecipeVersionForManifest, BaseRecipeSHA256: baseSHA,
		RecipeSHA256: rev.RecipeSHA256,
		Presentation: &cfg,
		Scene:        rendering.VideoScene{Plate: rendering.VideoBinding{Name: "plate.image", SHA256: rev.Plate.GetSha256(), SizeBytes: rev.Plate.GetSizeBytes()}, ActiveMask: rendering.VideoBinding{Name: "active-mask.png", SHA256: rev.ActiveMask.GetSha256(), SizeBytes: rev.ActiveMask.GetSizeBytes()}, SurfaceWidth: int(scene.GetDisplayWidthPixels()), SurfaceHeight: int(scene.GetDisplayHeightPixels()), Quad: [4]rendering.Corner{{X: quad.GetTopLeft().GetX(), Y: quad.GetTopLeft().GetY()}, {X: quad.GetTopRight().GetX(), Y: quad.GetTopRight().GetY()}, {X: quad.GetBottomRight().GetX(), Y: quad.GetBottomRight().GetY()}, {X: quad.GetBottomLeft().GetX(), Y: quad.GetBottomLeft().GetY()}}, Samples: 2},
		Inspection:   rendering.VideoInspection{Source: rendering.VideoBinding{Name: rendering.InspectionSourceName, SHA256: source.SHA256, SizeBytes: source.SizeBytes}, Manifest: rendering.VideoBinding{Name: rendering.AcceptedInspectionManifestName, SHA256: manifestSHA, SizeBytes: int64(len(manifestBytes))}, Receipt: rendering.VideoBinding{Name: rendering.InspectionReceiptName, SHA256: receiptSHA, SizeBytes: int64(len(receiptBytes))}, Clock: rendering.VideoBinding{Name: rendering.InspectionClockName, SHA256: clockSHA, SizeBytes: int64(len(clockBytes))}, ProfileID: inspectionReceipt.ProfileID, ProfileDigest: inspectionReceipt.ProfileDigest, ProtocolVersion: inspectionReceipt.ProtocolVersion, SelectedStream: uint32(*inspectionReceipt.VideoStreamIndex), TimeBase: composition.Rational{Num: inspectionReceipt.TimeBaseNum, Den: inspectionReceipt.TimeBaseDen}, StreamStartPTS: *inspectionReceipt.StreamStartPTS, SourceEndPTS: *inspectionReceipt.SourceEndPTS, LastFrameDurationTicks: inspectionReceipt.LastFrameDurationTicks, EOFPolicy: inspectionReceipt.EOFPolicy, FrameCount: inspectionReceipt.FrameCount, Width: int(inspectionReceipt.DisplayWidth), Height: int(inspectionReceipt.DisplayHeight), ColorPrimaries: inspectionReceipt.ColorPrimaries, ColorTransfer: inspectionReceipt.ColorTransfer, ColorMatrix: inspectionReceipt.ColorMatrix, ColorRange: inspectionReceipt.ColorRange, ColorPolicy: inspectionReceipt.ColorPolicy},
	}
	if camera := snapshot.GetCameraMotion(); camera != nil {
		matte := camera.GetMatteRgb()
		if matte == nil || matte.GetRed() > 255 || matte.GetGreen() > 255 || matte.GetBlue() > 255 {
			return videoClosure{}, ErrNotEligible
		}
		manifest.Camera = &rendering.Camera{
			ProfileID: camera.GetProfileId(), ProfileVersion: camera.GetProfileVersion(),
			DurationOutputFrames: camera.GetDurationOutputFrames(),
			MatteRGB:             &presentations.RGB{R: uint8(matte.GetRed()), G: uint8(matte.GetGreen()), B: uint8(matte.GetBlue())},
		}
		if r := camera.GetTargetRect(); r != nil {
			manifest.Camera.TargetRect = &presentations.BasisRect{X: int(r.GetX()), Y: int(r.GetY()), Width: int(r.GetWidth()), Height: int(r.GetHeight())}
		}
		if r := camera.GetSceneRect(); r != nil {
			manifest.Camera.SceneRect = &presentations.BasisRect{X: int(r.GetX()), Y: int(r.GetY()), Width: int(r.GetWidth()), Height: int(r.GetHeight())}
		}
		manifest.Scene.PlateWidth = int(rev.Plate.GetWidthPixels())
		manifest.Scene.PlateHeight = int(rev.Plate.GetHeightPixels())
	}
	if rev.HardwareMask != nil {
		manifest.Scene.HardwareMask = &rendering.VideoBinding{Name: "hardware-mask.png", SHA256: rev.HardwareMask.GetSha256(), SizeBytes: rev.HardwareMask.GetSizeBytes()}
	}
	if requiresPinnedVideoLayout(req.VideoMode, cfg) {
		layoutWidth, layoutHeight := int(rev.Screen.GetWidthPixels()), int(rev.Screen.GetHeightPixels())
		if req.VideoMode == rendering.VideoModeSample {
			layoutWidth, layoutHeight = int(inspectionReceipt.DisplayWidth), int(inspectionReceipt.DisplayHeight)
		}
		layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, layoutWidth, layoutHeight), cfg)
		if err != nil {
			return videoClosure{}, err
		}
		manifest.PresentationLayout = &layout
		if claim := timing.GetSampleBinding(); claim != nil {
			manifest.BaseShotRevision = claim.GetBaseShotRevision()
			manifest.BaseRecipeSHA256 = claim.GetBaseRecipeSha256()
			sample, sampleReceiptKey, sampleReceiptSHA, sampleReceiptSize, sampleErr := c.pinSampleBinding(ctx, route, req.ProjectID, req.ShotID, claim, manifest)
			if sampleErr != nil {
				return videoClosure{}, sampleErr
			}
			manifest.Sample = sample
			return videoClosure{Version: 1, ProjectID: req.ProjectID, ShotID: req.ShotID, ShotRevision: req.Revision, RecipeSHA256: rev.RecipeSHA256, Snapshot: snapshot, Manifest: manifest, WorkerKey: workerKey, WorkerSHA256: workerSHA, ToolProfileKey: profileKey, ToolProfileSHA256: profileSHA, InspectionManifest: manifestKey, InspectionSHA256: manifestSHA, InspectionReceipt: receiptKey, ReceiptSHA256: receiptSHA, ClockKey: clockKey, ClockSHA256: clockSHA, SourceMediaID: source.ID, PlateMediaID: rev.Plate.GetId(), ActiveMaskMediaID: rev.ActiveMask.GetId(), HardwareMaskID: mediaID(rev.HardwareMask), SampleReceiptKey: sampleReceiptKey, SampleReceiptSHA: sampleReceiptSHA, SampleReceiptSize: sampleReceiptSize}, nil
		}
	}
	return videoClosure{Version: 1, ProjectID: req.ProjectID, ShotID: req.ShotID, ShotRevision: req.Revision, RecipeSHA256: rev.RecipeSHA256, Snapshot: snapshot, Manifest: manifest, WorkerKey: workerKey, WorkerSHA256: workerSHA, ToolProfileKey: profileKey, ToolProfileSHA256: profileSHA, InspectionManifest: manifestKey, InspectionSHA256: manifestSHA, InspectionReceipt: receiptKey, ReceiptSHA256: receiptSHA, ClockKey: clockKey, ClockSHA256: clockSHA, SourceMediaID: source.ID, PlateMediaID: rev.Plate.GetId(), ActiveMaskMediaID: rev.ActiveMask.GetId(), HardwareMaskID: mediaID(rev.HardwareMask)}, nil
}

func isAutoPhoneVideoSample(mode rendering.VideoMode, cfg presentations.Config) bool {
	return mode == rendering.VideoModeSample && cfg.Phone != nil && cfg.Phone.ProfileVersion == presentations.PhoneProfileVersionAuto
}

func requiresPinnedVideoLayout(mode rendering.VideoMode, cfg presentations.Config) bool {
	return mode == rendering.VideoModeFinal || mode == rendering.VideoModeSample && !isAutoPhoneVideoSample(mode, cfg)
}

func (c *Coordinator) pinSampleBinding(ctx context.Context, route routePin, projectID, shotID string, claim *shotsv1.VideoSampleBinding, final rendering.VideoManifest) (*rendering.VideoSampleBinding, string, string, int64, error) {
	job, err := c.jobs.Get(ctx, claim.GetJobId())
	if err != nil || job.Input.ProjectID != projectID || job.Input.SubjectKind != jobs.SubjectVideoRender || job.Input.SubjectID != shotID || job.Input.ShotRevisionID != shotRevisionID(shotID, claim.GetBaseShotRevision()) || job.Input.ShotRevision != claim.GetBaseShotRevision() || job.State != jobs.Succeeded {
		return nil, "", "", 0, errors.Join(ErrNotEligible, err)
	}
	var attempt *jobs.Attempt
	for i := range job.Attempts {
		if job.Attempts[i].ID == claim.GetAttemptId() {
			attempt = &job.Attempts[i]
			break
		}
	}
	if attempt == nil || attempt.State != jobs.AttemptSucceeded || attempt.Completion == nil {
		return nil, "", "", 0, ErrNotEligible
	}
	var receiptRef, manifestRef, pngRef *jobs.OutputRef
	for i := range attempt.Completion.Outputs {
		o := &attempt.Completion.Outputs[i]
		switch o.Kind {
		case "video_receipt":
			receiptRef = o
		case "video_manifest":
			manifestRef = o
		case "video_sample_png":
			pngRef = o
		}
	}
	if receiptRef == nil || manifestRef == nil || pngRef == nil || receiptRef.SHA256 != claim.GetReceiptSha256() {
		return nil, "", "", 0, ErrNotEligible
	}
	manifestBytes, err := c.readBlob(ctx, manifestRef.ArtifactID, maxManifestBytes, manifestRef.SHA256)
	if err != nil {
		return nil, "", "", 0, err
	}
	receiptBytes, err := c.readBlob(ctx, receiptRef.ArtifactID, maxReceiptBytes, receiptRef.SHA256)
	if err != nil {
		return nil, "", "", 0, err
	}
	pngBytes, err := c.readBlob(ctx, pngRef.ArtifactID, maxOutputBytes, pngRef.SHA256)
	if err != nil {
		return nil, "", "", 0, err
	}
	verified, err := rendering.VerifyVideoArtifacts(ctx, manifestBytes, receiptBytes, bytes.NewReader(pngBytes), int64(len(pngBytes)))
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("verify prior video sample: %w", err)
	}
	var sampleManifest rendering.VideoManifest
	if err := json.Unmarshal(manifestBytes, &sampleManifest); err != nil {
		return nil, "", "", 0, err
	}
	clockBinding := bindingByRole(job.Input, "inspection_clock")
	clockBytes, err := c.readBlob(ctx, clockBinding.ArtifactID, maxInspectionClockBytes, clockBinding.SHA256)
	if err != nil || clockBinding.SHA256 != claim.GetClockSha256() {
		return nil, "", "", 0, errors.Join(ErrNotEligible, err)
	}
	sourceOrdinal, sourcePTS, err := rendering.PlanVideoSample(ctx, sampleManifest, clockBytes)
	if err != nil {
		return nil, "", "", 0, err
	}
	if verified.Mode != rendering.VideoModeSample || verified.JobID != job.ID || verified.AttemptID != attempt.ID || verified.ManifestSHA256 != manifestRef.SHA256 || verified.InspectionSHA256 != rendering.VideoInspectionDigest(sampleManifest.Inspection) || verified.ShotRevision != claim.GetBaseShotRevision() || verified.BaseShotRevision != claim.GetBaseShotRevision() || verified.BaseRecipeSHA256 != claim.GetBaseRecipeSha256() || verified.OutputSHA256 != claim.GetProjectedPngSha256() || verified.OutputSizeBytes != claim.GetProjectedPngSizeBytes() || verified.FirstFrameRGBSHA256 != claim.GetComposedFrameSha256() || sampleManifest.Inspection.Source.SHA256 != claim.GetSourceSha256() || sampleManifest.Inspection.Source.SizeBytes != claim.GetSourceSizeBytes() || sampleManifest.Inspection.Clock.SHA256 != claim.GetClockSha256() || sampleManifest.TrimStartPTS != claim.GetTrimStartPts() || sampleManifest.TrimEndPTS != claim.GetTrimEndPts() || sampleManifest.Mode != rendering.VideoModeSample || sourceOrdinal != claim.GetSourceOrdinal() || sourcePTS != claim.GetSourcePts() || final.Inspection.Source.SHA256 != sampleManifest.Inspection.Source.SHA256 || final.Inspection.Source.SizeBytes != sampleManifest.Inspection.Source.SizeBytes || final.Inspection.Clock.SHA256 != sampleManifest.Inspection.Clock.SHA256 || final.TrimStartPTS != sampleManifest.TrimStartPTS || final.TrimEndPTS != sampleManifest.TrimEndPTS || final.PlaybackRate != sampleManifest.PlaybackRate || final.OutputRate != sampleManifest.OutputRate || rendering.VideoInspectionDigest(final.Inspection) != verified.InspectionSHA256 {
		return nil, "", "", 0, ErrNotEligible
	}
	if claim.GetVideoCompositionProfileId() != final.VideoProfile || claim.GetVideoCompositionProfileVersion() != 1 || verified.OutputWidth <= 0 || verified.OutputHeight <= 0 {
		return nil, "", "", 0, ErrNotEligible
	}
	timingSHA := rendering.VideoTimingDigest(final)
	sample := &rendering.VideoSampleBinding{JobID: job.ID, AttemptID: attempt.ID, ReceiptSHA256: receiptRef.SHA256, ManifestSHA256: manifestRef.SHA256, InspectionSHA256: verified.InspectionSHA256, RecipeSHA256: verified.RecipeSHA256, SourceSHA256: claim.GetSourceSha256(), ClockSHA256: claim.GetClockSha256(), TimingSHA256: timingSHA, ShotRevision: verified.ShotRevision, BaseShotRevision: claim.GetBaseShotRevision(), BaseRecipeVersion: claim.GetBaseRecipeVersion(), BaseRecipeSHA256: claim.GetBaseRecipeSha256(), ResolvedTopFill: verified.ResolvedTopFill, ResolvedBottomFill: verified.ResolvedBottomFill, ResolvedStatusInk: verified.ResolvedStatusInk, ResolvedHomeInk: verified.ResolvedHomeInk, TopFillSourceResolved: verified.TopFillSourceResolved, BottomFillSourceResolved: verified.BottomFillSourceResolved, StatusInkSourceResolved: verified.StatusInkSourceResolved, HomeInkSourceResolved: verified.HomeInkSourceResolved, FirstFrameRGBSHA256: verified.FirstFrameRGBSHA256, PixelProfile: verified.PixelProfile, Width: verified.OutputWidth, Height: verified.OutputHeight, ReceiptArtifact: rendering.VideoBinding{Name: "sample-receipt.json", SHA256: receiptRef.SHA256, SizeBytes: int64(len(receiptBytes))}}
	key := inputBlobPrefix + receiptRef.SHA256 + "/sample-receipt.json"
	if err := c.putVerified(ctx, route, key, receiptBytes, "application/json", receiptRef.SHA256); err != nil {
		return nil, "", "", 0, err
	}
	return sample, key, receiptRef.SHA256, int64(len(receiptBytes)), nil
}

func mediaID(m *shotsv1.ResolvedSourceMedia) string {
	if m == nil {
		return ""
	}
	return m.GetId()
}

type videoSubjectAdapter struct{}

func (videoSubjectAdapter) prepare(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt) (preparedOperation, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return preparedOperation{}, err
	}
	closureBinding := bindingByRole(job.Input, "video_closure")
	if closureBinding.ArtifactID == "" || !validDigest(closureBinding.SHA256) {
		return preparedOperation{}, ErrUnknownOutcome
	}
	closureBytes, err := c.readBlob(ctx, closureBinding.ArtifactID, maxManifestBytes, closureBinding.SHA256)
	if err != nil {
		return preparedOperation{}, err
	}
	var closure videoClosure
	if err := decodeJSON(closureBytes, &closure); err != nil {
		return preparedOperation{}, err
	}
	canonical, err := json.Marshal(closure)
	if err != nil || !bytes.Equal(closureBytes, append(canonical, '\n')) || closure.Version != 1 || closure.ProjectID != job.Input.ProjectID || closure.ShotID != job.Input.SubjectID || closure.ShotRevision != job.Input.ShotRevision || closure.RecipeSHA256 != bindingDigest(job.Input, "shot_snapshot") {
		return preparedOperation{}, errors.New("renderjobs: video closure does not match immutable job input")
	}
	manifest := closure.Manifest
	manifest.JobID, manifest.AttemptID, manifest.InputFingerprint = job.ID, attempt.ID, job.InputFingerprint
	manifest.WorkerSHA256 = bindingDigest(job.Input, "worker_executable")
	if manifest.WorkerSHA256 != closure.WorkerSHA256 {
		return preparedOperation{}, ErrRouteMismatch
	}
	files := make([]stagedFile, 0, 10)
	// The media owner remains authoritative for source and plate bytes. Each
	// stream is reopened only after the durable attempt claim and is verified by
	// the shared workspace writer against this immutable job binding.
	openMedia := func(name, role, mediaID string, sha string, size int64, kind projects.MediaKind, max int64) error {
		if mediaID == "" || !validDigest(sha) || size <= 0 {
			return ErrNotEligible
		}
		b := bindingByRole(job.Input, role)
		if b.ArtifactID != mediaID || b.SHA256 != sha {
			return ErrRouteMismatch
		}
		files = append(files, stagedFile{Name: name, SHA256: sha, Size: size, Mode: manifestMode, MaxBytes: max, Open: func(openCtx context.Context) (io.ReadCloser, error) {
			opened, reader, openErr := c.media.Open(openCtx, job.Input.ProjectID, mediaID)
			if openErr != nil {
				return nil, openErr
			}
			if opened.ID != mediaID || opened.ProjectID != job.Input.ProjectID || opened.SHA256 != sha || opened.SizeBytes != size || opened.Kind != kind {
				_ = reader.Close()
				return nil, errors.New("renderjobs: source owner returned changed video input")
			}
			return reader, nil
		}})
		return nil
	}
	if err := openMedia(manifest.Inspection.Source.Name, "source_media", closure.SourceMediaID, manifest.Inspection.Source.SHA256, manifest.Inspection.Source.SizeBytes, projects.MediaKindVideo, maxInspectionInputBytes); err != nil {
		return preparedOperation{}, err
	}
	if err := openMedia(manifest.Scene.Plate.Name, "video_plate", closure.PlateMediaID, manifest.Scene.Plate.SHA256, manifest.Scene.Plate.SizeBytes, projects.MediaKindImage, 24<<20); err != nil {
		return preparedOperation{}, err
	}
	if err := openMedia(manifest.Scene.ActiveMask.Name, "video_active_mask", closure.ActiveMaskMediaID, manifest.Scene.ActiveMask.SHA256, manifest.Scene.ActiveMask.SizeBytes, projects.MediaKindImage, 24<<20); err != nil {
		return preparedOperation{}, err
	}
	if manifest.Scene.HardwareMask != nil {
		if err := openMedia(manifest.Scene.HardwareMask.Name, "video_hardware_mask", closure.HardwareMaskID, manifest.Scene.HardwareMask.SHA256, manifest.Scene.HardwareMask.SizeBytes, projects.MediaKindImage, 24<<20); err != nil {
			return preparedOperation{}, err
		}
	}
	for _, item := range []struct {
		name, role string
		max        int64
	}{
		{manifest.Inspection.Manifest.Name, "inspection_manifest", maxManifestBytes},
		{rendering.InspectionReceiptName, "inspection_receipt", maxReceiptBytes},
		{rendering.InspectionClockName, "inspection_clock", maxInspectionClockBytes},
		{rendering.VideoProfileName, "renderer_tool_profile", 64 << 10},
	} {
		b := bindingByRole(job.Input, item.role)
		if b.ArtifactID == "" || !validDigest(b.SHA256) {
			return preparedOperation{}, ErrUnknownOutcome
		}
		data, err := c.readBlob(ctx, b.ArtifactID, item.max, b.SHA256)
		if err != nil {
			return preparedOperation{}, err
		}
		if len(data) == 0 {
			return preparedOperation{}, errors.New("renderjobs: pinned video sidecar is empty")
		}
		files = append(files, stagedFile{Name: item.name, Data: data, SHA256: b.SHA256, Size: int64(len(data)), Mode: manifestMode, MaxBytes: item.max})
	}
	if manifest.Sample != nil {
		b := bindingByRole(job.Input, "video_sample_receipt")
		if b.ArtifactID == "" || !validDigest(b.SHA256) {
			return preparedOperation{}, ErrUnknownOutcome
		}
		manifest.Sample.ReceiptArtifact = rendering.VideoBinding{Name: "sample-receipt.json", SHA256: b.SHA256, SizeBytes: closure.SampleReceiptSize}
		receiptBytes, err := c.readBlob(ctx, b.ArtifactID, maxReceiptBytes, b.SHA256)
		if err != nil {
			return preparedOperation{}, err
		}
		files = append(files, stagedFile{Name: "sample-receipt.json", Data: receiptBytes, SHA256: b.SHA256, Size: closure.SampleReceiptSize, Mode: manifestMode, MaxBytes: maxReceiptBytes})
	}
	if err := rendering.ValidateVideoManifest(manifest); err != nil {
		return preparedOperation{}, fmt.Errorf("%w: prepared video manifest is not worker-valid: %v", errDeterministicPreparation, err)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return preparedOperation{}, err
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := c.requireRoute(ctx, route); err != nil {
		return preparedOperation{}, err
	}
	return preparedOperation{Mode: operationVideoRender, ManifestBytes: manifestBytes, Files: files, MaxInputBytes: maxVideoInputBytes}, nil
}

func (videoSubjectAdapter) validateCompletion(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt, intent persistedIntent, nativeKey, nativeSHA string) (jobs.CompletionEvidence, error) {
	if err := validateNativeEvidence(ctx, c, route, job, attempt, intent, operationVideoRender, nativeKey, nativeSHA); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	workspace, err := attemptWorkspace(route.Root, job.ID, attempt.ID)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	manifestBytes, err := readWorkspaceFile(workspace, rendering.VideoManifestName, maxManifestBytes)
	if err != nil || digestBytes(manifestBytes) != intent.ManifestSHA256 {
		return jobs.CompletionEvidence{}, errors.Join(errors.New("renderjobs: staged video manifest differs from persisted intent"), err)
	}
	receiptBytes, err := readWorkspaceFile(workspace, rendering.VideoReceiptName, maxReceiptBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	var receipt rendering.VideoReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	outputName := rendering.VideoOutputName
	mime := "video/mp4"
	kind := "video_mp4"
	max := maxVideoOutputBytes
	if receipt.Mode == rendering.VideoModeSample {
		outputName, mime, kind, max = rendering.VideoSampleName, "image/png", "video_sample_png", maxOutputBytes
	}
	file, err := os.Open(filepath.Join(workspace, outputName))
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > max {
		_ = file.Close()
		return jobs.CompletionEvidence{}, errors.New("renderjobs: video output size is invalid")
	}
	verified, verifyErr := rendering.VerifyVideoArtifacts(ctx, manifestBytes, receiptBytes, file, info.Size())
	closeErr := file.Close()
	if verifyErr != nil || closeErr != nil {
		return jobs.CompletionEvidence{}, errors.Join(verifyErr, closeErr)
	}
	if verified.JobID != job.ID || verified.AttemptID != attempt.ID || verified.InputFingerprint != job.InputFingerprint || verified.WorkerSHA256 != intent.WorkerSHA256 || verified.ManifestSHA256 != intent.ManifestSHA256 || verified.Mode != receipt.Mode {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: verified video receipt does not match the claimed job attempt")
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	key := outputBlobPrefix + job.ID + "/" + attempt.ID + "/" + verified.OutputSHA256 + outputSuffix(receipt.Mode)
	if err := c.publishWorkspaceOutput(ctx, route, workspace, outputName, key, verified.OutputSizeBytes, verified.OutputSHA256, mime); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	manifestSHA := digestBytes(manifestBytes)
	manifestKey := receiptBlobPrefix + job.ID + "/" + attempt.ID + "/manifest-" + manifestSHA + ".json"
	if err := c.putVerified(ctx, route, manifestKey, manifestBytes, "application/json", manifestSHA); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	receiptSHA := digestBytes(receiptBytes)
	receiptKey := receiptBlobPrefix + job.ID + "/" + attempt.ID + "/video-" + receiptSHA + ".json"
	if err := c.putVerified(ctx, route, receiptKey, receiptBytes, "application/json", receiptSHA); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	return jobs.CompletionEvidence{AttemptID: attempt.ID, ExecutionClaimID: attempt.Execution.ID, InputFingerprint: job.InputFingerprint, Outputs: []jobs.OutputRef{{Kind: kind, ArtifactID: key, SHA256: verified.OutputSHA256, MIMEType: mime, SizeBytes: verified.OutputSizeBytes, Width: verified.OutputWidth, Height: verified.OutputHeight}, {Kind: "video_receipt", ArtifactID: receiptKey, SHA256: receiptSHA, MIMEType: "application/json", SizeBytes: int64(len(receiptBytes))}, {Kind: "video_manifest", ArtifactID: manifestKey, SHA256: manifestSHA, MIMEType: "application/json", SizeBytes: int64(len(manifestBytes))}}, ReceiptArtifactID: receiptKey, ReceiptSHA256: receiptSHA, CompletedAt: c.now().UTC()}, nil
}

func (videoSubjectAdapter) finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error {
	return nil
}

var _ shots.VideoSampleVerifier = (*Coordinator)(nil)

func outputSuffix(mode rendering.VideoMode) string {
	if mode == rendering.VideoModeSample {
		return ".png"
	}
	return ".mp4"
}

func (c *Coordinator) publishWorkspaceOutput(ctx context.Context, route routePin, workspace, name, key string, size int64, sha, mime string) error {
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	file, err := os.Open(filepath.Join(workspace, name))
	if err != nil {
		return err
	}
	defer file.Close()
	h := sha256.New()
	r := &countingReader{r: io.TeeReader(io.LimitReader(file, size+1), h)}
	if err := c.blobs.Put(ctx, key, r, mime); err != nil {
		return err
	}
	if r.n != size || hex.EncodeToString(h.Sum(nil)) != sha {
		return errors.New("renderjobs: output changed while publishing immutable result")
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	return nil
}

// OpenVideoOutput opens only a succeeded video subject's current attempt output.
// Bytes remain streamed from the routed blob owner; the wrapper verifies exact
// size and digest at EOF and reports an error if a consumer stops early.
func (c *Coordinator) OpenVideoOutput(ctx context.Context, projectID, jobID string) (jobs.Job, jobs.OutputRef, io.ReadCloser, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	keepContext := false
	defer func() {
		if !keepContext {
			cancel()
		}
	}()
	route, err := c.route(streamCtx)
	if err != nil {
		return jobs.Job{}, jobs.OutputRef{}, nil, err
	}
	job, err := c.jobs.Get(streamCtx, jobID)
	if err != nil {
		return jobs.Job{}, jobs.OutputRef{}, nil, err
	}
	if err := verifyJobRoute(job, route); err != nil {
		return jobs.Job{}, jobs.OutputRef{}, nil, err
	}
	physicalMotion := job.Input.SubjectKind == jobs.SubjectPhysicalRender && (job.Input.OutputProfile == OutputProfilePhysicalMotionMP4 || job.Input.OutputProfile == OutputProfilePhysicalMotionMP4V2 || job.Input.OutputProfile == OutputProfilePhysicalRecordingMP4 || job.Input.OutputProfile == OutputProfilePhysicalDiagnosticMotionMP4)
	if job.Input.ProjectID != projectID || job.Input.SubjectKind != jobs.SubjectVideoRender && !physicalMotion {
		return jobs.Job{}, jobs.OutputRef{}, nil, jobs.ErrNotFound
	}
	if job.State != jobs.Succeeded {
		return jobs.Job{}, jobs.OutputRef{}, nil, ErrOutputUnavailable
	}
	attempt, ok := currentAttempt(job)
	if !ok || attempt.State != jobs.AttemptSucceeded || attempt.Completion == nil || attempt.Completion.AttemptID != attempt.ID {
		return jobs.Job{}, jobs.OutputRef{}, nil, ErrOutputUnavailable
	}
	wantedKind := "video_mp4"
	if physicalMotion {
		wantedKind = "physical_motion_mp4"
	} else if job.Input.OutputProfile == videoSampleOutputProfile {
		wantedKind = "video_sample_png"
	}
	var ref *jobs.OutputRef
	for i := range attempt.Completion.Outputs {
		if attempt.Completion.Outputs[i].Kind == wantedKind {
			ref = &attempt.Completion.Outputs[i]
			break
		}
	}
	if ref == nil || ref.ArtifactID == "" || ref.SizeBytes <= 0 || ref.SizeBytes > maxVideoOutputBytes || !validDigest(ref.SHA256) || ((wantedKind == "video_mp4" || wantedKind == "physical_motion_mp4") && ref.MIMEType != "video/mp4") || (wantedKind == "video_sample_png" && ref.MIMEType != "image/png") {
		return jobs.Job{}, jobs.OutputRef{}, nil, errors.New("renderjobs: video output metadata is invalid")
	}
	if err := c.requireRoute(streamCtx, route); err != nil {
		return jobs.Job{}, jobs.OutputRef{}, nil, err
	}
	reader, mime, err := c.blobs.Get(streamCtx, ref.ArtifactID)
	if err != nil {
		return jobs.Job{}, jobs.OutputRef{}, nil, err
	}
	if mime != ref.MIMEType {
		_ = reader.Close()
		return jobs.Job{}, jobs.OutputRef{}, nil, errors.New("renderjobs: routed video output MIME differs from completion")
	}
	keepContext = true
	return job, *ref, &verifiedOutputReader{reader: reader, expectedSize: ref.SizeBytes, expectedSHA: ref.SHA256, hash: sha256.New(), cancel: cancel}, nil
}

type verifiedOutputReader struct {
	reader       io.ReadCloser
	expectedSize int64
	expectedSHA  string
	read         int64
	hash         hash.Hash
	verified     bool
	finalByte    byte
	finalPending bool
	terminal     error
	cancel       context.CancelFunc
}

func (r *verifiedOutputReader) Read(p []byte) (int, error) {
	if r.terminal != nil {
		return 0, r.terminal
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.verified {
		if r.finalPending {
			r.finalPending = false
			p[0] = r.finalByte
			return 1, nil
		}
		return 0, io.EOF
	}
	if r.read > r.expectedSize {
		r.terminal = errors.New("renderjobs: output exceeds declared size")
		return 0, r.terminal
	}
	remaining := r.expectedSize - r.read
	if remaining == 0 {
		var probe [1]byte
		n, err := r.reader.Read(probe[:])
		if n != 0 {
			r.terminal = errors.New("renderjobs: output exceeds declared size")
			return 0, r.terminal
		}
		if err != io.EOF {
			if err == nil {
				return 0, nil
			}
			r.terminal = err
			return 0, err
		}
		if hex.EncodeToString(r.hash.Sum(nil)) != r.expectedSHA {
			r.terminal = errors.New("renderjobs: output digest mismatch")
			return 0, r.terminal
		}
		r.verified = true
		if r.finalPending {
			r.finalPending = false
			p[0] = r.finalByte
			return 1, nil
		}
		return 0, io.EOF
	}
	if remaining == 1 {
		var final [1]byte
		n, err := r.reader.Read(final[:])
		if n == 0 {
			if err == nil {
				return 0, nil
			}
			if err == io.EOF {
				r.terminal = io.ErrUnexpectedEOF
				return 0, r.terminal
			}
			r.terminal = err
			return 0, err
		}
		r.read++
		_, _ = r.hash.Write(final[:n])
		r.finalByte = final[0]
		r.finalPending = true
		if err == nil {
			for {
				var probe [1]byte
				pn, perr := r.reader.Read(probe[:])
				if pn > 0 {
					r.terminal = errors.New("renderjobs: output exceeds declared size")
					return 0, r.terminal
				}
				if perr == io.EOF {
					break
				}
				if perr != nil {
					r.terminal = perr
					return 0, perr
				}
				return 0, nil
			}
		} else if err != io.EOF {
			r.terminal = err
			return 0, err
		}
		if hex.EncodeToString(r.hash.Sum(nil)) != r.expectedSHA {
			r.terminal = errors.New("renderjobs: output digest mismatch")
			return 0, r.terminal
		}
		r.verified = true
		r.finalPending = false
		p[0] = r.finalByte
		return 1, nil
	}
	maxRead := remaining - 1
	if int64(len(p)) > maxRead {
		p = p[:maxRead]
	}
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if n > 0 {
		_, _ = r.hash.Write(p[:n])
	}
	if err != nil && err != io.EOF {
		r.terminal = err
		return n, err
	}
	if err == io.EOF && r.read != r.expectedSize {
		r.terminal = io.ErrUnexpectedEOF
		return n, r.terminal
	}
	return n, nil
}

func (r *verifiedOutputReader) Close() error {
	var verifyErr error
	if !r.verified {
		verifyErr = errors.New("renderjobs: output transfer ended before digest verification")
	}
	closeErr := r.reader.Close()
	if r.cancel != nil {
		r.cancel()
	}
	return errors.Join(verifyErr, closeErr)
}

type countingReader struct {
	r io.Reader
	n int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.n += int64(n)
	return n, err
}

var _ projects.SourceMediaService
