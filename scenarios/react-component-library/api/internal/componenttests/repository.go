package componenttests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type Repository interface {
	Save(context.Context, Report) error
	Get(context.Context, string) (Report, error)
	List(context.Context, string, int) ([]Report, error)
}

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Save(ctx context.Context, report Report) error {
	results, err := json.Marshal(struct {
		Results   []Result   `json:"results"`
		Artifacts []Artifact `json:"artifacts"`
	}{Results: report.Results, Artifacts: report.Artifacts})
	if err != nil {
		return fmt.Errorf("encode report results: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO component_test_reports (id, component_id, root_library_id, root_version, include_closure, created_at, verdict, results_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, report.ID, report.RootComponentID, report.RootLibraryID, report.RootVersion, report.IncludeClosure, report.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), report.Verdict, string(results))
	return err
}
func (r *SQLiteRepository) Get(ctx context.Context, id string) (Report, error) {
	rows, err := r.query(ctx, `WHERE id = ?`, id, 1)
	if err != nil {
		return Report{}, err
	}
	if len(rows) == 0 {
		return Report{}, sql.ErrNoRows
	}
	return rows[0], nil
}
func (r *SQLiteRepository) List(ctx context.Context, componentID string, limit int) ([]Report, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if componentID == "" {
		return r.query(ctx, "", nil, limit)
	}
	return r.query(ctx, `WHERE component_id = ?`, componentID, limit)
}
func (r *SQLiteRepository) query(ctx context.Context, where string, arg any, limit int) ([]Report, error) {
	query := `SELECT id, component_id, root_library_id, root_version, include_closure, created_at, verdict, results_json FROM component_test_reports ` + where + ` ORDER BY created_at DESC LIMIT ?`
	args := []any{}
	if arg != nil {
		args = append(args, arg)
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Report{}
	for rows.Next() {
		var report Report
		var created string
		var results string
		if err := rows.Scan(&report.ID, &report.RootComponentID, &report.RootLibraryID, &report.RootVersion, &report.IncludeClosure, &created, &report.Verdict, &results); err != nil {
			return nil, err
		}
		if err := report.CreatedAt.UnmarshalText([]byte(created)); err != nil {
			return nil, err
		}
		var persisted struct {
			Results   []Result   `json:"results"`
			Artifacts []Artifact `json:"artifacts"`
		}
		if err := json.Unmarshal([]byte(results), &persisted); err != nil {
			return nil, err
		}
		report.Results, report.Artifacts = persisted.Results, persisted.Artifacts
		out = append(out, report)
	}
	return out, rows.Err()
}
