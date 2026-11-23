package report

import (
	"testing"
	"time"

	"development-toolchain-validator/internal/report"
	vr "development-toolchain-validator/internal/validation_record"

	"github.com/stretchr/testify/require"
	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
)

func TestHistoryToProtoUsesCanonicalValidationRecordMapping(t *testing.T) {
	startedAt := time.Date(2026, time.June, 2, 10, 11, 12, 0, time.FixedZone("UTC+3", 3*60*60))
	endedAt := startedAt.Add(30 * time.Minute)
	record := vr.Record{
		ID: "record-9", TupleKind: vr.TupleKindTool, SubjectID: "unit-health", GoldenSlug: "reliable",
		StartedAt: startedAt, EndedAt: endedAt, DurationMS: 1_800_000, TokensUsed: 41, CostUSDMicro: 67,
		Verdict: vr.VerdictUnexpectedMutation, DiffHash: "sha256:abc", DiffPathCount: 2,
		AgentManagerRunID: "run-12", ManifestTemplateVersionAtRun: "template-5", ManifestSkillVersionAtRun: "skill-8",
		ErrorMessage: "unexpected change", ToolDetail: "2 of 4 phases passed", ToolRawOutput: "tool output",
	}

	got := historyToProto(report.TupleHistory{
		TupleKind: vr.TupleKindTool, SubjectID: record.SubjectID, GoldenSlug: record.GoldenSlug,
		Records: []vr.Record{record}, NextPageToken: "next",
	})
	require.Equal(t, "next", got.NextPageToken)
	require.Len(t, got.Records, 1)
	wire := got.Records[0]
	require.Equal(t, "record-9", wire.Id)
	require.Equal(t, vrv1.TupleKind_TUPLE_KIND_TOOL, wire.TupleKind)
	require.Equal(t, "unit-health", wire.SubjectId)
	require.Equal(t, "reliable", wire.GoldenSlug)
	require.True(t, wire.StartedAt.AsTime().Equal(startedAt.UTC()))
	require.True(t, wire.EndedAt.AsTime().Equal(endedAt.UTC()))
	require.Equal(t, int64(1_800_000), wire.DurationMs)
	require.Equal(t, int64(41), wire.TokensUsed)
	require.Equal(t, int64(67), wire.CostUsdMicro)
	require.Equal(t, vrv1.Verdict_VERDICT_UNEXPECTED_MUTATION, wire.Verdict)
	require.Equal(t, "sha256:abc", wire.DiffHash)
	require.Equal(t, int32(2), wire.DiffPathCount)
	require.Equal(t, "run-12", wire.AgentManagerRunId)
	require.Equal(t, "template-5", wire.ManifestTemplateVersionAtRun)
	require.Equal(t, "skill-8", wire.ManifestSkillVersionAtRun)
	require.Equal(t, "unexpected change", wire.ErrorMessage)
	require.Equal(t, "2 of 4 phases passed", wire.ToolDetail)
	require.Equal(t, "tool output", wire.ToolRawOutput)
}
