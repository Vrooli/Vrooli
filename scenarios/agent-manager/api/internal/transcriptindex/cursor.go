package transcriptindex

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/agenttranscript"

	_ "modernc.org/sqlite"
)

// ReadCursorSnapshot reads one composer and its bubbles in a read-only
// SQLite transaction. The transaction pins a single database view so a
// concurrent Cursor write cannot mix header and bubble generations.
func ReadCursorSnapshot(ctx context.Context, path, composerID string) ([]byte, error) {
	snapshot, exceeded, err := agenttranscript.ReadCursorConversationSnapshotFile(ctx, path, composerID, maxTranscriptDocumentBytes)
	if err != nil {
		return nil, err
	}
	if exceeded {
		return nil, fmt.Errorf("Cursor composer snapshot exceeds %d-byte import limit", maxTranscriptDocumentBytes)
	}
	return snapshot, nil
}

// cursorComposerRemoved is true only when the native composer header is
// definitively absent. A malformed or mismatched header remains present and
// is left to the import gate rather than treated as a deletion.
func cursorComposerRemoved(ctx context.Context, path, composerID string) (bool, error) {
	if strings.TrimSpace(composerID) == "" {
		return false, fmt.Errorf("Cursor composer ID is required")
	}
	uri := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&_pragma=busy_timeout(5000)"}).String()
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return false, err
	}
	defer db.Close()
	var present int
	err = db.QueryRowContext(ctx, `SELECT 1 FROM cursorDiskKV WHERE key = ?`, "composerData:"+composerID).Scan(&present)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func discoverCursorDB(path string, root Root) ([]Candidate, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Cursor state database is not a regular file")
	}
	uri := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&_pragma=busy_timeout(5000)"}).String()
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT key, value FROM cursorDiskKV WHERE key LIKE 'composerData:%' ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("discover Cursor composer headers: %w", err)
	}
	var out []Candidate
	for rows.Next() {
		var key string
		var header []byte
		if err := rows.Scan(&key, &header); err != nil {
			rows.Close()
			return nil, err
		}
		id := strings.TrimPrefix(key, "composerData:")
		if id == key || id == "" {
			continue
		}
		if agenttranscript.CursorConversationSessionID(header) != id {
			continue
		}
		snapshot, err := ReadCursorSnapshot(context.Background(), path, id)
		if err != nil {
			rows.Close()
			return nil, err
		}
		snapshotInfo, err := os.Stat(path)
		if err != nil || !snapshotInfo.Mode().IsRegular() {
			rows.Close()
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("Cursor state database changed type during composer discovery")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, Candidate{Root: root, Path: abs, Key: "cursor:" + abs + "#" + id, SessionID: id, Size: int64(len(snapshot)), MtimeUnixNano: snapshotInfo.ModTime().UnixNano(), ContentHash: agenttranscript.SHA256Bytes(snapshot)})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return out, nil
}
