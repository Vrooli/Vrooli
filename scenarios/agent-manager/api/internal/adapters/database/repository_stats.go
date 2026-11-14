package database

import (
	"context"
	"math"
	"time"

	"agent-manager/internal/repository"

	"github.com/sirupsen/logrus"
)

// statsRepository ranks models by run usage from the durable invocation read
// model for the pricing model picker. Run analytics are Measures procedures.
type statsRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.StatsRepository = (*statsRepository)(nil)

// nearestRank returns the nearest-rank percentile of an ascending slice.
func nearestRank(sorted []int64, percentile float64) int64 {
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// GetPopularModels returns the most used models by run count within a time window.
func (r *statsRepository) GetPopularModels(ctx context.Context, since time.Time, limit int) ([]string, error) {
	args := []interface{}{SQLiteTime(since), limit}

	query := `
		SELECT model
		FROM invocation_read_model_runs
		WHERE created_at >= ?
		  AND model != ''
		  AND model != 'unknown'
		GROUP BY model
		ORDER BY COUNT(*) DESC
		LIMIT ?`

	var models []string
	if err := r.db.SelectContext(ctx, &models, query, args...); err != nil {
		return nil, wrapDBError("get_popular_models", "Stats", "", err)
	}
	return models, nil
}

// GetRecentModels returns recently used models ordered by most recent use (system-wide).
func (r *statsRepository) GetRecentModels(ctx context.Context, limit int) ([]string, error) {
	args := []interface{}{limit}

	// SQLite: Group by model, get max created_at, order by that
	query := `
		SELECT model
		FROM invocation_read_model_runs
		WHERE model != ''
		  AND model != 'unknown'
		GROUP BY model
		ORDER BY MAX(created_at) DESC
		LIMIT ?`

	var models []string
	if err := r.db.SelectContext(ctx, &models, query, args...); err != nil {
		return nil, wrapDBError("get_recent_models", "Stats", "", err)
	}
	return models, nil
}
