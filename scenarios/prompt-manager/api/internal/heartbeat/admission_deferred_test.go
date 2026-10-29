package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	"prompt-manager/internal/paths"
	"prompt-manager/internal/requestpolicy"
	"prompt-manager/internal/store"
)

type admissionDeferredCallerVerifier struct{}

func (admissionDeferredCallerVerifier) Validate(_ context.Context, token string) (owneridentity.Identity, error) {
	return owneridentity.Identity{
		Subject:   token,
		Scopes:    []string{"agent-manager:write"},
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

func invokeAdmissionDeferredManualTrigger(h *Handlers, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/heartbeats/agent-1/trigger", nil)
	req.Header.Set("Authorization", token)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
	w := httptest.NewRecorder()
	h.TriggerHeartbeat(w, req)
	return w
}

func admissionDeferredLimitRefusal() error {
	return NewDispatchRejectedError(&AgentManagerHTTPError{StatusCode: 403, Message: "grant ceiling full", AdmissionReason: "grant_limit_reached"})
}

func admissionDeferredWaitExecution(t *testing.T, teamExecStore *TeamExecutionStore) {
	t.Helper()
	teamExecStore.Shutdown()
}

func admissionDeferredEntry(t *testing.T, teamExecStore *TeamExecutionStore, teamID, agentID string) runningEntry {
	t.Helper()
	teamExecStore.mu.RLock()
	tec := teamExecStore.contexts[teamID]
	teamExecStore.mu.RUnlock()
	if tec == nil {
		t.Fatalf("team execution context %s missing", teamID)
	}
	tec.mu.Lock()
	entry, ok := tec.running[agentID]
	if !ok || entry.State != ObligationAdmissionDeferred {
		tec.mu.Unlock()
		t.Fatalf("retained entry = %+v, present %t; want admission_deferred", entry, ok)
	}
	if entry.TaskRequestSnapshot == "" || entry.CreateRunRequestSnapshot == "" || entry.TaskID == "" || entry.IdempotencyKey == "" || !validCreateRunCallerMode(entry.CreateRunCallerMode) {
		tec.mu.Unlock()
		t.Fatalf("deferred entry lost frozen intent/caller binding: %+v", entry)
	}
	tec.mu.Unlock()
	status := tec.Status()
	if len(status.AdmissionDeferredAgentIDs) != 1 || status.AdmissionDeferredAgentIDs[0] != agentID || len(status.RunningAgentIDs) != 0 {
		t.Fatalf("deferred refusal must remain a held slot without claiming a running model: %+v", status)
	}
	if _, err := teamExecStore.Enqueue(context.Background(), teamID, agentID, entry.ProfileKey); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("deferred obligation must continue to block duplicate admission, got %v", err)
	}
	return entry
}

type deferredPersistenceFailingClient struct {
	*mockAgentClient
	store             *TeamExecutionStore
	blockedPath       string
	succeedAfterFirst bool
	calls             int
}

func (c *deferredPersistenceFailingClient) CreateTask(_ context.Context, task *Task) (*Task, error) {
	c.mockAgentClient.mu.Lock()
	c.mockAgentClient.createTaskCalls = append(c.mockAgentClient.createTaskCalls, task)
	c.calls++
	calls := c.calls
	c.mockAgentClient.mu.Unlock()
	if c.succeedAfterFirst && calls > 1 {
		created := *task
		created.Status = "queued"
		return &created, nil
	}
	if calls == 1 {
		c.store.mu.RLock()
		tec := c.store.contexts["team-1"]
		c.store.mu.RUnlock()
		if tec == nil {
			return nil, errors.New("team execution context missing")
		}
		tec.mu.Lock()
		tec.persistDir = c.blockedPath
		tec.mu.Unlock()
	}
	return nil, admissionDeferredLimitRefusal()
}

func TestAdmissionLimitDefersSameTaskAndRunIntentAcrossRestartAndEligibleCallbacks(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateTaskError(admissionDeferredLimitRefusal())
	firstExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	firstStore := NewTeamExecutionStore(teamStore, firstExecutor, dir, client)
	firstExecutor.SetTeamExecStore(firstStore)
	firstExecutor.OnRunComplete = firstStore.OnRunComplete
	firstScheduler := NewScheduler(firstExecutor, client, teamStore, firstStore)

	firstScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	admissionDeferredWaitExecution(t, firstStore)
	initial := admissionDeferredEntry(t, firstStore, "team-1", "agent-1")
	persistedBeforeRestart := readQueueFile(t, dir, "team-1")
	if len(persistedBeforeRestart.Running) != 1 || persistedBeforeRestart.Running[0].State != ObligationAdmissionDeferred {
		t.Fatalf("initial refusal was not durably deferred: %+v", persistedBeforeRestart.Running)
	}
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
		t.Fatalf("initial task refusal should make one task request and no run request: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	initialTask := *client.createTaskCalls[0]

	firstExecutor.Shutdown()
	// Recovery reloads the exact deferred owner obligation but does not perform
	// an ungated startup replay.
	secondExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	secondStore := NewTeamExecutionStore(teamStore, secondExecutor, dir, client)
	secondExecutor.SetTeamExecStore(secondStore)
	secondExecutor.OnRunComplete = secondStore.OnRunComplete
	secondStore.Recover(ctx)
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
		t.Fatalf("startup recovery must retain deferred admission without replay: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}

	// Existing eligibility gates run before the scheduled retry. A disabled
	// heartbeat does not replay the retained request.
	config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	config.Enabled = false
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}
	secondScheduler := NewScheduler(secondExecutor, client, teamStore, secondStore)
	secondScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
		t.Fatalf("disabled heartbeat replayed retained admission: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	config.Enabled = true
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
		t.Fatal(err)
	}

	// The exact task is now admitted, but the subsequent run is still denied.
	// Retain the confirmed task and frozen run request rather than canceling it.
	client.createTaskErr = nil
	client.createRunErr = admissionDeferredLimitRefusal()
	secondScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	admissionDeferredWaitExecution(t, secondStore)
	afterTaskRetry := admissionDeferredEntry(t, secondStore, "team-1", "agent-1")
	if afterTaskRetry.TaskID != initial.TaskID || afterTaskRetry.IdempotencyKey != initial.IdempotencyKey || afterTaskRetry.TaskRequestSnapshot != initial.TaskRequestSnapshot || afterTaskRetry.CreateRunRequestSnapshot != initial.CreateRunRequestSnapshot || !afterTaskRetry.TaskConfirmed {
		t.Fatalf("task retry changed the immutable intent: before=%+v after=%+v", initial, afterTaskRetry)
	}
	if len(client.createTaskCalls) != 2 || !reflect.DeepEqual(initialTask, *client.createTaskCalls[1]) || len(client.cancelTaskCalls) != 0 || len(client.deleteTaskCalls) != 0 || len(client.createRunCalls) != 1 {
		t.Fatalf("task-stage retry must use same task terms and retain confirmed task on run refusal: tasks=%d runs=%d cancel=%v delete=%v", len(client.createTaskCalls), len(client.createRunCalls), client.cancelTaskCalls, client.deleteTaskCalls)
	}
	initialRun := *client.createRunCalls[0]

	// Repeated limit refusal retries the same run, never creates another task,
	// key, or model execution.
	secondScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	admissionDeferredWaitExecution(t, secondStore)
	afterRepeatedLimit := admissionDeferredEntry(t, secondStore, "team-1", "agent-1")
	if afterRepeatedLimit.TaskID != initial.TaskID || afterRepeatedLimit.IdempotencyKey != initial.IdempotencyKey || afterRepeatedLimit.CreateRunRequestSnapshot != initial.CreateRunRequestSnapshot {
		t.Fatalf("repeated limit changed retained operation: %+v", afterRepeatedLimit)
	}
	if len(client.createTaskCalls) != 2 || len(client.createRunCalls) != 2 || !reflect.DeepEqual(initialRun, *client.createRunCalls[1]) || len(client.cancelTaskCalls) != 0 || len(client.deleteTaskCalls) != 0 {
		t.Fatalf("repeated limit must only replay the exact run request: tasks=%d runs=%d cancel=%v delete=%v", len(client.createTaskCalls), len(client.createRunCalls), client.cancelTaskCalls, client.deleteTaskCalls)
	}

	// A later eligible callback is the only retry clock. Success binds one run
	// and clears the deferred state without another task.
	client.createRunErr = nil
	client.waitRunErr = errors.New("observer intentionally unavailable for fixture")
	secondScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	secondStore.Shutdown()
	finalStatus := secondStore.Status("team-1")
	if len(client.createTaskCalls) != 2 || len(client.createRunCalls) != 3 || len(finalStatus.RunningAgentIDs) != 1 || len(finalStatus.UncertainAgentIDs) != 0 {
		t.Fatalf("successful scheduled retry did not converge on one run: tasks=%d runs=%d status=%+v", len(client.createTaskCalls), len(client.createRunCalls), finalStatus)
	}
	for i, request := range client.createRunCalls {
		if !reflect.DeepEqual(initialRun, *request) {
			t.Fatalf("run request %d changed immutable body: first=%+v got=%+v", i, initialRun, *request)
		}
	}
	secondExecutor.Shutdown()
}

func TestAdmissionDeferredDoesNotRetryWhenMemberRemoved(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relations := fileStore.Relations()
	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team One")); err != nil {
		t.Fatal(err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One"}); err != nil {
		t.Fatal(err)
	}
	if err := relations.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Enabled: true, Schedule: "0 * * * *"}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateTaskError(admissionDeferredLimitRefusal())
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnRunComplete = teamExecStore.OnRunComplete
	scheduler := NewScheduler(executor, client, teamStore, teamExecStore)
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	admissionDeferredWaitExecution(t, teamExecStore)
	if err := relations.DeleteTeamMember(ctx, "team-1", "agent-1"); err != nil {
		t.Fatal(err)
	}
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
		t.Fatalf("removed member's retained request was replayed: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	_ = admissionDeferredEntry(t, teamExecStore, "team-1", "agent-1")
	executor.Shutdown()
}

func TestAdmissionDeferredDoesNotRetryWhilePaused(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateTaskError(admissionDeferredLimitRefusal())
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnRunComplete = teamExecStore.OnRunComplete
	scheduler := NewScheduler(executor, client, teamStore, teamExecStore)
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	admissionDeferredWaitExecution(t, teamExecStore)

	control := newControlStoreForTest(t, time.Now())
	if _, err := control.Pause(ctx, "team-1", "fixture pause", operatorAttribution()); err != nil {
		t.Fatal(err)
	}
	scheduler.SetControlStore(control)
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
		t.Fatalf("paused heartbeat replayed retained admission: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	_ = admissionDeferredEntry(t, teamExecStore, "team-1", "agent-1")
	executor.Shutdown()
}

func setupAdmissionDeferredFailureEnvironment(t *testing.T) (*store.FileTeamStore, *store.FileAgentStore, store.RelationStore) {
	t.Helper()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relations := fileStore.Relations()
	ctx := context.Background()
	if err := teamStore.Create(ctx, newIndependentTestTeam("team-1", "Team One")); err != nil {
		t.Fatal(err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One"}); err != nil {
		t.Fatal(err)
	}
	if err := relations.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Enabled: true, Schedule: "0 * * * *"}); err != nil {
		t.Fatal(err)
	}
	return teamStore, agentStore, relations
}

func TestAdmissionDeferredPersistenceFailureRecoveryWaitsForCurrentEligibility(t *testing.T) {
	for _, gate := range []string{"disabled", "removed", "paused"} {
		t.Run(gate, func(t *testing.T) {
			teamStore, agentStore, relations := setupAdmissionDeferredFailureEnvironment(t)
			ctx := context.Background()
			dir := t.TempDir()
			blockedPath := filepath.Join(t.TempDir(), "not-a-directory")
			if err := os.WriteFile(blockedPath, []byte("block persistence"), 0o600); err != nil {
				t.Fatal(err)
			}
			client := &deferredPersistenceFailingClient{mockAgentClient: newMockAgentClient(), blockedPath: blockedPath}
			firstExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
			firstStore := NewTeamExecutionStore(teamStore, firstExecutor, dir, client)
			client.store = firstStore
			firstExecutor.SetTeamExecStore(firstStore)
			firstExecutor.OnRunComplete = firstStore.OnRunComplete
			firstScheduler := NewScheduler(firstExecutor, client, teamStore, firstStore)
			firstScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
			admissionDeferredWaitExecution(t, firstStore)

			status := firstStore.Status("team-1")
			if len(status.RunningAgentIDs) != 1 || len(status.UncertainAgentIDs) != 1 || len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
				t.Fatalf("failed deferred save must keep the preexisting durable intent held uncertain: status=%+v tasks=%d runs=%d", status, len(client.createTaskCalls), len(client.createRunCalls))
			}
			persisted := readQueueFile(t, dir, "team-1")
			if len(persisted.Running) != 1 || persisted.Running[0].State != ObligationDispatchUncertain || persisted.Running[0].AdmissionOrigin != AdmissionOriginScheduled || persisted.Running[0].TaskRequestSnapshot == "" || persisted.Running[0].CreateRunRequestSnapshot == "" {
				t.Fatalf("failed deferred save lost the original scheduled uncertain intent: %+v", persisted.Running)
			}

			control := (*HeartbeatControlStore)(nil)
			switch gate {
			case "disabled":
				config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
				if err != nil {
					t.Fatal(err)
				}
				config.Enabled = false
				if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
					t.Fatal(err)
				}
			case "removed":
				if err := relations.DeleteTeamMember(ctx, "team-1", "agent-1"); err != nil {
					t.Fatal(err)
				}
			case "paused":
				control = newControlStoreForTest(t, time.Now())
				if _, err := control.Pause(ctx, "team-1", "fixture pause", operatorAttribution()); err != nil {
					t.Fatal(err)
				}
			}

			secondExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
			secondStore := NewTeamExecutionStore(teamStore, secondExecutor, dir, client)
			client.store = secondStore
			secondExecutor.SetTeamExecStore(secondStore)
			secondExecutor.OnRunComplete = secondStore.OnRunComplete
			secondStore.Recover(ctx)
			if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
				t.Fatalf("restart replayed uncertain task before an eligible trigger: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
			}
			secondScheduler := NewScheduler(secondExecutor, client, teamStore, secondStore)
			if control != nil {
				secondScheduler.SetControlStore(control)
			}
			secondScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
			if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 || len(secondStore.Status("team-1").UncertainAgentIDs) != 1 {
				t.Fatalf("ineligible recovered intent was replayed or released: status=%+v tasks=%d runs=%d", secondStore.Status("team-1"), len(client.createTaskCalls), len(client.createRunCalls))
			}
			secondExecutor.Shutdown()
			firstExecutor.Shutdown()
		})
	}
}

func TestAdmissionDeferredPersistenceFailureRetriesExactIntentWhenEligible(t *testing.T) {
	teamStore, agentStore, _ := setupAdmissionDeferredFailureEnvironment(t)
	ctx := context.Background()
	dir := t.TempDir()
	blockedPath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedPath, []byte("block persistence"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &deferredPersistenceFailingClient{mockAgentClient: newMockAgentClient(), blockedPath: blockedPath, succeedAfterFirst: true}
	firstExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	firstStore := NewTeamExecutionStore(teamStore, firstExecutor, dir, client)
	client.store = firstStore
	firstExecutor.SetTeamExecStore(firstStore)
	firstExecutor.OnRunComplete = firstStore.OnRunComplete
	firstScheduler := NewScheduler(firstExecutor, client, teamStore, firstStore)
	firstScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	admissionDeferredWaitExecution(t, firstStore)
	initial := readQueueFile(t, dir, "team-1").Running[0]
	if initial.State != ObligationDispatchUncertain || initial.AdmissionOrigin != AdmissionOriginScheduled {
		t.Fatalf("initial failed persistence state = %+v", initial)
	}

	secondExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	secondStore := NewTeamExecutionStore(teamStore, secondExecutor, dir, client)
	client.store = secondStore
	secondExecutor.SetTeamExecStore(secondStore)
	secondExecutor.OnRunComplete = secondStore.OnRunComplete
	secondStore.Recover(ctx)
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 0 {
		t.Fatalf("startup replayed before a scheduled eligibility check: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	secondScheduler := NewScheduler(secondExecutor, client, teamStore, secondStore)
	secondScheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	secondStore.Shutdown()
	if len(client.createTaskCalls) != 2 || len(client.createRunCalls) != 1 {
		t.Fatalf("eligible callback did not retry exact intent once: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	if got := client.createTaskCalls[1].ID; got != initial.TaskID {
		t.Fatalf("retry task ID = %q, want retained %q", got, initial.TaskID)
	}
	secondExecutor.Shutdown()
	firstExecutor.Shutdown()
}

func TestLegacyDispatchWithoutTriggerOriginIsNeverPOSTReplayed(t *testing.T) {
	teamStore, agentStore, _ := setupAdmissionDeferredFailureEnvironment(t)
	ctx := context.Background()
	dir := t.TempDir()
	client := newMockAgentClient()
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnRunComplete = teamExecStore.OnRunComplete

	const taskID = "8a3f78d1-b406-4cf5-bc9d-047c900c0101"
	key := "prompt-manager-heartbeat:legacy:unknown-origin"
	taskSnapshot, err := freezeTaskDispatchRequest(&Task{ID: taskID, Title: "heartbeat", Description: "legacy intent", ScopePath: "/workspace"})
	if err != nil {
		t.Fatal(err)
	}
	entry := runningEntry{
		State: ObligationDispatchUncertain, DispatchStage: DispatchStageIntent,
		TaskID: taskID, TaskRequestSnapshot: taskSnapshot, IdempotencyKey: key,
		CreateRunRequestSnapshot: createRunSnapshotForTest(t, key, taskID, "profile-p", "legacy-heartbeat"),
		CreateRunCallerMode:      createRunCallerOriginUnqualified,
		// Old persisted records have no trigger origin. It must remain unknown.
	}
	owner := teamExecStore.GetOrCreate("team-1")
	owner.running["agent-1"] = entry
	if err := owner.persistLocked(); err != nil {
		t.Fatal(err)
	}
	teamExecStore.Recover(ctx)
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("legacy unknown-origin intent was POST-replayed during recovery: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	scheduler := NewScheduler(executor, client, teamStore, teamExecStore)
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("eligible cron inferred origin for a legacy intent: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	if len(teamExecStore.Status("team-1").UncertainAgentIDs) != 1 {
		t.Fatalf("legacy uncertain slot was not retained: %+v", teamExecStore.Status("team-1"))
	}
	executor.Shutdown()
}

func TestManualCapacityRetryBypassesWindowOnlyForSameFrozenCallerIntent(t *testing.T) {
	for _, serviceBacked := range []bool{false, true} {
		name := "caller"
		if serviceBacked {
			name = "service"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "")
			teamStore, agentStore, relations := setupAdmissionDeferredFailureEnvironment(t)
			ctx := context.Background()
			var taskID string
			var taskBodies []string
			var taskAuthorizations []string
			var runPosts int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
					w.WriteHeader(http.StatusNotFound)
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/tasks":
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("read task body: %v", err)
						http.Error(w, "bad request", http.StatusBadRequest)
						return
					}
					taskBodies = append(taskBodies, string(body))
					taskAuthorizations = append(taskAuthorizations, r.Header.Get("Authorization"))
					var requested CreateTaskRequest
					if err := json.Unmarshal(body, &requested); err != nil || requested.Task == nil {
						t.Errorf("decode frozen task request: task=%+v err=%v", requested.Task, err)
						http.Error(w, "bad request", http.StatusBadRequest)
						return
					}
					taskID = requested.Task.ID
					if len(taskBodies) == 1 {
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte(`{"message":"capacity full","admission_reason":"grant_limit_reached"}`))
						return
					}
					created := *requested.Task
					created.Status = "queued"
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					if err := json.NewEncoder(w).Encode(CreateTaskResponse{Task: &created}); err != nil {
						t.Errorf("encode queued task response: %v", err)
					}
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs":
					runPosts++
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					if err := json.NewEncoder(w).Encode(CreateRunResponse{Run: &Run{ID: "run-manual-retry", TaskID: taskID, Status: "RUN_STATUS_RUNNING"}}); err != nil {
						t.Errorf("encode run response: %v", err)
					}
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs/run-manual-retry":
					w.Header().Set("Content-Type", "application/json")
					if err := json.NewEncoder(w).Encode(GetRunResponse{Run: &Run{ID: "run-manual-retry", TaskID: taskID, Status: "RUN_STATUS_COMPLETE"}}); err != nil {
						t.Errorf("encode terminal run response: %v", err)
					}
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			client := &AgentManagerClient{
				createCallerVerifier: admissionDeferredCallerVerifier{},
				httpClient:           server.Client(),
				testBaseURL:          server.URL,
			}
			if serviceBacked {
				client.serviceToken = &rotatingCreateRunServiceToken{}
			}
			token := "Bearer fixture-owner"
			if serviceBacked {
				token = ""
			}
			invoke := func(h *Handlers, token string, allow bool) *httptest.ResponseRecorder {
				if !serviceBacked {
					return invokeAdmissionDeferredManualTrigger(h, token)
				}
				grant := &identity.OperationGrant{ID: "fixture-dispatch", Policy: "run-creation", Approver: "owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}
				if !allow {
					grant.Targets = []string{"agent-manager:run/other"}
				}
				req := httptest.NewRequest(http.MethodPost, heartbeatconnect.HeartbeatServiceTriggerHeartbeatProcedure, nil)
				req = mux.SetURLVars(req, map[string]string{"id": "team-1", "agentId": "agent-1"})
				req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: identity.ActorAgent, Subject: "fixture-dispatcher", Owner: "owner", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: grant}))
				rec := httptest.NewRecorder()
				requestpolicy.PreparationBoundary(http.HandlerFunc(h.TriggerHeartbeat)).ServeHTTP(rec, req)
				return rec
			}
			dir := t.TempDir()
			executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
			teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
			executor.SetTeamExecStore(teamExecStore)
			executor.OnRunComplete = teamExecStore.OnRunComplete
			handlers := NewHandlers(HandlersDeps{
				TeamStore: teamStore, AgentStore: agentStore, RelationStore: relations,
				Executor: executor, AgentClient: client, TeamExecStore: teamExecStore,
				RunCallerValidator: admissionDeferredCallerVerifier{},
			})
			first := invoke(handlers, token, true)
			if first.Code != http.StatusAccepted {
				t.Fatalf("initial manual request: status=%d body=%s", first.Code, first.Body.String())
			}
			teamExecStore.Shutdown()
			if len(taskBodies) != 1 || runPosts != 0 || len(teamExecStore.Status("team-1").AdmissionDeferredAgentIDs) != 1 {
				t.Fatalf("initial refusal did not retain one manual intent: tasks=%d runs=%d status=%+v", len(taskBodies), runPosts, teamExecStore.Status("team-1"))
			}
			executor.Shutdown()

			// Restart reloads the exact manual origin and caller binding without a
			// credential. A newly qualified request from the same caller can retry it.
			restartedExecutor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
			restartedStore := NewTeamExecutionStore(teamStore, restartedExecutor, dir, client)
			restartedExecutor.SetTeamExecStore(restartedStore)
			restartedExecutor.OnRunComplete = restartedStore.OnRunComplete
			restartedStore.Recover(ctx)
			if len(taskBodies) != 1 || runPosts != 0 || len(restartedStore.Status("team-1").AdmissionDeferredAgentIDs) != 1 {
				t.Fatalf("restart must retain the manual intent without posting: tasks=%d runs=%d status=%+v", len(taskBodies), runPosts, restartedStore.Status("team-1"))
			}
			admissionDeferredEntry(t, restartedStore, "team-1", "agent-1")
			handlers = NewHandlers(HandlersDeps{
				TeamStore: teamStore, AgentStore: agentStore, RelationStore: relations,
				Executor: restartedExecutor, AgentClient: client, TeamExecStore: restartedStore,
				RunCallerValidator: admissionDeferredCallerVerifier{},
			})
			other := invoke(handlers, "Bearer other-owner", false)
			wantDenied := http.StatusConflict
			if serviceBacked {
				wantDenied = http.StatusForbidden
			}
			if other.Code != wantDenied {
				t.Fatalf("different caller must not retry the retained manual request: status=%d body=%s", other.Code, other.Body.String())
			}
			if len(taskBodies) != 1 || runPosts != 0 {
				t.Fatalf("different caller caused an owner write: tasks=%d runs=%d", len(taskBodies), runPosts)
			}
			// The completed-trigger window is active, but an explicit repeat retries
			// this exact manual intent before that dedupe check.
			second := invoke(handlers, token, true)
			if second.Code != http.StatusAccepted {
				t.Fatalf("same-caller manual retry: status=%d body=%s", second.Code, second.Body.String())
			}
			restartedStore.Shutdown()
			if len(taskBodies) != 2 || taskBodies[1] != taskBodies[0] || runPosts != 1 {
				t.Fatalf("manual retry changed task terms or did not create the single run: taskBodies=%v runPosts=%d", taskBodies, runPosts)
			}
			if serviceBacked {
				if len(taskAuthorizations) != 2 || !strings.HasPrefix(taskAuthorizations[0], "Bearer service-token-") || !strings.HasPrefix(taskAuthorizations[1], "Bearer service-token-") || taskAuthorizations[0] == taskAuthorizations[1] {
					t.Fatalf("retry did not preserve fresh service authority: %v", taskAuthorizations)
				}
			}
			restartedExecutor.Shutdown()
		})
	}
}

func TestAdmissionDeferredPreservesPendingApprovalContract(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	ctx := context.Background()
	dir := t.TempDir()
	client := newMockAgentClient().
		WithCreateTaskResponse(&Task{ID: "task-approved", Title: "heartbeat"}).
		WithCreateRunError(NewApprovalPendingError("creating_run", nil))
	executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
	teamExecStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	executor.SetTeamExecStore(teamExecStore)
	executor.OnRunComplete = teamExecStore.OnRunComplete
	scheduler := NewScheduler(executor, client, teamStore, teamExecStore)
	scheduler.executeHeartbeat(ctx, "team-1", "agent-1")
	admissionDeferredWaitExecution(t, teamExecStore)
	status := teamExecStore.Status("team-1")
	if len(status.ApprovalPendingAgentIDs) != 1 || len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 1 || len(client.cancelTaskCalls) != 0 || len(client.deleteTaskCalls) != 0 {
		t.Fatalf("approval-pending semantics changed by capacity deferral: status=%+v tasks=%d runs=%d cancel=%v delete=%v", status, len(client.createTaskCalls), len(client.createRunCalls), client.cancelTaskCalls, client.deleteTaskCalls)
	}
	executor.Shutdown()
}

func TestAdmissionDeferredServiceReplayRefreshesCredentialAcrossRestart(t *testing.T) {
	tokens := &rotatingCreateRunServiceToken{}
	var bodies []string
	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs" {
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read run request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		bodies = append(bodies, string(body))
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"run ceiling full","admission_reason":"grant_limit_reached"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"run":{"id":"run-service-retry","task_id":"8a3f78d1-b406-4cf5-bc9d-047c900c0101","status":"RUN_STATUS_RUNNING"}}`))
	}))
	defer server.Close()
	client := &AgentManagerClient{serviceToken: tokens, httpClient: server.Client(), testBaseURL: server.URL}
	dir := t.TempDir()
	const taskID = "8a3f78d1-b406-4cf5-bc9d-047c900c0101"
	taskSnapshot, err := freezeTaskDispatchRequest(&Task{ID: taskID, Title: "heartbeat", Description: "frozen", ScopePath: "/workspace"})
	if err != nil {
		t.Fatal(err)
	}
	key := "prompt-manager-heartbeat:team-x:agent-1:capacity-retry"
	runSnapshot := createRunSnapshotForTest(t, key, taskID, "profile-p", "heartbeat-team-x-agent-1")
	entry := runningEntry{
		ProfileKey: "profile-p", State: ObligationAdmissionDeferred, DispatchStage: DispatchStageIntent,
		AdmissionOrigin: AdmissionOriginScheduled,
		TaskID:          taskID, TaskRequestSnapshot: taskSnapshot, TaskConfirmed: true,
		IdempotencyKey: key, CreateRunRequestSnapshot: runSnapshot, CreateRunCallerMode: createRunCallerOriginService,
	}
	first := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	first.running["agent-1"] = entry
	if err := first.persistLocked(); err != nil {
		t.Fatal(err)
	}
	if err := first.ReconcileDispatch(context.Background(), "agent-1"); err == nil || admissionReasonFromError(err) != "grant_limit_reached" {
		t.Fatalf("first service retry error = %v, want typed grant-limit refusal", err)
	}
	first.mu.Lock()
	retained := first.running["agent-1"]
	first.mu.Unlock()
	if retained.State != ObligationAdmissionDeferred || retained.CreateRunCallerMode != createRunCallerOriginService || retained.CreateRunCallerBinding != "" {
		t.Fatalf("service replay changed caller mode or did not remain deferred: %+v", retained)
	}
	first.Shutdown()

	queueBytes, err := os.ReadFile(first.queueFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(queueBytes), "service-token") {
		t.Fatal("deferred service replay persisted a bearer credential")
	}
	// Restart loading does not replay before an eligible scheduler callback.
	restarted := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	restarted.Recover(context.Background())
	if len(bodies) != 1 || tokens.calls != 1 {
		t.Fatalf("restart recovery sent an un-gated request: bodies=%d token calls=%d", len(bodies), tokens.calls)
	}
	if retried, err := restarted.RetryAdmissionDeferred(withHeartbeatAdmissionOrigin(context.Background(), AdmissionOriginScheduled), "agent-1"); err != nil || !retried {
		t.Fatalf("eligible service replay: retried=%t err=%v", retried, err)
	}
	if len(bodies) != 2 || bodies[1] != bodies[0] || tokens.calls != 2 || len(authorizations) != 2 || authorizations[0] != "Bearer service-token-1" || authorizations[1] != "Bearer service-token-2" {
		t.Fatalf("replay must use identical frozen body with a fresh service proof: bodies=%v auth=%v token calls=%d", bodies, authorizations, tokens.calls)
	}
	if strings.Contains(bodies[1], "service-token") {
		t.Fatal("service token leaked into frozen run body")
	}
	restarted.Shutdown()
}
