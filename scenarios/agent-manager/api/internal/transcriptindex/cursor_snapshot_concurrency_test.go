package transcriptindex_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"agent-manager/internal/transcriptindex"
	"github.com/vrooli/cli-core/agenttranscript"
	_ "modernc.org/sqlite"
)

func TestCursorComposerSnapshotKeepsGenerationAcrossConcurrentWriter(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.vscdb")
	writer, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Exec(`PRAGMA journal_mode=WAL;
		CREATE TABLE cursorDiskKV (key TEXT PRIMARY KEY, value BLOB);
		INSERT INTO cursorDiskKV(key,value) VALUES
		('composerData:snapshot-composer', '{"composerId":"snapshot-composer","fullConversationHeadersOnly":[{"bubbleId":"bubble-1","type":1}]}'),
		('bubbleId:snapshot-composer:bubble-1', '{"text":"synthetic-generation-one"}');`); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	reader.SetMaxOpenConns(1)
	readTx, err := reader.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	var pinned int
	if err := readTx.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM cursorDiskKV`).Scan(&pinned); err != nil {
		t.Fatal(err)
	}
	if pinned != 2 {
		t.Fatalf("reader snapshot did not pin fixture: rows=%d", pinned)
	}

	writeDB, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	writeTx, err := writeDB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeTx.Exec(`UPDATE cursorDiskKV SET value='{"composerId":"snapshot-composer","fullConversationHeadersOnly":[{"bubbleId":"bubble-1","type":1}]}' WHERE key='composerData:snapshot-composer'; UPDATE cursorDiskKV SET value='{"text":"synthetic-generation-two"}' WHERE key='bubbleId:snapshot-composer:bubble-1'`); err != nil {
		_ = writeTx.Rollback()
		_ = writeDB.Close()
		t.Fatal(err)
	}
	if err := writeTx.Commit(); err != nil {
		_ = writeDB.Close()
		t.Fatal(err)
	}
	if err := writeDB.Close(); err != nil {
		t.Fatal(err)
	}

	encoded, exceeded, err := agenttranscript.ReadCursorConversationSnapshotJSON(context.Background(), readTx, "snapshot-composer", 16<<20)
	if err != nil || exceeded {
		t.Fatalf("read pinned Cursor composer snapshot: exceeded=%v err=%v", exceeded, err)
	}
	snapshot, err := agenttranscript.DecodeCursorConversationSnapshot(encoded)
	if err != nil {
		t.Fatal(err)
	}
	messages := agenttranscript.ParseCursorConversation(snapshot.Composer, snapshot.Bubbles)
	if len(messages) != 1 || messages[0].Text != "synthetic-generation-one" {
		t.Fatalf("reader mixed Cursor generations: messages=%+v", messages)
	}
	latest, err := transcriptindex.ReadCursorSnapshot(context.Background(), path, "snapshot-composer")
	if err != nil {
		t.Fatal(err)
	}
	latestSnapshot, err := agenttranscript.DecodeCursorConversationSnapshot(latest)
	if err != nil {
		t.Fatal(err)
	}
	latestMessages := agenttranscript.ParseCursorConversation(latestSnapshot.Composer, latestSnapshot.Bubbles)
	if len(latestMessages) != 1 || latestMessages[0].Text != "synthetic-generation-two" {
		t.Fatalf("fresh Cursor snapshot missed writer commit: messages=%+v", latestMessages)
	}
}
