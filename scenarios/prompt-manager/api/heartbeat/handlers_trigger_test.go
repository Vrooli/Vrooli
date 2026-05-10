package heartbeat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"prompt-manager/store"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestTriggerHeartbeatRequiresConfig(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerHeartbeatRequiresMembership(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerHeartbeat_MemberAlreadyQueued(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	exec := &captureExecutor{}
	teamExecStore := NewTeamExecutionStore(teamStore, exec, t.TempDir())
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, teamExecStore)

	// First trigger should succeed (202 Accepted)
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()
	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("first trigger: expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Second trigger for same agent should return 409 (already queued/running)
	req2 := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w2 := httptest.NewRecorder()
	handlers.TriggerHeartbeat(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("second trigger: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestTriggerHeartbeat_FullPathWithTeamExecStore exercises the full production code
// path: handler → teamExecStore → executor → mock HTTP client.
func TestTriggerHeartbeat_FullPathWithTeamExecStore(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{
		TeamID: "team-1", AgentID: "agent-1", Schedule: "0 * * * *", Enabled: true,
		ProfileKey: "test-profile",
	}); err != nil {
		t.Fatalf("set config: %v", err)
	}

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1", Title: "test"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	registry := NewRunRegistry(t.TempDir())
	executor := NewExecutor(teamStore, agentStore, mockClient, t.TempDir(), registry, nil)
	executor.OnComplete = func(_, _ string) {}

	teamExecStore := NewTeamExecutionStore(teamStore, executor, t.TempDir())

	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, registry, mockClient, teamExecStore)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Give the async goroutine time to call the mock client
	time.Sleep(200 * time.Millisecond)

	mockClient.mu.Lock()
	taskCalls := len(mockClient.createTaskCalls)
	runCalls := len(mockClient.createRunCalls)
	mockClient.mu.Unlock()

	if taskCalls < 1 {
		t.Error("expected CreateTask to be called via teamExecStore path")
	}
	if runCalls < 1 {
		t.Error("expected CreateRun to be called via teamExecStore path")
	}

	// Verify the profile key was propagated correctly
	mockClient.mu.Lock()
	if runCalls > 0 {
		profileKey := mockClient.createRunCalls[0].ProfileRef.ProfileKey
		if profileKey != "test-profile" {
			t.Errorf("expected profile key 'test-profile', got %q", profileKey)
		}
	}
	mockClient.mu.Unlock()
}

// TestTriggerHeartbeat_DirectExecutionFallback exercises the fallback path when
// no teamExecStore is configured.
func TestTriggerHeartbeat_DirectExecutionFallback(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{
		TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{
		TeamID: "team-1", AgentID: "agent-1", Schedule: "0 * * * *", Enabled: true,
	}); err != nil {
		t.Fatalf("set config: %v", err)
	}

	mockClient := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-1", Title: "test"}).
		WithCreateRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"})

	executor := NewExecutor(teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	completed := make(chan struct{})
	executor.OnComplete = func(_, _ string) {
		close(completed)
	}

	// No teamExecStore — should use direct execution fallback
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, mockClient, nil)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Direct path should have created task and run
	mockClient.mu.Lock()
	if len(mockClient.createTaskCalls) != 1 {
		t.Errorf("expected 1 CreateTask call, got %d", len(mockClient.createTaskCalls))
	}
	if len(mockClient.createRunCalls) != 1 {
		t.Errorf("expected 1 CreateRun call, got %d", len(mockClient.createRunCalls))
	}
	mockClient.mu.Unlock()

	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for direct execution completion")
	}
}

func TestTriggerHeartbeatBlockedWhenTeamDisabled(t *testing.T) {
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	team := newIndependentTestTeam("team-1", "Team")
	team.Enabled = false
	if err := teamStore.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID:  "team-1",
		AgentID: "agent-1",
		Status:  store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-1", "agent-1", &store.HeartbeatConfig{
		TeamID:   "team-1",
		AgentID:  "agent-1",
		Schedule: "0 */6 * * *",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("set heartbeat config: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}
