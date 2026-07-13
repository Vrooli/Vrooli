package operatingmode

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/evidence"
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
	return InitiativeSnapshot{Name: name, Mode: string(ModeHolisticLoop), PlanRef: &ref}, nil
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
func (f *fakeAgent) GetRunMessages(_ context.Context, runID string) ([]agentmanager.RunMessage, error) {
	contents := f.messages[runID]
	messages := make([]agentmanager.RunMessage, 0, len(contents))
	for i, content := range contents {
		messages = append(messages, agentmanager.RunMessage{EventID: fmt.Sprintf("event-%d", i+1), Sequence: int64(i + 1), Content: content})
	}
	return messages, nil
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
	calls       []string
	sourceCalls []string
	err         error
	text        string
	// render, when set, deterministically substitutes variables so tests can
	// assert fixture data reaches the prompt. It also lets the parity test prove
	// the spawn path and preview path pass an identical variable map.
	render func(skillID string, variables map[string]string) string
	source func(skillID string, expectedVariables []string) (string, int)
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

func (f *fakePrompts) ReadSkillSource(_ context.Context, skillID string, expectedVariables []string) (promptmanager.SkillSourceSnapshot, error) {
	f.sourceCalls = append(f.sourceCalls, skillID)
	if f.err != nil {
		return promptmanager.SkillSourceSnapshot{}, f.err
	}
	placeholders := make(map[string]string, len(expectedVariables))
	for _, name := range expectedVariables {
		placeholders[name] = "{{" + name + "}}"
	}
	var content string
	revision := 1
	if f.source != nil {
		content, revision = f.source(skillID, expectedVariables)
	} else if f.render != nil {
		content = f.render(skillID, placeholders)
	} else if f.text != "" {
		content = f.text
	} else {
		var b strings.Builder
		fmt.Fprintf(&b, "rendered %s\n", skillID)
		for _, name := range expectedVariables {
			fmt.Fprintf(&b, "%s={{%s}}\n", name, name)
		}
		content = b.String()
	}
	digest := sha256.Sum256([]byte(content))
	return promptmanager.SkillSourceSnapshot{
		SkillID: skillID, Revision: revision, SelectedVariantID: "control",
		Content: content, ContentHash: fmt.Sprintf("sha256:%x", digest[:]),
		TemplateVariables: unsatisfiedTemplateSlots(content),
	}, nil
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
	if round.PromptTrace == nil || round.PromptTrace.SourceRevision == "" || round.PromptTrace.SourceHash == "" || round.PromptTrace.VariablesHash == "" || round.PromptTrace.RenderedPromptHash == "" {
		t.Fatalf("prompt trace = %+v", round.PromptTrace)
	}
	if round.PromptTrace.DefinitionDigest != round.DefinitionDigest || round.PromptTrace.InputContractDigest == "" {
		t.Fatalf("prompt trace execution provenance = %+v", round.PromptTrace)
	}
	if round.PromptTrace.RedactionMetadata["policy"] != "hashes_only" {
		t.Fatalf("prompt trace redaction = %+v", round.PromptTrace.RedactionMetadata)
	}
	if len(agent.spawned) != 1 {
		t.Fatalf("spawn count = %d, want 1", len(agent.spawned))
	}
	spawn := agent.spawned[0]
	if spawn.ProfileKey != ProfileDeepWork || spawn.Purpose != "holistic_loop_investigate" {
		t.Fatalf("spawn profile/purpose = %q/%q", spawn.ProfileKey, spawn.Purpose)
	}
	var pinnedInvestigate bool
	for _, skillID := range prompts.sourceCalls {
		pinnedInvestigate = pinnedInvestigate || skillID == "swarm-manager-holistic-loop-investigate"
	}
	if len(prompts.calls) != 0 || !pinnedInvestigate {
		t.Fatalf("prompt/source calls = %#v / %#v", prompts.calls, prompts.sourceCalls)
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

func TestStartPhaseRejectsCallerInputsBeforeExecutionOrRoundMutation(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
		Inputs:         map[string]any{"caller.undeclared": true},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown caller inputs") {
		t.Fatalf("StartPhase caller-input error = %v, want unknown input rejection", err)
	}
	if len(agent.spawned) != 0 {
		t.Fatalf("spawned = %d, want no spawn", len(agent.spawned))
	}
	executions, listErr := svc.store.ListExecutions("init-a", ModeHolisticLoop)
	if listErr != nil {
		t.Fatalf("ListExecutions: %v", listErr)
	}
	if len(executions) != 0 {
		t.Fatalf("executions = %+v, want no partial manifest", executions)
	}
	rounds, listErr := svc.store.ListRounds("init-a", ModeHolisticLoop)
	if listErr != nil {
		t.Fatalf("ListRounds: %v", listErr)
	}
	if len(rounds) != 0 {
		t.Fatalf("rounds = %+v, want no reserved round", rounds)
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
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "execute", completedDrainExecutePayload("blocked"))

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

// TestStartPhaseRejectsPlanTargetModeThroughInitiativeSurface pins the v2
// boundary: the generic phased-plan-drain targets a plan-manager plan, so an
// initiative whose mode field still names it gets a typed, actionable error
// from every initiative-keyed phase surface instead of a bogus plan lookup
// keyed on the initiative name.
func TestStartPhaseRejectsPlanTargetModeThroughInitiativeSurface(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-phased": {
				Name:               "init-phased",
				Title:              "Init Phased",
				Description:        "Initiative stranded on the plan-target drain",
				Mode:               string(ModePhasedPlanDrain),
				Items:              []string{"execute/do-thing"},
				AcceptanceCriteria: []string{"works end to end"},
			},
		}},
		planExecution: &fakePlanExecution{},
	})

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-phased",
		Phase:          "execute",
	})
	if err == nil || !strings.Contains(err.Error(), "targets plan-manager-plan") {
		t.Fatalf("StartPhase error = %v, want plan-target rejection", err)
	}
	if _, err := svc.RenderLivePrompt(context.Background(), "init-phased", "execute", 0, ""); err == nil || !strings.Contains(err.Error(), "targets plan-manager-plan") {
		t.Fatalf("RenderLivePrompt error = %v, want plan-target rejection", err)
	}
}

func TestStartPhaseFailsClosedWhenPromptRenderFails(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{err: errors.New("prompt-manager unavailable")})

	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err == nil || !strings.Contains(err.Error(), "pin prompt source") {
		t.Fatalf("StartPhase error = %v, want prompt source preflight failure", err)
	}
	if len(agent.spawned) != 0 {
		t.Fatalf("spawned = %d, want no spawn when prompt rendering fails", len(agent.spawned))
	}
	rounds, listErr := svc.store.ListRounds("init-a", ModeHolisticLoop)
	if listErr != nil || len(rounds) != 0 {
		t.Fatalf("rounds = %+v, %v; want no preflight mutation", rounds, listErr)
	}
	executions, listErr := svc.store.ListExecutions("init-a", ModeHolisticLoop)
	if listErr != nil || len(executions) != 0 {
		t.Fatalf("executions = %+v, %v; want no partial manifest", executions, listErr)
	}
	holder, inspectErr := svc.lock.Inspect("init-a")
	if inspectErr != nil {
		t.Fatalf("Inspect lock: %v", inspectErr)
	}
	if holder != nil {
		t.Fatalf("lock holder = %+v, want nil after prompt failure", holder)
	}
}

func TestStartResolvedPhaseDynamicPreflightFailureLeavesNoRoundOrLock(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	svc := newTestService(t, root, agent, &fakePrompts{})
	def, err := DefinitionFor(ModeHolisticLoop)
	if err != nil {
		t.Fatal(err)
	}
	phase := def.PhaseGraph.Phases[def.PhaseGraph.StartPhase]
	rc, err := svc.collectRunContext(context.Background(), def, phase, "init-a")
	if err != nil {
		t.Fatalf("collectRunContext: %v", err)
	}
	promptSources, err := svc.pinReachablePromptSources(context.Background(), def)
	if err != nil {
		t.Fatalf("pinReachablePromptSources: %v", err)
	}
	execution, err := svc.store.ContinueOrCreateExecutionWithPreflight(rc.Target.ID, def, nil, promptSources)
	if err != nil {
		t.Fatalf("ContinueOrCreateExecutionWithPreflight: %v", err)
	}
	// Simulate a dynamic preflight decode/provider failure after a valid
	// execution already exists. The persisted manifest remains valid; only the
	// request-local execution view is malformed.
	execution.CompiledInputContract = []byte("{")
	rc.Execution = &execution
	_, err = svc.startResolvedPhase(context.Background(), rc, "", nil, false, "tester")
	if err == nil || !strings.Contains(err.Error(), "decode execution input contract") {
		t.Fatalf("startResolvedPhase error = %v, want dynamic input preflight failure", err)
	}
	if len(agent.spawned) != 0 {
		t.Fatalf("dynamic preflight spawned %d runs", len(agent.spawned))
	}
	rounds, listErr := svc.store.ListRounds("init-a", ModeHolisticLoop)
	if listErr != nil || len(rounds) != 0 {
		t.Fatalf("dynamic preflight rounds = %+v err=%v, want none", rounds, listErr)
	}
	holder, inspectErr := svc.lock.Inspect("init-a")
	if inspectErr != nil || holder != nil {
		t.Fatalf("dynamic preflight lock = %+v err=%v, want none", holder, inspectErr)
	}
	if _, loadErr := svc.store.LoadExecution("init-a", ModeHolisticLoop, execution.ExecutionID); loadErr != nil {
		t.Fatalf("valid persisted execution was damaged: %v", loadErr)
	}
}

func TestStartPhaseRejectsPromptVariableMismatchBeforeMutation(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	prompts := &fakePrompts{source: func(skillID string, _ []string) (string, int) {
		return "skill=" + skillID + " {{UNDECLARED_PROMPT_VARIABLE}}", 1
	}}
	svc := newTestService(t, root, agent, prompts)
	_, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a", Phase: "investigate",
	})
	if err == nil || !strings.Contains(err.Error(), "do not exactly match compiled bindings") {
		t.Fatalf("StartPhase error = %v, want exact variable mismatch", err)
	}
	executions, listErr := svc.store.ListExecutions("init-a", ModeHolisticLoop)
	if listErr != nil || len(executions) != 0 {
		t.Fatalf("executions = %+v, %v; want no partial manifest", executions, listErr)
	}
	rounds, listErr := svc.store.ListRounds("init-a", ModeHolisticLoop)
	if listErr != nil || len(rounds) != 0 || len(agent.spawned) != 0 {
		t.Fatalf("rounds/spawns = %+v/%d, err=%v; want no side effects", rounds, len(agent.spawned), listErr)
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

func TestStartPhaseCreatesExecutionManifestAndRunOwnerIndex(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})
	svc.store.ExecutionID = func() string { return "execution-001" }
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "investigate",
	})
	if err != nil {
		t.Fatalf("StartPhase: %v", err)
	}
	if round.ExecutionID != "execution-001" || round.DefinitionDigest == "" {
		t.Fatalf("round execution provenance = %+v", round)
	}
	execution, err := svc.store.LoadExecution("init-a", ModeHolisticLoop, round.ExecutionID)
	if err != nil {
		t.Fatalf("LoadExecution: %v", err)
	}
	if execution.DefinitionDigest != round.DefinitionDigest || len(execution.DefinitionBundle.Definitions) != 2 {
		t.Fatalf("execution manifest = %+v", execution)
	}
	if len(execution.ReachablePromptSources) == 0 {
		t.Fatalf("execution prompt bundle was not pinned: %+v", execution)
	}
	for key, source := range execution.ReachablePromptSources {
		if source.Revision == "" || source.ContentHash == "" || source.Content == "" || len(source.TemplateVariables) == 0 {
			t.Fatalf("incomplete prompt source %q: %+v", key, source)
		}
	}
	owner, err := svc.store.ResolveRunOwner("init-a", ModeHolisticLoop, round.RunID)
	if err != nil {
		t.Fatalf("ResolveRunOwner: %v", err)
	}
	if owner.ExecutionID != round.ExecutionID || owner.Round != round.Round {
		t.Fatalf("run owner = %+v", owner)
	}
}

// [REQ:REQ-P1-011-OWNER-RECONCILIATION]
func TestLookupOwnersResolvesPlanTargetRunFromGlobalIndex(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})
	svc.store.ExecutionID = func() string { return "execution-plan-001" }
	execution, err := svc.store.CreateExecution("plan-123", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	if err := svc.store.IndexRunOwner(execution, "run-plan-001", 1); err != nil {
		t.Fatalf("IndexRunOwner: %v", err)
	}
	owners, err := svc.LookupOwners(context.Background(), "run-plan-001")
	if err != nil {
		t.Fatalf("LookupOwners: %v", err)
	}
	if len(owners) != 1 || owners[0].Kind != evidence.OwnerOperatingModeExecution || owners[0].ID != execution.ExecutionID || owners[0].Round != 1 {
		t.Fatalf("owners = %+v", owners)
	}
}

// [REQ:REQ-P1-011-OWNER-RECONCILIATION]
func TestLookupOwnersPreservesCrossTargetRunAmbiguity(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})
	ids := []string{"execution-plan-001", "execution-plan-002"}
	svc.store.ExecutionID = func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}
	first, err := svc.store.CreateExecution("plan-123", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution first: %v", err)
	}
	second, err := svc.store.CreateExecution("plan-456", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution second: %v", err)
	}
	for _, execution := range []OperatingModeExecution{first, second} {
		if err := svc.store.IndexRunOwner(execution, "run-shared", 1); err != nil {
			t.Fatalf("IndexRunOwner %q: %v", execution.ExecutionID, err)
		}
	}
	owners, err := svc.LookupOwners(context.Background(), "run-shared")
	if err != nil {
		t.Fatalf("LookupOwners: %v", err)
	}
	if len(owners) != 2 || owners[0].ID == owners[1].ID {
		t.Fatalf("owners = %+v, want both owners for ambiguity", owners)
	}
}

func TestRefreshRoundUsesPinnedDefinitionAfterRegistryMutation(t *testing.T) {
	root := t.TempDir()
	final := `{"operating_mode_result":{"handoff":{"summary":"investigated"},"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"# Findings"}]}}`
	agent := &fakeAgent{states: map[string]agentmanager.RunState{}}
	svc := newTestService(t, root, agent, &fakePrompts{})
	svc.store.ExecutionID = func() string { return "execution-pinned" }
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{InitiativeName: "init-a", Phase: "investigate"})
	if err != nil {
		t.Fatalf("StartPhase: %v", err)
	}
	execution, err := svc.store.LoadExecution("init-a", ModeHolisticLoop, round.ExecutionID)
	if err != nil {
		t.Fatalf("LoadExecution: %v", err)
	}

	registryMu.Lock()
	original := registry[ModeHolisticLoop]
	mutated, cloneErr := clonePinnedDefinition(original)
	if cloneErr != nil {
		registryMu.Unlock()
		t.Fatalf("clone registry definition: %v", cloneErr)
	}
	// If refresh accidentally consults the live registry, investigate now has
	// the review phase's verdict contract and the original output will abstain.
	mutated.PhaseGraph.Phases["investigate"] = mutated.PhaseGraph.Phases["review"]
	registry[ModeHolisticLoop] = mutated
	registryMu.Unlock()
	defer func() {
		registryMu.Lock()
		registry[ModeHolisticLoop] = original
		registryMu.Unlock()
	}()

	agent.states[round.RunID] = agentmanager.RunState{
		RunID: round.RunID, Status: "complete", Summary: final, FinishedAt: "2026-04-30T12:05:00Z",
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("refreshed = %+v, want pinned investigate contract completion", refreshed)
	}
	if refreshed.DefinitionDigest != execution.DefinitionDigest {
		t.Fatalf("round digest = %q, want %q", refreshed.DefinitionDigest, execution.DefinitionDigest)
	}
	_, liveDigest, err := pinDefinitionBundle(mutated, DefinitionFor)
	if err != nil {
		t.Fatalf("pin mutated definition: %v", err)
	}
	if liveDigest == execution.DefinitionDigest {
		t.Fatal("registry mutation did not change live definition digest")
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
	if refreshed.Status != RoundStatusNeedsAttention || !strings.Contains(refreshed.Error, "resolution abstained") {
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
	if candidates := svc.resolutionCandidates(context.Background(), round, agent.states["run-1"]); len(candidates) != 3 {
		t.Fatalf("resolution candidates = %d, want 3 (duplicate summary must not be appended)", len(candidates))
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
	if record.MessagesScanned != 2 {
		t.Fatalf("messages scanned = %d, want trailing event plus selected final event", record.MessagesScanned)
	}
	if record.SelectedMessage == nil || record.SelectedMessage.EventID != "event-2" || record.SelectedMessage.Sequence != 2 || record.SelectedMessage.FallbackReason != "earlier_contract_satisfying_assistant_event" {
		t.Fatalf("selected message = %+v, want stable event-2 provenance", record.SelectedMessage)
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

func TestInvalidResolvedEnvelopeDoesNotPartiallyApplySideEffects(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantErr    string
		artifact   string
		withBinder bool
	}{
		{
			name:       "late invalid artifact cannot leave earlier artifact or plan binding",
			output:     `{"operating_mode_result":{"plan_ref":{"provider":"plan-manager","plan_id":"plan-123","role":"operating_mode_plan"},"handoff":{"summary":"investigated"},"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"would leak"},{"path":"../outside.md","content":"invalid"}]}}`,
			wantErr:    "artifact path must be relative to initiative",
			artifact:   "modes/holistic-loop/findings.md",
			withBinder: true,
		},
		{
			name:     "missing required artifact cannot retain handoff or provenance",
			output:   `{"operating_mode_result":{"handoff":{"summary":"partial"}}}`,
			wantErr:  `requires artifact "modes/holistic-loop/findings.md"`,
			artifact: "modes/holistic-loop/findings.md",
		},
		{
			name:       "invalid plan ref cannot retain canonical envelope or provenance",
			output:     `{"operating_mode_result":{"plan_ref":{"provider":"other","plan_id":"plan-123","role":"operating_mode_plan"},"handoff":{"summary":"partial"},"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"would leak"}]}}`,
			wantErr:    `plan_ref provider must be "plan-manager"`,
			artifact:   "modes/holistic-loop/findings.md",
			withBinder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			binder := &fakePlanRefBinder{}
			opts := serviceOptions{}
			if tt.withBinder {
				opts.planRefBinder = binder
			}
			svc := newTestServiceWithOptions(t, root, opts)
			round := RoundEnvelope{
				Mode: string(ModeHolisticLoop), ScopeKind: string(TargetInitiative), ScopeID: "init-a",
				InitiativeName: "init-a", Phase: "investigate", Payload: map[string]any{"sentinel": "preserved"},
			}
			before := cloneRoundForPhaseResult(round)
			_, err := svc.applyPhaseResultWithPersistence(context.Background(), MustDefinition(ModeHolisticLoop), &round, []resolutionCandidate{{
				Content: tt.output, EventID: "event-invalid", Sequence: 99,
			}}, true)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("apply error = %v, want %q", err, tt.wantErr)
			}
			if !reflect.DeepEqual(round, before) {
				t.Fatalf("round mutated after rejected envelope:\n got: %#v\nwant: %#v", round, before)
			}
			if len(binder.refs) != 0 {
				t.Fatalf("plan refs = %#v, want no binding", binder.refs)
			}
			if _, err := svc.store.ReadArtifact("init-a", ModeHolisticLoop, tt.artifact); !errors.Is(err, ErrArtifactNotFound) {
				t.Fatalf("ReadArtifact error = %v, want ErrArtifactNotFound", err)
			}
		})
	}
}

// TestRefreshRoundBindsEmittedPlanRef proves the generic plan-ref binding seam:
// any completed round whose structured result carries a valid operating-mode
// plan_ref binds it to the initiative and persists it on the round payload.
// (Formerly exercised through the drain's prepare_plan phase; the generic drain
// has no plan-preparation phase, so holistic-loop investigate drives the seam.)
func TestRefreshRoundBindsEmittedPlanRef(t *testing.T) {
	root := t.TempDir()
	binder := &fakePlanRefBinder{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		agent: &fakeAgent{states: map[string]agentmanager.RunState{
			"run-1": {
				RunID:      "run-1",
				Status:     "complete",
				Summary:    `{"operating_mode_result":{"plan_ref":{"provider":"plan-manager","plan_id":"plan-123","slug":"initiative-plan","role":"operating_mode_plan"},"handoff":{"summary":"plan ready"},"artifacts":[{"path":"modes/holistic-loop/findings.md","content":"# Findings"}]}}`,
				FinishedAt: "2026-04-30T12:05:00Z",
			},
		}},
		planRefBinder: binder,
	})

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
		t.Fatalf("refreshed status = %q, want completed: %+v", refreshed.Status, refreshed)
	}
	if got := binder.refs["init-a"].PlanID; got != "plan-123" {
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
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "execute", completedDrainExecutePayload("complete"))
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
	store.TargetDir = func(kind TargetKind, scopeID string) string {
		return TargetScopeDir(filepath.Join(root, "data"), kind, scopeID)
	}
	store.RunOwnerDir = func() string {
		return filepath.Join(root, "data", "operating-mode-run-owners")
	}
	store.RunOwnerRecovery = func(runID string) ([]GlobalRunOwner, error) {
		return RecoverTargetRunOwners(filepath.Join(root, "data"), runID)
	}
	store.Clock = func() time.Time {
		return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	}
	lock := &initiativelock.Lock{
		Dir: func(key string) string {
			if kind, token, ok := ParseTargetOwnershipKey(key); ok {
				return TargetScopeDir(filepath.Join(root, "data"), kind, token)
			}
			return filepath.Join(root, "initiatives", key)
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
				PlanRef:            &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: "init-a-plan", Role: PlanRefRoleOperatingModePlan},
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
		PlanExecution:     planExecutionOrFake(opts.planExecution),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

// planExecutionOrFake defaults the plan-manager execution client to the fake
// so holistic-loop fixtures (which now require a bound plan) resolve context
// without every test wiring one explicitly.
func planExecutionOrFake(client PlanExecutionClient) PlanExecutionClient {
	if client != nil {
		return client
	}
	return &fakePlanExecution{}
}

// completedDrainExecutePayload is the payload a completed delegated execute
// round carries: the delegation markers plus the edge-classified progress
// value the parent's guards route on.
func completedDrainExecutePayload(progress string) map[string]any {
	return map[string]any{
		payloadDelegatedMode:  string(ModePhasedPlanDrain),
		payloadDelegatedPhase: "execute",
		"progress":            progress,
	}
}

func saveCompletedRound(t *testing.T, svc *Service, initiativeName string, mode Mode, phase Phase, payload map[string]any) RoundEnvelope {
	t.Helper()
	return saveCompletedRoundWith(t, svc, initiativeName, mode, phase, payload, nil)
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
