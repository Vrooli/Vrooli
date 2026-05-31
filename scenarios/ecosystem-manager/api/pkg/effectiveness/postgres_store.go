package effectiveness

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ecosystem-manager/api/pkg/dimensions"
)

// CreateSchema creates the skill_dimension_effectiveness table. It is idempotent
// and self-healing (CREATE TABLE IF NOT EXISTS), and is invoked from the
// controller's startup DDL (pkg/autosteer.EnsureTablesExist) so the ledger lives
// alongside the other controller tables.
func CreateSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	query := `
	CREATE TABLE IF NOT EXISTS skill_dimension_effectiveness (
		skill_id TEXT NOT NULL,
		dimension TEXT NOT NULL,
		closed_count BIGINT NOT NULL DEFAULT 0,
		introduced_count BIGINT NOT NULL DEFAULT 0,
		total_runs BIGINT NOT NULL DEFAULT 0,
		total_tokens BIGINT NOT NULL DEFAULT 0,
		last_run_at TIMESTAMPTZ NOT NULL DEFAULT to_timestamp(0),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (skill_id, dimension)
	);
	`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to ensure skill_dimension_effectiveness table: %w", err)
	}
	return nil
}

// PostgresStore is the durable effectiveness ledger.
type PostgresStore struct {
	db  *sql.DB
	now func() time.Time
}

var _ Store = (*PostgresStore)(nil)

// NewPostgresStore creates a Postgres-backed ledger.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db, now: time.Now}
}

const upsertStmt = `
	INSERT INTO skill_dimension_effectiveness
		(skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$7)
	ON CONFLICT (skill_id, dimension) DO UPDATE SET
		closed_count     = skill_dimension_effectiveness.closed_count + EXCLUDED.closed_count,
		introduced_count = skill_dimension_effectiveness.introduced_count + EXCLUDED.introduced_count,
		total_runs       = skill_dimension_effectiveness.total_runs + EXCLUDED.total_runs,
		total_tokens     = skill_dimension_effectiveness.total_tokens + EXCLUDED.total_tokens,
		last_run_at      = GREATEST(skill_dimension_effectiveness.last_run_at, EXCLUDED.last_run_at),
		updated_at       = EXCLUDED.updated_at
`

// Record applies one credit event atomically. Each affected (skill, dimension)
// pair is upserted with commutative increments, so concurrent steered tasks
// accumulate correctly.
func (p *PostgresStore) Record(ev CreditEvent) error {
	if p == nil || p.db == nil || ev.SkillID == "" {
		return nil
	}
	now := p.now()

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
		if _, err := tx.Exec(upsertStmt, ev.SkillID, string(dim), closed, introduced, runs, tokens, now); err != nil {
			return fmt.Errorf("upsert effectiveness (%s, %s): %w", ev.SkillID, dim, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit effectiveness tx: %w", err)
	}
	return nil
}

// Get implements Store.
func (p *PostgresStore) Get(skillID string, dim dimensions.Dimension) (Stat, bool, error) {
	if p == nil || p.db == nil {
		return Stat{}, false, nil
	}
	row := p.db.QueryRow(`
		SELECT skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at
		FROM skill_dimension_effectiveness
		WHERE skill_id = $1 AND dimension = $2
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
func (p *PostgresStore) Bulk(dim dimensions.Dimension) (map[string]Stat, error) {
	if p == nil || p.db == nil {
		return map[string]Stat{}, nil
	}
	rows, err := p.db.Query(`
		SELECT skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at
		FROM skill_dimension_effectiveness
		WHERE dimension = $1
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
func (p *PostgresStore) List(skillID string, dim dimensions.Dimension) ([]Stat, error) {
	if p == nil || p.db == nil {
		return nil, nil
	}
	rows, err := p.db.Query(`
		SELECT skill_id, dimension, closed_count, introduced_count, total_runs, total_tokens, last_run_at, updated_at
		FROM skill_dimension_effectiveness
		WHERE ($1 = '' OR skill_id = $1) AND ($2 = '' OR dimension = $2)
		ORDER BY dimension ASC, skill_id ASC
	`, skillID, string(dim))
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
