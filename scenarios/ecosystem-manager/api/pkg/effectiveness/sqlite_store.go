package effectiveness

import (
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"github.com/vrooli/maturity-go/dimensions"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the effectiveness ledger's SQLite DDL for registration with
// database.EnsureSchemas (see pkg/dbschema). It owns skill_dimension_effectiveness.
func Schema() string { return schemaSQL }

// SQLiteStore is the durable effectiveness ledger backed by SQLite.
type SQLiteStore struct {
	db  *sql.DB
	now func() time.Time
}

var _ Store = (*SQLiteStore)(nil)

// NewSQLiteStore creates a SQLite-backed ledger.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db, now: time.Now}
}

const upsertStmt = `
	INSERT INTO skill_dimension_effectiveness
		(skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at)
	VALUES (?,?,?,?,?,?,?,?)
	ON CONFLICT (skill_id, dimension) DO UPDATE SET
		closed_count     = skill_dimension_effectiveness.closed_count + excluded.closed_count,
		introduced_count = skill_dimension_effectiveness.introduced_count + excluded.introduced_count,
		total_runs       = skill_dimension_effectiveness.total_runs + excluded.total_runs,
		total_tokens     = skill_dimension_effectiveness.total_tokens + excluded.total_tokens,
		last_run_at      = max(skill_dimension_effectiveness.last_run_at, excluded.last_run_at),
		updated_at       = excluded.updated_at
`

// Record applies one credit event atomically. Each affected (skill, dimension)
// pair is upserted with commutative increments, so concurrent steered tasks
// accumulate correctly.
func (p *SQLiteStore) Record(ev CreditEvent) error {
	if p == nil || p.db == nil || ev.SkillID == "" {
		return nil
	}
	// UTC so the SQL max(last_run_at, ...) lexical comparison matches
	// chronological order regardless of host timezone.
	now := p.now().UTC()

	// Union of dimensions touched by this event: the target (always, to carry the
	// run count + token cost) plus every dimension with a closed/introduced delta.
	dims := map[dimensions.Dimension]bool{ev.TargetDimension: true}
	for d := range ev.ClosedByDimension {
		dims[d] = true
	}
	for d := range ev.IntroducedByDimension {
		dims[d] = true
	}

	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("begin effectiveness tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for dim := range dims {
		if dim == "" {
			continue
		}
		var runs, tokens int64
		if dim == ev.TargetDimension {
			runs = 1
			tokens = ev.Tokens
		}
		closed := int64(ev.ClosedByDimension[dim])
		introduced := int64(ev.IntroducedByDimension[dim])
		if _, err := tx.Exec(upsertStmt, ev.SkillID, string(dim), closed, introduced, runs, tokens, now, now); err != nil {
			return fmt.Errorf("upsert effectiveness (%s, %s): %w", ev.SkillID, dim, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit effectiveness tx: %w", err)
	}
	return nil
}

// Get implements Store.
func (p *SQLiteStore) Get(skillID string, dim dimensions.Dimension) (Stat, bool, error) {
	if p == nil || p.db == nil {
		return Stat{}, false, nil
	}
	row := p.db.QueryRow(`
		SELECT skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at
		FROM skill_dimension_effectiveness
		WHERE skill_id = ? AND dimension = ?
	`, skillID, string(dim))
	s, err := scanStat(row)
	if err == sql.ErrNoRows {
		return Stat{}, false, nil
	}
	if err != nil {
		return Stat{}, false, fmt.Errorf("get effectiveness: %w", err)
	}
	return s, true, nil
}

// Bulk implements Store.
func (p *SQLiteStore) Bulk(dim dimensions.Dimension) (map[string]Stat, error) {
	if p == nil || p.db == nil {
		return map[string]Stat{}, nil
	}
	rows, err := p.db.Query(`
		SELECT skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at
		FROM skill_dimension_effectiveness
		WHERE dimension = ?
	`, string(dim))
	if err != nil {
		return nil, fmt.Errorf("bulk effectiveness: %w", err)
	}
	defer rows.Close()

	out := make(map[string]Stat)
	for rows.Next() {
		s, err := scanStat(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bulk effectiveness: %w", err)
		}
		out[s.SkillID] = s
	}
	return out, rows.Err()
}

// List implements Store with optional skill/dimension filters.
func (p *SQLiteStore) List(skillID string, dim dimensions.Dimension) ([]Stat, error) {
	if p == nil || p.db == nil {
		return nil, nil
	}
	rows, err := p.db.Query(`
		SELECT skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at
		FROM skill_dimension_effectiveness
		WHERE (? = '' OR skill_id = ?) AND (? = '' OR dimension = ?)
		ORDER BY dimension ASC, skill_id ASC
	`, skillID, skillID, string(dim), string(dim))
	if err != nil {
		return nil, fmt.Errorf("list effectiveness: %w", err)
	}
	defer rows.Close()

	out := make([]Stat, 0)
	for rows.Next() {
		s, err := scanStat(rows)
		if err != nil {
			return nil, fmt.Errorf("scan list effectiveness: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// rowScanner abstracts *sql.Row and *sql.Rows for shared scanning.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanStat(sc rowScanner) (Stat, error) {
	var s Stat
	var dim string
	if err := sc.Scan(&s.SkillID, &dim, &s.ClosedCount, &s.IntroducedCount,
		&s.TotalRuns, &s.TotalTokens, &s.LastRunAt, &s.UpdatedAt); err != nil {
		return Stat{}, err
	}
	s.Dimension = dimensions.Dimension(dim)
	return s, nil
}
