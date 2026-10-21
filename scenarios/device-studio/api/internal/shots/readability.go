package shots

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

var ErrReadabilityReviewNotFound = errors.New("readability review not found")
var ErrReadabilityAlreadyReviewed = errors.New("this exact render output already has a readability verdict; revise the shot before changing the proof")

func validReviewDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func snapshotHasVideoOutput(snapshot *shotsv1.ShotSnapshot) bool {
	return snapshot != nil && (snapshot.GetVideoTiming() != nil || snapshot.GetPhysicalBinding().GetMotionOutput() != nil)
}

type ReadabilityOutputVerifier interface {
	VerifyReadabilityOutput(context.Context, string, string, int64, string, string) (ReadabilityOutput, error)
}

type ReviewReadabilityRequest struct {
	ProjectID, ShotID            string
	ShotRevision                 int64
	RenderID, OutputSHA256       string
	Verdict                      int32
	Note                         string
	ExplicitlyInspectedFinalSize bool
}

func validateReadabilityProof(proof *shotsv1.ReadabilityProof, video bool) error {
	if proof == nil {
		return nil
	}
	if strings.TrimSpace(proof.GetRegionName()) == "" || !validBoundedText(proof.GetRegionName(), 80) ||
		strings.TrimSpace(proof.GetRequiredContent()) == "" || !validBoundedText(proof.GetRequiredContent(), 512) ||
		strings.TrimSpace(proof.GetCriteria()) == "" || !validBoundedText(proof.GetCriteria(), 1000) {
		return ErrInvalidShot{Field: "readability_proof", Reason: "region name, required content, and criteria must be bounded non-empty text"}
	}
	x, y, w, h := proof.GetRegionX(), proof.GetRegionY(), proof.GetRegionWidth(), proof.GetRegionHeight()
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(w) || math.IsNaN(h) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(w, 0) || math.IsInf(h, 0) || x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > 1 || y+h > 1 {
		return ErrInvalidShot{Field: "readability_proof.region", Reason: "must be a finite positive rectangle inside normalized output bounds"}
	}
	if proof.GetMinimumOutputWidthPixels() == 0 || proof.GetMinimumOutputHeightPixels() == 0 || proof.GetMinimumRegionWidthPixels() == 0 || proof.GetMinimumRegionHeightPixels() == 0 || proof.GetMinimumOutputWidthPixels() > 16384 || proof.GetMinimumOutputHeightPixels() > 16384 || proof.GetMinimumRegionWidthPixels() > 16384 || proof.GetMinimumRegionHeightPixels() > 16384 {
		return ErrInvalidShot{Field: "readability_proof.minimum_pixels", Reason: "all minimum output and region dimensions must be within 1..16384 pixels"}
	}
	start, end := proof.GetProofStartMs(), proof.GetProofEndMs()
	if (!video && (start != 0 || end != 0)) || (video && (end <= start || end > 10*60*1000)) {
		return ErrInvalidShot{Field: "readability_proof.proof_interval", Reason: "stills use [0,0]; video uses a positive interval no longer than ten minutes"}
	}
	return nil
}

func readabilityProofSHA(proof *shotsv1.ReadabilityProof) (string, error) {
	if proof == nil {
		return "", ErrInvalidShot{Field: "readability_proof", Reason: "a proof declaration is required before review"}
	}
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(proof)
	if err != nil {
		return "", fmt.Errorf("encode readability proof: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func videoProofDurationMillis(timing *shotsv1.VideoTiming, motionOutput *shotsv1.PhysicalMotionOutputBinding) (uint64, error) {
	if motionOutput != nil {
		if motionOutput.GetFrameCount() == 0 || motionOutput.GetOutputRateNumerator() == 0 || motionOutput.GetOutputRateDenominator() == 0 {
			return 0, ErrInvalidShot{Field: "readability_proof.proof_interval", Reason: "physical video proof requires a bounded final frame schedule"}
		}
		duration := float64(motionOutput.GetFrameCount()) * float64(motionOutput.GetOutputRateDenominator()) / float64(motionOutput.GetOutputRateNumerator()) * 1000
		if math.IsInf(duration, 0) || math.IsNaN(duration) || duration < 1 || duration > 10*60*1000 {
			return 0, ErrInvalidShot{Field: "readability_proof.proof_interval", Reason: "physical video output duration is outside the bounded review interval"}
		}
		return uint64(math.Floor(duration)), nil
	}
	if timing == nil || timing.GetTrimEndPts() <= timing.GetTrimStartPts() || timing.GetTimeBaseNum() <= 0 || timing.GetTimeBaseDen() <= 0 || timing.GetPlaybackRateNum() <= 0 || timing.GetPlaybackRateDen() <= 0 {
		return 0, ErrInvalidShot{Field: "readability_proof.proof_interval", Reason: "video proof requires valid saved trim and timing"}
	}
	duration := float64(timing.GetTrimEndPts()-timing.GetTrimStartPts()) * float64(timing.GetTimeBaseNum()) / float64(timing.GetTimeBaseDen()) * 1000 * float64(timing.GetPlaybackRateDen()) / float64(timing.GetPlaybackRateNum())
	if math.IsInf(duration, 0) || math.IsNaN(duration) || duration < 1 || duration > 10*60*1000 {
		return 0, ErrInvalidShot{Field: "readability_proof.proof_interval", Reason: "video output duration is outside the bounded review interval"}
	}
	return uint64(math.Floor(duration)), nil
}

func validateVideoProofInterval(proof *shotsv1.ReadabilityProof, timing *shotsv1.VideoTiming, motionOutput *shotsv1.PhysicalMotionOutputBinding) error {
	duration, err := videoProofDurationMillis(timing, motionOutput)
	if err != nil {
		return err
	}
	if proof == nil || proof.GetProofEndMs() > duration {
		return ErrInvalidShot{Field: "readability_proof.proof_interval", Reason: "declared proof interval exceeds the saved video output duration"}
	}
	return nil
}

func (s *service) ReviewOutputReadability(ctx context.Context, request ReviewReadabilityRequest) (ReadabilityReview, error) {
	if strings.TrimSpace(request.ProjectID) == "" || strings.TrimSpace(request.ShotID) == "" || request.ShotRevision <= 0 || strings.TrimSpace(request.RenderID) == "" || !validReviewDigest(request.OutputSHA256) {
		return ReadabilityReview{}, ErrInvalidShot{Field: "readability_review.binding", Reason: "project, shot, positive revision, render, and output SHA-256 are required"}
	}
	if request.Verdict != int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED) && request.Verdict != int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_REJECTED) {
		return ReadabilityReview{}, ErrInvalidShot{Field: "readability_review.verdict", Reason: "choose accepted or rejected"}
	}
	if !request.ExplicitlyInspectedFinalSize {
		return ReadabilityReview{}, ErrInvalidShot{Field: "readability_review.attestation", Reason: "explicit final-size inspection is required"}
	}
	if len([]rune(request.Note)) > 1000 || !validBoundedText(request.Note, 1000) {
		return ReadabilityReview{}, ErrInvalidShot{Field: "readability_review.note", Reason: "must be bounded text without control characters"}
	}
	shot, err := s.repo.Get(ctx, request.ShotID)
	if err != nil {
		return ReadabilityReview{}, err
	}
	if shot.ProjectID != request.ProjectID {
		return ReadabilityReview{}, ErrShotNotFound{ID: request.ShotID}
	}
	revision, err := s.Compile(ctx, request.ShotID, request.ShotRevision)
	if err != nil {
		return ReadabilityReview{}, err
	}
	proof := revision.Snapshot.GetReadabilityProof()
	videoOutput := snapshotHasVideoOutput(revision.Snapshot)
	if err := validateReadabilityProof(proof, videoOutput); err != nil {
		return ReadabilityReview{}, err
	}
	proofSHA, err := readabilityProofSHA(proof)
	if err != nil {
		return ReadabilityReview{}, err
	}
	if videoOutput {
		if intervalErr := validateVideoProofInterval(proof, revision.Snapshot.GetVideoTiming(), revision.Snapshot.GetPhysicalBinding().GetMotionOutput()); intervalErr != nil {
			return ReadabilityReview{}, intervalErr
		}
	}
	if s.readability == nil {
		return ReadabilityReview{}, errors.New("readability output verifier is unavailable")
	}
	output, err := s.readability.VerifyReadabilityOutput(ctx, request.ProjectID, request.ShotID, request.ShotRevision, request.RenderID, request.OutputSHA256)
	if err != nil {
		return ReadabilityReview{}, err
	}
	if output.Width <= 0 || output.Height <= 0 || output.SHA256 != request.OutputSHA256 {
		return ReadabilityReview{}, errors.New("verified render output differs from the requested proof binding")
	}
	if !videoOutput && output.MediaType != "image/png" || videoOutput && output.MediaType != "video/mp4" {
		return ReadabilityReview{}, errors.New("output media type does not match the declared still or video proof interval")
	}
	regionWidth := uint32(math.Floor(proof.GetRegionWidth() * float64(output.Width)))
	regionHeight := uint32(math.Floor(proof.GetRegionHeight() * float64(output.Height)))
	meetsMinimum := uint32(output.Width) >= proof.GetMinimumOutputWidthPixels() && uint32(output.Height) >= proof.GetMinimumOutputHeightPixels() && regionWidth >= proof.GetMinimumRegionWidthPixels() && regionHeight >= proof.GetMinimumRegionHeightPixels()
	if request.Verdict == int32(shotsv1.ReadabilityVerdict_READABILITY_VERDICT_ACCEPTED) && !meetsMinimum {
		return ReadabilityReview{}, ErrInvalidShot{Field: "readability_review.adequacy", Reason: "this output or named proof region is below the declared minimum; it cannot be accepted"}
	}
	if previous, previousErr := s.repo.GetReadabilityReview(ctx, request.ShotID, request.ShotRevision, request.RenderID, request.OutputSHA256); previousErr == nil {
		if previous.ProjectID == request.ProjectID && previous.ProofSHA256 == proofSHA && previous.Verdict == request.Verdict && previous.Note == strings.TrimSpace(request.Note) {
			return previous, nil
		}
		return ReadabilityReview{}, ErrReadabilityAlreadyReviewed
	} else if !errors.Is(previousErr, ErrReadabilityReviewNotFound) {
		return ReadabilityReview{}, previousErr
	}
	review := ReadabilityReview{ID: s.newID(), ProjectID: request.ProjectID, ShotID: request.ShotID, ShotRevision: request.ShotRevision, RenderID: request.RenderID, OutputSHA256: request.OutputSHA256, ProofSHA256: proofSHA, Verdict: request.Verdict, Note: strings.TrimSpace(request.Note), OutputWidth: uint32(output.Width), OutputHeight: uint32(output.Height), RegionWidth: regionWidth, RegionHeight: regionHeight, ReviewedAt: s.now().UTC()}
	if err := s.repo.AppendReadabilityReview(ctx, review); err != nil {
		return ReadabilityReview{}, err
	}
	return review, nil
}

func (s *service) GetOutputReadabilityReview(ctx context.Context, projectID, shotID string, revision int64, renderID, outputSHA string) (ReadabilityReview, error) {
	shot, err := s.repo.Get(ctx, shotID)
	if err != nil {
		return ReadabilityReview{}, err
	}
	if shot.ProjectID != projectID {
		return ReadabilityReview{}, ErrShotNotFound{ID: shotID}
	}
	snapshot, err := s.Compile(ctx, shotID, revision)
	if err != nil {
		return ReadabilityReview{}, err
	}
	proofSHA, err := readabilityProofSHA(snapshot.Snapshot.GetReadabilityProof())
	if err != nil {
		return ReadabilityReview{}, err
	}
	review, err := s.repo.GetReadabilityReview(ctx, shotID, revision, renderID, outputSHA)
	if err != nil {
		return ReadabilityReview{}, err
	}
	if review.ProjectID != projectID || review.ProofSHA256 != proofSHA {
		return ReadabilityReview{}, ErrReadabilityReviewNotFound
	}
	return review, nil
}
