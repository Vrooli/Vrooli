package search

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	apidb "github.com/vrooli/api-core/database"
)

// SQLite implements Repository against SQLite.
type SQLite struct {
	DB *apidb.RoutedDB
}

var _ Repository = (*SQLite)(nil)

// NewSQLite returns a Repository backed by db.
func NewSQLite(db *apidb.RoutedDB) *SQLite { return &SQLite{DB: db} }

func (s *SQLite) InsertHistory(ctx context.Context, h History) (string, error) {
	if s == nil || s.DB == nil {
		return "", fmt.Errorf("search repository not configured")
	}
	h.Query = strings.TrimSpace(h.Query)
	if h.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if h.ID == "" {
		h.ID = uuid.NewString()
	}

	// A NaN or Inf score would round-trip as a corrupt REAL; drop it instead.
	var avg any
	if h.AvgScore != nil && !math.IsNaN(*h.AvgScore) && !math.IsInf(*h.AvgScore, 0) {
		avg = *h.AvgScore
	}

	_, err := s.DB.ExecContext(ctx, `
INSERT INTO search_history
  (id, query, collection, result_count, avg_score, response_time_ms, user_session, created_at)
VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING
`, h.ID, h.Query, strings.TrimSpace(h.Collection), h.ResultCount, avg,
		h.ResponseTimeMS, strings.TrimSpace(h.UserSession))
	if err != nil {
		return "", fmt.Errorf("insert search history: %w", err)
	}
	return h.ID, nil
}

func (s *SQLite) RecentHistory(ctx context.Context, limit int) ([]History, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("search repository not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, query, COALESCE(collection, ''), result_count, avg_score,
       COALESCE(response_time_ms, 0), COALESCE(user_session, ''), created_at
FROM search_history
ORDER BY created_at DESC, id DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query search history: %w", err)
	}
	defer rows.Close()

	var out []History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.ID, &h.Query, &h.Collection, &h.ResultCount, &h.AvgScore,
			&h.ResponseTimeMS, &h.UserSession, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan search history: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *SQLite) CountHistory(ctx context.Context) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("search repository not configured")
	}
	var n int64
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM search_history`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count search history: %w", err)
	}
	return n, nil
}

var _ = sql.ErrNoRows
