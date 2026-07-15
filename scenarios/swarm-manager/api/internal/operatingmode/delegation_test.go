package operatingmode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/evidence"
)

// delegationTestModeDoc renders a minimal initiative-target mode document
// whose execute phase delegates to the named sub-mode.
func delegationTestModeDoc(id, executedBy string) []byte {
	doc := fmt.Sprintf(`{
	  "kind": "operating-mode",
	  "id": %q,
	  "label": "Delegation Test",
	  "description": "Test mode delegating execution.",
	  "best_for": ["testing"],
	  "not_for": ["production"],
	  "tradeoffs": ["none"],
	  "target": {"kind": "initiative", "plan_ref": {"required": true, "role": "operating_mode_plan"}},
	  "input_contract": {"specs": [], "sources": [], "aliases": []},
	  "run_strategy": {"kind": "operator_gated_loop"},
	  "prompt": {"catalog_prefix": "swarm-manager-delegation-test"},
	  "artifact": {"root": "modes/delegation-test"},
	  "backlog_sync": {"capabilities": ["read_only"], "apply_mode": "operator-gated"},
	  "lock": {"initiative_exclusive": true},
	  "ui": {"workspace_tab_id": "operating-mode"},
	  "phase_graph": {
	    "start_phase": "execute",
	    "phases": [
	      {
	        "id": "execute",
	        "kind": "execute",
	        "executed_by": %q,
	        "transitions": [
	          {"when": {"op": "eq", "field": "progress", "value": "complete"}, "to": []}
	        ]
	      }
	    ]
	  }
	}`, id, executedBy)
	return []byte(doc)
}

func loadDelegationTestDefs(t *testing.T, id, executedBy string) map[Mode]Definition {
	t.Helper()
	def, err := LoadModeDefinition(delegationTestModeDoc(id, executedBy))
	if err != nil {
		t.Fatalf("LoadModeDefinition: %v", err)
	}
	drain := loadModeFromDisk(t, string(ModePhasedPlanDrain))
	item := loadModeFromDisk(t, string(ModeItemLevel))
	holistic := loadModeFromDisk(t, string(ModeHolisticLoop))
	return map[Mode]Definition{
		def.Mode:            def,
		ModePhasedPlanDrain: drain,
		ModeItemLevel:       item,
		ModeHolisticLoop:    holistic,
	}
}

// TestDelegationValidation pins the composition limits: unknown sub-mode,
// self-delegation, nesting (a sub-mode with its own delegated phase), and
// target-incompatible delegation are all rejected at load; a valid one-level
// delegation validates cleanly.
func TestDelegationValidation(t *testing.T) {
	t.Run("valid one-level delegation", func(t *testing.T) {
		defs := loadDelegationTestDefs(t, "delegation-test", string(ModePhasedPlanDrain))
		if err := validateDelegations(defs); err != nil {
			t.Fatalf("validateDelegations: %v", err)
		}
	})

	t.Run("unknown sub-mode rejected", func(t *testing.T) {
		defs := loadDelegationTestDefs(t, "delegation-test", "no-such-mode")
		err := validateDelegations(defs)
		if err == nil || !strings.Contains(err.Error(), "unknown sub-mode") {
			t.Fatalf("err = %v, want unknown sub-mode rejection", err)
		}
	})

	t.Run("self-delegation rejected", func(t *testing.T) {
		defs := loadDelegationTestDefs(t, "delegation-test", "delegation-test")
		err := validateDelegations(defs)
		if err == nil || !strings.Contains(err.Error(), "self-delegation") {
			t.Fatalf("err = %v, want self-delegation rejection", err)
		}
	})

	t.Run("delegating to a mode with no rounds rejected", func(t *testing.T) {
		defs := loadDelegationTestDefs(t, "delegation-test", string(ModeItemLevel))
		err := validateDelegations(defs)
		if err == nil || !strings.Contains(err.Error(), "runs no mode rounds") {
			t.Fatalf("err = %v, want no-mode-rounds rejection", err)
		}
	})

	t.Run("nested delegation rejected", func(t *testing.T) {
		// holistic-loop itself delegates execute to the drain, so delegating to
		// holistic-loop is exactly the forbidden second composition level.
		defs := loadDelegationTestDefs(t, "delegation-test", string(ModeHolisticLoop))
		err := validateDelegations(defs)
		if err == nil || !strings.Contains(err.Error(), "one level deep") {
			t.Fatalf("err = %v, want nesting rejection", err)
		}
	})

	t.Run("target-incompatible delegation rejected", func(t *testing.T) {
		raw := delegationTestModeDoc("delegation-test", string(ModePhasedPlanDrain))
		// Strip the bound-plan requirement: an initiative-target mode without a
		// required plan_ref cannot supply the drain's plan context.
		doc := strings.Replace(string(raw),
			`"target": {"kind": "initiative", "plan_ref": {"required": true, "role": "operating_mode_plan"}},`,
			`"target": {"kind": "initiative"},`, 1)
		def, err := LoadModeDefinition([]byte(doc))
		if err != nil {
			t.Fatalf("LoadModeDefinition: %v", err)
		}
		defs := map[Mode]Definition{
			def.Mode:            def,
			ModePhasedPlanDrain: loadModeFromDisk(t, string(ModePhasedPlanDrain)),
		}
		err = validateDelegations(defs)
		if err == nil || !strings.Contains(err.Error(), "target.plan_ref.required") {
			t.Fatalf("err = %v, want plan_ref.required rejection", err)
		}
	})
}

// TestDelegatedPhaseRejectsOwnExecutionSurface pins the schema/loader contract:
// a delegated phase declaring its own reads/prompt/declared_output is rejected.
func TestDelegatedPhaseRejectsOwnExecutionSurface(t *testing.T) {
	raw := strings.Replace(string(delegationTestModeDoc("delegation-test", string(ModePhasedPlanDrain))),
		`"executed_by": "phased-plan-drain",`,
		`"executed_by": "phased-plan-drain", "reads": ["OPERATOR_NOTE"],`, 1)
	if _, err := LoadModeDefinition([]byte(raw)); err == nil {
		t.Fatal("expected a delegated phase with its own reads to be rejected")
	}
}

// TestStartPhaseDelegatedExecuteRunsSubModeRound proves the runtime delegation
// semantics end-to-end on holistic-loop: starting the delegated execute phase
// spawns the SUB-mode's execute-next round (drain skill, drain profile, plan
// reads) under the parent run (parent mode/phase/scope, initiative lock), and
// after the round classifies `continue` the next startable phase is the
// delegated phase again, while `complete` routes to review.
func TestStartPhaseDelegatedExecuteRunsSubModeRound(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	prompts := &fakePrompts{render: echoRender}
	svc := newTestService(t, root, agent, prompts)

	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "plan", nil)

	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "execute",
	})
	if err != nil {
		t.Fatalf("StartPhase(execute): %v", err)
	}
	if round.Mode != string(ModeHolisticLoop) || round.Phase != "execute" {
		t.Fatalf("round = %s/%s, want holistic-loop/execute (parent identity)", round.Mode, round.Phase)
	}
	if got := round.Payload[payloadDelegatedMode]; got != string(ModePhasedPlanDrain) {
		t.Fatalf("payload delegated_mode = %v, want phased-plan-drain", got)
	}
	if got := round.Payload[payloadDelegatedPhase]; got != "execute" {
		t.Fatalf("payload delegated_phase = %v, want execute", got)
	}
	if got := round.Payload["skill_id"]; got != "swarm-manager-phased-plan-execute-next" {
		t.Fatalf("payload skill_id = %v, want the drain's execute-next skill", got)
	}
	spawn := agent.spawned[len(agent.spawned)-1]
	if spawn.Purpose != "phased_plan_execute_next" {
		t.Fatalf("spawn purpose = %q, want the sub-mode's activity purpose", spawn.Purpose)
	}
	if !strings.Contains(spawn.Prompt, "PLAN_ID=") {
		t.Fatalf("delegated prompt missing plan read:\n%s", spawn.Prompt)
	}
	if strings.Contains(spawn.Prompt, "MEMBER_ITEMS_JSON") {
		t.Fatalf("delegated prompt leaked initiative reads:\n%s", spawn.Prompt)
	}
	// The parent initiative holds the exclusive lock, not a plan key.
	holder, err := svc.lock.Inspect("init-a")
	if err != nil || holder == nil {
		t.Fatalf("initiative lock holder = %v err=%v, want held", holder, err)
	}

	// Complete the round with a continue handoff: the sub-route loops the
	// delegated phase inline.
	completeDelegatedRun(t, agent, round.RunID, "continue")
	refreshed, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound: %v", err)
	}
	if refreshed.Status != RoundStatusCompleted {
		t.Fatalf("refreshed status = %q (err=%q), want completed", refreshed.Status, refreshed.Error)
	}
	if got, _ := refreshed.Payload["progress"].(string); got != "continue" {
		t.Fatalf("derived progress = %q, want continue", got)
	}
	rounds, err := svc.store.ListRounds("init-a", ModeHolisticLoop)
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	def := MustDefinition(ModeHolisticLoop)
	next := allowedNextPhases(def, rounds)
	if !next["execute"] || next["review"] {
		t.Fatalf("after continue, allowed next = %v, want execute only (inline delegation loop)", next)
	}

	// Run the loop again and complete it: the sub-mode stop surfaces the
	// outcome and the PARENT guard routes to review.
	second, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "execute",
	})
	if err != nil {
		t.Fatalf("StartPhase(execute) second: %v", err)
	}
	completeDelegatedRun(t, agent, second.RunID, "complete")
	if _, err := svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, second.Round); err != nil {
		t.Fatalf("RefreshRound second: %v", err)
	}
	rounds, err = svc.store.ListRounds("init-a", ModeHolisticLoop)
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	next = allowedNextPhases(def, rounds)
	if !next["review"] || next["execute"] {
		t.Fatalf("after complete, allowed next = %v, want review only (parent guard routes)", next)
	}
}

// completeDelegatedRun installs a completed agent run whose summary carries the
// drain's declared handoff with the routing value inline (L1-derivable).
func completeDelegatedRun(t *testing.T, agent *fakeAgent, runID, progress string) {
	t.Helper()
	envelope := map[string]any{
		"operating_mode_result": map[string]any{
			"handoff": map[string]any{
				"summary":       "Drained one slice.",
				"blockers":      []string{"none"},
				"next_step":     "Continue from the frontier.",
				"changed_files": []string{"api/main.go"},
				"tests":         []string{"go test ./..."},
				"frontier":      "Next contiguous unit.",
				"progress":      progress,
			},
		},
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	agent.states[runID] = agentmanager.RunState{
		RunID: runID, Status: "complete",
		Summary:    string(body),
		FinishedAt: "2026-05-02T12:05:00Z",
	}
}

// [REQ:REQ-P1-011-OWNER-RECONCILIATION]
// TestStartTargetPhaseStartsDrainOnBarePlan is the plan-first entry point
// test: the generic drain starts directly on a plan-manager plan — no
// initiative — with plan-scoped rounds, plan reads, and the plan ownership
// lock key.
func TestStartTargetPhaseStartsDrainOnBarePlan(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	prompts := &fakePrompts{render: echoRender}
	svc := newTestService(t, root, agent, prompts)

	round, err := svc.StartTargetPhase(context.Background(), StartTargetPhaseRequest{
		Mode:      string(ModePhasedPlanDrain),
		TargetRef: "my-trivial-plan",
	})
	if err != nil {
		t.Fatalf("StartTargetPhase: %v", err)
	}
	if round.Mode != string(ModePhasedPlanDrain) || round.Phase != "execute" {
		t.Fatalf("round = %s/%s, want phased-plan-drain/execute", round.Mode, round.Phase)
	}
	if round.ScopeKind != string(TargetPlanExecution) {
		t.Fatalf("scope kind = %q, want plan-execution", round.ScopeKind)
	}
	// The fake plan execution resolves execution id "exec-1".
	if round.ScopeID != "exec-1" {
		t.Fatalf("scope id = %q, want the resolved plan execution id", round.ScopeID)
	}
	if round.InitiativeName != "" {
		t.Fatalf("initiative name = %q, want empty (no initiative ceremony)", round.InitiativeName)
	}
	spawn := agent.spawned[len(agent.spawned)-1]
	if !strings.Contains(spawn.Prompt, "PLAN_ID=exec-1") {
		t.Fatalf("prompt missing PLAN_ID read:\n%s", spawn.Prompt)
	}
	if strings.Contains(spawn.Prompt, "INITIATIVE_NAME") {
		t.Fatalf("plan-target prompt leaked initiative reads:\n%s", spawn.Prompt)
	}
	// The exclusive lock uses the plan ownership key, not an initiative name.
	holder, err := svc.lock.Inspect("plan--exec-1")
	if err != nil || holder == nil {
		t.Fatalf("plan lock holder = %v err=%v, want held", holder, err)
	}
	owners, err := svc.LookupOwners(context.Background(), round.RunID)
	if err != nil {
		t.Fatalf("LookupOwners: %v", err)
	}
	if len(owners) != 1 || owners[0].Kind != evidence.OwnerOperatingModeExecution || owners[0].ID != round.ExecutionID || owners[0].Round != round.Round {
		t.Fatalf("plan-target evidence owner = %+v", owners)
	}

	// The round is addressable through the ordinary round actions with the
	// plan scope id + explicit mode — cancel releases the plan lock.
	canceled, err := svc.CancelRound(context.Background(), "exec-1", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("CancelRound: %v", err)
	}
	if canceled.Status != RoundStatusCanceled {
		t.Fatalf("canceled status = %q", canceled.Status)
	}
	holder, err = svc.lock.Inspect("plan--exec-1")
	if err != nil {
		t.Fatalf("Inspect after cancel: %v", err)
	}
	if holder != nil {
		t.Fatalf("plan lock still held after cancel: %+v", holder)
	}
}

// TestStartTargetPhaseRejectsInitiativeTargetMode pins the boundary: the
// target surface never drives initiative-target modes.
func TestStartTargetPhaseRejectsInitiativeTargetMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root, &fakeAgent{}, &fakePrompts{})
	_, err := svc.StartTargetPhase(context.Background(), StartTargetPhaseRequest{
		Mode:      string(ModeHolisticLoop),
		TargetRef: "init-a",
	})
	if err == nil || !strings.Contains(err.Error(), "initiative surface") {
		t.Fatalf("err = %v, want initiative-surface rejection", err)
	}
}
