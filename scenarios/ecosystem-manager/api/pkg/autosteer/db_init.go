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
		"execution_feedback_entries",
	}

	if err := ensureExecutionFeedbackEntriesTable(db); err != nil {
		return fmt.Errorf("failed to ensure execution feedback entries table: %w", err)
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

	// Ensure new columns exist for backward compatibility
	if err := ensureColumnExists(db, "profile_execution_state", "auto_steer_iteration", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("failed to ensure column auto_steer_iteration exists: %w", err)
	}
	if err := ensureColumnExists(db, "profile_execution_state", "phase_started_at", "TIMESTAMP DEFAULT NOW()"); err != nil {
		return fmt.Errorf("failed to ensure column phase_started_at exists: %w", err)
	}
	if err := ensureProfileExecutionStateTrigger(db); err != nil {
		return fmt.Errorf("failed to ensure profile_execution_state trigger exists: %w", err)
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

// ensureColumnExists adds a column if it does not already exist. Useful for lightweight migrations.
func ensureColumnExists(db *sql.DB, table string, column string, definition string) error {
	query := fmt.Sprintf(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = '%s' AND column_name = '%s'
			) THEN
				EXECUTE 'ALTER TABLE %s ADD COLUMN %s %s';
			END IF;
		END;
		$$;
	`, table, column, table, column, definition)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to ensure column %s.%s: %w", table, column, err)
	}
	return nil
}

// ensureProfileExecutionStateTrigger aligns the trigger with the actual column name (last_updated).
func ensureProfileExecutionStateTrigger(db *sql.DB) error {
	query := `
		CREATE OR REPLACE FUNCTION update_profile_execution_state_last_updated()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.last_updated = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS trigger_profile_execution_state_updated ON profile_execution_state;
		CREATE TRIGGER trigger_profile_execution_state_updated
			BEFORE UPDATE ON profile_execution_state
			FOR EACH ROW
			EXECUTE FUNCTION update_profile_execution_state_last_updated();
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to ensure trigger trigger_profile_execution_state_updated: %w", err)
	}

	return nil
}

func ensureExecutionFeedbackEntriesTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS execution_feedback_entries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		execution_task_id VARCHAR(255) NOT NULL,
		category VARCHAR(100) NOT NULL,
		severity VARCHAR(20) NOT NULL,
		suggested_action TEXT,
		comments TEXT,
		metadata JSONB,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_execution_feedback_entries_task_id ON execution_feedback_entries(execution_task_id);
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to ensure execution_feedback_entries table: %w", err)
	}

	_, err := db.Exec(`
	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1
			FROM pg_constraint
			WHERE conname = 'execution_feedback_entries_execution_task_id_fkey'
		) THEN
			ALTER TABLE execution_feedback_entries DROP CONSTRAINT execution_feedback_entries_execution_task_id_fkey;
		END IF;
	END
	$$;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop legacy execution feedback foreign key: %w", err)
	}

	return nil
}
