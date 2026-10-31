package conversations

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestAskIdentityDuplicateGroupsAllowsEmptyLegacyReferences(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	for _, id := range []string{"legacy-empty-a", "legacy-empty-b"} {
		_, err := db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, requester_ref) VALUES (?, ?, 'question', '[]', '2026-10-18T00:00:00Z', 'pending', '2026-10-17T00:00:00Z', '2026-10-17T00:00:00Z', 'legacy-source', '')`, id, "notice-"+id)
		require.NoError(t, err)
	}

	groups, err := AskIdentityDuplicateGroups(ctx, db)
	require.NoError(t, err)
	require.Zero(t, groups, "the index predicate excludes empty legacy references")
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM asks WHERE requester_ref=''`).Scan(&count))
	require.Equal(t, 2, count)
}

func TestAskIdentityDuplicateGroupsFindsCrossRecipientAndStateConflicts(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	legacySchema := strings.ReplaceAll(Schema(), "CREATE UNIQUE INDEX IF NOT EXISTS idx_asks_requester_identity\n  ON asks(source, requester_ref) WHERE requester_ref <> '';\n", "")
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(func() string { return legacySchema })))
	_, err := db.ExecContext(ctx, `CREATE TABLE notifications (id TEXT PRIMARY KEY, requested_by TEXT NOT NULL)`)
	require.NoError(t, err)
	for _, row := range []struct{ id, recipient, state string }{
		{"ask-pending", "alice", "pending"},
		{"ask-answered", "bob", "answered"},
	} {
		noticeID := "notice-" + row.id
		_, err := db.ExecContext(ctx, `INSERT INTO notifications (id, requested_by) VALUES (?, ?)`, noticeID, row.recipient)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, requester_ref) VALUES (?, ?, 'private question', '["yes"]', '2026-10-18T00:00:00Z', ?, '2026-10-17T00:00:00Z', '2026-10-17T00:00:00Z', 'agent-manager', 'same-ref')`, row.id, noticeID, row.state)
		require.NoError(t, err)
	}

	groups, err := AskIdentityDuplicateGroups(ctx, db)
	require.NoError(t, err)
	require.EqualValues(t, 1, groups, "identity ownership is global across recipient and Ask state")
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM asks WHERE source='agent-manager' AND requester_ref='same-ref'`).Scan(&count))
	require.Equal(t, 2, count, "the preflight must not modify conflicting rows")
	var recipientCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT n.requested_by) FROM asks a JOIN notifications n ON n.id=a.notification_id WHERE a.source='agent-manager' AND a.requester_ref='same-ref'`).Scan(&recipientCount))
	require.Equal(t, 2, recipientCount, "fixture must prove duplicates across recipients")
}

func TestAskIdentityDuplicateGroupsTreatsMissingAskTableAsClear(t *testing.T) {
	db := databasetest.NewSQLite(t)
	groups, err := AskIdentityDuplicateGroups(context.Background(), db)
	require.NoError(t, err)
	require.Zero(t, groups)
}

func TestAskIdentityDuplicateGroupsFailsClosedWhenIdentityQueryIsUnavailable(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	_, err := db.ExecContext(ctx, `CREATE TABLE asks (id TEXT PRIMARY KEY)`)
	require.NoError(t, err)

	groups, err := AskIdentityDuplicateGroups(ctx, db)
	require.Error(t, err)
	require.Zero(t, groups, "unavailable identity columns must not produce a clear result")
	require.Contains(t, err.Error(), "count duplicate Ask requester identities")
}
