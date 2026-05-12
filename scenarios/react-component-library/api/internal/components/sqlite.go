package components

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"react-component-library/internal/clock"

	"github.com/google/uuid"
)

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct.
type sqliteRepository struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

// timeFormat matches the convention used by internal/notes: RFC3339
// with nanoseconds, parseable by time.Parse on read.
const timeFormat = time.RFC3339Nano

// tagSep is the in-row tag separator. Comma chosen for readability in
// SQL queries; tags themselves are forbidden from containing commas by
// the indexer's header parser.
const tagSep = ","

func (s *sqliteRepository) Upsert(ctx context.Context, in UpsertInput) (Component, error) {
	if strings.TrimSpace(in.LibraryID) == "" {
		return Component{}, ErrInvalidHeader{SourcePath: in.SourcePath, Field: "libraryId", Reason: "required"}
	}
	now := s.clock.Now().UTC()
	tagsCol := strings.Join(in.Tags, tagSep)

	existing, err := s.GetByLibraryID(ctx, in.LibraryID)
	if err != nil && !errors.As(err, &ErrComponentNotFound{}) {
		return Component{}, err
	}

	id := existing.ID
	indexedAt := existing.IndexedAt
	if id == "" {
		id = uuid.NewString()
		indexedAt = now
	}

	if _, err := s.db.ExecContext(ctx, `
INSERT INTO components (id, library_id, display_name, description, source_path, version, tags, indexed_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(library_id) DO UPDATE SET
  display_name = excluded.display_name,
  description  = excluded.description,
  source_path  = excluded.source_path,
  version      = excluded.version,
  tags         = excluded.tags,
  updated_at   = excluded.updated_at
`,
		id, in.LibraryID, in.DisplayName, in.Description, in.SourcePath, in.Version,
		tagsCol, indexedAt.Format(timeFormat), now.Format(timeFormat),
	); err != nil {
		return Component{}, fmt.Errorf("upsert component %q: %w", in.LibraryID, err)
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_headers WHERE component_id = ?`, id); err != nil {
		return Component{}, fmt.Errorf("clear headers for %q: %w", id, err)
	}
	for field, value := range in.Headers {
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_headers (component_id, field, value) VALUES (?, ?, ?)
`, id, field, value); err != nil {
			return Component{}, fmt.Errorf("insert header %s=%q for %q: %w", field, value, id, err)
		}
	}

	return s.Get(ctx, id)
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Component, error) {
	row := s.db.QueryRowContext(ctx, selectComponentByIDSQL, id)
	c, err := scanComponent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: id}
	}
	if err != nil {
		return Component{}, fmt.Errorf("get component %q: %w", id, err)
	}
	if err := s.loadHeaders(ctx, &c); err != nil {
		return Component{}, err
	}
	return c, nil
}

func (s *sqliteRepository) GetByLibraryID(ctx context.Context, libraryID string) (Component, error) {
	row := s.db.QueryRowContext(ctx, selectComponentByLibraryIDSQL, libraryID)
	c, err := scanComponent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: libraryID}
	}
	if err != nil {
		return Component{}, fmt.Errorf("get component by libraryId %q: %w", libraryID, err)
	}
	if err := s.loadHeaders(ctx, &c); err != nil {
		return Component{}, err
	}
	return c, nil
}

func (s *sqliteRepository) List(ctx context.Context, q SearchQuery) ([]Component, error) {
	limit := q.Limit
	if limit <= 0 {
		return nil, nil
	}
	var (
		clauses []string
		args    []any
	)
	if match := strings.TrimSpace(q.Match); match != "" {
		pat := "%" + strings.ToLower(match) + "%"
		clauses = append(clauses, `(
		  lower(library_id)   LIKE ? OR
		  lower(display_name) LIKE ? OR
		  lower(description)  LIKE ? OR
		  lower(source_path)  LIKE ?
		)`)
		args = append(args, pat, pat, pat, pat)
	}
	if tag := strings.TrimSpace(q.Tag); tag != "" {
		// Match the tag as a token within the comma-joined column.
		clauses = append(clauses, `(',' || lower(tags) || ',') LIKE ?`)
		args = append(args, "%,"+strings.ToLower(tag)+",%")
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	query := fmt.Sprintf(`
SELECT id, library_id, display_name, description, source_path, version, tags, indexed_at, updated_at
FROM components
%s
ORDER BY indexed_at DESC, library_id ASC
LIMIT ?
`, where)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list components: %w", err)
	}
	defer rows.Close()

	var out []Component
	for rows.Next() {
		c, err := scanComponent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan component: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate components: %w", err)
	}
	for i := range out {
		if err := s.loadHeaders(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *sqliteRepository) DeleteMissing(ctx context.Context, keep []string) (int, error) {
	if len(keep) == 0 {
		res, err := s.db.ExecContext(ctx, `DELETE FROM components`)
		if err != nil {
			return 0, fmt.Errorf("delete all components: %w", err)
		}
		n, _ := res.RowsAffected()
		return int(n), nil
	}
	placeholders := strings.Repeat("?,", len(keep))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(keep))
	for i, k := range keep {
		args[i] = k
	}
	res, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`DELETE FROM components WHERE library_id NOT IN (%s)`, placeholders), args...)
	if err != nil {
		return 0, fmt.Errorf("delete missing components: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *sqliteRepository) loadHeaders(ctx context.Context, c *Component) error {
	rows, err := s.db.QueryContext(ctx, `SELECT field, value FROM component_headers WHERE component_id = ?`, c.ID)
	if err != nil {
		return fmt.Errorf("load headers for %q: %w", c.ID, err)
	}
	defer rows.Close()
	headers := map[string]string{}
	for rows.Next() {
		var f, v string
		if err := rows.Scan(&f, &v); err != nil {
			return fmt.Errorf("scan header for %q: %w", c.ID, err)
		}
		headers[f] = v
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate headers for %q: %w", c.ID, err)
	}
	c.Headers = headers
	return nil
}

const (
	selectComponentByIDSQL = `
SELECT id, library_id, display_name, description, source_path, version, tags, indexed_at, updated_at
FROM components WHERE id = ?
`
	selectComponentByLibraryIDSQL = `
SELECT id, library_id, display_name, description, source_path, version, tags, indexed_at, updated_at
FROM components WHERE library_id = ?
`
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanComponent(s rowScanner) (Component, error) {
	var (
		c          Component
		tagsRaw    string
		indexedRaw string
		updatedRaw string
	)
	if err := s.Scan(&c.ID, &c.LibraryID, &c.DisplayName, &c.Description, &c.SourcePath, &c.Version, &tagsRaw, &indexedRaw, &updatedRaw); err != nil {
		return Component{}, err
	}
	if tagsRaw != "" {
		c.Tags = strings.Split(tagsRaw, tagSep)
	}
	indexed, err := time.Parse(timeFormat, indexedRaw)
	if err != nil {
		return Component{}, fmt.Errorf("parse indexed_at %q: %w", indexedRaw, err)
	}
	updated, err := time.Parse(timeFormat, updatedRaw)
	if err != nil {
		return Component{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	c.IndexedAt = indexed
	c.UpdatedAt = updated
	return c, nil
}
