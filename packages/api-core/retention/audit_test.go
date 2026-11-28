package retention

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// A manifest can only describe what it declares, so a set of entirely correct
// budgets still says nothing about the table nobody wrote a budget for. That is
// not a hypothetical gap: autoheal bounded three tables correctly while
// incident_observations grew unattended in the same file.
func TestAuditReportsLargeUndeclaredTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE budgeted (id INTEGER PRIMARY KEY, occurred_at TEXT, payload TEXT)`); err != nil {
		t.Fatalf("create budgeted: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE unbudgeted (id INTEGER PRIMARY KEY, payload TEXT)`); err != nil {
		t.Fatalf("create unbudgeted: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE tiny (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create tiny: %v", err)
	}

	// Push the undeclared table past the reporting floor.
	payload := strings.Repeat("x", 4096)
	tx, _ := db.Begin()
	stmt, err := tx.Prepare(`INSERT INTO unbudgeted (payload) VALUES (?)`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for range (UnbudgetedTableFloor / 4096) + 200 {
		if _, err := stmt.Exec(payload); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	m := &Manager{
		scenario:     "fixture",
		paths:        map[string]string{"budgeted": path},
		openDatabase: func(string) (Execer, error) { return db, nil },
		specs: []Spec{{
			Budget: Budget{Name: "budgeted", MaxBytes: 1 << 20},
			Target: Target{Kind: TargetSQLiteTable, Table: "budgeted", TimeColumn: "occurred_at"},
		}},
	}

	found, err := m.AuditUnbudgetedTables(context.Background())
	if err != nil {
		t.Fatalf("AuditUnbudgetedTables: %v", err)
	}

	var names []string
	for _, f := range found {
		names = append(names, f.Table)
	}
	if len(found) != 1 || found[0].Table != "unbudgeted" {
		t.Fatalf("audit reported %v, want exactly [unbudgeted]", names)
	}
	if found[0].Bytes < UnbudgetedTableFloor {
		t.Errorf("reported %d bytes, below the floor it was selected by", found[0].Bytes)
	}
	if found[0].Database != path {
		t.Errorf("Database = %q, want %q", found[0].Database, path)
	}
}

// The floor is what keeps the warning worth reading. A database whose only
// undeclared tables are bookkeeping must produce no findings at all.
func TestAuditIgnoresSmallAndInternalTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quiet.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// sqlite_sequence is created implicitly by AUTOINCREMENT and is SQLite's
	// own, not the component's to budget.
	if _, err := db.Exec(`CREATE TABLE budgeted (id INTEGER PRIMARY KEY AUTOINCREMENT, occurred_at TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO budgeted (occurred_at) VALUES ('now')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE cursors (k TEXT PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatalf("create cursors: %v", err)
	}

	m := &Manager{
		scenario:     "fixture",
		paths:        map[string]string{"budgeted": path},
		openDatabase: func(string) (Execer, error) { return db, nil },
		specs: []Spec{{
			Budget: Budget{Name: "budgeted", MaxBytes: 1 << 20},
			Target: Target{Kind: TargetSQLiteTable, Table: "budgeted", TimeColumn: "occurred_at"},
		}},
	}

	found, err := m.AuditUnbudgetedTables(context.Background())
	if err != nil {
		t.Fatalf("AuditUnbudgetedTables: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("audit reported %d findings for a database with nothing worth reporting; a noisy warning is one nobody reads", len(found))
	}
}
