package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
	amapi "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

type replacementOnStopClient struct {
	*mockAgentClient
	registry  *RunRegistry
	teamStore *store.FileTeamStore
}

type cancelledWaitClient struct {
	*mockAgentClient
	waitStarted chan struct{}
	waitOnce    sync.Once
}

func (c *cancelledWaitClient) WaitForRun(ctx context.Context, _ string, _ time.Duration) (*Run, error) {
	c.waitOnce.Do(func() { close(c.waitStarted) })
	<-ctx.Done()
	return nil, ctx.Err()
}

func (c *cancelledWaitClient) GetRun(ctx context.Context, runID string) (*Run, error) {
	return c.mockAgentClient.GetRun(ctx, runID)
}

func (c *replacementOnStopClient) StopRun(ctx context.Context, runID string) error {
	err := c.mockAgentClient.StopRun(ctx, runID)
	c.registry.Register("team-stop", "agent-stop", "replacement-run", time.Now(), nil)
	config, readErr := c.teamStore.GetHeartbeatConfig(ctx, "team-stop", "agent-stop")
	if readErr == nil && config != nil && config.LastExecution != nil {
		config.LastExecution.RunID = "replacement-run"
		config.LastExecution.Status = store.HeartbeatStatusRunning
		if writeErr := c.teamStore.SetHeartbeatConfig(ctx, "team-stop", "agent-stop", config); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func TestStopRunningCompletesOnlyCapturedRunAfterStop(t *testing.T) {
	registry := NewRunRegistry(t.TempDir())
	registry.Register("team-stop", "agent-stop", "original-run", time.Now(), nil)
	client := &replacementOnStopClient{
		mockAgentClient: newMockAgentClient(),
		registry:        registry,
	}
	test := newTestHandlers(t, func(cfg *testHandlersConfig) {
		cfg.agentClient = client
		cfg.runRegistry = registry
	})
	if err := test.TeamStore.Create(context.Background(), newIndependentTestTeam("team-stop", "Stop test")); err != nil {
		t.Fatal(err)
	}
	client.teamStore = test.TeamStore
	if err := test.TeamStore.SetHeartbeatConfig(context.Background(), "team-stop", "agent-stop", &store.HeartbeatConfig{
		LastExecution: &store.HeartbeatExecResult{RunID: "original-run", Status: store.HeartbeatStatusRunning},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/heartbeats/running/team-stop/agent-stop/stop", nil)
	request = mux.SetURLVars(request, map[string]string{"teamId": "team-stop", "agentId": "agent-stop"})
	response := httptest.NewRecorder()
	test.Handlers.StopRunning(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("stop status: %d %s", response.Code, response.Body.String())
	}
	if len(client.stopRunCalls) != 1 || client.stopRunCalls[0] != "original-run" {
		t.Fatalf("StopRun calls = %v, want original-run", client.stopRunCalls)
	}
	active, ok := registry.GetActiveRun("team-stop", "agent-stop")
	if !ok || active.RunID != "replacement-run" {
		t.Fatalf("stopping captured owner removed replacement: active=%+v, ok=%t", active, ok)
	}
	config, err := test.TeamStore.GetHeartbeatConfig(context.Background(), "team-stop", "agent-stop")
	if err != nil || config.LastExecution.RunID != "replacement-run" || config.LastExecution.Status != store.HeartbeatStatusRunning {
		t.Fatalf("stopping captured owner overwrote replacement execution: config=%+v err=%v", config, err)
	}
}

func TestStopRunningPreservesOwnerWhenStopFails(t *testing.T) {
	registry := NewRunRegistry(t.TempDir())
	registry.Register("team-stop", "agent-stop", "captured-run", time.Now(), nil)
	client := newMockAgentClient().WithStopRunError(errors.New("owner unavailable"))
	test := newTestHandlers(t, func(cfg *testHandlersConfig) {
		cfg.agentClient = client
		cfg.runRegistry = registry
	})
	if err := test.TeamStore.Create(context.Background(), newIndependentTestTeam("team-stop", "Stop test")); err != nil {
		t.Fatal(err)
	}
	if err := test.TeamStore.SetHeartbeatConfig(context.Background(), "team-stop", "agent-stop", &store.HeartbeatConfig{
		LastExecution: &store.HeartbeatExecResult{RunID: "captured-run", Status: store.HeartbeatStatusRunning},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/heartbeats/running/team-stop/agent-stop/stop", nil)
	request = mux.SetURLVars(request, map[string]string{"teamId": "team-stop", "agentId": "agent-stop"})
	response := httptest.NewRecorder()
	test.Handlers.StopRunning(response, request)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("failed owner stop status = %d, want %d: %s", response.Code, http.StatusBadGateway, response.Body.String())
	}
	active, ok := registry.GetActiveRun("team-stop", "agent-stop")
	if !ok || active.RunID != "captured-run" {
		t.Fatalf("failed stop discarded active owner: active=%+v ok=%t", active, ok)
	}
	config, err := test.TeamStore.GetHeartbeatConfig(context.Background(), "team-stop", "agent-stop")
	if err != nil || config.LastExecution.RunID != "captured-run" || config.LastExecution.Status != store.HeartbeatStatusRunning {
		t.Fatalf("failed stop changed execution projection: config=%+v err=%v", config, err)
	}
}

func TestStopRunningReconcilesCancelledWaiterBeforeReleasingExactQueueOwner(t *testing.T) {
	registry := NewRunRegistry(t.TempDir())
	client := &cancelledWaitClient{
		mockAgentClient: newMockAgentClient().WithGetRunResponse("cancel-run", &Run{ID: "cancel-run", Status: "RUN_STATUS_CANCELLED"}),
		waitStarted:     make(chan struct{}),
	}
	test := newTestHandlers(t, func(cfg *testHandlersConfig) {
		cfg.agentClient = client
		cfg.runRegistry = registry
	})
	if err := test.TeamStore.Create(context.Background(), newIndependentTestTeam("team-stop", "Stop test")); err != nil {
		t.Fatal(err)
	}
	if err := test.TeamStore.SetHeartbeatConfig(context.Background(), "team-stop", "agent-stop", &store.HeartbeatConfig{
		LastExecution: &store.HeartbeatExecResult{RunID: "cancel-run", Status: store.HeartbeatStatusRunning},
	}); err != nil {
		t.Fatal(err)
	}

	teamExecStore := NewTeamExecutionStore(test.TeamStore, &captureExecutor{}, t.TempDir(), client)
	teamExec := teamExecStore.GetOrCreate("team-stop")
	if _, err := teamExec.Enqueue(context.Background(), "agent-stop", "profile-stop"); err != nil {
		t.Fatal(err)
	}
	teamExec.SetRunningRunID("agent-stop", "cancel-run")
	test.Executor.teamExecStore = teamExecStore
	test.Executor.OnRunComplete = teamExecStore.OnRunComplete
	test.Handlers.teamExecStore = teamExecStore

	waitCtx, cancelWait := context.WithCancel(context.Background())
	registry.Register("team-stop", "agent-stop", "cancel-run", time.Now(), cancelWait)
	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		test.Executor.waitForCompletion(waitCtx, "team-stop", "agent-stop", "cancel-run", "attempt-stop", "profile-stop", "task-stop", "tag-stop", time.Now(), "", time.Minute)
	}()
	select {
	case <-client.waitStarted:
	case <-time.After(time.Second):
		t.Fatal("completion waiter did not start")
	}

	request := httptest.NewRequest(http.MethodPost, "/heartbeats/running/team-stop/agent-stop/stop", nil)
	request = mux.SetURLVars(request, map[string]string{"teamId": "team-stop", "agentId": "agent-stop"})
	response := httptest.NewRecorder()
	test.Handlers.StopRunning(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("confirmed stop status = %d: %s", response.Code, response.Body.String())
	}
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("cancelled waiter did not reconcile the terminal owner")
	}

	if _, ok := registry.GetActiveRun("team-stop", "agent-stop"); ok {
		t.Fatal("terminal run remained in the active registry")
	}
	if got := teamExecStore.Status("team-stop").RunningAgentIDs; len(got) != 0 {
		t.Fatalf("exact team queue owner was not released: %v", got)
	}
	config, err := test.TeamStore.GetHeartbeatConfig(context.Background(), "team-stop", "agent-stop")
	if err != nil || config.LastExecution.RunID != "cancel-run" || config.LastExecution.Status != store.HeartbeatStatusCancelled {
		t.Fatalf("confirmed cancellation projection = %+v, err=%v", config, err)
	}
}

func TestStopRunningReattachesRecoveredParkedOwnerWithoutCancelFn(t *testing.T) {
	registry := NewRunRegistry(t.TempDir())
	registry.RegisterRecovered("team-stop", "agent-stop", "parked-run", time.Time{}, ObligationPaused, nil)
	terminal := &Run{ID: "parked-run", TaskID: "task-stop", ProfileID: "owner-profile-id", Status: "RUN_STATUS_CANCELLED", StartedAt: "2026-10-20T08:00:00Z", EndedAt: "2026-10-20T08:01:00Z"}
	client := newMockAgentClient().WithGetRunResponse(terminal.ID, terminal).WithWaitRunResponse(terminal)
	test := newTestHandlers(t, func(cfg *testHandlersConfig) {
		cfg.agentClient = client
		cfg.runRegistry = registry
	})
	defer test.Executor.Shutdown()
	if err := test.TeamStore.Create(context.Background(), newIndependentTestTeam("team-stop", "Stop test")); err != nil {
		t.Fatal(err)
	}
	if err := test.TeamStore.SetHeartbeatConfig(context.Background(), "team-stop", "agent-stop", &store.HeartbeatConfig{
		LastExecution: &store.HeartbeatExecResult{RunID: terminal.ID, Status: store.HeartbeatStatusNeedsReview},
	}); err != nil {
		t.Fatal(err)
	}
	teamExecStore := NewTeamExecutionStore(test.TeamStore, &captureExecutor{}, t.TempDir(), client)
	teamExec := teamExecStore.GetOrCreate("team-stop")
	if _, err := teamExec.Enqueue(context.Background(), "agent-stop", "heartbeat-profile-key"); err != nil {
		t.Fatal(err)
	}
	teamExec.SetRunningRunID("agent-stop", terminal.ID, terminal.StartedAt)
	test.Executor.SetTeamExecStore(teamExecStore)
	test.Executor.OnRunComplete = teamExecStore.OnRunComplete
	test.Handlers.teamExecStore = teamExecStore

	request := httptest.NewRequest(http.MethodPost, "/heartbeats/running/team-stop/agent-stop/stop", nil)
	request = mux.SetURLVars(request, map[string]string{"teamId": "team-stop", "agentId": "agent-stop"})
	response := httptest.NewRecorder()
	test.Handlers.StopRunning(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("confirmed stop status = %d: %s", response.Code, response.Body.String())
	}
	waitForCondition(t, "recovered parked owner terminal projection and exact queue release", func() bool {
		config, readErr := test.TeamStore.GetHeartbeatConfig(context.Background(), "team-stop", "agent-stop")
		active, activeOK := registry.GetActiveRun("team-stop", "agent-stop")
		status := teamExec.Status()
		return readErr == nil && config.LastExecution != nil && config.LastExecution.RunID == terminal.ID &&
			config.LastExecution.Status == store.HeartbeatStatusCancelled && !activeOK && active == nil && len(status.RunningAgentIDs) == 0
	})
	attempts, total, err := test.TeamStore.ListHeartbeatAttempts(context.Background(), "team-stop", "agent-stop", store.HeartbeatStatusCancelled, "", 10, 0)
	if err != nil || total != 1 || len(attempts) != 1 || attempts[0].ProfileKey != "heartbeat-profile-key" {
		t.Fatalf("recovered stop attempt profile key = total %d %+v err=%v", total, attempts, err)
	}
}

func TestCancelledWaiterRetainsOwnerWhenTerminalProjectionCannotBeConfirmed(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	registry := NewRunRegistry(t.TempDir())
	client := &cancelledWaitClient{
		mockAgentClient: newMockAgentClient(),
		waitStarted:     make(chan struct{}),
	}
	client.mockAgentClient.getRunErr = errors.New("owner read unavailable")
	teamExecStore := NewTeamExecutionStore(teamStore, &captureExecutor{}, t.TempDir(), client)
	teamExec := teamExecStore.GetOrCreate("team-1")
	if _, err := teamExec.Enqueue(context.Background(), "agent-1", "profile-1"); err != nil {
		t.Fatal(err)
	}
	teamExec.SetRunningRunID("agent-1", "cancel-run")
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), registry, nil)
	executor.teamExecStore = teamExecStore
	executor.OnRunComplete = teamExecStore.OnRunComplete

	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-1", "agent-1", &store.HeartbeatConfig{
		LastExecution: &store.HeartbeatExecResult{RunID: "cancel-run", Status: store.HeartbeatStatusRunning},
	}); err != nil {
		t.Fatal(err)
	}
	waitCtx, cancelWait := context.WithCancel(context.Background())
	registry.Register("team-1", "agent-1", "cancel-run", time.Now(), cancelWait)
	cancelWait()
	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		executor.waitForCompletion(waitCtx, "team-1", "agent-1", "cancel-run", "attempt-stop", "profile-1", "task-1", "tag-stop", time.Now(), "", time.Minute)
	}()
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("cancelled waiter did not record the unconfirmed owner state")
	}

	active, ok := registry.GetActiveRun("team-1", "agent-1")
	if !ok || active.RunID != "cancel-run" || active.State != ObligationWatcherUnavailable {
		t.Fatalf("unconfirmed cancellation did not retain observer obligation: active=%+v ok=%t", active, ok)
	}
	if got := teamExecStore.Status("team-1").RunningAgentIDs; len(got) != 1 || got[0] != "agent-1" {
		t.Fatalf("unconfirmed cancellation released queue owner: %v", got)
	}
	config, err := teamStore.GetHeartbeatConfig(context.Background(), "team-1", "agent-1")
	if err != nil || config.LastExecution.RunID != "cancel-run" || config.LastExecution.Status != store.HeartbeatStatusObserverUnavailable {
		t.Fatalf("unconfirmed cancellation projection = %+v, err=%v", config, err)
	}
}

func TestListRunningKeepsRetainedOwnerStateAndOmitsUnknownStart(t *testing.T) {
	registry := NewRunRegistry(t.TempDir())
	registry.RegisterRecovered("team-a", "agent-a", "run-a", time.Time{}, ObligationNeedsReview, nil)
	knownStart := time.Date(2026, 10, 14, 12, 30, 0, 0, time.UTC)
	registry.RegisterRecovered("team-a", "agent-b", "run-b", knownStart, ObligationRunning, nil)
	h := &Handlers{runRegistry: registry}
	w := httptest.NewRecorder()
	h.ListRunning(w, httptest.NewRequest(http.MethodGet, "/heartbeats/running", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var response RunningAgentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Count != 2 || len(response.Agents) != 2 {
		t.Fatalf("unexpected running response: %+v", response)
	}
	entries := make(map[string]RunningAgentEntry, len(response.Agents))
	for _, entry := range response.Agents {
		entries[entry.RunID] = entry
	}
	unknown := entries["run-a"]
	if unknown.State != ObligationNeedsReview {
		t.Fatalf("owner state = %q, want %q", unknown.State, ObligationNeedsReview)
	}
	if unknown.StartedAt != "" || unknown.Duration != "" {
		t.Fatalf("unknown owner start was fabricated: %+v", unknown)
	}
	known := entries["run-b"]
	if known.StartedAt != knownStart.Format(time.RFC3339) || known.Duration == "" || known.State != ObligationRunning {
		t.Fatalf("known owner projection was lost: %+v", known)
	}
	if strings.Contains(w.Body.String(), "0001-01-01") {
		t.Fatalf("response contains zero-time placeholder: %s", w.Body.String())
	}
}

func TestTeamRunAccountingJoinsRecordedIDsAndDeduplicatesResumes(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team-a", "Team A")); err != nil {
		t.Fatal(err)
	}
	for i, row := range []struct{ team, member, run string }{
		{"team-a", "lead", "resumed"},
		{"team-a", "lead", "resumed"},
		{"team-a", "worker", "imported"},
		{"team-a", "worker", "missing"},
		{"team-b", "lead", "foreign"},
		{"team-a", "lead", ""},
	} {
		if err := teams.AppendHeartbeatAttempt(ctx, row.team, &store.HeartbeatAttempt{
			ID: time.Unix(int64(i), 0).String(), TeamID: row.team, AgentID: row.member, RunID: row.run,
			StartedAt: "2026-09-12T10:00:00Z", Status: "completed",
		}); err != nil {
			t.Fatal(err)
		}
	}
	h.runRegistry = NewRunRegistry(t.TempDir())
	h.runRegistry.Register("team-a", "lead", "resumed", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), nil)
	owner := map[string]*ampb.Run{
		"resumed": {
			Id: "resumed", Status: ampb.RunStatus_RUN_STATUS_COMPLETE, HarnessKind: "codex", SessionId: "session",
			RequestedModel: "requested", ActualModel: "actual", TerminalClass: "verdict", StopReason: "blocked",
			Summary: &ampb.RunSummary{TokensUsed: 120},
		},
		"imported": {
			Id: "imported", Status: ampb.RunStatus_RUN_STATUS_COMPLETE, ImportSourceHarness: "codex", ImportSourceSessionId: "session",
			ActualModel: "actual", TerminalClass: "verdict", StopReason: "blocked", Summary: &ampb.RunSummary{TokensUsed: 120},
		},
	}
	reads := make(chan string, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/v1/runs/") {
			t.Errorf("expected exact owner run read, got %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		id := r.URL.Path[len("/api/v1/runs/"):]
		reads <- id
		if owner[id] == nil {
			http.Error(w, "owner unavailable", http.StatusNotFound)
			return
		}
		wire, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(&amapi.GetRunResponse{Run: owner[id]})
		if err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(wire)
	}))
	defer server.Close()
	h.agentClient = newTestClient(t, server)
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team-a&start=2026-09-12T00:00:00Z&end=2026-09-13T00:00:00Z", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]float64{"knownRuns": 3, "observedRuns": 2, "unavailableRuns": 1, "unqueriedRuns": 0, "observedExecutions": 1, "duplicateExecutions": 1} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}
	if _, ok := got["actualModels"]; !ok {
		t.Fatalf("missing accounting projection: %s", w.Body.String())
	}
	if got["actualModels"].(map[string]any)["actual"] != float64(1) {
		t.Fatal("models not deduplicated")
	}
	if got["terminalReasons"].(map[string]any)["blocked"] != float64(1) {
		t.Fatal("blocked verdict became accepted completion")
	}
	usage := got["usage"].(map[string]any)
	if usage["tokens"] != nil || usage["costUSD"] != nil || usage["qualifiedRuns"] != float64(0) || usage["reportedTokenRuns"] != float64(2) {
		t.Fatalf("unqualified summary became qualified usage: %v", usage)
	}
	if len(reads) != 3 {
		t.Fatalf("expected one read per exact owner id, got %d", len(reads))
	}
}

func TestTeamRunAccountingUsesRecordAfterActiveRegistryRetires(t *testing.T) {
	h, teams, agents, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	if err := teams.SetHeartbeatConfig(ctx, "team", "lead", &store.HeartbeatConfig{TeamID: "team", AgentID: "lead"}); err != nil {
		t.Fatal(err)
	}
	registry := NewRunRegistry(t.TempDir())
	e := newTestExecutor(t, teams, agents, nil, t.TempDir(), registry, nil)
	started := time.Now().UTC().Add(-time.Hour)
	registry.Register("team", "lead", "original", started, nil)
	for _, status := range []string{store.HeartbeatStatusRunning, store.HeartbeatStatusFailed} {
		e.appendAttempt(ctx, &store.HeartbeatAttempt{ID: "wake", TeamID: "team", AgentID: "lead", RunID: "original", Status: status, StartedAt: started.Format(time.RFC3339)})
	}
	registry.CompleteRun("team", "lead", "original", true, "")
	if err := teams.SetHeartbeatConfig(ctx, "team", "lead", &store.HeartbeatConfig{TeamID: "team", AgentID: "lead", LastExecution: &store.HeartbeatExecResult{RunID: "original", StartedAt: started.Format(time.RFC3339), Status: store.HeartbeatStatusFailed}}); err != nil {
		t.Fatal(err)
	}
	if registry.Count() != 0 {
		t.Fatal("fixture has not retired active run")
	}
	// The current owner has resumed that same identity. Local terminal state
	// and repeated declaration rows must neither win nor count another run.
	h.runRegistry = registry
	h.agentClient = newMockAgentClient().WithGetRunResponse("original", &Run{ID: "original", Status: "RUN_STATUS_RUNNING", ActualModel: "actual"})
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 1 || got.ObservedExecutions != 1 || got.RuntimeStates["running"] != 1 || got.RuntimeStates["failed"] != 0 {
		t.Fatalf("local history overrode current resumed owner: %+v", got)
	}
}

func TestTeamRunAccountingBoundsOwnerReadsAndKeepsMissingIdentity(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if err := teams.AppendHeartbeatAttempt(ctx, "team", &store.HeartbeatAttempt{
			ID: fmt.Sprint(i), TeamID: "team", AgentID: "lead",
			RunID: fmt.Sprintf("run-%d", i), StartedAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
	}
	client := newMockAgentClient().WithGetRunResponse("run-0", &Run{ID: "wrong-id", Status: "complete"})
	h.agentClient = client
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team&limit=1", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 4 || got.ObservedRuns != 0 || got.UnavailableRuns != 1 || got.UnqueriedRuns != 3 || len(client.getRunCalls) != 1 || !got.Coverage.Partial {
		t.Fatalf("bounded query lost known identities or promoted wrong owner: %+v", got)
	}
	if got.Usage.Tokens != nil || got.Usage.CostUSD != nil {
		t.Fatal("outage became zero usage")
	}
}

func TestTeamRunAccountingWindowMemberAndMissingMetricCoverage(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	for i, row := range []struct{ member, id, started string }{
		{"lead", "wanted", "2026-09-12T10:00:00Z"},
		{"worker", "wrong-member", "2026-09-12T10:00:00Z"},
		{"lead", "old", "2026-09-11T10:00:00Z"},
		{"lead", "upper-bound", "2026-09-13T00:00:00Z"},
		{"lead", "invalid-time", "missing"},
		{"lead", "", "2026-09-12T10:00:00Z"},
	} {
		if err := teams.AppendHeartbeatAttempt(ctx, "team", &store.HeartbeatAttempt{
			ID: fmt.Sprint(i), TeamID: "team", AgentID: row.member,
			RunID: row.id, StartedAt: row.started,
		}); err != nil {
			t.Fatal(err)
		}
	}
	h.agentClient = newMockAgentClient().WithGetRunResponse("wanted", &Run{ID: "wanted", Status: "RUN_STATUS_RUNNING", RequestedModel: "not-actual", Summary: json.RawMessage(`{"cost_estimate":0}`)})
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team&agent_id=lead&start=2026-09-12T00:00:00Z&end=2026-09-13T00:00:00Z", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 1 || got.Coverage.InvalidTimestamps != 1 || got.UnknownModelExecutions != 1 || len(got.ActualModels) != 0 || got.Usage.ReportedCostRuns != 1 || got.Usage.ReportedTokenRuns != 0 {
		t.Fatalf("window/member scope or metric presence lost: %+v", got)
	}
	if got.Usage.CostUSD != nil {
		t.Fatal("reported estimate became actual charge")
	}
}

func TestTeamRunAccountingDecodesProtoSummaryAndMarksMalformedAvailability(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	started := "2026-09-12T10:00:00Z"
	for _, id := range []string{"large", "malformed", "missing", "null-token"} {
		if err := teams.AppendHeartbeatAttempt(ctx, "team", &store.HeartbeatAttempt{
			ID: id, TeamID: "team", AgentID: "lead", RunID: id, StartedAt: started,
		}); err != nil {
			t.Fatal(err)
		}
	}
	h.agentClient = newMockAgentClient().
		WithGetRunResponse("large", &Run{ID: "large", Status: "RUN_STATUS_COMPLETE", Summary: json.RawMessage(`{"tokens_used":"9007199254740993"}`)}).
		WithGetRunResponse("malformed", &Run{ID: "malformed", Status: "RUN_STATUS_COMPLETE", Summary: json.RawMessage(`{"tokens_used":"not-an-int64"}`)}).
		WithGetRunResponse("missing", &Run{ID: "missing", Status: "RUN_STATUS_COMPLETE"}).
		WithGetRunResponse("null-token", &Run{ID: "null-token", Status: "RUN_STATUS_COMPLETE", Summary: json.RawMessage(`{"tokens_used":null}`)})
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team&start=2026-09-12T00:00:00Z&end=2026-09-13T00:00:00Z", nil))
	var got struct {
		UnavailableSummaries int `json:"unavailableSummaries"`
		Usage                struct {
			Partial           bool `json:"partial"`
			ReportedTokenRuns int  `json:"reportedTokenRuns"`
		} `json:"usage"`
		Runs []struct {
			RunID               string `json:"runId"`
			SummaryAvailability string `json:"summaryAvailability"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.UnavailableSummaries != 1 || !got.Usage.Partial || got.Usage.ReportedTokenRuns != 1 {
		t.Fatalf("summary availability/accounting = unavailable:%d partial:%t reportedTokens:%d, want 1/true/1; body=%s", got.UnavailableSummaries, got.Usage.Partial, got.Usage.ReportedTokenRuns, w.Body.String())
	}
	availability := map[string]string{}
	for _, row := range got.Runs {
		availability[row.RunID] = row.SummaryAvailability
	}
	if availability["large"] != "available" || availability["malformed"] != "unavailable" || availability["missing"] != "not_reported" || availability["null-token"] != "available" {
		t.Fatalf("summary availability = %v, want large=available malformed=unavailable missing=not_reported null-token=available", availability)
	}
}

func TestTeamRunAccountingDoesNotMergeDifferentOrUnknownRunners(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient()
	for i, harness := range []string{"codex", "claude-code", "", ""} {
		id := fmt.Sprintf("run-%d", i)
		if err := teams.AppendHeartbeatAttempt(ctx, "team", &store.HeartbeatAttempt{
			ID: id, TeamID: "team", AgentID: "lead", RunID: id,
			StartedAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
		client.WithGetRunResponse(id, &Run{ID: id, HarnessKind: harness, SessionID: "same-session-text", Status: "RUN_STATUS_RUNNING"})
	}
	h.agentClient = client
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 4 || got.ObservedExecutions != 4 || got.DuplicateExecutions != 0 || got.UnknownModelExecutions != 4 {
		t.Fatalf("unqualified session identity merged unrelated runs: %+v", got)
	}
}

func TestTeamRunAccountingRejectsUnboundedOrInvalidQueries(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	if err := teams.Create(context.Background(), newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"", "&limit=0", "&limit=101", "&start=invalid", "&start=2026-01-01T00:00:00Z&end=2026-09-12T00:00:00Z"} {
		url := "/runs?accounting=true&team_id=team" + query
		if query == "" {
			url = "/runs?accounting=true"
		}
		w := httptest.NewRecorder()
		h.ListRuns(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("query %q: status %d", query, w.Code)
		}
	}
}

// Exercise the actual AM proto JSON -> PM client -> PM handler boundary.
// A local-log fixture cannot establish what survives this projection.
func TestListRunsPreservesOwnerAccountingEvidence(t *testing.T) {
	owner := &amapi.ListRunsResponse{Runs: []*ampb.Run{{
		Id: "owner-run", Status: ampb.RunStatus_RUN_STATUS_COMPLETE,
		RequestedModel: "requested", ActualModel: "actual", HarnessKind: "codex",
		SessionId: "resumed-session", ImportSourceHarness: "codex", ImportSourceSessionId: "resumed-session",
		TerminalClass: "verdict", StopReason: "blocked",
		WorkReferences: []*eventpb.WorkReference{{
			Kind: "effort", Id: "effort-a", Relationship: "supervisor",
			Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
			State:      eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE, Verified: true,
		}},
		Summary: &ampb.RunSummary{TokensUsed: 123, CostEstimate: 0.2},
	}, {Id: "missing-metrics", Status: ampb.RunStatus_RUN_STATUS_RUNNING}}, Total: 2, HasMore: true}
	wire, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/runs" {
			t.Errorf("unexpected owner request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(wire)
	}))
	defer server.Close()
	h := NewHandlers(HandlersDeps{AgentClient: newTestClient(t, server)})
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?limit=2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var got, want map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wire, &want); err != nil {
		t.Fatal(err)
	}
	run := got["runs"].([]any)[0].(map[string]any)
	expected := want["runs"].([]any)[0].(map[string]any)
	for _, key := range []string{"requested_model", "actual_model", "harness_kind", "session_id", "import_source_harness", "import_source_session_id", "terminal_class", "stop_reason", "summary", "work_references"} {
		a, _ := json.Marshal(run[key])
		b, _ := json.Marshal(expected[key])
		if string(a) != string(b) {
			t.Errorf("lost owner %s: got %s want %s", key, a, b)
		}
	}
	missing := got["runs"].([]any)[1].(map[string]any)
	for _, key := range []string{"actual_model", "summary", "terminal_class", "stop_reason"} {
		if _, exists := missing[key]; exists {
			t.Errorf("missing %s became fabricated evidence: %v", key, missing[key])
		}
	}
	if got["has_more"] != true {
		t.Fatal("partial owner page became complete")
	}
}

func TestListRuns_ResolvesProfileKeyToAgentProfileID(t *testing.T) {
	mockClient := newMockAgentClient().
		WithResolvedProfile(&AgentProfile{
			Id:         "6a89415d-9f95-4ee0-a534-00619ce6fd5a",
			ProfileKey: "prompt-manager-heartbeat",
		}).
		WithListRunsResponse(&ListRunsResponse{
			Runs:  []*Run{{ID: "run-1", Status: "RUN_STATUS_COMPLETE"}},
			Total: 1,
		})

	handlers := NewHandlers(HandlersDeps{
		AgentClient: mockClient,
	})

	req := httptest.NewRequest(http.MethodGet, "/runs?profile_key=prompt-manager-heartbeat&limit=5", nil)
	w := httptest.NewRecorder()
	handlers.ListRuns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()

	if len(mockClient.resolveProfileCalls) != 1 {
		t.Fatalf("expected 1 ResolveProfile call, got %d", len(mockClient.resolveProfileCalls))
	}
	if got := mockClient.resolveProfileCalls[0]; got != "prompt-manager-heartbeat" {
		t.Fatalf("expected ResolveProfile(profile_key=prompt-manager-heartbeat), got %+v", got)
	}
	if len(mockClient.listRunsCalls) != 1 {
		t.Fatalf("expected 1 ListRuns call, got %d", len(mockClient.listRunsCalls))
	}
	if mockClient.listRunsCalls[0].AgentProfileID != "6a89415d-9f95-4ee0-a534-00619ce6fd5a" {
		t.Fatalf("expected AgentProfileID to be resolved UUID, got %q", mockClient.listRunsCalls[0].AgentProfileID)
	}
}

func TestListRuns_ProfileKeyResolutionFailureReturnsBadGateway(t *testing.T) {
	mockClient := newMockAgentClient().WithResolvedProfile(&AgentProfile{})
	handlers := NewHandlers(HandlersDeps{
		AgentClient: mockClient,
	})

	req := httptest.NewRequest(http.MethodGet, "/runs?profile_key=prompt-manager-heartbeat", nil)
	w := httptest.NewRecorder()
	handlers.ListRuns(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListRuns_ForwardsAgentIDAsOwnerProfileID(t *testing.T) {
	mockClient := newMockAgentClient().WithListRunsResponse(&ListRunsResponse{
		Runs: []*Run{{ID: "agent-run", Status: "RUN_STATUS_COMPLETE"}}, Total: 1,
	})
	handlers := NewHandlers(HandlersDeps{AgentClient: mockClient})

	req := httptest.NewRequest(http.MethodGet, "/runs?agent_id=agent-profile-1&limit=10", nil)
	w := httptest.NewRecorder()
	handlers.ListRuns(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()
	if len(mockClient.listRunsCalls) != 1 {
		t.Fatalf("expected one owner list call, got %d", len(mockClient.listRunsCalls))
	}
	if got := mockClient.listRunsCalls[0].AgentProfileID; got != "agent-profile-1" {
		t.Fatalf("expected exact agent profile ID, got %q", got)
	}
}

func TestListRunsMixedIdentityRefusesConflictingFilter(t *testing.T) {
	client := newMockAgentClient().WithResolvedProfile(&AgentProfile{Id: "6a89415d-9f95-4ee0-a534-00619ce6fd5a", ProfileKey: "prompt-manager/heartbeat-judgment"})
	handlers := NewHandlers(HandlersDeps{AgentClient: client})
	for _, id := range []string{"6a89415d-9f95-4ee0-a534-00619ce6fd5a", "different-uuid"} {
		recorder := httptest.NewRecorder()
		handlers.ListRuns(recorder, httptest.NewRequest(http.MethodGet, "/runs?profile_key=prompt-manager/heartbeat-judgment&agent_id="+id, nil))
		if id == "different-uuid" && recorder.Code != http.StatusBadRequest {
			t.Fatalf("conflicting identities accepted: %d", recorder.Code)
		}
	}
	if len(client.listRunsCalls) != 1 {
		t.Fatalf("conflict forwarded owner read: %v", client.listRunsCalls)
	}
}
