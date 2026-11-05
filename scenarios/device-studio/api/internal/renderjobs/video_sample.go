package renderjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"device-studio/internal/jobs"
	"device-studio/internal/presentations"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

// GetVideoSampleBinding returns only a server-verified claim that can be saved
// into a later immutable shot revision. The client never assembles receipt
// fields from job status or output metadata.
func (c *Coordinator) GetVideoSampleBinding(ctx context.Context, projectID, shotID, jobID string) (*shotsv1.VideoSampleBinding, error) {
	job, err := c.GetProjectJob(ctx, projectID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Input.SubjectKind != jobs.SubjectVideoRender || job.Input.SubjectID != shotID || job.Input.OutputProfile != videoSampleOutputProfile || job.State != jobs.Succeeded {
		return nil, ErrOutputUnavailable
	}
	attempt, ok := currentAttempt(job)
	if !ok || attempt.State != jobs.AttemptSucceeded || attempt.Completion == nil {
		return nil, ErrOutputUnavailable
	}
	var manifestRef, receiptRef, pngRef *jobs.OutputRef
	for i := range attempt.Completion.Outputs {
		out := &attempt.Completion.Outputs[i]
		switch out.Kind {
		case "video_manifest":
			manifestRef = out
		case "video_receipt":
			receiptRef = out
		case "video_sample_png":
			pngRef = out
		}
	}
	if manifestRef == nil || receiptRef == nil || pngRef == nil {
		return nil, ErrOutputUnavailable
	}
	manifestBytes, err := c.readBlob(ctx, manifestRef.ArtifactID, maxManifestBytes, manifestRef.SHA256)
	if err != nil {
		return nil, err
	}
	receiptBytes, err := c.readBlob(ctx, receiptRef.ArtifactID, maxReceiptBytes, receiptRef.SHA256)
	if err != nil {
		return nil, err
	}
	pngBytes, err := c.readBlob(ctx, pngRef.ArtifactID, maxOutputBytes, pngRef.SHA256)
	if err != nil {
		return nil, err
	}
	receipt, err := rendering.VerifyVideoArtifacts(ctx, manifestBytes, receiptBytes, bytes.NewReader(pngBytes), int64(len(pngBytes)))
	if err != nil {
		return nil, err
	}
	var manifest rendering.VideoManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, err
	}
	if manifest.Presentation == nil || !isAutoPhoneVideoSample(manifest.Mode, *manifest.Presentation) {
		return nil, ErrOutputUnavailable
	}
	base, err := c.shots.ResolveForRender(ctx, shotID, job.Input.ShotRevision)
	if err != nil {
		return nil, err
	}
	baseSHA, err := shots.VideoBaseRecipeSHA(base.Revision.Snapshot)
	if err != nil {
		return nil, err
	}
	if base.ProjectID != projectID || base.Revision.Revision != job.Input.ShotRevision || receipt.Mode != rendering.VideoModeSample || receipt.JobID != job.ID || receipt.AttemptID != attempt.ID || receipt.InputFingerprint != job.InputFingerprint || receipt.ManifestSHA256 != manifestRef.SHA256 || receipt.InspectionSHA256 != rendering.VideoInspectionDigest(manifest.Inspection) || receipt.RecipeSHA256 != base.Revision.RecipeSHA256 || receipt.BaseRecipeSHA256 != baseSHA || receipt.ShotRevision != job.Input.ShotRevision || receipt.OutputSHA256 != pngRef.SHA256 || receipt.OutputSizeBytes != pngRef.SizeBytes || receipt.OutputWidth != int(pngRef.Width) || receipt.OutputHeight != int(pngRef.Height) {
		return nil, ErrOutputUnavailable
	}
	clockBinding := bindingByRole(job.Input, "inspection_clock")
	clockBytes, err := c.readBlob(ctx, clockBinding.ArtifactID, maxInspectionClockBytes, clockBinding.SHA256)
	if err != nil {
		return nil, err
	}
	ordinal, sourcePTS, err := rendering.PlanVideoSample(ctx, manifest, clockBytes)
	if err != nil {
		return nil, err
	}
	claim := &shotsv1.VideoSampleBinding{
		JobId: job.ID, AttemptId: attempt.ID, ReceiptId: job.ID + ":" + attempt.ID,
		ReceiptSha256: receiptRef.SHA256, SourceSha256: manifest.Inspection.Source.SHA256, SourceSizeBytes: manifest.Inspection.Source.SizeBytes,
		ClockSha256: manifest.Inspection.Clock.SHA256, TrimStartPts: manifest.TrimStartPTS, TrimEndPts: manifest.TrimEndPTS,
		VideoCompositionProfileId: manifest.VideoProfile, VideoCompositionProfileVersion: 1, SourceOrdinal: ordinal, SourcePts: sourcePTS,
		ProjectedPngSha256: pngRef.SHA256, ProjectedPngSizeBytes: pngRef.SizeBytes, ProjectedPngWidth: uint32(pngRef.Width), ProjectedPngHeight: uint32(pngRef.Height),
		ComposedFrameSha256: receipt.FirstFrameRGBSHA256, ComposedFrameWidth: uint32(receipt.OutputWidth), ComposedFrameHeight: uint32(receipt.OutputHeight), PixelProfile: receipt.PixelProfile,
		BaseRecipeSha256: receipt.BaseRecipeSHA256, BaseShotRevision: receipt.BaseShotRevision, BaseRecipeVersion: receipt.BaseRecipeVersion,
		ResolvedTopFillRed: uint32(receipt.ResolvedTopFill.R), ResolvedTopFillGreen: uint32(receipt.ResolvedTopFill.G), ResolvedTopFillBlue: uint32(receipt.ResolvedTopFill.B),
		ResolvedBottomFillRed: uint32(receipt.ResolvedBottomFill.R), ResolvedBottomFillGreen: uint32(receipt.ResolvedBottomFill.G), ResolvedBottomFillBlue: uint32(receipt.ResolvedBottomFill.B),
		TopFillSourceResolved: receipt.TopFillSourceResolved, BottomFillSourceResolved: receipt.BottomFillSourceResolved, StatusInkSourceResolved: receipt.StatusInkSourceResolved, HomeInkSourceResolved: receipt.HomeInkSourceResolved,
	}
	if receipt.ResolvedStatusInk != nil {
		claim.ResolvedStatusInkRed = uint32(receipt.ResolvedStatusInk.R)
		claim.ResolvedStatusInkGreen = uint32(receipt.ResolvedStatusInk.G)
		claim.ResolvedStatusInkBlue = uint32(receipt.ResolvedStatusInk.B)
	}
	if receipt.ResolvedHomeInk != nil {
		claim.ResolvedHomeInkRed = uint32(receipt.ResolvedHomeInk.R)
		claim.ResolvedHomeInkGreen = uint32(receipt.ResolvedHomeInk.G)
		claim.ResolvedHomeInkBlue = uint32(receipt.ResolvedHomeInk.B)
	}
	return claim, nil
}

// VerifyVideoSample binds a saved revision to the immutable sample attempt and
// worker receipt. It never treats a job's public state alone as sample proof.
func (c *Coordinator) VerifyVideoSample(ctx context.Context, projectID, shotID string, revision int64, snapshot *shotsv1.ShotSnapshot, baseSHA string, claim *shotsv1.VideoSampleBinding) error {
	if claim == nil || snapshot == nil || shotID == "" || claim.GetJobId() == "" || claim.GetAttemptId() == "" || !validDigest(claim.GetReceiptSha256()) || !validDigest(baseSHA) || claim.GetBaseShotRevision() <= 0 || claim.GetBaseShotRevision() >= revision {
		return shots.ErrVideoSampleUnavailable
	}
	job, err := c.jobs.Get(ctx, claim.GetJobId())
	if err != nil || job.Input.ProjectID != projectID || job.Input.SubjectKind != jobs.SubjectVideoRender || job.Input.SubjectID != shotID || job.Input.ShotRevisionID != shotRevisionID(shotID, claim.GetBaseShotRevision()) || job.Input.ShotRevision != claim.GetBaseShotRevision() || job.Input.OutputProfile != videoSampleOutputProfile || job.State != jobs.Succeeded {
		return errors.Join(shots.ErrVideoSampleUnavailable, err)
	}
	var attempt *jobs.Attempt
	for i := range job.Attempts {
		if job.Attempts[i].ID == claim.GetAttemptId() {
			attempt = &job.Attempts[i]
			break
		}
	}
	if attempt == nil || attempt.Completion == nil || attempt.State != jobs.AttemptSucceeded || attempt.Completion.AttemptID != attempt.ID {
		return shots.ErrVideoSampleUnavailable
	}
	manifestRef := uniqueCompletedOutput(*attempt, "video_manifest")
	receiptRef := uniqueCompletedOutput(*attempt, "video_receipt")
	outputRef := uniqueCompletedOutput(*attempt, "video_sample_png")
	if manifestRef == nil || receiptRef == nil || outputRef == nil || receiptRef.SHA256 != claim.GetReceiptSha256() || outputRef.MIMEType != "image/png" || outputRef.SizeBytes != claim.GetProjectedPngSizeBytes() || outputRef.SHA256 != claim.GetProjectedPngSha256() || int64(outputRef.Width) != int64(claim.GetProjectedPngWidth()) || int64(outputRef.Height) != int64(claim.GetProjectedPngHeight()) {
		return shots.ErrVideoSampleUnavailable
	}
	if manifestRef.ArtifactID == "" || !validDigest(manifestRef.SHA256) {
		return shots.ErrVideoSampleUnavailable
	}
	manifest, err := c.readBlob(ctx, manifestRef.ArtifactID, maxManifestBytes, manifestRef.SHA256)
	if err != nil {
		return errors.Join(shots.ErrVideoSampleUnavailable, err)
	}
	receipt, err := c.readBlob(ctx, receiptRef.ArtifactID, maxReceiptBytes, receiptRef.SHA256)
	if err != nil {
		return errors.Join(shots.ErrVideoSampleUnavailable, err)
	}
	pngBytes, err := c.readBlob(ctx, outputRef.ArtifactID, maxOutputBytes, outputRef.SHA256)
	if err != nil {
		return errors.Join(shots.ErrVideoSampleUnavailable, err)
	}
	verified, err := rendering.VerifyVideoArtifacts(ctx, manifest, receipt, bytes.NewReader(pngBytes), int64(len(pngBytes)))
	if err != nil {
		return fmt.Errorf("%w: verify sample result: %v", shots.ErrVideoSampleUnavailable, err)
	}
	var frozen rendering.VideoManifest
	if err := json.Unmarshal(manifest, &frozen); err != nil {
		return shots.ErrVideoSampleUnavailable
	}
	timing := snapshot.GetVideoTiming()
	clockBinding := bindingByRole(job.Input, "inspection_clock")
	clockBytes, clockErr := c.readBlob(ctx, clockBinding.ArtifactID, maxInspectionClockBytes, clockBinding.SHA256)
	if clockErr != nil || clockBinding.SHA256 != claim.GetClockSha256() {
		return errors.Join(shots.ErrVideoSampleUnavailable, clockErr)
	}
	sourceOrdinal, sourcePTS, planErr := rendering.PlanVideoSample(ctx, frozen, clockBytes)
	if planErr != nil {
		return fmt.Errorf("%w: verify trim-start source frame: %v", shots.ErrVideoSampleUnavailable, planErr)
	}
	chrome := snapshot.GetPresentation().GetPhoneSystemChrome()
	if chrome == nil {
		return shots.ErrVideoSampleUnavailable
	}
	claimStatus, claimHome := rgbClaim(claim.GetResolvedStatusInkRed(), claim.GetResolvedStatusInkGreen(), claim.GetResolvedStatusInkBlue()), rgbClaim(claim.GetResolvedHomeInkRed(), claim.GetResolvedHomeInkGreen(), claim.GetResolvedHomeInkBlue())
	expectedStatus, expectedHome := presentationRGB(chrome.GetStatusInk()), presentationRGB(chrome.GetHomeInk())
	if claim.GetStatusInkSourceResolved() {
		expectedStatus = &claimStatus
	}
	if claim.GetHomeInkSourceResolved() {
		expectedHome = &claimHome
	}
	if timing == nil || !videoSampleSourceTimingMatchesManifest(claim, frozen) || verified.Mode != rendering.VideoModeSample || verified.JobID != job.ID || verified.AttemptID != attempt.ID || claim.GetReceiptId() != job.ID+":"+attempt.ID || verified.InputFingerprint != job.InputFingerprint || verified.ManifestSHA256 != manifestRef.SHA256 || verified.InspectionSHA256 != rendering.VideoInspectionDigest(frozen.Inspection) || verified.ShotRevision != claim.GetBaseShotRevision() || verified.BaseShotRevision != claim.GetBaseShotRevision() || verified.BaseRecipeSHA256 != baseSHA || verified.BaseRecipeVersion != claim.GetBaseRecipeVersion() || verified.FirstFrameRGBSHA256 != claim.GetComposedFrameSha256() || verified.OutputSHA256 != outputRef.SHA256 || verified.OutputSizeBytes != outputRef.SizeBytes || verified.OutputWidth != int(outputRef.Width) || verified.OutputHeight != int(outputRef.Height) || verified.ResolvedTopFill != rgbClaim(claim.GetResolvedTopFillRed(), claim.GetResolvedTopFillGreen(), claim.GetResolvedTopFillBlue()) || verified.ResolvedBottomFill != rgbClaim(claim.GetResolvedBottomFillRed(), claim.GetResolvedBottomFillGreen(), claim.GetResolvedBottomFillBlue()) || !sameOptionalRGB(verified.ResolvedStatusInk, expectedStatus) || !sameOptionalRGB(verified.ResolvedHomeInk, expectedHome) || verified.TopFillSourceResolved != claim.GetTopFillSourceResolved() || verified.BottomFillSourceResolved != claim.GetBottomFillSourceResolved() || verified.StatusInkSourceResolved != claim.GetStatusInkSourceResolved() || verified.HomeInkSourceResolved != claim.GetHomeInkSourceResolved() || verified.PixelProfile != claim.GetPixelProfile() || claim.GetVideoCompositionProfileId() != frozen.VideoProfile || claim.GetVideoCompositionProfileVersion() != uint32(frozen.Version) || verified.OutputWidth != int(claim.GetProjectedPngWidth()) || verified.OutputHeight != int(claim.GetProjectedPngHeight()) || verified.OutputWidth != int(claim.GetComposedFrameWidth()) || verified.OutputHeight != int(claim.GetComposedFrameHeight()) || verified.FirstFrameRGBSHA256 != claim.GetComposedFrameSha256() || sourceOrdinal != claim.GetSourceOrdinal() || sourcePTS != claim.GetSourcePts() || claim.GetSourceSizeBytes() != frozen.Inspection.Source.SizeBytes || frozen.ShotRevision != claim.GetBaseShotRevision() || frozen.BaseShotRevision != claim.GetBaseShotRevision() || frozen.BaseRecipeSHA256 != baseSHA || frozen.BaseRecipeVersion != claim.GetBaseRecipeVersion() || frozen.RecipeSHA256 != verified.RecipeSHA256 || frozen.Inspection.Source.SHA256 != timing.GetSourceSha256() || frozen.Inspection.Source.SizeBytes != timing.GetSourceSizeBytes() || frozen.Inspection.Clock.SHA256 != timing.GetClockSha256() || frozen.TrimStartPTS != timing.GetTrimStartPts() || frozen.TrimEndPTS != timing.GetTrimEndPts() {
		return shots.ErrVideoSampleUnavailable
	}
	return nil
}

func videoSampleSourceTimingMatchesManifest(claim *shotsv1.VideoSampleBinding, manifest rendering.VideoManifest) bool {
	return claim != nil && claim.GetSourceSha256() == manifest.Inspection.Source.SHA256 &&
		claim.GetTrimStartPts() == manifest.TrimStartPTS && claim.GetTrimEndPts() == manifest.TrimEndPTS
}

func uniqueCompletedOutput(attempt jobs.Attempt, kind string) *jobs.OutputRef {
	if attempt.Completion == nil || kind == "" {
		return nil
	}
	var found *jobs.OutputRef
	for i := range attempt.Completion.Outputs {
		if attempt.Completion.Outputs[i].Kind != kind {
			continue
		}
		if found != nil {
			return nil
		}
		found = &attempt.Completion.Outputs[i]
	}
	return found
}

func rgbClaim(r, g, b uint32) presentations.RGB {
	return presentations.RGB{R: uint8(r), G: uint8(g), B: uint8(b)}
}

func presentationRGB(v *shotsv1.PresentationRGB) *presentations.RGB {
	if v == nil {
		return nil
	}
	return &presentations.RGB{R: uint8(v.GetRed()), G: uint8(v.GetGreen()), B: uint8(v.GetBlue())}
}

func sameOptionalRGB(a, b *presentations.RGB) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
