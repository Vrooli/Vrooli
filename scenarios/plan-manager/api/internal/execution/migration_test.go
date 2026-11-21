package execution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
)

func TestEnsureMigrationsInfersExecutionLifecycleAndPreservesRecords(t *testing.T) {
	database := db.NewSQLite(t)
	ctx := context.Background()

	_, err := database.ExecContext(ctx, `CREATE TABLE executions (
		id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, run_id TEXT NOT NULL DEFAULT '',
		verification_status TEXT NOT NULL DEFAULT 'absent', harness_session_id TEXT NOT NULL DEFAULT '',
		harness_kind TEXT NOT NULL DEFAULT '', current_phase_id TEXT NOT NULL DEFAULT '',
		complete INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		inputs_freshened_at TEXT NOT NULL DEFAULT '', freshen_status TEXT NOT NULL DEFAULT '',
		freshen_detail TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `CREATE TABLE execution_scope_states (
		execution_id TEXT PRIMARY KEY, document TEXT NOT NULL DEFAULT '{}', updated_at TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `CREATE TABLE handoffs (
		id TEXT PRIMARY KEY, execution_id TEXT NOT NULL, plan_id TEXT NOT NULL,
		completeness TEXT NOT NULL DEFAULT '', resume_phase_id TEXT NOT NULL DEFAULT '',
		document TEXT NOT NULL DEFAULT '{}', assembled_at TEXT NOT NULL)`)
	require.NoError(t, err)

	for _, row := range []struct {
		id, phase string
		complete  int
	}{
		{id: "active-with-phase", phase: "phase-2", complete: 0},
		{id: "ready-without-phase", phase: "", complete: 1},
		{id: "completed-with-handoff", phase: "", complete: 0},
		{id: "abandoned", phase: "phase-3", complete: 0},
	} {
		_, err = database.ExecContext(ctx, `INSERT INTO executions
			(id, plan_id, current_phase_id, complete, started_at, updated_at)
			VALUES (?, 'plan-1', ?, ?, 'started', 'updated')`, row.id, row.phase, row.complete)
		require.NoError(t, err)
	}
	_, err = database.ExecContext(ctx, `INSERT INTO execution_scope_states (execution_id, document, updated_at)
		VALUES ('abandoned', '{"lifecycle_state":"abandoned"}', 'scope-updated')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `INSERT INTO handoffs
		(id, execution_id, plan_id, completeness, document, assembled_at)
		VALUES ('handoff-1', 'completed-with-handoff', 'plan-1', 'full', '{"kept":true}', 'assembled')`)
	require.NoError(t, err)

	require.NoError(t, EnsureMigrations(ctx, database))
	require.NoError(t, EnsureMigrations(ctx, database), "migration is idempotent")

	rows, err := database.QueryContext(ctx, `SELECT id, lifecycle_state, started_at, updated_at FROM executions ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var id, state, started, updated string
		require.NoError(t, rows.Scan(&id, &state, &started, &updated))
		require.Equal(t, "started", started)
		require.Equal(t, "updated", updated)
		got[id] = state
	}
	require.NoError(t, rows.Err())
	require.Equal(t, map[string]string{
		"active-with-phase":      "active",
		"ready-without-phase":    "ready",
		"completed-with-handoff": "completed",
		"abandoned":              "abandoned",
	}, got)

	var handoffDocument string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT document FROM handoffs WHERE id='handoff-1'`).Scan(&handoffDocument))
	require.Equal(t, `{"kept":true}`, handoffDocument)
	var completeColumn int
	require.NoError(t, database.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('executions') WHERE name='complete'`).Scan(&completeColumn))
	require.Zero(t, completeColumn, "derived boolean column must be retired")

	_, err = database.ExecContext(ctx, `UPDATE executions SET lifecycle_state='unknown' WHERE id='active-with-phase'`)
	require.Error(t, err, "the persisted state constraint rejects unknown lifecycle values")
}

func TestEnsureMigrationsRetiresLegacyFlagAfterPartialMigration(t *testing.T) {
	database := db.NewSQLite(t)
	ctx := context.Background()
	_, err := database.ExecContext(ctx, `CREATE TABLE executions (
		id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, run_id TEXT NOT NULL DEFAULT '',
		verification_status TEXT NOT NULL DEFAULT 'absent', harness_session_id TEXT NOT NULL DEFAULT '',
		harness_kind TEXT NOT NULL DEFAULT '', current_phase_id TEXT NOT NULL DEFAULT '',
		lifecycle_state TEXT NOT NULL DEFAULT 'active' CHECK (lifecycle_state IN ('active','ready','completed','abandoned')),
		complete INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		inputs_freshened_at TEXT NOT NULL DEFAULT '', freshen_status TEXT NOT NULL DEFAULT '',
		freshen_detail TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, `INSERT INTO executions (id, plan_id, lifecycle_state, complete, started_at, updated_at)
		VALUES ('execution-1', 'plan-1', 'completed', 0, 'started', 'updated')`)
	require.NoError(t, err)

	require.NoError(t, EnsureMigrations(ctx, database))
	require.NoError(t, EnsureMigrations(ctx, database), "partial migration repair is idempotent")

	var state string
	require.NoError(t, database.QueryRowContext(ctx, `SELECT lifecycle_state FROM executions WHERE id='execution-1'`).Scan(&state))
	require.Equal(t, "completed", state, "the existing canonical state must not be recomputed from the legacy boolean")
	var completeColumn int
	require.NoError(t, database.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('executions') WHERE name='complete'`).Scan(&completeColumn))
	require.Zero(t, completeColumn, "partial migration repair must retire the candidate-guard's obsolete boolean column")
}
