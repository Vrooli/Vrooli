package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"{{SCENARIO_ID}}/internal/clock"
)

// sqliteNoteStore is the production NoteStore impl. Unexported so
// callers depend on the NoteStore interface — tests substitute the
// fake without reaching inside the struct (seam-discovery §4).
type sqliteNoteStore struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteNoteStore constructs the production NoteStore. db is the
// connection pool opened in main.go; clk supplies CreatedAt/UpdatedAt
// timestamps so tests can advance time deterministically via
// mocks.FakeClock.
func NewSQLiteNoteStore(db *sql.DB, clk clock.Clock) NoteStore {
	return &sqliteNoteStore{db: db, clock: clk}
}

// Compile-time guarantee.
var _ NoteStore = (*sqliteNoteStore)(nil)

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
SELECT id, title, body, created_at, updated_at
FROM notes
WHERE id = ?
`

	listNotesSQL = `
SELECT id, title, body, created_at, updated_at
FROM notes
ORDER BY created_at DESC, id DESC
LIMIT ?
`
)

func (s *sqliteNoteStore) Create(ctx context.Context, n Note) (Note, error) {
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

func (s *sqliteNoteStore) Get(ctx context.Context, id string) (Note, error) {
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

func (s *sqliteNoteStore) List(ctx context.Context, limit int) ([]Note, error) {
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

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan
// surface so scanNote works for both single-row Get and multi-row List.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanNote(s rowScanner) (Note, error) {
	var (
		n             Note
		createdRaw    string
		updatedRaw    string
	)
	if err := s.Scan(&n.ID, &n.Title, &n.Body, &createdRaw, &updatedRaw); err != nil {
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
	return n, nil
}
