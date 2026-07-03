package validation_record_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"development-toolchain-validator/internal/testutil/mocks"
	vr "development-toolchain-validator/internal/validation_record"
	vrmocks "development-toolchain-validator/internal/validation_record/mocks"

	"github.com/stretchr/testify/require"
)

func newSvc(t *testing.T) (vr.Service, *vrmocks.FakeRepository, *mocks.FakeClock) {
	t.Helper()
	repo := vrmocks.NewFakeRepository()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	return vr.NewService(repo, clk), repo, clk
}

func TestAppend_StampsAndAssignsID(t *testing.T) {
	svc, _, clk := newSvc(t)
	r, err := svc.Append(context.Background(), vr.AppendInput{
		TupleKind:  vr.TupleKindSkill,
		SubjectID:  "implementation-plan-authoring",
		GoldenSlug: "reference-react-vite",
		Verdict:    vr.VerdictPass,
	})
	require.NoError(t, err)
	require.NotEmpty(t, r.ID)
	require.Equal(t, clk.Now(), r.EndedAt)
	require.Equal(t, clk.Now(), r.StartedAt)
}

func TestAppend_ComputesDuration(t *testing.T) {
	svc, _, _ := newSvc(t)
	start := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	end := start.Add(750 * time.Millisecond)
	r, err := svc.Append(context.Background(), vr.AppendInput{
		TupleKind: vr.TupleKindSkill,
		SubjectID: "s", GoldenSlug: "g",
		Verdict:   vr.VerdictPass,
		StartedAt: start, EndedAt: end,
	})
	require.NoError(t, err)
	require.Equal(t, int64(750), r.DurationMS)
}

func TestAppend_RejectsMissingFields(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Append(context.Background(), vr.AppendInput{
		TupleKind: vr.TupleKindSkill, GoldenSlug: "g", Verdict: vr.VerdictPass,
	})
	var invalid vr.ErrInvalidRecord
	require.True(t, errors.As(err, &invalid))
	require.Equal(t, "subject_id", invalid.Field)
}

func TestAppend_RejectsUnspecifiedKindOrVerdict(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Append(context.Background(), vr.AppendInput{
		SubjectID: "s", GoldenSlug: "g", Verdict: vr.VerdictPass,
	})
	var invalid vr.ErrInvalidRecord
	require.True(t, errors.As(err, &invalid))

	_, err = svc.Append(context.Background(), vr.AppendInput{
		SubjectID: "s", GoldenSlug: "g", TupleKind: vr.TupleKindSkill,
	})
	require.True(t, errors.As(err, &invalid))
}

func TestGet_EmptyIDRejected(t *testing.T) {
	svc, _, _ := newSvc(t)
	_, err := svc.Get(context.Background(), "")
	var invalid vr.ErrInvalidRecord
	require.True(t, errors.As(err, &invalid))
}
