package roadmap

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
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

func (r *SQLiteRepository) ListSectors(ctx context.Context) ([]Sector, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT slug, name, description, created_at, updated_at FROM roadmap_sector ORDER BY slug`)
	if err != nil {
		return nil, fmt.Errorf("list sectors: %w", err)
	}
	defer rows.Close()
	var out []Sector
	for rows.Next() {
		sector, err := scanSector(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sector)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sectors: %w", err)
	}
	return out, nil
}

func (r *SQLiteRepository) UpsertSector(ctx context.Context, sector Sector) (Sector, error) {
	slug, err := NormalizeID("sector.slug", sector.Slug)
	if err != nil {
		return Sector{}, err
	}
	name := sector.Name
	if name == "" {
		name = DefaultSectorName(slug)
	}
	now := r.now().Format(time.RFC3339Nano)
	_, err = r.db.ExecContext(ctx, `
INSERT INTO roadmap_sector (slug, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    updated_at = excluded.updated_at`,
		slug, name, sector.Description, now, now)
	if err != nil {
		return Sector{}, fmt.Errorf("upsert sector: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `SELECT slug, name, description, created_at, updated_at FROM roadmap_sector WHERE slug = ?`, slug)
	return scanSector(row)
}

func (r *SQLiteRepository) ListMilestones(ctx context.Context) ([]Milestone, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, required_scenarios_json, created_at, updated_at FROM roadmap_milestone ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}
	defer rows.Close()
	var out []Milestone
	for rows.Next() {
		milestone, err := scanMilestone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, milestone)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate milestones: %w", err)
	}
	return out, nil
}

func (r *SQLiteRepository) UpsertMilestone(ctx context.Context, milestone Milestone) (Milestone, error) {
	id, err := NormalizeID("milestone.id", milestone.ID)
	if err != nil {
		return Milestone{}, err
	}
	if milestone.Name == "" {
		return Milestone{}, ErrInvalidArgument{Field: "milestone.name", Reason: "is required"}
	}
	required := normalizeRequiredScenarios(milestone.RequiredScenarios)
	rawRequired, err := json.Marshal(required)
	if err != nil {
		return Milestone{}, fmt.Errorf("encode required scenarios: %w", err)
	}
	now := r.now().Format(time.RFC3339Nano)
	_, err = r.db.ExecContext(ctx, `
INSERT INTO roadmap_milestone (id, name, description, required_scenarios_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    required_scenarios_json = excluded.required_scenarios_json,
    updated_at = excluded.updated_at`,
		id, milestone.Name, milestone.Description, string(rawRequired), now, now)
	if err != nil {
		return Milestone{}, fmt.Errorf("upsert milestone: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, required_scenarios_json, created_at, updated_at FROM roadmap_milestone WHERE id = ?`, id)
	return scanMilestone(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSector(row rowScanner) (Sector, error) {
	var sector Sector
	var created, updated string
	if err := row.Scan(&sector.Slug, &sector.Name, &sector.Description, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Sector{}, err
		}
		return Sector{}, fmt.Errorf("scan sector: %w", err)
	}
	sector.CreatedAt = parseTime(created)
	sector.UpdatedAt = parseTime(updated)
	return sector, nil
}

func scanMilestone(row rowScanner) (Milestone, error) {
	var milestone Milestone
	var rawRequired, created, updated string
	if err := row.Scan(&milestone.ID, &milestone.Name, &milestone.Description, &rawRequired, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Milestone{}, err
		}
		return Milestone{}, fmt.Errorf("scan milestone: %w", err)
	}
	if err := json.Unmarshal([]byte(rawRequired), &milestone.RequiredScenarios); err != nil {
		return Milestone{}, fmt.Errorf("decode milestone required scenarios: %w", err)
	}
	milestone.RequiredScenarios = normalizeRequiredScenarios(milestone.RequiredScenarios)
	milestone.CreatedAt = parseTime(created)
	milestone.UpdatedAt = parseTime(updated)
	return milestone, nil
}

func normalizeRequiredScenarios(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range in {
		value = stringsTrimLower(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringsTrimLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}
