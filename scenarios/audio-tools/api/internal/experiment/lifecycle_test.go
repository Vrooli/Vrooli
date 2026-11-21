package experiment_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"audio-tools/internal/experiment"
)

func TestExperimentTransitionMatrix(t *testing.T) {
	states := []experiment.Status{
		experiment.StatusQueued,
		experiment.StatusRunning,
		experiment.StatusSucceeded,
		experiment.StatusFailed,
		experiment.StatusCanceled,
	}
	allowed := map[[2]experiment.Status]bool{
		{experiment.StatusQueued, experiment.StatusRunning}:    true,
		{experiment.StatusQueued, experiment.StatusFailed}:     true,
		{experiment.StatusQueued, experiment.StatusCanceled}:   true,
		{experiment.StatusRunning, experiment.StatusSucceeded}: true,
		{experiment.StatusRunning, experiment.StatusFailed}:    true,
		{experiment.StatusRunning, experiment.StatusCanceled}:  true,
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				exp := experiment.Experiment{Status: from}
				err := exp.Transition(to)
				if from == to {
					require.NoError(t, err)
					require.Equal(t, from, exp.Status)
					return
				}
				if allowed[[2]experiment.Status{from, to}] {
					require.NoError(t, err)
					require.Equal(t, to, exp.Status)
					return
				}
				require.Error(t, err)
				require.Equal(t, from, exp.Status, "rejected transition must not mutate state")
			})
		}
	}
}

func TestExperimentTransitionRejectsUnknownAndNil(t *testing.T) {
	exp := experiment.Experiment{Status: "unknown"}
	require.Error(t, exp.Transition(experiment.StatusFailed))
	require.Equal(t, experiment.Status("unknown"), exp.Status)

	exp.Status = experiment.StatusQueued
	require.Error(t, exp.Transition("unknown"))
	require.Equal(t, experiment.StatusQueued, exp.Status)

	var nilExperiment *experiment.Experiment
	require.Error(t, nilExperiment.Transition(experiment.StatusQueued))
}

func TestRepositoryRejectsUnknownStatuses(t *testing.T) {
	ctx := context.Background()
	db := newSchemaDB(t)
	clk := scheduletest.New(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	repo := experiment.NewSQLiteRepository(db, clk)

	_, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "invalid", Status: "unrecognized"})
	require.Error(t, err)

	valid, err := repo.CreateExperiment(ctx, experiment.Experiment{Name: "valid"})
	require.NoError(t, err)
	valid.Status = "unrecognized"
	require.Error(t, repo.UpdateExperiment(ctx, valid))

	_, err = db.ExecContext(ctx, `INSERT INTO experiments (id, name, status, recipe_json, created_at, started_at, finished_at, error, result_ref, machine_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"unknown-status", "corrupt", "unrecognized", "{}", "2026-06-30T12:00:00Z", "", "", "", "", "{}")
	require.NoError(t, err)
	_, err = repo.GetExperiment(ctx, "unknown-status")
	require.ErrorContains(t, err, "unknown persisted status")
}
