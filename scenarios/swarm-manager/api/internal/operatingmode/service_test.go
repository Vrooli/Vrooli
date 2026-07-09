package operatingmode

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/promptmanager"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
)

type fakeInitiatives struct {
	items map[string]InitiativeSnapshot
}

func (f fakeInitiatives) LoadInitiative(name string) (InitiativeSnapshot, error) {
	item, ok := f.items[name]
	if !ok {
		return InitiativeSnapshot{}, fmt.Errorf("initiative %q not found", name)
	}
	return item, nil
}

func (f fakeInitiatives) ListInitiatives() ([]InitiativeSummary, error) {
	out := make([]InitiativeSummary, 0, len(f.items))
	for _, item := range f.items {
		out = append(out, InitiativeSummary{
			Name:   item.Name,
			Title:  item.Title,
			Mode:   item.Mode,
			Status: "active",
		})
	}
	return out, nil
}

type fakeBacklog struct {
	items map[string]BacklogItemSnapshot
}

func (f fakeBacklog) LoadBacklogItem(kind, name string) (BacklogItemSnapshot, error) {
	item, ok := f.items[kind+"/"+name]
	if !ok {
		return BacklogItemSnapshot{}, fmt.Errorf("backlog item %q/%q not found", kind, name)
	}
	return item, nil
}

type fakeBacklogMutator struct {
	completed []string
	sources   []BacklogMutationSource
}

func (f *fakeBacklogMutator) MarkBacklogItemCompleted(_ context.Context, kind, name string, source BacklogMutationSource) (BacklogCompletionResult, error) {
	ref := kind + "/" + name
	f.completed = append(f.completed, ref+"@"+source.Entrypoint)
	f.sources = append(f.sources, source)
	return BacklogCompletionResult{ItemRef: ref, FromStatus: "ready", ToStatus: "completed"}, nil
}

type fakeProposalReconciler struct {
	req ProposalReconcileRequest
}

func (f *fakeProposalReconciler) ApplyBacklogSyncProposal(_ context.Context, req ProposalReconcileRequest) (*ProposalApplyResult, error) {
	f.req = req
	return &ProposalApplyResult{
		Outcomes: []ProposalOutcome{{
			MutationID: "m1",
			Op:         "add_item",
			Target:     "fix/follow-up",
			Applied:    true,
		}},
		Applied: 1,
		Created: 1,
	}, nil
}

type fakePlanExecution struct {
	resumeReqs []*executionv1.ResumeRequest
}

func (f *fakePlanExecution) Resume(_ context.Context, req *executionv1.ResumeRequest) (*executionv1.ResumeResponse, error) {
	f.resumeReqs = append(f.resumeReqs, req)
	return &executionv1.ResumeResponse{
		Execution: &executionv1.Execution{
			Id:             "exec-1",
			PlanId:         req.GetPlanOrExecution(),
			CurrentPhaseId: "phase-1",
		},
		Context: &executionv1.PhaseContext{ResumePhaseId: "phase-1"},
	}, nil
}

func (f *fakePlanExecution) GetNext(context.Context, *executionv1.GetNextRequest) (*executionv1.GetNextResponse, error) {
	return &executionv1.GetNextResponse{}, nil
}

func (f *fakePlanExecution) GetStatus(context.Context, *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error) {
	return &executionv1.GetStatusResponse{}, nil
}

type fakeModeUpdater struct {
	updates []string
	items   map[string]InitiativeSnapshot
}

func (f *fakeModeUpdater) UpdateInitiativeMode(name, mode string) (InitiativeSnapshot, error) {
	f.updates = append(f.updates, name+"="+mode)
	item, ok := f.items[name]
	if !ok {
		return InitiativeSnapshot{}, fmt.Errorf("initiative %q not found", name)
	}
	item.Mode = mode
	f.items[name] = item
	return item, nil
}

type fakePlanRefBinder struct {
	refs map[string]PlanRef
}

func (f *fakePlanRefBinder) BindInitiativePlanRef(name string, ref PlanRef) (InitiativeSnapshot, error) {
	if f.refs == nil {
		f.refs = map[string]PlanRef{}
	}
	f.refs[name] = ref
	return InitiativeSnapshot{Name: name, Mode: string(ModePhasedPlanDrain), PlanRef: &ref}, nil
}

type fakeItemExecutions struct {
	active   []ActiveItemExecution
	canceled []ActiveItemExecution
}

func (f *fakeItemExecutions) ActiveExecutionsForInitiative(context.Context, InitiativeSnapshot) ([]ActiveItemExecution, error) {
	return append([]ActiveItemExecution(nil), f.active...), nil
}

func (f *fakeItemExecutions) CancelActiveExecutionsForInitiative(context.Context, InitiativeSnapshot) ([]ActiveItemExecution, error) {
	if len(f.canceled) == 0 {
		f.canceled = append([]ActiveItemExecution(nil), f.active...)
		for i := range f.canceled {
			f.canceled[i].Status = "canceled"
		}
	}
	return append([]ActiveItemExecution(nil), f.canceled...), nil
}

type fakeAgent struct {
	spawned  []agentmanager.InitiativeSpawnRequest
	states   map[string]agentmanager.RunState
	messages map[string][]string
	stopped  []string
	err      error
}

// GetRunMessages satisfies the optional RunMessageReader capability so the
// resolution ladder's L0 true-final-message detection can be exercised at the
// refresher level. A run with no configured messages returns nil, and the
// refresher falls back to the run summary as the sole candidate.
func (f *fakeAgent) GetRunMessages(_ context.Context, runID string) ([]string, error) {
	return f.messages[runID], nil
}

func (f *fakeAgent) SpawnInitiative(_ context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
	if f.err != nil {
		return agentmanager.RunResult{}, f.err
	}
	f.spawned = append(f.spawned, req)
	runID := fmt.Sprintf("run-%d", len(f.spawned))
	return agentmanager.RunResult{RunID: runID, TaskID: "task-" + runID}, nil
}

func (f *fakeAgent) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	if state, ok := f.states[runID]; ok {
		return state, nil
	}
	return agentmanager.RunState{RunID: runID, Status: "running"}, nil
}

func (f *fakeAgent) StopRun(_ context.Context, runID string) error {
	f.stopped = append(f.stopped, runID)
	return nil
}

type fakePrompts struct {
	calls []string
	err   error
	text  string
	// render, when set, deterministically substitutes variables so tests can
	// assert fixture data reaches the prompt. It also lets the parity test prove
	// the spawn path and preview path pass an identical variable map.
	render func(skillID string, variables map[string]string) string
}

func (f *fakePrompts) ReadSkill(_ context.Context, skillID string, variables map[string]string, _ bool) (string, error) {
	f.calls = append(f.calls, skillID)
	if f.err != nil {
		return "", f.err
	}
	if f.render != nil {
		return f.render(skillID, variables), nil
	}
	if f.text != "" {
		return f.text, nil
	}
	return "rendered " + skillID, nil
}

type failingOverrideLock struct {
	base           *initiativelock.Lock
	overrideCalls  int
	failOverrideAt int
	failErr        error
}

func (l *failingOverrideLock) Acquire(initiativeName string, holder initiativelock.Holder) error {
	return l.base.Acquire(initiativeName, holder)
}

func (l *failingOverrideLock) AcquireOverride(initiativeName string, holder initiativelock.Holder) error {
	l.overrideCalls++
	if l.failOverrideAt > 0 && l.overrideCalls == l.failOverrideAt {
		return l.failErr
	}
	return l.base.AcquireOverride(initiativeName, holder)
}

func (l *failingOverrideLock) Release(initiativeName, runID string) error {
	return l.base.Release(initiativeName, runID)
}

func (l *failingOverrideLock) Inspect(initiativeName string) (*initiativelock.Holder, error) {
	return l.base.Inspect(initiativeName)
}

func (f *fakePrompts) ReadSkillWithExperiment(context.Context, string, map[string]string, bool, string) (promptmanager.ReadSkillResult, error) {
	return promptmanager.ReadSkillResult{}, fmt.Errorf("not implemented")
}

func TestStartPhaseCreatesRoundLocksAndSpawnsWithProfile(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	prompts := &fakePrompts{}
	svc := newTestService(t, root, agent, prompts)

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
		RequestedBy:    "tester",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	if round.Round != 1 || round.Status != RoundStatusAgentRunning || round.RunID != "run-1" {
		t.Fatalf("round = %+v, want round 1 running with run-1", round)
	}
	if round.AgentProfileKey != ProfileDeepWork {
		t.Fatalf("profile = %q, want %q", round.AgentProfileKey, ProfileDeepWork)
	}
	if len(agent.spawned) != 1 {
		t.Fatalf("spawn count = %d, want 1", len(agent.spawned))
	}
	spawn := agent.spawned[0]
	if spawn.ProfileKey != ProfileDeepWork || spawn.Purpose != "holistic_loop_investigate" {
		t.Fatalf("spawn profile/purpose = %q/%q", spawn.ProfileKey, spawn.Purpose)
	}
	if len(prompts.calls) != 1 || prompts.calls[0] != "swarm-manager-holistic-loop-investigate" {
		t.Fatalf("prompt calls = %#v", prompts.calls)
	}
	holder, err := svc.lock.Inspect("init-a")
	if err != nil {
		t.Fatalf("inspect lock: %v", err)
	}
	if holder == nil || holder.RunID != "run-1" || holder.Purpose != "holistic_loop_investigate" {
		t.Fatalf("holder = %+v, want run-1 holistic_loop_investigate", holder)
	}
}

func TestStartPhaseRejectsInvalidFirstPhase(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "review",
	})
	if err == nil || !strings.Contains(err.Error(), `first phase must be "investigate"`) {
		t.Fatalf("StartPhase review first error = %v, want start phase error", err)
	}
	if len(agent.spawned) != 0 {
		t.Fatalf("spawned = %d, want no spawn for invalid phase", len(agent.spawned))
	}
	rounds, listErr := svc.store.ListRounds("init-a", ModeHolisticLoop)
	if listErr != nil {
		t.Fatalf("ListRounds: %v", listErr)
	}
	if len(rounds) != 0 {
		t.Fatalf("rounds = %+v, want no reserved round for invalid phase", rounds)
	}
}

func TestStartPhaseFollowsHolisticLoopTransitions(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "execute",
	})
	if err == nil || !strings.Contains(err.Error(), `does not transition to "execute"`) {
		t.Fatalf("StartPhase execute after investigate error = %v, want transition error", err)
	}

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "plan",
	})
	if err != nil {
		t.Fatalf("StartPhase plan after investigate returned error: %v", err)
	}
	if round.Phase != "plan" {
		t.Fatalf("round phase = %q, want plan", round.Phase)
	}
}

func TestWorkspaceExposesBackendPhaseActions(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)

	workspace, err := svc.Workspace(context.Background(), "init-a")
	if err != nil {
		t.Fatalf("Workspace returned error: %v", err)
	}
	actions := map[string]WorkspacePhase{}
	for _, phase := range workspace.Definition.Phases {
		actions[phase.Phase] = phase
	}
	if !actions["plan"].Startable || !actions["plan"].Next {
		t.Fatalf("plan action = %+v, want startable next phase", actions["plan"])
	}
	if actions["execute"].Startable || !strings.Contains(actions["execute"].Reason, "investigate") {
		t.Fatalf("execute action = %+v, want disabled with transition reason", actions["execute"])
	}
	capabilities := workspace.Definition.Capabilities
	if !capabilities.CanStartPhases || !capabilities.CanCompleteItems || !capabilities.CanApplyBacklogSyncProposals {
		t.Fatalf("workspace capabilities = %+v, want phase and backlog sync actions", capabilities)
	}
	if !capabilities.RequiresAcceptanceCriteria || !capabilities.SupportsArtifacts {
		t.Fatalf("workspace capabilities = %+v, want criteria and artifact support", capabilities)
	}
}

func TestStartPhaseUsesReplanSignalForHolisticExecuteNextPhase(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "plan", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "execute", map[string]any{"replan_needed": true})

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "review",
	})
	if err == nil || !strings.Contains(err.Error(), `does not transition to "review"`) {
		t.Fatalf("StartPhase review after replan execute error = %v, want transition error", err)
	}

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase investigate after replan execute returned error: %v", err)
	}
	if round.Phase != "investigate" {
		t.Fatalf("round phase = %q, want investigate", round.Phase)
	}
}

func TestStartPhaseFollowsPhasedPlanProgressTransitions(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-phased": {
				Name:               "init-phased",
				Title:              "Init Phased",
				Description:        "Test phased initiative",
				Mode:               string(ModePhasedPlanDrain),
				PlanRef:            testOperatingModePlanRef(),
				Items:              []string{"execute/do-thing"},
				AcceptanceCriteria: []string{"works end to end"},
			},
		}},
		planExecution: &fakePlanExecution{},
	})

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-phased",
		Phase:          "execute_next",
	})
	if err == nil || !strings.Contains(err.Error(), `first phase must be "prepare_plan"`) {
		t.Fatalf("StartPhase execute_next first error = %v, want start phase error", err)
	}

	saveCompletedRoundWithHandoff(t, svc, "init-phased", ModePhasedPlanDrain, "prepare_plan", nil)
	saveCompletedRoundWithHandoff(t, svc, "init-phased", ModePhasedPlanDrain, "execute_next", nil)
	saveCompletedRoundWithHandoff(t, svc, "init-phased", ModePhasedPlanDrain, "classify_progress", map[string]any{
		"progress": ProgressState{Decision: ProgressContinue},
	})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-phased",
		Phase:          "execute_next",
	})
	if err != nil {
		t.Fatalf("StartPhase execute_next after continue returned error: %v", err)
	}
	if round.Phase != "execute_next" {
		t.Fatalf("round phase = %q, want execute_next", round.Phase)
	}
}

func TestStartPhaseFailsClosedWhenPromptRenderFails(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{err: errors.New("prompt-manager unavailable")})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err == nil || !strings.Contains(err.Error(), "render operating-mode prompt") {
		t.Fatalf("StartPhase error = %v, want prompt render failure", err)
	}
	if len(agent.spawned) != 0 {
		t.Fatalf("spawned = %d, want no spawn when prompt rendering fails", len(agent.spawned))
	}
	loaded, loadErr := svc.store.LoadRound("init-a", ModeHolisticLoop, round.Round)
	if loadErr != nil {
		t.Fatalf("LoadRound: %v", loadErr)
	}
	if loaded.Status != RoundStatusFailed || !strings.Contains(loaded.Error, "prompt-manager unavailable") {
		t.Fatalf("round = %+v, want failed prompt round", loaded)
	}
	holder, inspectErr := svc.lock.Inspect("init-a")
	if inspectErr != nil {
		t.Fatalf("Inspect lock: %v", inspectErr)
	}
	if holder != nil {
		t.Fatalf("lock holder = %+v, want nil after prompt failure", holder)
	}
}

func TestStartPhaseLockConflictLeavesNoActiveReservedRound(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})
	if err := svc.lock.Acquire("init-a", initiativelock.Holder{
		RunID:   "run-existing",
		Purpose: "holistic_loop_investigate",
	}); err != nil {
		t.Fatalf("seed lock: %v", err)
	}

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err == nil {
		t.Fatal("StartPhase lock conflict error = nil, want error")
	}
	if len(agent.spawned) != 0 {
		t.Fatalf("spawned = %d, want no spawn when lock is held", len(agent.spawned))
	}
	rounds, listErr := svc.store.ListRounds("init-a", ModeHolisticLoop)
	if listErr != nil {
		t.Fatalf("ListRounds: %v", listErr)
	}
	for _, round := range rounds {
		if isRoundActive(round) {
			t.Fatalf("round = %+v, want no active round after lock conflict", round)
		}
	}
}

func TestStartPhaseSpawnFailurePersistsFailedRoundAndReleasesLock(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{err: errors.New("agent-manager unavailable")}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err == nil || !strings.Contains(err.Error(), "spawn operating-mode phase") {
		t.Fatalf("StartPhase error = %v, want spawn failure", err)
	}
	loaded, loadErr := svc.store.LoadRound("init-a", ModeHolisticLoop, round.Round)
	if loadErr != nil {
		t.Fatalf("LoadRound: %v", loadErr)
	}
	if loaded.Status != RoundStatusFailed || !strings.Contains(loaded.Error, "agent-manager unavailable") {
		t.Fatalf("round = %+v, want failed spawn round", loaded)
	}
	holder, inspectErr := svc.lock.Inspect("init-a")
	if inspectErr != nil {
		t.Fatalf("Inspect lock: %v", inspectErr)
	}
	if holder != nil {
		t.Fatalf("lock holder = %+v, want nil after spawn failure", holder)
	}
}

func TestStartPhaseLockSwapFailureFailsRoundStopsRunAndReleasesLock(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	baseLock := &initiativelock.Lock{
		Dir: func(name string) string {
			return filepath.Join(root, "initiatives", name)
		},
		Clock: func() time.Time {
			return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		},
	}
	lock := &failingOverrideLock{
		base:           baseLock,
		failOverrideAt: 1,
		failErr:        errors.New("disk write refused"),
	}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		agent: agent,
		lock:  lock,
	})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err == nil || !strings.Contains(err.Error(), "swap operating-mode lock holder") {
		t.Fatalf("StartPhase error = %v, want lock swap failure", err)
	}
	loaded, loadErr := svc.store.LoadRound("init-a", ModeHolisticLoop, round.Round)
	if loadErr != nil {
		t.Fatalf("LoadRound: %v", loadErr)
	}
	if loaded.Status != RoundStatusFailed || !strings.Contains(loaded.Error, "disk write refused") {
		t.Fatalf("round = %+v, want failed lock-swap round", loaded)
	}
	if loaded.RunID != "run-1" {
		t.Fatalf("round run id = %q, want spawned run id retained for audit", loaded.RunID)
	}
	if len(agent.stopped) != 1 || agent.stopped[0] != "run-1" {
		t.Fatalf("stopped runs = %v, want run-1 stopped after lock swap failure", agent.stopped)
	}
	holder, inspectErr := svc.lock.Inspect("init-a")
	if inspectErr != nil {
		t.Fatalf("Inspect lock: %v", inspectErr)
	}
	if holder != nil {
		t.Fatalf("lock holder = %+v, want nil after lock swap failure", holder)
	}
}

func TestWorkspaceRefreshesCompletedRoundAndReleasesLock(t *testing.T) {
	root := t.TempDir()
	summary := `{"operating_mode_result":{"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"# Findings"}],"handoff":{"summary":"investigated"}}}`
	agent := &fakeAgent{states: map[string]agentmanager.RunState{
		"run-1": {RunID: "run-1", Status: "complete", Summary: summary, FinishedAt: "2026-04-30T12:05:00Z"},
	}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	workspace, err := svc.Workspace(context.Background(), "init-a")
	if err != nil {
		t.Fatalf("Workspace returned error: %v", err)
	}
	if len(workspace.Rounds) != 1 || workspace.Rounds[0].Status != RoundStatusCompleted {
		t.Fatalf("workspace rounds = %+v, want one completed round", workspace.Rounds)
	}
	loaded, err := svc.store.LoadRound("init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("load round: %v", err)
	}
	if loaded.Status != RoundStatusCompleted {
		t.Fatalf("persisted round status = %q, want completed", loaded.Status)
	}
	holder, err := svc.lock.Inspect("init-a")
	if err != nil {
		t.Fatalf("inspect lock: %v", err)
	}
	if holder != nil {
		t.Fatalf("lock holder = %+v, want nil after completed refresh", holder)
	}
}

func TestRefreshRoundFailsWhenStructuredResultMissing(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{
		"run-1": {RunID: "run-1", Status: "complete", Summary: "plain human summary", FinishedAt: "2026-04-30T12:05:00Z"},
	}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	// A plain human summary with no structured envelope, no message stream, and
	// no classifier is an honest abstain: the ladder resolved nothing, so the
	// round stops safely with an abstain diagnostic rather than the old terse
	// parse error, and carries a durable resolution record marked abstained.
	if refreshed.Status != RoundStatusFailed || !strings.Contains(refreshed.Error, "resolution abstained") {
		t.Fatalf("refreshed = %+v, want abstained resolution", refreshed)
	}
	record, ok := RoundPayload(refreshed.Payload).Resolution()
	if !ok || record.Outcome != ResolutionAbstained {
		t.Fatalf("resolution record = %+v (ok=%v), want abstained", record, ok)
	}
}

func TestApplyPhaseResultUsesInjectedClassifier(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		classifier: &stubClassifier{answers: map[string]string{"verdict": "accepted"}},
	})
	round := RoundEnvelope{
		Mode:           string(ModeHolisticLoop),
		Phase:          "review",
		InitiativeName: "init-a",
		Payload:        map[string]any{},
	}
	def := MustDefinition(ModeHolisticLoop)
	resolved, err := svc.applyPhaseResultInMemory(context.Background(), def, &round, "After careful review I accept the work; it meets the acceptance criteria.")
	if err != nil {
		t.Fatalf("applyPhaseResultInMemory returned error: %v", err)
	}
	if resolved.Outcome != ResolutionRecovered || resolved.Layer != ResolutionLayerClassifier {
		t.Fatalf("resolved = %+v, want recovered via L2 classifier", resolved)
	}
	if got := RoundPayload(round.Payload).Verdict(); got != "accepted" {
		t.Fatalf("applied verdict = %q, want accepted", got)
	}
}

func TestRefreshRoundRecoversSubagentTailFromMessageStream(t *testing.T) {
	root := t.TempDir()
	final := `{"operating_mode_result":{"handoff":{"summary":"investigated"},"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"# Findings"}]}}`
	trailing := "[subagent] cleanup complete, 2 files touched"
	agent := &fakeAgent{
		states: map[string]agentmanager.RunState{
			// The run summary echoes the trailing subagent message, not the real
			// answer — the classic true-final-message problem.
			"run-1": {RunID: "run-1", Status: "complete", Summary: trailing, FinishedAt: "2026-04-30T12:05:00Z"},
		},
		messages: map[string][]string{
			"run-1": {"Investigating the initiative…", final, trailing},
		},
	}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("status = %q (err=%q), want completed via true-final-message recovery", refreshed.Status, refreshed.Error)
	}
	record, ok := RoundPayload(refreshed.Payload).Resolution()
	if !ok || record.Outcome != ResolutionRecovered || record.Layer != ResolutionLayerFinalMsg {
		t.Fatalf("resolution record = %+v (ok=%v), want recovered via L0", record, ok)
	}
	if _, err := svc.store.ReadArtifact("init-a", ModeHolisticLoop, "modes/holistic-loop/findings.md"); err != nil {
		t.Fatalf("recovered artifact not written: %v", err)
	}
}

func TestRefreshRoundFailsWhenRequiredArtifactMissing(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{
		"run-1": {RunID: "run-1", Status: "complete", Summary: `{"operating_mode_result":{"handoff":{"summary":"investigated"}}}`, FinishedAt: "2026-04-30T12:05:00Z"},
	}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	if refreshed.Status != RoundStatusFailed || !strings.Contains(refreshed.Error, `requires artifact "modes/holistic-loop/findings.md"`) {
		t.Fatalf("refreshed = %+v, want failed artifact contract", refreshed)
	}
}

func TestRefreshRoundStagesArtifactsBeforeApplyingResult(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{
		"run-1": {
			RunID:      "run-1",
			Status:     "complete",
			Summary:    `{"operating_mode_result":{"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"# Findings"},{"path":"../outside.md","content":"bad"}],"handoff":{"summary":"investigated"}}}`,
			FinishedAt: "2026-04-30T12:05:00Z",
		},
	}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	if refreshed.Status != RoundStatusFailed || !strings.Contains(refreshed.Error, "artifact path must be relative to initiative") {
		t.Fatalf("refreshed = %+v, want failed artifact staging contract", refreshed)
	}
	if _, err := svc.store.ReadArtifact("init-a", ModeHolisticLoop, "modes/holistic-loop/findings.md"); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("ReadArtifact error = %v, want ErrArtifactNotFound because writes are staged", err)
	}
}

func TestRefreshRoundRequiresProgressForClassifyProgress(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		agent: &fakeAgent{states: map[string]agentmanager.RunState{
			"run-1": {RunID: "run-1", Status: "complete", Summary: `{"operating_mode_result":{"handoff":{"summary":"classified"}}}`, FinishedAt: "2026-04-30T12:05:00Z"},
		}},
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-phased": {
				Name:               "init-phased",
				Title:              "Init Phased",
				Description:        "Test phased initiative",
				Mode:               string(ModePhasedPlanDrain),
				PlanRef:            testOperatingModePlanRef(),
				Items:              []string{"execute/do-thing"},
				AcceptanceCriteria: []string{"works end to end"},
			},
		}},
		planExecution: &fakePlanExecution{},
	})
	saveCompletedRoundWithHandoff(t, svc, "init-phased", ModePhasedPlanDrain, "prepare_plan", nil)
	saveCompletedRoundWithHandoff(t, svc, "init-phased", ModePhasedPlanDrain, "execute_next", nil)

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-phased",
		Phase:          "classify_progress",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-phased", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	// classify_progress declares progress + progress.decision required. A handoff-
	// only envelope resolves nothing for that contract, so the ladder abstains
	// (safe stop) with the unresolved declared fields named, and — with no message
	// stream or classifier — the round fails on the abstain rather than the old
	// bespoke "requires a valid progress decision" contract check.
	if refreshed.Status != RoundStatusFailed || !strings.Contains(refreshed.Error, "resolution abstained") {
		t.Fatalf("refreshed = %+v, want abstained progress resolution", refreshed)
	}
	if record, ok := RoundPayload(refreshed.Payload).Resolution(); !ok || record.Outcome != ResolutionAbstained {
		t.Fatalf("resolution record = %+v (ok=%v), want abstained", record, ok)
	}
}

func TestRefreshRoundCapturesPhasedPlanProgressWithoutArtifact(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		agent: &fakeAgent{states: map[string]agentmanager.RunState{
			"run-1": {
				RunID:      "run-1",
				Status:     "complete",
				Summary:    `{"operating_mode_result":{"progress":{"decision":"continue","completed_phases":["phase-1"],"current_phase":"phase-2","rationale":"keep going"}}}`,
				FinishedAt: "2026-04-30T12:05:00Z",
			},
		}},
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-phased": {
				Name:               "init-phased",
				Title:              "Init Phased",
				Description:        "Test phased initiative",
				Mode:               string(ModePhasedPlanDrain),
				PlanRef:            testOperatingModePlanRef(),
				Items:              []string{"execute/do-thing"},
				AcceptanceCriteria: []string{"works end to end"},
			},
		}},
		planExecution: &fakePlanExecution{},
	})
	saveCompletedRoundWithHandoff(t, svc, "init-phased", ModePhasedPlanDrain, "prepare_plan", nil)
	saveCompletedRoundWithHandoff(t, svc, "init-phased", ModePhasedPlanDrain, "execute_next", nil)

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-phased",
		Phase:          "classify_progress",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-phased", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("refreshed status = %q, want completed: %+v", refreshed.Status, refreshed)
	}
	if len(refreshed.ArtifactUpdates) != 0 {
		t.Fatalf("artifact updates = %+v", refreshed.ArtifactUpdates)
	}
	state, ok := RoundPayload(refreshed.Payload).Progress()
	if !ok || state.Decision != ProgressContinue || state.UpdatedAt == "" {
		t.Fatalf("progress state = %+v", state)
	}
}

func TestRefreshRoundBindsPreparePlanRef(t *testing.T) {
	root := t.TempDir()
	binder := &fakePlanRefBinder{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		agent: &fakeAgent{states: map[string]agentmanager.RunState{
			"run-1": {
				RunID:      "run-1",
				Status:     "complete",
				Summary:    `{"operating_mode_result":{"plan_ref":{"provider":"plan-manager","plan_id":"plan-123","slug":"initiative-plan","role":"operating_mode_plan"},"handoff":{"summary":"plan ready"}}}`,
				FinishedAt: "2026-04-30T12:05:00Z",
			},
		}},
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-phased": {
				Name:               "init-phased",
				Title:              "Init Phased",
				Description:        "Test phased initiative",
				Mode:               string(ModePhasedPlanDrain),
				Items:              []string{"execute/do-thing"},
				AcceptanceCriteria: []string{"works end to end"},
			},
		}},
		planRefBinder: binder,
	})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-phased",
		Phase:          "prepare_plan",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-phased", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("refreshed status = %q, want completed: %+v", refreshed.Status, refreshed)
	}
	if got := binder.refs["init-phased"].PlanID; got != "plan-123" {
		t.Fatalf("bound plan id = %q, want plan-123", got)
	}
	if got, _ := RoundPayload(refreshed.Payload).get(payloadPlanRef); got == nil {
		t.Fatalf("round payload missing plan_ref: %+v", refreshed.Payload)
	}
}

func TestRefreshRoundAppliesStructuredPhaseResult(t *testing.T) {
	root := t.TempDir()
	summary := "done\n```json\n{\"operating_mode_result\":{\"artifacts\":[{\"path\":\"modes/holistic-loop/findings.md\",\"content\":\"# Findings\\nCurrent state.\"}],\"handoff\":{\"summary\":\"investigated\"},\"verdict\":\"accepted\",\"replan_needed\":true}}\n```"
	agent := &fakeAgent{states: map[string]agentmanager.RunState{
		"run-1": {RunID: "run-1", Status: "complete", Summary: summary, FinishedAt: "2026-04-30T12:05:00Z"},
	}}
	svc := newTestService(t, root, agent, &fakePrompts{})

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}
	if len(refreshed.ArtifactUpdates) != 1 || refreshed.ArtifactUpdates[0].Path != "modes/holistic-loop/findings.md" {
		t.Fatalf("artifact updates = %+v", refreshed.ArtifactUpdates)
	}
	if len(refreshed.Handoffs) != 1 || refreshed.Handoffs[0].Summary != "investigated" {
		t.Fatalf("handoffs = %+v", refreshed.Handoffs)
	}
	artifact, err := svc.store.ReadArtifact("init-a", ModeHolisticLoop, "modes/holistic-loop/findings.md")
	if err != nil {
		t.Fatalf("ReadArtifact: %v", err)
	}
	if artifact.Content != "# Findings\nCurrent state." {
		t.Fatalf("artifact content = %q", artifact.Content)
	}
	if verdict, _ := refreshed.Payload["verdict"].(string); verdict != "accepted" {
		t.Fatalf("verdict payload = %q", verdict)
	}
}

func TestCompleteItemsRequiresRoundRunIDAndMembership(t *testing.T) {
	root := t.TempDir()
	mutator := &fakeBacklogMutator{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{backlogMutator: mutator})
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "plan", nil)
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "execute",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}

	result, err := svc.CompleteItems(context.Background(), CompleteItemsRequest{
		InitiativeName: "init-a",
		Mode:           string(ModeHolisticLoop),
		Round:          round.Round,
		RunID:          round.RunID,
		ItemRefs:       []string{"execute/do-thing"},
	})
	if err != nil {
		t.Fatalf("CompleteItems returned error: %v", err)
	}
	if len(result.CompletedItems) != 1 || result.CompletedItems[0].ItemRef != "execute/do-thing" {
		t.Fatalf("completed items = %+v", result.CompletedItems)
	}
	if got := fmt.Sprint(mutator.completed); got != "[execute/do-thing@initiative.operating_mode.complete_items]" {
		t.Fatalf("mutator completed = %s", got)
	}
	if len(mutator.sources) != 1 {
		t.Fatalf("mutation sources = %+v, want one", mutator.sources)
	}
	source := mutator.sources[0]
	if source.InitiativeName != "init-a" || source.Mode != string(ModeHolisticLoop) || source.Phase != "execute" || source.Round != round.Round || source.RunID != round.RunID || source.RequestedBy != "swarm-manager" {
		t.Fatalf("mutation source = %+v", source)
	}

	_, err = svc.CompleteItems(context.Background(), CompleteItemsRequest{
		InitiativeName: "init-a",
		Mode:           string(ModeHolisticLoop),
		Round:          round.Round,
		RunID:          "wrong-run",
		ItemRefs:       []string{"execute/do-thing"},
	})
	if err == nil {
		t.Fatal("CompleteItems wrong run error = nil, want error")
	}

	_, err = svc.CompleteItems(context.Background(), CompleteItemsRequest{
		InitiativeName: "init-a",
		Mode:           string(ModeHolisticLoop),
		Round:          round.Round,
		RunID:          round.RunID,
		ItemRefs:       []string{"execute/not-member"},
	})
	if err == nil {
		t.Fatal("CompleteItems non-member error = nil, want error")
	}
}

func TestApplyBacklogSyncAppliesProposalThroughProposalBoundary(t *testing.T) {
	root := t.TempDir()
	reconciler := &fakeProposalReconciler{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		reconciler: reconciler,
	})
	summary := `{"operating_mode_result":{"verdict":"accepted","backlog_sync":{"proposal":{"form":"mutation_list","mutations":[{"id":"m1","op":"add_item","item":{"kind":"fix","name":"follow-up","title":"Follow up"}}]}}}}`
	agent := svc.agent.(*fakeAgent)
	agent.states["run-1"] = agentmanager.RunState{RunID: "run-1", Status: "complete", Summary: summary, FinishedAt: "2026-04-30T12:05:00Z"}
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "plan", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "execute", map[string]any{"replan_needed": false})
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "review",
	})
	if err != nil {
		t.Fatalf("StartPhase returned error: %v", err)
	}
	round, err = svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound returned error: %v", err)
	}

	result, err := svc.ApplyBacklogSync(context.Background(), ApplyBacklogSyncRequest{
		InitiativeName:      "init-a",
		Mode:                string(ModeHolisticLoop),
		Round:               round.Round,
		RunID:               round.RunID,
		AcceptedMutationIDs: []string{"m1"},
		RequestedBy:         "tester",
	})
	if err != nil {
		t.Fatalf("ApplyBacklogSync returned error: %v", err)
	}
	if result.ProposalResult == nil || result.ProposalResult.Applied != 1 {
		t.Fatalf("proposal result = %+v", result.ProposalResult)
	}
	if !strings.Contains(string(reconciler.req.Proposal), `"add_item"`) {
		t.Fatalf("raw proposal = %s", string(reconciler.req.Proposal))
	}
	if got := fmt.Sprint(reconciler.req.AcceptedMutationIDs); got != "[m1]" {
		t.Fatalf("accepted = %s", got)
	}
	if reconciler.req.InitiativeName != "init-a" || reconciler.req.Mode != string(ModeHolisticLoop) || reconciler.req.Round != round.Round || reconciler.req.Phase != "review" || reconciler.req.RunID != round.RunID || reconciler.req.DecidedBy != "tester" {
		t.Fatalf("reconcile request = %+v", reconciler.req)
	}
	loaded, err := svc.store.LoadRound("init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("load round: %v", err)
	}
	if loaded.Payload["backlog_sync"] == nil {
		t.Fatalf("payload missing applied sync audit: %+v", loaded.Payload)
	}
}

func TestSwitchModeRequiresExplicitCancellationForActiveItemExecutions(t *testing.T) {
	root := t.TempDir()
	active := &fakeItemExecutions{active: []ActiveItemExecution{{
		ItemRef:     "execute/do-thing",
		ExecutionID: "exec-1",
		RunID:       "run-1",
		Status:      "running",
	}}}
	updater := &fakeModeUpdater{items: map[string]InitiativeSnapshot{
		"init-item": {
			Name:  "init-item",
			Title: "Init Item",
			Mode:  string(ModeItemLevel),
			Items: []string{"execute/do-thing"},
		},
	}}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-item": updater.items["init-item"],
		}},
		modeUpdater:    updater,
		itemExecutions: active,
	})

	_, err := svc.SwitchMode(context.Background(), SwitchModeRequest{
		InitiativeName: "init-item",
		Mode:           string(ModeHolisticLoop),
	})
	var conflict *ActiveItemExecutionsConflict
	if err == nil || !errors.As(err, &conflict) {
		t.Fatalf("SwitchMode error = %v, want ActiveItemExecutionsConflict", err)
	}
	if len(conflict.Executions) != 1 || conflict.Executions[0].ExecutionID != "exec-1" {
		t.Fatalf("conflict executions = %+v", conflict.Executions)
	}
	if len(updater.updates) != 0 {
		t.Fatalf("updates = %v, want none before cancellation confirmation", updater.updates)
	}
}

func TestSwitchModeCancelsActiveItemExecutionsThenUpdatesMode(t *testing.T) {
	root := t.TempDir()
	controller := &fakeItemExecutions{active: []ActiveItemExecution{{
		ItemRef:     "execute/do-thing",
		ExecutionID: "exec-1",
		RunID:       "run-1",
		Status:      "running",
	}}}
	updater := &fakeModeUpdater{items: map[string]InitiativeSnapshot{
		"init-item": {
			Name:  "init-item",
			Title: "Init Item",
			Mode:  string(ModeItemLevel),
			Items: []string{"execute/do-thing"},
		},
	}}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-item": updater.items["init-item"],
		}},
		modeUpdater:    updater,
		itemExecutions: controller,
	})

	result, err := svc.SwitchMode(context.Background(), SwitchModeRequest{
		InitiativeName:             "init-item",
		Mode:                       string(ModeHolisticLoop),
		CancelActiveItemExecutions: true,
	})
	if err != nil {
		t.Fatalf("SwitchMode returned error: %v", err)
	}
	if result.FromMode != string(ModeItemLevel) || result.ToMode != string(ModeHolisticLoop) {
		t.Fatalf("result modes = %q -> %q", result.FromMode, result.ToMode)
	}
	if len(result.CanceledItemExecutions) != 1 || result.CanceledItemExecutions[0].Status != "canceled" {
		t.Fatalf("canceled executions = %+v", result.CanceledItemExecutions)
	}
	if got := fmt.Sprint(updater.updates); got != "[init-item=holistic-loop]" {
		t.Fatalf("updates = %s", got)
	}
}

func TestSwitchModeRejectsActiveOperatingModeRound(t *testing.T) {
	root := t.TempDir()
	updater := &fakeModeUpdater{items: map[string]InitiativeSnapshot{
		"init-mode": {
			Name:  "init-mode",
			Title: "Init Mode",
			Mode:  string(ModeHolisticLoop),
		},
	}}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-mode": updater.items["init-mode"],
		}},
		modeUpdater: updater,
	})
	if _, err := svc.store.CreateRound(RoundEnvelope{
		Mode:           string(ModeHolisticLoop),
		InitiativeName: "init-mode",
		ScopeID:        "init-mode",
		Phase:          "investigate",
		Status:         RoundStatusAgentRunning,
		RunID:          "run-active",
	}); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	_, err := svc.SwitchMode(context.Background(), SwitchModeRequest{
		InitiativeName: "init-mode",
		Mode:           string(ModeItemLevel),
	})
	var conflict *ActiveOperatingModeRoundConflict
	if err == nil || !errors.As(err, &conflict) {
		t.Fatalf("SwitchMode error = %v, want ActiveOperatingModeRoundConflict", err)
	}
	if conflict.Round.RunID != "run-active" {
		t.Fatalf("conflict round = %+v, want run-active", conflict.Round)
	}
	if len(updater.updates) != 0 {
		t.Fatalf("updates = %v, want none while operating-mode round is active", updater.updates)
	}
}

func TestInitiativesUsingModeFiltersByMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"loop-a": {Name: "loop-a", Title: "Loop A", Mode: string(ModeHolisticLoop)},
			"loop-b": {Name: "loop-b", Title: "Loop B", Mode: string(ModeHolisticLoop)},
			"drain":  {Name: "drain", Title: "Drain", Mode: string(ModePhasedPlanDrain)},
			"item":   {Name: "item", Title: "Item", Mode: string(ModeItemLevel)},
		}},
	})

	loops, err := svc.InitiativesUsingMode(ModeHolisticLoop)
	if err != nil {
		t.Fatalf("InitiativesUsingMode: %v", err)
	}
	if len(loops) != 2 {
		t.Fatalf("holistic loops: got %d, want 2: %+v", len(loops), loops)
	}
	names := map[string]bool{}
	for _, ref := range loops {
		names[ref.Name] = true
	}
	if !names["loop-a"] || !names["loop-b"] {
		t.Fatalf("expected loop-a and loop-b in result; got %+v", loops)
	}

	drains, err := svc.InitiativesUsingMode(ModePhasedPlanDrain)
	if err != nil {
		t.Fatalf("InitiativesUsingMode drain: %v", err)
	}
	if len(drains) != 1 || drains[0].Name != "drain" {
		t.Fatalf("drain result mismatch: %+v", drains)
	}

	items, err := svc.InitiativesUsingMode(ModeItemLevel)
	if err != nil {
		t.Fatalf("InitiativesUsingMode item: %v", err)
	}
	if len(items) != 1 || items[0].Name != "item" {
		t.Fatalf("item result mismatch: %+v", items)
	}
}

func TestRegistryAllModesHaveDescription(t *testing.T) {
	for _, mode := range Modes() {
		def := MustDefinition(mode)
		if strings.TrimSpace(def.Description) == "" {
			t.Errorf("mode %q: empty description", mode)
		}
	}
}

func newTestService(t *testing.T, root string, agent *fakeAgent, prompts *fakePrompts) *Service {
	t.Helper()
	return newTestServiceWithOptions(t, root, serviceOptions{agent: agent, prompts: prompts})
}

type serviceOptions struct {
	agent          *fakeAgent
	prompts        *fakePrompts
	lock           InitiativeLock
	initiatives    fakeInitiatives
	modeUpdater    InitiativeModeUpdater
	planRefBinder  InitiativePlanRefBinder
	itemExecutions ItemExecutionController
	backlogMutator BacklogMutator
	reconciler     ProposalReconciler
	planExecution  PlanExecutionClient
	// classifier injects an L2 resolution classifier. When nil the test service
	// disables the live ollama classifier so unit tests never shell out.
	classifier FieldClassifier
}

func newTestServiceWithOptions(t *testing.T, root string, opts serviceOptions) *Service {
	t.Helper()
	agent := opts.agent
	if agent == nil {
		agent = &fakeAgent{}
	}
	prompts := opts.prompts
	if prompts == nil {
		prompts = &fakePrompts{}
	}
	if agent.states == nil {
		agent.states = map[string]agentmanager.RunState{}
	}
	store := NewStore(func(name string) string {
		return filepath.Join(root, "initiatives", name)
	})
	store.Clock = func() time.Time {
		return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	}
	lock := &initiativelock.Lock{
		Dir: func(name string) string {
			return filepath.Join(root, "initiatives", name)
		},
		Clock: store.Clock,
	}
	var serviceLock InitiativeLock
	if opts.lock != nil {
		serviceLock = opts.lock
	} else {
		serviceLock = lock
	}
	initiatives := opts.initiatives
	if initiatives.items == nil {
		initiatives = fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-a": {
				Name:               "init-a",
				Title:              "Init A",
				Description:        "Test initiative",
				Mode:               string(ModeHolisticLoop),
				Items:              []string{"execute/do-thing"},
				AcceptanceCriteria: []string{"works end to end"},
			},
		}}
	}
	overlay := NewOverlayStore(filepath.Join(root, ".vrooli", "operating-modes", "overrides.json"))
	svc, err := NewService(Config{
		Store:            store,
		Overlay:          overlay,
		Lock:             serviceLock,
		Initiatives:      initiatives,
		InitiativeLister: initiatives,
		ModeUpdater:      opts.modeUpdater,
		PlanRefBinder:    opts.planRefBinder,
		Backlog: fakeBacklog{items: map[string]BacklogItemSnapshot{
			"execute/do-thing": {Title: "Do thing", Status: "ready", Priority: 5, Effort: "M"},
		}},
		BacklogMutator: opts.backlogMutator,
		Reconciler:     opts.reconciler,
		ItemExecutions: opts.itemExecutions,
		Agent:          agent,
		PromptClient:   prompts,
		PromptCatalog: func(mode, phase string) (PromptCatalogEntry, bool) {
			return ExpectedPromptCatalogEntry(mode, phase)
		},
		ScenarioRoot:      root,
		Clock:             store.Clock,
		Classifier:        opts.classifier,
		DisableClassifier: opts.classifier == nil,
		PlanExecution:     opts.planExecution,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func testOperatingModePlanRef() *PlanRef {
	return &PlanRef{
		Provider: PlanRefProviderPlanManager,
		PlanID:   "plan-1",
		Slug:     "initiative-plan",
		Role:     PlanRefRoleOperatingModePlan,
	}
}

func saveCompletedRound(t *testing.T, svc *Service, initiativeName string, mode Mode, phase Phase, payload map[string]any) RoundEnvelope {
	t.Helper()
	return saveCompletedRoundWith(t, svc, initiativeName, mode, phase, payload, nil)
}

func saveCompletedRoundWithHandoff(t *testing.T, svc *Service, initiativeName string, mode Mode, phase Phase, payload map[string]any) RoundEnvelope {
	t.Helper()
	return saveCompletedRoundWith(t, svc, initiativeName, mode, phase, payload, []Handoff{{Summary: "handoff"}})
}

func saveCompletedRoundWith(t *testing.T, svc *Service, initiativeName string, mode Mode, phase Phase, payload map[string]any, handoffs []Handoff) RoundEnvelope {
	t.Helper()
	round, err := svc.store.CreateRound(RoundEnvelope{
		Mode:           string(mode),
		InitiativeName: initiativeName,
		ScopeID:        initiativeName,
		Phase:          string(phase),
		Status:         RoundStatusCompleted,
		Payload:        payload,
		Handoffs:       handoffs,
	})
	if err != nil {
		t.Fatalf("CreateRound %s/%s: %v", mode, phase, err)
	}
	return round
}

func TestActiveRoundsByInitiative(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-running": {
				Name:  "init-running",
				Title: "Running",
				Mode:  string(ModeHolisticLoop),
			},
			"init-idle": {
				Name:  "init-idle",
				Title: "Idle",
				Mode:  string(ModeHolisticLoop),
			},
			"init-item-level": {
				Name:  "init-item-level",
				Title: "Item Level",
				Mode:  string(ModeItemLevel),
			},
		}},
	})

	// Reserve a round on init-running so it has an active round.
	reserved, err := svc.store.CreateRound(RoundEnvelope{
		Mode:           string(ModeHolisticLoop),
		InitiativeName: "init-running",
		ScopeID:        "init-running",
		Phase:          "investigate",
		Status:         RoundStatusReserved,
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if reserved.Round == 0 {
		t.Fatalf("expected non-zero round number, got %+v", reserved)
	}

	// Save a completed round on init-idle — it should not appear.
	saveCompletedRound(t, svc, "init-idle", ModeHolisticLoop, "investigate", nil)

	rounds, err := svc.ActiveRoundsByInitiative(context.Background())
	if err != nil {
		t.Fatalf("ActiveRoundsByInitiative: %v", err)
	}
	if _, ok := rounds["init-running"]; !ok {
		t.Fatalf("init-running missing from active rounds: %+v", rounds)
	}
	if got := rounds["init-running"]; got.Mode != string(ModeHolisticLoop) || got.Phase != "investigate" || got.Status != string(RoundStatusReserved) {
		t.Fatalf("init-running summary = %+v, want holistic-loop/investigate/reserved", got)
	}
	if _, ok := rounds["init-idle"]; ok {
		t.Errorf("init-idle should not appear (only completed rounds); got %+v", rounds["init-idle"])
	}
	if _, ok := rounds["init-item-level"]; ok {
		t.Errorf("init-item-level should not appear (item-level mode skipped); got %+v", rounds["init-item-level"])
	}
}

func TestActiveRoundsByInitiative_EmptyWhenNoLister(t *testing.T) {
	svc := &Service{}
	rounds, err := svc.ActiveRoundsByInitiative(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rounds) != 0 {
		t.Fatalf("expected empty map, got %+v", rounds)
	}
}
