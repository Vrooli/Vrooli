package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"tunnel-manager/internal/clock"
)

// sqliteDNSLedger is the production DNSLedger over the dns_ownership table.
// It mirrors sqliteLedger (ingress ownership) so the two ledgers stay
// structurally identical — the only difference is the table and columns.
type sqliteDNSLedger struct {
	db    LedgerSQLExecutor
	clock clock.Clock
}

// NewSQLiteDNSLedger constructs the production DNSLedger.
func NewSQLiteDNSLedger(db LedgerSQLExecutor, clk clock.Clock) DNSLedger {
	if clk == nil {
		clk = clock.System{}
	}
	return &sqliteDNSLedger{db: db, clock: clk}
}

var _ DNSLedger = (*sqliteDNSLedger)(nil)

const (
	dnsLedgerColumns = `hostname, record_id, created_at`

	upsertDNSLedgerSQL = `
INSERT INTO dns_ownership (hostname, record_id, created_at)
VALUES (?, ?, ?)
ON CONFLICT(hostname) DO UPDATE SET
  record_id = excluded.record_id,
  created_at = excluded.created_at
`
)

func (s *sqliteDNSLedger) List(ctx context.Context) ([]DNSRecordEntry, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+dnsLedgerColumns+" FROM dns_ownership ORDER BY hostname ASC")
	if err != nil {
		return nil, fmt.Errorf("list dns ownership: %w", err)
	}
	defer rows.Close()

	var out []DNSRecordEntry
	for rows.Next() {
		e, err := scanDNSLedger(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dns ownership: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dns ownership: %w", err)
	}
	return out, nil
}

func (s *sqliteDNSLedger) Get(ctx context.Context, hostname string) (DNSRecordEntry, bool, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+dnsLedgerColumns+" FROM dns_ownership WHERE hostname = ?", hostname)
	e, err := scanDNSLedger(row)
	if errors.Is(err, sql.ErrNoRows) {
		return DNSRecordEntry{}, false, nil
	}
	if err != nil {
		return DNSRecordEntry{}, false, fmt.Errorf("get dns ownership %q: %w", hostname, err)
	}
	return e, true, nil
}

func (s *sqliteDNSLedger) Put(ctx context.Context, entry DNSRecordEntry) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, upsertDNSLedgerSQL,
		entry.Hostname, entry.RecordID, entry.CreatedAt.UTC().Format(ledgerTimeFormat),
	)
	if err != nil {
		return fmt.Errorf("upsert dns ownership %q: %w", entry.Hostname, err)
	}
	return nil
}

func (s *sqliteDNSLedger) Delete(ctx context.Context, hostname string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM dns_ownership WHERE hostname = ?", hostname)
	if err != nil {
		return false, fmt.Errorf("delete dns ownership %q: %w", hostname, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete dns ownership %q rows affected: %w", hostname, err)
	}
	return n > 0, nil
}

func scanDNSLedger(sc ledgerRowScanner) (DNSRecordEntry, error) {
	var (
		e          DNSRecordEntry
		createdRaw string
	)
	if err := sc.Scan(&e.Hostname, &e.RecordID, &createdRaw); err != nil {
		return DNSRecordEntry{}, err
	}
	if createdRaw != "" {
		created, err := time.Parse(ledgerTimeFormat, createdRaw)
		if err != nil {
			return DNSRecordEntry{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
		}
		e.CreatedAt = created
	}
	return e, nil
}
