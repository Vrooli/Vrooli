// DOC: docs/concepts/ARCHITECTURE.md#event-store-sqlite-wal
// DOC: docs/internal/SECURITY-POSTURE.md
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/match"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/sqlutil"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL UNIQUE,
  source_scenario TEXT NOT NULL,
  target_scenario TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  payload BLOB,
  metadata TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now')),
  expires_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_source ON events(source_scenario);
CREATE INDEX IF NOT EXISTS idx_events_correlation ON events(correlation_id) WHERE correlation_id != '';
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at);

CREATE TABLE IF NOT EXISTS store_meta (
  key TEXT PRIMARY KEY,
  value INTEGER NOT NULL DEFAULT 0
);
INSERT OR IGNORE INTO store_meta (key, value) VALUES ('total_payload_bytes', 0);
`

// Schema returns the event-store schema for the routed database registry.
func Schema() string { return schema }

const pragmas = `
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=10000;
PRAGMA synchronous=NORMAL;
`

// SQLiteConfig holds configuration for the SQLite store.
type SQLiteConfig struct {
	DBPath        string
	MaxAge        time.Duration // retention period (default 30 days)
	MaxSizeBytes  int64         // max total payload size (default 2GB)
	QueryLimit    int           // default query limit when not specified (default 100)
	QueryLimitMax int           // maximum allowed query limit (default 1000)
	// Now is the time source used by time-dependent operations (e.g. Prune
	// cutoff). Injectable for deterministic tests — defaults to time.Now.
	Now func() time.Time
}

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db     sqlutil.DB
	rawDB  *sql.DB
	config SQLiteConfig
}

// NewSQLiteStore opens (or creates) a SQLite database and applies the schema.
func NewSQLiteStore(ctx context.Context, cfg SQLiteConfig) (*SQLiteStore, error) {
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 30 * 24 * time.Hour
	}
	if cfg.MaxSizeBytes == 0 {
		cfg.MaxSizeBytes = 2 * 1024 * 1024 * 1024 // 2GB
	}
	if cfg.QueryLimit <= 0 {
		cfg.QueryLimit = 100
	}
	if cfg.QueryLimitMax <= 0 {
		cfg.QueryLimitMax = 1000
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	dsn := cfg.DBPath
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite allows only one writer at a time. Limiting to one open connection
	// prevents "database is locked" errors under concurrent request load.
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	// Existing deployments predate per-receipt retention. SQLite's CREATE TABLE
	// IF NOT EXISTS cannot add a column, so make the migration idempotent here.
	if _, err := db.ExecContext(ctx, `ALTER TABLE events ADD COLUMN expires_at TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		db.Close()
		return nil, fmt.Errorf("migrate receipt expiry: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_events_expires ON events(expires_at) WHERE expires_at IS NOT NULL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("index receipt expiry: %w", err)
	}

	s := &SQLiteStore{db: db, rawDB: db, config: cfg}
	if err := s.reconcileMeta(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("reconcile meta: %w", err)
	}
	return s, nil
}

// NewSQLiteStoreWithDB constructs an event store over an already-opened
// database seam. Production uses this with api-core's RoutedDB so requests
// carrying the Test Genie marker are isolated without restarting the API.
func NewSQLiteStoreWithDB(ctx context.Context, db sqlutil.DB, cfg SQLiteConfig) (*SQLiteStore, error) {
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 30 * 24 * time.Hour
	}
	if cfg.MaxSizeBytes == 0 {
		cfg.MaxSizeBytes = 2 * 1024 * 1024 * 1024
	}
	if cfg.QueryLimit <= 0 {
		cfg.QueryLimit = 100
	}
	if cfg.QueryLimitMax <= 0 {
		cfg.QueryLimitMax = 1000
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	s := &SQLiteStore{db: db, config: cfg}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE events ADD COLUMN expires_at TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, fmt.Errorf("migrate receipt expiry: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_events_expires ON events(expires_at) WHERE expires_at IS NOT NULL`); err != nil {
		return nil, fmt.Errorf("index receipt expiry: %w", err)
	}
	if err := s.reconcileMeta(ctx); err != nil {
		return nil, fmt.Errorf("reconcile meta: %w", err)
	}
	return s, nil
}

// reconcileMeta recalculates total_payload_bytes from actual data to correct any drift.
func (s *SQLiteStore) reconcileMeta(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE store_meta SET value = (SELECT COALESCE(SUM(LENGTH(payload)), 0) FROM events) WHERE key = 'total_payload_bytes'`)
	return err
}

func (s *SQLiteStore) Insert(ctx context.Context, e Event) (int64, error) {
	metadataJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		return 0, fmt.Errorf("marshal metadata: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO events (event_id, source_scenario, target_scenario, event_type, correlation_id, payload, metadata, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.EventID, e.SourceScenario, e.TargetScenario, e.EventType, e.CorrelationID, e.Payload, string(metadataJSON), nullableTime(e.ExpiresAt))
	if err != nil {
		if isUniqueConstraintError(err) {
			return 0, fmt.Errorf("%w: %s", ErrDuplicateEvent, e.EventID)
		}
		return 0, fmt.Errorf("insert event: %w", err)
	}

	payloadLen := len(e.Payload)
	if _, err := tx.ExecContext(ctx,
		`UPDATE store_meta SET value = value + ? WHERE key = 'total_payload_bytes'`, payloadLen); err != nil {
		return 0, fmt.Errorf("update meta: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	id, _ := res.LastInsertId()
	return id, nil
}

// isUniqueConstraintError checks if the error is a SQLite UNIQUE constraint violation.
// modernc.org/sqlite returns errors containing "UNIQUE constraint failed" for duplicate keys.
func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *SQLiteStore) Query(ctx context.Context, f QueryFilters) ([]Event, error) {
	var clauses []string
	var args []any

	hasGlob := false
	if f.EventType != "" {
		if strings.Contains(f.EventType, "*") {
			// Use LIKE for broad SQL filtering, then precise match in Go
			clauses = append(clauses, "event_type LIKE ?")
			args = append(args, sqliteLikePattern(f.EventType))
			hasGlob = true
		} else {
			clauses = append(clauses, "event_type = ?")
			args = append(args, f.EventType)
		}
	}
	if f.Source != "" {
		clauses = append(clauses, "source_scenario = ?")
		args = append(args, f.Source)
	}
	if f.Target != "" {
		clauses = append(clauses, "target_scenario = ?")
		args = append(args, f.Target)
	}
	if f.CorrelationID != "" {
		clauses = append(clauses, "correlation_id = ?")
		args = append(args, f.CorrelationID)
	}
	if f.Since > 0 {
		clauses = append(clauses, "id > ?")
		args = append(args, f.Since)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = s.config.QueryLimit
	}
	if limit > s.config.QueryLimitMax {
		limit = s.config.QueryLimitMax
	}

	// SQL LIKE can't express our segment-aware glob semantics (e.g. "*" = one
	// segment only), so we over-fetch from SQLite and post-filter in Go. The 3x
	// multiplier is a heuristic: most event types have 3-4 segments, so 3x the
	// requested limit usually yields enough candidates after precise filtering.
	sqlLimit := limit
	if hasGlob {
		sqlLimit = limit * 3
		if sqlLimit > 3000 {
			sqlLimit = 3000
		}
	}

	query := "SELECT id, event_id, source_scenario, target_scenario, event_type, correlation_id, payload, metadata, created_at, expires_at FROM events"
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY id ASC LIMIT ?"
	args = append(args, sqlLimit)

	events, err := s.scanEvents(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	// Apply precise glob matching in Go
	if hasGlob {
		filtered := events[:0]
		for _, e := range events {
			if match.Glob(f.EventType, e.EventType) && len(filtered) < limit {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	return events, nil
}

func (s *SQLiteStore) DeleteByEventType(ctx context.Context, eventType string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE event_type = ?`, eventType)
	if err != nil {
		return 0, fmt.Errorf("delete events by type: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("event delete count: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) GetSince(ctx context.Context, lastID int64, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.scanEvents(ctx,
		"SELECT id, event_id, source_scenario, target_scenario, event_type, correlation_id, payload, metadata, created_at, expires_at FROM events WHERE id > ? ORDER BY id ASC LIMIT ?",
		lastID, limit)
}

func (s *SQLiteStore) Prune(ctx context.Context) (PruneResult, error) {
	var result PruneResult

	count, err := s.pruneByTime(ctx)
	if err != nil {
		return result, err
	}
	result.TimeDeletedCount = count

	count, err = s.pruneBySize(ctx)
	if err != nil {
		return result, err
	}
	result.SizeDeletedCount = count

	return result, nil
}

// pruneByTime deletes events older than MaxAge and adjusts the payload byte counter.
func (s *SQLiteStore) pruneByTime(ctx context.Context) (int64, error) {
	cutoff := s.config.Now().UTC().Add(-s.config.MaxAge).Format(sqlutil.TimestampFormat)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var timeBytes int64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(LENGTH(payload)), 0) FROM events WHERE created_at < ? OR (expires_at IS NOT NULL AND expires_at <= ?)`, cutoff, s.config.Now().UTC().Format(sqlutil.TimestampFormat)).Scan(&timeBytes); err != nil {
		return 0, fmt.Errorf("sum time bytes: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM events WHERE created_at < ? OR (expires_at IS NOT NULL AND expires_at <= ?)`, cutoff, s.config.Now().UTC().Format(sqlutil.TimestampFormat))
	if err != nil {
		return 0, fmt.Errorf("time prune: %w", err)
	}
	deleted, _ := res.RowsAffected()

	if _, err := tx.ExecContext(ctx,
		`UPDATE store_meta SET value = value - ? WHERE key = 'total_payload_bytes'`, timeBytes); err != nil {
		return 0, fmt.Errorf("update meta after time prune: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit time prune: %w", err)
	}
	return deleted, nil
}

// pruneBySize deletes the oldest events when total payload exceeds MaxSizeBytes.
func (s *SQLiteStore) pruneBySize(ctx context.Context) (int64, error) {
	var totalBytes int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT value FROM store_meta WHERE key = 'total_payload_bytes'`).Scan(&totalBytes); err != nil {
		return 0, fmt.Errorf("read total bytes: %w", err)
	}

	if totalBytes <= s.config.MaxSizeBytes {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin size tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	idsToDelete, deletedBytes, err := s.collectExcessEvents(ctx, tx, totalBytes-s.config.MaxSizeBytes)
	if err != nil {
		return 0, err
	}
	if len(idsToDelete) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(idsToDelete))
	delArgs := make([]any, len(idsToDelete))
	for i, id := range idsToDelete {
		placeholders[i] = "?"
		delArgs[i] = id
	}
	res, err := tx.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM events WHERE id IN (%s)", strings.Join(placeholders, ",")),
		delArgs...)
	if err != nil {
		return 0, fmt.Errorf("size prune delete: %w", err)
	}
	deleted, _ := res.RowsAffected()

	if _, err := tx.ExecContext(ctx,
		`UPDATE store_meta SET value = value - ? WHERE key = 'total_payload_bytes'`, deletedBytes); err != nil {
		return 0, fmt.Errorf("update meta after size prune: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit size prune: %w", err)
	}
	return deleted, nil
}

// collectExcessEvents returns IDs and total bytes of the oldest events whose
// cumulative payload size meets or exceeds the given excess threshold.
func (s *SQLiteStore) collectExcessEvents(ctx context.Context, tx *sql.Tx, excess int64) ([]int64, int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, LENGTH(payload) FROM events ORDER BY id ASC`)
	if err != nil {
		return nil, 0, fmt.Errorf("query for size prune: %w", err)
	}
	defer rows.Close()

	var ids []int64
	var total int64
	for rows.Next() {
		var id, plen int64
		if err := rows.Scan(&id, &plen); err != nil {
			return nil, 0, fmt.Errorf("scan size prune: %w", err)
		}
		ids = append(ids, id)
		total += plen
		if total >= excess {
			break
		}
	}
	return ids, total, rows.Err()
}

func (s *SQLiteStore) Stats(ctx context.Context) (Stats, error) {
	var stats Stats

	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&stats.TotalEvents)
	if err != nil {
		return stats, err
	}

	err = s.db.QueryRowContext(ctx,
		`SELECT value FROM store_meta WHERE key = 'total_payload_bytes'`).Scan(&stats.TotalPayloadBytes)
	if err != nil {
		return stats, err
	}

	var oldest, newest sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT MIN(created_at), MAX(created_at) FROM events`).Scan(&oldest, &newest)
	if err != nil {
		return stats, err
	}
	if oldest.Valid {
		t := sqlutil.ParseTime(oldest.String)
		stats.OldestEvent = &t
	}
	if newest.Valid {
		t := sqlutil.ParseTime(newest.String)
		stats.NewestEvent = &t
	}

	return stats, nil
}

// DB returns the underlying *sql.DB so other packages (e.g. policy) can share
// the same database connection without opening a second handle.
func (s *SQLiteStore) DB() *sql.DB {
	return s.rawDB
}

func (s *SQLiteStore) Close() error {
	if s.rawDB == nil {
		return nil
	}
	return s.rawDB.Close()
}

func (s *SQLiteStore) scanEvents(ctx context.Context, query string, args ...any) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var metaStr sql.NullString
		var createdStr string
		var expiresStr sql.NullString
		var payload []byte

		if err := rows.Scan(&e.ID, &e.EventID, &e.SourceScenario, &e.TargetScenario,
			&e.EventType, &e.CorrelationID, &payload, &metaStr, &createdStr, &expiresStr); err != nil {
			return nil, err
		}

		e.Payload = payload
		if metaStr.Valid && metaStr.String != "" {
			if err := json.Unmarshal([]byte(metaStr.String), &e.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		e.CreatedAt = sqlutil.ParseTime(createdStr)
		if expiresStr.Valid && expiresStr.String != "" {
			expires := sqlutil.ParseTime(expiresStr.String)
			e.ExpiresAt = &expires
		}

		events = append(events, e)
	}
	return events, rows.Err()
}

func nullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format(sqlutil.TimestampFormat)
}

// sqliteLikePattern converts segment-aware globs to SQLite LIKE syntax.
// Our pattern syntax uses . as separator, * for one segment, ** for multiple.
// LIKE: % matches any chars, _ matches one char.
// We convert: * → segment without dots, ** → any sequence including dots.
func sqliteLikePattern(pattern string) string {
	// Escape existing LIKE special chars in non-wildcard segments
	segments := strings.Split(pattern, ".")
	var parts []string
	for i := 0; i < len(segments); i++ {
		seg := segments[i]
		switch seg {
		case "**":
			// ** matches one or more segments — consume consecutive ** as well
			parts = append(parts, "%")
		case "*":
			// * matches exactly one segment (no dots)
			// Use a placeholder that won't contain dots
			// We'll match "any chars except dot" which in LIKE isn't directly possible,
			// so we'll use a post-filter approach instead.
			// For SQL, we approximate: one segment = at least one char, no dots.
			// Since LIKE can't express "no dots", we match broadly and rely on
			// the segment count being right from surrounding literal segments.
			parts = append(parts, "%")
		default:
			// Escape % and _ in literal segments
			seg = strings.ReplaceAll(seg, "%", "\\%")
			seg = strings.ReplaceAll(seg, "_", "\\_")
			parts = append(parts, seg)
		}
	}
	return strings.Join(parts, ".")
}
