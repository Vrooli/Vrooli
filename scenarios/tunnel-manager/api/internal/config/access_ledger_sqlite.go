package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// sqliteAccessLedger is the production AccessLedger over the access_ownership
// table. It mirrors sqliteDNSLedger so the two ownership ledgers stay
// structurally identical — the only difference is the table and columns.
type sqliteAccessLedger struct {
	db    LedgerSQLExecutor
	clock schedule.Clock
}

// NewSQLiteAccessLedger constructs the production AccessLedger.
func NewSQLiteAccessLedger(db LedgerSQLExecutor, clk schedule.Clock) AccessLedger {
	if clk == nil {
		clk = schedule.System()
	}
	return &sqliteAccessLedger{db: db, clock: clk}
}

var _ AccessLedger = (*sqliteAccessLedger)(nil)

const (
	accessLedgerColumns = `host, app_id, policy_id, created_at`

	upsertAccessLedgerSQL = `
INSERT INTO access_ownership (host, app_id, policy_id, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(host) DO UPDATE SET
  app_id = excluded.app_id,
  policy_id = excluded.policy_id,
  created_at = excluded.created_at
`
)

func (s *sqliteAccessLedger) List(ctx context.Context) ([]AccessAppEntry, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+accessLedgerColumns+" FROM access_ownership ORDER BY host ASC")
	if err != nil {
		return nil, fmt.Errorf("list access ownership: %w", err)
	}
	defer rows.Close()

	var out []AccessAppEntry
	for rows.Next() {
		e, err := scanAccessLedger(rows)
		if err != nil {
			return nil, fmt.Errorf("scan access ownership: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access ownership: %w", err)
	}
	return out, nil
}

func (s *sqliteAccessLedger) Get(ctx context.Context, host string) (AccessAppEntry, bool, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+accessLedgerColumns+" FROM access_ownership WHERE host = ?", host)
	e, err := scanAccessLedger(row)
	if errors.Is(err, sql.ErrNoRows) {
		return AccessAppEntry{}, false, nil
	}
	if err != nil {
		return AccessAppEntry{}, false, fmt.Errorf("get access ownership %q: %w", host, err)
	}
	return e, true, nil
}

func (s *sqliteAccessLedger) Put(ctx context.Context, entry AccessAppEntry) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, upsertAccessLedgerSQL,
		entry.Host, entry.AppID, entry.PolicyID, entry.CreatedAt.UTC().Format(ledgerTimeFormat),
	)
	if err != nil {
		return fmt.Errorf("upsert access ownership %q: %w", entry.Host, err)
	}
	return nil
}

func (s *sqliteAccessLedger) Delete(ctx context.Context, host string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM access_ownership WHERE host = ?", host)
	if err != nil {
		return false, fmt.Errorf("delete access ownership %q: %w", host, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete access ownership %q rows affected: %w", host, err)
	}
	return n > 0, nil
}

func scanAccessLedger(sc ledgerRowScanner) (AccessAppEntry, error) {
	var (
		e          AccessAppEntry
		createdRaw string
	)
	if err := sc.Scan(&e.Host, &e.AppID, &e.PolicyID, &createdRaw); err != nil {
		return AccessAppEntry{}, err
	}
	if createdRaw != "" {
		created, err := time.Parse(ledgerTimeFormat, createdRaw)
		if err != nil {
			return AccessAppEntry{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
		}
		e.CreatedAt = created
	}
	return e, nil
}
