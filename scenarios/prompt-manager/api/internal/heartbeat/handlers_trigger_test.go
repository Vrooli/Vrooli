package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/requestpolicy"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
)

func TestTriggerHeartbeatRequiresConfig(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
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

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.runCallerValidator = auth01ConversationVerifier{}

	req.Header.Set("Authorization", "Bearer fixture-owner")

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerHeartbeatRequiresMembership(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.runCallerValidator = auth01ConversationVerifier{}

	req.Header.Set("Authorization", "Bearer fixture-owner")

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerHeartbeat_MemberAlreadyQueued(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
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
	teamExecStore := NewTeamExecutionStore(teamStore, exec, t.TempDir(), nil)
	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
		TeamExecStore: teamExecStore,
	})

	// First trigger should succeed (202 Accepted)
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()
	handlers.runCallerValidator = auth01ConversationVerifier{}
	req.Header.Set("Authorization", "Bearer fixture-owner")
	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("first trigger: expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Second trigger for same agent should return 409 (already queued/running)
	req2 := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w2 := httptest.NewRecorder()
	handlers.runCallerValidator = auth01ConversationVerifier{}
	req2.Header.Set("Authorization", "Bearer fixture-owner")
	handlers.TriggerHeartbeat(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("second trigger: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestTriggerHeartbeatReplacesRecoveredInterruptedPreparation(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatal(err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatal(err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	// The manual endpoint has historically remained available when a member's
	// scheduled heartbeat is disabled. Recovery must preserve that route's
	// existing admission contract; scheduler-only enabled checks stay separate.
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	executor := &blockingPreparationExecutor{started: make(chan string, 1), release: make(chan struct{})}
	teamExecStore := NewTeamExecutionStore(teamStore, executor, t.TempDir(), nil)
	queue := teamExecStore.GetOrCreate("team-1")
	defer func() {
		close(executor.release)
		queue.executionWG.Wait()
	}()
	queue.running["agent-1"] = runningEntry{DispatchStage: DispatchStagePreparing, ProfileKey: "old-profile"}
	if err := queue.persistLocked(); err != nil {
		t.Fatal(err)
	}
	queue.Recover(ctx)

	handlers := NewHandlers(HandlersDeps{
		TeamStore: teamStore, AgentStore: agentStore, RelationStore: relationStore,
		Executor: newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil), TeamExecStore: teamExecStore,
	})
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	req.Header.Set("Authorization", "Bearer fixture-owner")
	w := httptest.NewRecorder()
	handlers.runCallerValidator = auth01ConversationVerifier{}
	handlers.TriggerHeartbeat(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("manual trigger did not replace interrupted preparation: status=%d body=%s", w.Code, w.Body.String())
	}
	select {
	case agentID := <-executor.started:
		if agentID != "agent-1" {
			t.Fatalf("started agent %q, want agent-1", agentID)
		}
	case <-time.After(time.Second):
		t.Fatal("manual trigger did not start after replacing interrupted preparation")
	}
	status := teamExecStore.Status("team-1")
	if len(status.PreparationInterruptedAgentIDs) != 0 || len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("execution status does not reflect fresh manual admission: %#v", status)
	}
	manualConfig, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if manualConfig.Enabled || manualConfig.LastManualTriggerAt == "" {
		t.Fatalf("manual replacement changed disabled schedule policy or failed to record trigger: %+v", manualConfig)
	}
}

func TestTriggerHeartbeat_DeduplicatesCompletedManualTriggerWithinScheduleWindow(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatal(err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
		t.Fatal(err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Schedule: "0 * * * *", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, &captureExecutor{}, t.TempDir(), nil)
	handlers := NewHandlers(HandlersDeps{TeamStore: teamStore, AgentStore: agentStore, RelationStore: relationStore, Executor: executor, TeamExecStore: teamExecStore})

	trigger := func(query ...string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger"+strings.Join(query, ""), nil)
		req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
		w := httptest.NewRecorder()
		handlers.runCallerValidator = auth01ConversationVerifier{}
		req.Header.Set("Authorization", "Bearer fixture-owner")
		handlers.TriggerHeartbeat(w, req)
		return w
	}
	if first := trigger(); first.Code != http.StatusAccepted {
		t.Fatalf("first trigger: status=%d body=%s", first.Code, first.Body.String())
	}
	teamExecStore.contexts["team-1"].OnMemberComplete("agent-1")
	second := trigger()
	if second.Code != http.StatusAccepted || !strings.Contains(second.Body.String(), `"status":"deduplicated"`) {
		t.Fatalf("second trigger: status=%d body=%s", second.Code, second.Body.String())
	}
	// An operator force bypasses the window while the team iterates.
	forced := trigger("?force=true")
	if forced.Code != http.StatusAccepted || strings.Contains(forced.Body.String(), `"status":"deduplicated"`) {
		t.Fatalf("forced trigger: status=%d body=%s", forced.Code, forced.Body.String())
	}
}

// TestTriggerHeartbeat_FullPathWithTeamExecStore exercises the full production code
// path: handler → teamExecStore → executor → mock HTTP client.
func TestTriggerHeartbeat_FullPathWithTeamExecStore(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
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
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), registry, nil)
	executor.OnRunComplete = func(_, _, _ string) {}

	teamExecStore := NewTeamExecutionStore(teamStore, executor, t.TempDir(), nil)

	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
		RunRegistry:   registry,
		AgentClient:   mockClient,
		TeamExecStore: teamExecStore,
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.runCallerValidator = auth01ConversationVerifier{}

	req.Header.Set("Authorization", "Bearer fixture-owner")

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
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
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

	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), nil, nil)
	completed := make(chan struct{})
	executor.OnRunComplete = func(_, _, _ string) {
		close(completed)
	}

	// No teamExecStore — should use direct execution fallback
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
		AgentClient:   mockClient,
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.runCallerValidator = auth01ConversationVerifier{}

	req.Header.Set("Authorization", "Bearer fixture-owner")

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
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
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

	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()

	handlers.runCallerValidator = auth01ConversationVerifier{}

	req.Header.Set("Authorization", "Bearer fixture-owner")

	handlers.TriggerHeartbeat(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEffectiveExecutionStateDistinguishesRetainedConfiguration(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	team := newIndependentTestTeam("team-1", "Team")
	team.Enabled = false
	if err := teamStore.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	handlers := NewHandlers(HandlersDeps{TeamStore: teamStore})
	config := &store.HeartbeatConfig{TeamID: team.ID, AgentID: "agent-1", Enabled: true, Schedule: "*/5 * * * *"}
	teamEnabled, state, reason, controlState, scheduled := handlers.effectiveExecutionState(context.Background(), config)

	if teamEnabled {
		t.Fatal("retained heartbeat configuration must not make a disabled team executable")
	}
	if state != "team-disabled" {
		t.Fatalf("effective state = %q, want team-disabled", state)
	}
	if reason != "team is disabled" {
		t.Fatalf("effective reason = %q, want team is disabled", reason)
	}
	if controlState != "active" {
		t.Fatalf("control state = %q, want active", controlState)
	}
	if scheduled {
		t.Fatal("disabled team must not report a scheduler entry")
	}
}

func TestEffectiveExecutionStateReportsArchivedTeamAndBlocksTrigger(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()

	team := newIndependentTestTeam("team-archived", "Archived Team")
	team.Enabled = false
	team.Archived = true
	if err := teamStore.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := agentStore.Create(context.Background(), &store.Agent{ID: "agent-archived", DisplayName: "Agent"}); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := relationStore.SetTeamMember(context.Background(), &store.TeamMemberRelation{
		TeamID: "team-archived", AgentID: "agent-archived", Status: store.MemberStatusActive,
	}); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-archived", "agent-archived", &store.HeartbeatConfig{
		TeamID: "team-archived", AgentID: "agent-archived", Schedule: "0 */6 * * *", Enabled: true,
	}); err != nil {
		t.Fatalf("set heartbeat config: %v", err)
	}

	handlers := NewHandlers(HandlersDeps{
		TeamStore: teamStore, AgentStore: agentStore, RelationStore: relationStore,
		Executor: newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil),
	})
	config := &store.HeartbeatConfig{TeamID: team.ID, AgentID: "agent-archived", Enabled: true, Schedule: "*/5 * * * *"}
	teamEnabled, state, reason, _, scheduled := handlers.effectiveExecutionState(context.Background(), config)
	if teamEnabled || state != "team-archived" || reason != "team is archived" || scheduled {
		t.Fatalf("archived execution state = enabled:%v state:%q reason:%q scheduled:%v", teamEnabled, state, reason, scheduled)
	}

	req := httptest.NewRequest(http.MethodPost, "/teams/team-archived/heartbeats/agent-archived/trigger", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-archived", "agentId": "agent-archived"})
	w := httptest.NewRecorder()
	handlers.runCallerValidator = auth01ConversationVerifier{}
	req.Header.Set("Authorization", "Bearer fixture-owner")
	handlers.TriggerHeartbeat(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected archived team trigger status 409, got %d: %s", w.Code, w.Body.String())
	}
}

// A delivery team's orchestrator may wake a supervision team's member (its
// step-back route); any other cross-team trigger stays rejected.
func TestTriggerHeartbeat_DeliveryMemberMayWakeSupervisor(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	ctx := context.Background()
	for id, purpose := range map[string]string{"supervision": teamconfig.PurposeSupervision, "delivery": teamconfig.PurposeDelivery, "other": ""} {
		team := newIndependentTestTeam(id, id)
		team.Purpose = purpose
		if err := teamStore.Create(ctx, team); err != nil {
			t.Fatal(err)
		}
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Supervisor"}); err != nil {
		t.Fatal(err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "supervision", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "supervision", "agent-1", &store.HeartbeatConfig{TeamID: "supervision", AgentID: "agent-1", Schedule: "0 13 * * *", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, &captureExecutor{}, t.TempDir(), nil)
	handlers := NewHandlers(HandlersDeps{TeamStore: teamStore, AgentStore: agentStore, RelationStore: relationStore, Executor: executor, TeamExecStore: teamExecStore})

	trigger := func(fromTeam string) *httptest.ResponseRecorder {
		member := "orchestrator"
		req := httptest.NewRequest(http.MethodPost, "/teams/supervision/heartbeats/agent-1/trigger", nil)
		req.Header.Set(attributionHeaderName, encodeAttribution(t, store.AttributionInfo{Kind: store.KnowledgeKindAgentMember, SpawnOrigin: store.SpawnOriginHeartbeat, TeamID: &fromTeam, MemberID: &member}))
		req = mux.SetURLVars(req, map[string]string{"id": "supervision", "agentId": "agent-1"})
		w := httptest.NewRecorder()
		handlers.runCallerValidator = auth01ConversationVerifier{}
		req.Header.Set("Authorization", "Bearer fixture-owner")
		handlers.TriggerHeartbeat(w, req)
		return w
	}
	if got := trigger("other"); got.Code != http.StatusForbidden {
		t.Fatalf("non-delivery cross-team trigger: status=%d body=%s", got.Code, got.Body.String())
	}
	if got := trigger("delivery"); got.Code != http.StatusAccepted {
		t.Fatalf("delivery wake of supervisor: status=%d body=%s", got.Code, got.Body.String())
	}
}

// P-18: with AUTH-01 off (the default) an agent-attributed delivery member that
// offers only its run identity may wake a supervisor through the real owner
// client queue. With enforcement on the same request is refused before effects.
func TestTriggerHeartbeat_AgentDeliveryMemberWakesSupervisorWithEnforcementOff(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	ctx := context.Background()
	for id, purpose := range map[string]string{"supervision": teamconfig.PurposeSupervision, "delivery": teamconfig.PurposeDelivery} {
		team := newIndependentTestTeam(id, id)
		team.Purpose = purpose
		if err := teamStore.Create(ctx, team); err != nil {
			t.Fatal(err)
		}
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Supervisor"}); err != nil {
		t.Fatal(err)
	}
	if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "supervision", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "supervision", "agent-1", &store.HeartbeatConfig{TeamID: "supervision", AgentID: "agent-1", Schedule: "0 13 * * *", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	capture := &captureExecutor{}
	ownerClient := &AgentManagerClient{createCallerVerifier: auth01ConversationVerifier{}}
	teamExecStore := NewTeamExecutionStore(teamStore, capture, t.TempDir(), ownerClient)
	handlers := NewHandlers(HandlersDeps{TeamStore: teamStore, AgentStore: agentStore, RelationStore: relationStore, Executor: newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil), TeamExecStore: teamExecStore, RunCallerValidator: auth01ConversationVerifier{}})
	trigger := func() *httptest.ResponseRecorder {
		fromTeam, member := "delivery", "orchestrator"
		req := httptest.NewRequest(http.MethodPost, "/teams/supervision/heartbeats/agent-1/trigger", nil)
		req.Header.Set(attributionHeaderName, encodeAttribution(t, store.AttributionInfo{Kind: store.KnowledgeKindAgentMember, SpawnOrigin: store.SpawnOriginHeartbeat, TeamID: &fromTeam, MemberID: &member}))
		req.Header.Set("X-Agent-Identity-Token", "delivery-orchestrator-run")
		req = mux.SetURLVars(req, map[string]string{"id": "supervision", "agentId": "agent-1"})
		w := httptest.NewRecorder()
		handlers.TriggerHeartbeat(w, req)
		return w
	}

	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
	if got := trigger(); got.Code != http.StatusUnauthorized {
		t.Fatalf("agent wake admitted while enforced: status=%d body=%s", got.Code, got.Body.String())
	}
	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "")
	if got := trigger(); got.Code != http.StatusAccepted {
		t.Fatalf("delivery agent wake of supervisor refused while off: status=%d body=%s", got.Code, got.Body.String())
	}
	teamExecStore.GetOrCreate("supervision").Shutdown()
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if len(capture.calls) != 1 || capture.calls[0].agentID != "agent-1" {
		t.Fatalf("supervisor wake was not dispatched: %+v", capture.calls)
	}
}

func TestHeartbeatResponseExposesOnlyPersistedAdmissionOrigin(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Team")); err != nil {
		t.Fatal(err)
	}
	handlers := &Handlers{teamStore: teamStore}
	config := &store.HeartbeatConfig{
		TeamID: "team-1", AgentID: "agent-1",
		LastExecution: &store.HeartbeatExecResult{
			RunID: "run-known", Status: store.HeartbeatStatusCompleted, AdmissionOrigin: AdmissionOriginManual,
		},
		LastSuccessfulExecution: &store.HeartbeatExecResult{
			RunID: "run-known", Status: store.HeartbeatStatusCompleted, AdmissionOrigin: AdmissionOriginManual,
		},
	}
	response := handlers.toResponse(config)
	if response.LastExecution == nil || response.LastExecution.AdmissionOrigin != AdmissionOriginManual {
		t.Fatalf("response LastExecution = %+v; want persisted manual origin", response.LastExecution)
	}
	if response.LastSuccessfulExecution == nil || response.LastSuccessfulExecution.AdmissionOrigin != AdmissionOriginManual {
		t.Fatalf("response LastSuccessfulExecution = %+v; want persisted manual origin", response.LastSuccessfulExecution)
	}

	config.LastExecution.AdmissionOrigin = ""
	response = handlers.toResponse(config)
	encoded, err := json.Marshal(response.LastExecution)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "admissionOrigin") {
		t.Fatalf("legacy unknown origin should remain omitted, response JSON = %s", encoded)
	}
}

func TestBoundedManualTriggerRunCreateContractControlsActualQueueDispatch(t *testing.T) {
	teamPrepare := &identity.OperationGrant{
		ID: "team-prepare-only", Policy: "team-preparation", Approver: "owner",
		Rules:     []identity.OperationRule{{Policy: "team-preparation", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/team-1"}}},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	runCreate := &identity.OperationGrant{
		ID: "run-create", Policy: "run-creation", Approver: "owner",
		Rules:     []identity.OperationRule{{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	cases := []struct {
		name                 string
		principal            identity.Principal
		requestAuthorityDeny bool
		enforce              string
		wantStatus           int
		wantCalls            int
	}{
		{
			name: "team.prepare-only agent cannot dispatch while AUTH01 is unset",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: teamPrepare,
			},
			enforce: "", wantStatus: http.StatusForbidden, wantCalls: 0,
		},
		{
			name: "wrong-target run.create grant cannot dispatch",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: &identity.OperationGrant{
					ID: "wrong-target-run-create", Policy: "run-creation", Approver: "owner",
					Rules:     []identity.OperationRule{{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:task/other"}}},
					ExpiresAt: time.Now().Add(time.Hour),
				},
			},
			enforce: "", wantStatus: http.StatusForbidden, wantCalls: 0,
		},
		{
			name: "expired run.create grant cannot dispatch",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: &identity.OperationGrant{
					ID: "expired-run-create", Policy: "run-creation", Approver: "owner",
					Rules:     []identity.OperationRule{{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
					ExpiresAt: time.Now().Add(-time.Minute),
				},
			},
			enforce: "", wantStatus: http.StatusForbidden, wantCalls: 0,
		},
		{
			name: "denied request authority cannot dispatch",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"},
			},
			requestAuthorityDeny: true, enforce: "", wantStatus: http.StatusForbidden, wantCalls: 0,
		},
		{
			name: "existing run.create grant reaches member queue without browser login",
			principal: identity.Principal{
				Kind: identity.ActorAgent, Subject: "qa-run", Owner: "owner", Verified: true,
				Scopes: []string{"agent-manager:read", "agent-manager:write"}, Grant: runCreate,
			},
			enforce: "", wantStatus: http.StatusAccepted, wantCalls: 1,
		},
		{
			name: "verified human proof still passes strict AUTH01 mode",
			principal: identity.Principal{
				Kind: identity.ActorHuman, Subject: "owner", Verified: true,
				Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour), Grant: teamPrepare,
			},
			enforce: "true", wantStatus: http.StatusAccepted, wantCalls: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(owneridentity.CreateRunCallerEnforceEnv, tc.enforce)
			if owneridentity.CreateRunCallerEnforced() != (tc.enforce == "true") {
				t.Fatal("test did not establish requested AUTH01 mode")
			}
			roots := paths.RootsForTest(t)
			fileStore := newFileStore(t, roots)
			teamStore := fileStore.Teams().(*store.FileTeamStore)
			agentStore := fileStore.Agents().(*store.FileAgentStore)
			relationStore := fileStore.Relations()
			ctx := context.Background()
			if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team")); err != nil {
				t.Fatal(err)
			}
			if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent"}); err != nil {
				t.Fatal(err)
			}
			if err := relationStore.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
				t.Fatal(err)
			}
			if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Enabled: true, Schedule: "0 * * * *"}); err != nil {
				t.Fatal(err)
			}

			queueExecutor := &captureExecutor{}
			teamExecStore := NewTeamExecutionStore(teamStore, queueExecutor, t.TempDir(), nil)
			defer teamExecStore.Shutdown()
			handlers := NewHandlers(HandlersDeps{
				TeamStore: teamStore, AgentStore: agentStore, RelationStore: relationStore,
				Executor:      newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil),
				TeamExecStore: teamExecStore, RunCallerValidator: auth01ConversationVerifier{},
			})
			req := httptest.NewRequest(http.MethodPost, heartbeatconnect.HeartbeatServiceTriggerHeartbeatProcedure, nil)
			req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
			req = req.WithContext(identity.WithPrincipal(req.Context(), tc.principal))
			if tc.requestAuthorityDeny {
				req = req.WithContext(authn.WithRequestOperationAuthority(req.Context(), heartbeatManagementAuthority{
					capability: "agent-manager:write", operation: "team.prepare", target: "prompt-manager:team/team-1",
				}))
			}
			if tc.enforce == "true" {
				req.Header.Set("Authorization", "Bearer fixture-owner")
			}
			rec := httptest.NewRecorder()
			requestpolicy.PreparationBoundary(http.HandlerFunc(handlers.TriggerHeartbeat)).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("trigger status=%d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			teamExecStore.Shutdown()
			queueExecutor.mu.Lock()
			calls := len(queueExecutor.calls)
			queueExecutor.mu.Unlock()
			if calls != tc.wantCalls {
				t.Fatalf("queue dispatch calls=%d, want %d", calls, tc.wantCalls)
			}
		})
	}
}
