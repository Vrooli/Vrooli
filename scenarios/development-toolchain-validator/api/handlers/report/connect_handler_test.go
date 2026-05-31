package report_test

import (
	"context"
	"testing"

	reportH "development-toolchain-validator/handlers/report"
	reportdom "development-toolchain-validator/internal/report"
	vr "development-toolchain-validator/internal/validation_record"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"
	reportconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report/report_v1connect"
)

type fakeService struct {
	SummaryOut  reportdom.GoldenSummary
	SummaryErr  error
	HistoryOut  reportdom.TupleHistory
	HistoryErr  error
	CoverageOut reportdom.Coverage
	CoverageErr error
	FitnessOut  reportdom.SkillFitness
	FitnessErr  error
}

func (f *fakeService) GetGoldenSummary(context.Context, string) (reportdom.GoldenSummary, error) {
	return f.SummaryOut, f.SummaryErr
}

func (f *fakeService) GetTupleHistory(context.Context, vr.TupleKind, string, string, int, string) (reportdom.TupleHistory, error) {
	return f.HistoryOut, f.HistoryErr
}

func (f *fakeService) GetCoverage(context.Context, string) (reportdom.Coverage, error) {
	return f.CoverageOut, f.CoverageErr
}

func (f *fakeService) GetSkillFitness(context.Context, string) (reportdom.SkillFitness, error) {
	return f.FitnessOut, f.FitnessErr
}

var _ reportdom.Service = (*fakeService)(nil)

func newClient(t *testing.T, svc reportdom.Service) reportconnect.ReportServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := reportconnect.NewReportServiceHandler(reportH.NewConnectHandler(reportH.Deps{
		Service: svc, Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return reportconnect.NewReportServiceClient(server.Client(), server.URL)
}

func TestGoldenSummary_Passthrough(t *testing.T) {
	client := newClient(t, &fakeService{SummaryOut: reportdom.GoldenSummary{
		GoldenSlug:    "ref",
		SkillVerdicts: []reportdom.TupleVerdict{{TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", LatestVerdict: vr.VerdictPass}},
		StaleCount:    1,
	}})
	resp, err := client.GetGoldenSummary(context.Background(), connect.NewRequest(&reportv1.GetGoldenSummaryRequest{GoldenSlug: "ref"}))
	require.NoError(t, err)
	require.Equal(t, "ref", resp.Msg.Summary.GoldenSlug)
	require.Len(t, resp.Msg.Summary.SkillVerdicts, 1)
	require.Equal(t, int32(1), resp.Msg.Summary.StaleCount)
}

func TestGoldenSummary_RejectsEmpty(t *testing.T) {
	client := newClient(t, &fakeService{SummaryErr: reportdom.ErrInvalidReport{Field: "golden_slug", Reason: "required"}})
	_, err := client.GetGoldenSummary(context.Background(), connect.NewRequest(&reportv1.GetGoldenSummaryRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCoverage_Passthrough(t *testing.T) {
	client := newClient(t, &fakeService{CoverageOut: reportdom.Coverage{
		GoldenSlug: "ref",
		Rows: []reportdom.CoverageRow{
			{TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", Verdict: vr.VerdictPass, HasManifest: true},
		},
	}})
	resp, err := client.GetCoverage(context.Background(), connect.NewRequest(&reportv1.GetCoverageRequest{GoldenSlug: "ref"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Coverage.Rows, 1)
	require.True(t, resp.Msg.Coverage.Rows[0].HasManifest)
}

func TestSkillFitness_Passthrough(t *testing.T) {
	client := newClient(t, &fakeService{FitnessOut: reportdom.SkillFitness{
		SkillID:       "plan-skill",
		TotalRuns:     3,
		PassCount:     2,
		PassRate:      0.6667,
		AvgTokens:     1200,
		LatestVerdict: vr.VerdictPass,
		Verdict:       reportdom.SkillFitnessVerdictYellow,
		ByGolden: map[string]reportdom.GoldenSkillSnapshot{
			"ref": {GoldenSlug: "ref", LatestVerdict: vr.VerdictUnexpectedMutation, RunCount: 3, Stale: true},
		},
	}})
	resp, err := client.GetSkillFitness(context.Background(), connect.NewRequest(&reportv1.GetSkillFitnessRequest{SkillId: "plan-skill"}))
	require.NoError(t, err)
	f := resp.Msg.Fitness
	require.Equal(t, "plan-skill", f.SkillId)
	require.Equal(t, int64(3), f.TotalRuns)
	require.Equal(t, reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_YELLOW, f.Verdict)
	require.Len(t, f.ByGolden, 1)
	require.True(t, f.ByGolden["ref"].Stale)
	require.Equal(t, int32(3), f.ByGolden["ref"].RunCount)
}

func TestSkillFitness_RejectsEmpty(t *testing.T) {
	client := newClient(t, &fakeService{FitnessErr: reportdom.ErrInvalidReport{Field: "skill_id", Reason: "required"}})
	_, err := client.GetSkillFitness(context.Background(), connect.NewRequest(&reportv1.GetSkillFitnessRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
