package notes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"template-manager/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (used by
// repository unit tests via testutil/db.NewSQLite) and
// *database.RoutedDB (used in production by main.go) satisfy it, so the
// production wiring participates in per-request routing without forcing
// the test fixture to wrap its handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct (seam-discovery §4).
type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository. db is the
// connection pool opened in main.go (*database.RoutedDB in production,
// *sql.DB in unit tests via testutil/db.NewSQLite); clk supplies
// CreatedAt/UpdatedAt timestamps so tests can advance time
// deterministically via mocks.FakeClock.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

func NewSQLiteAttachmentsRepository(db SQLExecutor, clk clock.Clock) AttachmentsRepository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

const (
	// RFC3339 with nanosecond precision matches the format produced by
	// time.Time.Format(time.RFC3339Nano) — the same format the wire
	// (Note.CreatedAt / Note.UpdatedAt strings) uses.
	noteTimeFormat = time.RFC3339Nano

	insertNoteSQL = `
INSERT INTO notes (id, title, body, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
`

	selectNoteByIDSQL = `
SELECT n.id, n.title, n.body, n.created_at, n.updated_at,
       COALESCE((
         SELECT group_concat(a.key, char(31))
         FROM attachments a
         WHERE a.note_id = n.id
         ORDER BY a.uploaded_at DESC, a.key DESC
       ), '')
FROM notes n
WHERE n.id = ?
`

	listNotesSQL = `
SELECT n.id, n.title, n.body, n.created_at, n.updated_at,
       COALESCE((
         SELECT group_concat(a.key, char(31))
         FROM attachments a
         WHERE a.note_id = n.id
         ORDER BY a.uploaded_at DESC, a.key DESC
       ), '')
FROM notes n
ORDER BY n.created_at DESC, n.id DESC
LIMIT ?
`

	// countNotesInWindowSQL counts notes whose created_at is in the
	// half-open range [from, to). created_at is stored as RFC3339Nano
	// (noteTimeFormat); that format sorts lexicographically in time order
	// for a fixed zone, so a string range comparison is a correct filter.
	countNotesInWindowSQL = `
SELECT COUNT(*)
FROM notes n
WHERE n.created_at >= ? AND n.created_at < ?
`
)

func (s *sqliteRepository) Create(ctx context.Context, n Note) (Note, error) {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = s.clock.Now().UTC()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}

	_, err := s.db.ExecContext(ctx, insertNoteSQL,
		n.ID,
		n.Title,
		n.Body,
		n.CreatedAt.Format(noteTimeFormat),
		n.UpdatedAt.Format(noteTimeFormat),
	)
	if err != nil {
		return Note{}, fmt.Errorf("insert note %q: %w", n.ID, err)
	}
	return n, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Note, error) {
	row := s.db.QueryRowContext(ctx, selectNoteByIDSQL, id)
	n, err := scanNote(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, ErrNoteNotFound{ID: id}
	}
	if err != nil {
		return Note{}, fmt.Errorf("get note %q: %w", id, err)
	}
	return n, nil
}

func (s *sqliteRepository) List(ctx context.Context, limit int) ([]Note, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, listNotesSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notes: %w", err)
	}
	return notes, nil
}

func (s *sqliteRepository) Count(ctx context.Context, from, to time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, countNotesInWindowSQL,
		from.UTC().Format(noteTimeFormat),
		to.UTC().Format(noteTimeFormat),
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count notes in [%s, %s): %w", from, to, err)
	}
	return n, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan
// surface so scanNote works for both single-row Get and multi-row List.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanNote(s rowScanner) (Note, error) {
	var (
		n          Note
		createdRaw string
		updatedRaw string
		keysRaw    string
	)
	if err := s.Scan(&n.ID, &n.Title, &n.Body, &createdRaw, &updatedRaw, &keysRaw); err != nil {
		return Note{}, err
	}
	created, err := time.Parse(noteTimeFormat, createdRaw)
	if err != nil {
		return Note{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	updated, err := time.Parse(noteTimeFormat, updatedRaw)
	if err != nil {
		return Note{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	n.CreatedAt = created
	n.UpdatedAt = updated
	if keysRaw != "" {
		n.AttachmentKeys = strings.Split(keysRaw, string(rune(31)))
	}
	return n, nil
}
