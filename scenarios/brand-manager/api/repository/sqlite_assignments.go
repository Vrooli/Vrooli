// DOC: docs/reference/api-endpoints.md#assignments
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"brand-manager/domain"
)

// SQLiteAssignmentRepository implements AssignmentRepository using SQLite.
type SQLiteAssignmentRepository struct {
	db *sql.DB
}

// NewSQLiteAssignmentRepository creates a new SQLite-backed assignment repository.
func NewSQLiteAssignmentRepository(db *sql.DB) *SQLiteAssignmentRepository {
	return &SQLiteAssignmentRepository{db: db}
}

// Create inserts a new brand-to-scenario assignment. [REQ:BM-REQ-ASSIGN-LINK]
func (r *SQLiteAssignmentRepository) Create(ctx context.Context, a *domain.Assignment) error {
	elementsJSON, _ := json.Marshal(a.Elements)
	t, now := nowUTC()
	a.AppliedAt = t

	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO assignments (id, brand_id, scenario_name, brand_version, elements, applied_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.BrandID, a.ScenarioName, a.BrandVersion, string(elementsJSON), now,
	)
	if err != nil {
		return fmt.Errorf("insert assignment: %w", err)
	}
	return nil
}

// GetByScenario returns the current brand assignment for a scenario. [REQ:BM-REQ-ASSIGN-LINK]
func (r *SQLiteAssignmentRepository) GetByScenario(ctx context.Context, scenarioName string) (*domain.Assignment, error) {
	var a domain.Assignment
	var elementsJSON, appliedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, brand_id, scenario_name, brand_version, elements, applied_at
		 FROM assignments WHERE scenario_name = ?`, scenarioName).
		Scan(&a.ID, &a.BrandID, &a.ScenarioName, &a.BrandVersion, &elementsJSON, &appliedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(elementsJSON), &a.Elements)
	a.AppliedAt, _ = time.Parse(time.RFC3339, appliedAt)
	return &a, nil
}

// ListByBrandID returns all assignments for a brand. [REQ:BM-REQ-ASSIGN-MULTI]
func (r *SQLiteAssignmentRepository) ListByBrandID(ctx context.Context, brandID string) ([]*domain.Assignment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, brand_id, scenario_name, brand_version, elements, applied_at
		 FROM assignments WHERE brand_id = ? ORDER BY applied_at DESC`, brandID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*domain.Assignment
	for rows.Next() {
		var a domain.Assignment
		var elementsJSON, appliedAt string
		if err := rows.Scan(&a.ID, &a.BrandID, &a.ScenarioName, &a.BrandVersion, &elementsJSON, &appliedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(elementsJSON), &a.Elements)
		a.AppliedAt, _ = time.Parse(time.RFC3339, appliedAt)
		assignments = append(assignments, &a)
	}
	return assignments, rows.Err()
}

// Delete removes an assignment by ID.
func (r *SQLiteAssignmentRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM assignments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
