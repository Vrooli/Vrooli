package autosteer

import (
	"database/sql"
	"fmt"
	"log"
)

// EnsureTablesExist checks if the auto_steer tables exist and provides helpful error messages
func EnsureTablesExist(db *sql.DB) error {
	tables := []string{
		"auto_steer_profiles",
		"profile_executions",
		"profile_execution_state",
	}

	for _, table := range tables {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			);
		`
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if table %s exists: %w", table, err)
		}

		if !exists {
			return fmt.Errorf(
				"table %s does not exist - please run the schema initialization:\n"+
					"psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f initialization/postgres/schema.sql",
				table,
			)
		}
	}

	log.Println("✅ Auto Steer database tables verified")
	return nil
}

// GetTableCounts returns the count of records in each auto_steer table (for debugging)
func GetTableCounts(db *sql.DB) (map[string]int, error) {
	counts := make(map[string]int)

	tables := []string{
		"auto_steer_profiles",
		"profile_executions",
		"profile_execution_state",
	}

	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := db.QueryRow(query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to count rows in %s: %w", table, err)
		}
		counts[table] = count
	}

	return counts, nil
}
