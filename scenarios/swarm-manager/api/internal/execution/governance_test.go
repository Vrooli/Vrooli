package execution

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/promptmanager"
)

// stubGovernanceProvider implements GovernanceProvider for tests.
type stubGovernanceProvider struct {
	settings GovernanceSettings
}

func (s *stubGovernanceProvider) LoadGovernance() (GovernanceSettings, error) {
	return s.settings, nil
}

// laneLimits builds a four-lane LaneLimits map with the Execute lane set
// to executeCap. Other lanes default to the production values so tests
// never accidentally exercise lane starvation in lanes they aren't
// stressing.
func laneLimits(executeCap int) map[string]int {
	return map[string]int{
		"investigate": 6,
		"execute":     executeCap,
		"review":      8,
		"reconcile":   2,
	}
}

// findLaneStatus returns the LaneStatus for the named lane, or false if
// not present. Used by GovernanceStatus tests to assert per-lane
// utilization without relying on Lanes() ordering.
func findLaneStatus(lanes []LaneStatus, name string) (LaneStatus, bool) {
	for _, l := range lanes {
		if l.Lane == name {
			return l, true
		}
	}
	return LaneStatus{}, false
}

func TestCountActiveExecutions(t *testing.T) {
	records := []Record{
		{Status: StatusStarting},
		{Status: StatusRunning},
		{Status: StatusPending},
		{Status: StatusNeedsReview},
		{Status: StatusCompleted},
		{Status: StatusFailed},
	}
	got := countActiveExecutions(records)
	if got != 2 {
		t.Fatalf("expected 2 active (starting+running), got %d", got)
	}
}

func TestCountQueuedExecutions(t *testing.T) {
	records := []Record{
		{Status: StatusPending},
		{Status: StatusPending},
		{Status: StatusRunning},
		{Status: StatusStarting},
	}
	got := countQueuedExecutions(records)
	if got != 2 {
		t.Fatalf("expected 2 queued (pending), got %d", got)
	}
}

func TestConcurrencyGate_StartLocked(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "test-item", map[string]any{
		"name":        "test-item",
		"title":       "Test",
		"description": "desc",
		"status":      "queued",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "test-item")

	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")

	// Pre-populate with 2 running records (maxConcurrent=2).
	// RunID must be set to avoid migrateRecords marking them as failed.
	preExisting := []Record{
		{ExecutionID: "running-1", RunID: "run-1", Status: StatusRunning, BacklogKind: "idea", BacklogName: "a", Mode: ModeYOLO, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
		{ExecutionID: "running-2", RunID: "run-2", Status: StatusRunning, BacklogKind: "idea", BacklogName: "b", Mode: ModeYOLO, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
		{ExecutionID: "pending-1", Status: StatusPending, BacklogKind: "idea", BacklogName: "test-item", Mode: ModeManual, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
	}
	store := NewStore(storePath)
	if err := store.Save(preExisting); err != nil {
		t.Fatalf("Save preexisting: %v", err)
	}

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    storePath,
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
		GovernanceProvider: &stubGovernanceProvider{settings: GovernanceSettings{
			LaneLimits: map[string]int{
				"investigate": 6,
				"execute":     2,
				"review":      8,
				"reconcile":   2,
			},
			MaxQueueDepth:                 50,
			CircuitBreakerThreshold:       3,
			CircuitBreakerCooldownMinutes: 60,
			AgentMaxTurns:                 600,
		}},
	})

	// Try to start the pending record — should fail with errAtCapacity.
	_, err := service.Start(context.Background(), "pending-1")
	if err == nil {
		t.Fatal("expected errAtCapacity, got nil")
	}
	if !errors.Is(err, errAtCapacity) {
		t.Fatalf("expected errAtCapacity, got: %v", err)
	}

	// Agent should not have been called.
	if agent.spawnCalls > 0 {
		t.Fatalf("expected no agent spawns, got %d", agent.spawnCalls)
	}
}

// [REQ:SWM-P1-008] execution policy: queue depth cap enforced
func TestQueueDepthEnforcement(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "new-item", map[string]any{
		"name":        "new-item",
		"title":       "Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "new-item")
	// The queue depth test seeds two pending records. They must reference
	// real backlog items on disk or pruneOrphanedPendingRecords would drop
	// them before the depth check, defeating the test.
	mustWriteBacklogItem(t, root, "idea", "seed-a", map[string]any{"name": "seed-a", "status": "queued", "priority": 3, "tags": []string{}})
	mustWriteBacklogItem(t, root, "idea", "seed-b", map[string]any{"name": "seed-b", "status": "queued", "priority": 3, "tags": []string{}})

	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")

	// Pre-populate with 2 pending records (maxQueueDepth=2).
	preExisting := []Record{
		{ExecutionID: "pending-1", Status: StatusPending, BacklogKind: "idea", BacklogName: "seed-a", Mode: ModeManual, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
		{ExecutionID: "pending-2", Status: StatusPending, BacklogKind: "idea", BacklogName: "seed-b", Mode: ModeManual, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
	}
	store := NewStore(storePath)
	if err := store.Save(preExisting); err != nil {
		t.Fatalf("Save preexisting: %v", err)
	}

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    storePath,
		PlanRenderer: testPlanRenderer(),
		AgentService: &stubAgentService{},
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
		GovernanceProvider: &stubGovernanceProvider{settings: GovernanceSettings{
			LaneLimits:                    laneLimits(3),
			MaxQueueDepth:                 2,
			CircuitBreakerThreshold:       3,
			CircuitBreakerCooldownMinutes: 60,
			AgentMaxTurns:                 600,
		}},
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "new-item",
		Mode:        ModeManual,
	})
	if err == nil {
		t.Fatal("expected queue full error, got nil")
	}
	if !strings.Contains(err.Error(), "queue depth limit exceeded") {
		t.Fatalf("expected queue full error, got: %v", err)
	}
}

func TestCostCapEnforcement(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "expensive", map[string]any{
		"name":        "expensive",
		"title":       "Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "expensive")

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: &stubAgentService{},
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
		GovernanceProvider: &stubGovernanceProvider{settings: GovernanceSettings{
			LaneLimits:                    laneLimits(3),
			MaxQueueDepth:                 50,
			CircuitBreakerThreshold:       3,
			CircuitBreakerCooldownMinutes: 60,
			ExecutionCostCapPerRun:        2.0,
			CostPerTurnEstimate:           0.10,
			AgentMaxTurns:                 600,
		}},
	})

	// $0.10 * 60 = $6.00 > $2.00 cap — should fail.
	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "expensive",
		Mode:        ModeManual,
	})
	if err == nil {
		t.Fatal("expected cost cap error, got nil")
	}
	if !strings.Contains(err.Error(), "estimated cost") || !strings.Contains(err.Error(), "exceeds cap") {
		t.Fatalf("expected cost cap error, got: %v", err)
	}

	// With force=true, should succeed.
	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "expensive",
		Mode:        ModeManual,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("expected force to override cost cap, got: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending, got %s", record.Status)
	}
}

func TestYoloAtCapacity_LeavesPending(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "yolo-item", map[string]any{
		"name":        "yolo-item",
		"title":       "Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "yolo-item")

	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")

	// Pre-populate with 1 running (maxConcurrent=1).
	preExisting := []Record{
		{ExecutionID: "running-1", RunID: "run-1", Status: StatusRunning, BacklogKind: "idea", BacklogName: "other", Mode: ModeYOLO, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
	}
	store := NewStore(storePath)
	if err := store.Save(preExisting); err != nil {
		t.Fatalf("Save: %v", err)
	}

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    storePath,
		PlanRenderer: testPlanRenderer(),
		AgentService: &stubAgentService{},
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
		GovernanceProvider: &stubGovernanceProvider{settings: GovernanceSettings{
			LaneLimits:                    laneLimits(1),
			MaxQueueDepth:                 50,
			CircuitBreakerThreshold:       3,
			CircuitBreakerCooldownMinutes: 60,
			AgentMaxTurns:                 600,
		}},
	})

	// YOLO at capacity — should succeed but stay pending.
	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "yolo-item",
		Mode:        ModeYOLO,
	})
	if err != nil {
		t.Fatalf("expected success (pending), got: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending (at capacity), got %s", record.Status)
	}
}

func TestCircuitBreakerBlocksQueue(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "broken-item", map[string]any{
		"name":        "broken-item",
		"title":       "Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "broken-item")

	cbPath := filepath.Join(root, ".vrooli", "circuit-breaker.json")
	cb := NewCircuitBreaker(cbPath)
	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_ = cb.RecordFailure("idea/broken-item", 3)
	}

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: &stubAgentService{},
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
		GovernanceProvider: &stubGovernanceProvider{settings: GovernanceSettings{
			LaneLimits:                    laneLimits(3),
			MaxQueueDepth:                 50,
			CircuitBreakerThreshold:       3,
			CircuitBreakerCooldownMinutes: 60,
			AgentMaxTurns:                 600,
		}},
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "broken-item",
		Mode:        ModeManual,
	})
	if err == nil {
		t.Fatal("expected circuit breaker error, got nil")
	}
	if !strings.Contains(err.Error(), "circuit breaker tripped") {
		t.Fatalf("expected circuit breaker error, got: %v", err)
	}
}

func TestGovernanceStatus(t *testing.T) {
	root := t.TempDir()
	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")

	preExisting := []Record{
		{ExecutionID: "running-1", RunID: "run-1", Status: StatusRunning, BacklogKind: "idea", BacklogName: "a", CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
		{ExecutionID: "pending-1", Status: StatusPending, BacklogKind: "idea", BacklogName: "b", CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()},
	}
	store := NewStore(storePath)
	if err := store.Save(preExisting); err != nil {
		t.Fatalf("Save: %v", err)
	}

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    storePath,
		PlanRenderer: testPlanRenderer(),
		GovernanceProvider: &stubGovernanceProvider{settings: GovernanceSettings{
			LaneLimits:                    laneLimits(3),
			MaxQueueDepth:                 50,
			CircuitBreakerThreshold:       3,
			CircuitBreakerCooldownMinutes: 60,
			CostPerTurnEstimate:           0.10,
			AgentMaxTurns:                 600,
		}},
	})

	status, err := service.GovernanceStatus()
	if err != nil {
		t.Fatalf("GovernanceStatus: %v", err)
	}
	if status.ActiveExecutions != 1 {
		t.Fatalf("expected 1 active, got %d", status.ActiveExecutions)
	}
	if got := len(status.Lanes); got != 4 {
		t.Fatalf("expected 4 lanes, got %d", got)
	}
	executeLane, ok := findLaneStatus(status.Lanes, "execute")
	if !ok {
		t.Fatal("expected lanes to include execute")
	}
	if executeLane.Capacity != 3 {
		t.Fatalf("execute lane capacity = %d, want 3", executeLane.Capacity)
	}
	if executeLane.Active != 1 {
		t.Fatalf("execute lane active = %d, want 1", executeLane.Active)
	}
	if executeLane.Queue != 1 {
		t.Fatalf("execute lane queue = %d, want 1", executeLane.Queue)
	}
	if status.QueueDepth != 1 {
		t.Fatalf("expected 1 queued, got %d", status.QueueDepth)
	}
}
