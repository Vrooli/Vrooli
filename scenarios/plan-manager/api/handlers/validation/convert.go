package validation

import (
	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.shared) and the validation/plans domain vocabulary.
// The domain layer never imports proto (api-steer §7).

func resultToProto(r internalvalidation.Result) *sharedv1.ValidationResult {
	return &sharedv1.ValidationResult{
		Id:          r.ID,
		PlanId:      r.PlanID,
		PhaseId:     r.PhaseID,
		Verdict:     verdictToProto(r.Verdict),
		Staleness:   stalenessToProto(r.Staleness),
		CommandsRun: r.CommandsRun,
		Detail:      r.Detail,
		RanAt:       r.RanAt,
	}
}

func referencesToProto(refs []internalplans.Reference) []*sharedv1.Reference {
	out := make([]*sharedv1.Reference, 0, len(refs))
	for _, r := range refs {
		out = append(out, &sharedv1.Reference{
			Id:           r.ID,
			Kind:         refKindToProto(r.Kind),
			Target:       r.Target,
			Future:       r.Future,
			Resolution:   refResolutionToProto(r.Resolution),
			Staleness:    stalenessToProto(r.Staleness),
			ChangeFactor: r.ChangeFactor,
			Note:         r.Note,
		})
	}
	return out
}

func verdictToProto(v internalvalidation.Verdict) sharedv1.ValidationVerdict {
	switch v {
	case internalvalidation.VerdictPass:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS
	case internalvalidation.VerdictFail:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL
	case internalvalidation.VerdictUnknown:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN
	default:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED
	}
}

func refKindToProto(k internalplans.ReferenceKind) sharedv1.ReferenceKind {
	switch k {
	case internalplans.ReferenceCode:
		return sharedv1.ReferenceKind_REFERENCE_KIND_CODE
	case internalplans.ReferenceReq:
		return sharedv1.ReferenceKind_REFERENCE_KIND_REQ
	case internalplans.ReferenceDoc:
		return sharedv1.ReferenceKind_REFERENCE_KIND_DOC
	default:
		return sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED
	}
}

func refResolutionToProto(r internalplans.ReferenceResolution) sharedv1.ReferenceResolution {
	switch r {
	case internalplans.ResolutionResolved:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED
	case internalplans.ResolutionUnresolved:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED
	case internalplans.ResolutionFuture:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE
	case internalplans.ResolutionMissing:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING
	default:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED
	}
}

func stalenessToProto(s internalplans.StalenessTier) sharedv1.StalenessTier {
	switch s {
	case internalplans.StalenessFresh:
		return sharedv1.StalenessTier_STALENESS_TIER_FRESH
	case internalplans.StalenessLightlyStale:
		return sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE
	case internalplans.StalenessDefinitelyStale:
		return sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE
	default:
		return sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED
	}
}
