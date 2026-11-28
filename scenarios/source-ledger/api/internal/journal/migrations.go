package journal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	vectorcodec "source-ledger/internal/vector"
)

// EnsureMigrations applies additive, idempotent upgrades that CREATE TABLE IF
// NOT EXISTS cannot apply to an already-created SQLite table.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	if err := ensureEntryMigrations(ctx, db); err != nil {
		return err
	}
	if exists, err := tableExists(ctx, db, "embeddings"); err != nil {
		return err
	} else if exists {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('embeddings') WHERE name='vector_blob'`).Scan(&count); err != nil {
			return fmt.Errorf("inspect embeddings.vector_blob: %w", err)
		}
		if count == 0 {
			if _, err := db.ExecContext(ctx, `ALTER TABLE embeddings ADD COLUMN vector_blob BLOB NOT NULL DEFAULT X''`); err != nil {
				return fmt.Errorf("add embeddings.vector_blob: %w", err)
			}
		}
		rows, err := db.QueryContext(ctx, `SELECT id,vector_json FROM embeddings WHERE length(vector_blob)=0 AND vector_json<>''`)
		if err != nil {
			return err
		}
		defer rows.Close()
		legacy := make([]struct{ id, raw string }, 0)
		for rows.Next() {
			var id, raw string
			if err := rows.Scan(&id, &raw); err != nil {
				rows.Close()
				return err
			}
			legacy = append(legacy, struct{ id, raw string }{id: id, raw: raw})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, item := range legacy {
			var values []float64
			if err := json.Unmarshal([]byte(item.raw), &values); err != nil {
				return fmt.Errorf("decode legacy embedding %s: %w", item.id, err)
			}
			if _, err := db.ExecContext(ctx, `UPDATE embeddings SET vector_blob=?,vector_json='' WHERE id=?`, vectorcodec.Encode(values), item.id); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureEntryMigrations adds compatibility columns and captures the existing
// rowids before any full rewrite is eligible. The original entries and their
// dependent tables stay in place. All changes commit together.
func ensureEntryMigrations(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin journal entry migration: %w", err)
	}
	defer tx.Rollback()
	columns, err := entryColumns(ctx, tx)
	if err != nil || len(columns) == 0 {
		return err
	}
	for _, column := range []struct{ name, definition string }{
		{"verification_status", "TEXT NOT NULL DEFAULT 'absent'"},
		{"harness_session_id", "TEXT NOT NULL DEFAULT ''"},
		{"harness_kind", "TEXT NOT NULL DEFAULT ''"},
		{"source_harness", "TEXT NOT NULL DEFAULT ''"},
		{"source_path", "TEXT NOT NULL DEFAULT ''"},
		{"imported_at", "TEXT NOT NULL DEFAULT ''"},
	} {
		if columns[column.name] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `ALTER TABLE entries ADD COLUMN `+column.name+` `+column.definition); err != nil {
			return fmt.Errorf("add entries.%s: %w", column.name, err)
		}
	}

	stable, err := tableExists(ctx, tx, "journal_entry_positions")
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, cursorSchema); err != nil {
		return fmt.Errorf("install journal position schema: %w", err)
	}
	if !stable {
		if _, err := tx.ExecContext(ctx, `INSERT INTO journal_entry_positions(position,entry_id)
SELECT rowid,id FROM entries ORDER BY rowid`); err != nil {
			return fmt.Errorf("preserve legacy journal rowids: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, seedHighWaterMark); err != nil {
		return fmt.Errorf("initialize journal high-water mark: %w", err)
	}
	if err := validateHighWaterMark(ctx, tx, true); err != nil {
		return err
	}
	return tx.Commit()
}

func entryColumns(ctx context.Context, tx *sql.Tx) (map[string]bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM pragma_table_info('entries')`)
	if err != nil {
		return nil, fmt.Errorf("inspect entries schema: %w", err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// ValidateHighWaterMark checks the latest watermark and, on a migrated file,
// coverage of the durable entry positions. Legacy backups still use rowids.
func ValidateHighWaterMark(ctx context.Context, db *sql.DB) error {
	stable, err := tableExists(ctx, db, "journal_entry_positions")
	if err != nil {
		return err
	}
	return validateHighWaterMark(ctx, db, stable)
}

func validateHighWaterMark(ctx context.Context, db rowQuerier, stable bool) error {
	query := `SELECT COALESCE(MAX(rowid),0) FROM entries`
	if stable {
		query = `SELECT COALESCE(MAX(position),0) FROM journal_entry_positions`
	}
	var marked, current int64
	if err := db.QueryRowContext(ctx, `SELECT max_rowid FROM journal_high_water_mark WHERE id=1`).Scan(&marked); err != nil {
		return fmt.Errorf("read journal high-water mark: %w", err)
	}
	if err := db.QueryRowContext(ctx, query).Scan(&current); err != nil {
		return fmt.Errorf("read journal maximum append position: %w", err)
	}
	if current < marked {
		return fmt.Errorf("journal high-water mark %d exceeds current maximum append position %d", marked, current)
	}
	if !stable {
		return nil
	}
	var inconsistent bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (
  SELECT 1 FROM entries e WHERE NOT EXISTS (SELECT 1 FROM journal_entry_positions p WHERE p.entry_id=e.id)
  UNION ALL
  SELECT 1 FROM journal_entry_positions p WHERE NOT EXISTS (SELECT 1 FROM entries e WHERE e.id=p.entry_id)
)`).Scan(&inconsistent); err != nil {
		return fmt.Errorf("verify durable journal positions: %w", err)
	}
	if inconsistent {
		return fmt.Errorf("journal entries and durable append positions are inconsistent")
	}
	return nil
}

func tableExists(ctx context.Context, db rowQuerier, table string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil {
		return false, fmt.Errorf("inspect %s: %w", table, err)
	}
	return n != 0, nil
}
