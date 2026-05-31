package report_test

import (
	"context"
	"testing"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	report "development-toolchain-validator/internal/report"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"
	staleness "development-toolchain-validator/internal/staleness"
	"development-toolchain-validator/internal/testutil/mocks"
	vr "development-toolchain-validator/internal/validation_record"
	vrmocks "development-toolchain-validator/internal/validation_record/mocks"

	"github.com/stretchr/testify/require"
)

type fakeSkillSrc struct {
	out []skillcatalog.Skill
}

func (f *fakeSkillSrc) List(context.Context) ([]skillcatalog.Skill, error) {
	return f.out, nil
}

type fakeManifestSrc struct {
	out []manifest.Manifest
}

func (f *fakeManifestSrc) List(context.Context) ([]manifest.Manifest, error) {
	return f.out, nil
}

type fakeStaleSrc struct {
	out []staleness.Entry
}

func (f *fakeStaleSrc) ListStale(context.Context) ([]staleness.Entry, error) {
	return f.out, nil
}

func newSvc(t *testing.T, skills []skillcatalog.Skill, manifests []manifest.Manifest, stale []staleness.Entry, records []vr.AppendInput) report.Service {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	recordsRepo := vrmocks.NewFakeRepository()
	recordsSvc := vr.NewService(recordsRepo, clk)
	for _, in := range records {
		_, err := recordsSvc.Append(context.Background(), in)
		require.NoError(t, err)
	}
	return report.NewService(
		&fakeSkillSrc{out: skills},
		&fakeManifestSrc{out: manifests},
		recordsSvc,
		&fakeStaleSrc{out: stale},
	)
}

func TestGetGoldenSummary_LatestPerTupleWins(t *testing.T) {
	t1 := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)
	svc := newSvc(t, nil, nil, nil, []vr.AppendInput{
		{TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", GoldenSlug: "ref", Verdict: vr.VerdictUnexpectedMutation, StartedAt: t1, EndedAt: t1},
		{TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", GoldenSlug: "ref", Verdict: vr.VerdictPass, StartedAt: t2, EndedAt: t2},
	})
	got, err := svc.GetGoldenSummary(context.Background(), "ref")
	require.NoError(t, err)
	require.Len(t, got.SkillVerdicts, 1)
	require.Equal(t, vr.VerdictPass, got.SkillVerdicts[0].LatestVerdict, "the most-recent verdict must win")
}

func TestGetGoldenSummary_FiltersByGolden(t *testing.T) {
	now := time.Now()
	svc := newSvc(t, nil, nil, nil, []vr.AppendInput{
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictPass, StartedAt: now, EndedAt: now},
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "b", Verdict: vr.VerdictPass, StartedAt: now, EndedAt: now},
	})
	got, err := svc.GetGoldenSummary(context.Background(), "a")
	require.NoError(t, err)
	require.Len(t, got.SkillVerdicts, 1)
}

func TestGetGoldenSummary_CountsStale(t *testing.T) {
	svc := newSvc(t,
		nil, nil,
		[]staleness.Entry{
			{SkillID: "plan-skill", GoldenSlug: "ref", Kind: staleness.StaleKindTemplateDrift},
			{SkillID: "other", GoldenSlug: "ref", Kind: staleness.StaleKindSkillDrift},
			{SkillID: "x", GoldenSlug: "other-golden", Kind: staleness.StaleKindBoth},
		},
		nil,
	)
	got, err := svc.GetGoldenSummary(context.Background(), "ref")
	require.NoError(t, err)
	require.Equal(t, 2, got.StaleCount, "only entries for the requested golden count")
}

func TestGetTupleHistory_Filters(t *testing.T) {
	now := time.Now()
	svc := newSvc(t, nil, nil, nil, []vr.AppendInput{
		{TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", GoldenSlug: "ref", Verdict: vr.VerdictPass, StartedAt: now, EndedAt: now},
		{TupleKind: vr.TupleKindSkill, SubjectID: "other", GoldenSlug: "ref", Verdict: vr.VerdictPass, StartedAt: now, EndedAt: now},
	})
	got, err := svc.GetTupleHistory(context.Background(), vr.TupleKindSkill, "plan-skill", "ref", 0, "")
	require.NoError(t, err)
	require.Len(t, got.Records, 1)
	require.Equal(t, "plan-skill", got.Records[0].SubjectID)
}

func TestGetCoverage_RowPerCatalogSkill(t *testing.T) {
	svc := newSvc(t,
		[]skillcatalog.Skill{
			{ID: "plan-skill", Version: "v1"},
			{ID: "test", Version: "v1"},
		},
		[]manifest.Manifest{{SkillID: "plan-skill", GoldenSlug: "ref", WildcardAllowed: true}},
		nil,
		nil,
	)
	got, err := svc.GetCoverage(context.Background(), "ref")
	require.NoError(t, err)
	require.Len(t, got.Rows, 2)
	// Rows sorted by subject id
	require.Equal(t, "plan-skill", got.Rows[0].SubjectID)
	require.True(t, got.Rows[0].HasManifest, "plan-skill has a manifest")
	require.False(t, got.Rows[1].HasManifest, "test has no manifest pinned")
}

func TestGetGoldenSummary_RejectsEmpty(t *testing.T) {
	svc := newSvc(t, nil, nil, nil, nil)
	_, err := svc.GetGoldenSummary(context.Background(), "")
	require.Error(t, err)
	var invalid report.ErrInvalidReport
	require.ErrorAs(t, err, &invalid)
}

func TestGetSkillFitness_NoData_Unknown(t *testing.T) {
	svc := newSvc(t, nil, nil, nil, nil)
	got, err := svc.GetSkillFitness(context.Background(), "plan-skill")
	require.NoError(t, err)
	require.Equal(t, report.SkillFitnessVerdictUnknown, got.Verdict)
	require.Zero(t, got.TotalRuns)
	require.Empty(t, got.ByGolden)
}

func TestGetSkillFitness_AllPass_Green(t *testing.T) {
	start := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	svc := newSvc(t, nil, nil, nil, []vr.AppendInput{
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictPass, StartedAt: start, EndedAt: start.Add(50 * time.Millisecond), DiffHash: "h1", TokensUsed: 100, CostUSDMicro: 200},
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "b", Verdict: vr.VerdictPass, StartedAt: start, EndedAt: start.Add(150 * time.Millisecond), DiffHash: "h1", TokensUsed: 300, CostUSDMicro: 400},
	})
	got, err := svc.GetSkillFitness(context.Background(), "s")
	require.NoError(t, err)
	require.Equal(t, report.SkillFitnessVerdictGreen, got.Verdict)
	require.Equal(t, int64(2), got.TotalRuns)
	require.InDelta(t, 1.0, got.PassRate, 1e-9)
	require.InDelta(t, 200.0, got.AvgTokens, 1e-9)
	require.InDelta(t, 300.0, got.AvgCostUSDMicro, 1e-9)
	require.InDelta(t, 100.0, got.AvgDurationMS, 1e-9)
	require.Equal(t, 1, got.UniqueDiffHashes, "identical diff hash across runs => converged")
	require.InDelta(t, 1.0, got.ConvergenceRatio, 1e-9)
	require.Len(t, got.ByGolden, 2)
}

func TestGetSkillFitness_LatestRunFailure_Red(t *testing.T) {
	t1 := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)
	svc := newSvc(t, nil, nil, nil, []vr.AppendInput{
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictPass, StartedAt: t1, EndedAt: t1},
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictRunFailure, StartedAt: t2, EndedAt: t2},
	})
	got, err := svc.GetSkillFitness(context.Background(), "s")
	require.NoError(t, err)
	require.Equal(t, report.SkillFitnessVerdictRed, got.Verdict, "latest run-failure dominates")
	require.Equal(t, vr.VerdictRunFailure, got.LatestVerdict)
}

func TestGetSkillFitness_LatestMutation_Yellow(t *testing.T) {
	t1 := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)
	svc := newSvc(t, nil, nil, nil, []vr.AppendInput{
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictPass, StartedAt: t1, EndedAt: t1, DiffHash: "h1"},
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictUnexpectedMutation, StartedAt: t2, EndedAt: t2, DiffHash: "h2"},
	})
	got, err := svc.GetSkillFitness(context.Background(), "s")
	require.NoError(t, err)
	require.Equal(t, report.SkillFitnessVerdictYellow, got.Verdict)
	require.Equal(t, 2, got.UniqueDiffHashes)
	require.InDelta(t, 0.5, got.ConvergenceRatio, 1e-9, "two distinct diffs => 0.5")
}

func TestGetSkillFitness_MixedGoldens_RunFailureWins(t *testing.T) {
	now := time.Now()
	svc := newSvc(t, nil, nil, nil, []vr.AppendInput{
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictPass, StartedAt: now, EndedAt: now},
		{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "b", Verdict: vr.VerdictRunFailure, StartedAt: now, EndedAt: now},
	})
	got, err := svc.GetSkillFitness(context.Background(), "s")
	require.NoError(t, err)
	require.Equal(t, report.SkillFitnessVerdictRed, got.Verdict, "a run failure on any golden gates the skill")
}

func TestGetSkillFitness_SurfacesStale(t *testing.T) {
	now := time.Now()
	svc := newSvc(t, nil, nil,
		[]staleness.Entry{{SkillID: "s", GoldenSlug: "a", Kind: staleness.StaleKindTemplateDrift}},
		[]vr.AppendInput{
			{TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "a", Verdict: vr.VerdictPass, StartedAt: now, EndedAt: now},
		})
	got, err := svc.GetSkillFitness(context.Background(), "s")
	require.NoError(t, err)
	require.True(t, got.AnyStale)
	require.True(t, got.ByGolden["a"].Stale)
}

func TestGetSkillFitness_RejectsEmpty(t *testing.T) {
	svc := newSvc(t, nil, nil, nil, nil)
	_, err := svc.GetSkillFitness(context.Background(), "")
	require.Error(t, err)
	var invalid report.ErrInvalidReport
	require.ErrorAs(t, err, &invalid)
}
