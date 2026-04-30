package operatingmode

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/promptmanager"
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
	spawned []agentmanager.InitiativeSpawnRequest
	states  map[string]agentmanager.RunState
	stopped []string
}

func (f *fakeAgent) SpawnInitiative(_ context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
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
}

func (f *fakePrompts) ReadSkill(_ context.Context, skillID string, _ map[string]string, _ bool) (string, error) {
	f.calls = append(f.calls, skillID)
	return "rendered " + skillID, nil
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

func TestWorkspaceRefreshesCompletedRoundAndReleasesLock(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{
		"run-1": {RunID: "run-1", Status: "complete", Summary: "done", FinishedAt: "2026-04-30T12:05:00Z"},
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

func newTestService(t *testing.T, root string, agent *fakeAgent, prompts *fakePrompts) *Service {
	t.Helper()
	return newTestServiceWithOptions(t, root, serviceOptions{agent: agent, prompts: prompts})
}

type serviceOptions struct {
	agent          *fakeAgent
	prompts        *fakePrompts
	initiatives    fakeInitiatives
	modeUpdater    InitiativeModeUpdater
	itemExecutions ItemExecutionController
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
	svc, err := NewService(Config{
		Store:       store,
		Lock:        lock,
		Initiatives: initiatives,
		ModeUpdater: opts.modeUpdater,
		Backlog: fakeBacklog{items: map[string]BacklogItemSnapshot{
			"execute/do-thing": {Title: "Do thing", Status: "ready", Priority: 5, Effort: "M"},
		}},
		ItemExecutions: opts.itemExecutions,
		Agent:          agent,
		PromptClient:   prompts,
		ScenarioRoot:   root,
		Clock:          store.Clock,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}
