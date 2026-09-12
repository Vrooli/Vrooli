package docaccess

import (
	"context"
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

func (s *SQLite) LogAccess(ctx context.Context, a Access) error {
	if s == nil || s.DB == nil {
		return nil
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO doc_access_log (id, scenario_name, doc_type, operation, created_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING
`, a.ID, strings.TrimSpace(a.ScenarioName), strings.TrimSpace(a.DocType), strings.TrimSpace(a.Operation))
	if err != nil {
		return fmt.Errorf("log doc access: %w", err)
	}
	return nil
}

// QueryStats tallies operations per scenario and doc type.
//
// The aggregate FILTER clause carries over from Postgres unchanged: SQLite has
// supported it since 3.30.
func (s *SQLite) QueryStats(ctx context.Context, filter Filter) ([]Stat, error) {
	if s == nil || s.DB == nil {
		return nil, nil
	}

	query := `SELECT scenario_name, doc_type,
	COUNT(*) FILTER (WHERE operation = 'read') AS read_count,
	COUNT(*) FILTER (WHERE operation = 'write') AS write_count,
	COUNT(*) FILTER (WHERE operation = 'reset') AS reset_count
	FROM doc_access_log`

	var (
		conditions []string
		args       []any
	)
	if s := strings.TrimSpace(filter.ScenarioName); s != "" {
		conditions = append(conditions, "scenario_name = ?")
		args = append(args, s)
	}
	if d := strings.TrimSpace(filter.DocType); d != "" {
		conditions = append(conditions, "doc_type = ?")
		args = append(args, d)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " GROUP BY scenario_name, doc_type ORDER BY scenario_name, doc_type"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query doc access stats: %w", err)
	}
	defer rows.Close()

	var stats []Stat
	for rows.Next() {
		var st Stat
		if err := rows.Scan(&st.ScenarioName, &st.DocType, &st.ReadCount, &st.WriteCount, &st.ResetCount); err != nil {
			return nil, fmt.Errorf("scan doc access stat: %w", err)
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

func (s *SQLite) Recent(ctx context.Context, limit int) ([]Access, error) {
	if s == nil || s.DB == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, scenario_name, doc_type, operation, created_at
FROM doc_access_log
ORDER BY created_at DESC, id DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("query doc access log: %w", err)
	}
	defer rows.Close()

	var out []Access
	for rows.Next() {
		var a Access
		if err := rows.Scan(&a.ID, &a.ScenarioName, &a.DocType, &a.Operation, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan doc access: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
