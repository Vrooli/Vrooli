package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" database/sql driver

	"github.com/vrooli/api-core/database"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/modules"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

const schema = `
CREATE TABLE IF NOT EXISTS metrics (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	cycle_id TEXT NOT NULL,
	collector_name TEXT NOT NULL,
	metric_data TEXT NOT NULL,
	observed_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_metrics_observed_at ON metrics(observed_at);
-- (collector_name, observed_at) serves "latest row for one collector" as a
-- single index seek. The old single-column collector index forced a temp
-- B-tree sort over every row of that collector on each read (the 2026-09-09
-- idle-CPU burn), and it is a strict prefix of this one, so it is dropped.
CREATE INDEX IF NOT EXISTS idx_metrics_collector_observed_at ON metrics(collector_name, observed_at);
DROP INDEX IF EXISTS idx_metrics_collector;
CREATE INDEX IF NOT EXISTS idx_metrics_cycle ON metrics(cycle_id, observed_at);

CREATE TABLE IF NOT EXISTS investigations (
	id TEXT PRIMARY KEY,
	status TEXT NOT NULL,
	anomaly_id TEXT,
	start_time DATETIME NOT NULL,
	end_time DATETIME,
	findings TEXT,
	progress INTEGER DEFAULT 0,
	details TEXT,
	steps TEXT
);
CREATE INDEX IF NOT EXISTS idx_investigations_status ON investigations(status);

CREATE TABLE IF NOT EXISTS reports (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	generated_at DATETIME NOT NULL,
	time_range_start DATETIME,
	time_range_end DATETIME,
	time_range_duration TEXT,
	data TEXT,
	format TEXT
);

CREATE TABLE IF NOT EXISTS enhanced_reports (
	report_id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	generated_at DATETIME NOT NULL,
	report_data TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	severity TEXT NOT NULL,
	message TEXT NOT NULL,
	metric_name TEXT,
	metric_value REAL,
	threshold TEXT,
	details TEXT,
	timestamp DATETIME NOT NULL,
	acked_at DATETIME,
	resolved_at DATETIME,
	acked_by TEXT
);
CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved_at);

CREATE TABLE IF NOT EXISTS anomalies (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	severity TEXT NOT NULL,
	description TEXT,
	metric_data TEXT,
	detected_at DATETIME NOT NULL,
	resolved_at DATETIME,
	status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS threshold_violations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	metric_name TEXT NOT NULL,
	current_value REAL,
	threshold_value REAL,
	severity TEXT,
	violation_type TEXT,
	timestamp DATETIME NOT NULL,
	duration TEXT,
	previous_value REAL,
	trend TEXT
);
CREATE INDEX IF NOT EXISTS idx_violations_timestamp ON threshold_violations(timestamp);

-- Per-process samples (additive to the opaque metrics blob): one row per
-- process per sampling cycle, the substrate for the attribution timeline.
CREATE TABLE IF NOT EXISTS process_samples (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts DATETIME NOT NULL,
	pid INTEGER NOT NULL,
	ppid INTEGER NOT NULL,
	comm TEXT NOT NULL,
	cmdline TEXT,
	cwd TEXT,
	owner TEXT NOT NULL,
	cpu_pct REAL NOT NULL,
	cpu_seconds REAL NOT NULL DEFAULT 0,
	cpu_seconds_status TEXT NOT NULL DEFAULT 'not_yet_sampled',
	cpu_seconds_reason TEXT NOT NULL DEFAULT '',
	rss_kb INTEGER NOT NULL,
	threads INTEGER NOT NULL,
	gpu_vram_mb REAL NOT NULL DEFAULT 0,
	swap_kb INTEGER NOT NULL DEFAULT 0,
	major_faults_per_second REAL NOT NULL DEFAULT 0,
	metrics_status TEXT NOT NULL DEFAULT '',
	metrics_reason TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_process_samples_ts ON process_samples(ts);
CREATE INDEX IF NOT EXISTS idx_process_samples_owner_ts ON process_samples(owner, ts);

-- Per-owner / per-minute rollups: raw rows older than the raw-retention window
-- are downsampled here so longer windows stay cheap to query and store.
CREATE TABLE IF NOT EXISTS process_sample_rollups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	minute DATETIME NOT NULL,
	owner TEXT NOT NULL,
	comm TEXT NOT NULL,
	avg_cpu_pct REAL NOT NULL,
	max_cpu_pct REAL NOT NULL,
	cpu_seconds REAL NOT NULL DEFAULT 0,
	avg_rss_kb INTEGER NOT NULL,
	max_rss_kb INTEGER NOT NULL,
	avg_major_faults_per_second REAL NOT NULL DEFAULT 0,
	max_major_faults_per_second REAL NOT NULL DEFAULT 0,
	sample_count INTEGER NOT NULL,
	UNIQUE(minute, owner, comm)
);
CREATE INDEX IF NOT EXISTS idx_process_rollups_minute ON process_sample_rollups(minute);
CREATE INDEX IF NOT EXISTS idx_process_rollups_owner_minute ON process_sample_rollups(owner, minute);
`

// Schema returns the SQLite DDL owned by the system-monitor repository.
func Schema() string {
	return schema
}

// Repository implements repository.Repository backed by SQLite.
type Repository struct {
	db   *database.RoutedDB
	path string
	// compacting is set while Compact holds the write lock for a full VACUUM;
	// latest/timeline reads fail fast with ErrMaintenance instead of queueing
	// behind it, so health checks keep answering from in-memory state.
	compacting atomic.Bool
	mu         sync.RWMutex // Serialize SQLite writes
	thMu       sync.RWMutex
	th         map[string]*models.Threshold
}

// NewRepository opens a SQLite database at dbPath and initializes the schema.
func NewRepository(dbPath string) (*Repository, error) {
	// Open via api-core/database so the connection gets retry-with-backoff and
	// jitter (avoids thundering-herd on contended SQLite) instead of a bare
	// sql.Open. MaxOpenConns=1 preserves the single-writer SQLite discipline.
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dbPath,
		MaxOpenConns: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite pragmas for performance and correctness.
	primary := db.Primary()
	// auto_vacuum=INCREMENTAL must precede table creation to take effect on a
	// new database; an existing NONE-mode database adopts it on its next full
	// VACUUM (maintenance Compact). Retention then releases freed pages via
	// ReleaseFreePages instead of leaving the file at its high-water mark.
	for _, pragma := range []string{
		"PRAGMA auto_vacuum=INCREMENTAL",
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := primary.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %s: %w", pragma, err)
		}
	}

	providers := []database.SchemaProvider{database.SchemaProviderFunc(Schema)}
	providers = append(providers, modules.AllSchemas()...)
	if err := database.EnsureSchemas(context.Background(), primary, providers...); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	repo := NewRepositoryFromDB(db)
	repo.path = dbPath
	return repo, nil
}

// NewRepositoryFromDB wraps an already-open routed database.
func NewRepositoryFromDB(db *database.RoutedDB) *Repository {
	return &Repository{
		db: db,
		th: make(map[string]*models.Threshold),
	}
}

// RoutedDB returns the routed database used by this repository.
func (r *Repository) RoutedDB() *database.RoutedDB {
	return r.db
}

// NewInMemoryRepository creates a SQLite repository using an in-memory database.
// Uses shared-cache mode so all connections see the same data.
func NewInMemoryRepository() (*Repository, error) {
	return NewRepository("file::memory:?cache=shared")
}

// Close closes the underlying database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// ---------------------------------------------------------------------------
// MetricsRepository
// ---------------------------------------------------------------------------

func parseTime(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999 +0000 UTC",
		"2006-01-02 15:04:05.999999999+00:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable time: %s", s)
}

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func (r *Repository) scanInvestigation(row *sql.Row) (*models.Investigation, error) {
	var inv models.Investigation
	var endTime sql.NullTime
	var details, steps string

	err := row.Scan(&inv.ID, &inv.Status, &inv.AnomalyID, &inv.StartTime, &endTime,
		&inv.Findings, &inv.Progress, &details, &steps,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("investigation: %w", repository.ErrNotFound)
		}
		return nil, err
	}

	if endTime.Valid {
		inv.EndTime = &endTime.Time
	}
	json.Unmarshal([]byte(details), &inv.Details) //nolint:errcheck // malformed optional details fall back to zero values
	json.Unmarshal([]byte(steps), &inv.Steps)     //nolint:errcheck // malformed optional steps fall back to zero values

	return &inv, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func (r *Repository) scanInvestigationRow(row rowScanner) (*models.Investigation, error) {
	var inv models.Investigation
	var endTime sql.NullTime
	var details, steps string

	err := row.Scan(&inv.ID, &inv.Status, &inv.AnomalyID, &inv.StartTime, &endTime,
		&inv.Findings, &inv.Progress, &details, &steps,
	)
	if err != nil {
		return nil, err
	}

	if endTime.Valid {
		inv.EndTime = &endTime.Time
	}
	json.Unmarshal([]byte(details), &inv.Details) //nolint:errcheck // malformed optional details fall back to zero values
	json.Unmarshal([]byte(steps), &inv.Steps)     //nolint:errcheck // malformed optional steps fall back to zero values

	return &inv, nil
}

func (r *Repository) scanAlertRow(row rowScanner) (*models.Alert, error) {
	var alert models.Alert
	var threshold, details string
	var ackedAt, resolvedAt sql.NullTime

	err := row.Scan(&alert.ID, &alert.Type, &alert.Severity, &alert.Message,
		&alert.MetricName, &alert.MetricValue, &threshold, &details,
		&alert.Timestamp, &ackedAt, &resolvedAt, &alert.AckedBy,
	)
	if err != nil {
		return nil, err
	}

	if threshold != "" && threshold != "null" {
		var th models.Threshold
		if json.Unmarshal([]byte(threshold), &th) == nil {
			alert.Threshold = &th
		}
	}
	json.Unmarshal([]byte(details), &alert.Details) //nolint:errcheck // malformed optional alert details fall back to an empty map

	if ackedAt.Valid {
		alert.AckedAt = &ackedAt.Time
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}

	return &alert, nil
}
