package execution_test

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"plan-manager/internal/execution"
)

func TestTelemetryPersistsAndReloadsFromExecutionStore(t *testing.T) {
	database := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(execution.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := execution.NewSQLiteRepository(database, nil)
	event := execution.ExecutionTelemetryEvent{
		ID: "telemetry-restart-1", Kind: execution.TelemetryReuse,
		OccurredAt: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC),
		TaskID:     "task-1", PlanID: "plan-1", ValidationID: "validation-1",
		PolicyIdentity: "policy-1", ContentIdentity: "content-1",
	}
	if err := repo.SaveTelemetry(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveTelemetry(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	reloaded := execution.NewSQLiteRepository(database, nil)
	events, err := reloaded.ListTelemetry(context.Background(), "plan-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != event.ID || events[0].Kind != execution.TelemetryReuse || events[0].PolicyIdentity != event.PolicyIdentity {
		t.Fatalf("reloaded telemetry = %#v", events)
	}
}
