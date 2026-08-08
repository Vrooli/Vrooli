package eventlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

// Repository defines the append-only event log storage interface.
type Repository interface {
	// Append inserts a new event and returns its auto-generated ID.
	Append(ctx context.Context, e Event) (int64, error)
	// Since returns events with ID > afterID, ordered by ID, up to limit.
	Since(ctx context.Context, afterID int64, limit int) ([]Event, error)
	// All returns every event ordered by ID.
	All(ctx context.Context) ([]Event, error)
	// QueryByEntity returns events for one entity after afterID, ordered by ID.
	// Entity timelines use this instead of scanning unrelated event history.
	QueryByEntity(ctx context.Context, entityType EntityType, entityID string, afterID int64, limit int) ([]Event, error)
	// MaxID returns the highest event ID, or 0 if empty.
	MaxID(ctx context.Context) (int64, error)
	// QueryByTypesSince returns events of the given types at or after since,
	// ordered by ID. Analytical readers use this instead of All so the filter
	// runs in SQL and their cost does not grow with unrelated event history.
	QueryByTypesSince(ctx context.Context, types []EventType, since time.Time) ([]Event, error)
}

// SQLiteRepository implements Repository backed by a SQLite database.
//
// It holds a *database.RoutedDB rather than a raw *sql.DB so that, under a
// test-genie in-place e2e run, every query routes to the installed test pool
// instead of the live event log. RoutedDB's method surface mirrors *sql.DB, so
// the query bodies below are unchanged.
type SQLiteRepository struct {
	db *database.RoutedDB
}

// NewSQLiteRepository creates a repository using the given routed database
// handle. Callers outside the lifecycle (tests) wrap a raw *sql.DB with
// database.NewFromPrimary.
func NewSQLiteRepository(db *database.RoutedDB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// schemaSQL is the declarative event-log schema. It is the single source of
// truth applied both at boot (via database.EnsureSchemas) and by InitSchema.
const schemaSQL = `
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			actor_type TEXT NOT NULL DEFAULT 'user',
			actor_id TEXT NOT NULL DEFAULT '',
			run_id TEXT NOT NULL DEFAULT '',
			verification_status TEXT NOT NULL DEFAULT 'absent',
			harness_session_id TEXT NOT NULL DEFAULT '',
			harness_kind TEXT NOT NULL DEFAULT '',
			metadata TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_events_entity ON events(entity_type, entity_id, id);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
		CREATE INDEX IF NOT EXISTS idx_events_actor ON events(actor_id, id);
		CREATE TABLE IF NOT EXISTS evidence_observations (
			id TEXT PRIMARY KEY,
			producer TEXT NOT NULL,
			source_system TEXT NOT NULL,
			run_id TEXT NOT NULL,
			subject_kind TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			action TEXT NOT NULL,
			confidence TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			actor TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			observed_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS evidence_links (
			observation_id TEXT NOT NULL,
			attempt_ref TEXT NOT NULL,
			PRIMARY KEY (observation_id, attempt_ref)
		);
		CREATE TABLE IF NOT EXISTS evidence_watermarks (
			producer TEXT NOT NULL,
			run_id TEXT NOT NULL,
			fact_kind TEXT NOT NULL,
			terminal_at TEXT NOT NULL,
			PRIMARY KEY (producer, run_id, fact_kind)
		);
		CREATE TABLE IF NOT EXISTS evidence_checkpoints (
			producer TEXT NOT NULL,
			run_id TEXT NOT NULL,
			fact_kind TEXT NOT NULL,
			checkpoint_at TEXT NOT NULL,
			PRIMARY KEY (producer, run_id, fact_kind)
		);
		CREATE TABLE IF NOT EXISTS evidence_migration_audits (
			migration_key TEXT PRIMARY KEY,
			source_digest TEXT NOT NULL,
			projection_digest TEXT NOT NULL,
			source_count INTEGER NOT NULL,
			projection_count INTEGER NOT NULL,
			parity_proven INTEGER NOT NULL DEFAULT 0,
			audited_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_evidence_subject ON evidence_observations(subject_kind, subject_id);
	`

// Schema returns the declarative event-log schema for database.EnsureSchemas.
func Schema() string { return schemaSQL }

// InitSchema creates the events table and indexes if they don't exist.
func (r *SQLiteRepository) InitSchema(ctx context.Context) error {
	if err := r.migrateLegacyEvidenceSchema(ctx); err != nil {
		return fmt.Errorf("migrate legacy evidence schema: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}
	// Existing event databases predate the operator-verification columns. SQLite
	// lacks ADD COLUMN IF NOT EXISTS, so duplicate-column errors are benign.
	for _, statement := range []string{
		"ALTER TABLE events ADD COLUMN verification_status TEXT NOT NULL DEFAULT 'absent'",
		"ALTER TABLE events ADD COLUMN run_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE events ADD COLUMN harness_session_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE events ADD COLUMN harness_kind TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE evidence_observations ADD COLUMN actor TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE evidence_observations ADD COLUMN reason TEXT NOT NULL DEFAULT ''",
	} {
		if _, err := r.db.ExecContext(ctx, statement); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
	}
	return nil
}

// migrateLegacyEvidenceSchema upgrades the short-lived, pre-ledger evidence
// tables. That schema used integer observations and owner links, while the
// ledger requires deterministic string observation IDs and attempt links. The
// old tables are retained under explicit legacy names after their contents are
// copied, so an upgrade never silently discards operator evidence.
func (r *SQLiteRepository) migrateLegacyEvidenceSchema(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "PRAGMA table_info(evidence_observations)")
	if err != nil {
		return err
	}
	defer rows.Close()
	hasTable, hasProducer := false, false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		hasTable = true
		if name == "producer" {
			hasProducer = true
		}
	}
	if err := rows.Err(); err != nil || !hasTable {
		return err
	}
	if hasProducer {
		return r.migrateLegacyEvidenceProgressSchema(ctx)
	}
	for _, statement := range []string{
		"ALTER TABLE evidence_observations RENAME TO legacy_evidence_observations_v0",
		"ALTER TABLE evidence_links RENAME TO legacy_evidence_links_v0",
		"ALTER TABLE evidence_migration_audits RENAME TO legacy_evidence_migration_audits_v0",
		"ALTER TABLE evidence_checkpoints RENAME TO legacy_evidence_checkpoints_v0",
		"ALTER TABLE evidence_watermarks RENAME TO legacy_evidence_watermarks_v0",
	} {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := r.db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO evidence_observations (id, producer, source_system, run_id, subject_kind, subject_id, action, confidence, title, description, actor, reason, observed_at)
		SELECT 'legacy/' || id, 'legacy:' || source_system, source_system, run_id, subject_kind, subject_id, action, confidence, 'Legacy evidence ' || source_event_id, metadata_json, actor, reason, observed_at
		FROM legacy_evidence_observations_v0`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO evidence_links (observation_id, attempt_ref)
		SELECT 'legacy/' || observation_id, 'legacy/' || owner_kind || '/' || owner_id || '/' || owner_round
		FROM legacy_evidence_links_v0`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO evidence_migration_audits (migration_key, source_digest, projection_digest, source_count, projection_count, parity_proven, audited_at)
		SELECT migration_key, source_digest, projected_digest, source_count, projected_count, source_count = projected_count, completed_at
		FROM legacy_evidence_migration_audits_v0`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO evidence_checkpoints (producer, run_id, fact_kind, checkpoint_at)
		SELECT producer_id, run_id, fact_kind, updated_at FROM legacy_evidence_checkpoints_v0`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO evidence_watermarks (producer, run_id, fact_kind, terminal_at)
		SELECT producer_id, run_id, fact_kind, completed_at FROM legacy_evidence_watermarks_v0`); err != nil {
		return err
	}
	return nil
}

// migrateLegacyEvidenceProgressSchema covers installations interrupted between
// the observation-table upgrade and the checkpoint/watermark upgrade.
func (r *SQLiteRepository) migrateLegacyEvidenceProgressSchema(ctx context.Context) error {
	legacy, err := r.tableLacksColumn(ctx, "evidence_checkpoints", "producer")
	if err != nil || !legacy {
		return err
	}
	for _, statement := range []string{
		"ALTER TABLE evidence_checkpoints RENAME TO legacy_evidence_checkpoints_v0",
		"ALTER TABLE evidence_watermarks RENAME TO legacy_evidence_watermarks_v0",
	} {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := r.db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO evidence_checkpoints (producer, run_id, fact_kind, checkpoint_at)
		SELECT producer_id, run_id, fact_kind, updated_at FROM legacy_evidence_checkpoints_v0`); err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO evidence_watermarks (producer, run_id, fact_kind, terminal_at)
		SELECT producer_id, run_id, fact_kind, completed_at FROM legacy_evidence_watermarks_v0`)
	return err
}

func (r *SQLiteRepository) tableLacksColumn(ctx context.Context, table, column string) (bool, error) {
	rows, err := r.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	hasTable, hasColumn := false, false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, err
		}
		hasTable = true
		hasColumn = hasColumn || name == column
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return hasTable && !hasColumn, nil
}

// Append inserts a new event and returns its auto-generated ID.
func (r *SQLiteRepository) Append(ctx context.Context, e Event) (int64, error) {
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	var metaStr *string
	if len(e.Metadata) > 0 {
		s := string(e.Metadata)
		metaStr = &s
	}

	actorType := e.ActorType
	if actorType == "" {
		actorType = "user"
	}

	result, err := r.db.ExecContext(ctx,
		`INSERT INTO events (timestamp, entity_type, entity_id, event_type, actor_type, actor_id, run_id, verification_status, harness_session_id, harness_kind, metadata)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ts.Format(time.RFC3339Nano),
		string(e.EntityType),
		e.EntityID,
		string(e.EventType),
		actorType,
		e.ActorID,
		e.RunID,
		e.VerificationStatus,
		e.HarnessSessionID,
		e.HarnessKind,
		metaStr,
	)
	if err != nil {
		return 0, fmt.Errorf("eventlog append: %w", err)
	}
	return result.LastInsertId()
}

// Since returns events with ID > afterID, ordered by ID ascending, up to limit.
func (r *SQLiteRepository) Since(ctx context.Context, afterID int64, limit int) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, run_id, verification_status, harness_session_id, harness_kind, metadata
		 FROM events WHERE id > ? ORDER BY id ASC LIMIT ?`,
		afterID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("eventlog since: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// All returns every event ordered by ID ascending.
func (r *SQLiteRepository) All(ctx context.Context) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, run_id, verification_status, harness_session_id, harness_kind, metadata
		 FROM events ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("eventlog all: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// QueryByTypesSince pushes the read filter into SQL. An empty types slice
// matches nothing rather than everything: a caller that forgot to name its
// event types should read no evidence, not the entire log.
func (r *SQLiteRepository) QueryByTypesSince(ctx context.Context, types []EventType, since time.Time) ([]Event, error) {
	if len(types) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(types)+1)
	placeholders := make([]string, 0, len(types))
	for _, eventType := range types {
		placeholders = append(placeholders, "?")
		args = append(args, string(eventType))
	}
	query := `SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, run_id, verification_status, harness_session_id, harness_kind, metadata
		 FROM events WHERE event_type IN (` + strings.Join(placeholders, ",") + `)`
	if !since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, since.UTC().Format(time.RFC3339Nano))
	}
	query += " ORDER BY id ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("eventlog query by types: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// QueryByEntity returns a bounded, chronological slice of one entity's
// append-only history. Callers supply a cursor rather than relying on wall
// clock timestamps, which keeps concurrent writes deterministic.
func (r *SQLiteRepository) QueryByEntity(ctx context.Context, entityType EntityType, entityID string, afterID int64, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, run_id, verification_status, harness_session_id, harness_kind, metadata
		 FROM events WHERE entity_type = ? AND entity_id = ? AND id > ? ORDER BY id ASC LIMIT ?`,
		string(entityType), entityID, afterID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("eventlog query entity: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// MaxID returns the highest event ID, or 0 if the table is empty.
func (r *SQLiteRepository) MaxID(ctx context.Context) (int64, error) {
	var maxID int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM events`).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("eventlog maxid: %w", err)
	}
	return maxID, nil
}

// CountEventsInRange returns how many events of eventType have a Timestamp in
// the half-open range [from, to) — from inclusive, to exclusive, matching the
// packages/measures-go time-window resolver. It is the substrate the granular
// measures (api/handlers/measures) compute against.
//
// The event_type filter is pushed to SQL (the idx_events_type index), but the
// time-window comparison is applied in Go on parsed time.Time values rather
// than as a SQL string range: timestamps are stored as RFC3339Nano, whose
// variable-width fractional seconds make lexical comparison unsafe at
// sub-second boundaries. Scanning one event-type's rows is bounded and exact.
func (r *SQLiteRepository) CountEventsInRange(ctx context.Context, eventType EventType, from, to time.Time) (int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT timestamp FROM events WHERE event_type = ?`,
		string(eventType),
	)
	if err != nil {
		return 0, fmt.Errorf("eventlog count %s: %w", eventType, err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var tsStr string
		if err := rows.Scan(&tsStr); err != nil {
			return 0, fmt.Errorf("eventlog count scan: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return 0, fmt.Errorf("eventlog count parse timestamp %q: %w", tsStr, err)
		}
		if inRange(t, from, to) {
			count++
		}
	}
	return count, rows.Err()
}

// CountStatusTransitionsInRange counts events of eventType for which the typed
// StatusChangePayload's `to` equals toStatus and whose Timestamp is in
// [from, to). It backs status-transition measures such as
// "backlog items completed this week" (backlog.status_changed → to=completed),
// which a bare event-type count cannot express. Malformed metadata is skipped
// (never counted), never an error — the measure abstains rather than over-counts.
func (r *SQLiteRepository) CountStatusTransitionsInRange(ctx context.Context, eventType EventType, toStatus string, from, to time.Time) (int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT timestamp, metadata FROM events WHERE event_type = ?`,
		string(eventType),
	)
	if err != nil {
		return 0, fmt.Errorf("eventlog count transitions %s: %w", eventType, err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var tsStr string
		var metaStr sql.NullString
		if err := rows.Scan(&tsStr, &metaStr); err != nil {
			return 0, fmt.Errorf("eventlog count transitions scan: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return 0, fmt.Errorf("eventlog count transitions parse timestamp %q: %w", tsStr, err)
		}
		if !inRange(t, from, to) || !metaStr.Valid {
			continue
		}
		var p StatusChangePayload
		if err := json.Unmarshal([]byte(metaStr.String), &p); err != nil {
			continue // malformed payload — abstain, do not over-count
		}
		if p.To == toStatus {
			count++
		}
	}
	return count, rows.Err()
}

// inRange reports whether t is in the half-open range [from, to).
func inRange(t, from, to time.Time) bool {
	return !t.Before(from) && t.Before(to)
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		var tsStr string
		var metaStr sql.NullString
		err := rows.Scan(
			&e.ID,
			&tsStr,
			&e.EntityType,
			&e.EntityID,
			&e.EventType,
			&e.ActorType,
			&e.ActorID,
			&e.RunID,
			&e.VerificationStatus,
			&e.HarnessSessionID,
			&e.HarnessKind,
			&metaStr,
		)
		if err != nil {
			return nil, fmt.Errorf("eventlog scan: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return nil, fmt.Errorf("eventlog parse timestamp %q: %w", tsStr, err)
		}
		e.Timestamp = t
		if metaStr.Valid {
			e.Metadata = json.RawMessage(metaStr.String)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
