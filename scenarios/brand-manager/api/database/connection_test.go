package database

import (
	"path/filepath"
	"testing"

	"brand-manager/config"
)

// [REQ:BM-REQ-STORE-INIT] [REQ:BM-REQ-STORE-SCHEMA] Verify database connection and schema initialization.

func TestConnect_CreatesDB(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.SQLitePath = filepath.Join(tmpDir, "test.db")

	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer db.Close()

	// Verify tables exist by querying sqlite_master
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tables = append(tables, name)
	}

	expected := map[string]bool{
		"brands":         true,
		"brand_versions": true,
		"assignments":    true,
		"assets":         true,
	}
	for _, tbl := range tables {
		delete(expected, tbl)
	}
	if len(expected) > 0 {
		t.Errorf("missing tables: %v", expected)
	}
}

func TestConnect_IdempotentSchema(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.SQLitePath = filepath.Join(tmpDir, "test.db")

	// First connection creates schema
	db1, err := Connect(cfg)
	if err != nil {
		t.Fatalf("first Connect() error: %v", err)
	}
	db1.Close()

	// Second connection should not fail (IF NOT EXISTS)
	db2, err := Connect(cfg)
	if err != nil {
		t.Fatalf("second Connect() error: %v", err)
	}
	db2.Close()
}

func TestConnect_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.SQLitePath = filepath.Join(tmpDir, "nested", "dir", "test.db")

	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() with nested dir error: %v", err)
	}
	db.Close()
}

// [REQ:BM-REQ-STORE-SCHEMA] Verify schema column structure for core tables.
func TestConnect_SchemaColumns(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.SQLitePath = filepath.Join(tmpDir, "test.db")

	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer db.Close()

	tests := []struct {
		table   string
		columns []string
	}{
		{"brands", []string{"id", "name", "version", "created_at", "updated_at"}},
		{"brand_versions", []string{"id", "brand_id", "version", "snapshot", "created_at"}},
		{"assignments", []string{"id", "brand_id", "scenario_name", "brand_version", "applied_at"}},
		{"assets", []string{"id", "brand_id", "filename", "mime_type", "file_path", "size", "created_at"}},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			rows, err := db.Query("PRAGMA table_info(" + tt.table + ")")
			if err != nil {
				t.Fatalf("PRAGMA table_info(%s): %v", tt.table, err)
			}
			defer rows.Close()

			cols := make(map[string]bool)
			for rows.Next() {
				var cid int
				var name, typ string
				var notnull int
				var dflt *string
				var pk int
				if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
					t.Fatalf("scan: %v", err)
				}
				cols[name] = true
			}

			for _, expected := range tt.columns {
				if !cols[expected] {
					t.Errorf("table %s: missing column %q", tt.table, expected)
				}
			}
		})
	}
}

// [REQ:BM-REQ-STORE-INIT] Verify WAL mode is enabled after connection.
func TestConnect_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.SQLitePath = filepath.Join(tmpDir, "wal-test.db")

	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}
}
