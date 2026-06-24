package autosteer

import (
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the Auto Steer controller's SQLite DDL for registration with
// database.EnsureSchemas (see pkg/dbschema). It owns profile_executions,
// profile_execution_state, and decision_trace.
func Schema() string { return schemaSQL }

// GetTableCounts returns the count of records in each controller table (for
// startup diagnostics).
func GetTableCounts(db *sql.DB) (map[string]int, error) {
	counts := make(map[string]int)

	tables := []string{
		"profile_executions",
		"profile_execution_state",
		"decision_trace",
	}

	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table) // #nosec G201 — table from fixed allowlist
		if err := db.QueryRow(query).Scan(&count); err != nil {
			return nil, fmt.Errorf("failed to count rows in %s: %w", table, err)
		}
		counts[table] = count
	}

	return counts, nil
}
