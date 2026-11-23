package opencodearchive

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

	_ "modernc.org/sqlite"
)

// OpenCodeSessionArchives reads OpenCode's local SQLite archive in read-only
// mode and maintains JSONL snapshots in a private derived cache. The
// snapshots are source adapters, not operator evidence; records retain the
// native {info, parts} message dialect consumed by agenttranscript.
//
// OpenCode changed its storage tables in v2. Both generations store message
// info and parts as JSON data columns, so the adapter probes table names and
// does not depend on version-specific message field layouts.
func Refresh(ctx context.Context, databasePath, cacheRoot string) (int, error) {
	databasePath = strings.TrimSpace(databasePath)
	cacheRoot = strings.TrimSpace(cacheRoot)
	if databasePath == "" || cacheRoot == "" {
		return 0, nil
	}
	info, err := os.Stat(databasePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("OpenCode archive is not a regular file: %s", databasePath)
	}
	db, err := sql.Open("sqlite", openCodeReadOnlyDSN(databasePath))
	if err != nil {
		return 0, fmt.Errorf("open OpenCode archive read-only: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return 0, fmt.Errorf("read OpenCode archive: %w", err)
	}
	messageTable, partTable, err := openCodeArchiveTables(ctx, db)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(cacheRoot, 0o700); err != nil {
		return 0, fmt.Errorf("create OpenCode archive cache: %w", err)
	}
	rows, err := db.QueryContext(ctx, "SELECT DISTINCT session_id FROM "+messageTable+" WHERE session_id <> '' ORDER BY session_id")
	if err != nil {
		return 0, fmt.Errorf("list OpenCode sessions: %w", err)
	}
	var sessions []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			rows.Close()
			return 0, err
		}
		sessions = append(sessions, sessionID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	keep := make(map[string]bool, len(sessions))
	for _, sessionID := range sessions {
		keep[archiveSessionName(sessionID)] = true
	}
	// Reclaim deleted-session snapshots before calculating the bounded cache
	// budget. This directory is isolated to this native database.
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		staleTemp := strings.HasPrefix(entry.Name(), ".opencode-archive-") && strings.HasSuffix(entry.Name(), ".tmp")
		staleSnapshot := strings.HasSuffix(entry.Name(), ".jsonl") && !keep[entry.Name()]
		if !entry.IsDir() && (staleTemp || staleSnapshot) {
			if err := os.Remove(filepath.Join(cacheRoot, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return 0, err
			}
		}
	}
	cacheBytes, err := archiveCacheBytes(cacheRoot)
	if err != nil {
		return 0, err
	}
	for _, sessionID := range sessions {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		name := archiveSessionName(sessionID)
		path := filepath.Join(cacheRoot, name)
		oldSize, _ := fileSize(path)
		remaining := maxOpenCodeCacheBytes - cacheBytes + oldSize
		if remaining <= 0 {
			return 0, fmt.Errorf("OpenCode archive cache limit reached (%d bytes)", maxOpenCodeCacheBytes)
		}
		limit := min(maxOpenCodeSnapshotBytes, remaining)
		newSize, err := materializeOpenCodeSession(ctx, db, messageTable, partTable, sessionID, path, limit)
		if err != nil {
			return 0, fmt.Errorf("materialize OpenCode session %s: %w", sessionID, err)
		}
		cacheBytes += newSize - oldSize
	}
	return len(sessions), nil
}

// CacheDirectory isolates snapshots for one native database inside the
// caller-owned cache root. It prevents pruning one OpenCode archive from
// deleting another database's derived files.
func CacheDirectory(databasePath, cacheRoot string) string {
	canonical, err := filepath.Abs(filepath.Clean(databasePath))
	if err != nil {
		canonical = filepath.Clean(databasePath)
	}
	sum := sha256.Sum256([]byte(canonical))
	return filepath.Join(cacheRoot, hex.EncodeToString(sum[:]))
}

// DataDirectoryForEnv is OpenCode's native data root after XDG/HOME overrides.
func DataDirectoryForEnv(env map[string]string) string {
	base, overridden := env["XDG_DATA_HOME"]
	if !overridden {
		base = os.Getenv("XDG_DATA_HOME")
	}
	if base = strings.TrimSpace(base); base != "" {
		return filepath.Join(base, "opencode")
	}
	home, overridden := env["HOME"]
	if !overridden {
		home, _ = os.UserHomeDir()
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode")
}

// DatabasePathForEnv resolves OpenCode's local archive in the selected
// environment without probing or modifying the database.
func DatabasePathForEnv(env map[string]string) string {
	if root := DataDirectoryForEnv(env); root != "" {
		return filepath.Join(root, "opencode.db")
	}
	return ""
}

// SessionExists checks OpenCode's version-selected session catalog without
// initializing or migrating the user's native store.
func SessionExists(ctx context.Context, databasePath, table, sessionID string) (bool, error) {
	if table != "session" && table != "session_v2" {
		return false, fmt.Errorf("unsupported OpenCode session table %q", table)
	}
	db, err := sql.Open("sqlite", openCodeReadOnlyDSN(databasePath))
	if err != nil {
		return false, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	var id string
	err = db.QueryRowContext(ctx, "SELECT id FROM "+table+" WHERE id = ?", sessionID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func openCodeArchiveTables(ctx context.Context, db *sql.DB) (string, string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return "", "", err
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
	return "", "", errors.New("OpenCode archive has no supported message/part tables")
}

func openCodeReadOnlyDSN(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: "mode=ro"}).String()
}

func archiveSessionName(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(sum[:]) + ".jsonl"
}

const maxOpenCodeSnapshotBytes = int64(1 << 30)
const maxOpenCodeCacheBytes = int64(8 << 30)

func materializeOpenCodeSession(ctx context.Context, db *sql.DB, messageTable, partTable, sessionID, path string, limit int64) (int64, error) {
	query := "SELECT m.id, m.data, p.data FROM " + messageTable + " AS m LEFT JOIN " + partTable + " AS p ON p.message_id = m.id AND p.session_id = m.session_id WHERE m.session_id = ? ORDER BY m.seq, m.id, p.ordinal"
	messageRows, err := db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return 0, fmt.Errorf("read messages: %w", err)
	}
	defer messageRows.Close()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".opencode-archive-*.tmp")
	if err != nil {
		return 0, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	writer := &boundedArchiveWriter{w: bufio.NewWriterSize(tmp, 64*1024), remaining: limit}
	var currentID string
	var currentInfo []byte
	var parts []json.RawMessage
	flush := func() error {
		if currentID == "" {
			return nil
		}
		line, err := archiveMessageLine(sessionID, currentID, currentInfo, parts)
		if err != nil {
			return err
		}
		_, err = writer.Write(append(line, '\n'))
		return err
	}
	for messageRows.Next() {
		if err := ctx.Err(); err != nil {
			tmp.Close()
			return 0, err
		}
		var messageID string
		var data, partData []byte
		if err := messageRows.Scan(&messageID, &data, &partData); err != nil {
			tmp.Close()
			return 0, err
		}
		if currentID != messageID {
			if err := flush(); err != nil {
				tmp.Close()
				return 0, err
			}
			currentID, currentInfo, parts = messageID, append(currentInfo[:0], data...), nil
		}
		if len(partData) > 0 {
			if !json.Valid(partData) {
				tmp.Close()
				return 0, fmt.Errorf("OpenCode part for message %s is not JSON", messageID)
			}
			parts = append(parts, append(json.RawMessage(nil), partData...))
		}
	}
	if err := messageRows.Err(); err != nil {
		tmp.Close()
		return 0, err
	}
	if err := flush(); err != nil {
		tmp.Close()
		return 0, err
	}
	if err := writer.Flush(); err != nil {
		tmp.Close()
		return 0, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return 0, err
	}
	info, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		return 0, err
	}
	newSize := info.Size()
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if equalFiles(path, tmpPath) {
		return newSize, nil
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return 0, err
	}
	return newSize, nil
}

func archiveMessageLine(sessionID, messageID string, rawInfo []byte, parts []json.RawMessage) ([]byte, error) {
	var info map[string]any
	if err := json.Unmarshal(rawInfo, &info); err != nil || info == nil {
		return nil, fmt.Errorf("OpenCode message %s has invalid info JSON", messageID)
	}
	if _, ok := info["sessionID"]; !ok {
		info["sessionID"] = sessionID
	}
	if parts == nil {
		parts = []json.RawMessage{}
	}
	return json.Marshal(struct {
		Info  map[string]any    `json:"info"`
		Parts []json.RawMessage `json:"parts"`
	}{Info: info, Parts: parts})
}

func archiveCacheBytes(root string) (int64, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return 0, err
		}
		total += info.Size()
	}
	return total, nil
}

func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

type boundedArchiveWriter struct {
	w         io.Writer
	remaining int64
}

func (w *boundedArchiveWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.remaining {
		return 0, fmt.Errorf("OpenCode session snapshot exceeds %d bytes", maxOpenCodeSnapshotBytes)
	}
	n, err := w.w.Write(data)
	w.remaining -= int64(n)
	return n, err
}

func (w *boundedArchiveWriter) Flush() error {
	if flusher, ok := w.w.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

func equalFiles(first, second string) bool {
	firstFile, err := os.Open(first)
	if err != nil {
		return false
	}
	defer firstFile.Close()
	secondFile, err := os.Open(second)
	if err != nil {
		return false
	}
	defer secondFile.Close()
	firstInfo, err := firstFile.Stat()
	if err != nil {
		return false
	}
	secondInfo, err := secondFile.Stat()
	if err != nil || firstInfo.Size() != secondInfo.Size() {
		return false
	}
	firstHash, secondHash := sha256.New(), sha256.New()
	if _, err := io.Copy(firstHash, firstFile); err != nil {
		return false
	}
	if _, err := io.Copy(secondHash, secondFile); err != nil {
		return false
	}
	return string(firstHash.Sum(nil)) == string(secondHash.Sum(nil))
}
