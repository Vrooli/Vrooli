package assignments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"brand-manager/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (repository unit
// tests via testutil/db.NewSQLite) and *database.RoutedDB (production) satisfy
// it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sqliteRepository is the production impl of Repository. Unexported so callers
// depend on the interface — tests substitute fakes without reaching inside.
type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production assignment Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// assignmentTimeFormat matches brands (RFC3339Nano), which sorts
// lexicographically in time order for a fixed zone so string ordering on the
// column is correct.
const assignmentTimeFormat = time.RFC3339Nano

const (
	// scenario_name is the conflict target (one assignment per scenario): an
	// upsert replaces the row in place, re-pinning brand_id/version/elements.
	upsertAssignmentSQL = `
INSERT INTO assignments (id, brand_id, scenario_name, brand_version, elements, applied_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(scenario_name) DO UPDATE SET
  brand_id = excluded.brand_id,
  brand_version = excluded.brand_version,
  elements = excluded.elements,
  applied_at = excluded.applied_at
`
	selectAssignmentColumns = `id, brand_id, scenario_name, brand_version, elements, applied_at`

	selectByScenarioSQL = `SELECT ` + selectAssignmentColumns + ` FROM assignments WHERE scenario_name = ?`

	deleteByScenarioSQL = `DELETE FROM assignments WHERE scenario_name = ?`
)

func (s *sqliteRepository) Upsert(ctx context.Context, a Assignment) (Assignment, error) {
	if a.ID == "" {
		// Preserve a stable id across re-assignments by reusing the existing
		// row's id when one is present; otherwise mint a new one.
		if existing, err := s.GetByScenario(ctx, a.ScenarioName); err == nil {
			a.ID = existing.ID
		} else {
			var notFound ErrAssignmentNotFound
			if !errors.As(err, &notFound) {
				return Assignment{}, err
			}
			a.ID = uuid.NewString()
		}
	}
	a.AppliedAt = s.clock.Now().UTC()

	elementsJSON, err := json.Marshal(a.Elements)
	if err != nil {
		return Assignment{}, fmt.Errorf("marshal elements: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, upsertAssignmentSQL,
		a.ID, a.BrandID, a.ScenarioName, a.BrandVersion, string(elementsJSON),
		a.AppliedAt.Format(assignmentTimeFormat),
	); err != nil {
		return Assignment{}, fmt.Errorf("upsert assignment for %q: %w", a.ScenarioName, err)
	}
	return a, nil
}

func (s *sqliteRepository) GetByScenario(ctx context.Context, scenarioName string) (Assignment, error) {
	row := s.db.QueryRowContext(ctx, selectByScenarioSQL, scenarioName)
	a, err := scanAssignment(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Assignment{}, ErrAssignmentNotFound{Scenario: scenarioName}
	}
	if err != nil {
		return Assignment{}, fmt.Errorf("get assignment for %q: %w", scenarioName, err)
	}
	return a, nil
}

func (s *sqliteRepository) ListByBrand(ctx context.Context, brandID string) ([]Assignment, error) {
	query := `SELECT ` + selectAssignmentColumns + ` FROM assignments`
	var args []any
	if brandID != "" {
		query += ` WHERE brand_id = ?`
		args = append(args, brandID)
	}
	query += ` ORDER BY applied_at DESC, scenario_name ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []Assignment
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignments: %w", err)
	}
	return assignments, nil
}

func (s *sqliteRepository) DeleteByScenario(ctx context.Context, scenarioName string) error {
	res, err := s.db.ExecContext(ctx, deleteByScenarioSQL, scenarioName)
	if err != nil {
		return fmt.Errorf("delete assignment for %q: %w", scenarioName, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete assignment for %q rows: %w", scenarioName, err)
	}
	if n == 0 {
		return ErrAssignmentNotFound{Scenario: scenarioName}
	}
	return nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAssignment(sc rowScanner) (Assignment, error) {
	var (
		a            Assignment
		elementsRaw  string
		appliedAtRaw string
	)
	if err := sc.Scan(&a.ID, &a.BrandID, &a.ScenarioName, &a.BrandVersion, &elementsRaw, &appliedAtRaw); err != nil {
		return Assignment{}, err
	}
	if err := unmarshalElements(elementsRaw, &a.Elements); err != nil {
		return Assignment{}, fmt.Errorf("parse elements: %w", err)
	}
	applied, err := time.Parse(assignmentTimeFormat, appliedAtRaw)
	if err != nil {
		return Assignment{}, fmt.Errorf("parse applied_at %q: %w", appliedAtRaw, err)
	}
	a.AppliedAt = applied
	return a, nil
}

// unmarshalElements decodes the JSON elements column, treating the empty string
// and the JSON null literal as no elements.
func unmarshalElements(raw string, dst *[]string) error {
	if raw == "" || raw == "null" {
		return nil
	}
	return json.Unmarshal([]byte(raw), dst)
}
