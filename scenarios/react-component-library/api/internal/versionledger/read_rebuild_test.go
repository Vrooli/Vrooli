package versionledger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/databasetest"
)

func TestListWindowFiltersByLibraryAndRecentCreationTime(t *testing.T) {
	db := databasetest.NewSQLite(t)
	_, err := db.Exec(Schema() + `
CREATE TABLE component_versions (library_id TEXT, version TEXT, presence TEXT);
INSERT INTO version_ledger (library_id, version, created_at, lifecycle_state) VALUES
 ('lib-a', '1.0.0', '2026-11-19T00:00:00Z', 'released'),
 ('lib-a', '1.1.0', '2026-10-01T00:00:00Z', 'archived'),
 ('lib-b', '1.0.0', '2026-11-20T00:00:00Z', 'released');
INSERT INTO component_versions (library_id, version, presence) VALUES ('lib-a', '1.0.0', 'evicted');`)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db, t.TempDir())

	recent, err := repo.ListWindow(context.Background(), "lib-a", "last_7d")
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 1 || recent[0].LibraryID != "lib-a" || recent[0].Version != "1.0.0" || recent[0].Presence != "evicted" {
		t.Fatalf("recent library window = %#v, want only evicted lib-a 1.0.0", recent)
	}
	all, err := repo.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].LibraryID != "lib-a" || all[2].LibraryID != "lib-b" {
		t.Fatalf("unfiltered ledger list = %#v, want stable library/version order", all)
	}
}

func TestRebuildProjectsVersionFactsAndPreservesRetirement(t *testing.T) {
	db := databasetest.NewSQLite(t)
	_, err := db.Exec(Schema() + `
CREATE TABLE component_versions (
 id TEXT, component_id TEXT, library_id TEXT, version TEXT, status TEXT,
 created_at TEXT, released_at TEXT, content TEXT, presence TEXT DEFAULT 'materialized'
);
CREATE TABLE catalog_gate_evidence (asset_id TEXT, version TEXT, result TEXT);
CREATE TABLE component_version_test_rollup (library_id TEXT, version TEXT, runs_total INTEGER, runs_passed INTEGER);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT);
CREATE TABLE component_asset_dependencies (component_id TEXT, version TEXT);
CREATE TABLE component_version_files (version_id TEXT, path TEXT);
INSERT INTO component_versions VALUES
 ('v1', 'cmp-a', 'lib-a', '1.0.0', 'released', '2026-11-18T00:00:00Z', '2026-11-19T00:00:00Z', 'one' || char(10) || 'two' || char(10), 'materialized'),
 ('v2', 'cmp-b', 'lib-b', '2.0.0', 'released', '2026-11-18T00:00:00Z', '2026-11-19T00:00:00Z', 'historical', 'evicted');
WITH RECURSIVE extra(n) AS (SELECT 3 UNION ALL SELECT n + 1 FROM extra WHERE n < 50)
INSERT INTO component_versions
SELECT 'v' || n, 'cmp-extra-' || n, 'lib-extra-' || n, '1.0.0', 'released', '', '', 'extra', 'materialized' FROM extra;
INSERT INTO catalog_gate_evidence VALUES
 ('controls.button', '1.0.0', 'pass'), ('controls.button', '1.0.0', 'PASS'),
 ('controls.button', '1.0.0', 'fail'), ('controls.button', '1.0.0', 'Failed');
INSERT INTO component_version_test_rollup VALUES ('lib-a', '1.0.0', 6, 5);
INSERT INTO adoption_records VALUES
 ('a1', 'cmp-a', 'lib-a', '1.0.0'), ('a2', 'cmp-a', 'lib-a', '1.0.0'), ('a3', 'cmp-a', 'lib-a', '1.0.0');
INSERT INTO component_asset_dependencies VALUES ('cmp-a', '1.0.0'), ('cmp-a', '1.0.0');
INSERT INTO component_version_files VALUES ('v1', 'Button.tsx'), ('v1', 'index.ts');
INSERT INTO version_ledger (library_id, version, retired_at, lifecycle_state, adoption_peak)
 VALUES ('lib-b', '2.0.0', '2026-11-20T00:00:00Z', 'retired', 7);`)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	componentDir := filepath.Join(root, "components", "Button")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(`{"libraryId":"lib-a","catalogId":"controls.button"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db, root)
	queryCount := 0
	if err := repo.rebuild(context.Background(), func(string) { queryCount++ }); err != nil {
		t.Fatal(err)
	}
	if queryCount != 7 {
		t.Fatalf("rebuild source query count = %d, want fixed count 7 for 50 versions", queryCount)
	}
	rows, err := repo.ListWindow(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 50 {
		t.Fatalf("rebuilt row count = %d, want 50", len(rows))
	}
	first := rows[0]
	if first.LinesOfCode != 3 || first.FileCount != 2 || first.LifecycleState != "released" || first.GatePassCount != 2 || first.GateFailCount != 2 || first.TestRuns != 6 || first.TestPassRate != 5.0/6 || first.AdoptionCurrent != 3 || first.DependencyCount != 2 || !first.CreatedAt.Equal(time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("rebuilt release metrics = %#v", first)
	}
	retired := rows[1]
	if retired.LifecycleState != "retired" || retired.AdoptionPeak != 7 || retired.Presence != "evicted" {
		t.Fatalf("rebuild lost retained lifecycle facts: %#v", retired)
	}
}

func TestRebuildPropagatesGroupedReadFailures(t *testing.T) {
	componentVersions := `CREATE TABLE component_versions (
 id TEXT, component_id TEXT, library_id TEXT, version TEXT, status TEXT,
 created_at TEXT, released_at TEXT, content TEXT
);`
	t.Run("gate query", func(t *testing.T) {
		db := databasetest.NewSQLite(t)
		if _, err := db.Exec(componentVersions); err != nil {
			t.Fatal(err)
		}
		err := NewRepository(db, t.TempDir()).Rebuild(context.Background())
		if err == nil || !strings.Contains(err.Error(), "read version rebuild catalog gate evidence") {
			t.Fatalf("Rebuild error = %v, want contextual catalog evidence query error", err)
		}
	})
	t.Run("gate scan", func(t *testing.T) {
		db := databasetest.NewSQLite(t)
		_, err := db.Exec(componentVersions + `
CREATE TABLE catalog_gate_evidence (asset_id TEXT, version TEXT, result TEXT);
INSERT INTO catalog_gate_evidence VALUES (NULL, '1.0.0', 'pass');`)
		if err != nil {
			t.Fatal(err)
		}
		err = NewRepository(db, t.TempDir()).Rebuild(context.Background())
		if err == nil || !strings.Contains(err.Error(), "scan version rebuild catalog gate evidence") {
			t.Fatalf("Rebuild error = %v, want contextual catalog evidence scan error", err)
		}
	})
	t.Run("test rollup query", func(t *testing.T) {
		db := databasetest.NewSQLite(t)
		_, err := db.Exec(componentVersions + `CREATE TABLE catalog_gate_evidence (asset_id TEXT, version TEXT, result TEXT);`)
		if err != nil {
			t.Fatal(err)
		}
		err = NewRepository(db, t.TempDir()).Rebuild(context.Background())
		if err == nil || !strings.Contains(err.Error(), "read version rebuild test rollups") {
			t.Fatalf("Rebuild error = %v, want contextual test rollup query error", err)
		}
	})
	t.Run("adoption aggregate query", func(t *testing.T) {
		db := databasetest.NewSQLite(t)
		_, err := db.Exec(componentVersions + `
CREATE TABLE catalog_gate_evidence (asset_id TEXT, version TEXT, result TEXT);
CREATE TABLE component_version_test_rollup (library_id TEXT, version TEXT, runs_total INTEGER, runs_passed INTEGER);`)
		if err != nil {
			t.Fatal(err)
		}
		err = NewRepository(db, t.TempDir()).Rebuild(context.Background())
		if err == nil || !strings.Contains(err.Error(), "read version rebuild adoption records") {
			t.Fatalf("Rebuild error = %v, want contextual adoption aggregate query error", err)
		}
	})
	t.Run("adoption aggregate scan", func(t *testing.T) {
		db := databasetest.NewSQLite(t)
		_, err := db.Exec(componentVersions + `
CREATE TABLE catalog_gate_evidence (asset_id TEXT, version TEXT, result TEXT);
CREATE TABLE component_version_test_rollup (library_id TEXT, version TEXT, runs_total INTEGER, runs_passed INTEGER);
CREATE TABLE adoption_records (library_id TEXT, adopted_version TEXT);
INSERT INTO adoption_records VALUES (NULL, '1.0.0');`)
		if err != nil {
			t.Fatal(err)
		}
		err = NewRepository(db, t.TempDir()).Rebuild(context.Background())
		if err == nil || !strings.Contains(err.Error(), "scan version rebuild adoption records") {
			t.Fatalf("Rebuild error = %v, want contextual adoption aggregate scan error", err)
		}
	})
	t.Run("retirement query", func(t *testing.T) {
		db := databasetest.NewSQLite(t)
		_, err := db.Exec(Schema() + componentVersions + `
CREATE TABLE catalog_gate_evidence (asset_id TEXT, version TEXT, result TEXT);
CREATE TABLE component_version_test_rollup (library_id TEXT, version TEXT, runs_total INTEGER, runs_passed INTEGER);
CREATE TABLE adoption_records (library_id TEXT, adopted_version TEXT);
CREATE TABLE component_asset_dependencies (component_id TEXT, version TEXT);
CREATE TABLE component_version_files (version_id TEXT);
DROP TABLE version_ledger;`)
		if err != nil {
			t.Fatal(err)
		}
		err = NewRepository(db, t.TempDir()).Rebuild(context.Background())
		if err == nil || !strings.Contains(err.Error(), "read version rebuild retirement records") {
			t.Fatalf("Rebuild error = %v, want contextual retirement query error", err)
		}
	})
	t.Run("retirement scan", func(t *testing.T) {
		db := databasetest.NewSQLite(t)
		_, err := db.Exec(Schema() + componentVersions + `
CREATE TABLE catalog_gate_evidence (asset_id TEXT, version TEXT, result TEXT);
CREATE TABLE component_version_test_rollup (library_id TEXT, version TEXT, runs_total INTEGER, runs_passed INTEGER);
CREATE TABLE adoption_records (library_id TEXT, adopted_version TEXT);
CREATE TABLE component_asset_dependencies (component_id TEXT, version TEXT);
CREATE TABLE component_version_files (version_id TEXT);
DROP TABLE version_ledger;
CREATE TABLE version_ledger (library_id TEXT, version TEXT, retired_at TEXT);
INSERT INTO version_ledger VALUES (NULL, '1.0.0', '');`)
		if err != nil {
			t.Fatal(err)
		}
		err = NewRepository(db, t.TempDir()).Rebuild(context.Background())
		if err == nil || !strings.Contains(err.Error(), "scan version rebuild retirement records") {
			t.Fatalf("Rebuild error = %v, want contextual retirement scan error", err)
		}
	})
}
