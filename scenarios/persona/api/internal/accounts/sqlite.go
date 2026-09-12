package accounts

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	sqliteRepository struct {
		db    SQLExecutor
		clock schedule.Clock
	}
)

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) Link(ctx context.Context, a AccountLink) (AccountLink, error) {
	a.ID = uuid.NewString()
	a.CreatedAt = r.clock.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO persona_account_links (id, persona_id, site, login_seam, recovery_path, created_at) VALUES (?, ?, ?, ?, ?, ?)`, a.ID, a.PersonaID, a.Site, a.LoginSeam, a.RecoveryPath, a.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return AccountLink{}, err
	}
	return a, nil
}

func (r *sqliteRepository) ListAccounts(ctx context.Context, id string) ([]AccountLink, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, site, login_seam, recovery_path, created_at FROM persona_account_links WHERE persona_id = ? ORDER BY created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountLink
	for rows.Next() {
		var a AccountLink
		var created string
		if err := rows.Scan(&a.ID, &a.PersonaID, &a.Site, &a.LoginSeam, &a.RecoveryPath, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) AddAddress(ctx context.Context, a Address) (Address, error) {
	a.ID = uuid.NewString()
	a.CreatedAt = r.clock.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO persona_addresses (id, persona_id, label, line1, line2, city, region, postal_code, country, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, a.ID, a.PersonaID, a.Label, a.Line1, a.Line2, a.City, a.Region, a.PostalCode, a.Country, a.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Address{}, err
	}
	return a, nil
}

func (r *sqliteRepository) ListAddresses(ctx context.Context, id string) ([]Address, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, label, line1, line2, city, region, postal_code, country, created_at FROM persona_addresses WHERE persona_id = ? ORDER BY created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Address
	for rows.Next() {
		a, err := scanAddress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) GetAddress(ctx context.Context, personaID, id string) (Address, error) {
	a, err := scanAddress(r.db.QueryRowContext(ctx, `SELECT id, persona_id, label, line1, line2, city, region, postal_code, country, created_at FROM persona_addresses WHERE persona_id = ? AND id = ?`, personaID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Address{}, ErrMissingAddress
	}
	return a, err
}

func (r *sqliteRepository) AddObligation(ctx context.Context, o Obligation) (Obligation, error) {
	o.ID = uuid.NewString()
	o.CreatedAt = r.clock.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO persona_obligations (id, persona_id, account_link_id, description, renewal_at, cancel_path, cancelled, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, o.ID, o.PersonaID, o.AccountLinkID, o.Description, o.RenewalAt.UTC().Format(time.RFC3339Nano), boolInt(o.Cancelled), o.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Obligation{}, err
	}
	return o, nil
}

func (r *sqliteRepository) ListObligations(ctx context.Context, id string) ([]Obligation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, account_link_id, description, renewal_at, cancel_path, cancelled, created_at FROM persona_obligations WHERE persona_id = ? ORDER BY renewal_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Obligation
	for rows.Next() {
		o, err := scanObligation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) GetObligation(ctx context.Context, id string) (Obligation, error) {
	return scanObligation(r.db.QueryRowContext(ctx, `SELECT id, persona_id, account_link_id, description, renewal_at, cancel_path, cancelled, created_at FROM persona_obligations WHERE id = ?`, id))
}

func (r *sqliteRepository) CancelObligation(ctx context.Context, id string) (Obligation, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE persona_obligations SET cancelled = 1 WHERE id = ?`, id)
	if err != nil {
		return Obligation{}, err
	}
	return r.GetObligation(ctx, id)
}

type rowScanner interface{ Scan(...any) error }

func scanAddress(row rowScanner) (Address, error) {
	var a Address
	var created string
	if err := row.Scan(&a.ID, &a.PersonaID, &a.Label, &a.Line1, &a.Line2, &a.City, &a.Region, &a.PostalCode, &a.Country, &created); err != nil {
		return Address{}, err
	}
	var err error
	a.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return a, err
}

func scanObligation(row rowScanner) (Obligation, error) {
	var o Obligation
	var renewal, created string
	var cancelled int
	if err := row.Scan(&o.ID, &o.PersonaID, &o.AccountLinkID, &o.Description, &renewal, &o.CancelPath, &cancelled, &created); err != nil {
		return Obligation{}, err
	}
	var err error
	o.RenewalAt, err = time.Parse(time.RFC3339Nano, renewal)
	if err != nil {
		return Obligation{}, err
	}
	o.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	o.Cancelled = cancelled == 1
	return o, err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
