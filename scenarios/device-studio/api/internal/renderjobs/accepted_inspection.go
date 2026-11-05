package renderjobs

import (
	"context"
	"encoding/json"
	"errors"

	"device-studio/internal/jobs"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// acceptedInspectionArtifacts reconstructs the original inspection closure
// from its succeeded job and attempt, then verifies the exact historical
// manifest, receipt, and clock bytes held by the media owner. Consumers must
// preserve the returned manifest identity; they may choose a local staged name
// only when the worker protocol explicitly permits it.
type acceptedInspectionArtifacts struct {
	Manifest      rendering.InspectionManifest
	ManifestBytes []byte
	Receipt       rendering.InspectionReceipt
	ReceiptBytes  []byte
	ClockBytes    []byte
	Profile       *cliv1.RendererToolProfile
	ProfileBytes  []byte
}

func (c *Coordinator) reconstructAcceptedInspection(ctx context.Context, route routePin, projectID string, source projects.SourceMedia) (acceptedInspectionArtifacts, error) {
	var result acceptedInspectionArtifacts
	if source.Kind != projects.MediaKindVideo || source.Readiness != projects.ReadinessReady || source.InspectionReceipt == nil || source.InspectionReceipt.Classification != string(rendering.InspectionAccepted) || source.InspectionJobID == "" || source.InspectionAttemptID == "" {
		return result, ErrNotEligible
	}
	job, err := c.jobs.Get(ctx, source.InspectionJobID)
	if err != nil || job.Input.SubjectKind != jobs.SubjectSourceInspection || job.Input.ProjectID != projectID || job.Input.SubjectID != source.ID || job.State != jobs.Succeeded {
		return result, errors.Join(ErrNotEligible, err)
	}
	var attempt jobs.Attempt
	found := false
	for _, candidate := range job.Attempts {
		if candidate.ID == source.InspectionAttemptID && candidate.State == jobs.AttemptSucceeded && candidate.Completion != nil {
			attempt, found = candidate, true
			break
		}
	}
	if !found || source.InspectionReceipt.JobID != job.ID || source.InspectionReceipt.AttemptID != attempt.ID {
		return result, ErrNotEligible
	}
	profile, profileBytes, err := c.inspectionProfileForJob(ctx, route, job)
	if err != nil {
		return result, err
	}
	manifest, err := inspectionManifestFromInput(job, attempt, profile)
	if err != nil {
		return result, err
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return result, err
	}
	manifestBytes = append(manifestBytes, '\n')
	receiptBytes, clockBytes, err := c.media.OpenInspectionArtifacts(ctx, projectID, source.ID, job.ID, attempt.ID, source.InspectionReceipt.SHA256, source.InspectionReceipt.ClockSHA256)
	if err != nil {
		return result, err
	}
	accepted, err := rendering.VerifyInspectionArtifacts(manifestBytes, receiptBytes, clockBytes)
	if err != nil {
		return result, err
	}
	if accepted.Classification != rendering.InspectionAccepted || accepted.JobID != job.ID || accepted.AttemptID != attempt.ID || accepted.SourceSHA256 != source.SHA256 || accepted.SourceSizeBytes != source.SizeBytes || accepted.ClockSHA256 != source.InspectionReceipt.ClockSHA256 || digestBytes(receiptBytes) != source.InspectionReceipt.SHA256 {
		return result, errors.Join(ErrNotEligible, errors.New("accepted inspection facts differ from the source media projection"))
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return result, err
	}
	result = acceptedInspectionArtifacts{Manifest: manifest, ManifestBytes: manifestBytes, Receipt: accepted, ReceiptBytes: receiptBytes, ClockBytes: clockBytes, Profile: profile, ProfileBytes: profileBytes}
	return result, nil
}
