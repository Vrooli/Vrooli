package integrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
)

type SQLiteRepository struct {
	db    *database.RoutedDB
	clock schedule.Clock
}

func NewSQLiteRepository(db *database.RoutedDB, clock schedule.Clock) *SQLiteRepository {
	return &SQLiteRepository{db: db, clock: clock}
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Connection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,provider,display_name,source_kind,status,health_message,read_only,calendar_count,imported_event_count,busy_minutes,revision,last_sync_at FROM provider_connections ORDER BY display_name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Connection
	for rows.Next() {
		item, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CreateFixture(ctx context.Context, name string) (Connection, error) {
	id := fmt.Sprintf("fixture-%d", r.clock.Now().UnixNano())
	_, err := r.db.ExecContext(ctx, `INSERT INTO provider_connections (id,provider,display_name,source_kind,status,health_message,read_only,revision) VALUES (?,?,?,?,?,?,?,?)`, id, "fixture", name, "fixture", "connected", "Synthetic read-only adapter; no provider account is connected.", 1, 1)
	if err != nil {
		return Connection{}, err
	}
	return r.get(ctx, id)
}

func (r *SQLiteRepository) Sync(ctx context.Context, in SyncInput) (Connection, error) {
	now := r.clock.Now().UTC()
	result, err := r.db.ExecContext(ctx, `UPDATE provider_connections SET status='synced', health_message='Fixture data refreshed; live provider credentials are not configured.', calendar_count=1, imported_event_count=3, busy_minutes=120, revision=revision+1, last_sync_at=? WHERE id=? AND revision=?`, now.Format(time.RFC3339Nano), in.ID, in.ExpectedRevision)
	if err != nil {
		return Connection{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return Connection{}, r.conflictOrNotFound(ctx, in.ID)
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM imported_events WHERE connection_id=?`, in.ID); err != nil {
		return Connection{}, err
	}
	day := now.Format("2006-01-02")
	for i, event := range []struct {
		date            string
		start, duration int
		title           string
	}{{day, 600, 60, "Sample team sync"}, {day, 720, 30, "Sample appointment"}, {now.AddDate(0, 0, 1).Format("2006-01-02"), 900, 30, "Sample planning block"}} {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO imported_events (id,connection_id,remote_event_id,local_date,start_minutes,duration_minutes,title,busy,status,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, fmt.Sprintf("%s-event-%d", in.ID, i+1), in.ID, fmt.Sprintf("sample-%d", i+1), event.date, event.start, event.duration, event.title, 1, "active", now.Format(time.RFC3339Nano)); err != nil {
			return Connection{}, err
		}
	}
	return r.get(ctx, in.ID)
}

func (r *SQLiteRepository) Disconnect(ctx context.Context, in SyncInput) (Connection, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE provider_connections SET status='disconnected', health_message='Disconnected; imported facts are no longer included in capacity.', revision=revision+1 WHERE id=? AND revision=?`, in.ID, in.ExpectedRevision)
	if err != nil {
		return Connection{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return Connection{}, r.conflictOrNotFound(ctx, in.ID)
	}
	return r.get(ctx, in.ID)
}

func (r *SQLiteRepository) get(ctx context.Context, id string) (Connection, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,provider,display_name,source_kind,status,health_message,read_only,calendar_count,imported_event_count,busy_minutes,revision,last_sync_at FROM provider_connections WHERE id=?`, id)
	item, err := scanConnection(row)
	if err == sql.ErrNoRows {
		return Connection{}, ErrNotFound{id}
	}
	return item, err
}

func (r *SQLiteRepository) conflictOrNotFound(ctx context.Context, id string) error {
	if _, err := r.get(ctx, id); err != nil {
		return err
	}
	return ErrConflict{id}
}

type scanner interface{ Scan(...any) error }

func scanConnection(row scanner) (Connection, error) {
	var c Connection
	var readOnly int
	var last string
	if err := row.Scan(&c.ID, &c.Provider, &c.DisplayName, &c.SourceKind, &c.Status, &c.HealthMessage, &readOnly, &c.CalendarCount, &c.ImportedEventCount, &c.BusyMinutes, &c.Revision, &last); err != nil {
		return Connection{}, err
	}
	c.ReadOnly = readOnly == 1
	if strings.TrimSpace(last) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, last)
		if err != nil {
			return Connection{}, err
		}
		c.LastSyncAt = parsed
	}
	return c, nil
}

func Schema() string { return schema }

const schema = `CREATE TABLE IF NOT EXISTS provider_connections (id TEXT PRIMARY KEY, provider TEXT NOT NULL, display_name TEXT NOT NULL, source_kind TEXT NOT NULL, status TEXT NOT NULL, health_message TEXT NOT NULL DEFAULT '', read_only INTEGER NOT NULL DEFAULT 1, calendar_count INTEGER NOT NULL DEFAULT 0, imported_event_count INTEGER NOT NULL DEFAULT 0, busy_minutes INTEGER NOT NULL DEFAULT 0, revision INTEGER NOT NULL DEFAULT 1, last_sync_at TEXT NOT NULL DEFAULT ''); CREATE INDEX IF NOT EXISTS idx_provider_connections_status ON provider_connections(status); CREATE TABLE IF NOT EXISTS imported_events (id TEXT PRIMARY KEY, connection_id TEXT NOT NULL, remote_event_id TEXT NOT NULL, local_date TEXT NOT NULL, start_minutes INTEGER NOT NULL, duration_minutes INTEGER NOT NULL, title TEXT NOT NULL, busy INTEGER NOT NULL DEFAULT 1, status TEXT NOT NULL DEFAULT 'active', updated_at TEXT NOT NULL, UNIQUE(connection_id, remote_event_id)); CREATE INDEX IF NOT EXISTS idx_imported_events_date ON imported_events(local_date, status, busy);`
