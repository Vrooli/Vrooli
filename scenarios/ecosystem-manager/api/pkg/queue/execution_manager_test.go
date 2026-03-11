package queue

import (
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/steering"
	"github.com/ecosystem-manager/api/pkg/tasks"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Mock steering provider ---

type mockSteeringProvider struct {
	strategy         steering.SteeringStrategy
	afterDecision    *steering.SteeringDecision
	afterErr         error
	enhanceResult    *steering.PromptEnhancement
	enhanceErr       error
	currentSet       []string
	currentSetErr    error
	initErr          error
	resetErr         error
	afterCallCount   int
	enhanceCallCount int
}

var _ steering.SteeringProvider = (*mockSteeringProvider)(nil)

func (m *mockSteeringProvider) Strategy() steering.SteeringStrategy { return m.strategy }

func (m *mockSteeringProvider) GetCurrentSet(_ *tasks.TaskItem) ([]string, error) {
	return m.currentSet, m.currentSetErr
}

func (m *mockSteeringProvider) EnhancePrompt(_ *tasks.TaskItem) (*steering.PromptEnhancement, error) {
	m.enhanceCallCount++
	return m.enhanceResult, m.enhanceErr
}

func (m *mockSteeringProvider) AfterExecution(_ *tasks.TaskItem, _ string) (*steering.SteeringDecision, error) {
	m.afterCallCount++
	return m.afterDecision, m.afterErr
}

func (m *mockSteeringProvider) Initialize(_ *tasks.TaskItem) error { return m.initErr }
func (m *mockSteeringProvider) Reset(_ string) error               { return m.resetErr }

// --- Mock steering registry ---

type mockSteeringRegistry struct {
	provider steering.SteeringProvider
	strategy steering.SteeringStrategy
}

var _ steering.RegistryAPI = (*mockSteeringRegistry)(nil)

func (m *mockSteeringRegistry) GetProvider(_ *tasks.TaskItem) steering.SteeringProvider {
	return m.provider
}

func (m *mockSteeringRegistry) DetermineStrategy(_ *tasks.TaskItem) steering.SteeringStrategy {
	return m.strategy
}

// --- mapAgentManagerResult tests ---

func TestMapAgentManagerResult_ErrorMsgWith429(t *testing.T) {
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status:   domainpb.RunStatus_RUN_STATUS_FAILED,
		ErrorMsg: "error 429: rate limit exceeded",
	}

	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t1"}, "agent", "some output", nil)

	if resp.Success {
		t.Fatal("expected success=false")
	}
	if !resp.RateLimited {
		t.Fatal("expected RateLimited=true for error 429 in ErrorMsg")
	}
}

func TestMapAgentManagerResult_SummaryWithRateLimit(t *testing.T) {
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
		Summary: &domainpb.RunSummary{
			Description: "Rate limit exceeded - please try again later",
		},
	}

	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t2"}, "agent", "normal output", nil)

	if resp.Success {
		t.Fatal("expected success=false when summary mentions rate limit")
	}
	if !resp.RateLimited {
		t.Fatal("expected RateLimited=true when summary mentions rate limit")
	}
}

func TestMapAgentManagerResult_OutputOnlyRateLimitIgnored(t *testing.T) {
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
	}
	// Output mentions rate limits but error/summary don't
	output := "I implemented rate limit handling.\nLine 429 fixed.\nUsage limit docs updated."

	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t3"}, "agent", output, nil)

	if !resp.Success {
		t.Fatal("expected success=true when rate limit text is only in output")
	}
	if resp.RateLimited {
		t.Fatal("expected RateLimited=false when rate limit text is only in output")
	}
}

func TestMapAgentManagerResult_MaxTurnsExceeded(t *testing.T) {
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status:   domainpb.RunStatus_RUN_STATUS_FAILED,
		ErrorMsg: "max turns reached",
	}

	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t4"}, "agent", "max turns reached", nil)

	if resp.Success {
		t.Fatal("expected success=false for max_turns_exceeded")
	}
	if !resp.MaxTurnsExceeded {
		t.Fatal("expected MaxTurnsExceeded=true")
	}
}

func TestMapAgentManagerResult_TimeoutDetected(t *testing.T) {
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status:   domainpb.RunStatus_RUN_STATUS_FAILED,
		ErrorMsg: "timeout reached",
	}

	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t5"}, "agent", "", nil)

	if resp.Success {
		t.Fatal("expected success=false for timeout")
	}
	if resp.Error == "" || resp.Error == "timeout reached" {
		t.Fatal("expected timeout error to be prefixed with TIMEOUT")
	}
}

func TestMapAgentManagerResult_NormalSuccess(t *testing.T) {
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
		Summary: &domainpb.RunSummary{
			Description: "Task completed successfully",
		},
	}

	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t6"}, "agent", "all done", nil)

	if !resp.Success {
		t.Fatal("expected success=true for normal completion")
	}
	if resp.RateLimited {
		t.Fatal("expected RateLimited=false for normal completion")
	}
	if resp.Message != "Task completed successfully" {
		t.Fatalf("expected message from summary, got: %s", resp.Message)
	}
}

func TestMapAgentManagerResult_DurationRecordedInHistory(t *testing.T) {
	em := &ExecutionManager{}
	now := timestamppb.Now()
	later := timestamppb.New(now.AsTime().Add(60 * time.Second))
	run := &domainpb.Run{
		Status:    domainpb.RunStatus_RUN_STATUS_COMPLETE,
		StartedAt: now,
		EndedAt:   later,
	}
	history := &ExecutionHistory{}

	em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t7"}, "agent", "", history)

	if history.Duration == "" {
		t.Fatal("expected duration to be recorded in history")
	}
}

func TestMapAgentManagerResult_AlreadyRateLimitedNotOverridden(t *testing.T) {
	// If the error message already detected rate limiting, the defensive
	// detection should skip re-scanning (guarded by !response.RateLimited).
	em := &ExecutionManager{}
	run := &domainpb.Run{
		Status:   domainpb.RunStatus_RUN_STATUS_FAILED,
		ErrorMsg: "rate limit exceeded",
	}

	resp := em.mapAgentManagerResult(run, tasks.TaskItem{ID: "t8"}, "agent", "", nil)

	if !resp.RateLimited {
		t.Fatal("expected RateLimited=true")
	}
	if resp.RetryAfter != DefaultRateLimitRetry {
		t.Fatalf("expected default retry %d, got %d", DefaultRateLimitRetry, resp.RetryAfter)
	}
}

// --- handleSteeringContinuation tests ---

func TestHandleSteeringContinuation_SkipsNonScenarioImprover(t *testing.T) {
	tests := []struct {
		name      string
		taskType  string
		operation string
	}{
		{"resource_generator", "resource", "generator"},
		{"resource_improver", "resource", "improver"},
		{"scenario_generator", "scenario", "generator"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := &ExecutionManager{}
			task := &tasks.TaskItem{
				ID:        "non-improver",
				Type:      tc.taskType,
				Operation: tc.operation,
				Status:    "completed",
			}

			em.handleSteeringContinuation(task, false)

			if task.Status != "completed" {
				t.Fatalf("expected status unchanged for %s/%s, got %s", tc.taskType, tc.operation, task.Status)
			}
		})
	}
}

func TestHandleSteeringContinuation_NoRegistryWarns(t *testing.T) {
	em := &ExecutionManager{steeringRegistry: nil}
	task := &tasks.TaskItem{
		ID:        "no-registry",
		Type:      "scenario",
		Operation: "improver",
		Status:    "completed",
	}

	em.handleSteeringContinuation(task, false)

	if task.Status != "completed" {
		t.Fatalf("expected status unchanged, got %s", task.Status)
	}
}

func TestHandleSteeringContinuation_ExhaustedMovesToFinalized(t *testing.T) {
	provider := &mockSteeringProvider{
		strategy: steering.StrategyProfile,
		afterDecision: &steering.SteeringDecision{
			Exhausted:     true,
			ShouldRequeue: false,
			Reason:        "all phases completed",
		},
	}
	registry := &mockSteeringRegistry{provider: provider, strategy: steering.StrategyProfile}
	em := &ExecutionManager{steeringRegistry: registry}

	task := &tasks.TaskItem{
		ID:                   "exhausted-task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "completed",
		ProcessorAutoRequeue: true,
	}

	em.handleSteeringContinuation(task, false)

	if task.Status != tasks.StatusCompletedFinalized {
		t.Fatalf("expected status=completed-finalized when exhausted, got %s", task.Status)
	}
	if task.ProcessorAutoRequeue {
		t.Fatal("expected ProcessorAutoRequeue=false when exhausted")
	}
}

func TestHandleSteeringContinuation_ShouldNotRequeueMovesToFinalized(t *testing.T) {
	provider := &mockSteeringProvider{
		strategy: steering.StrategyManual,
		afterDecision: &steering.SteeringDecision{
			Exhausted:     false,
			ShouldRequeue: false,
			Reason:        "manual stop",
		},
	}
	registry := &mockSteeringRegistry{provider: provider, strategy: steering.StrategyManual}
	em := &ExecutionManager{steeringRegistry: registry}

	task := &tasks.TaskItem{
		ID:                   "no-requeue-task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "completed",
		ProcessorAutoRequeue: true,
	}

	em.handleSteeringContinuation(task, false)

	if task.Status != tasks.StatusCompletedFinalized {
		t.Fatalf("expected status=completed-finalized when ShouldRequeue=false, got %s", task.Status)
	}
}

func TestHandleSteeringContinuation_ContinuousRequeuesToPending(t *testing.T) {
	provider := &mockSteeringProvider{
		strategy: steering.StrategyQueue,
		afterDecision: &steering.SteeringDecision{
			Exhausted:     false,
			ShouldRequeue: true,
			Reason:        "next phase",
		},
	}
	registry := &mockSteeringRegistry{provider: provider, strategy: steering.StrategyQueue}
	em := &ExecutionManager{steeringRegistry: registry}

	task := &tasks.TaskItem{
		ID:                   "continue-task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "completed",
		ProcessorAutoRequeue: true,
	}

	em.handleSteeringContinuation(task, false)

	if task.Status != "pending" {
		t.Fatalf("expected status=pending when continuing, got %s", task.Status)
	}
	if !task.ProcessorAutoRequeue {
		t.Fatal("ProcessorAutoRequeue should remain true when continuing")
	}
}

func TestHandleSteeringContinuation_ContinueIgnoredWhenAutoRequeueDisabled(t *testing.T) {
	provider := &mockSteeringProvider{
		strategy: steering.StrategyQueue,
		afterDecision: &steering.SteeringDecision{
			ShouldRequeue: true,
			Reason:        "next phase",
		},
	}
	registry := &mockSteeringRegistry{provider: provider, strategy: steering.StrategyQueue}
	em := &ExecutionManager{steeringRegistry: registry}

	task := &tasks.TaskItem{
		ID:                   "locked-task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "completed",
		ProcessorAutoRequeue: false,
	}

	em.handleSteeringContinuation(task, false)

	// Status should remain "completed" since ProcessorAutoRequeue is false
	if task.Status != "completed" {
		t.Fatalf("expected status=completed when auto-requeue disabled, got %s", task.Status)
	}
}

func TestHandleSteeringContinuation_ErrorMovesToFinalized(t *testing.T) {
	provider := &mockSteeringProvider{
		strategy:      steering.StrategyProfile,
		afterDecision: nil,
		afterErr:      errSteeringFailed,
	}
	registry := &mockSteeringRegistry{provider: provider, strategy: steering.StrategyProfile}
	em := &ExecutionManager{steeringRegistry: registry}

	task := &tasks.TaskItem{
		ID:                   "error-task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "completed",
		ProcessorAutoRequeue: true,
	}

	em.handleSteeringContinuation(task, false)

	if task.Status != tasks.StatusCompletedFinalized {
		t.Fatalf("expected status=completed-finalized on error, got %s", task.Status)
	}
	if task.ProcessorAutoRequeue {
		t.Fatal("expected ProcessorAutoRequeue=false on error")
	}
}

func TestHandleSteeringContinuation_NoProviderLeavesUnchanged(t *testing.T) {
	registry := &mockSteeringRegistry{provider: nil, strategy: steering.StrategyNone}
	em := &ExecutionManager{steeringRegistry: registry}

	task := &tasks.TaskItem{
		ID:        "no-provider-task",
		Type:      "scenario",
		Operation: "improver",
		Status:    "completed",
	}

	em.handleSteeringContinuation(task, false)

	if task.Status != "completed" {
		t.Fatalf("expected status unchanged when no provider, got %s", task.Status)
	}
}

// --- processExecutionResult routing tests ---

func TestProcessExecutionResult_RoutesSuccess(t *testing.T) {
	var finalizeCalled bool
	var finalizeStatus string

	em := &ExecutionManager{
		storage:   &mockStorageForExec{},
		assembler: &mockAssemblerForExec{},
		registry:  NewMockExecutionRegistry(),
		broadcast: make(chan any, 10),
	}
	em.SetFinalizeFunc(func(task *tasks.TaskItem, status string) error {
		finalizeCalled = true
		finalizeStatus = status
		return nil
	})

	result := &tasks.ClaudeCodeResponse{
		Success: true,
		Output:  "done",
		Message: "completed",
	}
	task := &tasks.TaskItem{
		ID:        "success-route",
		Type:      "resource",
		Operation: "generator",
		Status:    "in-progress",
	}
	history := &ExecutionHistory{}
	now := time.Now()

	execResult, err := em.processExecutionResult(result, task, history, "exec-1", now, 30*time.Second, 30*time.Minute, 1.0, false, func() {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !execResult.Success {
		t.Fatal("expected Success=true in result")
	}
	if !finalizeCalled {
		t.Fatal("expected finalize to be called")
	}
	if finalizeStatus != "completed" {
		t.Fatalf("expected finalize status=completed, got %s", finalizeStatus)
	}
}

func TestProcessExecutionResult_RoutesRateLimited(t *testing.T) {
	var finalizeStatus string

	em := &ExecutionManager{
		broadcast: make(chan any, 10),
		registry:  NewMockExecutionRegistry(),
	}
	em.SetFinalizeFunc(func(task *tasks.TaskItem, status string) error {
		finalizeStatus = status
		return nil
	})

	result := &tasks.ClaudeCodeResponse{
		Success:     false,
		RateLimited: true,
		RetryAfter:  600,
	}
	task := &tasks.TaskItem{
		ID:     "ratelimit-route",
		Status: "in-progress",
	}
	history := &ExecutionHistory{}
	now := time.Now()

	execResult, err := em.processExecutionResult(result, task, history, "exec-2", now, 5*time.Second, 30*time.Minute, 1.0, false, func() {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execResult.Success {
		t.Fatal("expected Success=false for rate limited")
	}
	if !execResult.RateLimited {
		t.Fatal("expected RateLimited=true")
	}
	if finalizeStatus != "pending" {
		t.Fatalf("expected finalize status=pending for rate limited, got %s", finalizeStatus)
	}
}

func TestProcessExecutionResult_RoutesFailure(t *testing.T) {
	var finalizeStatus string

	em := &ExecutionManager{
		storage:   &mockStorageForExec{},
		assembler: &mockAssemblerForExec{},
		registry:  NewMockExecutionRegistry(),
		broadcast: make(chan any, 10),
	}
	em.SetFinalizeFunc(func(task *tasks.TaskItem, status string) error {
		finalizeStatus = status
		return nil
	})

	result := &tasks.ClaudeCodeResponse{
		Success: false,
		Error:   "something broke",
	}
	task := &tasks.TaskItem{
		ID:     "fail-route",
		Status: "in-progress",
	}
	history := &ExecutionHistory{}
	now := time.Now()

	execResult, err := em.processExecutionResult(result, task, history, "exec-3", now, 10*time.Second, 30*time.Minute, 1.0, false, func() {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execResult.Success {
		t.Fatal("expected Success=false for failure")
	}
	if finalizeStatus != "failed" {
		t.Fatalf("expected finalize status=failed, got %s", finalizeStatus)
	}
}

func TestProcessExecutionResult_SuccessWithSteeringContinuation(t *testing.T) {
	// Verify that successful execution correctly delegates to steering
	// continuation, which can change the finalized status to "pending"
	provider := &mockSteeringProvider{
		strategy: steering.StrategyQueue,
		afterDecision: &steering.SteeringDecision{
			ShouldRequeue: true,
			Reason:        "continue",
		},
	}
	registry := &mockSteeringRegistry{provider: provider, strategy: steering.StrategyQueue}

	var finalizeStatus string
	em := &ExecutionManager{
		storage:          &mockStorageForExec{},
		assembler:        &mockAssemblerForExec{},
		steeringRegistry: registry,
		registry:         NewMockExecutionRegistry(),
		broadcast:        make(chan any, 10),
	}
	em.SetFinalizeFunc(func(task *tasks.TaskItem, status string) error {
		finalizeStatus = status
		return nil
	})

	result := &tasks.ClaudeCodeResponse{
		Success: true,
		Output:  "done",
	}
	task := &tasks.TaskItem{
		ID:                   "steer-continue",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "in-progress",
		ProcessorAutoRequeue: true,
	}
	history := &ExecutionHistory{}
	now := time.Now()

	_, err := em.processExecutionResult(result, task, history, "exec-4", now, 30*time.Second, 30*time.Minute, 1.0, false, func() {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Steering said continue, so finalize should be called with "pending"
	if finalizeStatus != "pending" {
		t.Fatalf("expected finalize status=pending (steering continuation), got %s", finalizeStatus)
	}
	if provider.afterCallCount != 1 {
		t.Fatalf("expected AfterExecution to be called once, got %d", provider.afterCallCount)
	}
}

func TestProcessExecutionResult_SuccessWithSteeringExhausted(t *testing.T) {
	// Verify that steering exhaustion correctly finalizes to completed-finalized
	provider := &mockSteeringProvider{
		strategy: steering.StrategyProfile,
		afterDecision: &steering.SteeringDecision{
			Exhausted:     true,
			ShouldRequeue: false,
			Reason:        "all done",
		},
	}
	registry := &mockSteeringRegistry{provider: provider, strategy: steering.StrategyProfile}

	var finalizeStatus string
	em := &ExecutionManager{
		storage:          &mockStorageForExec{},
		assembler:        &mockAssemblerForExec{},
		steeringRegistry: registry,
		registry:         NewMockExecutionRegistry(),
		broadcast:        make(chan any, 10),
	}
	em.SetFinalizeFunc(func(task *tasks.TaskItem, status string) error {
		finalizeStatus = status
		return nil
	})

	result := &tasks.ClaudeCodeResponse{
		Success: true,
	}
	task := &tasks.TaskItem{
		ID:                   "steer-exhausted",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "in-progress",
		ProcessorAutoRequeue: true,
	}
	history := &ExecutionHistory{}
	now := time.Now()

	_, err := em.processExecutionResult(result, task, history, "exec-5", now, 30*time.Second, 30*time.Minute, 1.0, false, func() {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if finalizeStatus != tasks.StatusCompletedFinalized {
		t.Fatalf("expected finalize status=completed-finalized when steering exhausted, got %s", finalizeStatus)
	}
}

// --- Helpers ---

var errSteeringFailed = errSentinel("steering evaluation failed")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

// mockStorageForExec provides minimal StorageAPI for execution tests.
type mockStorageForExec struct{}

func (m *mockStorageForExec) GetTaskByID(_ string) (*tasks.TaskItem, string, error) {
	return nil, "", nil
}

func (m *mockStorageForExec) SaveQueueItem(_ tasks.TaskItem, _ string) error            { return nil }
func (m *mockStorageForExec) SaveQueueItemSkipCleanup(_ tasks.TaskItem, _ string) error { return nil }

func (m *mockStorageForExec) MoveTaskTo(_ string, _ string) (*tasks.TaskItem, string, error) {
	return nil, "", nil
}

func (m *mockStorageForExec) GetQueueItems(_ string) ([]tasks.TaskItem, error) { return nil, nil }
func (m *mockStorageForExec) CurrentStatus(_ string) (string, error)           { return "", nil }

func (m *mockStorageForExec) FindActiveTargetTask(_, _, _ string) (*tasks.TaskItem, string, error) {
	return nil, "", nil
}

func (m *mockStorageForExec) CleanupDuplicates() error            { return nil }
func (m *mockStorageForExec) DeleteTask(_ string) (string, error) { return "", nil }
func (m *mockStorageForExec) GetQueueDir() string                 { return "" }

// mockAssemblerForExec provides minimal AssemblerAPI for execution tests.
type mockAssemblerForExec struct{}

func (m *mockAssemblerForExec) AssemblePromptForTask(_ tasks.TaskItem) (prompts.PromptAssembly, error) {
	return prompts.PromptAssembly{Prompt: ""}, nil
}

func (m *mockAssemblerForExec) GetPromptsDir() string { return "" }
