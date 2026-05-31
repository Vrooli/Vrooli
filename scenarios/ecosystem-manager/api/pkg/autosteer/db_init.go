package autosteer

import (
	"database/sql"
	"fmt"
	"log"
)

// EnsureTablesExist creates (or cuts over) the controller's tables. It is
// self-healing: tables are created with the objective-controller shape if
// absent, and a one-time cutover drops the legacy phase-list
// profile_execution_state table (active in-flight runs are intentionally
// dropped — see the greenfield cutover note in the plan).
func EnsureTablesExist(db *sql.DB) error {
	if err := ensureProfileExecutionsTable(db); err != nil {
		return err
	}
	if err := ensureExecutionFeedbackEntriesTable(db); err != nil {
		return fmt.Errorf("failed to ensure execution feedback entries table: %w", err)
	}
	if err := cutoverLegacyExecutionState(db); err != nil {
		return fmt.Errorf("failed legacy execution-state cutover: %w", err)
	}
	if err := ensureProfileExecutionStateTable(db); err != nil {
		return err
	}
	if err := ensureDecisionTraceTable(db); err != nil {
		return err
	}
	if err := ensureProfileExecutionStateTrigger(db); err != nil {
		return fmt.Errorf("failed to ensure profile_execution_state trigger exists: %w", err)
	}

	log.Println("✅ Auto Steer controller tables verified")
	return nil
}

// GetTableCounts returns the count of records in each controller table (for debugging)
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

func ensureProfileExecutionsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS profile_executions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		profile_id VARCHAR(255) NOT NULL,
		task_id UUID NOT NULL,
		scenario_name VARCHAR(255) NOT NULL,
		start_metrics JSONB,
		end_metrics JSONB,
		phase_breakdown JSONB,
		total_iterations INTEGER,
		total_duration_ms BIGINT,
		user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
		user_comments TEXT,
		user_feedback_at TIMESTAMP,
		executed_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_profile_executions_profile_id ON profile_executions(profile_id);
	CREATE INDEX IF NOT EXISTS idx_profile_executions_scenario ON profile_executions(scenario_name);
	CREATE INDEX IF NOT EXISTS idx_profile_executions_executed_at ON profile_executions(executed_at DESC);
	`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to ensure profile_executions table: %w", err)
	}
	return nil
}

// cutoverLegacyExecutionState drops the legacy phase-list state table so the
// new objective-controller shape can be created cleanly. One-time, idempotent.
func cutoverLegacyExecutionState(db *sql.DB) error {
	var hasLegacy bool
	check := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = 'profile_execution_state'
			AND column_name = 'current_phase_index'
		);
	`
	if err := db.QueryRow(check).Scan(&hasLegacy); err != nil {
		return fmt.Errorf("failed to detect legacy execution-state shape: %w", err)
	}
	if hasLegacy {
		log.Println("Auto Steer: migrating profile_execution_state to objective-controller shape (dropping legacy phase-list table)")
		if _, err := db.Exec(`DROP TABLE IF EXISTS profile_execution_state`); err != nil {
			return fmt.Errorf("failed to drop legacy profile_execution_state: %w", err)
		}
	}
	return nil
}

func ensureProfileExecutionStateTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS profile_execution_state (
		task_id UUID PRIMARY KEY,
		profile_id VARCHAR(255) NOT NULL,
		iteration INTEGER NOT NULL DEFAULT 0,
		current_skill VARCHAR(255) NOT NULL DEFAULT '',
		current_rationale TEXT NOT NULL DEFAULT '',
		findings JSONB,
		score_history JSONB,
		trace JSONB,
		metrics JSONB,
		started_at TIMESTAMP DEFAULT NOW(),
		last_updated TIMESTAMP DEFAULT NOW()
	);
	`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to ensure profile_execution_state table: %w", err)
	}
	return nil
}

// ensureDecisionTraceTable persists per-iteration controller decisions so the
// reasoning survives finalization (the live state.Trace is deleted on finish).
func ensureDecisionTraceTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS decision_trace (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		task_id VARCHAR(255) NOT NULL,
		profile_id VARCHAR(255) NOT NULL,
		scenario_name VARCHAR(255) NOT NULL DEFAULT '',
		iteration INTEGER NOT NULL,
		chosen_skill VARCHAR(255) NOT NULL DEFAULT '',
		heaviest_dimension VARCHAR(100) NOT NULL DEFAULT '',
		rationale TEXT NOT NULL DEFAULT '',
		dimension_scores JSONB,
		fingerprint VARCHAR(128) NOT NULL DEFAULT '',
		score_before DOUBLE PRECISION NOT NULL DEFAULT 0,
		score_after DOUBLE PRECISION NOT NULL DEFAULT 0,
		realized_delta DOUBLE PRECISION NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_decision_trace_task ON decision_trace(task_id, iteration);
	`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to ensure decision_trace table: %w", err)
	}
	return nil
}

// ensureProfileExecutionStateTrigger aligns the trigger with the last_updated column.
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

	return nil
}
