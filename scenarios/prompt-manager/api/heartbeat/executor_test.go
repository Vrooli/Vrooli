package heartbeat

import (
	"context"
	"testing"

	"prompt-manager/store"
)

func TestExecutorExecuteFailsWhenConfigMissing(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)

	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := teamStore.Create(ctx, &store.Team{ID: "team-1", DisplayName: "Team"}); err != nil {
		t.Fatalf("create team: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "")
	result, err := executor.Execute(ctx, "team-1", "agent-1", "profile-key")

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if result == nil || result.Status != store.HeartbeatStatusFailed {
		t.Fatalf("expected failed status, got %+v", result)
	}
}
