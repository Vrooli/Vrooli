package planning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type sqlDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct {
	db  sqlDB
	now func() time.Time
}

func NewSQLiteRepository(db sqlDB) *SQLiteRepository {
	return &SQLiteRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

var _ Repository = (*SQLiteRepository)(nil)

func (r *SQLiteRepository) CreateScenario(ctx context.Context, in CreateInput) (Scenario, error) {
	slug, err := NormalizeSlug(in.Slug)
	if err != nil {
		return Scenario{}, err
	}
	now := r.now().Format(time.RFC3339Nano)
	displayName := firstNonEmpty(in.DisplayName, DefaultDisplayName(slug))
	targetStability := firstNonEmpty(in.TargetStability, DefaultTargetStability)
	_, err = r.db.ExecContext(ctx, `
INSERT INTO planned_scenario (slug, display_name, sector, tier, target_stability, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET
    display_name = excluded.display_name,
    sector = excluded.sector,
    tier = excluded.tier,
    target_stability = excluded.target_stability,
    updated_at = excluded.updated_at`,
		slug, displayName, in.Sector, in.Tier, targetStability, now, now)
	if err != nil {
		return Scenario{}, fmt.Errorf("upsert planned scenario: %w", err)
	}
	return r.GetScenario(ctx, slug)
}

func (r *SQLiteRepository) ListScenarios(ctx context.Context, filter ListFilter) ([]Scenario, error) {
	query := `SELECT slug, display_name, sector, tier, target_stability, created_at, updated_at FROM planned_scenario`
	args := []any{}
	switch {
	case filter.Sector != "" && filter.Tier != "":
		query += ` WHERE sector = ? AND tier = ?`
		args = append(args, filter.Sector, filter.Tier)
	case filter.Sector != "":
		query += ` WHERE sector = ?`
		args = append(args, filter.Sector)
	case filter.Tier != "":
		query += ` WHERE tier = ?`
		args = append(args, filter.Tier)
	}
	query += ` ORDER BY slug`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list planned scenarios: %w", err)
	}
	defer rows.Close()
	var out []Scenario
	for rows.Next() {
		scenario, err := scanScenario(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, scenario)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate planned scenarios: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close planned scenario rows: %w", err)
	}
	for i := range out {
		files, err := r.listFiles(ctx, out[i].Slug)
		if err != nil {
			return nil, err
		}
		out[i].Files = files
	}
	return out, nil
}

func (r *SQLiteRepository) GetScenario(ctx context.Context, slug string) (Scenario, error) {
	slug, err := NormalizeSlug(slug)
	if err != nil {
		return Scenario{}, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT slug, display_name, sector, tier, target_stability, created_at, updated_at FROM planned_scenario WHERE slug = ?`, slug)
	scenario, err := scanScenario(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Scenario{}, ErrScenarioNotFound{Slug: slug}
	}
	if err != nil {
		return Scenario{}, err
	}
	files, err := r.listFiles(ctx, slug)
	if err != nil {
		return Scenario{}, err
	}
	scenario.Files = files
	return scenario, nil
}

func (r *SQLiteRepository) PutFile(ctx context.Context, in PutFileInput) (ProtoFile, error) {
	slug, err := NormalizeSlug(in.Slug)
	if err != nil {
		return ProtoFile{}, err
	}
	path, err := NormalizeProtoPath(in.Path)
	if err != nil {
		return ProtoFile{}, err
	}
	if _, err := r.GetScenario(ctx, slug); err != nil {
		return ProtoFile{}, err
	}
	now := r.now().Format(time.RFC3339Nano)
	_, err = r.db.ExecContext(ctx, `
INSERT INTO planned_proto_file (planned_slug, path, text, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(planned_slug, path) DO UPDATE SET text = excluded.text, updated_at = excluded.updated_at`,
		slug, path, in.Text, now)
	if err != nil {
		return ProtoFile{}, fmt.Errorf("put planned proto file: %w", err)
	}
	return ProtoFile{Path: path, Text: in.Text, UpdatedAt: parseTime(now)}, nil
}

func (r *SQLiteRepository) DeleteFile(ctx context.Context, slug, path string) (bool, error) {
	slug, err := NormalizeSlug(slug)
	if err != nil {
		return false, err
	}
	path, err = NormalizeProtoPath(path)
	if err != nil {
		return false, err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM planned_proto_file WHERE planned_slug = ? AND path = ?`, slug, path)
	if err != nil {
		return false, fmt.Errorf("delete planned proto file: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read delete count: %w", err)
	}
	return n > 0, nil
}

func (r *SQLiteRepository) listFiles(ctx context.Context, slug string) ([]ProtoFile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT path, text, updated_at FROM planned_proto_file WHERE planned_slug = ? ORDER BY path`, slug)
	if err != nil {
		return nil, fmt.Errorf("list planned proto files: %w", err)
	}
	defer rows.Close()
	var files []ProtoFile
	for rows.Next() {
		var file ProtoFile
		var updated string
		if err := rows.Scan(&file.Path, &file.Text, &updated); err != nil {
			return nil, fmt.Errorf("scan planned proto file: %w", err)
		}
		file.UpdatedAt = parseTime(updated)
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate planned proto files: %w", err)
	}
	return files, nil
}

type scenarioScanner interface {
	Scan(dest ...any) error
}

func scanScenario(row scenarioScanner) (Scenario, error) {
	var scenario Scenario
	var created, updated string
	if err := row.Scan(&scenario.Slug, &scenario.DisplayName, &scenario.Sector, &scenario.Tier, &scenario.TargetStability, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Scenario{}, err
		}
		return Scenario{}, fmt.Errorf("scan planned scenario: %w", err)
	}
	scenario.CreatedAt = parseTime(created)
	scenario.UpdatedAt = parseTime(updated)
	return scenario, nil
}

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}
