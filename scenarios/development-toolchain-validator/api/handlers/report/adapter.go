package report

import (
	report "development-toolchain-validator/internal/report"
	vr "development-toolchain-validator/internal/validation_record"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"
	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func summaryToProto(s report.GoldenSummary) *reportv1.GoldenSummary {
	out := &reportv1.GoldenSummary{
		GoldenSlug:    s.GoldenSlug,
		StaleCount:    int32(s.StaleCount),
		SkillVerdicts: make([]*reportv1.TupleVerdict, 0, len(s.SkillVerdicts)),
		ToolVerdicts:  make([]*reportv1.TupleVerdict, 0, len(s.ToolVerdicts)),
	}
	for _, v := range s.SkillVerdicts {
		out.SkillVerdicts = append(out.SkillVerdicts, tupleVerdictToProto(v))
	}
	for _, v := range s.ToolVerdicts {
		out.ToolVerdicts = append(out.ToolVerdicts, tupleVerdictToProto(v))
	}
	return out
}

func tupleVerdictToProto(v report.TupleVerdict) *reportv1.TupleVerdict {
	return &reportv1.TupleVerdict{
		TupleKind:      tupleKindDomainToProto(v.TupleKind),
		SubjectId:      v.SubjectID,
		LatestVerdict:  verdictDomainToProto(v.LatestVerdict),
		LatestRecordId: v.LatestRecordID,
		Stale:          v.Stale,
	}
}

func historyToProto(h report.TupleHistory) *reportv1.TupleHistory {
	out := &reportv1.TupleHistory{
		TupleKind:     tupleKindDomainToProto(h.TupleKind),
		SubjectId:     h.SubjectID,
		GoldenSlug:    h.GoldenSlug,
		NextPageToken: h.NextPageToken,
		Records:       make([]*vrv1.ValidationRecord, 0, len(h.Records)),
	}
	for _, r := range h.Records {
		out.Records = append(out.Records, recordToProto(r))
	}
	return out
}

func coverageToProto(c report.Coverage) *reportv1.Coverage {
	out := &reportv1.Coverage{
		GoldenSlug: c.GoldenSlug,
		Rows:       make([]*reportv1.CoverageRow, 0, len(c.Rows)),
	}
	for _, r := range c.Rows {
		out.Rows = append(out.Rows, &reportv1.CoverageRow{
			TupleKind:   tupleKindDomainToProto(r.TupleKind),
			SubjectId:   r.SubjectID,
			Verdict:     verdictDomainToProto(r.Verdict),
			Stale:       r.Stale,
			HasManifest: r.HasManifest,
		})
	}
	return out
}

func recordToProto(r vr.Record) *vrv1.ValidationRecord {
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

func skillFitnessToProto(f report.SkillFitness) *reportv1.SkillFitness {
	out := &reportv1.SkillFitness{
		SkillId:                 f.SkillID,
		PassCount:               f.PassCount,
		UnexpectedMutationCount: f.UnexpectedMutationCount,
		RunFailureCount:         f.RunFailureCount,
		ToolFailureCount:        f.ToolFailureCount,
		TotalRuns:               f.TotalRuns,
		PassRate:                f.PassRate,
		TotalTokens:             f.TotalTokens,
		AvgTokens:               f.AvgTokens,
		TotalCostUsdMicro:       f.TotalCostUSDMicro,
		AvgCostUsdMicro:         f.AvgCostUSDMicro,
		TotalDurationMs:         f.TotalDurationMS,
		AvgDurationMs:           f.AvgDurationMS,
		UniqueDiffHashes:        int32(f.UniqueDiffHashes),
		ConvergenceRatio:        f.ConvergenceRatio,
		LatestVerdict:           verdictDomainToProto(f.LatestVerdict),
		AnyStale:                f.AnyStale,
		Verdict:                 skillFitnessVerdictToProto(f.Verdict),
		ByGolden:                make(map[string]*reportv1.GoldenSkillSnapshot, len(f.ByGolden)),
	}
	for slug, snap := range f.ByGolden {
		out.ByGolden[slug] = &reportv1.GoldenSkillSnapshot{
			GoldenSlug:    snap.GoldenSlug,
			LatestVerdict: verdictDomainToProto(snap.LatestVerdict),
			Stale:         snap.Stale,
			RunCount:      int32(snap.RunCount),
		}
	}
	return out
}

func skillFitnessVerdictToProto(v report.SkillFitnessVerdict) reportv1.SkillFitnessVerdict {
	switch v {
	case report.SkillFitnessVerdictUnknown:
		return reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_UNKNOWN
	case report.SkillFitnessVerdictGreen:
		return reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_GREEN
	case report.SkillFitnessVerdictYellow:
		return reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_YELLOW
	case report.SkillFitnessVerdictRed:
		return reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_RED
	default:
		return reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_UNSPECIFIED
	}
}
