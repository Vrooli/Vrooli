package personas

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

var _ Repository = (*sqliteRepository)(nil)

const timeFormat = time.RFC3339Nano

func (r *sqliteRepository) Create(ctx context.Context, p Persona) (Persona, error) {
	p.ID = uuid.NewString()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = r.clock.Now().UTC()
	}
	if p.Status == "" {
		p.Status = StatusActive
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO personas (id, kind, legal_subject_id, legal_subject_name, legal_basis_type, display_name, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, p.ID, p.Kind, p.LegalBasis.SubjectID, p.LegalBasis.SubjectName, p.LegalBasis.BasisType, p.DisplayName, p.Status, p.CreatedAt.Format(timeFormat))
	if err != nil {
		return Persona{}, fmt.Errorf("insert persona: %w", err)
	}
	for _, identifier := range p.Identifiers {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO persona_identifiers (persona_id, identifier_type, identifier_value) VALUES (?, ?, ?)`, p.ID, identifier.Type, identifier.Value); err != nil {
			return Persona{}, fmt.Errorf("insert persona identifier: %w", err)
		}
	}
	return p, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Persona, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, kind, legal_subject_id, legal_subject_name, legal_basis_type, display_name, status, created_at, archived_at FROM personas WHERE id = ?`, id)
	p, err := scanPersona(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Persona{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if err != nil {
		return Persona{}, fmt.Errorf("get persona: %w", err)
	}
	p.Identifiers, err = r.listIdentifiers(ctx, id)
	if err != nil {
		return Persona{}, err
	}
	return p, nil
}

func (r *sqliteRepository) List(ctx context.Context, includeArchived bool, limit int) ([]Persona, error) {
	query := `SELECT id, kind, legal_subject_id, legal_subject_name, legal_basis_type, display_name, status, created_at, archived_at FROM personas`
	args := []any{}
	if !includeArchived {
		query += ` WHERE status = 'active'`
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list personas: %w", err)
	}
	defer rows.Close()
	var out []Persona
	for rows.Next() {
		p, err := scanPersona(rows)
		if err != nil {
			return nil, fmt.Errorf("scan persona: %w", err)
		}
		p.Identifiers, err = r.listIdentifiers(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate personas: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) Archive(ctx context.Context, id string) (Persona, error) {
	at := r.clock.Now().UTC()
	result, err := r.db.ExecContext(ctx, `UPDATE personas SET status = 'archived', archived_at = ? WHERE id = ? AND status = 'active'`, at.Format(timeFormat), id)
	if err != nil {
		return Persona{}, fmt.Errorf("archive persona: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return Persona{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return r.Get(ctx, id)
}

func (r *sqliteRepository) listIdentifiers(ctx context.Context, id string) ([]Identifier, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT identifier_type, identifier_value FROM persona_identifiers WHERE persona_id = ? ORDER BY identifier_type`, id)
	if err != nil {
		return nil, fmt.Errorf("list persona identifiers: %w", err)
	}
	defer rows.Close()
	var out []Identifier
	for rows.Next() {
		var i Identifier
		if err := rows.Scan(&i.Type, &i.Value); err != nil {
			return nil, fmt.Errorf("scan persona identifier: %w", err)
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanPersona(row rowScanner) (Persona, error) {
	var p Persona
	var kind, status, created string
	var archived sql.NullString
	if err := row.Scan(&p.ID, &kind, &p.LegalBasis.SubjectID, &p.LegalBasis.SubjectName, &p.LegalBasis.BasisType, &p.DisplayName, &status, &created, &archived); err != nil {
		return Persona{}, err
	}
	p.Kind = Kind(kind)
	p.Status = Status(status)
	p.CreatedAt, _ = time.Parse(timeFormat, created)
	if archived.Valid && archived.String != "" {
		at, err := time.Parse(timeFormat, archived.String)
		if err != nil {
			return Persona{}, fmt.Errorf("parse archived_at: %w", err)
		}
		p.ArchivedAt = &at
	}
	return p, nil
}
