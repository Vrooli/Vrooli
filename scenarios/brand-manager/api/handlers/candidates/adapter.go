// Package candidates is the Connect boundary for logo candidates.
package candidates

import (
	"brand-manager/internal/candidates"

	candsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/candidates"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func candidateToProto(c candidates.Candidate) *candsv1.LogoCandidate {
	return &candsv1.LogoCandidate{
		Id:        c.ID,
		BrandId:   c.BrandID,
		AssetId:   c.AssetID,
		MediaType: c.MediaType,
		Concept:   c.Concept,
		Prompt:    c.Prompt,
		Role:      c.Role,
		Model:     c.Model,
		Seed:      c.Seed,
		Origin:    originToProto(c.Origin),
		ParentId:  c.ParentID,
		Status:    statusToProto(c.Status),
		Note:      c.Note,
		CreatedAt: timestamppb.New(c.CreatedAt.UTC()),
	}
}

func originToProto(o candidates.Origin) candsv1.CandidateOrigin {
	switch o {
	case candidates.OriginGenerated:
		return candsv1.CandidateOrigin_CANDIDATE_ORIGIN_GENERATED
	case candidates.OriginImported:
		return candsv1.CandidateOrigin_CANDIDATE_ORIGIN_IMPORTED
	case candidates.OriginEdited:
		return candsv1.CandidateOrigin_CANDIDATE_ORIGIN_EDITED
	case candidates.OriginObjectRemoved:
		return candsv1.CandidateOrigin_CANDIDATE_ORIGIN_OBJECT_REMOVED
	case candidates.OriginBackgroundRemoved:
		return candsv1.CandidateOrigin_CANDIDATE_ORIGIN_BACKGROUND_REMOVED
	case candidates.OriginVectorized:
		return candsv1.CandidateOrigin_CANDIDATE_ORIGIN_VECTORIZED
	default:
		return candsv1.CandidateOrigin_CANDIDATE_ORIGIN_UNSPECIFIED
	}
}

func statusToProto(s candidates.Status) candsv1.CandidateStatus {
	switch s {
	case candidates.StatusProposed:
		return candsv1.CandidateStatus_CANDIDATE_STATUS_PROPOSED
	case candidates.StatusPicked:
		return candsv1.CandidateStatus_CANDIDATE_STATUS_PICKED
	case candidates.StatusRejected:
		return candsv1.CandidateStatus_CANDIDATE_STATUS_REJECTED
	case candidates.StatusSuperseded:
		return candsv1.CandidateStatus_CANDIDATE_STATUS_SUPERSEDED
	default:
		return candsv1.CandidateStatus_CANDIDATE_STATUS_UNSPECIFIED
	}
}

func statusFromProto(s candsv1.CandidateStatus) candidates.Status {
	switch s {
	case candsv1.CandidateStatus_CANDIDATE_STATUS_PROPOSED:
		return candidates.StatusProposed
	case candsv1.CandidateStatus_CANDIDATE_STATUS_PICKED:
		return candidates.StatusPicked
	case candsv1.CandidateStatus_CANDIDATE_STATUS_REJECTED:
		return candidates.StatusRejected
	case candsv1.CandidateStatus_CANDIDATE_STATUS_SUPERSEDED:
		return candidates.StatusSuperseded
	default:
		return ""
	}
}

func vectorizeFromProto(v *candsv1.VectorizeOptions) *candidates.VectorizeOptions {
	if v == nil {
		return nil
	}
	return &candidates.VectorizeOptions{
		Colors:                     int(v.GetColors()),
		KeepColors:                 v.GetKeepColors(),
		DropBackgroundLayers:       v.GetDropBackgroundLayers(),
		ClipToLargestRoundedRegion: v.GetClipToLargestRoundedRegion(),
		InsetPx:                    v.GetInsetPx(),
		TolerancePx:                v.GetTolerancePx(),
		Smoothing:                  v.GetSmoothing(),
		MinAreaPx:                  v.GetMinAreaPx(),
	}
}
