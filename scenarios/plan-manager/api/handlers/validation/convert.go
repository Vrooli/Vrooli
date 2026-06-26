package validation

import (
	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/planproto"
	internalvalidation "plan-manager/internal/validation"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.shared) and the validation/plans domain vocabulary.
// The domain layer never imports proto (api-steer §7).

func resultToProto(r internalvalidation.Result) *sharedv1.ValidationResult {
	out := &sharedv1.ValidationResult{
		Id:          r.ID,
		PlanId:      r.PlanID,
		PhaseId:     r.PhaseID,
		Verdict:     verdictToProto(r.Verdict),
		Staleness:   stalenessToProto(r.Staleness),
		CommandsRun: r.CommandsRun,
		Detail:      r.Detail,
		RanAt:       r.RanAt,
	}
	for _, finding := range r.CommandFindings {
		out.CommandFindings = append(out.CommandFindings, &sharedv1.CommandValidationFinding{
			CommandText:     finding.CommandText,
			Verdict:         finding.Verdict,
			ValidationLevel: finding.Level,
			Message:         finding.Message,
			Location:        finding.Location,
			IssueCodes:      append([]string(nil), finding.IssueCodes...),
			Suggestions:     append([]string(nil), finding.Suggestions...),
			Guidance:        append([]string(nil), finding.Guidance...),
		})
	}
	return out
}

func referencesToProto(refs []planmodel.Reference) []*sharedv1.Reference {
	return planproto.ReferencesToProto(refs)
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

func refKindToProto(k planmodel.ReferenceKind) sharedv1.ReferenceKind {
	return planproto.RefKindToProto(k)
}

func refResolutionToProto(r planmodel.ReferenceResolution) sharedv1.ReferenceResolution {
	return planproto.RefResolutionToProto(r)
}

func stalenessToProto(s planmodel.StalenessTier) sharedv1.StalenessTier {
	return planproto.StalenessToProto(s)
}
