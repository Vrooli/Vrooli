package heartbeat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"

	"github.com/gorilla/mux"
)

type finiteFixture struct {
	runtime   *FiniteLeaderRuntime
	teams     *store.FileTeamStore
	relations store.RelationStore
	agent     *mockAgentClient
	queue     *effortQueueFake
}

func newFiniteFixture(t *testing.T) *finiteFixture {
	t.Helper()
	ctx := context.Background()
	files := newFileStore(t, paths.RootsForTest(t))
	teams := files.Teams().(*store.FileTeamStore)
	agents := files.Agents().(*store.FileAgentStore)
	team := newIndependentTestTeam("committee", "Finite committee")
	team.Coordination, _ = teamconfig.BuildCoordinationPreset(teamconfig.CoordinationPatternLeaderLed, teamconfig.RuntimeModeMultiProcess, "lead")
	team.Execution = teamconfig.Execution{QueuePolicy: teamconfig.QueuePolicySerialized, MaxConcurrentRuns: 1}
	if err := teams.Create(ctx, team); err != nil {
		t.Fatal(err)
	}
	if err := agents.Create(ctx, &store.Agent{ID: "lead", DisplayName: "Coordinator"}); err != nil {
		t.Fatal(err)
	}
	if err := files.Relations().SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "committee", AgentID: "lead", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	cfg := &store.HeartbeatConfig{Enabled: true, Schedule: "@hourly", ProfileKey: "qualified-profile", FiniteLeader: &teamconfig.FiniteLeader{
		EffortRef: "arbitrary:new-effort/42", AcceptedRevision: "accepted:revision/7", CoordinatorPromptRef: "prompt-manager://teams/committee/members/lead/heartbeat",
		SourceRefs: []string{"owner:accepted-assignment/7"},
	}}
	if err := teams.SetHeartbeatConfig(ctx, "committee", "lead", cfg); err != nil {
		t.Fatal(err)
	}
	if err := teams.SetHeartbeatInstructions(ctx, "committee", "lead", "Perform the accepted assignment; retain the human input wait."); err != nil {
		t.Fatal(err)
	}
	agent := newMockAgentClient().WithCreateTaskResponse(&Task{ID: "task-1"}).WithCreateRunResponse(&Run{ID: "leader-1", TaskID: "task-1", Status: "running"})
	agent.getRuns["leader-1"] = &Run{ID: "leader-1", TaskID: "task-1", Status: "running"}
	executor := newTestExecutor(t, teams, agents, agent, t.TempDir(), nil, nil)
	queue := &effortQueueFake{}
	f := &FiniteLeaderRuntime{Executor: executor, Queue: queue}
	executor.FiniteLeader = f
	return &finiteFixture{runtime: f, teams: teams, relations: files.Relations(), agent: agent, queue: queue}
}

func (f *finiteFixture) dispatch(t *testing.T) *store.FiniteLeaderState {
	t.Helper()
	if _, err := f.runtime.Tick(context.Background(), "committee", "lead"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.runtime.Executor.Execute(context.Background(), "committee", "lead", "ignored-override"); err != nil {
		t.Fatal(err)
	}
	state, err := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestFiniteLeaderExactBindingAndConcurrentAdmission(t *testing.T) {
	f := newFiniteFixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.runtime.Tick(context.Background(), "committee", "lead"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if f.queue.enqueues != 1 {
		t.Fatalf("concurrent triggers queued %d leaders", f.queue.enqueues)
	}
	state := f.dispatch(t)
	request := f.agent.createRunCalls[0]
	if state.RunID != "leader-1" || request.ProfileRef.ProfileKey != "qualified-profile" || request.IdempotencyKey != "finite-leader-"+state.ID {
		t.Fatalf("exact dispatch identity lost: state=%+v request=%+v", state, request)
	}
	refs := request.WorkReferences
	if len(refs) != 1 || refs[0].Id != state.Binding.EffortRef || refs[0].Revision != "accepted:revision/7" || refs[0].Relationship != "orchestrator" || !refs[0].Verified {
		t.Fatalf("missing exact AM effort association: %+v", refs)
	}
	description := f.agent.createTaskCalls[0].Description
	if !strings.Contains(description, "retain the human input wait") || !strings.Contains(description, state.Binding.SourceRefs[0]) {
		t.Fatal("normal coordinator prompt or accepted source references lost")
	}
	for _, guidance := range []string{
		"durable owner reads", "independent, verifiable work", "Park or checkpoint", "final handoff", "explicit completion receipt",
		"planners or workers", "parent handoff identity", "Independent review is bounded work", "successful finite child stays terminal",
	} {
		if !strings.Contains(description, guidance) {
			t.Fatalf("finite coordinator guidance missing %q", guidance)
		}
	}
}

func TestFiniteLeaderRestartRetainsEveryOwnerDisposition(t *testing.T) {
	for _, status := range []string{"queued", "running", "parked", "needs_review", "failed", "cancelled", "complete", "unknown"} {
		t.Run(status, func(t *testing.T) {
			f := newFiniteFixture(t)
			original := f.dispatch(t)
			f.agent.getRuns["leader-1"].Status = status
			// Lose all ephemeral queue/runtime state. Persistent binding must win.
			f.runtime = &FiniteLeaderRuntime{Executor: f.runtime.Executor, Queue: &effortQueueFake{}}
			for i := 0; i < 3; i++ {
				state, err := f.runtime.Tick(context.Background(), "committee", "lead")
				if err != nil || state.ID != original.ID || state.RunID != "leader-1" || state.Status != status {
					t.Fatalf("owner identity lost after restart: %+v %v", state, err)
				}
			}
			if len(f.agent.createRunCalls) != 1 || len(f.agent.stopRunCalls) != 0 {
				t.Fatal("tick replaced or stopped the retained owner run")
			}
		})
	}
}

func TestFiniteLeaderLostDispatchResponseReconcilesOnlyExactRun(t *testing.T) {
	f := newFiniteFixture(t)
	f.agent.createRunErr = errors.New("response lost")
	if _, err := f.runtime.Dispatch(context.Background(), "committee", "lead"); err == nil {
		t.Fatal("expected uncertain dispatch")
	}
	state, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if state == nil || !state.DispatchStarted || state.TaskID != "task-1" || state.RunID != "" {
		t.Fatalf("dispatch intent not durable: %+v", state)
	}
	f.agent.createRunErr = nil
	f.runtime = &FiniteLeaderRuntime{Executor: f.runtime.Executor, Queue: &effortQueueFake{}}
	for _, rows := range []*ListRunsResponse{
		{},
		{HasMore: true},
		{Runs: []*Run{{ID: "other", TaskID: "task-1", Tag: "wrong"}}},
		{Runs: []*Run{{ID: "a"}, {ID: "b"}}},
	} {
		f.agent.listRunsResp = rows
		if _, err := f.runtime.Tick(context.Background(), "committee", "lead"); err == nil {
			t.Fatal("uncertain owner evidence released the fence")
		}
	}
	f.agent.listRunsResp = &ListRunsResponse{Runs: []*Run{{ID: "recovered-exact", TaskID: "task-1", Tag: "finite-leader-" + state.ID, Status: "parked"}}, Total: 1}
	recovered, err := f.runtime.Tick(context.Background(), "committee", "lead")
	if err != nil || recovered.RunID != "recovered-exact" || len(f.agent.createRunCalls) != 2 || f.agent.createRunCalls[0].IdempotencyKey != f.agent.createRunCalls[1].IdempotencyKey {
		t.Fatalf("lost response not reconciled to original: %+v %v", recovered, err)
	}
}

func TestFiniteLeaderMissingOwnerAndTaskResponseRetainReservation(t *testing.T) {
	t.Run("missing-owner", func(t *testing.T) {
		f := newFiniteFixture(t)
		original := f.dispatch(t)
		delete(f.agent.getRuns, "leader-1")
		if _, err := f.runtime.Tick(context.Background(), "committee", "lead"); err == nil {
			t.Fatal("missing owner must remain uncertain")
		}
		state, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
		if state.ID != original.ID || state.RunID != original.RunID || len(f.agent.createRunCalls) != 1 {
			t.Fatal("missing owner released the original identity")
		}
	})
	t.Run("task-response-lost", func(t *testing.T) {
		f := newFiniteFixture(t)
		f.agent.createTaskErr = errors.New("task response lost")
		for i := 0; i < 2; i++ {
			if _, err := f.runtime.Dispatch(context.Background(), "committee", "lead"); err == nil {
				t.Fatal("uncertain task admission allowed progress")
			}
		}
		if len(f.agent.createTaskCalls) != 1 || len(f.agent.createRunCalls) != 0 {
			t.Fatal("task uncertainty generated a replacement operation")
		}
	})
}

func TestFiniteLeaderQueuedControlsFenceDispatch(t *testing.T) {
	for _, control := range []string{"disable", "retire", "team-disable", "global-pause", "team-pause"} {
		t.Run(control, func(t *testing.T) {
			f := newFiniteFixture(t)
			ctx := context.Background()
			original, err := f.runtime.Tick(ctx, "committee", "lead")
			if err != nil {
				t.Fatal(err)
			}
			cfg, _ := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
			switch control {
			case "disable":
				cfg.Enabled = false
			case "retire":
				cfg.FiniteLeader.Retired = true
			case "team-disable":
				team, _ := f.teams.Get(ctx, "committee")
				team.Enabled = false
				team.EnabledSet = true
				if err := f.teams.Update(ctx, "committee", team); err != nil {
					t.Fatal(err)
				}
			case "global-pause", "team-pause":
				f.runtime.Control = NewHeartbeatControlStore(t.TempDir())
				scope := ""
				if control == "team-pause" {
					scope = "committee"
				}
				if _, err := f.runtime.Control.Pause(ctx, scope, "operator stop", store.AttributionInfo{}); err != nil {
					t.Fatal(err)
				}
			}
			if err := f.teams.SetHeartbeatConfig(ctx, "committee", "lead", cfg); err != nil {
				t.Fatal(err)
			}
			if _, err := f.runtime.Dispatch(ctx, "committee", "lead"); err == nil {
				t.Fatal("queued dispatch bypassed latest control")
			}
			retained, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
			if retained.ID != original.ID || len(f.agent.createTaskCalls) != 0 {
				t.Fatal("control lost reservation or bought inference")
			}
		})
	}
}

func TestFiniteLeaderOwnerContinuationAndHeartbeatRead(t *testing.T) {
	f := newFiniteFixture(t)
	state := f.dispatch(t)
	run := f.agent.getRuns[state.RunID]
	run.Status, run.StartedAt, run.EndedAt = "failed", "2026-09-12T10:00:00Z", "2026-09-12T10:01:00Z"
	for i := 0; i < 2; i++ {
		if _, err := f.runtime.Tick(context.Background(), "committee", "lead"); err != nil {
			t.Fatal(err)
		}
	}
	cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	if cfg.ConsecutiveFailures != 1 || cfg.LastExecution.RunID != state.RunID {
		t.Fatalf("exact terminal observation was lost or double-counted: %+v", cfg)
	}
	// AM resumes the original identity; PM observes the new attempt's clock.
	run.Status, run.StartedAt, run.EndedAt = "running", "2026-09-12T10:02:00Z", ""
	if _, err := f.runtime.Tick(context.Background(), "committee", "lead"); err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(HandlersDeps{TeamStore: f.teams, RelationStore: f.relations, Executor: f.runtime.Executor})
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{"id": "committee", "agentId": "lead"})
	w := httptest.NewRecorder()
	h.GetHeartbeat(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"finiteLeaderState"`) || !strings.Contains(w.Body.String(), `"runId":"leader-1"`) || strings.Contains(w.Body.String(), `"endedAt"`) {
		t.Fatalf("heartbeat read lost current owner attempt: %s", w.Body.String())
	}
	if len(f.agent.createRunCalls) != 1 {
		t.Fatal("owner continuation created a replacement")
	}
}

func TestFiniteLeaderSchedulerWiringUsesExistingQueue(t *testing.T) {
	f := newFiniteFixture(t)
	e := f.runtime.Executor
	queue := NewTeamExecutionStore(f.teams, e, t.TempDir(), f.agent)
	e.SetTeamExecStore(queue)
	scheduler := NewScheduler(e, f.agent, f.teams, queue)
	control := NewHeartbeatControlStore(t.TempDir())
	scheduler.SetControlStore(control)
	if e.FiniteLeader == nil || e.FiniteLeader.Queue != queue || e.FiniteLeader.Control != control {
		t.Fatal("finite leader not wired to normal queue and controls")
	}
	// Hold capacity with a non-executing member, then enqueue through cron.
	queue.GetOrCreate("committee").running["other"] = runningEntry{}
	scheduler.executeHeartbeat(context.Background(), "committee", "lead")
	state, err := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if err != nil || state.ID == "" || state.DispatchStarted || len(queue.Status("committee").Queue) != 1 {
		t.Fatalf("scheduler did not reserve before existing queue: %+v %v", state, err)
	}
	cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	cfg.FiniteLeader.Retired = true
	if err := f.teams.SetHeartbeatConfig(context.Background(), "committee", "lead", cfg); err != nil {
		t.Fatal(err)
	}
	// Exercise the actual recovered queue callback, and join it deterministically.
	done := make(chan struct{})
	queue.GetOrCreate("committee").executor = finiteCompletionProbe{executor: e, done: done}
	queue.OnComplete("committee", "other")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("queue dispatch did not return")
	}
	if len(f.agent.createRunCalls) != 0 {
		t.Fatal("queued retirement launched a leader")
	}
}

type finiteCompletionProbe struct {
	executor HeartbeatExecutor
	done     chan struct{}
}

func (p finiteCompletionProbe) Execute(ctx context.Context, team, agent, profile string) (*ExecutionResult, error) {
	defer close(p.done)
	return p.executor.Execute(ctx, team, agent, profile)
}

func TestFiniteLeaderCompletionClosesQueuedDispatchAndSurvivesRestart(t *testing.T) {
	f := newFiniteFixture(t)
	ctx := context.Background()
	reserved, err := f.runtime.Tick(ctx, "committee", "lead")
	if err != nil || reserved.ID == "" || reserved.DispatchStarted {
		t.Fatalf("reservation not queued: %+v %v", reserved, err)
	}
	receipt, changed, err := f.runtime.Complete(ctx, "committee", "lead", "accepted:revision/7", "evidence:owner/launch/1")
	if err != nil || !changed || receipt == nil || receipt.ReceiptID == "" {
		t.Fatalf("completion refused: %+v %v %v", receipt, changed, err)
	}
	if again, changed, err := f.runtime.Complete(ctx, "committee", "lead", "accepted:revision/7", "evidence:owner/launch/1"); err != nil || changed || again.ReceiptID != receipt.ReceiptID {
		t.Fatalf("duplicate completion not idempotent: %+v %v %v", again, changed, err)
	}
	if _, err := f.runtime.Dispatch(ctx, "committee", "lead"); !errors.Is(err, store.ErrFiniteLeaderCompleted) {
		t.Fatalf("manual trigger reopened completed work: %v", err)
	}
	// Lose ephemeral runtime state; the durable receipt must still refuse a tick.
	restarted := &FiniteLeaderRuntime{Executor: f.runtime.Executor, Queue: &effortQueueFake{}}
	if _, err := restarted.Tick(ctx, "committee", "lead"); !errors.Is(err, store.ErrFiniteLeaderCompleted) {
		t.Fatalf("hourly tick reopened completed work: %v", err)
	}
	if len(f.agent.createTaskCalls) != 0 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("completed effort bought owner inference")
	}
	state, err := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if err != nil || state.Completed == nil || state.Completed.ReceiptID != receipt.ReceiptID {
		t.Fatalf("completion receipt not durable: %+v %v", state, err)
	}
	// Only the explicit authorized reopen returns the effort to scheduling.
	if err := restarted.Reopen(ctx, "committee", "lead", "accepted:revision/8", "evidence:owner/launch/2"); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Tick(ctx, "committee", "lead"); err != nil {
		t.Fatalf("reopened effort could not resume: %v", err)
	}
}

func TestFiniteLeaderCompletionTransportIsExplicitAndRevisionChecked(t *testing.T) {
	f := newFiniteFixture(t)
	h := NewHandlers(HandlersDeps{TeamStore: f.teams, RelationStore: f.relations, Executor: f.runtime.Executor})
	put := func(body string) *httptest.ResponseRecorder {
		req := mux.SetURLVars(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body)), map[string]string{"id": "committee", "agentId": "lead"})
		w := httptest.NewRecorder()
		h.UpdateHeartbeat(w, req)
		return w
	}
	const completion = `{"finiteEffortTransition":{"operation":"complete","revision":"accepted:revision/7","evidenceRef":"evidence:owner/1"}}`
	if w := put(`{"finiteEffortTransition":{"operation":"complete","revision":"accepted:revision/7","evidenceRef":"evidence:owner/1"},"enabled":true}`); w.Code != http.StatusBadRequest {
		t.Fatalf("combined transition/config accepted: %d %s", w.Code, w.Body.String())
	}
	if w := put(`{"finiteEffortTransition":{"operation":"complete","revision":"accepted:revision/9","evidenceRef":"evidence:owner/1"}}`); w.Code != http.StatusConflict {
		t.Fatalf("stale revision accepted: %d %s", w.Code, w.Body.String())
	}
	if w := put(completion); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"completed"`) {
		t.Fatalf("completion not recorded: %d %s", w.Code, w.Body.String())
	}
	if w := put(completion); w.Code != http.StatusOK {
		t.Fatalf("idempotent completion refused: %d %s", w.Code, w.Body.String())
	}
	if w := put(`{"finiteEffortTransition":{"operation":"complete","revision":"accepted:revision/7","evidenceRef":"evidence:owner/2"}}`); w.Code != http.StatusConflict {
		t.Fatalf("conflicting completion accepted: %d %s", w.Code, w.Body.String())
	}
	if w := put(`{"finiteEffortTransition":{"operation":"reopen","revision":"accepted:revision/7","evidenceRef":"evidence:owner/3"}}`); w.Code != http.StatusConflict {
		t.Fatalf("reopen with the completed revision accepted: %d %s", w.Code, w.Body.String())
	}
	if w := put(`{"finiteEffortTransition":{"operation":"reopen","revision":"accepted:revision/8","evidenceRef":"evidence:owner/3"}}`); w.Code != http.StatusOK {
		t.Fatalf("reopen refused: %d %s", w.Code, w.Body.String())
	}
	state, err := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if err != nil || state.Completed != nil || len(state.CompletionHistory) != 1 {
		t.Fatalf("reopen did not retain prior receipt: %+v %v", state, err)
	}
	if w := put(`{"finiteEffortTransition":{"operation":"reopen","revision":"accepted:revision/9","evidenceRef":"evidence:owner/4"}}`); w.Code != http.StatusConflict {
		t.Fatalf("reopen of a non-completed effort accepted: %d %s", w.Code, w.Body.String())
	}
}

func TestFiniteLeaderCompletionRetainsActiveRunAccounting(t *testing.T) {
	f := newFiniteFixture(t)
	ctx := context.Background()
	state := f.dispatch(t)
	if _, _, err := f.runtime.Complete(ctx, "committee", "lead", "accepted:revision/7", "evidence:owner/launch/1"); err != nil {
		t.Fatal(err)
	}
	run := f.agent.getRuns[state.RunID]
	run.Status, run.StartedAt, run.EndedAt = "complete", "2026-09-12T10:00:00Z", "2026-09-12T10:05:00Z"
	if _, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil {
		t.Fatalf("completed active run was not settled: %v", err)
	}
	cfg, _ := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
	if cfg.LastExecution == nil || cfg.LastExecution.RunID != state.RunID {
		t.Fatalf("settled accounting lost: %+v", cfg.LastExecution)
	}
	if len(f.agent.createRunCalls) != 1 {
		t.Fatal("completion dispatched a replacement run")
	}
	retained, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if retained.Completed == nil || retained.RunID != state.RunID {
		t.Fatalf("completion receipt or owner identity lost after settling: %+v", retained)
	}
}

func TestFiniteLeaderRestartRequiresTerminalOwnerAndPreservesHistory(t *testing.T) {
	f := newFiniteFixture(t)
	ctx := context.Background()
	state := f.dispatch(t)
	if err := f.runtime.Restart(ctx, "committee", "lead", "accepted:revision/7", "evidence:owner/terminal"); err == nil {
		t.Fatal("nonterminal owner run allowed a replacement")
	}
	f.agent.getRuns[state.RunID].Status = "complete"
	if err := f.runtime.Restart(ctx, "committee", "lead", "accepted:revision/7", "evidence:owner/terminal"); err != nil {
		t.Fatalf("terminal owner restart refused: %v", err)
	}
	retained, err := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if err != nil || retained.DispatchStarted || retained.RunID != "" || len(retained.RestartHistory) != 1 {
		t.Fatalf("restart did not clear reusable reservation or retain history: %+v %v", retained, err)
	}
	if retained.RestartHistory[0].RunID != state.RunID || retained.RestartHistory[0].EvidenceRef != "evidence:owner/terminal" {
		t.Fatalf("restart history lost prior identity/evidence: %+v", retained.RestartHistory)
	}
	if _, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil {
		t.Fatalf("fresh reservation after restart refused: %v", err)
	}
	if _, err := f.runtime.Executor.Execute(ctx, "committee", "lead", "ignored"); err != nil {
		t.Fatalf("replacement reservation did not dispatch: %v", err)
	}
	if len(f.agent.createRunCalls) != 2 {
		t.Fatalf("restart did not permit exactly one replacement run: %d", len(f.agent.createRunCalls))
	}
}
