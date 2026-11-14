// Package shapes derives the governed binding shape of a program.
package shapes

import (
	"context"
	"strings"

	"program-runtime/internal/sessions"
)

type SQLExecutor = sessions.SQLExecutor

// Derive returns the distinct bindings that completed successfully for a program.
func Derive(ctx context.Context, db SQLExecutor, programID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT binding_id FROM binding_invocations WHERE program_id = ? AND outcome = 'success' AND binding_id <> '' ORDER BY binding_id`, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// Key returns the stable shape key for sorted binding IDs.
func Key(bindingIDs []string) string { return strings.Join(bindingIDs, "|") }
