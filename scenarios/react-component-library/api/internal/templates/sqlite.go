package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// timestampLayout is fixed-width UTC so lexical ordering matches time order
// and the value matches the JavaScript `Date.prototype.toISOString()` shape.
const timestampLayout = "2006-01-02T15:04:05.000Z"

const templateColumns = `id, name, tagline, category, palette_json, display, radius, spec_json, created_at, updated_at`

type sqliteRepository struct {
	db  *sql.DB
	now func() time.Time
}

// NewSQLiteRepository returns the SQLite-backed templates repository. The
// caller supplies the scenario's primary *sql.DB; shelf writes happen on the
// request path and share no state with the job pool.
func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db, now: time.Now}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) List(ctx context.Context) ([]Template, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+templateColumns+` FROM studio_templates
		ORDER BY updated_at DESC, rowid DESC`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	templates := make([]Template, 0)
	for rows.Next() {
		template, err := scanTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, template)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate templates: %w", err)
	}
	return templates, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Template, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+templateColumns+` FROM studio_templates WHERE id = ?`, id)
	template, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("get template %q: %w", id, err)
	}
	return template, nil
}

func (r *sqliteRepository) Create(ctx context.Context, input Input) (Template, error) {
	if err := validateID(input.ID); err != nil {
		return Template{}, err
	}
	clean, err := normalize(input)
	if err != nil {
		return Template{}, err
	}
	specDocument, err := clean.specDocument()
	if err != nil {
		return Template{}, err
	}

	now := r.timestamp()
	// ON CONFLICT DO NOTHING makes the duplicate-id check atomic without a
	// read-then-insert race.
	result, err := r.db.ExecContext(ctx, `INSERT INTO studio_templates
		(id, name, tagline, category, palette_json, display, radius, spec_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING`,
		input.ID, clean.name, clean.tagline, clean.category, string(clean.palette),
		clean.display, clean.radius, string(specDocument), now, now)
	if err != nil {
		return Template{}, fmt.Errorf("create template: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Template{}, fmt.Errorf("create template rows affected: %w", err)
	}
	if affected == 0 {
		return Template{}, ErrConflict
	}
	return r.Get(ctx, input.ID)
}

func (r *sqliteRepository) Update(ctx context.Context, id string, input Input) (Template, error) {
	if err := validateID(id); err != nil {
		return Template{}, err
	}
	clean, err := normalize(input)
	if err != nil {
		return Template{}, err
	}
	specDocument, err := clean.specDocument()
	if err != nil {
		return Template{}, err
	}

	result, err := r.db.ExecContext(ctx, `UPDATE studio_templates
		SET name = ?, tagline = ?, category = ?, palette_json = ?, display = ?, radius = ?,
		    spec_json = ?, updated_at = ?
		WHERE id = ?`,
		clean.name, clean.tagline, clean.category, string(clean.palette),
		clean.display, clean.radius, string(specDocument), r.timestamp(), id)
	if err != nil {
		return Template{}, fmt.Errorf("update template: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Template{}, fmt.Errorf("update template rows affected: %w", err)
	}
	if affected == 0 {
		return Template{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	// Idempotent: deleting a template that is already gone is success, matching
	// the client's fire-and-forget delete and the previous local store.
	if _, err := r.db.ExecContext(ctx, `DELETE FROM studio_templates WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(scanner rowScanner) (Template, error) {
	var (
		template Template
		palette  string
		spec     string
	)
	if err := scanner.Scan(
		&template.ID, &template.Name, &template.Tagline, &template.Category,
		&palette, &template.Display, &template.Radius, &spec,
		&template.CreatedAt, &template.UpdatedAt,
	); err != nil {
		return Template{}, err
	}
	template.Palette = validJSONObject(json.RawMessage(palette))
	template.Spec, template.SlotHint, template.Idea = decodeSpecDocument(spec)
	return template, nil
}

func (r *sqliteRepository) timestamp() string {
	now := r.now
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(timestampLayout)
}
