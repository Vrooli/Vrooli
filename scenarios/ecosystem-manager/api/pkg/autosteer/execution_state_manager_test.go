package autosteer

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestExecutionStateManager_Get_NullJSON(t *testing.T) {
	container, cleanup := SetupTestDatabase(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	taskID := uuid.New().String()
	profileID := "profile-null-json"
	startedAt := time.Now().Add(-1 * time.Minute).UTC()
	lastUpdated := time.Now().UTC()

	_, err := container.db.Exec(`
		INSERT INTO profile_execution_state (
			task_id,
			profile_id,
			current_phase_index,
			current_phase_iteration,
			auto_steer_iteration,
			phase_started_at,
			phase_history,
			metrics,
			phase_start_metrics,
			started_at,
			last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, taskID, profileID, 0, 0, 0, nil, nil, nil, nil, startedAt, lastUpdated)
	if err != nil {
		t.Fatalf("failed to insert execution state: %v", err)
	}

	manager := NewExecutionStateManager(container.db)
	state, err := manager.Get(taskID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if state == nil {
		t.Fatal("expected execution state, got nil")
	}
	if state.ProfileID != profileID {
		t.Fatalf("expected profile ID %s, got %s", profileID, state.ProfileID)
	}
	if state.PhaseHistory == nil {
		t.Error("expected phase history to be initialized")
	}
	if state.Metrics.Timestamp.IsZero() {
		t.Error("expected metrics timestamp to be populated")
	}
	if state.PhaseStartMetrics.Timestamp.IsZero() {
		t.Error("expected phase start metrics timestamp to be populated")
	}
}
