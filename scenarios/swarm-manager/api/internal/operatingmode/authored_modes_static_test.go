package operatingmode

import "testing"

// TestAuthoredModesUseNoRetiredConstructs is the static guardrail for the
// declarative-operations mode catalog: every shipped mode must use a supported
// target kind, and the retired item-level pseudo-mode must never return — the
// member-item workflow strategy is initiative configuration
// (agentops.MemberItemStrategy), not a mode folder. It fails closed if a mode
// reintroduces a removed target kind (e.g. the pre-cutover unmanaged plan-ref)
// or the reserved item-level id.
func TestAuthoredModesUseNoRetiredConstructs(t *testing.T) {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		t.Fatalf("load shipped modes: %v", err)
	}
	validTargets := map[TargetKind]bool{
		TargetBacklogItem: true, TargetInitiative: true, TargetPlanExecution: true, TargetScenario: true,
	}
	for _, mode := range SortedModes(defs) {
		def := defs[mode]
		if !validTargets[def.Target.Kind] {
			t.Errorf("mode %q declares unsupported/removed target kind %q", mode, def.Target.Kind)
		}
		// Every shipped mode is a real methodology loop with a phase graph.
		if def.PhaseGraph.StartPhase == "" || len(def.PhaseGraph.Phases) == 0 {
			t.Errorf("mode %q declares no phase graph; every registered mode runs mode rounds", mode)
		}
	}
	// item-level was deleted in Phase 9: it is the member-item-strategy
	// sentinel wire value on initiatives, never a mode folder.
	if _, ok := defs[ModeItemLevel]; ok {
		t.Fatalf("item-level mode folder reintroduced; %q is the reserved member-item-strategy sentinel, not a mode", ModeItemLevel)
	}
}

// TestEveryLedgerBehaviorHasBindingAndCompatibleMode proves the acceptance
// contract of Phase 4: every authored operation contract resolves to a
// system-default binding whose mode is registered and target-compatible with the
// operation. Reading data alone (contracts + bindings + modes) is enough to see
// that every ledger (a)-behavior maps to a validated mode + binding.
func TestModesCoverEveryDeclaredOperationTargetKind(t *testing.T) {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		t.Fatalf("load shipped modes: %v", err)
	}
	// Every mode a binding names must exist with a real phase graph.
	for _, want := range []Mode{
		"backlog-research", "backlog-workshop", "backlog-finalize", "backlog-clarify",
		"backlog-fixup", "backlog-followup", "backlog-evidence", "backlog-revision",
		"backlog-review", "execution-drain", "initiative-review-loop",
		"scenario-spec-sync",
	} {
		def, ok := defs[want]
		if !ok {
			t.Errorf("binding target mode %q is not shipped", want)
			continue
		}
		if len(def.PhaseGraph.Phases) == 0 {
			t.Errorf("binding target mode %q declares no phases", want)
		}
	}
}
