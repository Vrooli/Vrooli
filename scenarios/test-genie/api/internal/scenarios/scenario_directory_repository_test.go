package scenarios

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/testsqlite"
)

func TestScenarioDirectoryRepositorySummarizesExecutionEvidence(t *testing.T) {
	db := testsqlite.Open(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO suite_executions (id, run_id, scenario_name, requested_phases, requested_skip_phases, planned_phases, fail_fast, success, phases, started_at, completed_at) VALUES (?, ?, ?, '[]', '[]', '[]', 0, 0, '[]', ?, ?)`, "11111111-1111-1111-1111-111111111111", "run-1", "demo", now.Add(-time.Minute).Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	summaries, err := NewScenarioDirectoryRepository(db).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].ScenarioName != "demo" || summaries[0].TotalExecutions != 1 || summaries[0].LastExecutionID == nil || summaries[0].LastRunID != "run-1" {
		t.Fatalf("summaries=%+v", summaries)
	}
}
