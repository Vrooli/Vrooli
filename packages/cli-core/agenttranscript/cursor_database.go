package agenttranscript

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CursorKVDatabaseQueryer is the query surface shared by database/sql DBs and
// transactions. The caller owns the read-only connection and transaction.
type CursorKVDatabaseQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readCursorFileSnapshot(ctx context.Context, path string, maxBytes int64, read func(context.Context, CursorKVDatabaseQueryer, int64) ([]byte, bool, error)) ([]byte, bool, error) {
	uri := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&_pragma=busy_timeout(5000)"}).String()
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, false, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	encoded, exceeded, err := read(ctx, tx, maxBytes)
	if err != nil || exceeded {
		return nil, exceeded, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return encoded, false, nil
}

// ReadCursorWorkspaceSnapshotFile opens a local Cursor database read-only,
// pins one SQLite view, and emits a bounded full-workspace snapshot. The
// scenario must register the SQLite driver; this shared package imports none.
func ReadCursorWorkspaceSnapshotFile(ctx context.Context, path string, maxBytes int64) ([]byte, bool, error) {
	return readCursorFileSnapshot(ctx, path, maxBytes, ReadCursorWorkspaceSnapshotJSON)
}

// ReadCursorConversationSnapshotFile performs the same bounded read for one
// composer while retaining the Agent Manager per-composer transport format.
func ReadCursorConversationSnapshotFile(ctx context.Context, path, composerID string, maxBytes int64) ([]byte, bool, error) {
	if strings.TrimSpace(composerID) == "" {
		return nil, false, fmt.Errorf("Cursor composer ID is required")
	}
	return readCursorFileSnapshot(ctx, path, maxBytes, func(ctx context.Context, queryer CursorKVDatabaseQueryer, maxBytes int64) ([]byte, bool, error) {
		return ReadCursorConversationSnapshotJSON(ctx, queryer, composerID, maxBytes)
	})
}

// ReadCursorWorkspaceSnapshotJSON reads and groups every native composer and
// bubble row from one caller-pinned database view. The byte limit and query
// policy stay explicit at the caller boundary. The returned snapshot preserves
// embedded native JSON values, as expected by the Web Console archive tailer.
func ReadCursorWorkspaceSnapshotJSON(ctx context.Context, queryer CursorKVDatabaseQueryer, maxBytes int64) ([]byte, bool, error) {
	if queryer == nil {
		return nil, false, fmt.Errorf("Cursor workspace snapshot requires a database queryer")
	}
	rows, err := queryer.QueryContext(ctx, `SELECT key, value FROM cursorDiskKV WHERE key LIKE 'composerData:%' OR key LIKE 'bubbleId:%' ORDER BY key`)
	if err != nil {
		return nil, false, fmt.Errorf("read Cursor transcript key/value rows: %w", err)
	}
	builder := NewCursorWorkspaceSnapshotBuilder()
	_, exceeded, err := builder.AddRows(rows, 0, maxBytes)
	if err != nil || exceeded {
		return nil, exceeded, err
	}
	encoded, err := json.Marshal(builder.WorkspaceSnapshot())
	if err != nil {
		return nil, false, fmt.Errorf("encode Cursor workspace snapshot: %w", err)
	}
	return encoded, false, nil
}

// ReadCursorConversationSnapshotJSON reads one native composer and its bubble
// rows, bounded by their source bytes, in the caller's pinned database view.
// Byte slices retain the Agent Manager import snapshot's base64 JSON shape.
func ReadCursorConversationSnapshotJSON(ctx context.Context, queryer CursorKVDatabaseQueryer, composerID string, maxBytes int64) ([]byte, bool, error) {
	if queryer == nil {
		return nil, false, fmt.Errorf("Cursor conversation snapshot requires a database queryer")
	}
	if strings.TrimSpace(composerID) == "" {
		return nil, false, fmt.Errorf("Cursor composer ID is required")
	}
	var composer []byte
	if err := queryer.QueryRowContext(ctx, `SELECT value FROM cursorDiskKV WHERE key = ?`, "composerData:"+composerID).Scan(&composer); err != nil {
		return nil, false, fmt.Errorf("read Cursor composer header: %w", err)
	}
	builder := NewCursorWorkspaceSnapshotBuilder()
	builder.AddKV("composerData:"+composerID, composer)
	prefix := "bubbleId:" + composerID + ":"
	rows, err := queryer.QueryContext(ctx, `SELECT key, value FROM cursorDiskKV WHERE substr(key, 1, length(?)) = ? ORDER BY key`, prefix, prefix)
	if err != nil {
		return nil, false, fmt.Errorf("read Cursor composer bubbles: %w", err)
	}
	_, exceeded, err := builder.AddRows(rows, int64(len(composer)), maxBytes)
	if err != nil || exceeded {
		return nil, exceeded, err
	}
	encoded, err := json.Marshal(builder.ConversationSnapshot(composerID))
	if err != nil {
		return nil, false, fmt.Errorf("encode Cursor composer snapshot: %w", err)
	}
	return encoded, false, nil
}
