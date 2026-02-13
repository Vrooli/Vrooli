package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
)

type stubAgentService struct {
	spawnCalls int
}

func (s *stubAgentService) IsEnabled() bool { return true }

func (s *stubAgentService) SpawnBacklog(_ context.Context, _ agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	s.spawnCalls++
	return agentmanager.RunResult{TaskID: "task-1", RunID: "run-1"}, nil
}

func TestQueueAndStartManualExecution(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "test-idea", map[string]any{
		"name":        "test-idea",
		"title":       "Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "test-idea",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "test-idea", "spec.json"))
	if storedItem["status"] != "queued" {
		t.Fatalf("expected backlog status queued, got %#v", storedItem["status"])
	}

	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if started.Status != StatusRunning {
		t.Fatalf("expected running status, got %s", started.Status)
	}
	if started.TaskID != "task-1" || started.RunID != "run-1" {
		t.Fatalf("expected task/run IDs set, got task=%s run=%s", started.TaskID, started.RunID)
	}
	if agent.spawnCalls != 1 {
		t.Fatalf("expected 1 spawn call, got %d", agent.spawnCalls)
	}
}

func TestQueueBacklog_UsesPolicyDefaultsWhenModeMissing(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "policy-idea", map[string]any{
		"name":        "policy-idea",
		"title":       "Policy Idea",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PolicyPath:   filepath.Join(root, ".vrooli", "execution-policy.json"),
		AgentService: agent,
	})
	_, err := service.UpdatePolicy(context.Background(), Policy{
		DefaultMode:         ModeScheduled,
		DefaultDelaySeconds: 600,
	})
	if err != nil {
		t.Fatalf("UpdatePolicy error: %v", err)
	}

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "policy-idea",
		Mode:        "",
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Mode != ModeScheduled {
		t.Fatalf("expected scheduled mode from policy, got %s", record.Mode)
	}
	if record.Status != StatusScheduled {
		t.Fatalf("expected scheduled status, got %s", record.Status)
	}
	if record.ScheduledAt == "" {
		t.Fatalf("expected scheduled_at to be populated")
	}
}

func mustWriteBacklogItem(t *testing.T, root, kind, name string, payload map[string]any) {
	t.Helper()
	kindDir := "ideas"
	switch kind {
	case "research":
		kindDir = "research"
	case "fix":
		kindDir = "fix"
	case "execute":
		kindDir = "execute"
	}
	dir := filepath.Join(root, kindDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir backlog item: %v", err)
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), bytes, 0o644); err != nil {
		t.Fatalf("write spec.json: %v", err)
	}
}

func mustLoadBacklogItem(t *testing.T, path string) map[string]any {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(bytes, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return value
}
