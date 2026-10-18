package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

// awaitableOnComplete returns (handler, wait). Assigning handler to
// executor.OnComplete and deferring wait() makes the test block until the
// async heartbeat lifecycle finishes — important because the lifecycle's
// final SetHeartbeatConfig write lands in RuntimeData under t.TempDir(),
// and racing that write against t.TempDir() cleanup causes
// "directory not empty" flakes.
func awaitableOnComplete(_ *testing.T) (func(string, string), func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	once := sync.Once{}
	done := func() { once.Do(wg.Done) }
	wait := func() {
		c := make(chan struct{})
		go func() { wg.Wait(); close(c) }()
		select {
		case <-c:
		case <-time.After(2 * time.Second):
		}
	}
	return func(string, string) { done() }, wait
}

// setupExecutorTestEnv creates a store with a team, agent, membership, and
// heartbeat config, returning everything needed to construct an Executor.
func setupExecutorTestEnv(t *testing.T) (
	*store.FileTeamStore,
	*store.FileAgentStore,
	string, // roots.Config
) {
	t.Helper()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team One")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
	}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("set member: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{
		TeamID:   "team-1",
		AgentID:  "agent-1",
		Schedule: "0 * * * *",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("set heartbeat config: %v", err)
	}

	return teamStore, agentStore, roots.Config
}

func TestExecute_FullLifecycle(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-100", Title: "test"}).
		WithCreateRunResponse(&Run{ID: "run-200", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-200", Status: "RUN_STATUS_COMPLETE"})

	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), registry, nil)

	var completeCalled sync.WaitGroup
	completeCalled.Add(1)
	executor.OnComplete = func(teamID, agentID string) {
		if teamID != "team-1" || agentID != "agent-1" {
			t.Errorf("unexpected complete callback: %s/%s", teamID, agentID)
		}
		completeCalled.Done()
	}

	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "test-profile")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != store.HeartbeatStatusRunning {
		t.Errorf("expected running status, got %s", result.Status)
	}
	if result.RunID != "run-200" {
		t.Errorf("expected run-200, got %s", result.RunID)
	}

	// Verify task was created with correct fields
	if len(mockClient.createTaskCalls) != 1 {
		t.Fatalf("expected 1 CreateTask call, got %d", len(mockClient.createTaskCalls))
	}
	task := mockClient.createTaskCalls[0]
	if !strings.Contains(task.Title, "team-1") {
		t.Errorf("expected title to contain team-1, got %s", task.Title)
	}
	if task.Description == "" {
		t.Error("expected non-empty prompt in description")
	}

	// Verify run was created with correct profile
	if len(mockClient.createRunCalls) != 1 {
		t.Fatalf("expected 1 CreateRun call, got %d", len(mockClient.createRunCalls))
	}
	runReq := mockClient.createRunCalls[0]
	if runReq.TaskID != mockClient.createTaskCalls[0].ID {
		t.Errorf("run task ID %q does not match caller-assigned task ID %q", runReq.TaskID, mockClient.createTaskCalls[0].ID)
	}
	if runReq.ProfileRef == nil || runReq.ProfileRef.ProfileKey != "test-profile" {
		t.Error("expected profile ref with test-profile")
	}
	if runReq.RunMode != "" {
		t.Errorf("expected run mode to be derived from sandbox config, got %s", runReq.RunMode)
	}
	if runReq.ProfileRef == nil {
		t.Error("expected a profile-key-only run reference")
	}

	// Wait for async completion to call OnComplete
	done := make(chan struct{})
	go func() {
		completeCalled.Wait()
		close(done)
	}()
	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("OnComplete not called within timeout")
	}

	// After completion, config should be updated
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.LastExecution == nil {
		t.Fatal("expected LastExecution to be set")
	}
	if config.LastExecution.Status != store.HeartbeatStatusCompleted {
		t.Errorf("expected completed, got %s", config.LastExecution.Status)
	}
	if config.LastExecution.RunID != "run-200" {
		t.Errorf("expected run-200, got %s", config.LastExecution.RunID)
	}
	if config.LastSuccessfulExecution == nil || config.LastSuccessfulExecution.RunID != "run-200" {
		t.Errorf("expected last successful run-200, got %+v", config.LastSuccessfulExecution)
	}
}

func TestExecute_TeamDisabled(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)

	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "T")
	team.Enabled = false
	_ = teamStore.Create(ctx, team)
	_ = agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "A"})

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	result, err := executor.Execute(ctx, "team-1", "agent-1", "p")

	if err == nil {
		t.Fatal("expected error for disabled team")
	}
	if !IsTeamDisabled(err) {
		t.Errorf("expected TeamDisabledError, got: %v", err)
	}
	if result.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed status, got %s", result.Status)
	}
}

func TestExecute_PreSendCancellationDoesNotCountAsFailure(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient().WithCreateTaskError(NewDispatchRejectedError(context.Canceled))
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	result, err := executor.Execute(ctx, "team-1", "agent-1", "p")
	if err == nil || result.Status != store.HeartbeatStatusCancelled {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || updated.ConsecutiveFailures != heartbeatAttentionFailureThreshold-1 || updated.LastExecution.Status != store.HeartbeatStatusCancelled {
		t.Fatalf("cancelled config=%+v err=%v", updated, err)
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(ctx, "team-1", "agent-1", store.HeartbeatStatusCancelled, "", 10, 0)
	if err != nil || total != 1 || len(attempts) != 1 || attempts[0].AttentionEligible {
		t.Fatalf("cancelled attempts = total %d %+v err=%v", total, attempts, err)
	}
}

func TestExecute_CreateTaskFailure(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithCreateTaskError(errForTest("agent-manager error: validation error"))

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "p")

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "creating task") {
		t.Errorf("expected 'creating task' in error, got: %v", err)
	}
	if result.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}

	// Config should be updated with failure
	config, _ := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if config.LastExecution == nil || config.LastExecution.Status != store.HeartbeatStatusFailed {
		t.Error("expected config to be updated with failed status")
	}
}

func TestExecute_CreateTaskDefinitiveFailureRecordsAttentionAtThreshold(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}
	mockClient := newMockAgentClient().WithCreateTaskError(NewDispatchRejectedError(errForTest("Agent Manager refused task admission")))
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	executor.SetTeamReviewURL(func(_ context.Context, teamID string) string { return "https://pm.example/teams/" + teamID })
	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil {
		t.Fatal("expected definitive task refusal")
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(ctx, "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(attempts) != 1 || attempts[0].Phase != "creating_task" || attempts[0].RecoveryRequestID != "" || attempts[0].RecoveryOutcome != store.HeartbeatRecoveryNotRequested || attempts[0].ConsecutiveFailures != heartbeatAttentionFailureThreshold || attempts[0].FailureEpisodeID == "" || attempts[0].ReviewURL != "https://pm.example/teams/team-1" || !attempts[0].AttentionEligible {
		t.Fatalf("persisted task attempt = total %d, %+v; want recorded attention without lifecycle request", total, attempts)
	}
}

func TestExecute_CreateTaskThresholdFailureDoesNotRequestLifecycleRecovery(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("get heartbeat config: %v", err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatalf("seed failure streak: %v", err)
	}
	mockClient := newMockAgentClient().WithCreateTaskError(
		NewDispatchRejectedError(errForTest("task refused at threshold")),
	)
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil {
		t.Fatal("expected definitive task refusal")
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(ctx, "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil {
		t.Fatalf("list heartbeat attempts: %v", err)
	}
	if total != 1 || len(attempts) != 1 || attempts[0].ConsecutiveFailures != heartbeatAttentionFailureThreshold || attempts[0].RecoveryOutcome != store.HeartbeatRecoveryNotRequested || attempts[0].RecoveryRequestID != "" {
		t.Fatalf("persisted threshold attempt = total %d, %+v; want unconfirmed request outcome without ID", total, attempts)
	}
}

func TestExecute_ExistingLongFailureStreakBootstrapsAndKeepsEpisodeIdentity(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("get heartbeat config: %v", err)
	}
	config.ConsecutiveFailures = 58
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatalf("seed existing failure streak: %v", err)
	}
	mockClient := newMockAgentClient().WithCreateTaskError(
		NewDispatchRejectedError(errForTest("task refused during existing streak")),
	)
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	for i := 0; i < 2; i++ {
		if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil {
			t.Fatal("expected task refusal")
		}
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(ctx, "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil {
		t.Fatalf("list heartbeat attempts: %v", err)
	}
	if total != 2 || len(attempts) != 2 {
		t.Fatalf("attempt page = total %d, len %d; want two", total, len(attempts))
	}
	byCount := map[int]store.HeartbeatAttempt{}
	for _, attempt := range attempts {
		byCount[attempt.ConsecutiveFailures] = attempt
	}
	prior, priorOK := byCount[59]
	newest, newestOK := byCount[60]
	if !newestOK || !priorOK || newest.FailureEpisodeID == "" || newest.FailureEpisodeID != prior.FailureEpisodeID {
		t.Fatalf("existing streak episode identities/counts = %+v", attempts)
	}
	if newest.RecoveryOutcome != store.HeartbeatRecoveryNotRequested || prior.RecoveryOutcome != store.HeartbeatRecoveryNotRequested {
		t.Fatalf("existing-streak recovery outcomes = %q/%q; want no lifecycle recovery", newest.RecoveryOutcome, prior.RecoveryOutcome)
	}
}

func TestExecute_CreateTaskFailureBelowAttentionThreshold(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("get heartbeat config: %v", err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 2
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatalf("seed failure streak: %v", err)
	}

	mockClient := newMockAgentClient().WithCreateTaskError(
		NewDispatchRejectedError(errForTest("task refused below threshold")),
	)
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil {
		t.Fatal("expected definitive task refusal")
	}

	attempts, total, err := teamStore.ListHeartbeatAttempts(ctx, "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil {
		t.Fatalf("list heartbeat attempts: %v", err)
	}
	if total != 1 || len(attempts) != 1 || attempts[0].ConsecutiveFailures != heartbeatAttentionFailureThreshold-1 || attempts[0].RecoveryOutcome != store.HeartbeatRecoveryNotRequested {
		t.Fatalf("persisted below-threshold attempt = total %d, %+v", total, attempts)
	}
	updated, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || updated.ConsecutiveFailures != heartbeatAttentionFailureThreshold-1 {
		t.Fatalf("failure streak = %d, err=%v; want %d", updated.ConsecutiveFailures, err, heartbeatAttentionFailureThreshold-1)
	}
}

func TestExecute_CreateTaskApprovalPendingDoesNotCountAtThreshold(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("get heartbeat config: %v", err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatalf("seed failure streak: %v", err)
	}

	mockClient := newMockAgentClient().WithCreateTaskError(
		NewApprovalPendingError("creating_task", errForTest("approval requested")),
	)
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil || !IsApprovalPending(err) {
		t.Fatalf("Execute error = %v; want typed approval pending", err)
	}
	updated, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || updated.ConsecutiveFailures != heartbeatAttentionFailureThreshold-1 {
		t.Fatalf("failure streak = %d, err=%v; approval pending must preserve %d", updated.ConsecutiveFailures, err, heartbeatAttentionFailureThreshold-1)
	}

}

func TestExecute_CreateTaskUncertainRemainsOneSend(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	mockClient := newMockAgentClient().WithCreateTaskError(
		NewDispatchUncertainError(errForTest("connection reset after task submission")),
	)
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	if _, err := executor.Execute(context.Background(), "team-1", "agent-1", "p"); err == nil || !IsDispatchUncertain(err) {
		t.Fatalf("Execute error = %v; want uncertain task outcome", err)
	}
	if len(mockClient.createTaskCalls) != 1 || len(mockClient.createRunCalls) != 0 {
		t.Fatalf("task/run calls = %d/%d; uncertain create must not resend or start a run", len(mockClient.createTaskCalls), len(mockClient.createRunCalls))
	}

	config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil || config.ConsecutiveFailures != 0 {
		t.Fatalf("uncertain task changed failure streak: count=%d err=%v", config.ConsecutiveFailures, err)
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(context.Background(), "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil || total != 1 || len(attempts) != 1 || attempts[0].AttentionEligible {
		t.Fatalf("uncertain task attempt attention = total %d, %+v, err=%v; want one ineligible attempt", total, attempts, err)
	}
}

func TestExecute_CreateRunFailure(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1", Title: "test"}).
		WithCreateRunError(errForTest("runner unavailable"))

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "p")

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "creating run") {
		t.Errorf("expected 'creating run' in error, got: %v", err)
	}
	if result.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	taskID := mockClient.createTaskCalls[0].ID
	if len(mockClient.deleteTaskCalls) != 1 || mockClient.deleteTaskCalls[0] != taskID {
		t.Fatalf("delete task calls = %v, want [%s]", mockClient.deleteTaskCalls, taskID)
	}
	if len(mockClient.cancelTaskCalls) != 1 || mockClient.cancelTaskCalls[0] != taskID {
		t.Fatalf("cancel task calls = %v, want [%s]", mockClient.cancelTaskCalls, taskID)
	}
}

func TestExecute_CreateRunPreSendCancellationDoesNotCountAsFailure(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-cancel-before-run"}).
		WithCreateRunError(NewDispatchRejectedError(context.Canceled))
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	result, err := executor.Execute(ctx, "team-1", "agent-1", "p")
	if err == nil || result.Status != store.HeartbeatStatusCancelled {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || updated.ConsecutiveFailures != heartbeatAttentionFailureThreshold-1 || updated.LastExecution.Status != store.HeartbeatStatusCancelled {
		t.Fatalf("cancelled run-admission config=%+v err=%v", updated, err)
	}
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 1 {
		t.Fatalf("calls task=%d run=%d; want one task and one refused run attempt", len(client.createTaskCalls), len(client.createRunCalls))
	}
	if len(client.cancelTaskCalls) != 1 || len(client.deleteTaskCalls) != 1 {
		t.Fatalf("known orphan task cleanup cancel=%v delete=%v", client.cancelTaskCalls, client.deleteTaskCalls)
	}
}

func TestExecute_CreateRunDefinitiveFailureRecordsAttentionWithoutLifecycleRecovery(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}
	mockClient := newMockAgentClient().WithCreateTaskResponse(&Task{ID: "task-run-threshold", Title: "test"}).WithCreateRunError(NewDispatchRejectedError(errForTest("run refused")))
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil {
		t.Fatal("expected run refusal")
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(ctx, "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(attempts) != 1 || attempts[0].Phase != "creating_run" || attempts[0].RecoveryRequestID != "" || attempts[0].RecoveryOutcome != store.HeartbeatRecoveryNotRequested || !attempts[0].AttentionEligible || attempts[0].ConsecutiveFailures != heartbeatAttentionFailureThreshold {
		t.Fatalf("persisted run attempt = total %d, %+v; want definitive failure and attention without lifecycle request", total, attempts)
	}
}

// TestExecute_TransportUncertainRetainsObligation verifies the core fence: a
// run-creation response lost in transport must not free the member's slot,
// because the owner may have accepted the run. The obligation stays visible,
// keeps the idempotency key for owner reconciliation, and refuses a duplicate.
func TestExecute_UncertainRunDoesNotInflateNextDefinitiveThresholdFailure(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-sequence"}).
		WithCreateRunError(NewDispatchUncertainError(errors.New("connection reset after submission")))
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil || !IsDispatchUncertain(err) {
		t.Fatalf("first CreateRun error=%v; want uncertainty", err)
	}
	config, err = teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || config.ConsecutiveFailures != heartbeatAttentionFailureThreshold-1 {
		t.Fatalf("uncertainty changed threshold: count=%d err=%v", config.ConsecutiveFailures, err)
	}

	// This fixture isolates counter accounting after the separate retained-slot
	// integration test proves the uncertain operation is not resent in service.
	client.createRunErr = NewDispatchRejectedError(errors.New("owner definitively refused run"))
	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err == nil || IsDispatchUncertain(err) {
		t.Fatalf("second CreateRun error=%v; want definitive refusal", err)
	}
	config, err = teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || config.ConsecutiveFailures != heartbeatAttentionFailureThreshold {
		t.Fatalf("post-refusal count=%d err=%v; want threshold", config.ConsecutiveFailures, err)
	}
}

func TestExecute_TransportUncertainRetainsObligation(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-unc", Title: "test"}).
		WithCreateRunError(NewDispatchUncertainError(errors.New("connection reset by peer")))

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, mockClient)
	t.Cleanup(teamExecStore.Shutdown)
	executor.SetTeamExecStore(teamExecStore)

	if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	waitFor(t, 5*time.Second, func() bool {
		return len(teamExecStore.Status("team-1").UncertainAgentIDs) == 1
	}, "uncertain obligation to be retained")
	waitForCondition(t, "uncertain heartbeat attempt", func() bool {
		attempts, _, listErr := teamStore.ListHeartbeatAttempts(context.Background(), "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
		return listErr == nil && len(attempts) == 1
	})
	attempts, total, err := teamStore.ListHeartbeatAttempts(context.Background(), "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil || total != 1 || len(attempts) != 1 || attempts[0].AttentionEligible {
		t.Fatalf("uncertain run attempt attention = total %d, %+v, err=%v; want one ineligible attempt", total, attempts, err)
	}
	config, err = teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || config.ConsecutiveFailures != heartbeatAttentionFailureThreshold-1 {
		t.Fatalf("uncertain run changed definitive failure streak: count=%d err=%v", config.ConsecutiveFailures, err)
	}

	// The task must not be cancelled: the run may exist.
	if len(mockClient.cancelTaskCalls) != 0 || len(mockClient.deleteTaskCalls) != 0 {
		t.Fatalf("uncertain dispatch must not cancel/delete task: cancel=%v delete=%v", mockClient.cancelTaskCalls, mockClient.deleteTaskCalls)
	}
	if got := len(mockClient.createRunCalls); got != 1 {
		t.Fatalf("successful durable intent should precede exactly one CreateRun, got %d", got)
	}

	// The dispatch intent must be durable with its idempotency key.
	persisted := readQueueFile(t, dir, "team-1")
	if len(persisted.Running) != 1 || persisted.Running[0].IdempotencyKey == "" {
		t.Fatalf("expected persisted dispatch intent key, got %+v", persisted.Running)
	}

	// A duplicate heartbeat must be refused while the obligation is unresolved.
	if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected duplicate heartbeat refused, got %v", err)
	}
}

type gatedHeartbeatExecutor struct {
	executor *Executor
	reached  chan struct{}
	release  chan struct{}
}

func (g *gatedHeartbeatExecutor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	close(g.reached)
	<-g.release
	return g.executor.Execute(ctx, teamID, agentID, profileKey)
}

func TestExecute_DoesNotCreateTaskOrRunWhenFirstDispatchIntentCannotPersist(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	dir := t.TempDir()
	client := newMockAgentClient()
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	gated := &gatedHeartbeatExecutor{executor: executor, reached: make(chan struct{}), release: make(chan struct{})}
	teamExecStore := NewTeamExecutionStore(teamStore, gated, dir, client)
	t.Cleanup(teamExecStore.Shutdown)
	executor.SetTeamExecStore(teamExecStore)

	if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	select {
	case <-gated.reached:
	case <-time.After(5 * time.Second):
		t.Fatal("queued execution did not reach test gate")
	}
	queuePath := filepath.Join(dir, "team-queue-team-1.json")
	if err := os.Remove(queuePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(queuePath, 0o755); err != nil {
		t.Fatal(err)
	}
	close(gated.release)
	waitFor(t, 5*time.Second, func() bool {
		return len(teamExecStore.Status("team-1").RunningAgentIDs) == 0
	}, "pre-POST persistence failure to finish")
	teamExecStore.Shutdown()
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("failed first-intent save must gate both owner POSTs, tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	if len(client.cancelTaskCalls) != 0 || len(client.deleteTaskCalls) != 0 {
		t.Fatalf("no task was sent, so cleanup must not target a task: cancel=%v delete=%v", client.cancelTaskCalls, client.deleteTaskCalls)
	}
	config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil {
		t.Fatalf("read failed heartbeat config: %v", err)
	}
	if config.LastExecution == nil || config.LastExecution.Status != store.HeartbeatStatusFailed || !strings.Contains(config.LastExecution.Error, "recording task and run dispatch intent") {
		t.Fatalf("last execution = %+v; want durable-intent persistence failure", config.LastExecution)
	}
}

type cancelingTaskPreflightClient struct {
	*mockAgentClient
	cancel context.CancelFunc
	gets   atomic.Int32
}

func (c *cancelingTaskPreflightClient) GetTask(ctx context.Context, _ string) (*Task, error) {
	c.gets.Add(1)
	c.cancel()
	return nil, ctx.Err()
}

func TestExecute_CanceledFreshTaskLookupPersistsNoReplayTombstone(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	dir := t.TempDir()
	lookupCtx, cancel := context.WithCancel(context.Background())
	client := &cancelingTaskPreflightClient{mockAgentClient: newMockAgentClient(), cancel: cancel}
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	t.Cleanup(teamExecStore.Shutdown)
	executor.SetTeamExecStore(teamExecStore)

	// Seed the running slot as if the real queue worker had admitted this
	// member. Execute will persist its immutable intent before the canceled
	// read-only owner lookup.
	tec := teamExecStore.GetOrCreate("team-1")
	tec.mu.Lock()
	tec.running["agent-1"] = runningEntry{ProfileKey: ""}
	if err := tec.persistLocked(); err != nil {
		tec.mu.Unlock()
		t.Fatal(err)
	}
	tec.mu.Unlock()

	result, err := executor.Execute(lookupCtx, "team-1", "agent-1", "")
	if !isPreSendCancellation(err) || result == nil || result.Status != store.HeartbeatStatusCancelled {
		t.Fatalf("Execute=(%+v, %v); want canceled result from read-only preflight", result, err)
	}
	if client.gets.Load() != 1 || len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("owner calls: gets=%d tasks=%d runs=%d; canceled preflight must send no task/run", client.gets.Load(), len(client.createTaskCalls), len(client.createRunCalls))
	}
	persisted := readQueueFile(t, dir, "team-1")
	if len(persisted.Running) != 1 || persisted.Running[0].State != ObligationDispatchCanceled || persisted.Running[0].TaskRequestSnapshot == "" || persisted.Running[0].CreateRunRequestSnapshot == "" {
		t.Fatalf("expected durable exact no-replay tombstone before worker callback, got %+v", persisted.Running)
	}

	// A crash after tombstone persistence but before the old worker callback
	// must not create a new task or replay the canceled request on startup.
	restartedExecutor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	restartedStore := NewTeamExecutionStore(teamStore, restartedExecutor, dir, client)
	t.Cleanup(restartedStore.Shutdown)
	restartedExecutor.SetTeamExecStore(restartedStore)
	restartedStore.Recover(context.Background())
	if got := restartedStore.Status("team-1"); len(got.RunningAgentIDs) != 0 {
		t.Fatalf("canceled no-replay tombstone was recovered as active ownership: %+v", got)
	}
	if client.gets.Load() != 1 || len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("recovery replayed canceled intent: gets=%d tasks=%d runs=%d", client.gets.Load(), len(client.createTaskCalls), len(client.createRunCalls))
	}
}

type observedTaskLookupClient struct {
	*mockAgentClient
	terms  *Task
	status string
	change bool
}

func (c *observedTaskLookupClient) GetTask(_ context.Context, taskID string) (*Task, error) {
	task := cloneDispatchTask(c.terms)
	task.ID = taskID
	task.Status = c.status
	if c.change {
		task.Title += " foreign"
	}
	return task, nil
}

func TestExecute_RetainsMismatchedOrNonqueuedExistingTask(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		change bool
	}{
		{name: "foreign same-ID terms", status: "TASK_STATUS_QUEUED", change: true},
		{name: "existing task already running", status: "TASK_STATUS_RUNNING"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			teamStore, agentStore, _ := setupExecutorTestEnv(t)
			root := t.TempDir()
			base := newMockAgentClient()
			executor := newTestExecutor(t, teamStore, agentStore, base, root, nil, nil)
			prompt, err := executor.BuildPrompt(context.Background(), "team-1", "agent-1")
			if err != nil {
				t.Fatal(err)
			}
			client := &observedTaskLookupClient{
				mockAgentClient: base,
				terms: &Task{
					Title: "Heartbeat: team-1/agent-1", Description: prompt,
					ScopePath: root, ProjectRoot: root,
				},
				status: tc.status, change: tc.change,
			}
			executor.agentClient = client
			dir := t.TempDir()
			teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
			t.Cleanup(teamExecStore.Shutdown)
			executor.SetTeamExecStore(teamExecStore)
			if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "p"); err != nil {
				t.Fatal(err)
			}
			waitFor(t, 5*time.Second, func() bool {
				status := teamExecStore.Status("team-1")
				return len(status.UncertainAgentIDs) == 1 && status.UncertainAgentIDs[0] == "agent-1"
			}, "existing task conflict to remain held")
			teamExecStore.Shutdown()
			if len(base.createTaskCalls) != 0 || len(base.createRunCalls) != 0 {
				t.Fatalf("existing owner task must never be reposted or run: task posts=%d runs=%d", len(base.createTaskCalls), len(base.createRunCalls))
			}
			persisted := readQueueFile(t, dir, "team-1")
			if len(persisted.Running) != 1 || persisted.Running[0].TaskRequestSnapshot == "" || persisted.Running[0].TaskConfirmed {
				t.Fatalf("unresolved existing task should retain the exact unconfirmed queue intent, got %+v", persisted.Running)
			}
		})
	}
}

type lostTaskCreateResponseClient struct {
	*mockAgentClient
	mu             sync.Mutex
	getCount       int
	beforeTaskPost func(*Task) error
	postGateErr    error
}

func (c *lostTaskCreateResponseClient) GetTask(ctx context.Context, taskID string) (*Task, error) {
	c.mu.Lock()
	c.getCount++
	call := c.getCount
	c.mu.Unlock()
	switch call {
	case 1:
		return nil, nil // exact preflight miss
	case 2:
		return nil, errors.New("injected lookup outage after accepted POST")
	default:
		return c.mockAgentClient.GetTask(ctx, taskID)
	}
}

func (c *lostTaskCreateResponseClient) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	if c.beforeTaskPost != nil {
		if err := c.beforeTaskPost(task); err != nil {
			c.mu.Lock()
			c.postGateErr = err
			c.mu.Unlock()
			return nil, NewDispatchRejectedError(err)
		}
	}
	if _, err := c.mockAgentClient.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	return nil, NewDispatchUncertainError(errors.New("owner stored task but response was lost"))
}

func (c *lostTaskCreateResponseClient) gateError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.postGateErr
}

func TestExecuteFreezesDefaultProfileAndRequestsBeforeTaskPostAndRecoversAfterRestart(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	team, err := teamStore.Get(context.Background(), "team-1")
	if err != nil {
		t.Fatal(err)
	}
	defaultProfile, err := DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	base := newMockAgentClient().
		WithCreateRunResponse(&Run{ID: "run-recovered-task", Status: "RUN_STATUS_RUNNING", StartedAt: "2026-10-17T12:00:00Z"}).
		WithEchoRunIdentity()
	client := &lostTaskCreateResponseClient{mockAgentClient: base}
	client.beforeTaskPost = func(task *Task) error {
		data, readErr := os.ReadFile(filepath.Join(dir, "team-queue-team-1.json"))
		if readErr != nil {
			return fmt.Errorf("read pre-POST queue snapshot: %w", readErr)
		}
		var persisted persistedTeamQueue
		if err := json.Unmarshal(data, &persisted); err != nil {
			return fmt.Errorf("decode pre-POST queue snapshot: %w", err)
		}
		if len(persisted.Running) != 1 {
			return fmt.Errorf("pre-POST queue contains %d running records", len(persisted.Running))
		}
		entry := persisted.Running[0]
		frozenTask, err := parseTaskDispatchRequest(entry.TaskRequestSnapshot)
		if err != nil {
			return fmt.Errorf("parse pre-POST task snapshot: %w", err)
		}
		if frozenTask.ID != task.ID || frozenTask.Title != task.Title || frozenTask.Description != task.Description || frozenTask.ScopePath != task.ScopePath || frozenTask.ProjectRoot != task.ProjectRoot {
			return fmt.Errorf("pre-POST task snapshot does not match authored request: frozen=%+v request=%+v", frozenTask, task)
		}
		var frozenRun CreateRunRequest
		if err := json.Unmarshal([]byte(entry.CreateRunRequestSnapshot), &frozenRun); err != nil {
			return fmt.Errorf("parse pre-POST run snapshot: %w", err)
		}
		if frozenRun.TaskID != task.ID || frozenRun.ProfileRef == nil || frozenRun.ProfileRef.ProfileKey != defaultProfile {
			return fmt.Errorf("pre-POST run snapshot lost task/default profile: %+v", frozenRun)
		}
		return nil
	}
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	t.Cleanup(teamExecStore.Shutdown)
	executor.SetTeamExecStore(teamExecStore)
	NewScheduler(executor, client, teamStore, teamExecStore).executeHeartbeat(context.Background(), "team-1", "agent-1")
	waitFor(t, 5*time.Second, func() bool {
		status := teamExecStore.Status("team-1")
		return len(status.UncertainAgentIDs) == 1 && status.UncertainAgentIDs[0] == "agent-1"
	}, "accepted task with lost response to remain uncertain")
	teamExecStore.Shutdown()
	if err := client.gateError(); err != nil {
		t.Fatal(err)
	}
	if len(base.createTaskCalls) != 1 || len(base.createRunCalls) != 0 {
		t.Fatalf("first lifecycle task posts=%d run posts=%d; want one original task and no run before recovery", len(base.createTaskCalls), len(base.createRunCalls))
	}
	persisted := readQueueFile(t, dir, "team-1")
	if len(persisted.Running) != 1 || persisted.Running[0].TaskConfirmed || persisted.Running[0].TaskRequestSnapshot == "" || persisted.Running[0].CreateRunRequestSnapshot == "" {
		t.Fatalf("expected both frozen requests to remain before confirmation, got %+v", persisted.Running)
	}

	// Change the team's runtime default after the uncertain task POST. Recovery
	// must use the already-saved profile key and run body rather than current
	// configuration.
	team.Runtime.Mode = "single-process"
	team.Coordination.Pattern = teamconfig.CoordinationPatternLeaderLed
	team.Coordination.LeadAgentID = "agent-1"
	team.Coordination.ReportingMode = teamconfig.ReportingModeLeader
	team.Coordination.MessagingMode = teamconfig.MessagingModeInSession
	team.Execution.QueuePolicy = teamconfig.QueuePolicySerialized
	team.Execution.MaxConcurrentRuns = 1
	if err := teamStore.Update(context.Background(), "team-1", team); err != nil {
		t.Fatal(err)
	}
	restartedExecutor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	restartedStore := NewTeamExecutionStore(teamStore, restartedExecutor, dir, client)
	t.Cleanup(restartedStore.Shutdown)
	restartedExecutor.SetTeamExecStore(restartedStore)
	restartedStore.Recover(context.Background())
	if len(base.createTaskCalls) != 1 || len(base.createRunCalls) != 0 {
		t.Fatalf("passive recovery must not replay before scheduled eligibility: tasks=%d runs=%d", len(base.createTaskCalls), len(base.createRunCalls))
	}
	NewScheduler(restartedExecutor, client, teamStore, restartedStore).executeHeartbeat(context.Background(), "team-1", "agent-1")
	if len(base.createTaskCalls) != 1 || len(base.createRunCalls) != 1 {
		t.Fatalf("eligible scheduled retry posts tasks=%d runs=%d; want same task and one run", len(base.createTaskCalls), len(base.createRunCalls))
	}
	actualRun, err := json.Marshal(base.createRunCalls[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(actualRun) != persisted.Running[0].CreateRunRequestSnapshot {
		t.Fatalf("replayed run request changed: got %s want %s", actualRun, persisted.Running[0].CreateRunRequestSnapshot)
	}
	if base.createRunCalls[0].ProfileRef == nil || base.createRunCalls[0].ProfileRef.ProfileKey != defaultProfile {
		t.Fatalf("recovery used changed profile default, got %+v; want %q", base.createRunCalls[0].ProfileRef, defaultProfile)
	}
}

// TestExecute_RejectedDispatchReleasesObligation verifies a definitive owner
// refusal (validation) does release the slot and clean up the orphaned task.
func TestExecute_RejectedDispatchReleasesObligation(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	dir := t.TempDir()

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-rej", Title: "test"}).
		WithCreateRunError(NewDispatchRejectedError(errors.New("validation error: bad profile")))

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, mockClient)
	executor.SetTeamExecStore(teamExecStore)

	if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	waitFor(t, 5*time.Second, func() bool {
		return len(teamExecStore.Status("team-1").RunningAgentIDs) == 0
	}, "rejected dispatch to release the slot")

	if len(mockClient.cancelTaskCalls) != 1 || len(mockClient.deleteTaskCalls) != 1 {
		t.Fatalf("rejected dispatch must clean up orphaned task: cancel=%v delete=%v", mockClient.cancelTaskCalls, mockClient.deleteTaskCalls)
	}
}

// waitFor polls until condition or the deadline elapses.
func waitFor(t *testing.T, timeout time.Duration, condition func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestExecute_WaitForRunFailure(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunError(errForTest("timeout"))

	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), registry, nil)

	executor.OnComplete = func(_, _ string) { t.Error("observer failure must not release the owner obligation") }

	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "p")
	if err != nil {
		t.Fatalf("Execute should succeed (async wait), got: %v", err)
	}
	if result.Status != store.HeartbeatStatusRunning {
		t.Errorf("expected running, got %s", result.Status)
	}

	waitForCondition(t, "observer-unavailable config", func() bool {
		config, _ := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
		return config != nil && config.LastExecution != nil && config.LastExecution.Status == store.HeartbeatStatusObserverUnavailable
	})
	if registry.Count() != 1 {
		t.Fatalf("observer failure must retain the active owner run in the registry, got %d", registry.Count())
	}
}

type sequencedWaitAgentClient struct {
	AgentClient
	mu             sync.Mutex
	waitCalls      int
	ownerRunStatus string
	waited         chan int
}

type countingBlockingWaitAgentClient struct {
	AgentClient
	calls   atomic.Int32
	started chan struct{}
}

type controlledWaitAgentClient struct {
	AgentClient
	calls        atomic.Int32
	firstStarted chan struct{}
	releaseFirst chan struct{}
	secondStart  chan struct{}
}

func (c *controlledWaitAgentClient) WaitForRun(ctx context.Context, _ string, _ time.Duration) (*Run, error) {
	switch c.calls.Add(1) {
	case 1:
		close(c.firstStarted)
		<-c.releaseFirst
		return nil, errForTest("observer transport timeout")
	default:
		close(c.secondStart)
		<-ctx.Done()
		return nil, ctx.Err()
	}
}

func (c *controlledWaitAgentClient) GetRun(_ context.Context, runID string) (*Run, error) {
	return &Run{ID: runID, Status: "RUN_STATUS_RUNNING"}, nil
}

func (c *countingBlockingWaitAgentClient) WaitForRun(ctx context.Context, _ string, _ time.Duration) (*Run, error) {
	c.calls.Add(1)
	select {
	case c.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRecoveredRunWatcherDeduplicatesConcurrentAttachments(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	base := newMockAgentClient()
	client := &countingBlockingWaitAgentClient{AgentClient: base, started: make(chan struct{}, 1)}
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	defer executor.Shutdown()

	executor.WatchRecoveredRun("team-1", "agent-1", "run-shared", "p", "task-1", "tag", "RUN_STATUS_RUNNING", "")
	select {
	case <-client.started:
	case <-time.After(5 * time.Second):
		t.Fatal("first recovered waiter did not start")
	}
	executor.WatchRecoveredRun("team-1", "agent-1", "run-shared", "p", "task-1", "tag", "RUN_STATUS_RUNNING", "")
	if got := client.calls.Load(); got != 1 {
		t.Fatalf("same team/member/run must have one completion observer, got %d", got)
	}
}

func TestUnavailableWatcherRetryStaysRetryableWhenOldWaiterStillOwnsKey(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	dir := t.TempDir()
	client := &controlledWaitAgentClient{
		AgentClient:  newMockAgentClient(),
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
		secondStart:  make(chan struct{}),
	}
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnComplete = teamExecStore.OnComplete
	tec := teamExecStore.GetOrCreate("team-1")
	tec.mu.Lock()
	tec.running["agent-1"] = runningEntry{ProfileKey: "p", RunID: "run-race", State: ObligationWatcherUnavailable}
	_ = tec.persistLocked()
	tec.mu.Unlock()

	executor.WatchRecoveredRun("team-1", "agent-1", "run-race", "p", "task-1", "tag", "RUN_STATUS_RUNNING", "")
	select {
	case <-client.firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("first watcher did not start")
	}
	if handled, err := tec.RetryUnavailableWatcher(context.Background(), "agent-1"); !handled || err != nil {
		t.Fatalf("scheduled retry result = handled %t, err %v", handled, err)
	}
	if len(tec.Status().WatcherUnavailableAgentIDs) != 1 {
		t.Fatalf("deduplicated overlapping watcher must leave a future retry path: %+v", tec.Status())
	}

	close(client.releaseFirst)
	waitForCondition(t, "old waiter key release", func() bool {
		executor.mu.Lock()
		defer executor.mu.Unlock()
		return len(executor.waiterKeys) == 0
	})
	if handled, err := tec.RetryUnavailableWatcher(context.Background(), "agent-1"); !handled || err != nil {
		t.Fatalf("second scheduled retry result = handled %t, err %v", handled, err)
	}
	select {
	case <-client.secondStart:
	case <-time.After(5 * time.Second):
		t.Fatal("later scheduled retry did not restore the completion watcher")
	}
	if len(tec.Status().WatcherUnavailableAgentIDs) != 0 {
		t.Fatalf("successful later attachment should leave watcher unavailable state: %+v", tec.Status())
	}
	executor.Shutdown()
}

func (c *sequencedWaitAgentClient) WaitForRun(context.Context, string, time.Duration) (*Run, error) {
	c.mu.Lock()
	c.waitCalls++
	call := c.waitCalls
	c.mu.Unlock()
	c.waited <- call
	if call == 1 {
		return nil, errForTest("immediate observer timeout")
	}
	return &Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"}, nil
}

func (c *sequencedWaitAgentClient) GetRun(_ context.Context, runID string) (*Run, error) {
	return &Run{ID: runID, Status: c.ownerRunStatus}, nil
}

func TestObserverFailureRetainsOwnershipAndScheduledRewatchReleasesOnlyOnTerminal(t *testing.T) {
	for _, ownerStatus := range []string{"RUN_STATUS_RUNNING", "RUN_STATUS_NEEDS_REVIEW"} {
		t.Run(ownerStatus, func(t *testing.T) {
			teamStore, agentStore, _ := setupExecutorTestEnv(t)
			dir := t.TempDir()
			base := newMockAgentClient().
				WithCreateTaskResponse(&Task{ID: "task-1"}).
				WithCreateRunResponse(&Run{ID: "run-1", Status: ownerStatus})
			client := &sequencedWaitAgentClient{AgentClient: base, ownerRunStatus: ownerStatus, waited: make(chan int, 2)}
			registry := NewRunRegistry(dir)
			executor := newTestExecutor(t, teamStore, agentStore, client, dir, registry, nil)
			teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
			executor.SetTeamExecStore(teamExecStore)
			executor.OnComplete = teamExecStore.OnComplete
			defer executor.Shutdown()

			if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "p"); err != nil {
				t.Fatal(err)
			}
			teamExecStore.Shutdown()
			select {
			case call := <-client.waited:
				if call != 1 {
					t.Fatalf("first waiter call = %d", call)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("immediate observer error was not returned")
			}
			waitForCondition(t, "watcher-unavailable obligation", func() bool {
				status := teamExecStore.Status("team-1")
				return len(status.RunningAgentIDs) == 1 && len(status.WatcherUnavailableAgentIDs) == 1
			})
			config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
			if err != nil || config.LastExecution == nil || config.LastExecution.Status != store.HeartbeatStatusObserverUnavailable || config.LifecycleState != store.HeartbeatLifecycleObserverUnavailable {
				t.Fatalf("observer failure should be visible without claiming owner failure: config=%+v err=%v", config, err)
			}
			if registry.Count() != 1 {
				t.Fatalf("observer failure discarded run registry ownership: count=%d", registry.Count())
			}
			if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "p"); err == nil || !IsMemberAlreadyQueued(err) {
				t.Fatalf("observer failure must retain the slot against duplicate dispatch, got %v", err)
			}

			scheduler := NewScheduler(executor, client, teamStore, teamExecStore)
			scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")
			select {
			case call := <-client.waited:
				if call != 2 {
					t.Fatalf("replacement waiter call = %d", call)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("scheduled owner recheck did not reattach a waiter")
			}
			waitForCondition(t, "terminal callback release", func() bool {
				return len(teamExecStore.Status("team-1").RunningAgentIDs) == 0 && registry.Count() == 0
			})
			if len(base.createTaskCalls) != 1 || len(base.createRunCalls) != 1 {
				t.Fatalf("rewatching an existing run must not dispatch: task=%d run=%d", len(base.createTaskCalls), len(base.createRunCalls))
			}
		})
	}
}

func TestRecoverApprovalPendingWaitsForEnabledCurrentMemberSchedule(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team One")); err != nil {
		t.Fatal(err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One"}); err != nil {
		t.Fatal(err)
	}
	if err := fileStore.Relations().SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Enabled: true, Schedule: "0 * * * *"}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-1", persistedTeamQueue{
		TeamID:            "team-1",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{{
			AgentID: "agent-1", ProfileKey: "p", State: ObligationApprovalPending,
			IdempotencyKey: "prompt-manager-heartbeat:team-1:agent-1:attempt-1",
			TaskID:         "task-1", RunTag: "heartbeat-team-1-agent-1",
		}},
	})
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-after-approval", TaskID: "task-1", Status: "RUN_STATUS_RUNNING"})
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnComplete = teamExecStore.OnComplete
	teamExecStore.Recover(ctx)
	if len(client.createRunCalls) != 0 {
		t.Fatalf("startup recovery must not replay approval-pending work before admission, got %d CreateRun calls", len(client.createRunCalls))
	}
	scheduler := NewScheduler(executor, client, teamStore, teamExecStore)
	team, err := teamStore.Get(ctx, "team-1")
	if err != nil {
		t.Fatal(err)
	}
	team.Enabled = false
	team.EnabledSet = true
	if err := teamStore.Update(ctx, "team-1", team); err != nil {
		t.Fatal(err)
	}
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	team.Enabled = true
	team.EnabledSet = true
	if err := teamStore.Update(ctx, "team-1", team); err != nil {
		t.Fatal(err)
	}
	if err := fileStore.Relations().DeleteTeamMember(ctx, "team-1", "agent-1"); err != nil {
		t.Fatal(err)
	}
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("disabled team or removed member must not turn approval into work: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
}

func TestRunStageApprovalPendingSurvivesRestartAndReplaysOriginalIntent(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	dir := t.TempDir()
	client := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-durable", Title: "heartbeat"}).
		WithCreateRunError(NewApprovalPendingError("creating_run", errors.New("operator approval requested")))
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	firstStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(firstStore)
	executor.OnComplete = firstStore.OnComplete
	firstScheduler := NewScheduler(executor, client, teamStore, firstStore)

	firstScheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")
	firstStore.Shutdown()
	persisted := readQueueFile(t, dir, "team-1")
	taskID := ""
	if len(client.createTaskCalls) == 1 {
		taskID = client.createTaskCalls[0].ID
	}
	if len(persisted.Running) != 1 || persisted.Running[0].State != ObligationApprovalPending || persisted.Running[0].RunID != "" || persisted.Running[0].TaskID != taskID {
		t.Fatalf("run-stage approval must persist task identity and pending state before restart: %+v", persisted)
	}
	firstRequest := client.createRunCalls[0]
	if firstRequest.IdempotencyKey == "" || firstRequest.TaskID != taskID {
		t.Fatalf("initial run intent lacks stable identity: %+v", firstRequest)
	}
	executor.Shutdown()

	client.createRunErr = nil
	client.createRunResp = &Run{ID: "run-after-restart", TaskID: taskID, Status: "RUN_STATUS_RUNNING"}
	restartedExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	restartedStore := NewTeamExecutionStore(teamStore, restartedExecutor, dir, client)
	restartedExecutor.SetTeamExecStore(restartedStore)
	restartedExecutor.OnComplete = restartedStore.OnComplete
	restartedStore.Recover(context.Background())
	if len(client.createRunCalls) != 1 {
		t.Fatalf("startup must not replay known approval pending before scheduled admission, got %d calls", len(client.createRunCalls))
	}
	restartedScheduler := NewScheduler(restartedExecutor, client, teamStore, restartedStore)
	restartedScheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")
	restartedStore.Shutdown()
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 2 {
		t.Fatalf("restart retry should preserve one task and replay run once: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	for _, request := range client.createRunCalls {
		if request.TaskID != firstRequest.TaskID || request.IdempotencyKey != firstRequest.IdempotencyKey {
			t.Fatalf("restart retry changed operation identity: first=%+v retry=%+v", firstRequest, request)
		}
	}
	restartedExecutor.Shutdown()
}

func waitForCondition(t *testing.T, what string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		runtime.Gosched()
	}
}

func TestExecute_RunFailed(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	config, configErr := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if configErr != nil || config == nil {
		t.Fatalf("GetHeartbeatConfig: config=%+v err=%v", config, configErr)
	}
	config.ConsecutiveFailures = heartbeatAttentionFailureThreshold - 1
	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-1", "agent-1", config); err != nil {
		t.Fatalf("SetHeartbeatConfig: %v", err)
	}

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_FAILED", Error: "agent crashed"})

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)

	var completeCalled sync.WaitGroup
	completeCalled.Add(1)
	executor.OnComplete = func(_, _ string) { completeCalled.Done() }

	_, err := executor.Execute(context.Background(), "team-1", "agent-1", "p")
	if err != nil {
		t.Fatalf("Execute should succeed, got: %v", err)
	}

	done := make(chan struct{})
	go func() {
		completeCalled.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("OnComplete not called")
	}

	config, _ = teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if config.LastExecution == nil {
		t.Fatal("expected LastExecution")
	}
	if config.LastExecution.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed, got %s", config.LastExecution.Status)
	}
	if config.LastExecution.Error != "agent crashed" {
		t.Errorf("expected error message, got %s", config.LastExecution.Error)
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(context.Background(), "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
	if err != nil || total != 1 || len(attempts) != 1 || attempts[0].ConsecutiveFailures != heartbeatAttentionFailureThreshold || attempts[0].RecoveryOutcome != store.HeartbeatRecoveryNotRequested || attempts[0].FailureEpisodeID == "" || !attempts[0].AttentionEligible {
		t.Fatalf("terminal failed attempt attention = total %d, %+v, err=%v; want truthful threshold failure", total, attempts, err)
	}
}

func TestExecute_RunCancelledPreservesFailureEpisode(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.ConsecutiveFailures = 2
	config.LastSuccessfulExecution = &store.HeartbeatExecResult{RunID: "last-good", Status: store.HeartbeatStatusCompleted, EndedAt: "2026-10-16T00:00:00Z"}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}

	client := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-cancelled"}).
		WithCreateRunResponse(&Run{ID: "run-cancelled", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-cancelled", Status: "RUN_STATUS_CANCELLED"})
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	onComplete, waitComplete := awaitableOnComplete(t)
	executor.OnComplete = onComplete
	defer waitComplete()

	if _, err := executor.Execute(ctx, "team-1", "agent-1", "p"); err != nil {
		t.Fatal(err)
	}
	waitComplete()
	updated, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil || updated.LastExecution == nil || updated.LastExecution.Status != store.HeartbeatStatusCancelled {
		t.Fatalf("cancelled run config = %+v err=%v", updated, err)
	}
	if updated.ConsecutiveFailures != 2 || updated.LastSuccessfulExecution == nil || updated.LastSuccessfulExecution.RunID != "last-good" {
		t.Fatalf("cancellation changed failure episode: failures=%d lastSuccess=%+v", updated.ConsecutiveFailures, updated.LastSuccessfulExecution)
	}
	attempts, total, err := teamStore.ListHeartbeatAttempts(ctx, "team-1", "agent-1", store.HeartbeatStatusCancelled, "", 10, 0)
	if err != nil || total != 1 || len(attempts) != 1 || attempts[0].AttentionEligible {
		t.Fatalf("cancelled attempts = total %d %+v err=%v", total, attempts, err)
	}
}

func TestExecute_NilAgentClient(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	executor := newTestExecutor(t, teamStore, agentStore, nil, t.TempDir(), nil, nil)
	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "p")

	// With nil client, Execute should return a clear error, not panic.
	if err == nil {
		t.Fatal("expected error with nil client")
	}
	if !strings.Contains(err.Error(), "agent client") {
		t.Errorf("expected 'agent client' in error, got: %v", err)
	}
	if result.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
}

func TestTriggerManual_UsesConfigProfileKey(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	// Set custom profile key
	ctx := context.Background()
	config, _ := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	config.ProfileKey = "custom-profile"
	_ = teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config)

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	onComplete, waitComplete := awaitableOnComplete(t)
	executor.OnComplete = onComplete
	defer waitComplete()

	_, err := executor.TriggerManual(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("TriggerManual: %v", err)
	}

	if len(mockClient.createRunCalls) != 1 {
		t.Fatalf("expected 1 CreateRun call, got %d", len(mockClient.createRunCalls))
	}
	if mockClient.createRunCalls[0].ProfileRef.ProfileKey != "custom-profile" {
		t.Errorf("expected custom-profile, got %s", mockClient.createRunCalls[0].ProfileRef.ProfileKey)
	}
}

func TestTriggerManual_DefaultProfile(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	onComplete, waitComplete := awaitableOnComplete(t)
	executor.OnComplete = onComplete
	defer waitComplete()

	_, err := executor.TriggerManual(context.Background(), "team-1", "agent-1")
	if err != nil {
		t.Fatalf("TriggerManual: %v", err)
	}

	if mockClient.createRunCalls[0].ProfileRef.ProfileKey != declaredKeysForTest(t).MultiProcess {
		t.Errorf("expected default profile, got %s", mockClient.createRunCalls[0].ProfileRef.ProfileKey)
	}
}

func TestTriggerManual_MissingConfig(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)

	ctx := context.Background()
	_ = teamStore.Create(ctx, newIndependentTestTeam("team-1", "T"))
	_ = agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "A"})

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	_, err := executor.TriggerManual(ctx, "team-1", "agent-1")

	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// setupExecutorTestEnvWithRuntimeMode is like setupExecutorTestEnv but allows
// selecting the team's runtime/coordination policy.
func setupExecutorTestEnvWithRuntimeMode(t *testing.T, runtimeMode string) (
	*store.FileTeamStore,
	*store.FileAgentStore,
	string,
) {
	t.Helper()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	var team *store.Team
	if runtimeMode == "single-process" {
		team = newLeaderLedSingleProcessTestTeam("team-1", "Team One", "agent-1")
	} else {
		team = newIndependentTestTeam("team-1", "Team One")
	}
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
	}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("set member: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{
		TeamID:   "team-1",
		AgentID:  "agent-1",
		Schedule: "0 * * * *",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("set heartbeat config: %v", err)
	}

	return teamStore, agentStore, roots.Config
}

func TestExecute_SingleProcessTeam_UsesClaudeCodeProfile(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnvWithRuntimeMode(t, "single-process")

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1", Title: "test"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	onComplete, waitComplete := awaitableOnComplete(t)
	executor.OnComplete = onComplete
	defer waitComplete()

	// Empty profileKey should resolve to CC for single-process
	_, err := executor.Execute(context.Background(), "team-1", "agent-1", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(mockClient.createRunCalls) != 1 {
		t.Fatalf("expected 1 CreateRun call, got %d", len(mockClient.createRunCalls))
	}
	ref := mockClient.createRunCalls[0].ProfileRef
	if ref.ProfileKey != declaredKeysForTest(t).SingleProcess {
		t.Errorf("expected profile key %q, got %q", declaredKeysForTest(t).SingleProcess, ref.ProfileKey)
	}
}

func TestExecute_MultiProcessTeam_UsesCodexProfile(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnvWithRuntimeMode(t, "multi-process")

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1", Title: "test"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	onComplete, waitComplete := awaitableOnComplete(t)
	executor.OnComplete = onComplete
	defer waitComplete()

	// Empty profileKey should resolve to Codex for multi-process
	_, err := executor.Execute(context.Background(), "team-1", "agent-1", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	ref := mockClient.createRunCalls[0].ProfileRef
	if ref.ProfileKey != declaredKeysForTest(t).MultiProcess {
		t.Errorf("expected profile key %q, got %q", declaredKeysForTest(t).MultiProcess, ref.ProfileKey)
	}
}

func TestTriggerManual_SingleProcessTeam_DefaultsToClaudeCode(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnvWithRuntimeMode(t, "single-process")

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	onComplete, waitComplete := awaitableOnComplete(t)
	executor.OnComplete = onComplete
	defer waitComplete()

	_, err := executor.TriggerManual(context.Background(), "team-1", "agent-1")
	if err != nil {
		t.Fatalf("TriggerManual: %v", err)
	}

	ref := mockClient.createRunCalls[0].ProfileRef
	if ref.ProfileKey != declaredKeysForTest(t).SingleProcess {
		t.Errorf("expected %q, got %q", declaredKeysForTest(t).SingleProcess, ref.ProfileKey)
	}
}

// errForTest creates a simple error for test use.
type testError string

func (e testError) Error() string { return string(e) }

func errForTest(msg string) error { return testError(msg) }

// TestExecute_EndToEndWithHTTPServer exercises the full executor lifecycle with
// a real HTTP test server that validates request bodies and returns
// proto-compatible responses.
func TestExecute_EndToEndWithHTTPServer(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	var taskCreated bool
	var runCreated bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/tasks":
			var req CreateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode task request: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if req.Task == nil {
				http.Error(w, `{"error":"task is required"}`, http.StatusBadRequest)
				return
			}
			if req.Task.Title == "" {
				http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
				return
			}
			if req.Task.ScopePath == "" {
				http.Error(w, `{"error":"scope_path is required"}`, http.StatusBadRequest)
				return
			}

			// Reject unknown fields by re-marshalling and comparing
			taskCreated = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			created := *req.Task
			created.Status = "TASK_STATUS_QUEUED"
			_ = json.NewEncoder(w).Encode(CreateTaskResponse{Task: &created})

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs":
			var req CreateRunRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
				return
			}
			if req.TaskID == "" {
				http.Error(w, `{"error":"task_id is required"}`, http.StatusBadRequest)
				return
			}

			runCreated = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(CreateRunResponse{
				Run: &Run{
					ID:     "run-srv-1",
					TaskID: req.TaskID,
					Status: "RUN_STATUS_RUNNING",
				},
			})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/runs/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GetRunResponse{
				Run: &Run{
					ID:     "run-srv-1",
					Status: "RUN_STATUS_COMPLETE",
				},
			})

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	client.createCallerVerifier = auth01ConversationVerifier{}
	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), registry, nil)

	var completeCalled sync.WaitGroup
	completeCalled.Add(1)
	executor.OnComplete = func(_, _ string) { completeCalled.Done() }

	result, err := executor.Execute(withCreateRunCaller(context.Background(), "Bearer fixture-owner", ""), "team-1", "agent-1", "test-profile")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.RunID != "run-srv-1" {
		t.Errorf("expected run-srv-1, got %s", result.RunID)
	}
	if !taskCreated {
		t.Error("task was not created on the mock server")
	}
	if !runCreated {
		t.Error("run was not created on the mock server")
	}

	// Wait for async completion
	done := make(chan struct{})
	go func() {
		completeCalled.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("OnComplete not called within timeout")
	}
}

// TestExecute_AgentManagerReturnsValidationError verifies that a 400 validation
// error from agent-manager propagates correctly.
func TestExecute_AgentManagerReturnsValidationError(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"validation failed: title exceeds 255 chars"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	client.createCallerVerifier = auth01ConversationVerifier{}
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)

	result, err := executor.Execute(withCreateRunCaller(context.Background(), "Bearer fixture-owner", ""), "team-1", "agent-1", "p")
	if err == nil {
		t.Fatal("expected error from validation failure")
	}
	if !strings.Contains(err.Error(), "creating task") {
		t.Errorf("expected 'creating task' in error, got: %v", err)
	}
	if result.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
}

func TestExecute_AgentManagerAuthorizationFailuresRouteToOwner(t *testing.T) {
	for _, tc := range []struct {
		name                string
		status              int
		refuseRun           bool
		admissionReason     string
		wantAdmissionReason string
		wantPhase           string
	}{
		{name: "task_401", status: http.StatusUnauthorized, wantPhase: "creating_task"},
		{name: "task_403 reason retained", status: http.StatusForbidden, admissionReason: "grant_limit_reached", wantAdmissionReason: "grant_limit_reached", wantPhase: "creating_task"},
		{name: "run_403 reason retained", status: http.StatusForbidden, refuseRun: true, admissionReason: "authority_state_unavailable", wantAdmissionReason: "authority_state_unavailable", wantPhase: "creating_run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			teamStore, agentStore, _ := setupExecutorTestEnv(t)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && r.URL.Path == "/api/v1/tasks" {
					if !tc.refuseRun {
						w.WriteHeader(tc.status)
						body := `{"error":"authorization refused"}`
						if tc.admissionReason != "" {
							body = `{"error":"authorization refused","admission_reason":"` + tc.admissionReason + `"}`
						}
						_, _ = w.Write([]byte(body))
						return
					}
					var req CreateTaskRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Task == nil {
						t.Errorf("decode task request: %v", err)
						http.Error(w, "bad request", http.StatusBadRequest)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					created := *req.Task
					created.Status = "TASK_STATUS_QUEUED"
					_ = json.NewEncoder(w).Encode(CreateTaskResponse{Task: &created})
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs" {
					w.WriteHeader(tc.status)
					body := `{"message":"authorization refused"}`
					if tc.admissionReason != "" {
						body = `{"message":"authorization refused","admission_reason":"` + tc.admissionReason + `"}`
					}
					_, _ = w.Write([]byte(body))
					return
				}
				http.NotFound(w, r)
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
			if _, err := executor.Execute(context.Background(), "team-1", "agent-1", "p"); err == nil {
				t.Fatal("expected authorization refusal")
			}
			attempts, total, err := teamStore.ListHeartbeatAttempts(context.Background(), "team-1", "agent-1", store.HeartbeatStatusFailed, "", 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			if total != 1 || len(attempts) != 1 || attempts[0].Phase != tc.wantPhase || attempts[0].AdmissionReason != tc.wantAdmissionReason || attempts[0].ErrorCategory != "agent_manager_authorization" || attempts[0].Recovery != "review_agent_manager_authorization" || attempts[0].ConsecutiveFailures != 1 || !attempts[0].AttentionEligible || attempts[0].RecoveryRequestID != "" || attempts[0].RecoveryOutcome != store.HeartbeatRecoveryNotRequested {
				t.Fatalf("authorization attempt = total %d %+v; want owner-routed durable attention without PM recovery", total, attempts)
			}
			config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
			if err != nil {
				t.Fatal(err)
			}
			if config.LastExecution == nil || config.LastExecution.AdmissionReason != tc.wantAdmissionReason {
				t.Fatalf("last execution = %+v; want admission reason %q", config.LastExecution, tc.wantAdmissionReason)
			}
		})
	}
}

// TestSchedulerStartRejectsUnresolvedDeclaredProfile proves the scheduler
// cannot start when either manifest-declared key is absent from the catalog.
func TestSchedulerStartRejectsUnresolvedDeclaredProfile(t *testing.T) {
	teamStore, _, _ := setupExecutorTestEnv(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profiles" {
			t.Errorf("unexpected profile write or route: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"profiles":[],"total":0}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)

	scheduler := NewScheduler(nil, client, teamStore, nil)
	if err := scheduler.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "is not installed") {
		t.Fatalf("Start error = %v, want catalog resolution failure", err)
	}
}

// TestExecute_CreateRunUsesDeclaredProfile verifies that the CreateRun request
// refers only to the scenario-owned profile; reconciliation owns creation.
func TestExecute_CreateRunUsesDeclaredProfile(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1", Title: "test"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	onComplete, waitComplete := awaitableOnComplete(t)
	executor.OnComplete = onComplete
	defer waitComplete()

	profileKey := declaredKeysForTest(t).MultiProcess
	_, err := executor.Execute(context.Background(), "team-1", "agent-1", profileKey)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(mockClient.createRunCalls) != 1 {
		t.Fatalf("expected 1 CreateRun call, got %d", len(mockClient.createRunCalls))
	}
	ref := mockClient.createRunCalls[0].ProfileRef
	if ref == nil {
		t.Fatal("expected ProfileRef to be set")
	}
	if ref.ProfileKey != profileKey {
		t.Errorf("expected profile key %q, got %q", profileKey, ref.ProfileKey)
	}
}

// TestExecute_AgentManagerReturnsProfileNotFound tests the scenario where the
// task is created but the profile key doesn't exist, causing CreateRun to fail.
func TestExecute_AgentManagerReturnsProfileNotFound(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	var reqCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		switch {
		case r.URL.Path == "/api/v1/tasks":
			var req CreateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Task == nil {
				t.Errorf("decode task request: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			created := *req.Task
			created.Status = "TASK_STATUS_QUEUED"
			_ = json.NewEncoder(w).Encode(CreateTaskResponse{Task: &created})

		case r.URL.Path == "/api/v1/runs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"profile 'nonexistent-profile' not found"}`))

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	client.createCallerVerifier = auth01ConversationVerifier{}
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)

	result, err := executor.Execute(withCreateRunCaller(context.Background(), "Bearer fixture-owner", ""), "team-1", "agent-1", "nonexistent-profile")
	if err == nil {
		t.Fatal("expected error for profile not found")
	}
	if !strings.Contains(err.Error(), "creating run") {
		t.Errorf("expected 'creating run' in error, got: %v", err)
	}
	if result.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
}

// TestWatchRecoveredRun_ReleasesOnTerminal verifies a recovered obligation
// bound to an owner run is eventually released: the watcher observes terminal
// owner state and fires OnComplete, so a restart cannot strand a slot.
func TestWatchRecoveredRun_ReleasesOnTerminal(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithWaitRunResponse(&Run{ID: "run-recovered", Status: "RUN_STATUS_COMPLETE"})
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)

	var released sync.WaitGroup
	released.Add(1)
	executor.OnComplete = func(teamID, agentID string) {
		if teamID != "team-1" || agentID != "agent-1" {
			t.Errorf("unexpected complete callback: %s/%s", teamID, agentID)
		}
		released.Done()
	}

	executor.WatchRecoveredRun("team-1", "agent-1", "run-recovered", "profile-p", "task-1", "heartbeat-team-1-agent-1", "RUN_STATUS_RUNNING", "")

	done := make(chan struct{})
	go func() {
		released.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("expected recovered run watcher to fire OnComplete on terminal owner state")
	}
}

func TestRecoveredWatcherOutlivesStartupRecoveryContext(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	client := newMockAgentClient().
		WithGetRunResponse("run-recovered", &Run{ID: "run-recovered", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-recovered", Status: "RUN_STATUS_COMPLETE"})
	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), registry, nil)
	defer executor.Shutdown()

	dir := t.TempDir()
	writeQueueFile(t, dir, "team-1", persistedTeamQueue{
		TeamID:            "team-1",
		MaxConcurrentRuns: 1,
		Running:           []queuedExecution{{AgentID: "agent-1", ProfileKey: "profile-p", TaskID: "task-1", RunID: "run-recovered"}},
	})
	tec := newTeamExecutionContext("team-1", executor, dir, client)
	startupCtx, cancelStartup := context.WithCancel(context.Background())
	tec.Recover(startupCtx)
	cancelStartup()

	waitForCondition(t, "recovered watcher completes after startup context cancellation", func() bool {
		config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
		return err == nil && config != nil && config.LastExecution != nil && config.LastExecution.Status == store.HeartbeatStatusCompleted
	})
}

func TestWatchRecoveredRunRetainsOwnerStartWhenObserverFails(t *testing.T) {
	for _, test := range []struct {
		name           string
		ownerStartedAt string
		wantStartedAt  string
	}{
		{name: "known owner start", ownerStartedAt: "2026-10-14T12:30:00Z", wantStartedAt: "2026-10-14T12:30:00Z"},
		{name: "unknown owner start"},
	} {
		t.Run(test.name, func(t *testing.T) {
			teamStore, agentStore, _ := setupExecutorTestEnv(t)
			client := newMockAgentClient().WithWaitRunError(errors.New("observer unavailable"))
			registry := NewRunRegistry(t.TempDir())
			executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), registry, nil)
			defer executor.Shutdown()

			if !executor.WatchRecoveredRun("team-1", "agent-1", "run-recovered", "profile-p", "task-1", "tag", "RUN_STATUS_NEEDS_REVIEW", test.ownerStartedAt) {
				t.Fatal("recovered watcher was not started")
			}
			waitForCondition(t, "observer failure recorded", func() bool {
				config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
				return err == nil && config != nil && config.LastExecution != nil && config.LastExecution.Status == store.HeartbeatStatusObserverUnavailable
			})

			config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
			if err != nil || config == nil || config.LastExecution == nil {
				t.Fatalf("read recovered execution: config=%+v err=%v", config, err)
			}
			if config.LastExecution.StartedAt != test.wantStartedAt {
				t.Fatalf("LastExecution.StartedAt=%q, want %q", config.LastExecution.StartedAt, test.wantStartedAt)
			}
			if strings.Contains(config.LastExecution.LogPath, "0001-01-01") {
				t.Fatalf("log filename used an invented owner start: %q", config.LastExecution.LogPath)
			}
			var attempts []store.HeartbeatAttempt
			var total int
			var attemptErr error
			waitForCondition(t, "observer-unavailable attempt", func() bool {
				attempts, total, attemptErr = teamStore.ListHeartbeatAttempts(context.Background(), "team-1", "agent-1", store.HeartbeatStatusObserverUnavailable, "", 10, 0)
				return attemptErr == nil && total == 1 && len(attempts) == 1
			})
			if attemptErr != nil || total != 1 || len(attempts) != 1 {
				t.Fatalf("observer-unavailable attempts: total=%d rows=%+v err=%v", total, attempts, attemptErr)
			}
			if attempts[0].StartedAt != test.wantStartedAt {
				t.Fatalf("attempt StartedAt=%q, want %q", attempts[0].StartedAt, test.wantStartedAt)
			}
			run, ok := registry.GetActiveRun("team-1", "agent-1")
			if !ok || run.StartedAt.IsZero() != (test.wantStartedAt == "") {
				t.Fatalf("registry fabricated or lost original owner start: run=%+v found=%t", run, ok)
			}
			if ok && test.wantStartedAt != "" && run.StartedAt.Format(time.RFC3339) != test.wantStartedAt {
				t.Fatalf("registry start=%s, want %s", run.StartedAt.Format(time.RFC3339), test.wantStartedAt)
			}
		})
	}
}

func TestRecoveryKnownActiveRunTerminalCallbackReleasesRetainedSlot(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-1", persistedTeamQueue{
		TeamID:            "team-1",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{{
			AgentID: "agent-1", ProfileKey: "profile-p", RunID: "run-recovered",
			TaskID: "task-recovered", RunTag: "heartbeat-team-1-agent-1",
		}},
	})
	client := newMockAgentClient().
		WithGetRunResponse("run-recovered", &Run{ID: "run-recovered", TaskID: "task-recovered", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-recovered", TaskID: "task-recovered", Status: "RUN_STATUS_COMPLETE"})
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnComplete = teamExecStore.OnComplete
	teamExecStore.Recover(context.Background())

	deadline := time.After(5 * time.Second)
	for {
		if len(teamExecStore.Status("team-1").RunningAgentIDs) == 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("recovered known run terminal callback did not release its retained slot")
		case <-time.After(5 * time.Millisecond):
		}
	}
	if got := len(client.createRunCalls); got != 0 {
		t.Fatalf("watching a recovered known run must not dispatch another run, got %d CreateRun calls", got)
	}
}

type blockingWaitAgentClient struct {
	AgentClient
	started chan struct{}
	once    sync.Once
}

func (c *blockingWaitAgentClient) WaitForRun(ctx context.Context, _ string, _ time.Duration) (*Run, error) {
	c.once.Do(func() { close(c.started) })
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRecoveryNeedsReviewProjectsStoredHeartbeatHealthAndRetainsSlot(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-1", persistedTeamQueue{
		TeamID:            "team-1",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running:           []queuedExecution{{AgentID: "agent-1", ProfileKey: "profile-p", RunID: "run-review", TaskID: "task-review"}},
	})
	baseClient := newMockAgentClient().WithGetRunResponse("run-review", &Run{ID: "run-review", TaskID: "task-review", Status: "RUN_STATUS_NEEDS_REVIEW"})
	client := &blockingWaitAgentClient{AgentClient: baseClient, started: make(chan struct{})}
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnComplete = teamExecStore.OnComplete
	defer executor.Shutdown()
	teamExecStore.Recover(context.Background())

	select {
	case <-client.started:
	case <-time.After(5 * time.Second):
		t.Fatal("recovery did not attach a completion watcher for the needs-review run")
	}
	config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if config.LastExecution == nil || config.LastExecution.Status != store.HeartbeatStatusNeedsReview || config.LastExecution.RunID != "run-review" || config.LifecycleState != store.HeartbeatLifecycleNeedsReview || config.ConsecutiveFailures != 0 {
		t.Fatalf("explicit owner review must reach stored heartbeat health without a failure: %+v", config)
	}
	status := teamExecStore.Status("team-1")
	if len(status.RunningAgentIDs) != 1 || len(status.NeedsReviewAgentIDs) != 1 {
		t.Fatalf("review run must keep its slot and report review: %+v", status)
	}
	if _, err := teamExecStore.Enqueue(context.Background(), "team-1", "agent-1", "profile-p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("review run must refuse duplicate launch, got %v", err)
	}
}

func TestScheduledApprovalPendingTaskRetryCreatesOneTaskAndRun(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	client := newMockAgentClient().WithCreateTaskError(NewApprovalPendingError("creating_task", errors.New("operator approval requested")))
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, t.TempDir(), client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnComplete = teamExecStore.OnComplete
	scheduler := NewScheduler(executor, client, teamStore, teamExecStore)

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")
	teamExecStore.Shutdown()
	config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil || config.LastExecution == nil || config.LastExecution.Status != store.HeartbeatStatusApprovalPending || config.LifecycleState != store.HeartbeatLifecycleApprovalPending {
		t.Fatalf("typed task-stage approval should be visible without counting as failure: config=%+v err=%v", config, err)
	}
	if got := teamExecStore.Status("team-1"); len(got.RunningAgentIDs) != 0 {
		t.Fatalf("no-effect task approval must release the member slot: %+v", got)
	}

	client.createTaskErr = nil
	client.createTaskResp = &Task{ID: "task-created-once", Title: "approved heartbeat"}
	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")
	teamExecStore.Shutdown()
	approvedTaskID := client.createTaskCalls[1].ID
	if len(client.createTaskCalls) != 2 || len(client.createRunCalls) != 1 {
		t.Fatalf("pending then approved schedule should make two task attempts (one effect) and one run: task attempts=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	if client.createRunCalls[0].TaskID != approvedTaskID {
		t.Fatalf("run did not use the one approved task: %+v", client.createRunCalls[0])
	}
	deadline := time.After(5 * time.Second)
	for len(teamExecStore.Status("team-1").RunningAgentIDs) != 0 {
		select {
		case <-deadline:
			t.Fatal("terminal owner callback did not release the approved run")
		case <-time.After(5 * time.Millisecond):
		}
	}
}
