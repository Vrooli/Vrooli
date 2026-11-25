package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vrooli/cli-core/agenttranscript"
)

var openCodeSnapshotMu sync.Mutex
var openCodeSnapshotSignatures = map[string]string{}

const (
	openCodeDatabaseRoot = "opencode-native-db"
	openCodeSnapshotRoot = "opencode-native-snapshot"
	openCodeSnapshotMax  = int64(1 << 30)
	openCodeSnapshotCap  = int64(8 << 30)
)

// prepareOpenCodeNativeRoots adapts the installed OpenCode SQLite archive to
// the ordinary Web Console tailer. It reads the database in read-only mode,
// writes an owned per-session JSONL snapshot, then routes each record through
// the shared agenttranscript wire projection. Agent Manager is not involved.
func prepareOpenCodeNativeRoots(ctx context.Context, roots NativeArchiveRoots) (NativeArchiveRoots, error) {
	prepared := roots.Clone()
	databasePaths := prepared[openCodeDatabaseRoot]
	cacheRoots := prepared[openCodeSnapshotRoot]
	delete(prepared, openCodeDatabaseRoot)
	delete(prepared, openCodeSnapshotRoot)
	for index, databasePath := range databasePaths {
		if strings.TrimSpace(databasePath) == "" {
			continue
		}
		if _, err := os.Stat(databasePath); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("stat OpenCode native archive: %w", err)
		}
		if index >= len(cacheRoots) || strings.TrimSpace(cacheRoots[index]) == "" {
			return nil, errors.New("OpenCode native archive snapshot root is missing")
		}
		signature, err := openCodeDatabaseSignature(databasePath)
		if err != nil {
			return nil, err
		}
		openCodeSnapshotMu.Lock()
		if openCodeSnapshotSignatures[databasePath] != signature {
			if err := refreshOpenCodeDatabaseSnapshots(ctx, databasePath, cacheRoots[index]); err != nil {
				openCodeSnapshotMu.Unlock()
				return nil, err
			}
			openCodeSnapshotSignatures[databasePath] = signature
		}
		openCodeSnapshotMu.Unlock()
		prepared["opencode"] = append(prepared["opencode"], cacheRoots[index])
	}
	return prepared, nil
}

func openCodeDatabaseSignature(databasePath string) (string, error) {
	var signature strings.Builder
	for _, path := range []string{databasePath, databasePath + "-wal"} {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			signature.WriteString(path + "=missing\n")
			continue
		}
		if err != nil {
			return "", fmt.Errorf("stat OpenCode native archive generation: %w", err)
		}
		fmt.Fprintf(&signature, "%s=%d:%d:%d\n", path, info.Size(), info.ModTime().UnixNano(), info.Mode())
	}
	return signature.String(), nil
}

func refreshOpenCodeDatabaseSnapshots(ctx context.Context, databasePath, cacheRoot string) error {
	db, err := sql.Open("sqlite", openCodeReadOnlyURI(databasePath))
	if err != nil {
		return fmt.Errorf("open OpenCode archive read-only: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("read OpenCode archive: %w", err)
	}
	messageTable, partTable, err := openCodeNativeTables(ctx, db)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheRoot, 0o700); err != nil {
		return fmt.Errorf("create Web Console OpenCode snapshot cache: %w", err)
	}
	query := "SELECT m.id,m.session_id,m.seq,m.data,p.data FROM " + messageTable + " AS m LEFT JOIN " + partTable + " AS p ON p.message_id=m.id AND p.session_id=m.session_id ORDER BY m.session_id,m.seq,m.id,p.ordinal"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("read OpenCode native messages: %w", err)
	}
	defer rows.Close()
	writer := openCodeSessionSnapshotWriter{root: cacheRoot, active: map[string]struct{}{}}
	var sessionID, messageID string
	var info []byte
	var parts []json.RawMessage
	var sessionBytes int64
	var totalBytes int64
	flushMessage := func() error {
		if messageID == "" {
			return nil
		}
		line, err := agenttranscript.MarshalOpenCodeArchiveRecord(sessionID, info, parts)
		if err != nil {
			return fmt.Errorf("project OpenCode native message: %w", err)
		}
		line = append(line, '\n')
		if int64(len(line))+sessionBytes > openCodeSnapshotMax || int64(len(line))+totalBytes > openCodeSnapshotCap {
			return errors.New("OpenCode native archive exceeds Web Console snapshot limit")
		}
		if err := writer.write(sessionID, line); err != nil {
			return err
		}
		sessionBytes += int64(len(line))
		totalBytes += int64(len(line))
		return nil
	}
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var rowMessageID, rowSessionID string
		var sequence int64
		var rowInfo, part []byte
		if err := rows.Scan(&rowMessageID, &rowSessionID, &sequence, &rowInfo, &part); err != nil {
			return fmt.Errorf("scan OpenCode native message: %w", err)
		}
		if rowSessionID == "" || filepath.Base(rowSessionID) != rowSessionID || rowSessionID == "." || rowSessionID == ".." {
			return errors.New("OpenCode native archive contains an unsafe session identity")
		}
		if rowSessionID != sessionID || rowMessageID != messageID {
			if err := flushMessage(); err != nil {
				return err
			}
			if rowSessionID != sessionID {
				if err := writer.finishSession(); err != nil {
					return err
				}
				if err := writer.beginSession(rowSessionID); err != nil {
					return err
				}
				sessionBytes = 0
				sessionID = rowSessionID
			}
			messageID, info, parts = rowMessageID, append(info[:0], rowInfo...), nil
		}
		_ = sequence // ordering is supplied by the native query; it is not content.
		if len(part) != 0 {
			if !json.Valid(part) {
				return errors.New("OpenCode native part is not valid JSON")
			}
			parts = append(parts, append(json.RawMessage(nil), part...))
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read OpenCode native message rows: %w", err)
	}
	if err := flushMessage(); err != nil {
		return err
	}
	if err := writer.finishSession(); err != nil {
		return err
	}
	if err := writer.removeMissingSessions(); err != nil {
		return err
	}
	return nil
}

func openCodeReadOnlyURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: "mode=ro"}).String()
}

func openCodeNativeTables(ctx context.Context, db *sql.DB) (string, string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return "", "", fmt.Errorf("inspect OpenCode native schema: %w", err)
	}
	defer rows.Close()
	available := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", "", err
		}
		available[name] = true
	}
	if err := rows.Err(); err != nil {
		return "", "", err
	}
	for _, pair := range [][2]string{{"session_message", "session_part"}, {"message", "part"}} {
		if available[pair[0]] && available[pair[1]] {
			return pair[0], pair[1], nil
		}
	}
	return "", "", errors.New("OpenCode native archive has no supported message/part tables")
}

type openCodeSessionSnapshotWriter struct {
	root     string
	active   map[string]struct{}
	session  string
	file     *os.File
	buffered *bufio.Writer
	tempPath string
}

func (w *openCodeSessionSnapshotWriter) beginSession(sessionID string) error {
	w.session = sessionID
	w.active[sessionID] = struct{}{}
	directory := filepath.Join(w.root, sessionID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".opencode-snapshot-*.tmp")
	if err != nil {
		return err
	}
	w.file, w.tempPath = file, file.Name()
	w.buffered = bufio.NewWriterSize(file, 64*1024)
	return nil
}

func (w *openCodeSessionSnapshotWriter) write(_ string, line []byte) error {
	if w.file == nil || w.buffered == nil {
		return errors.New("cannot create OpenCode native session snapshot")
	}
	_, err := w.buffered.Write(line)
	return err
}

func (w *openCodeSessionSnapshotWriter) finishSession() error {
	if w.file == nil {
		return nil
	}
	if err := w.buffered.Flush(); err != nil {
		_ = w.file.Close()
		w.file = nil
		return err
	}
	if err := w.file.Sync(); err != nil {
		_ = w.file.Close()
		w.file = nil
		return err
	}
	if err := w.file.Close(); err != nil {
		w.file = nil
		return err
	}
	destination := filepath.Join(w.root, w.session, "messages.json")
	if existing, err := os.Open(destination); err == nil {
		defer existing.Close()
		currentHash, newHash := sha256.New(), sha256.New()
		if _, err := io.Copy(currentHash, existing); err != nil {
			return err
		}
		temporary, err := os.Open(w.tempPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(newHash, temporary); err != nil {
			temporary.Close()
			return err
		}
		if _, err := temporary.Seek(0, io.SeekStart); err != nil {
			temporary.Close()
			return err
		}
		if string(currentHash.Sum(nil)) == string(newHash.Sum(nil)) {
			temporary.Close()
			_ = os.Remove(w.tempPath)
		} else {
			destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_TRUNC, 0o600)
			if err == nil {
				_, err = io.Copy(destinationFile, temporary)
				if err == nil {
					err = destinationFile.Sync()
				}
				closeErr := destinationFile.Close()
				if err == nil {
					err = closeErr
				}
			}
			temporary.Close()
			if err != nil {
				return fmt.Errorf("update OpenCode native session snapshot: %w", err)
			}
			_ = os.Remove(w.tempPath)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(w.tempPath, destination); err != nil {
			return fmt.Errorf("commit OpenCode native session snapshot: %w", err)
		}
	} else {
		return err
	}
	w.file, w.buffered, w.tempPath = nil, nil, ""
	return nil
}

func (w *openCodeSessionSnapshotWriter) removeMissingSessions() error {
	entries, err := os.ReadDir(w.root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, ok := w.active[entry.Name()]; ok {
			continue
		}
		if err := os.RemoveAll(filepath.Join(w.root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func openCodeSnapshotCachePath(cacheHome, databasePath string) string {
	base := filepath.Join(cacheHome, "web-console", "native-archives", "opencode")
	canonical, err := filepath.Abs(filepath.Clean(databasePath))
	if err != nil {
		canonical = filepath.Clean(databasePath)
	}
	key := sha256.Sum256([]byte(canonical))
	return filepath.Join(base, hex.EncodeToString(key[:12]))
}
