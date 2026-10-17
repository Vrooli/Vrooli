package wiring

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
)

func TestMaintenanceBatchHydratesDurableIdentityWithoutActionOrConfig(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	task := &domain.Task{ID: uuid.New(), Title: "scope fixture", ScopePath: "."}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	start, end := time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(-time.Minute)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "durable-tag", Status: domain.RunStatusFailed, RunnerPID: 315576, RunnerPGID: 315576, StartedAt: &start, EndedAt: &end, CustomEnv: map[string]string{"secret": "must-not-hydrate"}, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode}}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	identities, err := readMaintenanceExecutorIdentities(t.Context(), db, []maintenance.ExecutorRef{{WorkRef: maintenance.WorkRef{ID: run.ID.String()}}})
	if err != nil || len(identities) != 1 {
		t.Fatalf("scope identities=%+v err=%v", identities, err)
	}
	got := identities[0]
	if got.ID != run.ID || got.Tag != run.Tag || got.RunnerPID != run.RunnerPID || got.RunnerPGID != run.RunnerPGID || got.StartedAt == nil || !got.StartedAt.Equal(start) || got.EndedAt == nil || !got.EndedAt.Equal(end) {
		t.Fatalf("durable physical binding lost: %+v", got)
	}
	if got.ResolvedConfig != nil || got.CustomEnv != nil || got.Actions != nil {
		t.Fatal("inventory hydrated unrelated action/config/secret data")
	}
}
