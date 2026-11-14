package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"notification-hub/internal/conversations"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"
)

func TestReportMissingAskTableIsClear(t *testing.T) {
	db := databasetest.NewSQLite(t)
	var stdout, stderr bytes.Buffer
	err := report(context.Background(), db, &stdout, &stderr)
	require.NoError(t, err)
	require.Equal(t, "Ask identity preflight: clear (duplicate_groups=0)\n", stdout.String())
	require.Empty(t, stderr.String())
}

func TestReportDuplicateSummaryOmitsIdentityAndAskContent(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	legacySchema := strings.ReplaceAll(conversations.Schema(), "CREATE UNIQUE INDEX IF NOT EXISTS idx_asks_requester_identity\n  ON asks(source, requester_ref) WHERE requester_ref <> '';\n", "")
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(func() string { return legacySchema })))
	_, err := db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, requester_ref) VALUES ('ask-a', 'notice-a', 'private question', '["yes"]', '2026-10-18T00:00:00Z', 'pending', '2026-10-17T00:00:00Z', '2026-10-17T00:00:00Z', 'private-source', 'private-reference')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, requester_ref) VALUES ('ask-b', 'notice-b', 'private question', '["yes"]', '2026-10-18T00:00:00Z', 'answered', '2026-10-17T00:00:00Z', '2026-10-17T00:00:00Z', 'private-source', 'private-reference')`)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	err = report(ctx, db, &stdout, &stderr)
	require.Error(t, err)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "duplicate_groups=1")
	require.NotContains(t, stderr.String(), "private-reference")
	require.NotContains(t, stderr.String(), "private question")
	require.NotContains(t, stderr.String(), "private-source")
}

func TestReportQueryFailureIsUnavailableNotClear(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	_, err := db.ExecContext(ctx, `CREATE TABLE asks (id TEXT PRIMARY KEY)`)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	err = report(ctx, db, &stdout, &stderr)
	require.Error(t, err)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "preflight unavailable")
	require.NotContains(t, stderr.String(), "clear")
}

func envOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// The preflight must look at the database the live Hub serves from. The API
// gets the lifecycle's environment; this program is also run with none of it.
func TestServingDatabasePath_IsTheLiveHubsDatabaseWhateverTheEnvironment(t *testing.T) {
	scenarioDir := filepath.Join(t.TempDir(), "scenarios", "notification-hub")
	liveHub, err := storage.SQLitePath(storage.SQLiteConfig{Scenario: "notification-hub", EnvGet: envOf(map[string]string{
		"SCENARIO_NAME": "notification-hub", "VROOLI_SCENARIO": "notification-hub", "SCENARIO_DATA_DIR": storage.LifecycleDataDir(scenarioDir),
	})})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(scenarioDir, "data", "notification-hub.db"), liveHub, "what the live API opens")

	for name, env := range map[string]map[string]string{
		"no lifecycle environment at all (run by hand, by a harness, by an agent)": {},
		"the lifecycle's own environment":                                          {"SCENARIO_NAME": "notification-hub", "SCENARIO_DATA_DIR": storage.LifecycleDataDir(scenarioDir)},
		"an environment inherited from another scenario": {
			"SCENARIO_NAME": "agent-manager", "VROOLI_SCENARIO": "agent-manager", "SCENARIO_DATA_DIR": "/elsewhere/agent-manager/data",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := servingDatabasePath(envOf(env), scenarioDir)
			require.NoError(t, err)
			require.Equal(t, liveHub, got)
		})
	}
}

func TestServingDatabasePath_AShadowVariantChecksItsOwnDatabaseNotLives(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // a variant resolves under the runtime home; never the operator's
	scenarioDir := filepath.Join(t.TempDir(), "scenarios", "notification-hub")
	got, err := servingDatabasePath(envOf(map[string]string{
		"VROOLI_STORAGE_NAMESPACE": "notification-hub_shadow", "VROOLI_VARIANT": "shadow",
		"SCENARIO_DATA_DIR": storage.LifecycleDataDir(scenarioDir),
	}), scenarioDir)
	require.NoError(t, err)
	require.Contains(t, got, "notification-hub_shadow")
	require.NotContains(t, got, scenarioDir)
}

func TestScenarioDir_UsesTheLifecyclesValueElseTheParentOfTheAPIDirectory(t *testing.T) {
	apiDir := filepath.Join(t.TempDir(), "scenarios", "notification-hub", "api")
	getwd := func() (string, error) { return apiDir, nil }
	got, err := scenarioDir(envOf(nil), getwd)
	require.NoError(t, err)
	require.Equal(t, filepath.Dir(apiDir), got)
	got, err = scenarioDir(envOf(map[string]string{"VROOLI_SCENARIO_DIR": "/workspace/scenarios/notification-hub"}), getwd)
	require.NoError(t, err)
	require.Equal(t, "/workspace/scenarios/notification-hub", got)
}

func TestRun_ChecksTheServingDatabaseAndSaysWhichOne(t *testing.T) {
	scenarioDir := filepath.Join(t.TempDir(), "scenarios", "notification-hub")
	databasePath := filepath.Join(scenarioDir, "data", "notification-hub.db")
	dsn, err := storage.SQLiteDSNAt(databasePath, storage.SQLiteTuning{})
	require.NoError(t, err)
	serving, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	_, err = serving.Exec(`CREATE TABLE asks (id TEXT PRIMARY KEY, source TEXT, requester_ref TEXT)`)
	require.NoError(t, err)
	_, err = serving.Exec(`INSERT INTO asks VALUES ('a', 'src', 'ref'), ('b', 'src', 'ref')`)
	require.NoError(t, err)
	require.NoError(t, serving.Close())

	var stdout, stderr bytes.Buffer
	// No lifecycle environment: it must still find, and condemn, the serving database.
	code := run(&stdout, &stderr, envOf(nil), scenarioDir)

	require.Equal(t, 1, code)
	require.Contains(t, stdout.String(), databasePath, "the output names the file that was checked")
	require.Contains(t, stderr.String(), "duplicate_groups=1")
}

func TestRun_AMissingDatabaseIsClearAndIsNotCreated(t *testing.T) {
	scenarioDir := filepath.Join(t.TempDir(), "scenarios", "notification-hub")
	var stdout, stderr bytes.Buffer

	code := run(&stdout, &stderr, envOf(nil), scenarioDir)

	require.Zero(t, code)
	require.Contains(t, stdout.String(), "clear (no database yet)")
	require.Empty(t, stderr.String())
	_, err := os.Stat(filepath.Join(scenarioDir, "data"))
	require.ErrorIs(t, err, os.ErrNotExist, "checking must not create an empty database")
}
