package toolexecution

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
)

// mockOrchestrator implements the Orchestrator interface for testing.
type mockOrchestrator struct {
	createTaskFn func(ctx context.Context, task *domain.Task) (*domain.Task, error)
	createRunFn  func(ctx context.Context, req orchestration.CreateRunRequest) (*domain.Run, error)
	getRunFn     func(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	listRunsFn   func(ctx context.Context, opts orchestration.RunListOptions) ([]*domain.Run, error)
	stopRunFn    func(ctx context.Context, id uuid.UUID) error
	getRunDiffFn func(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error)
	approveRunFn func(ctx context.Context, req orchestration.ApproveRequest) (*orchestration.ApproveResult, error)
}

// Verify mockOrchestrator implements Orchestrator interface
var _ Orchestrator = (*mockOrchestrator)(nil)

// Orchestrator interface implementation
func (m *mockOrchestrator) CreateTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if m.createTaskFn != nil {
		return m.createTaskFn(ctx, task)
	}
	task.ID = uuid.New()
	task.CreatedAt = time.Now()
	return task, nil
}

func (m *mockOrchestrator) CreateRun(ctx context.Context, req orchestration.CreateRunRequest) (*domain.Run, error) {
	if m.createRunFn != nil {
		return m.createRunFn(ctx, req)
	}
	return &domain.Run{
		ID:        uuid.New(),
		TaskID:    req.TaskID,
		Status:    domain.RunStatusPending,
		Phase:     domain.RunPhaseQueued,
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockOrchestrator) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	if m.getRunFn != nil {
		return m.getRunFn(ctx, id)
	}
	return nil, &domain.NotFoundError{EntityType: "run", ID: id.String()}
}

func (m *mockOrchestrator) ListRuns(ctx context.Context, opts orchestration.RunListOptions) ([]*domain.Run, error) {
	if m.listRunsFn != nil {
		return m.listRunsFn(ctx, opts)
	}
	return []*domain.Run{}, nil
}

func (m *mockOrchestrator) StopRun(ctx context.Context, id uuid.UUID) error {
	if m.stopRunFn != nil {
		return m.stopRunFn(ctx, id)
	}
	return nil
}

func (m *mockOrchestrator) GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error) {
	if m.getRunDiffFn != nil {
		return m.getRunDiffFn(ctx, runID)
	}
	return nil, &domain.NotFoundError{EntityType: "run", ID: runID.String()}
}

func (m *mockOrchestrator) ApproveRun(ctx context.Context, req orchestration.ApproveRequest) (*orchestration.ApproveResult, error) {
	if m.approveRunFn != nil {
		return m.approveRunFn(ctx, req)
	}
	return &orchestration.ApproveResult{Success: true}, nil
}

// --- Tests ---

func TestExecute_UnknownTool(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		Orchestrator: &mockOrchestrator{},
	})

	result, err := executor.Execute(context.Background(), "unknown_tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for unknown tool")
	}
	if result.Code != CodeUnknownTool {
		t.Errorf("expected code %q, got %q", CodeUnknownTool, result.Code)
	}
}

func TestSpawnCodingAgent_Success(t *testing.T) {
	taskID := uuid.New()
	runID := uuid.New()

	mock := &mockOrchestrator{
		createTaskFn: func(ctx context.Context, task *domain.Task) (*domain.Task, error) {
			task.ID = taskID
			task.CreatedAt = time.Now()
			return task, nil
		},
		createRunFn: func(ctx context.Context, req orchestration.CreateRunRequest) (*domain.Run, error) {
			return &domain.Run{
				ID:        runID,
				TaskID:    req.TaskID,
				Status:    domain.RunStatusPending,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	result, err := executor.Execute(context.Background(), "spawn_coding_agent", map[string]interface{}{
		"task": "Write a hello world program",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !result.IsAsync {
		t.Error("spawn_coding_agent should return async result")
	}
	if result.RunID != runID.String() {
		t.Errorf("expected run_id %s, got %s", runID.String(), result.RunID)
	}
}

func TestSpawnCodingAgent_MissingTask(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		Orchestrator: &mockOrchestrator{},
	})

	result, err := executor.Execute(context.Background(), "spawn_coding_agent", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing task")
	}
	if result.Code != CodeInvalidArgs {
		t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
	}
}

func TestSpawnCodingAgent_WithContextAttachments(t *testing.T) {
	var capturedTask *domain.Task

	mock := &mockOrchestrator{
		createTaskFn: func(ctx context.Context, task *domain.Task) (*domain.Task, error) {
			capturedTask = task
			task.ID = uuid.New()
			return task, nil
		},
		createRunFn: func(ctx context.Context, req orchestration.CreateRunRequest) (*domain.Run, error) {
			return &domain.Run{
				ID:        uuid.New(),
				TaskID:    req.TaskID,
				Status:    domain.RunStatusPending,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	_, err := executor.Execute(context.Background(), "spawn_coding_agent", map[string]interface{}{
		"task": "Test task",
		"_context_attachments": []interface{}{
			map[string]interface{}{
				"type":    "skill",
				"key":     "test-skill",
				"label":   "Test Skill",
				"content": "Some skill content",
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTask == nil {
		t.Fatal("task was not captured")
	}
	if len(capturedTask.ContextAttachments) != 1 {
		t.Errorf("expected 1 context attachment, got %d", len(capturedTask.ContextAttachments))
	}
	if capturedTask.ContextAttachments[0].Key != "test-skill" {
		t.Errorf("expected key 'test-skill', got %q", capturedTask.ContextAttachments[0].Key)
	}
}

func TestCheckAgentStatus_Success(t *testing.T) {
	runID := uuid.New()
	taskID := uuid.New()

	mock := &mockOrchestrator{
		getRunFn: func(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
			return &domain.Run{
				ID:        runID,
				TaskID:    taskID,
				Status:    domain.RunStatusRunning,
				Phase:     domain.RunPhaseExecuting,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	result, err := executor.Execute(context.Background(), "check_agent_status", map[string]interface{}{
		"run_id": runID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	runData, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	run, ok := runData["run"].(map[string]interface{})
	if !ok {
		t.Fatal("expected run field in result")
	}
	if run["status"] != "running" {
		t.Errorf("expected status 'running', got %v", run["status"])
	}
}

func TestCheckAgentStatus_NotFound(t *testing.T) {
	mock := &mockOrchestrator{
		getRunFn: func(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
			return nil, &domain.NotFoundError{EntityType: "run", ID: id.String()}
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	result, err := executor.Execute(context.Background(), "check_agent_status", map[string]interface{}{
		"run_id": uuid.New().String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for not found")
	}
	if result.Code != CodeNotFound {
		t.Errorf("expected code %q, got %q", CodeNotFound, result.Code)
	}
}

func TestCheckAgentStatus_MissingRunID(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		Orchestrator: &mockOrchestrator{},
	})

	result, err := executor.Execute(context.Background(), "check_agent_status", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing run_id")
	}
	if result.Code != CodeInvalidArgs {
		t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
	}
}

func TestStopAgent_Success(t *testing.T) {
	runID := uuid.New()
	stopCalled := false

	mock := &mockOrchestrator{
		stopRunFn: func(ctx context.Context, id uuid.UUID) error {
			stopCalled = true
			if id != runID {
				t.Errorf("expected run_id %s, got %s", runID.String(), id.String())
			}
			return nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	result, err := executor.Execute(context.Background(), "stop_agent", map[string]interface{}{
		"run_id": runID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !stopCalled {
		t.Error("stopRun was not called")
	}
}

func TestListActiveAgents_Success(t *testing.T) {
	runID1 := uuid.New()
	runID2 := uuid.New()

	mock := &mockOrchestrator{
		listRunsFn: func(ctx context.Context, opts orchestration.RunListOptions) ([]*domain.Run, error) {
			// Verify we're filtering by running status
			if opts.Status == nil || *opts.Status != domain.RunStatusRunning {
				t.Error("expected filter by running status")
			}
			return []*domain.Run{
				{ID: runID1, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, CreatedAt: time.Now()},
				{ID: runID2, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, CreatedAt: time.Now()},
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	result, err := executor.Execute(context.Background(), "list_active_agents", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	runs, ok := data["runs"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected runs field to be a slice")
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestGetAgentDiff_Success(t *testing.T) {
	runID := uuid.New()
	sandboxID := uuid.New()

	mock := &mockOrchestrator{
		getRunDiffFn: func(ctx context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
			return &sandbox.DiffResult{
				SandboxID:   sandboxID,
				UnifiedDiff: "diff --git a/test.go b/test.go\n+new line",
				Files:       []sandbox.FileChange{},
				Stats: sandbox.DiffStats{
					FilesChanged: 1,
					LinesAdded:   1,
					LinesRemoved: 0,
				},
				Generated: time.Now(),
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	result, err := executor.Execute(context.Background(), "get_agent_diff", map[string]interface{}{
		"run_id": runID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	if _, ok := data["unified_diff"]; !ok {
		t.Error("expected unified_diff in result")
	}
}

func TestApproveAgentChanges_Success(t *testing.T) {
	runID := uuid.New()

	mock := &mockOrchestrator{
		approveRunFn: func(ctx context.Context, req orchestration.ApproveRequest) (*orchestration.ApproveResult, error) {
			if req.RunID != runID {
				t.Errorf("expected run_id %s, got %s", runID.String(), req.RunID.String())
			}
			if req.Actor != "agent-inbox" {
				t.Errorf("expected actor 'agent-inbox', got %q", req.Actor)
			}
			return &orchestration.ApproveResult{
				Success:    true,
				Applied:    3,
				Remaining:  0,
				IsPartial:  false,
				CommitHash: "abc123",
			}, nil
		},
	}

	executor := NewServerExecutor(ServerExecutorConfig{Orchestrator: mock})

	result, err := executor.Execute(context.Background(), "approve_agent_changes", map[string]interface{}{
		"run_id": runID.String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	if data["commit_hash"] != "abc123" {
		t.Errorf("expected commit_hash 'abc123', got %v", data["commit_hash"])
	}
}

func TestInvalidRunID(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{
		Orchestrator: &mockOrchestrator{},
	})

	tools := []string{"check_agent_status", "stop_agent", "get_agent_diff", "approve_agent_changes"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tool, map[string]interface{}{
				"run_id": "not-a-valid-uuid",
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Success {
				t.Error("expected failure for invalid run_id")
			}
			if result.Code != CodeInvalidArgs {
				t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
			}
		})
	}
}
