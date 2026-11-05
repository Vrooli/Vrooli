package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"notification-hub/internal/conversations"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
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
