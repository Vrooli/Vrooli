// Package testing provides LLM-based skill testing via Ollama.
package testing

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// Repository handles database operations for test results.
// This is a testing seam: inject a mock Repository in tests to avoid database access.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new testing repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Save stores a test result in the database.
func (r *Repository) Save(result *TestResult) error {
	if result.ID == "" {
		result.ID = uuid.New().String()
	}

	varsJSON, _ := json.Marshal(result.InputVars)

	_, err := r.db.Exec(`
		INSERT INTO test_results (id, skill_id, model, input_variables, response, response_time, token_count, tested_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
	`, result.ID, result.SkillID, result.Role, string(varsJSON), result.Response, result.ResponseTime, result.TokenCount)

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
		WHERE skill_id = $1
		ORDER BY tested_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(query, skillID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TestResult
	for rows.Next() {
		var tr TestResult
		if err := rows.Scan(&tr.ID, &tr.SkillID, &tr.Role, &tr.InputVars, &tr.Response, &tr.ResponseTime, &tr.TokenCount, &tr.Rating, &tr.Notes, &tr.TestedAt); err != nil {
			continue
		}
		results = append(results, tr)
	}

	return results, nil
}
