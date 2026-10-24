// Package testing provides LLM-based skill testing via Ollama.
package testing

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Repository handles database operations for test results.
// This is a testing seam: inject a mock Repository in tests to avoid database access.
type Repository struct {
	db  databaseExecutor
	ctx context.Context
}

type databaseExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// NewRepository creates a new testing repository.
func NewRepository(db databaseExecutor) *Repository {
	return &Repository{db: db, ctx: context.Background()}
}

func (r *Repository) WithContext(ctx context.Context) *Repository {
	copy := *r
	copy.ctx = ctx
	return &copy
}

func (r *Repository) WithRequestContext(ctx context.Context) TestRepository {
	return r.WithContext(ctx)
}

// Save stores a test result in the database.
func (r *Repository) Save(result *TestResult) error {
	if result.ID == "" {
		result.ID = uuid.New().String()
	}

	var inputVars any
	if result.InputVars != nil {
		if json.Valid([]byte(*result.InputVars)) {
			inputVars = *result.InputVars
		} else {
			varsJSON, _ := json.Marshal(result.InputVars)
			inputVars = string(varsJSON)
		}
	}
	testedAt := result.TestedAt
	if testedAt.IsZero() {
		testedAt = time.Now()
	}

	_, err := r.db.ExecContext(r.ctx, `
		INSERT INTO test_results (id, skill_id, model, input_variables, response, response_time, token_count, tested_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, result.ID, result.SkillID, result.Role, inputVars, result.Response, result.ResponseTime, result.TokenCount, testedAt.UTC().Format(time.RFC3339Nano))

	return err
}

// GetHistory retrieves test history for a skill, newest first.
func (r *Repository) GetHistory(skillID string, limit int) ([]TestResult, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT id, skill_id, model, input_variables, response, response_time, token_count, rating, notes, tested_at
		FROM test_results
		WHERE skill_id = ?
		ORDER BY tested_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(r.ctx, query, skillID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TestResult
	for rows.Next() {
		var (
			tr          TestResult
			inputVars   sql.NullString
			testedAtRaw string
		)
		if err := rows.Scan(&tr.ID, &tr.SkillID, &tr.Role, &inputVars, &tr.Response, &tr.ResponseTime, &tr.TokenCount, &tr.Rating, &tr.Notes, &testedAtRaw); err != nil {
			continue
		}
		if inputVars.Valid {
			tr.InputVars = &inputVars.String
		}
		testedAt, err := time.Parse(time.RFC3339Nano, testedAtRaw)
		if err != nil {
			continue
		}
		tr.TestedAt = testedAt
		results = append(results, tr)
	}

	return results, nil
}
