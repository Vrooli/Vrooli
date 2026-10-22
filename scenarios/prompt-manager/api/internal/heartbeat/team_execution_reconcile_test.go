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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"prompt-manager/internal/store"

	"github.com/google/uuid"

	"github.com/vrooli/api-core/owneridentity"
)

func createRunSnapshotForTest(t *testing.T, key, taskID, profileKey, tag string) string {
	t.Helper()
	parent := "parent-original"
	request := CreateRunRequest{
		ParentRunID:     &parent,
		RequestedScopes: &RunIdentityScopeRequest{Scopes: []string{"agent-manager:read"}, ExpectedOwnerSubject: "owner-original"},
		IdempotencyKey:  key,
		TaskID:          taskID,
		ProfileRef:      &ProfileRef{ProfileKey: profileKey},
		Tag:             &tag,
		RunMode:         "interactive",
		Environment: map[string]string{
			"VROOLI_PROMPT_MANAGER_ATTRIBUTION": "heartbeat;team=team-x;member=agent-1",
			workloadKindEnv:                     "scheduled",
			workloadKeyEnv:                      "effort:original",
			workloadInstanceEnv:                 "attempt-original",
			"VROOLI_SNAPSHOT_SENTINEL":          "original-environment",
		},
	}
	encoded, err := json.Marshal(&request)
	if err != nil {
		t.Fatalf("marshal CreateRun request snapshot: %v", err)
	}
	return string(encoded)
}

type rotatingCreateRunServiceToken struct{ calls int }

func (s *rotatingCreateRunServiceToken) Token(context.Context) (string, error) {
	s.calls++
	return "service-token-" + strconv.Itoa(s.calls), nil
}

func TestServiceCallerReplayRefreshesProofWithoutChangingRequestBody(t *testing.T) {
	tokens := &rotatingCreateRunServiceToken{}
	var bodies []string
	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bodies = append(bodies, string(body))
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"run":{"id":"run-original","task_id":"task-1","status":"RUN_STATUS_RUNNING"}}`))
	}))
	defer server.Close()
	client := &AgentManagerClient{serviceToken: tokens, httpClient: server.Client(), testBaseURL: server.URL}
	request := CreateRunRequest{
		IdempotencyKey: "key-snapshot",
		TaskID:         "task-1",
		ProfileRef:     &ProfileRef{ProfileKey: "resolved-profile"},
	}
	firstBody, err := json.Marshal(&request)
	if err != nil {
		t.Fatalf("marshal initial CreateRun body: %v", err)
	}
	initialCtx, err := client.prepareCreateRunCaller(context.Background())
	if err != nil || createRunCaller(initialCtx).origin != createRunCallerOriginService {
		t.Fatalf("initial service proof qualification: origin=%q err=%v", createRunCaller(initialCtx).origin, err)
	}
	if _, err := client.CreateRun(initialCtx, &request); err != nil {
		t.Fatalf("initial CreateRun: %v", err)
	}
	replayCtx, err := prepareReplayCaller(context.Background(), client, createRunCallerOriginService, "")
	if err != nil {
		t.Fatalf("refresh service proof for replay: %v", err)
	}
	if _, err := client.CreateRun(replayCtx, &request); err != nil {
		t.Fatalf("replayed CreateRun: %v", err)
	}
	if tokens.calls != 2 || len(bodies) != 2 || len(authorizations) != 2 {
		t.Fatalf("expected two fresh proofs and requests, calls=%d bodies=%d auth=%d", tokens.calls, len(bodies), len(authorizations))
	}
	if bodies[0] != string(firstBody) || bodies[1] != bodies[0] {
		t.Fatalf("service token refresh changed the immutable CreateRun body: first=%s second=%s", bodies[0], bodies[1])
	}
	if authorizations[0] != "Bearer service-token-1" || authorizations[1] != "Bearer service-token-2" {
		t.Fatalf("expected rotated service proof per send, got %v", authorizations)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := prepareReplayCaller(canceled, client, createRunCallerOriginService, ""); err == nil {
		t.Fatal("service replay must honor reconciliation cancellation")
	}
	if tokens.calls != 2 {
		t.Fatalf("canceled service replay must not refresh credentials, token calls=%d", tokens.calls)
	}
}

type replayCallerVerifier struct{ subject string }

func (v replayCallerVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: v.subject, Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func TestReplayCallerModeBindsSameVerifiedSubjectAndDoesNotSubstituteAuthority(t *testing.T) {
	tokens := &rotatingCreateRunServiceToken{}
	client := &AgentManagerClient{serviceToken: tokens, createCallerVerifier: replayCallerVerifier{subject: "human-a"}}
	if _, err := prepareReplayCaller(context.Background(), client, createRunCallerOriginCaller, "v1:binding-a"); err == nil {
		t.Fatal("caller-origin replay must remain held when original caller proof is unavailable")
	}
	callerCtx := withCreateRunCaller(context.Background(), "Bearer current-human-proof", "")
	qualified, err := client.prepareCreateRunCaller(callerCtx)
	if err != nil {
		t.Fatalf("qualify current human: %v", err)
	}
	binding := createRunCaller(qualified).binding
	if binding == "" {
		t.Fatal("verified human must have a stable non-secret caller binding")
	}
	if _, err := prepareReplayCaller(callerCtx, client, createRunCallerOriginCaller, binding); err != nil {
		t.Fatalf("same verified human should be allowed to replay: %v", err)
	}
	client.createCallerVerifier = replayCallerVerifier{subject: "human-b"}
	if _, err := prepareReplayCaller(callerCtx, client, createRunCallerOriginCaller, binding); err == nil {
		t.Fatal("different verified human must not take over an uncertain caller-owned dispatch")
	}
	if _, err := prepareReplayCaller(callerCtx, client, createRunCallerOriginCaller, ""); err == nil {
		t.Fatal("legacy caller-mode intent without a verified subject binding must remain held")
	}
	serviceContext := withCreateRunCallerOrigin(context.Background(), "Bearer service-token", "", createRunCallerOriginService)
	if _, err := prepareReplayCaller(serviceContext, client, createRunCallerOriginCaller, binding); err == nil {
		t.Fatal("service-origin context must not be reused to replay caller-owned work")
	}
	if tokens.calls != 0 {
		t.Fatalf("caller-origin replay must not substitute service authority, token calls=%d", tokens.calls)
	}
	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
	if _, err := prepareReplayCaller(context.Background(), client, createRunCallerOriginUnqualified, ""); err == nil {
		t.Fatal("unqualified legacy replay must remain held when owner policy requires a caller")
	}
	if tokens.calls != 0 {
		t.Fatalf("unqualified replay must not silently mint service authority, token calls=%d", tokens.calls)
	}
}

// writeQueueFile writes a persistedTeamQueue to disk for Recover tests.
func writeQueueFile(t *testing.T, dir, teamID string, q persistedTeamQueue) {
	t.Helper()
	path := filepath.Join(dir, "team-queue-"+teamID+".json")
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		t.Fatalf("marshal queue: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write queue: %v", err)
	}
}

func readQueueFile(t *testing.T, dir, teamID string) persistedTeamQueue {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "team-queue-"+teamID+".json"))
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	var q persistedTeamQueue
	if err := json.Unmarshal(data, &q); err != nil {
		t.Fatalf("unmarshal queue: %v", err)
	}
	return q
}

func TestRecover_DropsRunningWithTerminalRunID(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-terminal"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-terminal", &Run{ID: "run-terminal", Status: "RUN_STATUS_COMPLETE"})

	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 0 {
		t.Fatalf("expected terminal entry dropped, got running=%v", status.RunningAgentIDs)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 0 {
		t.Fatalf("expected disk-running cleared, got %v", persisted.Running)
	}
}

func TestRecover_RetainsUncertainRunningWithEmptyRunID(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: ""},
		},
	})

	client := newMockAgentClient()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected uncertain obligation retained, got running=%v", status.RunningAgentIDs)
	}
	if len(status.UncertainAgentIDs) != 1 || status.UncertainAgentIDs[0] != "agent-1" {
		t.Fatalf("expected uncertain classification, got %v", status.UncertainAgentIDs)
	}
	if len(client.getRunCalls) != 0 {
		t.Fatalf("expected no GetRun calls for empty RunID, got %v", client.getRunCalls)
	}
}

func TestRecover_RetainsUncertainRunningWhenOwnerReportsMissing(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-gone"},
		},
	})

	// (nil, nil) — owner reached, run not found. Absence of terminal evidence is
	// not terminal evidence, so the obligation is retained as uncertain.
	client := newMockAgentClient().WithGetRunResponse("run-gone", nil)
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected missing-run obligation retained, got %v", status.RunningAgentIDs)
	}
	if len(status.UncertainAgentIDs) != 1 {
		t.Fatalf("expected uncertain classification, got %v", status.UncertainAgentIDs)
	}
}

func TestRecover_PreservesRunningWithActiveRun(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-active", AdmissionOrigin: AdmissionOriginManual},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-active", &Run{ID: "run-active", Status: "RUN_STATUS_RUNNING", StartedAt: "2026-10-19T15:07:22Z"})

	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected active entry preserved, got %v", status.RunningAgentIDs)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-active" {
		t.Fatalf("expected RunID preserved on disk, got %+v", persisted.Running)
	}
	if len(exec.watched) != 1 || exec.watched[0].runID != "run-active" {
		t.Fatalf("expected recovered known run completion watcher, got %+v", exec.watched)
	}
	if exec.watched[0].ownerStartedAt != "2026-10-19T15:07:22Z" || exec.watched[0].ownerTerminalAt != "" || exec.watched[0].admissionOrigin != AdmissionOriginManual {
		t.Fatalf("recovered watcher lost owner projection fields: %+v", exec.watched[0])
	}
}

func TestScheduledRetryApprovalPendingRunReplaysSameIntent(t *testing.T) {
	client := newMockAgentClient().WithCreateRunError(NewApprovalPendingError("creating_run", errors.New("operator approval requested")))
	teamExecStore := NewTeamExecutionStore(nil, &captureExecutor{}, t.TempDir(), client)
	tec := teamExecStore.GetOrCreate("team-x")
	tec.mu.Lock()
	tec.running["agent-1"] = runningEntry{
		ProfileKey:               "profile-p",
		State:                    ObligationApprovalPending,
		TaskID:                   "task-1",
		RunTag:                   "heartbeat-team-x-agent-1",
		IdempotencyKey:           "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
		CreateRunRequestSnapshot: createRunSnapshotForTest(t, "prompt-manager-heartbeat:team-x:agent-1:attempt-1", "task-1", "profile-p", "heartbeat-team-x-agent-1"),
		CreateRunCallerMode:      createRunCallerOriginUnqualified,
	}
	_ = tec.persistLocked()
	tec.mu.Unlock()
	configStore := &stubConfigStore{config: &store.HeartbeatConfig{TeamID: "team-x", AgentID: "agent-1", Enabled: true, Schedule: "0 * * * *"}}
	scheduler := NewScheduler(&captureExecutor{}, client, configStore, teamExecStore)

	scheduler.executeHeartbeat(context.Background(), "team-x", "agent-1")
	if got := teamExecStore.Status("team-x"); len(got.ApprovalPendingAgentIDs) != 1 {
		t.Fatalf("first scheduled retry must retain typed pending state: %+v", got)
	}
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 1 {
		t.Fatalf("first retry must only replay the persisted run intent; tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}

	client.createRunErr = nil
	client.createRunResp = &Run{ID: "run-approved", TaskID: "task-1", Status: "RUN_STATUS_RUNNING"}
	scheduler.executeHeartbeat(context.Background(), "team-x", "agent-1")
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 2 {
		t.Fatalf("approval retry must reuse task and run operation; tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
	for _, req := range client.createRunCalls {
		if req.TaskID != "task-1" || req.IdempotencyKey != "prompt-manager-heartbeat:team-x:agent-1:attempt-1" {
			t.Fatalf("retry changed run operation identity: %+v", req)
		}
	}
	status := teamExecStore.Status("team-x")
	if len(status.RunningAgentIDs) != 1 || len(status.ApprovalPendingAgentIDs) != 0 || len(status.UncertainAgentIDs) != 0 {
		t.Fatalf("accepted retry should bind one known run and clear pending/uncertain state: %+v", status)
	}
}

type taskApprovalSequenceExecutor struct {
	calls int
}

func (e *taskApprovalSequenceExecutor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	e.calls++
	if e.calls == 1 {
		return nil, NewApprovalPendingError("creating_task", errors.New("operator approval requested"))
	}
	return &ExecutionResult{TeamID: teamID, AgentID: agentID, Status: "running", RunID: "run-after-approval"}, nil
}

func TestScheduledTaskApprovalPendingReleasesSlotForNextSchedule(t *testing.T) {
	exec := &taskApprovalSequenceExecutor{}
	teamExecStore := NewTeamExecutionStore(nil, exec, t.TempDir(), nil)
	configStore := &stubConfigStore{config: &store.HeartbeatConfig{TeamID: "team-x", AgentID: "agent-1", Enabled: true, Schedule: "0 * * * *"}}
	scheduler := NewScheduler(exec, nil, configStore, teamExecStore)

	scheduler.executeHeartbeat(context.Background(), "team-x", "agent-1")
	teamExecStore.Shutdown()
	if got := teamExecStore.Status("team-x"); len(got.RunningAgentIDs) != 0 {
		t.Fatalf("task-stage approval pending guarantees no task/run effect and must release slot: %+v", got)
	}
	scheduler.executeHeartbeat(context.Background(), "team-x", "agent-1")
	teamExecStore.Shutdown()
	if exec.calls != 2 {
		t.Fatalf("next scheduled retry should be admitted after task-stage approval, Execute calls=%d", exec.calls)
	}
}

func TestRecoverNeedsReviewRetainsSlotAndReportsReview(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{TeamID: "team-x", MaxConcurrentRuns: 1, Running: []queuedExecution{{AgentID: "agent-1", ProfileKey: "p", RunID: "run-review"}}})
	client := newMockAgentClient().WithGetRunResponse("run-review", &Run{ID: "run-review", Status: "needs_review"})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)
	tec.Recover(context.Background())
	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || len(status.NeedsReviewAgentIDs) != 1 || len(exec.watched) != 1 {
		t.Fatalf("review run must remain owned, visible, and watched: status=%+v watchers=%+v", status, exec.watched)
	}
	if _, err := tec.Enqueue(context.Background(), "agent-1", "p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("review run must refuse duplicate launch, got %v", err)
	}
}

func TestRecover_RetainsOwnerUnreachableOnAgentManagerError(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-err"},
		},
	})

	client := newMockAgentClient()
	client.getRunErr = errors.New("connection refused")
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected owner-unreachable obligation retained, got %v", status.RunningAgentIDs)
	}
	if len(status.UnreachableAgentIDs) != 1 || status.UnreachableAgentIDs[0] != "agent-1" {
		t.Fatalf("expected owner_unreachable classification, got %v", status.UnreachableAgentIDs)
	}

	// The retained obligation must fence a duplicate heartbeat during the outage.
	if _, err := tec.Enqueue(context.Background(), "agent-1", "p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected MemberAlreadyQueuedError while owner unreachable, got %v", err)
	}
}

type budgetBlockingRunClient struct {
	*mockAgentClient
	mu        sync.Mutex
	blockNext bool
}

func (c *budgetBlockingRunClient) GetRun(ctx context.Context, runID string) (*Run, error) {
	c.mu.Lock()
	block := c.blockNext
	c.blockNext = false
	c.mu.Unlock()
	if block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return c.mockAgentClient.GetRun(ctx, runID)
}

func TestRecoverOwnerReadDeadlineRetainsSlotAndScheduledRetryRecovers(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running:           []queuedExecution{{AgentID: "agent-1", ProfileKey: "p", RunID: "run-retained"}},
	})
	client := &budgetBlockingRunClient{
		mockAgentClient: newMockAgentClient().WithGetRunResponse("run-retained", &Run{ID: "run-retained", Status: "RUN_STATUS_RUNNING"}),
		blockNext:       true,
	}
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)

	recoveryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	started := time.Now()
	tec.Recover(recoveryCtx)
	cancel()
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("owner read did not stop at its recovery deadline: %s", elapsed)
	}
	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || len(status.UnreachableAgentIDs) != 1 || status.UnreachableAgentIDs[0] != "agent-1" {
		t.Fatalf("deadline must retain the owned slot as owner-unreachable: %+v", status)
	}

	handled, err := tec.RetryUnavailableWatcher(context.Background(), "agent-1")
	if err != nil || !handled {
		t.Fatalf("scheduled retry should reconcile retained owner state, handled=%v err=%v", handled, err)
	}
	status = tec.Status()
	if len(status.RunningAgentIDs) != 1 || len(status.UnreachableAgentIDs) != 0 || len(exec.watched) != 1 {
		t.Fatalf("successful retry should keep the active slot and attach one watcher: status=%+v watchers=%+v", status, exec.watched)
	}
}

func TestRecover_DropsStaleQueuedTicks(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-active"},
		},
		Queue: []queuedExecution{
			{AgentID: "agent-2", ProfileKey: "p"},
			{AgentID: "agent-3", ProfileKey: "p"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-active", &Run{ID: "run-active", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.Queue) != 0 {
		t.Fatalf("expected stale queued ticks dropped, got %v", status.Queue)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Queue) != 0 {
		t.Fatalf("expected stale queue cleared on disk, got %v", persisted.Queue)
	}
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected active obligation retained, got %v", status.RunningAgentIDs)
	}
}

func TestRecover_PreservesPausedParkedObligation(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-parked"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-parked", &Run{ID: "run-parked", Status: "RUN_STATUS_PARKED"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.PausedAgentIDs) != 1 || status.PausedAgentIDs[0] != "agent-1" {
		t.Fatalf("expected parked run classified paused, got %v", status.PausedAgentIDs)
	}
	if len(status.RunningAgentIDs) != 1 {
		t.Fatalf("expected parked obligation retained, got %v", status.RunningAgentIDs)
	}
}

func TestRetryPausedObligationReattachesWatcherWhenOwnerResumes(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running:           []queuedExecution{{AgentID: "agent-1", ProfileKey: "p", RunID: "run-parked"}},
	})

	client := newMockAgentClient().WithGetRunResponse("run-parked", &Run{ID: "run-parked", Status: "RUN_STATUS_PARKED"})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)
	tec.Recover(context.Background())
	if status := tec.Status(); len(status.PausedAgentIDs) != 1 {
		t.Fatalf("expected parked owner to remain paused after recovery: %+v", status)
	}

	client.mu.Lock()
	client.getRuns["run-parked"] = &Run{ID: "run-parked", Status: "RUN_STATUS_RUNNING"}
	client.mu.Unlock()
	handled, err := tec.RetryUnavailableWatcher(context.Background(), "agent-1")
	if err != nil || !handled {
		t.Fatalf("scheduled owner recheck should handle a paused owner, handled=%t err=%v", handled, err)
	}
	status := tec.Status()
	if len(status.PausedAgentIDs) != 0 || len(status.RunningAgentIDs) != 1 {
		client.mu.Lock()
		calls := append([]string(nil), client.getRunCalls...)
		observed := client.getRuns["run-parked"]
		client.mu.Unlock()
		t.Fatalf("resumed owner should keep its original queue slot and leave paused state: status=%+v calls=%v owner=%+v", status, calls, observed)
	}
	if len(exec.watched) != 1 || exec.watched[0].runID != "run-parked" || exec.watched[0].ownerStatus != "RUN_STATUS_RUNNING" {
		t.Fatalf("resumed owner should reattach exactly one watcher to its original run: %+v", exec.watched)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.getRunCalls) != 2 {
		t.Fatalf("expected one startup observation and one scheduled owner recheck, got calls %v", client.getRunCalls)
	}
}

func TestRetryPausedObligationRejectsMismatchedOwnerRunIdentity(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithGetRunResponse("run-expected", &Run{ID: "run-other", Status: "RUN_STATUS_COMPLETE"})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)
	tec.mu.Lock()
	tec.running["agent-1"] = runningEntry{ProfileKey: "p", RunID: "run-expected", State: ObligationPaused}
	if err := tec.persistLocked(); err != nil {
		t.Fatal(err)
	}
	tec.mu.Unlock()

	handled, err := tec.RetryUnavailableWatcher(context.Background(), "agent-1")
	if !handled || err == nil {
		t.Fatalf("mismatched owner response should remain handled but unresolved, handled=%t err=%v", handled, err)
	}
	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || len(status.PausedAgentIDs) != 1 || len(exec.watched) != 0 {
		t.Fatalf("mismatched terminal response must not project or release the retained owner: status=%+v watchers=%+v", status, exec.watched)
	}
}

func TestRecover_TerminalReleaseReopensCapacity(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-done"},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-done", &Run{ID: "run-done", Status: "RUN_STATUS_COMPLETE"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected terminal obligation released, got %v", tec.Status().RunningAgentIDs)
	}
	// Proven terminal evidence releases the slot for the next scheduled tick.
	result, err := tec.Enqueue(context.Background(), "agent-2", "p")
	if err != nil {
		t.Fatalf("expected capacity reopened after terminal release, got %v", err)
	}
	if result.Status != "started" {
		t.Fatalf("expected agent-2 to start, got %q", result.Status)
	}
}

func TestRecover_PreservesRecordedObligationWithoutReprobing(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", RunID: "run-x", State: ObligationOwnerUnreachable},
		},
	})

	client := newMockAgentClient().
		WithGetRunResponse("run-x", &Run{ID: "run-x", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	status := tec.Status()
	if len(status.UnreachableAgentIDs) != 1 {
		t.Fatalf("expected recorded obligation preserved, got %+v", status)
	}
	if len(client.getRunCalls) != 0 {
		t.Fatalf("expected no owner probe for a recorded obligation, got %v", client.getRunCalls)
	}
}

func TestSetRunningRunID_UpdatesEntry(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	tec.SetRunningRunID("agent-1", "run-123")

	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-123" {
		t.Fatalf("expected RunID persisted, got %+v", persisted.Running)
	}
}

func TestCancelUnsentTaskDispatchSaveFailureRetainsExactIntent(t *testing.T) {
	dir := t.TempDir()
	key := "prompt-manager-heartbeat:team-x:agent-1:cancel-save"
	snapshot, err := freezeTaskDispatchRequest(&Task{
		ID: uuid.NewString(), Title: "heartbeat", Description: "frozen",
		ScopePath: "/workspace", Status: "queued",
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := parseTaskDispatchRequest(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	tec.running["agent-1"] = runningEntry{
		State: ObligationDispatchUncertain, IdempotencyKey: key,
		TaskID: task.ID, TaskRequestSnapshot: snapshot,
	}
	if err := tec.persistLocked(); err != nil {
		t.Fatal(err)
	}
	queuePath := tec.queueFilePath()
	if err := os.Remove(queuePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(queuePath, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := tec.CancelUnsentTaskDispatch("agent-1", task.ID, key, snapshot); err == nil {
		t.Fatal("cancellation must fail when its durable tombstone cannot be saved")
	}
	entry, ok := tec.running["agent-1"]
	if !ok || entry.TaskID != task.ID || entry.IdempotencyKey != key || entry.TaskRequestSnapshot != snapshot || entry.State != ObligationDispatchUncertain {
		t.Fatalf("failed cancellation must restore the exact held intent, got %+v (present=%v)", entry, ok)
	}
}

func TestReconcileTaskDispatchCancellationRetainsPriorUnknownPost(t *testing.T) {
	dir := t.TempDir()
	task := dispatchTaskFixture()
	taskSnapshot, err := freezeTaskDispatchRequest(task)
	if err != nil {
		t.Fatal(err)
	}
	key := "prompt-manager-heartbeat:team-x:agent-1:prior-unknown"
	runSnapshot := createRunSnapshotForTest(t, key, task.ID, "profile-p", "heartbeat-team-x-agent-1")
	ctx, cancel := context.WithCancel(context.Background())
	client := &cancelingTaskPreflightClient{mockAgentClient: newMockAgentClient(), cancel: cancel}
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.running["agent-1"] = runningEntry{
		ProfileKey: "profile-p", State: ObligationDispatchUncertain,
		IdempotencyKey: key, TaskID: task.ID, TaskRequestSnapshot: taskSnapshot,
		CreateRunRequestSnapshot: runSnapshot, CreateRunCallerMode: createRunCallerOriginUnqualified,
	}
	if err := tec.persistLocked(); err != nil {
		t.Fatal(err)
	}

	if err := tec.ReconcileTaskDispatch(ctx, "agent-1"); err == nil {
		t.Fatal("canceled retained task lookup must report the interruption")
	}
	status := tec.Status()
	if len(status.UncertainAgentIDs) != 1 || status.UncertainAgentIDs[0] != "agent-1" {
		t.Fatalf("canceled replay must retain the earlier potentially accepted POST, got %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].State != ObligationDispatchUncertain || persisted.Running[0].TaskRequestSnapshot != taskSnapshot {
		t.Fatalf("canceled replay lost its frozen uncertain obligation: %+v", persisted.Running)
	}
	if client.gets.Load() != 1 || len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("canceled replay sent owner mutations: gets=%d tasks=%d runs=%d", client.gets.Load(), len(client.createTaskCalls), len(client.createRunCalls))
	}
}

func TestReconcileTaskDispatchFailureIsPersistedWithoutOwnerErrorText(t *testing.T) {
	if got := dispatchFailureCode(&AgentManagerHTTPError{
		StatusCode: http.StatusForbidden, AdmissionReason: "untrusted-private-reason", Message: "private response detail",
	}); got != "owner_forbidden" {
		t.Fatalf("unrecognized owner reasons must collapse to the stable HTTP class, got %q", got)
	}

	dir := t.TempDir()
	task := dispatchTaskFixture()
	taskSnapshot, err := freezeTaskDispatchRequest(task)
	if err != nil {
		t.Fatal(err)
	}
	key := "prompt-manager-heartbeat:team-x:agent-1:diagnostic"
	client := newMockAgentClient()
	client.getTaskErr = &AgentManagerHTTPError{
		StatusCode:      http.StatusForbidden,
		AdmissionReason: "operation_not_admitted",
		Message:         "secret owner detail must not be persisted",
	}
	ctx := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	ctx.running["agent-1"] = runningEntry{
		ProfileKey:               "profile-p",
		State:                    ObligationDispatchUncertain,
		DispatchStage:            DispatchStageIntent,
		IdempotencyKey:           key,
		TaskID:                   task.ID,
		TaskRequestSnapshot:      taskSnapshot,
		CreateRunRequestSnapshot: createRunSnapshotForTest(t, key, task.ID, "profile-p", "heartbeat-team-x-agent-1"),
		CreateRunCallerMode:      createRunCallerOriginUnqualified,
	}
	if err := ctx.persistLocked(); err != nil {
		t.Fatal(err)
	}

	if err := ctx.ReconcileTaskDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatal("owner lookup failure must remain uncertain")
	}
	status := ctx.Status()
	if len(status.Owners) != 1 || status.Owners[0].DispatchFailurePhase != dispatchFailurePhaseTaskLookup || status.Owners[0].DispatchFailureCode != "operation_not_admitted" {
		t.Fatalf("expected bounded task-resolution diagnosis, got %+v", status.Owners)
	}
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("diagnostic recovery must not mutate owner state: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}

	statusBytes, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	queueBytes, err := os.ReadFile(ctx.queueFilePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{statusBytes, queueBytes} {
		if strings.Contains(string(data), "secret owner detail") {
			t.Fatalf("private owner error text leaked into retained diagnostics: %s", data)
		}
	}

	restarted := newTeamExecutionContext("team-x", &captureExecutor{}, dir, newMockAgentClient())
	restarted.Recover(context.Background())
	recovered := restarted.Status()
	if len(recovered.Owners) != 1 || recovered.Owners[0].State != ObligationDispatchUncertain ||
		recovered.Owners[0].DispatchFailurePhase != dispatchFailurePhaseTaskLookup ||
		recovered.Owners[0].DispatchFailureCode != "operation_not_admitted" {
		t.Fatalf("bounded failure diagnosis should survive recovery: %+v", recovered.Owners)
	}
}

func TestReconcileTaskDispatchMismatchHasSafeSubcause(t *testing.T) {
	dir := t.TempDir()
	task := dispatchTaskFixture()
	taskSnapshot, err := freezeTaskDispatchRequest(task)
	if err != nil {
		t.Fatal(err)
	}
	key := "prompt-manager-heartbeat:team-x:agent-1:mismatch"
	ownerTask := *task
	ownerTask.Description = "different frozen terms"
	client := newMockAgentClient()
	client.getTasks = map[string]*Task{task.ID: &ownerTask}
	ctx := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	ctx.running["agent-1"] = runningEntry{
		ProfileKey:               "profile-p",
		State:                    ObligationDispatchUncertain,
		DispatchStage:            DispatchStageIntent,
		IdempotencyKey:           key,
		TaskID:                   task.ID,
		TaskRequestSnapshot:      taskSnapshot,
		CreateRunRequestSnapshot: createRunSnapshotForTest(t, key, task.ID, "profile-p", "heartbeat-team-x-agent-1"),
		CreateRunCallerMode:      createRunCallerOriginUnqualified,
	}
	if err := ctx.persistLocked(); err != nil {
		t.Fatal(err)
	}

	if err := ctx.ReconcileTaskDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatal("same-ID task with different frozen terms must remain uncertain")
	}
	owners := ctx.Status().Owners
	if len(owners) != 1 || owners[0].DispatchFailurePhase != dispatchFailurePhaseTaskMatch || owners[0].DispatchFailureCode != "task_terms_mismatch" {
		t.Fatalf("expected bounded frozen-term mismatch diagnosis, got %+v", owners)
	}
	if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
		t.Fatalf("task mismatch must not trigger owner writes: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
	}
}

func TestReconcileTaskDispatchPreservesRunStageOwnerStates(t *testing.T) {
	for _, tc := range []struct {
		name      string
		createErr error
		wantState string
	}{
		{name: "approval pending", createErr: NewApprovalPendingError("creating_run", errors.New("approval required")), wantState: ObligationApprovalPending},
		{name: "capacity deferred", createErr: &AgentManagerHTTPError{StatusCode: http.StatusForbidden, AdmissionReason: "grant_limit_reached"}, wantState: ObligationAdmissionDeferred},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			task := dispatchTaskFixture()
			taskSnapshot, err := freezeTaskDispatchRequest(task)
			if err != nil {
				t.Fatal(err)
			}
			key := "prompt-manager-heartbeat:team-x:agent-1:run-stage-state"
			client := newMockAgentClient().WithCreateRunError(tc.createErr)
			client.getTasks = map[string]*Task{task.ID: task}
			ctx := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
			ctx.running["agent-1"] = runningEntry{
				ProfileKey:               "profile-p",
				State:                    ObligationDispatchUncertain,
				DispatchStage:            DispatchStageIntent,
				IdempotencyKey:           key,
				TaskID:                   task.ID,
				TaskRequestSnapshot:      taskSnapshot,
				CreateRunRequestSnapshot: createRunSnapshotForTest(t, key, task.ID, "profile-p", "heartbeat-team-x-agent-1"),
				CreateRunCallerMode:      createRunCallerOriginUnqualified,
			}
			if err := ctx.persistLocked(); err != nil {
				t.Fatal(err)
			}

			if err := ctx.ReconcileTaskDispatch(context.Background(), "agent-1"); err == nil {
				t.Fatal("run-stage refusal should be returned")
			}
			status := ctx.Status()
			if len(status.Owners) != 1 || status.Owners[0].State != tc.wantState {
				t.Fatalf("task confirmation must preserve the run-stage owner state %q, got %+v", tc.wantState, status.Owners)
			}
		})
	}
}

// TestBeginDispatch_PersistsIntentBeforeRequest verifies the dispatch intent
// (state plus owner idempotency key) is durable before the start request, so a
// crash mid-dispatch recovers as an obligation rather than a free slot.
func TestBeginDispatch_PersistsIntentBeforeRequest(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	tec.BeginDispatch("agent-1", DispatchIntent{
		IdempotencyKey:           "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
		TaskID:                   "task-1",
		CreateRunRequestSnapshot: createRunSnapshotForTest(t, "prompt-manager-heartbeat:team-x:agent-1:attempt-1", "task-1", "profile-1", "heartbeat-team-x-agent-1"),
		CreateRunCallerMode:      createRunCallerOriginUnqualified,
		RunTag:                   "heartbeat-team-x-agent-1",
		WorkloadKind:             "scheduled",
		WorkloadKey:              "effort:test",
		WorkloadInstance:         "attempt-1",
	})

	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 {
		t.Fatalf("expected one persisted running entry, got %+v", persisted.Running)
	}
	entry := persisted.Running[0]
	if entry.State != ObligationDispatchUncertain {
		t.Fatalf("expected dispatch_uncertain persisted before request, got state=%q", entry.State)
	}
	if entry.IdempotencyKey != "prompt-manager-heartbeat:team-x:agent-1:attempt-1" {
		t.Fatalf("expected persisted idempotency key, got %q", entry.IdempotencyKey)
	}
	if entry.TaskID != "task-1" || entry.RunTag != "heartbeat-team-x-agent-1" {
		t.Fatalf("expected persisted owner request essentials, got taskID=%q runTag=%q", entry.TaskID, entry.RunTag)
	}
	if entry.CreateRunRequestSnapshot == "" || entry.ProfileKey != "profile-1" {
		t.Fatalf("expected original request snapshot and resolved profile persisted, got profile=%q snapshot=%q", entry.ProfileKey, entry.CreateRunRequestSnapshot)
	}
	var persistedRequest CreateRunRequest
	if err := json.Unmarshal([]byte(entry.CreateRunRequestSnapshot), &persistedRequest); err != nil || persistedRequest.ProfileRef == nil || persistedRequest.ProfileRef.ProfileKey != "profile-1" {
		t.Fatalf("persisted original request does not retain resolved profile: request=%+v err=%v", persistedRequest, err)
	}
	if got := tec.Status(); len(got.UncertainAgentIDs) != 1 {
		t.Fatalf("expected the in-flight dispatch to be visible as uncertain, got %+v", got)
	}
}

// TestSetRunningRunID_ClearsDispatchIntent verifies a confirmed owner response
// resolves the pre-request uncertainty.
func TestSetRunningRunID_ClearsDispatchIntent(t *testing.T) {
	dir := t.TempDir()
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, nil)
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{IdempotencyKey: "key-1", TaskID: "task-1", CreateRunRequestSnapshot: createRunSnapshotForTest(t, "key-1", "task-1", "profile-1", "tag-key-1"), CreateRunCallerMode: createRunCallerOriginUnqualified})
	tec.SetRunningRunID("agent-1", "run-confirmed")

	got := tec.Status()
	if len(got.UncertainAgentIDs) != 0 {
		t.Fatalf("expected confirmed run to clear uncertainty, got %+v", got)
	}
	if len(got.RunningAgentIDs) != 1 {
		t.Fatalf("expected run retained, got %+v", got)
	}
}

// TestRecover_PreservesDispatchIntentKey verifies a crash mid-dispatch keeps the
// recorded idempotency key across restart for later owner reconciliation.
func TestRecover_PreservesDispatchIntentKey(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{AgentID: "agent-1", ProfileKey: "p", State: ObligationDispatchUncertain, IdempotencyKey: "key-1"},
		},
	})

	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, newMockAgentClient())
	tec.Recover(context.Background())

	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].IdempotencyKey != "key-1" {
		t.Fatalf("expected recorded dispatch intent key preserved, got %+v", persisted.Running)
	}
	if got := tec.Status(); len(got.UncertainAgentIDs) != 1 {
		t.Fatalf("expected uncertain obligation preserved, got %+v", got)
	}
	if got := len(tec.agentClient.(*mockAgentClient).createRunCalls); got != 0 {
		t.Fatalf("legacy incomplete entry must not issue CreateRun, got %d calls", got)
	}
}

func TestRecover_HoldsLegacyCompleteIntentWithoutFrozenRequest(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{{
			AgentID: "agent-1", ProfileKey: "profile-p", State: ObligationDispatchUncertain,
			IdempotencyKey: "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
			TaskID:         "task-1", RunTag: "heartbeat-team-x-agent-1",
			WorkloadKind: "scheduled", WorkloadKey: "effort:test", WorkloadInstance: "attempt-1",
		}},
	})
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "must-not-be-created", TaskID: "task-1", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)
	tec.Recover(context.Background())

	if len(client.createRunCalls) != 0 {
		t.Fatalf("legacy entry without frozen request must not be replayed, got %d CreateRun calls", len(client.createRunCalls))
	}
	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || len(status.UncertainAgentIDs) != 1 {
		t.Fatalf("legacy owner obligation must remain held and visible, got %+v", status)
	}
}

// watcherExecutor records retained-run watcher calls so the Recover/Reconcile
// path can be verified to re-attach a completion waiter.
type watcherExecutor struct {
	captureExecutor
	watched []struct {
		teamID, agentID, runID, profileKey, taskID, runTag string
		ownerStatus, ownerStartedAt, ownerTerminalAt       string
		admissionOrigin                                    string
	}
}

func (w *watcherExecutor) WatchRecoveredRunWithProjection(teamID, agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt, ownerTerminalAt, admissionOrigin string) bool {
	w.watched = append(w.watched, struct {
		teamID, agentID, runID, profileKey, taskID, runTag string
		ownerStatus, ownerStartedAt, ownerTerminalAt       string
		admissionOrigin                                    string
	}{teamID, agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt, ownerTerminalAt, admissionOrigin})
	return true
}

// TestReconcileDispatch_ReplaysRecordedIntentAndBindsRun verifies a retained
// uncertain dispatch converges on the owner's run by replaying the recorded
// request with the same idempotency key, and that a completion waiter is
// re-attached so the recovered obligation can eventually release.
func TestReconcileDispatch_ReplaysRecordedIntentAndBindsRun(t *testing.T) {
	dir := t.TempDir()
	ownerStarted := "2026-10-14T12:30:00Z"
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-v1", TaskID: "task-1", Status: "RUN_STATUS_RUNNING", StartedAt: ownerStarted})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)

	if _, err := tec.Enqueue(withHeartbeatAdmissionOrigin(context.Background(), AdmissionOriginScheduled), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{
		IdempotencyKey:           "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
		TaskID:                   "task-1",
		CreateRunRequestSnapshot: createRunSnapshotForTest(t, "prompt-manager-heartbeat:team-x:agent-1:attempt-1", "task-1", "profile-p", "heartbeat-team-x-agent-1"),
		CreateRunCallerMode:      createRunCallerOriginUnqualified,
		RunTag:                   "heartbeat-team-x-agent-1",
		WorkloadKind:             "scheduled",
		WorkloadKey:              "effort:test",
		WorkloadInstance:         "attempt-1",
		AdmissionOrigin:          AdmissionOriginScheduled,
	})

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err != nil {
		t.Fatalf("ReconcileDispatch: %v", err)
	}

	if len(client.createRunCalls) != 1 {
		t.Fatalf("expected exactly one replayed CreateRun, got %d", len(client.createRunCalls))
	}
	req := client.createRunCalls[0]
	if req.IdempotencyKey != "prompt-manager-heartbeat:team-x:agent-1:attempt-1" {
		t.Fatalf("expected replayed idempotency key preserved, got %q", req.IdempotencyKey)
	}
	if req.TaskID != "task-1" {
		t.Fatalf("expected replayed task preserved, got %q", req.TaskID)
	}
	if req.Environment[workloadKindEnv] != "scheduled" || req.Environment[workloadKeyEnv] != "effort:original" || req.Environment[workloadInstanceEnv] != "attempt-original" || req.Environment["VROOLI_SNAPSHOT_SENTINEL"] != "original-environment" {
		t.Fatalf("expected replayed workload identity preserved, got %+v", req.Environment)
	}

	status := tec.Status()
	if len(status.UncertainAgentIDs) != 0 {
		t.Fatalf("expected uncertainty cleared after confirmed replay, got %+v", status)
	}
	if len(status.RunningAgentIDs) != 1 || status.RunningAgentIDs[0] != "agent-1" {
		t.Fatalf("expected bound run retained, got %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-v1" {
		t.Fatalf("expected confirmed RunID persisted, got %+v", persisted.Running)
	}
	if persisted.Running[0].OwnerStartedAt != ownerStarted {
		t.Fatalf("owner run start time was not persisted, got %+v", persisted.Running[0])
	}
	if len(exec.watched) != 1 || exec.watched[0].runID != "run-v1" || exec.watched[0].teamID != "team-x" {
		t.Fatalf("expected one recovered-run watcher for run-v1, got %+v", exec.watched)
	}
	if exec.watched[0].profileKey != "profile-p" {
		t.Fatalf("watcher must use the frozen resolved profile, got %+v", exec.watched[0])
	}
	if exec.watched[0].ownerStartedAt != ownerStarted {
		t.Fatalf("watcher must use the owner run's original start time, got %+v", exec.watched[0])
	}
	if exec.watched[0].ownerTerminalAt != "" || exec.watched[0].admissionOrigin != AdmissionOriginScheduled {
		t.Fatalf("watcher must receive terminal projection and dispatch origin, got %+v", exec.watched[0])
	}
	if req.ParentRunID == nil || *req.ParentRunID != "parent-original" || req.RequestedScopes == nil || req.RequestedScopes.ExpectedOwnerSubject != "owner-original" || req.RunMode != "interactive" {
		t.Fatalf("replay lost optional original request terms: %+v", req)
	}
	if encoded, err := json.Marshal(req); err != nil || string(encoded) != createRunSnapshotForTest(t, req.IdempotencyKey, req.TaskID, "profile-p", "heartbeat-team-x-agent-1") {
		t.Fatalf("replay request bytes changed from the persisted snapshot: bytes=%s err=%v", encoded, err)
	}
	restartedExecutor := &watcherExecutor{}
	restartedClient := newMockAgentClient().WithGetRunResponse("run-v1", &Run{ID: "run-v1", Status: "RUN_STATUS_RUNNING"})
	restarted := newTeamExecutionContext("team-x", restartedExecutor, dir, restartedClient)
	restarted.Recover(context.Background())
	if len(restartedExecutor.watched) != 1 || restartedExecutor.watched[0].profileKey != "profile-p" || restartedExecutor.watched[0].ownerStartedAt != ownerStarted {
		t.Fatalf("later restart must retain the resolved profile for owner observation, got %+v", restartedExecutor.watched)
	}
}

// TestReconcileDispatch_UncertainOwnerRetainsObligation verifies an owner that
// is still unreachable does not release the slot or start a duplicate.
func TestReconcileDispatch_UncertainOwnerRetainsObligation(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateRunError(NewDispatchUncertainError(errors.New("connection reset")))
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{IdempotencyKey: "key-1", TaskID: "task-1", CreateRunRequestSnapshot: createRunSnapshotForTest(t, "key-1", "task-1", "profile-p", "tag-key-1"), CreateRunCallerMode: createRunCallerOriginUnqualified})

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatalf("expected uncertain replay to return an error")
	}
	if len(client.createRunCalls) != 1 {
		t.Fatalf("expected one replay attempt, got %d", len(client.createRunCalls))
	}
	status := tec.Status()
	if len(status.UncertainAgentIDs) != 1 || status.UncertainAgentIDs[0] != "agent-1" {
		t.Fatalf("expected uncertain obligation retained, got %+v", status)
	}
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected duplicate refused while uncertain, got %v", err)
	}
}

// TestReconcileDispatch_ReplayRefusalRetainsSlot verifies a replay-time
// rejection is not treated as proof that no run exists for the key.
func TestReconcileDispatch_ReplayRefusalRetainsSlot(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateRunError(NewDispatchRejectedError(errors.New("validation error")))
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.BeginDispatch("agent-1", DispatchIntent{IdempotencyKey: "key-1", TaskID: "task-1", CreateRunRequestSnapshot: createRunSnapshotForTest(t, "key-1", "task-1", "profile-p", "tag-key-1"), CreateRunCallerMode: createRunCallerOriginUnqualified})

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatalf("expected definitive refusal to return an error")
	}
	if got := tec.Status(); len(got.RunningAgentIDs) != 1 || len(got.UncertainAgentIDs) != 1 {
		t.Fatalf("expected rejected replay to retain uncertain slot, got %+v", got)
	}
	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err == nil || !IsMemberAlreadyQueued(err) {
		t.Fatalf("expected retained member to remain protected from duplicate admission, got %v", err)
	}
}

// TestReconcileDispatch_IncompleteIntentNotReplayed verifies a recorded
// obligation without enough owner request detail is never sent to the owner.
func TestReconcileDispatch_IncompleteIntentNotReplayed(t *testing.T) {
	dir := t.TempDir()
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-v1", Status: "RUN_STATUS_RUNNING"})
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, dir, client)

	if _, err := tec.Enqueue(context.Background(), "agent-1", "profile-p"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	tec.MarkDispatchUncertain("agent-1", dispatchFailurePhaseRunReconciliation, errors.New("no recorded intent"))

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err == nil {
		t.Fatalf("expected incomplete intent to return an error")
	}
	if len(client.createRunCalls) != 0 {
		t.Fatalf("expected no owner replay for incomplete intent, got %d", len(client.createRunCalls))
	}
	if len(tec.Status().UncertainAgentIDs) != 1 {
		t.Fatalf("expected obligation retained, got %+v", tec.Status())
	}
}

// TestRecover_HoldsLegacyUncertainDispatchWithoutKnownOrigin verifies a
// restart does not infer scheduler or manual authority for a legacy uncertain
// intent. A later owner-qualified path may observe it, but startup cannot POST.
func TestRecover_HoldsLegacyUncertainDispatchWithoutKnownOrigin(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{
			{
				AgentID:                  "agent-1",
				ProfileKey:               "profile-p",
				State:                    ObligationDispatchUncertain,
				IdempotencyKey:           "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
				TaskID:                   "task-1",
				RunTag:                   "heartbeat-team-x-agent-1",
				CreateRunRequestSnapshot: createRunSnapshotForTest(t, "prompt-manager-heartbeat:team-x:agent-1:attempt-1", "task-1", "profile-p", "heartbeat-team-x-agent-1"),
				CreateRunCallerMode:      createRunCallerOriginUnqualified,
			},
		},
	})
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-recovered", TaskID: "task-1", Status: "RUN_STATUS_RUNNING"})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)

	tec.Recover(context.Background())

	if len(client.createRunCalls) != 0 {
		t.Fatalf("startup must not replay an intent with unknown trigger origin, got %d POSTs", len(client.createRunCalls))
	}
	status := tec.Status()
	if len(status.RunningAgentIDs) != 1 || len(status.UncertainAgentIDs) != 1 {
		t.Fatalf("expected unknown-origin obligation retained as uncertain, got %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "" || persisted.Running[0].AdmissionOrigin != AdmissionOriginUnknown {
		t.Fatalf("expected original unknown-origin intent retained without an inferred RunID, got %+v", persisted.Running)
	}
	if len(exec.watched) != 0 {
		t.Fatalf("an unknown RunID cannot have a watcher attached, got %+v", exec.watched)
	}
}

func TestRecoverPassesOwnerStartedAtToRecoveredWatcher(t *testing.T) {
	dir := t.TempDir()
	ownerStarted := "2026-10-14T12:30:00Z"
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{{
			AgentID: "agent-1", ProfileKey: "profile-p", State: ObligationRunning,
			RunID: "run-existing", TaskID: "task-1", RunTag: "heartbeat-team-x-agent-1",
		}},
	})
	client := newMockAgentClient().WithGetRunResponse("run-existing", &Run{
		ID: "run-existing", TaskID: "task-1", Status: "RUN_STATUS_NEEDS_REVIEW", StartedAt: ownerStarted,
	})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)

	tec.Recover(context.Background())

	if len(exec.watched) != 1 || exec.watched[0].ownerStatus != "RUN_STATUS_NEEDS_REVIEW" || exec.watched[0].ownerStartedAt != ownerStarted {
		t.Fatalf("recovery watcher did not receive authoritative owner state/start: %+v", exec.watched)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].State != ObligationNeedsReview || persisted.Running[0].OwnerStartedAt != ownerStarted {
		t.Fatalf("reconciled owner projection was not durably retained: %+v", persisted.Running)
	}
}

func TestReconcileDispatchProjectsOwnerNeedsReviewAndClearsPriorRunStart(t *testing.T) {
	dir := t.TempDir()
	key := "prompt-manager-heartbeat:team-x:agent-1:review-replay"
	client := newMockAgentClient().WithCreateRunResponse(&Run{
		ID: "run-new", TaskID: "task-1", Status: "RUN_STATUS_NEEDS_REVIEW",
	})
	exec := &watcherExecutor{}
	tec := newTeamExecutionContext("team-x", exec, dir, client)
	tec.running["agent-1"] = runningEntry{
		ProfileKey: "profile-p", RunID: "run-old", State: ObligationDispatchUncertain,
		IdempotencyKey: key, TaskID: "task-1", TaskConfirmed: true,
		OwnerStartedAt:           "2026-10-10T10:00:00Z",
		CreateRunRequestSnapshot: createRunSnapshotForTest(t, key, "task-1", "profile-p", "tag-review"),
		CreateRunCallerMode:      createRunCallerOriginUnqualified,
	}
	if err := tec.persistLocked(); err != nil {
		t.Fatal(err)
	}

	if err := tec.ReconcileDispatch(context.Background(), "agent-1"); err != nil {
		t.Fatal(err)
	}
	status := tec.Status()
	if len(status.NeedsReviewAgentIDs) != 1 || status.NeedsReviewAgentIDs[0] != "agent-1" {
		t.Fatalf("replayed needs-review owner status was projected as green/running: %+v", status)
	}
	persisted := readQueueFile(t, dir, "team-x")
	if len(persisted.Running) != 1 || persisted.Running[0].RunID != "run-new" || persisted.Running[0].State != ObligationNeedsReview || persisted.Running[0].OwnerStartedAt != "" {
		t.Fatalf("queue projection must retain review state and unknown original start without borrowing the old run time: %+v", persisted.Running)
	}
	if len(exec.watched) != 1 || exec.watched[0].ownerStatus != "RUN_STATUS_NEEDS_REVIEW" || exec.watched[0].ownerStartedAt != "" {
		t.Fatalf("watcher did not receive the same owner state and unknown start: %+v", exec.watched)
	}
}

func TestConcurrentScheduledApprovalRetriesUseOneDispatchAndWatcher(t *testing.T) {
	dir := t.TempDir()
	writeQueueFile(t, dir, "team-x", persistedTeamQueue{
		TeamID:            "team-x",
		QueuePolicy:       "serialized",
		MaxConcurrentRuns: 1,
		Running: []queuedExecution{{
			AgentID: "agent-1", ProfileKey: "profile-p", State: ObligationApprovalPending,
			IdempotencyKey: "prompt-manager-heartbeat:team-x:agent-1:attempt-1",
			TaskID:         "task-1", RunTag: "heartbeat-team-x-agent-1",
			CreateRunRequestSnapshot: createRunSnapshotForTest(t, "prompt-manager-heartbeat:team-x:agent-1:attempt-1", "task-1", "profile-p", "heartbeat-team-x-agent-1"),
			CreateRunCallerMode:      createRunCallerOriginUnqualified,
		}},
	})
	client := newMockAgentClient().WithCreateRunResponse(&Run{ID: "run-approved", TaskID: "task-1", Status: "RUN_STATUS_RUNNING"})
	exec := &watcherExecutor{}
	store := NewTeamExecutionStore(nil, exec, dir, client)
	store.Recover(context.Background())
	scheduler := NewScheduler(exec, client, nil, store)

	var callbacks sync.WaitGroup
	callbacks.Add(2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			defer callbacks.Done()
			<-start
			scheduler.executeHeartbeat(context.Background(), "team-x", "agent-1")
		}()
	}
	close(start)
	callbacks.Wait()

	if len(client.createRunCalls) != 1 {
		t.Fatalf("concurrent approval retries must issue one same-intent request, got %d", len(client.createRunCalls))
	}
	if got := client.createRunCalls[0].IdempotencyKey; got != "prompt-manager-heartbeat:team-x:agent-1:attempt-1" {
		t.Fatalf("approval retry lost the recorded idempotency key: %q", got)
	}
	if len(client.createTaskCalls) != 0 || len(exec.watched) != 1 {
		t.Fatalf("approval retry should create no task and attach one watcher: tasks=%d watchers=%d", len(client.createTaskCalls), len(exec.watched))
	}
}

func TestSetRunningRunID_NoopWhenAgentNotRunning(t *testing.T) {
	tec := newTeamExecutionContext("team-x", &captureExecutor{}, t.TempDir(), nil)
	// Does not panic; just logs and returns.
	tec.SetRunningRunID("ghost-agent", "run-x")
	if len(tec.Status().RunningAgentIDs) != 0 {
		t.Fatalf("expected no running entries after no-op SetRunningRunID")
	}
}

// fakeTeamExecStoreRegistrar captures SetRunningRunID calls for executor tests.
type fakeTeamExecStoreRegistrar struct {
	calls        []struct{ TeamID, AgentID, RunID string }
	beginCalls   []struct{ TeamID, AgentID, Key string }
	confirmCalls []struct{ TeamID, AgentID string }
	beginErr     error
	deferErr     error
	pendingCalls []struct{ TeamID, AgentID string }
}

func (f *fakeTeamExecStoreRegistrar) SetRunningRunID(teamID, agentID, runID string, _ ...string) {
	f.calls = append(f.calls, struct{ TeamID, AgentID, RunID string }{teamID, agentID, runID})
}

func (f *fakeTeamExecStoreRegistrar) BeginDispatch(teamID, agentID string, intent DispatchIntent) error {
	f.beginCalls = append(f.beginCalls, struct{ TeamID, AgentID, Key string }{teamID, agentID, intent.IdempotencyKey})
	return f.beginErr
}

func (f *fakeTeamExecStoreRegistrar) ConfirmTaskDispatch(teamID, agentID string) error {
	f.confirmCalls = append(f.confirmCalls, struct{ TeamID, AgentID string }{teamID, agentID})
	return nil
}

func (f *fakeTeamExecStoreRegistrar) CancelUnsentTaskDispatch(teamID, agentID, taskID, idempotencyKey, taskSnapshot string) error {
	return nil
}

func (f *fakeTeamExecStoreRegistrar) MarkApprovalPending(teamID, agentID string) {
	f.pendingCalls = append(f.pendingCalls, struct{ TeamID, AgentID string }{teamID, agentID})
}

func (f *fakeTeamExecStoreRegistrar) MarkAdmissionDeferred(string, string) error { return f.deferErr }

func TestExecutor_SetTeamExecStore_RegistersRunIDAfterCreateRun(t *testing.T) {
	// This exercises the Executor.SetTeamExecStore wiring. A full Execute
	// integration test would need a configured team — keep this focused on
	// the seam: the registrar is captured and invokable.
	registrar := &fakeTeamExecStoreRegistrar{}
	e := &Executor{}
	e.SetTeamExecStore(registrar)
	if e.teamExecStore == nil {
		t.Fatalf("expected teamExecStore wired")
	}
	e.teamExecStore.SetRunningRunID("t", "a", "r")
	if len(registrar.calls) != 1 || registrar.calls[0].RunID != "r" {
		t.Fatalf("expected registrar.SetRunningRunID called once, got %+v", registrar.calls)
	}
}

// --- HTTP handler tests ---
