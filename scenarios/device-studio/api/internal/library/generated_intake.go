package library

import (
	"bytes"
	"context"
	"fmt"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"math"
)

// PersistGenerated stores local construction output through the same immutable
// intake/blob/revision owner as a user import. It always remains unreviewed.
func (s *IntakeService) PersistGenerated(ctx context.Context, claims IntakeClaims, jobID string, lineage CreationLineage, data []byte) (IntakeCandidate, error) {
	if s == nil || !safeID.MatchString(jobID) || lineage.JobID != jobID || len(data) == 0 || !sha256Pattern.MatchString(lineage.RecipeSHA256) || !sha256Pattern.MatchString(lineage.InferenceOutcomeSHA256) || lineage.ProviderCalls != 1 || len(lineage.ReferenceSHA256) == 0 || len(lineage.ReferenceSHA256) > 4 || len(lineage.Provider) > 256 || len(lineage.Model) > 256 || lineage.InputTokens < 0 || lineage.OutputTokens < 0 || lineage.CostMicros < 0 || !validUsageProvenance(lineage.InputTokensProvenance) || !validUsageProvenance(lineage.OutputTokensProvenance) || !validUsageProvenance(lineage.CostProvenance) || !validCreationAppliedSettings(lineage.AppliedSettings) {
		return IntakeCandidate{}, fmt.Errorf("%w: generated geometry lineage is incomplete", ErrInvalidIntake)
	}
	for _, digest := range lineage.ReferenceSHA256 {
		if !sha256Pattern.MatchString(digest) {
			return IntakeCandidate{}, fmt.Errorf("%w: generated reference digest is invalid", ErrInvalidIntake)
		}
	}
	staged, err := s.BeginImport(ctx, claims, "generated-device.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		return IntakeCandidate{}, err
	}
	owned, ok := staged.(*stagedIntake)
	if !ok {
		_ = staged.Abort(ctx)
		return IntakeCandidate{}, fmt.Errorf("%w: generated intake owner is unavailable", ErrInvalidIntake)
	}
	lineage.ReferenceSHA256 = append([]string(nil), lineage.ReferenceSHA256...)
	owned.record.Candidate.ImportMethod = "local_ai_geometry"
	owned.record.Candidate.Creation = &lineage
	digest, err := intakeCandidateDigest(owned.record.Candidate)
	if err != nil {
		_ = owned.Abort(ctx)
		return IntakeCandidate{}, err
	}
	owned.record.Candidate.RevisionSHA256 = digest
	return owned.Commit(ctx)
}

func validUsageProvenance(value inferencev1.UsageProvenance) bool {
	return value >= inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN && value <= inferencev1.UsageProvenance_USAGE_PROVENANCE_MIXED
}

func validCreationAppliedSettings(value *sharedv1.AppliedSettings) bool {
	return value != nil && value.GetTemperatureSupport() >= sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED && value.GetTemperatureSupport() <= sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN && value.GetMaxOutputTokensSource() == sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST && value.GetMaxOutputTokensEffective() == 4096 && (value.TemperatureSent == nil || !math.IsNaN(value.GetTemperatureSent()) && !math.IsInf(value.GetTemperatureSent(), 0))
}
