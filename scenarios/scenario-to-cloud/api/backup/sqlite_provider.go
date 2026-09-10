package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/packages/recoverypoint"
	_ "modernc.org/sqlite" // embedded database engine for sqlite bindings
)

// SQLite captures an embedded database binding with database-native
// consistency (`VACUUM INTO` writes a transactionally consistent copy) and
// restores it only into a path that holds no data. The locator is the
// absolute database file path. Its inventory is the row count plus a
// checksum over every user table's ordered rows, so a restore is comparable
// byte-for-byte at the logical level even though the file bytes are not.
//
// It is the package-lane stand-in for the PostgreSQL provider in the
// fresh-host drill and the production provider for scenarios whose declared
// binding is `sqlite:<path>`.
type SQLite struct{}

// Mode implements recoverypoint.Provider.
func (SQLite) Mode() string { return recoverypoint.ModeDatabaseNative }

func openSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}

// Capture implements recoverypoint.Provider.
func (p SQLite) Capture(ctx context.Context, b recoverypoint.Binding, stageDir string) (recoverypoint.CaptureResult, error) {
	if !filepath.IsAbs(b.Locator) {
		return recoverypoint.CaptureResult{}, fmt.Errorf("binding %s: sqlite locator must be an absolute path", b.ID)
	}
	if _, err := os.Stat(b.Locator); err != nil {
		return recoverypoint.CaptureResult{}, fmt.Errorf("binding %s: database %s is not readable: %w", b.ID, b.Locator, err)
	}
	db, err := openSQLite(b.Locator)
	if err != nil {
		return recoverypoint.CaptureResult{}, fmt.Errorf("binding %s: open database: %w", b.ID, err)
	}
	defer db.Close()
	artifact := filepath.Join(stageDir, b.ID+".sqlite")
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", artifact); err != nil {
		return recoverypoint.CaptureResult{}, fmt.Errorf("binding %s: VACUUM INTO: %w", b.ID, err)
	}
	inv, err := p.Inventory(ctx, b, artifact)
	if err != nil {
		return recoverypoint.CaptureResult{}, err
	}
	return recoverypoint.CaptureResult{ArtifactPath: artifact, Inventory: inv}, nil
}

// EnsureClean implements recoverypoint.Provider: the target file must not
// exist or must be empty.
func (SQLite) EnsureClean(_ context.Context, b recoverypoint.Binding, target string) error {
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("binding %s: inspect %s: %w", b.ID, target, err)
	}
	if info.Size() > 0 {
		return &recoverypoint.Error{Code: recoverypoint.CodeRestoreTargetNotClean, Message: fmt.Sprintf("binding %s: %s already holds a database; restore never overwrites operator-owned data", b.ID, target), Details: map[string]any{"binding": b.ID, "target": target}}
	}
	return nil
}

// Restore implements recoverypoint.Provider by placing the consistent copy at
// the clean target path.
func (p SQLite) Restore(ctx context.Context, b recoverypoint.Binding, artifactPath, target string) error {
	if err := p.EnsureClean(ctx, b, target); err != nil {
		return err
	}
	raw, err := os.ReadFile(artifactPath) //nolint:gosec // engine-owned staged artifact
	if err != nil {
		return fmt.Errorf("binding %s: read artifact: %w", b.ID, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("binding %s: create target directory: %w", b.ID, err)
	}
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		return fmt.Errorf("binding %s: write database: %w", b.ID, err)
	}
	return nil
}

// Discard implements recoverypoint.Provider.
func (SQLite) Discard(_ context.Context, _ recoverypoint.Binding, target string) error {
	return os.Remove(target)
}

// Inventory implements recoverypoint.Provider: total rows across user tables
// and a checksum over each table's ordered rows.
func (SQLite) Inventory(ctx context.Context, b recoverypoint.Binding, target string) (recoverypoint.Inventory, error) {
	db, err := openSQLite(target)
	if err != nil {
		return recoverypoint.Inventory{}, fmt.Errorf("binding %s: open %s: %w", b.ID, target, err)
	}
	defer db.Close()
	tables, err := sqliteTables(ctx, db)
	if err != nil {
		return recoverypoint.Inventory{}, fmt.Errorf("binding %s: list tables: %w", b.ID, err)
	}
	h := sha256.New()
	var count int64
	for _, table := range tables {
		rows, err := db.QueryContext(ctx, `SELECT * FROM "`+table+`" ORDER BY 1`)
		if err != nil {
			return recoverypoint.Inventory{}, fmt.Errorf("binding %s: read %s: %w", b.ID, table, err)
		}
		fmt.Fprintf(h, "table %s\n", table)
		cols, _ := rows.Columns()
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return recoverypoint.Inventory{}, fmt.Errorf("binding %s: scan %s: %w", b.ID, table, err)
			}
			for i, v := range values {
				if raw, ok := v.([]byte); ok {
					values[i] = string(raw)
				}
			}
			line, _ := json.Marshal(values)
			h.Write(line)
			h.Write([]byte("\n"))
			count++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return recoverypoint.Inventory{}, fmt.Errorf("binding %s: iterate %s: %w", b.ID, table, err)
		}
		rows.Close()
	}
	return recoverypoint.Inventory{Count: count, Checksum: hex.EncodeToString(h.Sum(nil)), Comparable: true}, nil
}

func sqliteTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}
