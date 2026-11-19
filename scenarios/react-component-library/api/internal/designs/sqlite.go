package designs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// timestampLayout is fixed-width UTC so lexical ordering matches time order
// and the value matches the JavaScript `Date.prototype.toISOString()` shape
// the client already persisted in localStorage.
const timestampLayout = "2006-01-02T15:04:05.000Z"

type sqliteRepository struct {
	db  *sql.DB
	now func() time.Time
}

// NewSQLiteRepository returns the SQLite-backed designs repository. The
// caller supplies the scenario's primary *sql.DB; designs share no state with
// the job pool because saves happen on the request path.
func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db, now: time.Now}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Summary, error) {
	query := `SELECT id, name, updated_at, frame_count, scenario, page FROM studio_designs`
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if scenario := strings.TrimSpace(filter.Scenario); scenario != "" {
		clauses = append(clauses, "scenario = ?")
		args = append(args, scenario)
	}
	if page := strings.TrimSpace(filter.Page); page != "" {
		clauses = append(clauses, "page = ?")
		args = append(args, page)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY updated_at DESC, rowid DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list designs: %w", err)
	}
	defer rows.Close()

	summaries := make([]Summary, 0)
	for rows.Next() {
		var (
			s        Summary
			scenario sql.NullString
			page     sql.NullString
		)
		if err := rows.Scan(&s.ID, &s.Name, &s.UpdatedAt, &s.FrameCount, &scenario, &page); err != nil {
			return nil, fmt.Errorf("scan design summary: %w", err)
		}
		s.Scenario = bindingFromSQL(scenario)
		s.Page = bindingFromSQL(page)
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate designs: %w", err)
	}
	return summaries, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Document, error) {
	return r.get(ctx, id)
}

func (r *sqliteRepository) Create(ctx context.Context, input CreateInput) (Document, error) {
	cleanName, err := normalizeName(input.Name)
	if err != nil {
		return Document{}, err
	}
	cleanFrames, frameCount, err := normalizeFrames(input.Frames)
	if err != nil {
		return Document{}, err
	}
	cleanTheme, err := normalizeTheme(input.Theme)
	if err != nil {
		return Document{}, err
	}
	cleanOptions, err := normalizeOptions(input.Options)
	if err != nil {
		return Document{}, err
	}
	scenario := normalizeBinding(input.Scenario)
	page := normalizeBinding(input.Page)

	id := "design-" + uuid.NewString()
	now := r.timestamp()
	if _, err := r.db.ExecContext(ctx, `INSERT INTO studio_designs (id, name, revision, frame_count, frames_json, theme_json, options_json, scenario, page, created_at, updated_at)
		VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, cleanName, frameCount, string(cleanFrames), string(cleanTheme), string(cleanOptions), bindingArg(scenario), bindingArg(page), now, now); err != nil {
		return Document{}, fmt.Errorf("create design: %w", err)
	}
	return r.get(ctx, id)
}

func (r *sqliteRepository) Save(ctx context.Context, id string, input UpdateInput) (Document, error) {
	cleanName, err := normalizeName(input.Name)
	if err != nil {
		return Document{}, err
	}
	cleanFrames, frameCount, err := normalizeFrames(input.Frames)
	if err != nil {
		return Document{}, err
	}
	cleanTheme, err := normalizeTheme(input.Theme)
	if err != nil {
		return Document{}, err
	}
	cleanOptions, err := normalizeOptions(input.Options)
	if err != nil {
		return Document{}, err
	}
	scenario := normalizeBinding(input.Scenario)
	page := normalizeBinding(input.Page)

	query := `UPDATE studio_designs
		SET name = ?, frames_json = ?, theme_json = ?, options_json = ?, frame_count = ?, scenario = ?, page = ?, revision = revision + 1, updated_at = ?
		WHERE id = ?`
	args := []any{cleanName, string(cleanFrames), string(cleanTheme), string(cleanOptions), frameCount, bindingArg(scenario), bindingArg(page), r.timestamp(), id}
	if input.Revision != nil {
		// Conditional in SQL so the compare-and-set is atomic with the write;
		// never read the revision, compare, then update.
		query += ` AND revision = ?`
		args = append(args, *input.Revision)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return Document{}, fmt.Errorf("save design: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Document{}, fmt.Errorf("save design rows affected: %w", err)
	}
	if affected == 0 {
		if input.Revision == nil {
			return Document{}, ErrNotFound
		}
		var stored int
		err := r.db.QueryRowContext(ctx, `SELECT revision FROM studio_designs WHERE id = ?`, id).Scan(&stored)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Document{}, ErrNotFound
		case err != nil:
			return Document{}, fmt.Errorf("inspect design revision: %w", err)
		default:
			return Document{}, ErrConflict
		}
	}
	return r.get(ctx, id)
}

func (r *sqliteRepository) Delete(ctx context.Context, id string) error {
	// Idempotent: deleting a design that is already gone is success, matching
	// the previous local store and the caller's fire-and-forget delete.
	// Assets are deleted in the same transaction so no row outlives its design.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete design: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM studio_design_assets WHERE design_id = ?`, id); err != nil {
		return fmt.Errorf("delete design assets: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM studio_designs WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete design: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete design commit: %w", err)
	}
	return nil
}

func (r *sqliteRepository) get(ctx context.Context, id string) (Document, error) {
	var (
		doc         Document
		framesJSON  string
		themeJSON   string
		optionsJSON string
		scenario    sql.NullString
		page        sql.NullString
	)
	err := r.db.QueryRowContext(ctx, `SELECT id, name, revision, updated_at, frames_json, theme_json, options_json, scenario, page FROM studio_designs WHERE id = ?`, id).
		Scan(&doc.ID, &doc.Name, &doc.Revision, &doc.UpdatedAt, &framesJSON, &themeJSON, &optionsJSON, &scenario, &page)
	if errors.Is(err, sql.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("get design %q: %w", id, err)
	}
	doc.Frames = json.RawMessage(framesJSON)
	doc.Theme = json.RawMessage(themeJSON)
	doc.Options = optionsFromSQL(optionsJSON)
	doc.Scenario = bindingFromSQL(scenario)
	doc.Page = bindingFromSQL(page)
	return doc, nil
}

// optionsFromSQL guarantees the wire shape is always an object. Rows written
// before the column existed read the column default; a damaged value degrades
// to {} rather than failing the read, because chrome is never load-bearing.
func optionsFromSQL(value string) json.RawMessage {
	trimmed := bytes.TrimSpace([]byte(value))
	if len(trimmed) == 0 || trimmed[0] != '{' || !json.Valid(trimmed) {
		return json.RawMessage("{}")
	}
	return json.RawMessage(trimmed)
}

// bindingFromSQL maps the nullable column to the wire shape: nil is unbound,
// including an empty string written by an older client.
func bindingFromSQL(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// bindingArg stores unbound as NULL so exact-match filters stay honest.
func bindingArg(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func (r *sqliteRepository) timestamp() string {
	now := r.now
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(timestampLayout)
}
