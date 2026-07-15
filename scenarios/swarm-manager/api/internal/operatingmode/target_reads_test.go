package operatingmode

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// planDrainTestModeJSON is a minimal, valid plan-execution-target mode used
// to exercise the target/reads substrate. Optional overrides let rejection
// tests swap the target kind or the declared reads.
func planDrainTestModeJSON(targetKind string, reads []string) string {
	quoted := make([]string, 0, len(reads))
	for _, r := range reads {
		quoted = append(quoted, `"`+r+`"`)
	}
	return `{
	  "kind": "operating-mode",
	  "id": "plan-drain-test",
	  "label": "Plan Drain Test",
	  "description": "Synthetic plan-target drain used by unit tests.",
	  "best_for": ["Draining a stable plan without initiative ceremony"],
	  "not_for": ["Work that needs member-item tracking"],
	  "tradeoffs": ["No backlog reconciliation"],
	  "target": { "kind": "` + targetKind + `" },
	  "input_contract": ` + testInputContractJSON(targetKind, reads) + `,
	  "run_strategy": { "kind": "sequential_handoff" },
	  "prompt": { "catalog_prefix": "swarm-manager-plan-drain-test" },
	  "artifact": { "root": "modes/plan-drain-test" },
	  "profile": { "default_profile_key": "swarm-manager/deep-work" },
	  "metrics": { "event_source": "plan-drain-test" },
	  "ui": { "workspace_tab_id": "operating-mode" },
	  "phase_graph": {
	    "start_phase": "execute",
	    "terminal": ["execute"],
	    "phases": [
	      {
	        "id": "execute",
	        "kind": "execute",
	        "activity_purpose": "plan_drain_test_execute",
	        "reads": [` + strings.Join(quoted, ", ") + `],
	        "writes_repo": true,
	        "declared_output": {
	          "fields": [ { "name": "handoff", "type": "object", "required": true } ]
	        }
	      }
	    ]
	  }
	}`
}

func testInputContractJSON(_ string, reads []string) string {
	contract := InputContractDefinition{}
	for _, read := range reads {
		id, capability, valueType, sourceKind := testInputDescriptor(read)
		contract.Specs = append(contract.Specs, InputSpec{
			ID: id, Type: valueType, Required: true, Sensitivity: InputSensitivityInternal,
			Retention: InputRetentionValue, Description: "Test input for " + read + ".",
		})
		contract.Sources = append(contract.Sources, InputSourceBinding{
			InputID: id, Kind: sourceKind, Capability: capability,
		})
		contract.Aliases = append(contract.Aliases, InputAlias{Name: read, InputID: id})
	}
	data, err := json.Marshal(contract)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func testInputDescriptor(read string) (string, string, InputValueType, InputSourceKind) {
	switch read {
	case ReadOperatingMode:
		return "execution.mode_id", "generic.operating_mode", InputTypeString, InputSourceGenericProvider
	case ReadModeLabel:
		return "execution.mode_label", "generic.mode_label", InputTypeString, InputSourceGenericProvider
	case ReadPhase:
		return "execution.phase_id", "generic.phase", InputTypeString, InputSourceGenericProvider
	case ReadRunStrategy:
		return "execution.run_strategy", "generic.run_strategy", InputTypeString, InputSourceGenericProvider
	case ReadRoundNumber:
		return "execution.round_number", "generic.round_number", InputTypeInteger, InputSourceGenericProvider
	case ReadAgentProfileKey:
		return "execution.agent_profile_key", "generic.agent_profile_key", InputTypeString, InputSourceGenericProvider
	case ReadOperatorNote:
		return "execution.operator_note", "generic.operator_note", InputTypeString, InputSourceGenericProvider
	case ReadPriorRoundsJSON:
		return "execution.prior_rounds", "generic.prior_rounds", InputTypeArray, InputSourceGenericProvider
	case ReadModeArtifactsJSON:
		return "execution.mode_artifacts", "generic.mode_artifacts", InputTypeArray, InputSourceGenericProvider
	case "BACKLOG_SYNC_PROPOSAL_SNIPPET":
		return "execution.backlog_sync_guidance", "generic.backlog_sync_proposal", InputTypeString, InputSourceGenericProvider
	case "ELASTIC_SLICE_SNIPPET":
		return "execution.elastic_slice_guidance", "generic.elastic_slice", InputTypeString, InputSourceGenericProvider
	case ReadInitiativeName:
		return "initiative.name", "target.initiative_name", InputTypeString, InputSourceTargetAdapter
	case ReadInitiativeTitle:
		return "initiative.title", "target.initiative_title", InputTypeString, InputSourceTargetAdapter
	case ReadInitiativeDescription:
		return "initiative.description", "target.initiative_description", InputTypeString, InputSourceTargetAdapter
	case ReadAcceptanceCriteria:
		return "initiative.acceptance_criteria", "target.acceptance_criteria", InputTypeString, InputSourceTargetAdapter
	case ReadMemberItemsJSON:
		return "initiative.member_items", "target.member_items", InputTypeArray, InputSourceTargetAdapter
	case ReadPlanContextJSON:
		return "plan.context", "target.plan_context", InputTypeObject, InputSourceTargetAdapter
	case ReadPlanID:
		return "plan.id", "target.plan_id", InputTypeString, InputSourceTargetAdapter
	default:
		token := strings.ToLower(read)
		return "test." + token, "generic." + token, InputTypeString, InputSourceGenericProvider
	}
}

func defaultPlanDrainReads() []string {
	return []string{
		ReadOperatingMode, ReadPhase, ReadRoundNumber, ReadOperatorNote,
		ReadPriorRoundsJSON, ReadPlanID, ReadPlanContextJSON,
	}
}

func loadPlanDrainTestMode(t *testing.T, targetKind string, reads []string) (Definition, error) {
	t.Helper()
	def, err := LoadModeDefinition([]byte(planDrainTestModeJSON(targetKind, reads)))
	if err != nil {
		return Definition{}, err
	}
	return def, ValidateLoadedModes(map[Mode]Definition{def.Mode: def})
}

// TestTargetCapabilityDescriptorsMatchAdapterValues pins the one-vocabulary
// rule: target adapters implement exactly the typed capabilities the compiler
// registry declares for their target kind.
func TestTargetCapabilityDescriptorsMatchAdapterValues(t *testing.T) {
	descriptors := InputProviderCapabilities()
	for _, kind := range []TargetKind{TargetBacklogItem, TargetInitiative, TargetPlanExecution, TargetScenario} {
		adapter, err := AdapterFor(kind)
		if err != nil {
			t.Fatalf("AdapterFor(%s): %v", kind, err)
		}
		instance := TargetInstance{Kind: kind, ID: "x", Title: "x", Plan: &PlanExecutionContext{}, Initiative: InitiativeSnapshot{Name: "x"}}
		values := adapter.Values(instance)
		want := map[string]bool{}
		for id, descriptor := range descriptors {
			if descriptor.SourceKind == InputSourceTargetAdapter && containsTargetKind(descriptor.TargetKinds, kind) {
				want[id] = true
			}
		}
		if len(values) != len(want) {
			t.Fatalf("adapter %s implements %d capabilities, registry declares %d: values=%v want=%v", kind, len(values), len(want), values, want)
		}
		for id := range want {
			if _, ok := values[id]; !ok {
				t.Fatalf("adapter %s does not implement declared capability %q", kind, id)
			}
		}
	}
}

// TestBacklogItemTargetAdapterRegistered pins that backlog-item now has a runtime
// adapter that provides exactly the item-scoped capabilities the compiler
// registry declares and gives item runs a distinct ownership namespace.
func TestBacklogItemTargetAdapterRegistered(t *testing.T) {
	if !IsValidTargetKind(TargetBacklogItem) {
		t.Fatalf("backlog-item must be valid target vocabulary")
	}
	adapter, err := AdapterFor(TargetBacklogItem)
	if err != nil {
		t.Fatalf("AdapterFor(backlog-item): %v", err)
	}
	if adapter.Kind() != TargetBacklogItem {
		t.Fatalf("adapter kind = %q", adapter.Kind())
	}
	key := adapter.OwnershipKey("fix/flaky-test")
	if !strings.HasPrefix(key, "item--") {
		t.Fatalf("backlog-item ownership key %q must be item-namespaced", key)
	}
	// Its ownership namespace is disjoint from an initiative of the same name.
	if initKey, _ := OwnershipKeyFor(TargetInitiative, "fix/flaky-test"); initKey == key {
		t.Fatalf("backlog-item and initiative ownership keys collide: %q", key)
	}
}

// TestBacklogItemAdapterResolvesFromReader proves the adapter resolves an item
// target through the wired rich reader and projects its capabilities.
func TestBacklogItemAdapterResolvesFromReader(t *testing.T) {
	svc := &Service{backlogTargets: fakeBacklogTargetReader{item: BacklogItemTarget{
		Title: "Fix flaky test", Description: "It flakes on CI", Status: "in_progress",
		SpecDocument: `{"title":"Fix flaky test"}`, PlanRef: &PlanRef{Provider: "plan-manager", PlanID: "p-1"},
	}}}
	adapter := backlogItemTargetAdapter{}
	inst, err := adapter.Resolve(context.Background(), svc, Definition{}, PhaseDefinition{}, "fix/flaky-test")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if inst.ID != "fix/flaky-test" || inst.Item.Ref != "fix/flaky-test" {
		t.Fatalf("resolved instance identity = %+v", inst)
	}
	vals := adapter.Values(inst)
	if vals["target.item_title"] != "Fix flaky test" || vals["target.item_status"] != "in_progress" {
		t.Fatalf("adapter values = %+v", vals)
	}
	if vals["target.item_plan_ref"] == nil {
		t.Fatalf("plan_ref projection missing")
	}
}

type fakeBacklogTargetReader struct{ item BacklogItemTarget }

func (f fakeBacklogTargetReader) LoadBacklogItemTarget(ref string) (BacklogItemTarget, error) {
	f.item.Ref = ref
	return f.item, nil
}

func TestPlanManagerContextCarriesStableIdentityAndDigest(t *testing.T) {
	root := t.TempDir()
	client := &fakePlanExecution{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{planExecution: client})
	def, err := DefinitionFor(ModePhasedPlanDrain)
	if err != nil {
		t.Fatal(err)
	}
	phase := def.PhaseGraph.Phases[def.PhaseGraph.StartPhase]
	ctx, err := svc.resolvePlanExecution(context.Background(), def, phase, "plan-stable")
	if err != nil {
		t.Fatalf("resolvePlanExecution: %v", err)
	}
	if ctx.PlanID != "plan-stable" || ctx.ExecutionID != "exec-1" || ctx.PhaseContextDigest == "" {
		t.Fatalf("plan context provenance = %+v", ctx)
	}
	if len(client.resumeReqs) != 1 {
		t.Fatalf("resume calls = %d, want one non-advancing resolution", len(client.resumeReqs))
	}
}

// TestRunContextComposesOnlyItsTargetReads pins that a plan-target run context
// exposes plan reads and none of the initiative reads, and vice versa — reads
// are absent, not empty.
func TestRunContextComposesOnlyItsTargetReads(t *testing.T) {
	planDef, err := loadPlanDrainTestMode(t, string(TargetPlanExecution), defaultPlanDrainReads())
	if err != nil {
		t.Fatalf("load plan-target mode: %v", err)
	}
	planRC := RunContext{
		Def:      planDef,
		PhaseDef: planDef.PhaseGraph.Phases["execute"],
		Target:   TargetInstance{Kind: TargetPlanExecution, ID: "exec-123", Plan: &PlanExecutionContext{ExecutionID: "exec-123"}},
	}
	planReads, err := planRC.AvailableReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("plan AvailableReads: %v", err)
	}
	for _, forbidden := range []string{ReadInitiativeName, ReadMemberItemsJSON, ReadAcceptanceCriteria} {
		if _, ok := planReads[forbidden]; ok {
			t.Fatalf("plan-target reads must not contain %q", forbidden)
		}
	}
	if planReads[ReadPlanID] != "exec-123" {
		t.Fatalf("plan-target PLAN_ID = %q, want exec-123", planReads[ReadPlanID])
	}

	initDef, err := DefinitionFor(ModeHolisticLoop)
	if err != nil {
		t.Fatalf("DefinitionFor: %v", err)
	}
	initRC := RunContext{
		Def:      initDef,
		PhaseDef: initDef.PhaseGraph.Phases[initDef.PhaseGraph.StartPhase],
		Target: TargetInstance{
			Kind:       TargetInitiative,
			ID:         "init-a",
			Initiative: InitiativeSnapshot{Name: "init-a", Title: "Init A"},
		},
	}
	initReads, err := initRC.AvailableReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("initiative AvailableReads: %v", err)
	}
	if _, ok := initReads[ReadPlanID]; ok {
		t.Fatalf("initiative-target reads must not contain %q", ReadPlanID)
	}
	if initReads[ReadInitiativeName] != "init-a" {
		t.Fatalf("initiative-target INITIATIVE_NAME = %q, want init-a", initReads[ReadInitiativeName])
	}
}

// TestDeclaredReadsUsesCompiledBindings pins the compiled contract as the
// runtime SSOT: mutating a detached PhaseDefinition cannot inject an input
// alias that the compiled mode contract did not bind.
func TestDeclaredReadsUsesCompiledBindings(t *testing.T) {
	def, err := loadPlanDrainTestMode(t, string(TargetPlanExecution), defaultPlanDrainReads())
	if err != nil {
		t.Fatalf("load plan-target mode: %v", err)
	}
	phaseDef := def.PhaseGraph.Phases["execute"]
	phaseDef.Reads = append(phaseDef.Reads, ReadMemberItemsJSON)
	rc := RunContext{Def: def, PhaseDef: phaseDef, Target: TargetInstance{
		Kind: TargetPlanExecution, ID: "exec-123", Plan: &PlanExecutionContext{ExecutionID: "exec-123"},
	}}
	reads, err := rc.DeclaredReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("DeclaredReads: %v", err)
	}
	if _, injected := reads[ReadMemberItemsJSON]; injected {
		t.Fatalf("detached phase mutation injected %q outside the compiled contract", ReadMemberItemsJSON)
	}
}

// TestLoaderRejectsForeignAdapterRead: a plan-target mode declaring an
// initiative-adapter read fails load-time read-side validation with an error
// naming the providing target.
func TestLoaderRejectsForeignAdapterRead(t *testing.T) {
	_, err := loadPlanDrainTestMode(t, string(TargetPlanExecution), append(defaultPlanDrainReads(), ReadMemberItemsJSON))
	if err == nil || !strings.Contains(err.Error(), "does not support target") {
		t.Fatalf("err = %v, want target-capability mismatch validation error", err)
	}
	if !strings.Contains(err.Error(), string(TargetPlanExecution)) {
		t.Fatalf("err = %v, want error naming the incompatible target %q", err, TargetPlanExecution)
	}
}

// TestLoaderRejectsUnsatisfiableReadSlot: a declared read no provider supplies
// is an unsatisfiable template slot and fails the load.
func TestLoaderRejectsUnsatisfiableReadSlot(t *testing.T) {
	_, err := loadPlanDrainTestMode(t, string(TargetPlanExecution), append(defaultPlanDrainReads(), "NOT_A_REAL_READ"))
	if err == nil || !strings.Contains(err.Error(), "unavailable capability") {
		t.Fatalf("err = %v, want unavailable-capability validation error", err)
	}
}

// TestLoaderRejectsMissingReads: every phase must declare its input contract.
func TestLoaderRejectsMissingReads(t *testing.T) {
	doc := strings.Replace(planDrainTestModeJSON(string(TargetPlanExecution), nil), `"reads": [],`, ``, 1)
	if _, err := LoadModeDefinition([]byte(doc)); err == nil {
		t.Fatalf("LoadModeDefinition accepted a phase without reads")
	}
}

// TestLoaderRejectsUnknownTargetKind: an unknown/invalid target kind is
// rejected at load (schema enum + typed loader check).
func TestLoaderRejectsUnknownTargetKind(t *testing.T) {
	if _, err := LoadModeDefinition([]byte(planDrainTestModeJSON("galaxy", defaultPlanDrainReads()))); err == nil {
		t.Fatalf("LoadModeDefinition accepted unknown target kind")
	}
}

// TestPlanTargetPhasePromptRendersDeclaredReads registers the synthetic
// plan-target mode and renders its execute prompt through the real render
// seam: the substituted variable map is exactly the declared reads — plan
// reads present, initiative reads absent.
func TestPlanTargetPhasePromptRendersDeclaredReads(t *testing.T) {
	def, err := loadPlanDrainTestMode(t, string(TargetPlanExecution), defaultPlanDrainReads())
	if err != nil {
		t.Fatalf("load plan-target mode: %v", err)
	}
	withSyntheticModeRegistry(t, def)
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})

	rc := RunContext{
		Def:      def,
		PhaseDef: def.PhaseGraph.Phases["execute"],
		Target:   TargetInstance{Kind: TargetPlanExecution, ID: "exec-123", Plan: &PlanExecutionContext{ExecutionID: "exec-123", PlanID: "plan-9"}},
	}
	rendered, err := svc.renderPhasePrompt(context.Background(), rc, RoundEnvelope{Round: 2}, "drain the next slice")
	if err != nil {
		t.Fatalf("renderPhasePrompt: %v", err)
	}
	if len(rendered.Variables) != len(rc.PhaseDef.Reads) {
		t.Fatalf("rendered variables = %d entries, want exactly the %d declared reads", len(rendered.Variables), len(rc.PhaseDef.Reads))
	}
	if !strings.Contains(rendered.Prompt, "PLAN_ID=exec-123") {
		t.Fatalf("prompt missing substituted PLAN_ID:\n%s", rendered.Prompt)
	}
	if strings.Contains(rendered.Prompt, ReadInitiativeName+"=") || strings.Contains(rendered.Prompt, ReadMemberItemsJSON+"=") {
		t.Fatalf("plan-target prompt must not carry initiative reads:\n%s", rendered.Prompt)
	}
}

// TestRenderPhasePromptRejectsUnsatisfiedTemplateSlot: a rendered prompt with
// a leftover {{VARIABLE}} slot fails typed instead of reaching an agent with
// an unfilled template.
func TestRenderPhasePromptRejectsUnsatisfiedTemplateSlot(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{
		render: func(string, map[string]string) string { return "do the work using {{SOME_UNDECLARED_READ}}" },
	}})
	def, err := DefinitionFor(ModeHolisticLoop)
	if err != nil {
		t.Fatalf("DefinitionFor: %v", err)
	}
	rc := RunContext{
		Def:      def,
		PhaseDef: def.PhaseGraph.Phases[def.PhaseGraph.StartPhase],
		Target:   TargetInstance{Kind: TargetInitiative, ID: "init-a", Initiative: InitiativeSnapshot{Name: "init-a"}},
	}
	_, err = svc.renderPhasePrompt(context.Background(), rc, RoundEnvelope{Round: 1}, "")
	if err == nil || !strings.Contains(err.Error(), "unsatisfied template slot") {
		t.Fatalf("renderPhasePrompt = %v, want unsatisfied-template-slot error", err)
	}
	if !strings.Contains(err.Error(), "SOME_UNDECLARED_READ") {
		t.Fatalf("error should name the offending slot: %v", err)
	}
}

// TestOwnershipKeysPerTarget pins the lock-identity contract: initiative
// targets keep the initiative name; plan targets get plan-scoped keys that
// never collide with initiative names; refresh/cancel derive the same key from
// a persisted round.
func TestOwnershipKeysPerTarget(t *testing.T) {
	if key, _ := OwnershipKeyFor(TargetInitiative, "init-a"); key != "init-a" {
		t.Fatalf("initiative ownership key = %q, want init-a", key)
	}
	planKey, err := OwnershipKeyFor(TargetPlanExecution, "exec-123")
	if err != nil || planKey != "plan--exec-123" {
		t.Fatalf("plan ownership key = %q (%v), want plan--exec-123", planKey, err)
	}
	if _, err := OwnershipKeyFor(TargetKind("galaxy"), "x"); err == nil {
		t.Fatalf("OwnershipKeyFor accepted unknown target kind")
	}

	round := RoundEnvelope{ScopeKind: string(TargetPlanExecution), ScopeID: "exec-123"}
	if got := roundOwnershipKey(round); got != "plan--exec-123" {
		t.Fatalf("roundOwnershipKey(plan round) = %q, want plan--exec-123", got)
	}
	legacy := RoundEnvelope{InitiativeName: "init-a"}
	if got := roundOwnershipKey(legacy); got != "init-a" {
		t.Fatalf("roundOwnershipKey(legacy round) = %q, want init-a", got)
	}
}

// TestPhaseReadsSummaryGroupsByProvider pins the catalog/UI metadata grouping:
// declared reads split into base vs target-adapter groups, from data.
func TestPhaseReadsSummaryGroupsByProvider(t *testing.T) {
	def, err := loadPlanDrainTestMode(t, string(TargetPlanExecution), []string{ReadOperatingMode, ReadPlanID, ReadPriorRoundsJSON, ReadPlanContextJSON})
	if err != nil {
		t.Fatalf("load plan drain mode: %v", err)
	}
	summary := summarizePhaseReads(def, def.PhaseGraph.Phases["execute"])
	if strings.Join(summary.Base, ",") != ReadOperatingMode+","+ReadPriorRoundsJSON {
		t.Fatalf("base group = %v", summary.Base)
	}
	if strings.Join(summary.Target, ",") != ReadPlanID+","+ReadPlanContextJSON {
		t.Fatalf("target group = %v", summary.Target)
	}
}
