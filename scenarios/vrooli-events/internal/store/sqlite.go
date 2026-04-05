package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/match"
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
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
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

const pragmas = `
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=10000;
PRAGMA synchronous=NORMAL;
`

// SQLiteConfig holds configuration for the SQLite store.
type SQLiteConfig struct {
	DBPath       string
	MaxAge       time.Duration // retention period (default 30 days)
	MaxSizeBytes int64         // max total payload size (default 2GB)
}

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db     *sql.DB
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

	dsn := cfg.DBPath
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite single-writer

	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	s := &SQLiteStore{db: db, config: cfg}
	if err := s.reconcileMeta(ctx); err != nil {
		db.Close()
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
		`INSERT INTO events (event_id, source_scenario, target_scenario, event_type, correlation_id, payload, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.EventID, e.SourceScenario, e.TargetScenario, e.EventType, e.CorrelationID, e.Payload, string(metadataJSON))
	if err != nil {
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
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// Fetch more rows than limit when using glob (SQL LIKE is broader than our glob)
	sqlLimit := limit
	if hasGlob {
		sqlLimit = limit * 3
		if sqlLimit > 3000 {
			sqlLimit = 3000
		}
	}

	query := "SELECT id, event_id, source_scenario, target_scenario, event_type, correlation_id, payload, metadata, created_at FROM events"
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

func (s *SQLiteStore) GetSince(ctx context.Context, lastID int64, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.scanEvents(ctx,
		"SELECT id, event_id, source_scenario, target_scenario, event_type, correlation_id, payload, metadata, created_at FROM events WHERE id > ? ORDER BY id ASC LIMIT ?",
		lastID, limit)
}

func (s *SQLiteStore) Prune(ctx context.Context) (PruneResult, error) {
	var result PruneResult

	cutoff := time.Now().UTC().Add(-s.config.MaxAge).Format("2006-01-02T15:04:05.000")

	// Time-based pruning
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Calculate bytes to subtract for time-based deletion
	var timeBytes int64
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(LENGTH(payload)), 0) FROM events WHERE created_at < ?`, cutoff).Scan(&timeBytes)
	if err != nil {
		return result, fmt.Errorf("sum time bytes: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM events WHERE created_at < ?`, cutoff)
	if err != nil {
		return result, fmt.Errorf("time prune: %w", err)
	}
	result.TimeDeletedCount, _ = res.RowsAffected()

	if _, err := tx.ExecContext(ctx,
		`UPDATE store_meta SET value = value - ? WHERE key = 'total_payload_bytes'`, timeBytes); err != nil {
		return result, fmt.Errorf("update meta after time prune: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit time prune: %w", err)
	}

	// Size-based pruning
	var totalBytes int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT value FROM store_meta WHERE key = 'total_payload_bytes'`).Scan(&totalBytes); err != nil {
		return result, fmt.Errorf("read total bytes: %w", err)
	}

	if totalBytes > s.config.MaxSizeBytes {
		tx2, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return result, fmt.Errorf("begin size tx: %w", err)
		}
		defer func() { _ = tx2.Rollback() }()

		excess := totalBytes - s.config.MaxSizeBytes
		// Delete oldest events until we're under the limit
		var deletedBytes int64
		rows, err := tx2.QueryContext(ctx,
			`SELECT id, LENGTH(payload) FROM events ORDER BY id ASC`)
		if err != nil {
			return result, fmt.Errorf("query for size prune: %w", err)
		}

		var idsToDelete []int64
		for rows.Next() {
			var id int64
			var plen int64
			if err := rows.Scan(&id, &plen); err != nil {
				rows.Close()
				return result, fmt.Errorf("scan size prune: %w", err)
			}
			idsToDelete = append(idsToDelete, id)
			deletedBytes += plen
			if deletedBytes >= excess {
				break
			}
		}
		rows.Close()

		if len(idsToDelete) > 0 {
			placeholders := make([]string, len(idsToDelete))
			delArgs := make([]any, len(idsToDelete))
			for i, id := range idsToDelete {
				placeholders[i] = "?"
				delArgs[i] = id
			}
			res, err := tx2.ExecContext(ctx,
				fmt.Sprintf("DELETE FROM events WHERE id IN (%s)", strings.Join(placeholders, ",")),
				delArgs...)
			if err != nil {
				return result, fmt.Errorf("size prune delete: %w", err)
			}
			result.SizeDeletedCount, _ = res.RowsAffected()

			if _, err := tx2.ExecContext(ctx,
				`UPDATE store_meta SET value = value - ? WHERE key = 'total_payload_bytes'`, deletedBytes); err != nil {
				return result, fmt.Errorf("update meta after size prune: %w", err)
			}
		}

		if err := tx2.Commit(); err != nil {
			return result, fmt.Errorf("commit size prune: %w", err)
		}
	}

	return result, nil
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
		t, _ := time.Parse("2006-01-02T15:04:05.000", oldest.String)
		stats.OldestEvent = &t
	}
	if newest.Valid {
		t, _ := time.Parse("2006-01-02T15:04:05.000", newest.String)
		stats.NewestEvent = &t
	}

	return stats, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
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
		var payload []byte

		if err := rows.Scan(&e.ID, &e.EventID, &e.SourceScenario, &e.TargetScenario,
			&e.EventType, &e.CorrelationID, &payload, &metaStr, &createdStr); err != nil {
			return nil, err
		}

		e.Payload = payload
		if metaStr.Valid && metaStr.String != "" {
			if err := json.Unmarshal([]byte(metaStr.String), &e.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		e.CreatedAt, _ = time.Parse("2006-01-02T15:04:05.000", createdStr)

		events = append(events, e)
	}
	return events, rows.Err()
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
