package renderjobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"device-studio/internal/jobs"
	"device-studio/internal/shots"
)

func (c *Coordinator) VerifyReadabilityOutput(ctx context.Context, projectID, shotID string, revision int64, renderID, outputSHA string) (shots.ReadabilityOutput, error) {
	job, err := c.GetProjectJob(ctx, projectID, renderID)
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			return shots.ReadabilityOutput{}, shots.ErrReadabilityRenderNotFound
		}
		return shots.ReadabilityOutput{}, err
	}
	if !readabilityJobMatches(job, projectID, shotID, revision) {
		return shots.ReadabilityOutput{}, shots.ErrInvalidShot{Field: "readability_review.binding", Reason: "render must be a succeeded output for this exact shot revision"}
	}
	if isFinalVideoProfile(job.Input.OutputProfile) {
		videoJob, ref, reader, err := c.OpenVideoOutput(ctx, projectID, renderID)
		if err != nil {
			if errors.Is(err, jobs.ErrNotFound) {
				return shots.ReadabilityOutput{}, shots.ErrReadabilityRenderNotFound
			}
			return shots.ReadabilityOutput{}, err
		}
		_, copyErr := io.Copy(io.Discard, reader)
		closeErr := reader.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return shots.ReadabilityOutput{}, err
		}
		if !readabilityJobMatches(videoJob, projectID, shotID, revision) {
			return shots.ReadabilityOutput{}, shots.ErrInvalidShot{Field: "readability_review.binding", Reason: "render must be a succeeded output for this exact shot revision"}
		}
		if ref.SHA256 != outputSHA {
			return shots.ReadabilityOutput{}, shots.ErrInvalidShot{Field: "readability_review.output_sha256", Reason: "must match the selected render output"}
		}
		if ref.MIMEType != "video/mp4" || ref.Width <= 0 || ref.Height <= 0 {
			return shots.ReadabilityOutput{}, errors.New("renderjobs: verified video output metadata is invalid")
		}
		return shots.ReadabilityOutput{SHA256: ref.SHA256, Width: ref.Width, Height: ref.Height, MediaType: ref.MIMEType}, nil
	}
	if !readabilityFinalStillProfile(job.Input.OutputProfile) {
		return shots.ReadabilityOutput{}, shots.ErrInvalidShot{Field: "readability_review.render_id", Reason: "must reference a final still or video output, not a preview or sample"}
	}
	job, data, err := c.OpenOutput(ctx, projectID, renderID)
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			return shots.ReadabilityOutput{}, shots.ErrReadabilityRenderNotFound
		}
		return shots.ReadabilityOutput{}, err
	}
	if !readabilityJobMatches(job, projectID, shotID, revision) {
		return shots.ReadabilityOutput{}, shots.ErrInvalidShot{Field: "readability_review.binding", Reason: "render must be a succeeded output for this exact shot revision"}
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != outputSHA {
		return shots.ReadabilityOutput{}, shots.ErrInvalidShot{Field: "readability_review.output_sha256", Reason: "must match the selected render output"}
	}
	if len(job.Attempts) == 0 || job.Attempts[len(job.Attempts)-1].Completion == nil {
		return shots.ReadabilityOutput{}, errors.New("render output has no verified completion record")
	}
	for _, output := range job.Attempts[len(job.Attempts)-1].Completion.Outputs {
		if output.SHA256 == actual && output.MIMEType == "image/png" && output.Width > 0 && output.Height > 0 {
			return shots.ReadabilityOutput{SHA256: actual, Width: output.Width, Height: output.Height, MediaType: output.MIMEType}, nil
		}
	}
	return shots.ReadabilityOutput{}, errors.New("render output is not a verified PNG bound to the succeeded job")
}

func readabilityFinalStillProfile(profile string) bool {
	return profile == OutputProfilePNG
}

func readabilityJobMatches(job jobs.Job, projectID, shotID string, revision int64) bool {
	boundShot, boundRevision, err := parseShotRevision(job.Input)
	return err == nil && job.State == jobs.Succeeded && job.Input.ProjectID == projectID && boundShot == shotID && boundRevision == revision
}

func isFinalVideoProfile(profile string) bool {
	return profile == OutputProfileMP4 || profile == OutputProfilePhysicalMotionMP4 || profile == OutputProfilePhysicalMotionMP4V2 || profile == OutputProfilePhysicalRecordingMP4
}
