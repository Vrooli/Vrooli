package conversationsearch

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSQLiteSourcePagesMessagesInStableOrderAndResolvesDeletion(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	createSourceFixtureSchema(t, db)
	seedSourceFixture(t, db)
	source, err := NewSQLiteSource(db, fixtureNormalizer(t, "recipe-v1", 512, 32))
	require.NoError(t, err)

	first, err := source.LoadSourcePage(context.Background(), nil, 1)
	require.NoError(t, err)
	require.Len(t, first.Documents, 1)
	require.Equal(t, "event-1", first.Documents[0].SourceEventID)
	require.NotNil(t, first.NextCursor)

	second, err := source.LoadSourcePage(context.Background(), first.NextCursor, 1)
	require.NoError(t, err)
	require.Len(t, second.Documents, 1)
	require.Equal(t, "event-3", second.Documents[0].SourceEventID)
	require.Nil(t, second.NextCursor)
	require.Equal(t, "claude-code", second.Documents[0].Harness)
	require.Equal(t, "fixture-session", second.Documents[0].SourceSessionID)
	require.Equal(t, "/workspace/project", second.Documents[0].ProjectScope)
	require.Equal(t, []string{"implementation", "search"}, second.Documents[0].Workloads)

	// event-2 was logically deleted and must never reach either projection.
	require.NotEqual(t, "event-2", first.Documents[0].SourceEventID)
	require.NotEqual(t, "event-2", second.Documents[0].SourceEventID)
}

func TestSQLiteSourceSnapshotExcludesLaterBackdatedImports(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	createSourceFixtureSchema(t, db)
	seedSourceFixture(t, db)
	source, err := NewSQLiteSource(db, fixtureNormalizer(t, "recipe-v1", 512, 32))
	require.NoError(t, err)
	snapshot, err := source.SnapshotCursor(context.Background())
	require.NoError(t, err)
	require.Positive(t, snapshot.SnapshotMaxEventRowID)

	first, err := source.LoadSourcePage(context.Background(), snapshot, 1)
	require.NoError(t, err)
	require.Equal(t, "event-1", first.Documents[0].SourceEventID)
	require.Equal(t, snapshot.SnapshotMaxEventRowID, first.NextCursor.SnapshotMaxEventRowID)

	_, err = db.Exec(`INSERT INTO run_events VALUES ('late-import','run-1',9,'message',?,1,'{"role":"assistant","content":"backdated import","messageId":"late-message"}')`, fixtureTime(2))
	require.NoError(t, err)
	second, err := source.LoadSourcePage(context.Background(), first.NextCursor, 10)
	require.NoError(t, err)
	require.Len(t, second.Documents, 1)
	require.Equal(t, "event-3", second.Documents[0].SourceEventID)

	fresh, err := source.LoadSourcePage(context.Background(), nil, 10)
	require.NoError(t, err)
	require.Len(t, fresh.Documents, 3)
}

func TestSQLiteSourceDecodesMinimalMessagePayloadVersion(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	createSourceFixtureSchema(t, db)
	insertRunFixture(t, db)
	_, err := db.Exec(`INSERT INTO run_events(id, run_id, sequence, event_type, timestamp, schema_version, data)
        VALUES ('legacy-event', 'run-1', 1, 'message', ?, 1,
		'{"role":"assistant","content":"legacy payload","messageId":"legacy-message","providerOrigin":"codex"}')`, fixtureTime(1))
	require.NoError(t, err)
	source, err := NewSQLiteSource(db, fixtureNormalizer(t, "recipe-v1", 512, 32))
	require.NoError(t, err)

	page, err := source.LoadSourcePage(context.Background(), nil, 10)
	require.NoError(t, err)
	require.Len(t, page.Documents, 1)
	require.Equal(t, "legacy-message", page.Documents[0].SourceMessageID)
	require.Equal(t, "codex", page.Documents[0].ProviderOrigin)
}

func TestSQLiteSourceClassifiesToolEventsForExplicitRetrieval(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	createSourceFixtureSchema(t, db)
	insertRunFixture(t, db)
	db.MustExec(`INSERT INTO run_events VALUES
        ('tool-call','run-1',1,'tool_call','` + fixtureTime(1) + `',1,'{"toolName":"Search","toolCallId":"call-1","input":{"query":"capacity"}}'),
        ('tool-result','run-1',2,'tool_result','` + fixtureTime(2) + `',1,'{"toolName":"Search","toolCallId":"call-1","output":"bounded result","success":true}')`)
	source, err := NewSQLiteSource(db, fixtureNormalizer(t, "recipe-v1", 512, 32))
	require.NoError(t, err)

	page, err := source.LoadSourcePage(context.Background(), nil, 10)
	require.NoError(t, err)
	require.Len(t, page.Documents, 2)
	require.Equal(t, ContentClassToolCall, page.Documents[0].ContentClass)
	require.Equal(t, ContentClassToolResult, page.Documents[1].ContentClass)
}

func TestSQLiteSourceLoadsOnlyNamedRunForIncrementalProjection(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	createSourceFixtureSchema(t, db)
	seedSourceFixture(t, db)
	db.MustExec(`INSERT INTO runs VALUES (
        'run-2','task-1','Other run','complete','','codex',
        'other-session','','','model-actual','model-requested','profile-1','recall',
        'implementation','other'
    )`)
	db.MustExec(`INSERT INTO run_events VALUES
        ('other-event','run-2',1,'message','` + fixtureTime(1) + `',1,
        '{"role":"assistant","content":"must not be loaded","messageId":"other-message"}')`)
	source, err := NewSQLiteSource(db, fixtureNormalizer(t, "recipe-v1", 512, 32))
	require.NoError(t, err)

	documents, err := source.LoadRunDocuments(context.Background(), "run-1")
	require.NoError(t, err)
	require.Len(t, documents, 2)
	require.Equal(t, "event-1", documents[0].SourceEventID)
	require.Equal(t, "event-3", documents[1].SourceEventID)
	for _, document := range documents {
		require.Equal(t, "run-1", document.SourceRunID)
	}

	_, err = source.LoadRunDocuments(context.Background(), " ")
	require.ErrorContains(t, err, "run id")
}

func TestSQLiteSourceRejectsUnboundedPagesAndInvalidCursor(t *testing.T) {
	t.Parallel()
	db := openProjectionTestDB(t)
	createSourceFixtureSchema(t, db)
	source, err := NewSQLiteSource(db, fixtureNormalizer(t, "recipe-v1", 512, 32))
	require.NoError(t, err)
	_, err = source.LoadSourcePage(context.Background(), nil, maxSourcePageSize+1)
	require.ErrorContains(t, err, "limit")
	_, err = source.LoadSourcePage(context.Background(), &SourceCursor{}, 1)
	require.ErrorContains(t, err, "cursor")
}

func createSourceFixtureSchema(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`CREATE TABLE tasks (
        id TEXT PRIMARY KEY, project_root TEXT, scope_path TEXT
    );
    CREATE TABLE runs (
        id TEXT PRIMARY KEY, task_id TEXT, label TEXT, status TEXT,
        import_source_harness TEXT, harness_kind TEXT,
        import_source_session_id TEXT, harness_session_id TEXT, session_id TEXT,
        actual_model TEXT, requested_model TEXT, agent_profile_id TEXT, tag TEXT,
        workload_kind TEXT, workload_key TEXT
    );
    CREATE TABLE run_events (
        id TEXT PRIMARY KEY, run_id TEXT NOT NULL, sequence INTEGER NOT NULL,
        event_type TEXT NOT NULL, timestamp TEXT NOT NULL, schema_version INTEGER NOT NULL, data TEXT NOT NULL
    );`)
}

func seedSourceFixture(t *testing.T, db *sqlx.DB) {
	t.Helper()
	insertRunFixture(t, db)
	for _, statement := range []string{
		`INSERT INTO run_events VALUES ('event-1','run-1',1,'message','` + fixtureTime(1) + `',1,'{"role":"operator","content":"first message","messageId":"message-1","providerOrigin":"claude"}')`,
		`INSERT INTO run_events VALUES ('event-2','run-1',2,'message','` + fixtureTime(2) + `',1,'{"role":"assistant","content":"deleted secret","messageId":"message-2"}')`,
		`INSERT INTO run_events VALUES ('delete-2','run-1',3,'message_deleted','` + fixtureTime(3) + `',1,'{"targetEventId":"event-2"}')`,
		`INSERT INTO run_events VALUES ('event-3','run-1',4,'message','` + fixtureTime(4) + `',1,'{"role":"assistant","content":"corrected result","messageId":"message-3"}')`,
	} {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}
}

func insertRunFixture(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`INSERT INTO tasks VALUES ('task-1','/workspace/project','/workspace/project')`)
	db.MustExec(`INSERT INTO runs VALUES (
        'run-1','task-1','Fixture run','complete','claude-code','claude',
        'fixture-session','','','model-actual','model-requested','profile-1','recall',
        'implementation','search'
    )`)
}

func fixtureTime(minute int) string {
	return time.Date(2026, 9, 4, 12, minute, 0, 0, time.UTC).Format(time.RFC3339Nano)
}
