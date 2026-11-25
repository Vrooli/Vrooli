package derivation

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
	_ "modernc.org/sqlite"
)

func TestSQLiteStoreIsAppendOnly(t *testing.T) {
	db, err := sql.Open("sqlite", "file:derivation-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	store := NewSQLiteStore(db)
	version, err := store.NextVersion(context.Background(), "sha256-x")
	require.NoError(t, err)
	require.Equal(t, 1, version)
	require.NoError(t, store.Append(context.Background(), Result{DocumentHash: "sha256-x", Version: version, State: 1}))
	version, err = store.NextVersion(context.Background(), "sha256-x")
	require.NoError(t, err)
	require.Equal(t, 2, version)
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM derivation_versions`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestSQLiteStorePersistsTerminalDispositionFields(t *testing.T) { // [REQ:DOC-P0-027]
	db, err := sql.Open("sqlite", "file:derivation-terminal-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	store := NewSQLiteStore(db)
	want := Result{
		DocumentHash: "sha256-terminal", Version: 1, Tier: 1,
		State:  documentpb.TerminalState_TERMINAL_STATE_HANDLER_FAILED,
		Reason: "malformed handler response", Remedy: "inspect the handler error and retry with another chain",
		Skipped: []string{},
	}
	require.NoError(t, store.Append(context.Background(), want))

	var state int
	var reason, remedy, skippedJSON string
	err = db.QueryRowContext(context.Background(), `SELECT terminal_state, reason, remedy, skipped_json FROM derivation_versions WHERE document_hash = ? AND version = ?`, want.DocumentHash, want.Version).Scan(&state, &reason, &remedy, &skippedJSON)
	require.NoError(t, err)
	require.Equal(t, int(want.State), state)
	require.Equal(t, want.Reason, reason)
	require.Equal(t, want.Remedy, remedy)
	require.JSONEq(t, `[]`, skippedJSON)
}
