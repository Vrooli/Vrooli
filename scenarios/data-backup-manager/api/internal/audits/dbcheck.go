package audits

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sort"

	"github.com/vrooli/api-core/database"
)

// SQLiteChecker inspects a SQLite database file with generic, read-only checks:
// PRAGMA integrity_check, page count/size, a normalized schema hash, and a
// generic count of schema objects. It never reads row data and never interprets
// any table by name — it carries no domain meaning.
//
// seam: SQLiteChecker is the read-only DB inspection boundary. Production wires
// pragmaChecker (opens the file with the modernc.org/sqlite driver in
// immutable read-only mode); tests wire a fake to assert call shape without a
// real DB.
type SQLiteChecker interface {
	// Check inspects the SQLite file at abs and returns its generic facts. rel
	// is the tree-relative path recorded on the result. A file that is not a
	// usable SQLite database yields IntegrityStatus="failed" and no error — a
	// corrupt DB is a finding, not an audit failure.
	Check(ctx context.Context, abs, rel string) SqliteInventory
}

// pragmaChecker is the production SQLiteChecker. It opens each file with a
// dedicated read-only, immutable connection so it never takes a write lock on a
// live database and never mutates it.
type pragmaChecker struct{}

// NewSQLiteChecker returns the production read-only SQLite checker.
func NewSQLiteChecker() SQLiteChecker { return pragmaChecker{} }

var _ SQLiteChecker = pragmaChecker{}

func (pragmaChecker) Check(ctx context.Context, abs, rel string) SqliteInventory {
	inv := SqliteInventory{Path: rel, IntegrityStatus: "not_checked"}

	// immutable=1 + mode=ro: never write-lock, never create, never modify. This
	// is the read-only-by-construction open that lets us inspect even a live DB
	// without contending with writers. Opened through api-core/database (the
	// sanctioned connection path, matching the sqlite source capturer) rather
	// than sql.Open directly, so this external read gets the same retry/backoff
	// behavior as scenario DB opens.
	dsn := fmt.Sprintf("file:%s?mode=ro&immutable=1&_pragma=query_only(1)", abs)
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		inv.IntegrityStatus = "failed"
		return inv
	}
	defer db.Close()

	// integrity_check returns the single row "ok" on a healthy database, or one
	// row per problem otherwise.
	var integrity string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check(1)").Scan(&integrity); err != nil {
		inv.IntegrityStatus = "failed"
		return inv
	}
	if integrity == "ok" {
		inv.IntegrityStatus = "ok"
	} else {
		inv.IntegrityStatus = "failed"
	}

	inv.PageCount = scalarInt(ctx, db, "PRAGMA page_count")
	inv.PageSize = scalarInt(ctx, db, "PRAGMA page_size")
	inv.SchemaSHA256, inv.TableCount = schemaFingerprint(ctx, db)
	return inv
}

// scalarInt runs a single-int PRAGMA, returning 0 on any error.
func scalarInt(ctx context.Context, db *sql.DB, query string) int64 {
	var n int64
	if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
		return 0
	}
	return n
}

// schemaFingerprint hashes the normalized sqlite_master schema and counts its
// objects. Rows are sorted by (type, name, tbl_name) so the hash is stable
// regardless of creation order — two databases with the same DDL produce the
// same fingerprint. Only DDL (the CREATE statements already in sqlite_master)
// is read; no row data and no domain interpretation.
func schemaFingerprint(ctx context.Context, db *sql.DB) (string, int64) {
	rows, err := db.QueryContext(ctx,
		`SELECT type, name, tbl_name, COALESCE(sql, '') FROM sqlite_master WHERE name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return "", 0
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var typ, name, tbl, ddl string
		if err := rows.Scan(&typ, &name, &tbl, &ddl); err != nil {
			return "", 0
		}
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s", typ, name, tbl, ddl))
	}
	if err := rows.Err(); err != nil {
		return "", 0
	}
	sort.Strings(lines)

	h := sha256.New()
	for _, l := range lines {
		fmt.Fprintf(h, "%s\n", l)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), int64(len(lines))
}
