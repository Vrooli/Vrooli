package validation_record

import (
	vr "development-toolchain-validator/internal/validation_record"

	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func domainToProto(r vr.Record) *vrv1.ValidationRecord {
	return &vrv1.ValidationRecord{
		Id:                           r.ID,
		TupleKind:                    tupleKindDomainToProto(r.TupleKind),
		SubjectId:                    r.SubjectID,
		GoldenSlug:                   r.GoldenSlug,
		StartedAt:                    timestamppb.New(r.StartedAt.UTC()),
		EndedAt:                      timestamppb.New(r.EndedAt.UTC()),
		DurationMs:                   r.DurationMS,
		TokensUsed:                   r.TokensUsed,
		CostUsdMicro:                 r.CostUSDMicro,
		Verdict:                      verdictDomainToProto(r.Verdict),
		DiffHash:                     r.DiffHash,
		DiffPathCount:                r.DiffPathCount,
		AgentManagerRunId:            r.AgentManagerRunID,
		ManifestTemplateVersionAtRun: r.ManifestTemplateVersionAtRun,
		ManifestSkillVersionAtRun:    r.ManifestSkillVersionAtRun,
		ErrorMessage:                 r.ErrorMessage,
		ToolDetail:                   r.ToolDetail,
		ToolRawOutput:                r.ToolRawOutput,
	}
}

func tupleKindDomainToProto(t vr.TupleKind) vrv1.TupleKind {
	switch t {
	case vr.TupleKindSkill:
		return vrv1.TupleKind_TUPLE_KIND_SKILL
	case vr.TupleKindTool:
		return vrv1.TupleKind_TUPLE_KIND_TOOL
	default:
		return vrv1.TupleKind_TUPLE_KIND_UNSPECIFIED
	}
}

func tupleKindProtoToDomain(t vrv1.TupleKind) vr.TupleKind {
	switch t {
	case vrv1.TupleKind_TUPLE_KIND_SKILL:
		return vr.TupleKindSkill
	case vrv1.TupleKind_TUPLE_KIND_TOOL:
		return vr.TupleKindTool
	default:
		return vr.TupleKindUnspecified
	}
}

func verdictDomainToProto(v vr.Verdict) vrv1.Verdict {
	switch v {
	case vr.VerdictPass:
		return vrv1.Verdict_VERDICT_PASS
	case vr.VerdictUnexpectedMutation:
		return vrv1.Verdict_VERDICT_UNEXPECTED_MUTATION
	case vr.VerdictRunFailure:
		return vrv1.Verdict_VERDICT_RUN_FAILURE
	case vr.VerdictToolFailure:
		return vrv1.Verdict_VERDICT_TOOL_FAILURE
	default:
		return vrv1.Verdict_VERDICT_UNSPECIFIED
	}
}
