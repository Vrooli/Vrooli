package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
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
	if runReq.TaskID != "task-100" {
		t.Errorf("expected task ID task-100, got %s", runReq.TaskID)
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
}

func TestExecute_WaitForRunFailure(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunError(errForTest("timeout"))

	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), registry, nil)

	var completeCalled sync.WaitGroup
	completeCalled.Add(1)
	executor.OnComplete = func(_, _ string) { completeCalled.Done() }

	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "p")
	if err != nil {
		t.Fatalf("Execute should succeed (async wait), got: %v", err)
	}
	if result.Status != store.HeartbeatStatusRunning {
		t.Errorf("expected running, got %s", result.Status)
	}

	// Wait for async completion
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

	// Config should show failed
	config, _ := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if config.LastExecution == nil || config.LastExecution.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed after wait error, got %+v", config.LastExecution)
	}
}

func TestExecute_RunFailed(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

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

	config, _ := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if config.LastExecution == nil {
		t.Fatal("expected LastExecution")
	}
	if config.LastExecution.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed, got %s", config.LastExecution.Status)
	}
	if config.LastExecution.Error != "agent crashed" {
		t.Errorf("expected error message, got %s", config.LastExecution.Error)
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

	if mockClient.createRunCalls[0].ProfileRef.ProfileKey != DefaultProfileKeyMultiProcess {
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
	if ref.ProfileKey != DefaultProfileKeySingleProcess {
		t.Errorf("expected profile key %q, got %q", DefaultProfileKeySingleProcess, ref.ProfileKey)
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
	if ref.ProfileKey != DefaultProfileKeyMultiProcess {
		t.Errorf("expected profile key %q, got %q", DefaultProfileKeyMultiProcess, ref.ProfileKey)
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
	if ref.ProfileKey != DefaultProfileKeySingleProcess {
		t.Errorf("expected %q, got %q", DefaultProfileKeySingleProcess, ref.ProfileKey)
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
			_ = json.NewEncoder(w).Encode(CreateTaskResponse{
				Task: &Task{
					ID:          "task-srv-1",
					Title:       req.Task.Title,
					Description: req.Task.Description,
					ScopePath:   req.Task.ScopePath,
				},
			})

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
	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), registry, nil)

	var completeCalled sync.WaitGroup
	completeCalled.Add(1)
	executor.OnComplete = func(_, _ string) { completeCalled.Done() }

	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "test-profile")
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
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)

	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "p")
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

// TestEnsureProfileFailure_CausesCreateRunProfileNotFound tests the real failure
// chain: EnsureProfile fails at startup → profile never created → CreateRun
// fails with "profile not found". This is the exact bug that was happening in
// production due to the Timeout serialization issue.
func TestEnsureProfileFailure_CausesCreateRunProfileNotFound(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)

	// Simulate the failure chain with an HTTP server:
	// 1. EnsureProfile returns 400 (bad Duration format)
	// 2. CreateTask succeeds
	// 3. CreateRun returns 400 "profile not found" (because EnsureProfile failed)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/profiles/ensure":
			// EnsureProfile fails due to invalid Duration format
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"proto: (line 1:174): invalid google.protobuf.Duration value 600000000000"}`))

		case r.URL.Path == "/api/v1/tasks":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(CreateTaskResponse{
				Task: &Task{ID: "task-1", Title: "test"},
			})

		case r.URL.Path == "/api/v1/runs":
			// CreateRun fails because the profile was never created
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"profile 'prompt-manager-heartbeat' not found"}`))

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)

	// Create scheduler and attempt to start it (EnsureProfile will fail)
	scheduler := NewScheduler(nil, client, teamStore, nil)
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("Start should not fail (EnsureProfile failure is non-fatal): %v", err)
	}
	defer scheduler.Stop()

	// Now execute a heartbeat - it should fail at CreateRun because profile doesn't exist
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "prompt-manager-heartbeat")
	if err == nil {
		t.Fatal("expected error from CreateRun (profile not found)")
	}
	if !strings.Contains(err.Error(), "creating run") {
		t.Errorf("expected 'creating run' in error, got: %v", err)
	}
	if result.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected failed status, got %s", result.Status)
	}

	// Verify the config was updated to reflect the failure
	config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.LastExecution == nil {
		t.Fatal("expected LastExecution to be set after failure")
	}
	if config.LastExecution.Status != store.HeartbeatStatusFailed {
		t.Errorf("expected config status failed, got %s", config.LastExecution.Status)
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

	profileKey := "prompt-manager/internal/heartbeat"
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(CreateTaskResponse{
				Task: &Task{ID: "task-1", Title: "test"},
			})

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
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)

	result, err := executor.Execute(context.Background(), "team-1", "agent-1", "nonexistent-profile")
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
