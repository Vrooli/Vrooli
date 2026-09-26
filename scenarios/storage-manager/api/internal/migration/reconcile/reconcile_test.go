package reconcile

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func openAt(t *testing.T, path string) *sql.DB {
	t.Helper()
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		t.Fatalf("build dsn: %v", err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// newPair builds a source database holding hijacked rows and an empty target
// that owns them, mirroring the real shape: a suite_executions parent with
// suite_execution_phases children.
func newPair(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "supervisor.sqlite")
	targetPath := filepath.Join(dir, "owner.db")

	const schema = `
CREATE TABLE suite_executions (id TEXT PRIMARY KEY, scenario TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE suite_execution_phases (id TEXT PRIMARY KEY, run_id TEXT NOT NULL, phase TEXT NOT NULL);`

	src := openAt(t, sourcePath)
	if _, err := src.Exec(schema); err != nil {
		t.Fatalf("source schema: %v", err)
	}
	if _, err := src.Exec(`
INSERT INTO suite_executions VALUES ('run-1','test-genie','passed'),('run-2','test-genie','failed');
INSERT INTO suite_execution_phases VALUES ('p-1','run-1','unit'),('p-2','run-1','security'),('p-3','run-2','unit');`); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}

	target := openAt(t, targetPath)
	if _, err := target.Exec(schema); err != nil {
		t.Fatalf("target schema: %v", err)
	}
	return sourcePath, target
}

func TestRunMovesRowsAndReportsCounts(t *testing.T) {
	sourcePath, target := newPair(t)

	res, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{"suite_executions", "suite_execution_phases"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := res.TotalInserted(); got != 5 {
		t.Fatalf("expected 5 rows moved, got %d", got)
	}
	for _, tr := range res.Tables {
		if tr.Inserted != tr.SourceRows {
			t.Fatalf("%s: inserted %d of %d source rows", tr.Table, tr.Inserted, tr.SourceRows)
		}
		if len(tr.DroppedColumns) != 0 {
			t.Fatalf("%s: unexpected dropped columns %v", tr.Table, tr.DroppedColumns)
		}
	}
	var n int
	if err := target.QueryRow(`SELECT COUNT(*) FROM suite_execution_phases`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 phase rows in the target, got %d", n)
	}
}

// TestRunIsIdempotent is the property that lets an operator re-run the move
// after a partial failure without thinking about what already landed.
func TestRunIsIdempotent(t *testing.T) {
	sourcePath, target := newPair(t)
	opts := Options{SourcePath: sourcePath, Tables: []string{"suite_executions", "suite_execution_phases"}}

	first, err := Run(context.Background(), target, opts)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}
	second, err := Run(context.Background(), target, opts)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if first.TotalInserted() != 5 {
		t.Fatalf("first run moved %d rows, want 5", first.TotalInserted())
	}
	if second.TotalInserted() != 0 {
		t.Fatalf("second run moved %d rows, want 0", second.TotalInserted())
	}
}

// TestRunNeverOverwritesOwnedRows is the safety property that matters most:
// a row the owning scenario wrote itself must survive the move untouched.
func TestRunNeverOverwritesOwnedRows(t *testing.T) {
	sourcePath, target := newPair(t)
	if _, err := target.Exec(`INSERT INTO suite_executions VALUES ('run-1','test-genie','own-value')`); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if _, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{"suite_executions"},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var status string
	if err := target.QueryRow(`SELECT status FROM suite_executions WHERE id='run-1'`).Scan(&status); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if status != "own-value" {
		t.Fatalf("the target's own row was overwritten: status=%q", status)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	sourcePath, target := newPair(t)

	res, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{"suite_executions"},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.DryRun {
		t.Fatal("result does not report a dry run")
	}
	if res.TotalInserted() != 0 {
		t.Fatalf("a dry run inserted %d rows", res.TotalInserted())
	}
	if res.Tables[0].SourceRows != 2 {
		t.Fatalf("a dry run must still report what it would move, got %d", res.Tables[0].SourceRows)
	}
	var n int
	if err := target.QueryRow(`SELECT COUNT(*) FROM suite_executions`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("a dry run wrote %d rows", n)
	}
}

// TestSchemaDriftIsReportedNotSilent covers the case where the two databases
// hold different versions of the same table.
func TestSchemaDriftIsReportedNotSilent(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "supervisor.sqlite")
	src := openAt(t, sourcePath)
	if _, err := src.Exec(`CREATE TABLE plans (id TEXT PRIMARY KEY, title TEXT, retired_field TEXT)`); err != nil {
		t.Fatalf("source schema: %v", err)
	}
	if _, err := src.Exec(`INSERT INTO plans VALUES ('p1','a plan','legacy')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = src.Close()

	target := openAt(t, filepath.Join(dir, "owner.db"))
	if _, err := target.Exec(`CREATE TABLE plans (id TEXT PRIMARY KEY, title TEXT)`); err != nil {
		t.Fatalf("target schema: %v", err)
	}

	res, err := Run(context.Background(), target, Options{SourcePath: sourcePath, Tables: []string{"plans"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	tr := res.Tables[0]
	if len(tr.DroppedColumns) != 1 || tr.DroppedColumns[0] != "retired_field" {
		t.Fatalf("schema drift was not reported: %v", tr.DroppedColumns)
	}
	if tr.Inserted != 1 {
		t.Fatalf("expected the row to move on the shared columns, inserted %d", tr.Inserted)
	}
	var title string
	if err := target.QueryRow(`SELECT title FROM plans WHERE id='p1'`).Scan(&title); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if title != "a plan" {
		t.Fatalf("unexpected title %q", title)
	}
}

func TestMissingTablesAreSkippedWithAReason(t *testing.T) {
	sourcePath, target := newPair(t)

	res, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{"suite_executions", "not_a_table"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	skipped := res.Tables[1]
	if !skipped.Skipped || skipped.Reason == "" {
		t.Fatalf("a missing table must be skipped with a reason: %+v", skipped)
	}
}

// TestSourceIsNeverModified pins the read-only attachment. The real source is a
// live database with an active writer.
func TestSourceIsNeverModified(t *testing.T) {
	sourcePath, target := newPair(t)

	if _, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{"suite_executions", "suite_execution_phases"},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	src := openAt(t, sourcePath)
	var n int
	if err := src.QueryRow(`SELECT COUNT(*) FROM suite_executions`).Scan(&n); err != nil {
		t.Fatalf("read source: %v", err)
	}
	if n != 2 {
		t.Fatalf("the source was modified: %d rows remain, want 2", n)
	}
}

func TestIdentifiersAreValidated(t *testing.T) {
	sourcePath, target := newPair(t)

	_, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{`suite_executions"; DROP TABLE suite_executions; --`},
	})
	if err == nil {
		t.Fatal("expected an invalid identifier to be rejected")
	}
	var n int
	if err := target.QueryRow(`SELECT COUNT(*) FROM suite_executions`).Scan(&n); err != nil {
		t.Fatalf("the target table was dropped: %v", err)
	}
}

func TestRunRejectsEmptyInput(t *testing.T) {
	_, target := newPair(t)
	if _, err := Run(context.Background(), target, Options{Tables: []string{"suite_executions"}}); err == nil {
		t.Fatal("expected a missing source path to be rejected")
	}
	if _, err := Run(context.Background(), target, Options{SourcePath: "x.db"}); err == nil {
		t.Fatal("expected an empty table list to be rejected")
	}
}

// --- surrogate keys -----------------------------------------------------
//
// These pin the branch that was missing when this tool was first run against
// real data: an audit log with an autoincrement id reported success and moved
// nothing, because every source id was already taken in the target by an
// unrelated row.

func newAuditPair(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "supervisor.sqlite")

	const schema = `CREATE TABLE git_audit_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	operation TEXT NOT NULL,
	repo_dir TEXT NOT NULL,
	branch TEXT,
	created_at TEXT NOT NULL)`

	src := openAt(t, sourcePath)
	if _, err := src.Exec(schema); err != nil {
		t.Fatalf("source schema: %v", err)
	}
	if _, err := src.Exec(`INSERT INTO git_audit_log (operation, repo_dir, branch, created_at) VALUES
		('commit','/repo/a','agi','2026-08-18T04:35:38Z'),
		('push','/repo/a',NULL,'2026-08-18T04:36:00Z')`); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	_ = src.Close()

	target := openAt(t, filepath.Join(dir, "owner.db"))
	if _, err := target.Exec(schema); err != nil {
		t.Fatalf("target schema: %v", err)
	}
	// The target already holds unrelated rows occupying ids 1 and 2 — the same
	// numbers the source rows carry.
	if _, err := target.Exec(`INSERT INTO git_audit_log (operation, repo_dir, branch, created_at) VALUES
		('status','/repo/b','main','2026-08-01T00:00:00Z'),
		('fetch','/repo/b','main','2026-08-01T00:01:00Z')`); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	return sourcePath, target
}

// TestSurrogateKeysDoNotSwallowRows is the regression test for real data loss.
func TestSurrogateKeysDoNotSwallowRows(t *testing.T) {
	sourcePath, target := newAuditPair(t)

	res, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{"git_audit_log"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	tr := res.Tables[0]
	if tr.SurrogateKey != "id" {
		t.Fatalf("expected the surrogate key to be detected, got %q", tr.SurrogateKey)
	}
	if tr.Inserted != 2 {
		t.Fatalf("expected both source rows to move despite colliding ids, moved %d", tr.Inserted)
	}

	var n int
	if err := target.QueryRow(`SELECT COUNT(*) FROM git_audit_log WHERE repo_dir='/repo/a'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 recovered rows, got %d", n)
	}
	// The target's own rows must be untouched.
	if err := target.QueryRow(`SELECT COUNT(*) FROM git_audit_log WHERE repo_dir='/repo/b'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("the target's own rows were disturbed: %d remain", n)
	}
}

// TestSurrogateKeyCopyIsIdempotent proves the replacement for key-based
// idempotency actually holds — including for rows containing NULL, which a
// naive equality comparison would re-insert on every run.
func TestSurrogateKeyCopyIsIdempotent(t *testing.T) {
	sourcePath, target := newAuditPair(t)
	opts := Options{SourcePath: sourcePath, Tables: []string{"git_audit_log"}}

	if _, err := Run(context.Background(), target, opts); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	second, err := Run(context.Background(), target, opts)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if second.TotalInserted() != 0 {
		t.Fatalf("second run re-inserted %d rows", second.TotalInserted())
	}

	var n int
	if err := target.QueryRow(`SELECT COUNT(*) FROM git_audit_log`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 rows after two runs, got %d", n)
	}
	// Specifically the NULL-bearing row must not have been duplicated.
	if err := target.QueryRow(`SELECT COUNT(*) FROM git_audit_log WHERE branch IS NULL`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("the NULL-bearing row was duplicated: %d copies", n)
	}
}

// TestMeaningfulKeysStillUseKeyIdentity confirms the surrogate branch did not
// take over tables whose key genuinely identifies the row.
func TestMeaningfulKeysStillUseKeyIdentity(t *testing.T) {
	sourcePath, target := newPair(t)

	res, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath,
		Tables:     []string{"suite_executions"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Tables[0].SurrogateKey != "" {
		t.Fatalf("a TEXT primary key must not be treated as a surrogate, got %q", res.Tables[0].SurrogateKey)
	}
	var id string
	if err := target.QueryRow(`SELECT id FROM suite_executions WHERE scenario='test-genie' ORDER BY id LIMIT 1`).Scan(&id); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if id != "run-1" {
		t.Fatalf("the meaningful key was not preserved, got %q", id)
	}
}

// TestUniqueConflictIsReportedNotFatal covers the case a live run hit: a source
// row that is not a duplicate but collides with a DIFFERENT target row on a
// unique constraint. The same repository path was recorded under two ids. Such
// a row cannot be merged automatically; it must be counted and reported rather
// than abort the whole move or be silently dropped.
func TestUniqueConflictIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "supervisor.sqlite")

	const schema = `CREATE TABLE git_repos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT NOT NULL UNIQUE,
	label TEXT)`

	src := openAt(t, sourcePath)
	if _, err := src.Exec(schema); err != nil {
		t.Fatalf("source schema: %v", err)
	}
	if _, err := src.Exec(`INSERT INTO git_repos (path, label) VALUES ('/repo/shared','from-source'),('/repo/new','fresh')`); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	_ = src.Close()

	target := openAt(t, filepath.Join(dir, "owner.db"))
	if _, err := target.Exec(schema); err != nil {
		t.Fatalf("target schema: %v", err)
	}
	if _, err := target.Exec(`INSERT INTO git_repos (path, label) VALUES ('/repo/shared','from-target')`); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	res, err := Run(context.Background(), target, Options{SourcePath: sourcePath, Tables: []string{"git_repos"}})
	if err != nil {
		t.Fatalf("a unique conflict must not abort the move: %v", err)
	}
	tr := res.Tables[0]
	if tr.Inserted != 1 {
		t.Fatalf("expected the non-conflicting row to move, inserted %d", tr.Inserted)
	}
	if tr.Conflicted != 1 {
		t.Fatalf("expected 1 conflict to be reported, got %d", tr.Conflicted)
	}

	// The target's own version of the shared row must win.
	var label string
	if err := target.QueryRow(`SELECT label FROM git_repos WHERE path='/repo/shared'`).Scan(&label); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if label != "from-target" {
		t.Fatalf("a conflicting source row overwrote the target's own: %q", label)
	}
}

// TestDryRunReportsWhatIsAlreadyThere keeps a preview honest. After a
// successful move, re-previewing must show the rows as already present rather
// than as still outstanding — otherwise an operator cannot tell a completed
// reconciliation from one that has not started.
func TestDryRunReportsWhatIsAlreadyThere(t *testing.T) {
	sourcePath, target := newPair(t)
	opts := Options{SourcePath: sourcePath, Tables: []string{"suite_executions"}}

	preview, err := Run(context.Background(), target, opts)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	_ = preview

	if _, err := Run(context.Background(), target, opts); err != nil {
		t.Fatalf("apply: %v", err)
	}

	after, err := Run(context.Background(), target, Options{
		SourcePath: sourcePath, Tables: []string{"suite_executions"}, DryRun: true,
	})
	if err != nil {
		t.Fatalf("second preview: %v", err)
	}
	tr := after.Tables[0]
	if tr.Identical != tr.SourceRows {
		t.Fatalf("a completed move must preview as fully identical: %d of %d", tr.Identical, tr.SourceRows)
	}
}
