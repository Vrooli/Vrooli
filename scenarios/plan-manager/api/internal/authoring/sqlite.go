package authoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"plan-manager/internal/clock"
)

// sessionTimeFormat matches the rest of the scenario (RFC3339Nano sorts
// lexicographically in time order for a fixed zone).
const sessionTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the sessions repository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (tests, via
// testutil/db.NewSQLite) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteStore struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteStore constructs the production authoring SessionStore.
func NewSQLiteStore(db SQLExecutor, clk clock.Clock) SessionStore {
	return &sqliteStore{db: db, clock: clk}
}

var _ SessionStore = (*sqliteStore)(nil)

// sessionDocument is the JSON payload stored in authoring_sessions.document —
// every structured field that isn't a first-class queryable column. The ordered
// sections[] and the current-section pointer live here because they round-trip
// with the session and are never queried across sessions.
type sessionDocument struct {
	Sections          []Section  `json:"sections"`
	CurrentSectionKey SectionKey `json:"current_section_key"`
}

const (
	upsertSessionSQL = `
INSERT INTO authoring_sessions (id, title, slug, finalized, plan_id, document, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  title=excluded.title,
  slug=excluded.slug,
  finalized=excluded.finalized,
  plan_id=excluded.plan_id,
  document=excluded.document,
  updated_at=excluded.updated_at`

	getSessionSQL = `
SELECT id, title, slug, finalized, plan_id, document, created_at, updated_at
FROM authoring_sessions WHERE id = ? LIMIT 1`
)

func (r *sqliteStore) Save(ctx context.Context, s Session) error {
	doc := sessionDocument{Sections: s.Sections, CurrentSectionKey: s.CurrentSectionKey}
	raw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal session document %q: %w", s.ID, err)
	}
	created := s.CreatedAt
	if created == "" {
		created = r.clock.Now().UTC().Format(sessionTimeFormat)
	}
	updated := s.UpdatedAt
	if updated == "" {
		updated = r.clock.Now().UTC().Format(sessionTimeFormat)
	}
	finalized := 0
	if s.Finalized {
		finalized = 1
	}
	if _, err := r.db.ExecContext(ctx, upsertSessionSQL,
		s.ID, s.Title, s.Slug, finalized, s.PlanID, string(raw), created, updated,
	); err != nil {
		return fmt.Errorf("upsert authoring session %q: %w", s.ID, err)
	}
	return nil
}

func (r *sqliteStore) Get(ctx context.Context, id string) (Session, bool, error) {
	s, err := scanSession(r.db.QueryRowContext(ctx, getSessionSQL, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, fmt.Errorf("get authoring session %q: %w", id, err)
	}
	return s, true, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(sc rowScanner) (Session, error) {
	var (
		s         Session
		finalized int
		document  string
	)
	if err := sc.Scan(&s.ID, &s.Title, &s.Slug, &finalized, &s.PlanID, &document, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return Session{}, err
	}
	s.Finalized = finalized != 0
	var doc sessionDocument
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		return Session{}, fmt.Errorf("unmarshal session document %q: %w", s.ID, err)
	}
	s.Sections = doc.Sections
	s.CurrentSectionKey = doc.CurrentSectionKey
	return s, nil
}
