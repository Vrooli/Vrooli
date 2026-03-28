package fix

import (
	"context"
	"strings"
	"testing"
	"time"

	"test-genie/agentmanager"
)

func newDisabledAgentService() *agentmanager.AgentService {
	return agentmanager.NewAgentService(agentmanager.Config{
		ProfileName: "Test Genie Agent",
		ProfileKey:  "test-genie",
		Enabled:     false,
		Timeout:     time.Second,
	})
}

func TestSpawnRejectsUnavailableAgentManager(t *testing.T) {
	svc := NewService(newDisabledAgentService())

	result, err := svc.Spawn(context.Background(), SpawnRequest{
		ScenarioName: "test-genie",
	})
	if err == nil || !strings.Contains(err.Error(), "agent-manager is not available") {
		t.Fatalf("expected unavailable-agent error, got result=%v err=%v", result, err)
	}
	if svc.GetActiveForScenario("test-genie") != nil {
		t.Fatal("expected no active fix to be stored when spawn fails early")
	}
}

func TestStopRejectsTerminalAndRunlessFixes(t *testing.T) {
	svc := NewService(newDisabledAgentService())

	now := time.Now()
	svc.store.Create(&Record{
		ID:           "completed",
		ScenarioName: "test-genie",
		Status:       StatusCompleted,
		StartedAt:    now,
		CompletedAt:  &now,
	})
	if err := svc.Stop(context.Background(), "completed"); err == nil || !strings.Contains(err.Error(), "terminal state") {
		t.Fatalf("expected terminal-state error, got %v", err)
	}

	svc.store.Create(&Record{
		ID:           "pending",
		ScenarioName: "test-genie",
		Status:       StatusPending,
		StartedAt:    now,
	})
	if err := svc.Stop(context.Background(), "pending"); err == nil || !strings.Contains(err.Error(), "no associated run") {
		t.Fatalf("expected missing-run error, got %v", err)
	}
}
