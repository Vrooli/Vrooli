package rangedomain

import (
	"context"

	"extraction-range/internal/rangeengine"
	"github.com/vrooli/api-core/database"
)

// Repository is the persistence seam for durable range score receipts.
type Repository interface {
	SaveBest(context.Context, rangeengine.Result) error
	List(context.Context) ([]rangeengine.Result, error)
}

type sqlRepository struct{ db *database.RoutedDB }

// NewRepository creates the SQLite-backed implementation over the shared
// routed database. Context-aware methods preserve Test Genie's isolated pool.
func NewRepository(db *database.RoutedDB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) SaveBest(ctx context.Context, score rangeengine.Result) error {
	passed := 0
	if score.Passed {
		passed = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO extraction_range_scores(tool,level_id,family,correct,submitted,expected,precision,recall,f1,passed) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(tool,level_id) DO UPDATE SET family=excluded.family,correct=excluded.correct,submitted=excluded.submitted,expected=excluded.expected,precision=excluded.precision,recall=excluded.recall,f1=excluded.f1,passed=excluded.passed WHERE excluded.f1 > extraction_range_scores.f1 OR (excluded.f1 = extraction_range_scores.f1 AND excluded.precision > extraction_range_scores.precision)`, score.Tool, score.LevelID, score.Family, score.Correct, score.Submitted, score.Expected, score.Precision, score.Recall, score.F1, passed)
	return err
}

func (r *sqlRepository) List(ctx context.Context) ([]rangeengine.Result, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tool,level_id,family,correct,submitted,expected,precision,recall,f1,passed FROM extraction_range_scores ORDER BY tool,level_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rangeengine.Result{}
	for rows.Next() {
		var score rangeengine.Result
		var passed int
		if err := rows.Scan(&score.Tool, &score.LevelID, &score.Family, &score.Correct, &score.Submitted, &score.Expected, &score.Precision, &score.Recall, &score.F1, &passed); err != nil {
			return nil, err
		}
		score.Passed = passed != 0
		out = append(out, score)
	}
	return out, rows.Err()
}
