package preferences

import (
	"context"
	"database/sql"
	"fmt"
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

func (s *SQLite) Upsert(ctx context.Context, p Preference) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("preferences repository not configured")
	}
	p.UserID = strings.TrimSpace(p.UserID)
	if p.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO user_preferences
  (id, user_id, default_collection, saved_queries, dashboard_layout,
   alert_preferences, created_at, updated_at)
VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''),
        CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(user_id) DO UPDATE SET
  default_collection = excluded.default_collection,
  saved_queries = COALESCE(excluded.saved_queries, user_preferences.saved_queries),
  dashboard_layout = COALESCE(excluded.dashboard_layout, user_preferences.dashboard_layout),
  alert_preferences = COALESCE(excluded.alert_preferences, user_preferences.alert_preferences),
  updated_at = CURRENT_TIMESTAMP
`, p.ID, p.UserID, strings.TrimSpace(p.DefaultCollection), strings.TrimSpace(p.SavedQueries),
		strings.TrimSpace(p.DashboardLayout), strings.TrimSpace(p.AlertPreferences))
	if err != nil {
		return fmt.Errorf("upsert user preferences: %w", err)
	}
	return nil
}

func (s *SQLite) Get(ctx context.Context, userID string) (Preference, bool, error) {
	if s == nil || s.DB == nil {
		return Preference{}, false, fmt.Errorf("preferences repository not configured")
	}
	var p Preference
	err := s.DB.QueryRowContext(ctx, `
SELECT id, COALESCE(user_id, ''), COALESCE(default_collection, ''), COALESCE(saved_queries, ''),
       COALESCE(dashboard_layout, ''), COALESCE(alert_preferences, ''), created_at, updated_at
FROM user_preferences
WHERE user_id = ?
`, strings.TrimSpace(userID)).Scan(&p.ID, &p.UserID, &p.DefaultCollection, &p.SavedQueries,
		&p.DashboardLayout, &p.AlertPreferences, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return Preference{}, false, nil
	}
	if err != nil {
		return Preference{}, false, fmt.Errorf("get user preferences: %w", err)
	}
	return p, true, nil
}
