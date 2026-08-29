package baselinefloor

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	// Register the pure-Go SQLite driver ("sqlite"). It is already a platform
	// dependency (used by internal/scenarioruntime), so the migration runner adds
	// no new dependency — a hard constraint for the trusted base.
	_ "modernc.org/sqlite"
)

// Engine identifies the storage engine a baseline migration set targets. v1
// implements SQLite only; any other engine is a deliberate, surfaced gap so the
// promote decision tree routes a scenario carrying a non-SQLite schema migration
// to live mode rather than risk silent corruption (Baseline Modes §8, Risk §11,
// open design question #1). Postgres support needs a driver the platform does not
// ship and is intentionally out of v1 scope.
type Engine string

// EngineSQLite is the only engine the v1 migration runner applies.
const EngineSQLite Engine = "sqlite"

// migrationTrackingTable records which scripts a database has had applied, so a
// re-run is a no-op (idempotency-replay-safety) and an edit-after-apply is caught
// rather than silently re-run. It is created on demand inside the apply
// transaction; the leading underscore keeps it clear of a scenario's own tables.
const migrationTrackingTable = "_vrooli_baseline_migrations"

// migrationScriptExt is the suffix LoadScripts discovers, applied in lexical
// (filename) order — author scripts as 001-*.sql, 002-*.sql, ... .
const migrationScriptExt = ".sql"

// Script is one ordered migration file: its base Name (the tracking key), raw
// SQL, and a content Checksum used to detect an edit after a prior apply.
type Script struct {
	Name     string `json:"name"`
	SQL      string `json:"-"`
	Checksum string `json:"checksum"`
}

// MigrateOptions tunes a run.
type MigrateOptions struct {
	// DryRun validates the scripts against a throwaway copy of the database and
	// reports the outcome WITHOUT touching the real database.
	DryRun bool
}

// MigrationResult reports what a run did — the --json shape, embedded by the
// recovery service output.
type MigrationResult struct {
	Engine   Engine `json:"engine"`
	Database string `json:"database,omitempty"`
	DryRun   bool   `json:"dryRun"`
	// FastPath is true when no scripts were authored: the shape-unchanged fast
	// path skips DB handling entirely (no database is even opened).
	FastPath    bool     `json:"fastPath"`
	ScriptsSeen int      `json:"scriptsSeen"`
	Applied     []string `json:"applied"`
	Skipped     []string `json:"skipped"`
}

// LoadScripts reads every *.sql file in dir in lexical (filename) order. A
// missing directory returns (nil, nil): "no scripts authored" is the
// shape-unchanged fast path — promote skips DB handling. Each script's checksum
// is the sha256 of its bytes, so an edit after a prior apply is detectable.
func LoadScripts(dir string) ([]Script, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("baselinefloor: read migrations dir %q: %w", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), migrationScriptExt) {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	scripts := make([]Script, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("baselinefloor: read migration %q: %w", name, err)
		}
		sum := sha256.Sum256(data)
		scripts = append(scripts, Script{Name: name, SQL: string(data), Checksum: hex.EncodeToString(sum[:])})
	}
	return scripts, nil
}

// RunMigrations applies ordered scripts to a database — the promote step's schema
// runner (Baseline Modes §8). No scripts ⇒ FastPath no-op. Otherwise it ALWAYS
// dry-runs the scripts against a throwaway copy of the current database first (so
// a script incompatible with the live shape becomes a detected bounce, never
// silent corruption) and only then applies them to the real database in a single
// transaction (a partial failure rolls the whole batch back). A previously
// applied, unchanged script is skipped (re-run is a no-op); a previously applied
// script whose contents changed is a hard error.
//
// Only EngineSQLite is implemented; any other engine is a surfaced error so the
// caller bounces to live mode rather than guessing.
func RunMigrations(engine Engine, dbPath string, scripts []Script, opts MigrateOptions) (MigrationResult, error) {
	res := MigrationResult{Engine: engine, Database: dbPath, DryRun: opts.DryRun, ScriptsSeen: len(scripts)}
	if len(scripts) == 0 {
		res.FastPath = true
		return res, nil
	}
	if engine != EngineSQLite {
		return res, fmt.Errorf("baselinefloor: migration engine %q not supported (v1: sqlite only) — route this scenario's schema change to live mode", engine)
	}
	if strings.TrimSpace(dbPath) == "" {
		return res, fmt.Errorf("baselinefloor: %d migration script(s) to apply but no database path given", len(scripts))
	}

	// Validate against a throwaway copy of current-live first; bounce on failure.
	applied, skipped, err := runAgainstCopy(dbPath, scripts)
	if err != nil {
		return res, fmt.Errorf("baselinefloor: migration dry-run failed (live untouched): %w", err)
	}
	if opts.DryRun {
		// Validation only — report what an apply WOULD do without mutating live.
		// The copy carries live's tracking table, so its plan matches the live apply.
		res.Applied, res.Skipped = applied, skipped
		return res, nil
	}

	applied, skipped, err = applySQLite(dbPath, scripts)
	if err != nil {
		return res, fmt.Errorf("baselinefloor: apply migrations to %q: %w", dbPath, err)
	}
	res.Applied, res.Skipped = applied, skipped
	return res, nil
}

// runAgainstCopy copies the database (and any SQLite sidecar files) into a temp
// dir and applies the scripts there, leaving the real database untouched. It is
// the dry-run / pre-apply validation half of the promote sequence.
func runAgainstCopy(dbPath string, scripts []Script) (applied, skipped []string, err error) {
	tmpDir, err := os.MkdirTemp("", "vrooli-migrate-*")
	if err != nil {
		return nil, nil, fmt.Errorf("dry-run scratch dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	candidate := filepath.Join(tmpDir, "candidate.db")
	if err := copySQLiteDatabase(dbPath, candidate); err != nil {
		return nil, nil, fmt.Errorf("copy database for dry-run: %w", err)
	}
	return applySQLite(candidate, scripts)
}

// applySQLite opens dbPath (creating it if absent) and applies the scripts within
// a single transaction, tracking applied scripts in migrationTrackingTable. A
// previously applied unchanged script is skipped; a checksum mismatch is a hard
// error; any execution error rolls the whole batch back.
func applySQLite(dbPath string, scripts []Script) (applied, skipped []string, err error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite %q: %w", dbPath, err)
	}
	// A migration holds one connection for the whole transaction; cap the pool so
	// SQLite never trips "database is locked" against itself.
	db.SetMaxOpenConns(1)
	defer func() { _ = db.Close() }()

	tx, err := db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin: %w", err)
	}
	rollback := func(cause error) (applied, skipped []string, err error) {
		_ = tx.Rollback()
		return nil, nil, cause
	}

	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS " + migrationTrackingTable +
		" (name TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL)"); err != nil {
		return rollback(fmt.Errorf("ensure tracking table: %w", err))
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, s := range scripts {
		var existing string
		queryErr := tx.QueryRow("SELECT checksum FROM "+migrationTrackingTable+" WHERE name = ?", s.Name).Scan(&existing)
		switch {
		case queryErr == nil:
			if existing != s.Checksum {
				return rollback(fmt.Errorf("migration %q already applied with a different checksum (edited after apply); refusing to re-run", s.Name))
			}
			skipped = append(skipped, s.Name)
			continue
		case errors.Is(queryErr, sql.ErrNoRows):
			// Not yet applied — fall through and execute it.
		default:
			return rollback(fmt.Errorf("look up migration %q: %w", s.Name, queryErr))
		}

		if _, err := tx.Exec(s.SQL); err != nil {
			return rollback(fmt.Errorf("execute migration %q: %w", s.Name, err))
		}
		if _, err := tx.Exec("INSERT INTO "+migrationTrackingTable+" (name, checksum, applied_at) VALUES (?, ?, ?)",
			s.Name, s.Checksum, now); err != nil {
			return rollback(fmt.Errorf("record migration %q: %w", s.Name, err))
		}
		applied = append(applied, s.Name)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit: %w", err)
	}
	return applied, skipped, nil
}

// copySQLiteDatabase copies a SQLite database file (and its -wal/-shm sidecars,
// if present) from src to dst for dry-run validation. A missing src is not an
// error: the candidate simply starts empty, which is correct for a migration set
// that creates the schema from scratch.
func copySQLiteDatabase(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		from := src + suffix
		if _, err := os.Stat(from); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		if err := copyFileBytes(from, dst+suffix); err != nil {
			return err
		}
	}
	return nil
}

func copyFileBytes(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
