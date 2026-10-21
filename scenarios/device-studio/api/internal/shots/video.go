package shots

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"device-studio/internal/projects"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

const (
	videoBaseRecipeVersion = "device-studio-video-base-recipe-v1"
)

// CameraMotionProfileID identifies the typed photographic screen-entry motion profile.
const CameraMotionProfileID = "screen-entry-projective-contain-v1"

// CameraMotionProfileVersion is the schema/behavior version of CameraMotionProfileID.
const CameraMotionProfileVersion = 1

// maxCameraOutputFrames bounds screen-entry motion allocation to 12 seconds
// at the highest supported output rate, 30 frames per second.
const maxCameraOutputFrames = 360

const maxCameraDurationSeconds = 12

func maxCameraScheduleFrames(t *shotsv1.VideoTiming) int64 {
	if t == nil || t.GetOutputRateDen() != 1 || (t.GetOutputRateNum() != 24 && t.GetOutputRateNum() != 30) {
		return 0
	}
	return int64(t.GetOutputRateNum()) * maxCameraDurationSeconds
}

// VideoProfileFixed identifies the byte-compatible fixed-plate video profile.
const VideoProfileFixed = "device-studio-fixed-plate-video-v1"

// VideoProfileScreenEntry identifies the separate photographic screen-entry profile.
const VideoProfileScreenEntry = "device-studio-screen-entry-video-v1"

func VideoBaseRecipeSHA(snapshot *shotsv1.ShotSnapshot) (string, error) {
	return videoBaseRecipeSHA(snapshot)
}

var ErrVideoSampleUnavailable = errors.New("verified video sample is unavailable")

// videoBaseRecipeSHA excludes only the sample claim and the colors/inks that
// claim resolves. All source, timing, scene and presentation choices remain
// part of the canonical base recipe.
func videoBaseRecipeSHA(snapshot *shotsv1.ShotSnapshot) (string, error) {
	if snapshot == nil || snapshot.GetVideoTiming() == nil {
		return "", ErrInvalidShot{Field: "video_timing", Reason: "required for a video base recipe"}
	}
	base := proto.Clone(snapshot).(*shotsv1.ShotSnapshot)
	timing := base.GetVideoTiming()
	claim := timing.GetSampleBinding()
	timing.SampleBinding = nil
	timing.SampleBaseRecipeVersion = 0
	if chrome := base.GetPresentation().GetPhoneSystemChrome(); chrome != nil {
		if claim != nil {
			if claim.GetTopFillSourceResolved() {
				chrome.TopFill = nil
			}
			if claim.GetBottomFillSourceResolved() {
				chrome.BottomFill = nil
			}
			if claim.GetStatusInkSourceResolved() {
				chrome.StatusInk = nil
			}
			if claim.GetHomeInkSourceResolved() {
				chrome.HomeInk = nil
			}
		}
	}
	data, err := (proto.MarshalOptions{Deterministic: true}).Marshal(base)
	if err != nil {
		return "", err
	}
	data = append([]byte(videoBaseRecipeVersion+"\x00"), data...)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func applySampledPhoneValues(snapshot *shotsv1.ShotSnapshot) error {
	timing := snapshot.GetVideoTiming()
	if timing == nil || timing.GetSampleBinding() == nil {
		return nil
	}
	presentation := snapshot.GetPresentation()
	if presentation == nil || presentation.GetKind() != shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM {
		return ErrInvalidShot{Field: "video_timing.sample_binding", Reason: "sampled colors require phone-system presentation"}
	}
	chrome := presentation.GetPhoneSystemChrome()
	if chrome == nil {
		chrome = &shotsv1.PhoneSystemChrome{}
		presentation.PhoneSystemChrome = chrome
	}
	b := timing.GetSampleBinding()
	values := []struct {
		dst     **shotsv1.PresentationRGB
		v       [3]uint32
		derived bool
	}{
		{&chrome.TopFill, [3]uint32{b.GetResolvedTopFillRed(), b.GetResolvedTopFillGreen(), b.GetResolvedTopFillBlue()}, b.GetTopFillSourceResolved()},
		{&chrome.BottomFill, [3]uint32{b.GetResolvedBottomFillRed(), b.GetResolvedBottomFillGreen(), b.GetResolvedBottomFillBlue()}, b.GetBottomFillSourceResolved()},
		{&chrome.StatusInk, [3]uint32{b.GetResolvedStatusInkRed(), b.GetResolvedStatusInkGreen(), b.GetResolvedStatusInkBlue()}, b.GetStatusInkSourceResolved()},
		{&chrome.HomeInk, [3]uint32{b.GetResolvedHomeInkRed(), b.GetResolvedHomeInkGreen(), b.GetResolvedHomeInkBlue()}, b.GetHomeInkSourceResolved()},
	}
	for _, item := range values {
		for _, component := range item.v {
			if component > 255 {
				return ErrInvalidShot{Field: "video_timing.sample_binding", Reason: "resolved color component exceeds 255"}
			}
		}
		resolved := &shotsv1.PresentationRGB{Red: item.v[0], Green: item.v[1], Blue: item.v[2]}
		if !item.derived && (*item.dst == nil || !proto.Equal(*item.dst, resolved)) {
			return ErrInvalidShot{Field: "presentation.phone_system_chrome", Reason: "saved colors do not match the verified trim-start sample"}
		}
		if item.derived {
			if *item.dst != nil && !proto.Equal(*item.dst, resolved) {
				return ErrInvalidShot{Field: "video_timing.sample_binding", Reason: "source-resolved color conflicts with an explicit saved color"}
			}
			if *item.dst == nil {
				*item.dst = resolved
			}
		}
	}
	timing.SampleBaseRecipeVersion = 1
	return nil
}

func validateVideoTiming(t *shotsv1.VideoTiming, source projects.SourceMedia, cameraMotion bool) *shotsv1.ShotIssue {
	issue := func(code shotsv1.ShotIssueCode, field, summary, repair string) *shotsv1.ShotIssue {
		return &shotsv1.ShotIssue{Code: code, FieldPath: field, Summary: summary, Repair: repair, BlocksRender: true}
	}
	if source.Readiness != projects.ReadinessReady || source.InspectionReceipt == nil || source.InspectionReceipt.Classification != "accepted" {
		return issue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA, "video_timing", "The source video has not passed inspection.", "Wait for inspection or retry it before saving a renderable revision.")
	}
	r := source.InspectionReceipt
	if t.GetInspectionJobId() == "" || t.GetInspectionAttemptId() == "" || t.GetInspectionReceiptId() == "" || t.GetInspectionReceiptSha256() == "" || t.GetClockArtifactId() == "" || t.GetClockSha256() == "" || t.GetClockSizeBytes() <= 0 || t.StreamStartPts == nil || t.GetSourceEndPts() <= 0 || t.GetLastFrameDurationTicks() <= 0 || t.GetSelectedVideoStream() == 0 && r.VideoStreamIndex != nil && *r.VideoStreamIndex != 0 || t.GetTimeBaseNum() <= 0 || t.GetTimeBaseDen() <= 0 || t.GetTrimEndPts() <= t.GetTrimStartPts() || t.GetPlaybackRateNum() == 0 || t.GetPlaybackRateDen() == 0 || t.GetOutputRateDen() != 1 || (t.GetOutputRateNum() != 24 && t.GetOutputRateNum() != 30) {
		return issue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_DRAFT_INCOMPLETE, "video_timing", "A complete inspected source-clock trim and output rate are required.", "Choose a trim inside the inspected source and select 24 or 30 fps.")
	}
	wantProfile := VideoProfileFixed
	if cameraMotion {
		wantProfile = VideoProfileScreenEntry
	}
	if t.GetEofPolicy() != shotsv1.VideoEOFPolicy_VIDEO_EOF_POLICY_REJECT || t.GetVideoCompositionProfileId() != wantProfile || t.GetVideoCompositionProfileVersion() != 1 {
		return issue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_PROFILE, "video_timing", "The selected video timing profile is unsupported for this motion mode.", "Use the fixed-plate profile for fixed motion or the screen-entry profile for crop-push.")
	}
	if r.JobID != t.GetInspectionJobId() || r.AttemptID != t.GetInspectionAttemptId() || r.ID != t.GetInspectionReceiptId() || r.SHA256 != t.GetInspectionReceiptSha256() || r.SourceSHA256 != source.SHA256 || r.SourceSizeBytes != source.SizeBytes || r.ClockSHA256 != t.GetClockSha256() || r.ClockArtifactRef != t.GetClockArtifactId() || r.ClockArtifactSizeBytes != t.GetClockSizeBytes() || r.VideoStreamIndex == nil || uint32(*r.VideoStreamIndex) != t.GetSelectedVideoStream() || r.TimeBaseNumerator != t.GetTimeBaseNum() || r.TimeBaseDenominator != t.GetTimeBaseDen() || r.StreamStartPTS == nil || *r.StreamStartPTS != t.GetStreamStartPts() || r.SourceEndPTS == nil || *r.SourceEndPTS != t.GetSourceEndPts() || r.LastFrameDurationTicks != t.GetLastFrameDurationTicks() {
		return issue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA, "video_timing.inspection_receipt", "Saved timing facts do not match the accepted inspection receipt.", "Reload source inspection facts and save the revision again.")
	}
	if t.GetTrimStartPts() < t.GetStreamStartPts() || t.GetTrimEndPts() > t.GetSourceEndPts() {
		return issue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_MALFORMED_INPUT, "video_timing.trim", "Trim endpoints must lie inside the inspected source clock.", "Choose endpoints within the accepted source range.")
	}
	if !validVideoDigest(t.GetSourceSha256()) || !validVideoDigest(t.GetClockSha256()) {
		return issue(shotsv1.ShotIssueCode_SHOT_ISSUE_CODE_MALFORMED_INPUT, "video_timing", "Source and clock digests are malformed.", "Reload the source metadata.")
	}
	return nil
}

// plannedVideoFrameCount mirrors the count arithmetic in composition.PlanFrames
// without selecting source frames. Source selection remains exclusively owned
// by PlanFrames; this helper only enforces the camera duration bound at save.
func plannedVideoFrameCount(t *shotsv1.VideoTiming) (int64, error) {
	if t == nil || t.GetTrimEndPts() <= t.GetTrimStartPts() || t.GetTimeBaseNum() <= 0 || t.GetTimeBaseDen() <= 0 || t.GetPlaybackRateNum() == 0 || t.GetPlaybackRateDen() == 0 || t.GetOutputRateNum() == 0 || t.GetOutputRateDen() == 0 {
		return 0, errors.New("video timing is incomplete")
	}
	delta := new(big.Int).Sub(big.NewInt(t.GetTrimEndPts()), big.NewInt(t.GetTrimStartPts()))
	duration := new(big.Rat).SetInt(delta)
	duration.Mul(duration, new(big.Rat).SetFrac(big.NewInt(t.GetTimeBaseNum()), big.NewInt(t.GetTimeBaseDen())))
	duration.Mul(duration, new(big.Rat).SetFrac(big.NewInt(int64(t.GetPlaybackRateDen())), big.NewInt(int64(t.GetPlaybackRateNum()))))
	duration.Mul(duration, new(big.Rat).SetFrac(big.NewInt(int64(t.GetOutputRateNum())), big.NewInt(int64(t.GetOutputRateDen()))))
	numerator, denominator := duration.Num(), duration.Denom()
	count := new(big.Int).Quo(new(big.Int).Add(numerator, new(big.Int).Sub(denominator, big.NewInt(1))), denominator)
	if !count.IsInt64() {
		return 0, errors.New("video output frame count overflows")
	}
	return count.Int64(), nil
}

func validVideoDigest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
