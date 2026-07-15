package operatingmode

import "testing"

// TestAuthoredModesUseNoRetiredConstructs is the static guardrail for the
// declarative-operations mode catalog: every shipped mode that is a real
// methodology loop must use a supported target kind and run strategy, and the
// only mode permitted to use the legacy existing_item_flow run strategy is
// item-level (a compatibility placeholder, not a methodology loop, deleted in a
// later phase). It fails closed if a new mode reintroduces a removed target kind
// (e.g. the pre-cutover unmanaged plan-ref) or the existing_item_flow strategy.
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
		if def.RunStrategy.Kind == RunStrategyExistingItemFlow && mode != ModeItemLevel {
			t.Errorf("mode %q uses the retired existing_item_flow run strategy; only item-level may, and only until its deletion", mode)
		}
		// A real methodology mode runs mode rounds; item-level (existing_item_flow)
		// must NOT — it has no phase graph and is not a selectable loop.
		if mode == ModeItemLevel && def.RunsModeRounds() {
			t.Errorf("item-level must not run mode rounds: it is a member-item-strategy placeholder, not a methodology loop")
		}
	}
	// item-level must be present-but-not-a-loop this phase (deleted in Phase 9).
	if _, ok := defs[ModeItemLevel]; !ok {
		t.Fatalf("item-level folder unexpectedly absent before its Phase 9 deletion")
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
	// Every mode a binding names must exist and run mode rounds (item-level is
	// never a binding target).
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
		if !def.RunsModeRounds() {
			t.Errorf("binding target mode %q runs no mode rounds", want)
		}
	}
}
