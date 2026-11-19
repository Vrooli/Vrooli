package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestEnsureProcessRollupColumnsMigratesExistingStore(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE process_sample_rollups (
		minute DATETIME NOT NULL, owner TEXT NOT NULL, comm TEXT NOT NULL,
		avg_cpu_pct REAL NOT NULL, max_cpu_pct REAL NOT NULL,
		cpu_seconds REAL NOT NULL DEFAULT 0, avg_rss_kb INTEGER NOT NULL,
		max_rss_kb INTEGER NOT NULL, avg_major_faults_per_second REAL NOT NULL DEFAULT 0,
		max_major_faults_per_second REAL NOT NULL DEFAULT 0, sample_count INTEGER NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO process_sample_rollups (minute, owner, comm, avg_cpu_pct, max_cpu_pct, avg_rss_kb, max_rss_kb, sample_count)
		VALUES ('2026-11-15T10:00:00Z', 'legacy', 'worker', 1, 1, 1024, 1024, 3)`); err != nil {
		t.Fatalf("seed legacy rollup: %v", err)
	}
	if err := ensureProcessRollupColumns(context.Background(), db); err != nil {
		t.Fatalf("migrate legacy rollup table: %v", err)
	}

	rows, err := db.Query("PRAGMA table_info(process_sample_rollups)")
	if err != nil {
		t.Fatalf("read migrated columns: %v", err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		columns[name] = true
	}
	for _, name := range []string{"avg_swap_kb", "max_swap_kb", "max_gpu_vram_mb"} {
		if !columns[name] {
			t.Errorf("migration did not add %q", name)
		}
	}
	var measured, unknown, samples int64
	if err := db.QueryRow(`SELECT swap_measured_samples, swap_unknown_samples, sample_count FROM process_sample_rollups WHERE owner='legacy'`).Scan(&measured, &unknown, &samples); err != nil {
		t.Fatalf("read migrated swap coverage: %v", err)
	}
	if measured != 0 || unknown != samples || samples != 3 {
		t.Fatalf("legacy swap coverage measured=%d unknown=%d samples=%d; want 0/3/3", measured, unknown, samples)
	}
}

func TestEnsureProcessSampleColumnsMarksLegacySwapUnknown(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy-process.db"))
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE process_samples (
		ts DATETIME NOT NULL, pid INTEGER NOT NULL, ppid INTEGER NOT NULL, comm TEXT NOT NULL,
		cmdline TEXT, cwd TEXT, owner TEXT NOT NULL, cpu_pct REAL NOT NULL,
		cpu_seconds REAL NOT NULL DEFAULT 0, cpu_seconds_status TEXT NOT NULL DEFAULT 'not_yet_sampled',
		cpu_seconds_reason TEXT NOT NULL DEFAULT '', rss_kb INTEGER NOT NULL, threads INTEGER NOT NULL,
		gpu_vram_mb REAL NOT NULL DEFAULT 0, swap_kb INTEGER NOT NULL DEFAULT 0,
		major_faults_per_second REAL NOT NULL DEFAULT 0, metrics_status TEXT NOT NULL DEFAULT '',
		metrics_reason TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatalf("create legacy process table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO process_samples (ts, pid, ppid, comm, owner, cpu_pct, rss_kb, threads, swap_kb)
		VALUES ('2026-11-15T10:00:00Z', 7, 1, 'worker', 'legacy', 1, 1024, 2, 0)`); err != nil {
		t.Fatalf("seed legacy process sample: %v", err)
	}
	if err := ensureProcessSampleColumns(context.Background(), db); err != nil {
		t.Fatalf("migrate legacy process table: %v", err)
	}
	var status string
	if err := db.QueryRow(`SELECT swap_status FROM process_samples WHERE pid=7`).Scan(&status); err != nil {
		t.Fatalf("read migrated sample: %v", err)
	}
	if status != "unknown" {
		t.Fatalf("legacy swap status = %q, want unknown", status)
	}
}
