package focus

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/spacedoc"
)

// gapTimeFormat matches the coverage domain (RFC3339Nano sorts lexicographically
// in time order for a fixed zone).
const gapTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the gaps repository depends on
// (declared at the consumer per seam-discovery). Both *sql.DB (tests) and
// *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepo struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production gaps Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepo{db: db, clock: clk}
}

var _ Repository = (*sqliteRepo)(nil)

const (
	listGapsSQL = `SELECT id, projection, title, status, source_cell_id, global, notes, approaches, follow_ups
FROM gaps ORDER BY id`
	getGapSQL = `SELECT id, projection, title, status, source_cell_id, global, notes, approaches, follow_ups
FROM gaps WHERE id = ?`
	upsertGapSQL = `INSERT INTO gaps (id, projection, title, status, source_cell_id, global, notes, approaches, follow_ups, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  projection=excluded.projection,
  title=excluded.title,
  status=excluded.status,
  source_cell_id=excluded.source_cell_id,
  global=excluded.global,
  notes=excluded.notes,
  approaches=excluded.approaches,
  follow_ups=excluded.follow_ups,
  updated_at=excluded.updated_at`
)

func (r *sqliteRepo) List(ctx context.Context) ([]Gap, error) {
	rows, err := r.db.QueryContext(ctx, listGapsSQL)
	if err != nil {
		return nil, fmt.Errorf("list gaps: %w", err)
	}
	defer rows.Close()
	var out []Gap
	for rows.Next() {
		g, err := scanGap(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gaps: %w", err)
	}
	return out, nil
}

func (r *sqliteRepo) Get(ctx context.Context, id string) (Gap, bool, error) {
	g, err := scanGap(r.db.QueryRowContext(ctx, getGapSQL, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Gap{}, false, nil
	}
	if err != nil {
		return Gap{}, false, fmt.Errorf("get gap %q: %w", id, err)
	}
	return g, true, nil
}

func (r *sqliteRepo) Upsert(ctx context.Context, g Gap) error {
	now := r.clock.Now().UTC().Format(gapTimeFormat)
	globalInt := 0
	if g.Global {
		globalInt = 1
	}
	if _, err := r.db.ExecContext(ctx, upsertGapSQL,
		g.ID, string(g.Projection), g.Title, string(g.Status), g.SourceCellID, globalInt,
		mustJSON(g.Notes), mustJSON(g.Approaches), mustJSON(g.FollowUps), now, now,
	); err != nil {
		return fmt.Errorf("upsert gap %q: %w", g.ID, err)
	}
	return nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanGap(s rowScanner) (Gap, error) {
	var (
		g                             Gap
		projection, status            string
		globalInt                     int
		notesJSON, apprJSON, follJSON string
	)
	if err := s.Scan(&g.ID, &projection, &g.Title, &status, &g.SourceCellID, &globalInt, &notesJSON, &apprJSON, &follJSON); err != nil {
		return Gap{}, err
	}
	g.Projection = spacedoc.Projection(projection)
	g.Status = spacedoc.CellStatus(status)
	g.Global = globalInt != 0
	g.Notes = parseJSONArray(notesJSON)
	g.Approaches = parseJSONArray(apprJSON)
	g.FollowUps = parseJSONArray(follJSON)
	return g, nil
}

func mustJSON(v []string) string {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func parseJSONArray(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}
