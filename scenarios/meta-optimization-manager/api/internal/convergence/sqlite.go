package convergence

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const fitnessTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the repository depends on (declared
// at the consumer per seam-discovery). Both *sql.DB (tests) and
// *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepo struct {
	db SQLExecutor
}

// NewSQLiteRepository constructs the production fitness-audit Repository.
func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepo{db: db} }

var _ Repository = (*sqliteRepo)(nil)

const (
	insertFitnessSQL = `INSERT INTO convergence_fitness
(template, per_replica_cost, drift_surfaces, comment_only_contracts, coordinated_edits, tier, captured_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`
	insertReferenceSQL = `INSERT INTO reference_health
(scenario, stale_from_template, last_template_sync, clean_on_all_tools, stability_days, breadth, eligibility, captured_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	trendAllSQL = `SELECT template, captured_at, per_replica_cost, coordinated_edits
FROM convergence_fitness ORDER BY captured_at, template`
	trendTemplateSQL = `SELECT template, captured_at, per_replica_cost, coordinated_edits
FROM convergence_fitness WHERE template = ? ORDER BY captured_at`
)

func (r *sqliteRepo) SaveFitness(ctx context.Context, fitness []TemplateFitness, at time.Time) error {
	ts := at.UTC().Format(fitnessTimeFormat)
	for _, tf := range fitness {
		if _, err := r.db.ExecContext(ctx, insertFitnessSQL,
			tf.Template, tf.PerReplicaCost, tf.DriftSurfaceCount, tf.CommentOnlyContractCount,
			tf.CoordinatedEditCount, int(tf.Tier), ts,
		); err != nil {
			return fmt.Errorf("insert fitness %q: %w", tf.Template, err)
		}
	}
	return nil
}

func (r *sqliteRepo) SaveReferences(ctx context.Context, refs []ReferenceHealth, at time.Time) error {
	ts := at.UTC().Format(fitnessTimeFormat)
	for _, h := range refs {
		sync := ""
		if !h.LastTemplateSync.IsZero() {
			sync = h.LastTemplateSync.UTC().Format(fitnessTimeFormat)
		}
		if _, err := r.db.ExecContext(ctx, insertReferenceSQL,
			h.Scenario, boolInt(h.StaleFromTemplate), sync, boolInt(h.CleanOnAllTools),
			h.StabilityDays, h.Breadth, int(h.Eligibility), ts,
		); err != nil {
			return fmt.Errorf("insert reference %q: %w", h.Scenario, err)
		}
	}
	return nil
}

func (r *sqliteRepo) Trend(ctx context.Context, template string) ([]FitnessTrendPoint, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if template == "" {
		rows, err = r.db.QueryContext(ctx, trendAllSQL)
	} else {
		rows, err = r.db.QueryContext(ctx, trendTemplateSQL, template)
	}
	if err != nil {
		return nil, fmt.Errorf("query trend: %w", err)
	}
	defer rows.Close()
	var out []FitnessTrendPoint
	for rows.Next() {
		var (
			p     FitnessTrendPoint
			atRaw string
		)
		if err := rows.Scan(&p.Template, &atRaw, &p.PerReplicaCost, &p.CoordinatedEditCount); err != nil {
			return nil, err
		}
		if t, perr := time.Parse(fitnessTimeFormat, atRaw); perr == nil {
			p.At = t
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trend: %w", err)
	}
	return out, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
