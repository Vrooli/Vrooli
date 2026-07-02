package experiment_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/database"
	"audio-tools/internal/experiment"
	"audio-tools/internal/testutil/db"
	"audio-tools/internal/testutil/mocks"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(database.SystemSchema),
		apidb.SchemaProviderFunc(experiment.Schema),
	))
	return d
}

func newRepo(t *testing.T) (experiment.Repository, *mocks.FakeClock) {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	return experiment.NewSQLiteRepository(newSchemaDB(t), clk), clk
}

func TestRepository_CreateGetListAndUpdateExperiment(t *testing.T) {
	ctx := context.Background()
	repo, clk := newRepo(t)

	saved, err := repo.CreateExperiment(ctx, experiment.Experiment{
		Name:       "default trio",
		RecipeJSON: []byte(`{"seed":42}`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID)
	require.Equal(t, experiment.StatusQueued, saved.Status)
	require.Equal(t, []byte(`{"seed":42}`), saved.RecipeJSON)
	require.False(t, saved.CreatedAt.IsZero())

	started := clk.Now().UTC()
	finished := started.Add(time.Minute)
	saved.Status = experiment.StatusSucceeded
	saved.StartedAt = &started
	saved.FinishedAt = &finished
	saved.ResultRef = "reports/2026-06/report.json"
	saved.MachineJSON = []byte(`{"host":"test"}`)
	require.NoError(t, repo.UpdateExperiment(ctx, saved))

	got, err := repo.GetExperiment(ctx, saved.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusSucceeded, got.Status)
	require.Equal(t, "reports/2026-06/report.json", got.ResultRef)
	require.Equal(t, []byte(`{"host":"test"}`), got.MachineJSON)
	require.NotNil(t, got.StartedAt)
	require.NotNil(t, got.FinishedAt)

	list, err := repo.ListExperiments(ctx, experiment.ListFilter{Status: experiment.StatusSucceeded})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, saved.ID, list[0].ID)
}

func TestRepository_ListNonTerminal(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	queued, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "queued"})
	require.NoError(t, err)
	running, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "running", Status: experiment.StatusRunning})
	require.NoError(t, err)
	_, err = repo.CreateExperiment(ctx, experiment.Experiment{Name: "done", Status: experiment.StatusSucceeded})
	require.NoError(t, err)

	got, err := repo.ListNonTerminal(ctx)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []string{queued.ID, running.ID}, []string{got[0].ID, got[1].ID})
}

func TestRepository_ListPagination(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	for i := 0; i < 5; i++ {
		_, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "exp"})
		require.NoError(t, err)
	}

	limited, err := repo.ListExperiments(ctx, experiment.ListFilter{Limit: 2})
	require.NoError(t, err)
	require.Len(t, limited, 2)

	paged, err := repo.ListExperiments(ctx, experiment.ListFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	require.Len(t, paged, 2)

	// Offset without a limit must not throw a SQLite "OFFSET without LIMIT"
	// syntax error; it should skip rows and return the remainder.
	offsetOnly, err := repo.ListExperiments(ctx, experiment.ListFilter{Offset: 3})
	require.NoError(t, err)
	require.Len(t, offsetOnly, 2)
}

func TestRepository_RunsRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	exp, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "with runs"})
	require.NoError(t, err)

	run, err := repo.CreateRun(ctx, experiment.Run{
		ExperimentID:  exp.ID,
		Strategy:      "overlap_agree",
		ConditionJSON: []byte(`{"snr_db":12}`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, run.ID)

	runs, err := repo.ListRuns(ctx, exp.ID)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, "overlap_agree", runs[0].Strategy)
	require.Equal(t, []byte(`{"snr_db":12}`), runs[0].ConditionJSON)
}

func TestRepository_CompleteSucceededPersistsRunsAndTerminalStatusAtomically(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	exp, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "complete"})
	require.NoError(t, err)
	exp.Status = experiment.StatusSucceeded

	err = repo.CompleteSucceeded(ctx, exp, []experiment.Run{
		{Strategy: "batch", ConditionJSON: []byte(`{"strategy":"batch"}`)},
		{Strategy: "vad_segment", ConditionJSON: []byte(`{"strategy":"vad_segment"}`)},
	})
	require.NoError(t, err)

	got, err := repo.GetExperiment(ctx, exp.ID)
	require.NoError(t, err)
	require.Equal(t, experiment.StatusSucceeded, got.Status)
	runs, err := repo.ListRuns(ctx, exp.ID)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	conditionsByStrategy := map[string][]byte{}
	for _, run := range runs {
		conditionsByStrategy[run.Strategy] = run.ConditionJSON
	}
	require.Equal(t, []byte(`{"strategy":"batch"}`), conditionsByStrategy["batch"])
	require.Equal(t, []byte(`{"strategy":"vad_segment"}`), conditionsByStrategy["vad_segment"])
}

func TestRepository_ForeignKeysCascadeRunRows(t *testing.T) {
	ctx := context.Background()
	d := newSchemaDB(t)
	repo := experiment.NewSQLiteRepository(d, mocks.NewFakeClock(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)))
	exp, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "cascade"})
	require.NoError(t, err)
	_, err = repo.CreateRun(ctx, experiment.Run{
		ExperimentID:  exp.ID,
		Strategy:      "batch",
		ConditionJSON: []byte(`{"strategy":"batch"}`),
	})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteExperiment(ctx, exp.ID))
	runs, err := repo.ListRuns(ctx, exp.ID)
	require.NoError(t, err)
	require.Empty(t, runs)
}

func TestRepository_NotFound(t *testing.T) {
	repo, _ := newRepo(t)

	_, err := repo.GetExperiment(context.Background(), "missing")
	require.ErrorAs(t, err, &experiment.ErrNotFound{})

	err = repo.UpdateExperiment(context.Background(), experiment.Experiment{ID: "missing"})
	require.ErrorAs(t, err, &experiment.ErrNotFound{})
}
