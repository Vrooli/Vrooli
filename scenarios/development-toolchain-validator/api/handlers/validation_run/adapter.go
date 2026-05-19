package validation_run

import (
	vrun "development-toolchain-validator/internal/validation_run"
	vr "development-toolchain-validator/internal/validation_record"

	vrunv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run"
	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func domainToProto(r vrun.Run) *vrunv1.ValidationRun {
	out := &vrunv1.ValidationRun{
		Id:                r.ID,
		TupleKind:         tupleKindDomainToProto(r.TupleKind),
		SubjectId:         r.SubjectID,
		GoldenSlug:        r.GoldenSlug,
		Status:            statusDomainToProto(r.Status),
		TerminalVerdict:   verdictDomainToProto(r.TerminalVerdict),
		AgentManagerRunId: r.AgentManagerRunID,
		CreatedAt:         timestamppb.New(r.CreatedAt.UTC()),
		ErrorMessage:      r.ErrorMessage,
	}
	if !r.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(r.StartedAt.UTC())
	}
	if !r.EndedAt.IsZero() {
		out.EndedAt = timestamppb.New(r.EndedAt.UTC())
	}
	return out
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

func statusDomainToProto(s vrun.Status) vrunv1.Status {
	switch s {
	case vrun.StatusQueued:
		return vrunv1.Status_STATUS_QUEUED
	case vrun.StatusRunning:
		return vrunv1.Status_STATUS_RUNNING
	case vrun.StatusEvaluating:
		return vrunv1.Status_STATUS_EVALUATING
	case vrun.StatusTerminal:
		return vrunv1.Status_STATUS_TERMINAL
	default:
		return vrunv1.Status_STATUS_UNSPECIFIED
	}
}
