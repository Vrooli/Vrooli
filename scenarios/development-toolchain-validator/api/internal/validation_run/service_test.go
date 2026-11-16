package validation_run_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	vr "development-toolchain-validator/internal/validation_record"
	vrun "development-toolchain-validator/internal/validation_run"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "development-toolchain-validator/internal/database"
)

func newSvc(t *testing.T) (vrun.Service, vrun.Repository, *scheduletest.FakeClock, *sql.DB) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(vrun.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	repo := vrun.NewSQLiteRepository(d)
	return vrun.NewService(repo, clk, nil), repo, clk, d
}

func TestStart_QueuesRun(t *testing.T) {
	svc, repo, clk, _ := newSvc(t)
	r, err := svc.Start(context.Background(), vrun.StartInput{
		TupleKind:  vr.TupleKindSkill,
		SubjectID:  "implementation-plan-authoring",
		GoldenSlug: "reference-react-vite",
	})
	require.NoError(t, err)
	require.NotEmpty(t, r.ID)
	require.Equal(t, vrun.StatusQueued, r.Status)
	require.Equal(t, clk.Now(), r.CreatedAt)

	got, err := repo.Get(context.Background(), r.ID)
	require.NoError(t, err)
	require.Equal(t, vrun.StatusQueued, got.Status)
}

func TestStart_RejectsMissing(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	_, err := svc.Start(context.Background(), vrun.StartInput{
		TupleKind: vr.TupleKindSkill, SubjectID: "", GoldenSlug: "g",
	})
	var invalid vrun.ErrInvalidRun
	require.True(t, errors.As(err, &invalid))
}

func TestStart_RejectsUnspecifiedKind(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	_, err := svc.Start(context.Background(), vrun.StartInput{
		SubjectID: "s", GoldenSlug: "g",
	})
	var invalid vrun.ErrInvalidRun
	require.True(t, errors.As(err, &invalid))
}

func TestStart_NotifiesWorker(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(vrun.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	called := false
	svc := vrun.NewService(vrun.NewSQLiteRepository(d), clk, func() { called = true })
	_, err := svc.Start(context.Background(), vrun.StartInput{
		TupleKind: vr.TupleKindSkill, SubjectID: "s", GoldenSlug: "g",
	})
	require.NoError(t, err)
	require.True(t, called, "notify must fire on successful Start")
}

func TestGet_NotFound(t *testing.T) {
	svc, _, _, _ := newSvc(t)
	_, err := svc.Get(context.Background(), "missing")
	require.True(t, vrun.IsNotFound(err))
}

func TestListActive_ExcludesTerminal(t *testing.T) {
	svc, repo, _, _ := newSvc(t)
	ctx := context.Background()
	queued, err := svc.Start(ctx, vrun.StartInput{TupleKind: vr.TupleKindSkill, SubjectID: "a", GoldenSlug: "g"})
	require.NoError(t, err)
	terminal, err := svc.Start(ctx, vrun.StartInput{TupleKind: vr.TupleKindSkill, SubjectID: "b", GoldenSlug: "g"})
	require.NoError(t, err)
	require.NoError(t, terminal.TransitionStatus(vrun.StatusRunning))
	require.NoError(t, terminal.TransitionStatus(vrun.StatusTerminal))
	terminal.TerminalVerdict = vr.VerdictPass
	require.NoError(t, repo.UpdateStatus(ctx, terminal))

	active, err := svc.ListActive(ctx)
	require.NoError(t, err)
	require.Len(t, active, 1)
	require.Equal(t, queued.ID, active[0].ID)
}

func TestRunLifecycleOwnerAndRepositoryRejectInvalidStates(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, d := newSvc(t)
	run, err := svc.Start(ctx, vrun.StartInput{TupleKind: vr.TupleKindSkill, SubjectID: "skill", GoldenSlug: "golden"})
	require.NoError(t, err)

	if err := run.TransitionStatus(vrun.StatusTerminal); !errors.Is(err, vrun.ErrInvalidTransition) || run.Status != vrun.StatusQueued {
		t.Fatalf("queued -> terminal error=%v left status=%d", err, run.Status)
	}
	invalidCreate := run
	invalidCreate.ID = "invalid-create"
	invalidCreate.Status = vrun.StatusTerminal
	require.ErrorIs(t, repo.Create(ctx, invalidCreate), vrun.ErrInvalidTransition)

	stale := run
	require.NoError(t, run.TransitionStatus(vrun.StatusRunning))
	require.NoError(t, repo.UpdateStatus(ctx, run))
	current, err := repo.Get(ctx, run.ID)
	require.NoError(t, err)
	require.NoError(t, current.TransitionStatus(vrun.StatusEvaluating))
	require.NoError(t, repo.UpdateStatus(ctx, current))
	require.NoError(t, current.TransitionStatus(vrun.StatusTerminal))
	require.NoError(t, repo.UpdateStatus(ctx, current))
	require.NoError(t, stale.TransitionStatus(vrun.StatusRunning))
	require.NoError(t, stale.TransitionStatus(vrun.StatusEvaluating))
	require.ErrorIs(t, repo.UpdateStatus(ctx, stale), vrun.ErrInvalidTransition)

	_, err = d.ExecContext(ctx, `UPDATE validation_runs SET status = ? WHERE id = ?`, 99, run.ID)
	require.NoError(t, err)
	_, err = repo.Get(ctx, run.ID)
	require.ErrorIs(t, err, vrun.ErrInvalidTransition)
}
