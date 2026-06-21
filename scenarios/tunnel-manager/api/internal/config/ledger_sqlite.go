package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"tunnel-manager/internal/clock"
)

// ledgerTimeFormat matches the RFC3339Nano round-trip used elsewhere in the
// scenario (see routes/sqlite.go) and the string stored in adopted_at.
const ledgerTimeFormat = time.RFC3339Nano

// LedgerSQLExecutor is the narrow database surface sqliteLedger depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (unit tests via
// testutil/db.NewSQLite) and *database.RoutedDB (production main.go) satisfy
// it.
type LedgerSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteLedger struct {
	db    LedgerSQLExecutor
	clock clock.Clock
}

// NewSQLiteLedger constructs the production OwnershipLedger.
func NewSQLiteLedger(db LedgerSQLExecutor, clk clock.Clock) OwnershipLedger {
	if clk == nil {
		clk = clock.System{}
	}
	return &sqliteLedger{db: db, clock: clk}
}

// Compile-time guarantee.
var _ OwnershipLedger = (*sqliteLedger)(nil)

const (
	ledgerColumns = `hostname, owner, scenario, note, adopted_at`

	upsertLedgerSQL = `
INSERT INTO ingress_ownership (hostname, owner, scenario, note, adopted_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(hostname) DO UPDATE SET
  owner = excluded.owner,
  scenario = excluded.scenario,
  note = excluded.note,
  adopted_at = excluded.adopted_at
`
)

func (s *sqliteLedger) List(ctx context.Context) ([]LedgerEntry, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+ledgerColumns+" FROM ingress_ownership ORDER BY hostname ASC")
	if err != nil {
		return nil, fmt.Errorf("list ownership: %w", err)
	}
	defer rows.Close()

	var out []LedgerEntry
	for rows.Next() {
		e, err := scanLedger(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ownership: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ownership: %w", err)
	}
	return out, nil
}

func (s *sqliteLedger) Get(ctx context.Context, hostname string) (LedgerEntry, bool, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+ledgerColumns+" FROM ingress_ownership WHERE hostname = ?", hostname)
	e, err := scanLedger(row)
	if errors.Is(err, sql.ErrNoRows) {
		return LedgerEntry{}, false, nil
	}
	if err != nil {
		return LedgerEntry{}, false, fmt.Errorf("get ownership %q: %w", hostname, err)
	}
	return e, true, nil
}

func (s *sqliteLedger) Put(ctx context.Context, entry LedgerEntry) error {
	if entry.AdoptedAt.IsZero() {
		entry.AdoptedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, upsertLedgerSQL,
		entry.Hostname, string(entry.Owner), entry.Scenario, entry.Note,
		entry.AdoptedAt.UTC().Format(ledgerTimeFormat),
	)
	if err != nil {
		return fmt.Errorf("upsert ownership %q: %w", entry.Hostname, err)
	}
	return nil
}

func (s *sqliteLedger) Delete(ctx context.Context, hostname string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM ingress_ownership WHERE hostname = ?", hostname)
	if err != nil {
		return false, fmt.Errorf("delete ownership %q: %w", hostname, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete ownership %q rows affected: %w", hostname, err)
	}
	return n > 0, nil
}

type ledgerRowScanner interface {
	Scan(dest ...any) error
}

func scanLedger(sc ledgerRowScanner) (LedgerEntry, error) {
	var (
		e          LedgerEntry
		ownerRaw   string
		adoptedRaw string
	)
	if err := sc.Scan(&e.Hostname, &ownerRaw, &e.Scenario, &e.Note, &adoptedRaw); err != nil {
		return LedgerEntry{}, err
	}
	e.Owner = Owner(ownerRaw)
	if adoptedRaw != "" {
		adopted, err := time.Parse(ledgerTimeFormat, adoptedRaw)
		if err != nil {
			return LedgerEntry{}, fmt.Errorf("parse adopted_at %q: %w", adoptedRaw, err)
		}
		e.AdoptedAt = adopted
	}
	return e, nil
}
