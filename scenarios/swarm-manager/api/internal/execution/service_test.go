package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"
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
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
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
	if started.PromptTrace == nil {
		t.Fatal("expected prompt trace to be captured")
	}
	if started.PromptTrace.SkillID != "swarm-manager-process-idea" {
		t.Fatalf("expected process idea prompt skill ID, got %q", started.PromptTrace.SkillID)
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

func TestQueueBacklog_RejectsDelayForNonScheduledModes(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "delay-manual", map[string]any{
		"name":        "delay-manual",
		"title":       "Delay Manual",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind:  "idea",
		BacklogName:  "delay-manual",
		Mode:         ModeManual,
		DelaySeconds: 10,
	})
	if err == nil {
		t.Fatal("expected error when delay_seconds is provided for manual mode")
	}
}

func TestQueueBacklog_RejectsScheduledModeWithoutEffectiveDelay(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "scheduled-no-delay", map[string]any{
		"name":        "scheduled-no-delay",
		"title":       "Scheduled No Delay",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	testPolicyPath := filepath.Join(root, ".vrooli", "execution-policy.json")
	mustWritePolicy(t, testPolicyPath, map[string]any{
		"default_mode":          "scheduled",
		"default_delay_seconds": 0,
	})

	service := NewService(ServiceConfig{
		RootDir:    root,
		StorePath:  filepath.Join(root, ".vrooli", "execution-runs.json"),
		PolicyPath: testPolicyPath,
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "scheduled-no-delay",
		Mode:        ModeScheduled,
	})
	if err == nil {
		t.Fatal("expected error for scheduled mode without effective delay")
	}
}

func TestQueueBacklog_AllowsArchivedIdeas(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-idea", map[string]any{
		"name":        "archived-idea",
		"title":       "Archived Idea",
		"description": "desc",
		"status":      "archived",
		"priority":    3,
		"tags":        []string{},
	})

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "archived-idea",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}
}

func TestUpdatePolicy_RejectsInvalidScheduledDefaults(t *testing.T) {
	root := t.TempDir()
	service := NewService(ServiceConfig{
		RootDir:    root,
		StorePath:  filepath.Join(root, ".vrooli", "execution-runs.json"),
		PolicyPath: filepath.Join(root, ".vrooli", "execution-policy.json"),
	})

	_, err := service.UpdatePolicy(context.Background(), Policy{
		DefaultMode:         ModeScheduled,
		DefaultDelaySeconds: 0,
	})
	if err == nil {
		t.Fatal("expected error when scheduled default mode has non-positive delay")
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

func mustWritePolicy(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal policy payload: %v", err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		t.Fatalf("write policy file: %v", err)
	}
}
