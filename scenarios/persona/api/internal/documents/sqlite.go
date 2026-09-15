package documents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (r *sqliteRepository) Create(ctx context.Context, b Binding) (Binding, error) {
	b.ID = uuid.NewString()
	b.CreatedAt = r.clock.Now().UTC()
	var valid any
	if !b.ValidUntil.IsZero() {
		valid = b.ValidUntil.UTC().Format(time.RFC3339Nano)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO persona_document_bindings (id, persona_id, document_id, document_kind, valid_until, created_at) VALUES (?, ?, ?, ?, ?, ?)`, b.ID, b.PersonaID, b.DocumentID, b.DocumentKind, valid, b.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Binding{}, fmt.Errorf("insert document binding: %w", err)
	}
	return b, nil
}

func (r *sqliteRepository) List(ctx context.Context, personaID string) ([]Binding, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, document_id, document_kind, valid_until, created_at FROM persona_document_bindings WHERE persona_id = ? ORDER BY created_at DESC, id DESC`, personaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Binding
	for rows.Next() {
		b, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, personaID, documentID string) (Binding, error) {
	b, err := scanBinding(r.db.QueryRowContext(ctx, `SELECT id, persona_id, document_id, document_kind, valid_until, created_at FROM persona_document_bindings WHERE persona_id = ? AND document_id = ?`, personaID, documentID))
	if errors.Is(err, sql.ErrNoRows) {
		return Binding{}, ErrBindingNotFound
	}
	return b, err
}

func (r *sqliteRepository) CreateRelease(ctx context.Context, release Release) (Release, error) {
	if release.ID == "" {
		release.ID = uuid.NewString()
	}
	if release.ReleasedAt.IsZero() {
		release.ReleasedAt = r.clock.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO persona_document_releases (id, persona_id, handoff_id, document_id, released_at) VALUES (?, ?, ?, ?, ?)`, release.ID, release.PersonaID, release.HandoffID, release.DocumentID, release.ReleasedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Release{}, fmt.Errorf("insert document release: %w", err)
	}
	return release, nil
}

func (r *sqliteRepository) GetRelease(ctx context.Context, handoffID, documentID string) (Release, error) {
	var release Release
	var releasedAt string
	err := r.db.QueryRowContext(ctx, `SELECT id, persona_id, handoff_id, document_id, released_at FROM persona_document_releases WHERE handoff_id = ? AND document_id = ?`, handoffID, documentID).Scan(&release.ID, &release.PersonaID, &release.HandoffID, &release.DocumentID, &releasedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Release{}, ErrReleaseNotFound
	}
	if err != nil {
		return Release{}, err
	}
	release.ReleasedAt, err = time.Parse(time.RFC3339Nano, releasedAt)
	return release, err
}

type rowScanner interface{ Scan(...any) error }

func scanBinding(row rowScanner) (Binding, error) {
	var b Binding
	var valid, created sql.NullString
	if err := row.Scan(&b.ID, &b.PersonaID, &b.DocumentID, &b.DocumentKind, &valid, &created); err != nil {
		return Binding{}, err
	}
	var err error
	if valid.Valid {
		b.ValidUntil, err = time.Parse(time.RFC3339Nano, valid.String)
		if err != nil {
			return Binding{}, err
		}
	}
	b.CreatedAt, err = time.Parse(time.RFC3339Nano, created.String)
	return b, err
}
