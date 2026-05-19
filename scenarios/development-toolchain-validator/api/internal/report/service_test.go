package report_test

import (
	"context"
	"testing"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	report "development-toolchain-validator/internal/report"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"
	staleness "development-toolchain-validator/internal/staleness"
	vr "development-toolchain-validator/internal/validation_record"
	vrmocks "development-toolchain-validator/internal/validation_record/mocks"
	"development-toolchain-validator/internal/testutil/mocks"

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
