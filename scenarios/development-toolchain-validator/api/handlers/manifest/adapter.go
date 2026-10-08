package manifest

import (
	manifest "development-toolchain-validator/internal/manifest"

	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// domainToProto converts a domain Manifest into the wire shape.
func domainToProto(m manifest.Manifest) *manifestv1.Manifest {
	out := &manifestv1.Manifest{
		SkillId:               m.SkillID,
		GoldenSlug:            m.GoldenSlug,
		AllowedPaths:          append([]string(nil), m.AllowedPaths...),
		WildcardAllowed:       m.WildcardAllowed,
		ConvergenceTarget:     convergenceDomainToProto(m.ConvergenceTarget),
		TemplateVersionPinned: m.TemplateVersionPinned,
		SkillVersionPinned:    m.SkillVersionPinned,
		UpdatedAt:             timestamppb.New(m.UpdatedAt.UTC()),
	}
	out.ContentRules = make([]*manifestv1.ContentRule, 0, len(m.ContentRules))
	for _, r := range m.ContentRules {
		out.ContentRules = append(out.ContentRules, &manifestv1.ContentRule{
			PathGlob:       r.PathGlob,
			MustContain:    append([]string(nil), r.MustContain...),
			MustNotContain: append([]string(nil), r.MustNotContain...),
		})
	}
	return out
}

// protoToUpsertInput converts the wire Manifest into the service's
// UpsertInput. Timestamps and update bookkeeping are server-owned and
// dropped here intentionally.
func protoToUpsertInput(p *manifestv1.Manifest) manifest.UpsertInput {
	in := manifest.UpsertInput{
		SkillID:               p.GetSkillId(),
		GoldenSlug:            p.GetGoldenSlug(),
		AllowedPaths:          append([]string(nil), p.GetAllowedPaths()...),
		WildcardAllowed:       p.GetWildcardAllowed(),
		ConvergenceTarget:     convergenceProtoToDomain(p.GetConvergenceTarget()),
		TemplateVersionPinned: p.GetTemplateVersionPinned(),
		SkillVersionPinned:    p.GetSkillVersionPinned(),
	}
	in.ContentRules = make([]manifest.ContentRule, 0, len(p.GetContentRules()))
	for _, r := range p.GetContentRules() {
		in.ContentRules = append(in.ContentRules, manifest.ContentRule{
			PathGlob:       r.GetPathGlob(),
			MustContain:    append([]string(nil), r.GetMustContain()...),
			MustNotContain: append([]string(nil), r.GetMustNotContain()...),
		})
	}
	return in
}

func convergenceDomainToProto(c manifest.ConvergenceTarget) manifestv1.ConvergenceTarget {
	switch c {
	case manifest.ConvergenceTargetNone:
		return manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_NONE
	case manifest.ConvergenceTargetEmptyDiff:
		return manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF
	default:
		return manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_UNSPECIFIED
	}
}

func convergenceProtoToDomain(c manifestv1.ConvergenceTarget) manifest.ConvergenceTarget {
	switch c {
	case manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_NONE:
		return manifest.ConvergenceTargetNone
	case manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF:
		return manifest.ConvergenceTargetEmptyDiff
	default:
		return manifest.ConvergenceTargetUnspecified
	}
}
