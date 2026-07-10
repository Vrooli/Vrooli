package operatingmode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// planDrainTestModeJSON is a minimal, valid plan-manager-plan-target mode used
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
		retention := InputRetentionValue
		if read == ReadPlanContent {
			retention = InputRetentionDigest
		}
		contract.Specs = append(contract.Specs, InputSpec{
			ID: id, Type: valueType, Required: true, Sensitivity: InputSensitivityInternal,
			Retention: retention, Description: "Test input for " + read + ".",
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
	case ReadPlanPath:
		return "plan.path", "target.plan_path", InputTypeString, InputSourceTargetAdapter
	case ReadPlanContent:
		return "plan.content", "target.plan_content", InputTypeString, InputSourceTargetAdapter
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
	for _, kind := range []TargetKind{TargetPlanManagerPlan, TargetPlanRef, TargetInitiative} {
		adapter, err := AdapterFor(kind)
		if err != nil {
			t.Fatalf("AdapterFor(%s): %v", kind, err)
		}
		instance := TargetInstance{Kind: kind, ID: "x", PlanPath: "plan.md", PlanContent: "# Plan", Plan: &PlanExecutionContext{}, Initiative: InitiativeSnapshot{Name: "x"}}
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

func TestPlanRefTargetResolvesContainedBoundedUTF8Content(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "# Safe plan\n\nDo the work.\n"
	if err := os.WriteFile(filepath.Join(root, "plans", "safe.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	def, err := loadPlanDrainTestMode(t, string(TargetPlanRef), []string{ReadPlanPath, ReadPlanContent, ReadPlanContextJSON})
	if err != nil {
		t.Fatalf("load plan-ref mode: %v", err)
	}
	svc := newTestServiceWithOptions(t, root, serviceOptions{})
	adapter, err := AdapterFor(TargetPlanRef)
	if err != nil {
		t.Fatal(err)
	}
	target, err := adapter.Resolve(context.Background(), svc, def, def.PhaseGraph.Phases["execute"], "plans/../plans/safe.md")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if target.ID != "plans/safe.md" || target.PlanPath != "plans/safe.md" {
		t.Fatalf("canonical target path = id:%q path:%q", target.ID, target.PlanPath)
	}
	if target.PlanContent != content {
		t.Fatalf("plan content = %q, want %q", target.PlanContent, content)
	}
	if target.PlanContentHash == "" || target.Plan == nil || target.Plan.ContentHash != target.PlanContentHash {
		t.Fatalf("plan content provenance = target:%q context:%+v", target.PlanContentHash, target.Plan)
	}
	reads, err := (RunContext{Def: def, PhaseDef: def.PhaseGraph.Phases["execute"], Target: target}).DeclaredReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("DeclaredReads: %v", err)
	}
	if reads[ReadPlanPath] != "plans/safe.md" || reads[ReadPlanContent] != content {
		t.Fatalf("plan-ref reads = %+v", reads)
	}
	compiled, err := CompileInputContract(map[Mode]Definition{def.Mode: def}, def)
	if err != nil {
		t.Fatalf("CompileInputContract: %v", err)
	}
	compiledJSON, err := json.Marshal(compiled)
	if err != nil {
		t.Fatal(err)
	}
	retention, err := promptInputRetentionTrace(OperatingModeExecution{CompiledInputContract: compiledJSON}, PinnedPromptSource{
		Mode: string(def.Mode), Phase: "execute",
	}, reads)
	if err != nil {
		t.Fatalf("promptInputRetentionTrace: %v", err)
	}
	contentRetention, ok := retention[ReadPlanContent].(map[string]any)
	if !ok || contentRetention["retention"] != InputRetentionDigest || contentRetention["value_digest"] == "" {
		t.Fatalf("plan content retention trace = %+v", retention[ReadPlanContent])
	}
	if _, leaked := contentRetention["value"]; leaked {
		t.Fatalf("plan content retention trace leaked raw content: %+v", contentRetention)
	}
}

func TestResolveWorkspacePlanRefRejectsUnsafeOrInvalidContent(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "plans", "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "plans", "real", "safe.md"), []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("real", "safe.md"), filepath.Join(root, "plans", "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(root, "plans", "link-dir")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "invalid.md"), []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "oversized.md"), []byte(strings.Repeat("x", maxPlanRefContentBytes+1)), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		ref  string
		want string
	}{
		{name: "absolute", ref: outside, want: "workspace-relative"},
		{name: "traversal", ref: "../outside.md", want: "workspace-relative"},
		{name: "symlink file", ref: "plans/link.md", want: "symlink component"},
		{name: "symlink directory", ref: "plans/link-dir/safe.md", want: "symlink component"},
		{name: "directory", ref: "plans", want: "regular file"},
		{name: "invalid utf8", ref: "invalid.md", want: "valid UTF-8"},
		{name: "oversized", ref: "oversized.md", want: "exceeds maximum"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := resolveWorkspacePlanRef(root, tt.ref)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("resolveWorkspacePlanRef(%q) error = %v, want %q", tt.ref, err, tt.want)
			}
		})
	}
}

func TestStartTargetPhaseRejectsUnsafePlanRefBeforeAnyMutation(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("# outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked-plan.md")); err != nil {
		t.Fatal(err)
	}
	def, err := loadPlanDrainTestMode(t, string(TargetPlanRef), []string{ReadPlanPath, ReadPlanContent, ReadPlanContextJSON})
	if err != nil {
		t.Fatalf("load plan-ref mode: %v", err)
	}
	withSyntheticModeRegistry(t, def)
	agent := &fakeAgent{}
	prompts := &fakePrompts{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{agent: agent, prompts: prompts})
	_, err = svc.StartTargetPhase(context.Background(), StartTargetPhaseRequest{
		Mode: string(def.Mode), TargetRef: "linked-plan.md",
	})
	if err == nil || !strings.Contains(err.Error(), "symlink component") {
		t.Fatalf("StartTargetPhase error = %v, want symlink rejection", err)
	}
	if len(agent.spawned) != 0 || len(prompts.sourceCalls) != 0 {
		t.Fatalf("unsafe plan ref caused external preflight: spawned=%d prompt_sources=%v", len(agent.spawned), prompts.sourceCalls)
	}
	if _, statErr := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe plan ref created target state: %v", statErr)
	}
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
	ctx, err := svc.resolvePlanManagerPlan(context.Background(), def, phase, "plan-stable")
	if err != nil {
		t.Fatalf("resolvePlanManagerPlan: %v", err)
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
	planDef, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), defaultPlanDrainReads())
	if err != nil {
		t.Fatalf("load plan-target mode: %v", err)
	}
	planRC := RunContext{
		Def:      planDef,
		PhaseDef: planDef.PhaseGraph.Phases["execute"],
		Target:   TargetInstance{Kind: TargetPlanManagerPlan, ID: "exec-123", Plan: &PlanExecutionContext{ExecutionID: "exec-123"}},
	}
	planReads, err := planRC.AvailableReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("plan AvailableReads: %v", err)
	}
	for _, forbidden := range []string{ReadInitiativeName, ReadMemberItemsJSON, ReadAcceptanceCriteria, ReadPlanPath} {
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
	for _, forbidden := range []string{ReadPlanID, ReadPlanPath} {
		if _, ok := initReads[forbidden]; ok {
			t.Fatalf("initiative-target reads must not contain %q", forbidden)
		}
	}
	if initReads[ReadInitiativeName] != "init-a" {
		t.Fatalf("initiative-target INITIATIVE_NAME = %q, want init-a", initReads[ReadInitiativeName])
	}
}

// TestDeclaredReadsUsesCompiledBindings pins the compiled contract as the
// runtime SSOT: mutating a detached PhaseDefinition cannot inject an input
// alias that the compiled mode contract did not bind.
func TestDeclaredReadsUsesCompiledBindings(t *testing.T) {
	def, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), defaultPlanDrainReads())
	if err != nil {
		t.Fatalf("load plan-target mode: %v", err)
	}
	phaseDef := def.PhaseGraph.Phases["execute"]
	phaseDef.Reads = append(phaseDef.Reads, ReadMemberItemsJSON)
	rc := RunContext{Def: def, PhaseDef: phaseDef, Target: TargetInstance{
		Kind: TargetPlanManagerPlan, ID: "exec-123", Plan: &PlanExecutionContext{ExecutionID: "exec-123"},
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
	_, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), append(defaultPlanDrainReads(), ReadMemberItemsJSON))
	if err == nil || !strings.Contains(err.Error(), "does not support target") {
		t.Fatalf("err = %v, want target-capability mismatch validation error", err)
	}
	if !strings.Contains(err.Error(), string(TargetPlanManagerPlan)) {
		t.Fatalf("err = %v, want error naming the incompatible target %q", err, TargetPlanManagerPlan)
	}
}

// TestLoaderRejectsUnsatisfiableReadSlot: a declared read no provider supplies
// is an unsatisfiable template slot and fails the load.
func TestLoaderRejectsUnsatisfiableReadSlot(t *testing.T) {
	_, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), append(defaultPlanDrainReads(), "NOT_A_REAL_READ"))
	if err == nil || !strings.Contains(err.Error(), "unavailable capability") {
		t.Fatalf("err = %v, want unavailable-capability validation error", err)
	}
}

// TestLoaderRejectsMissingReads: every phase must declare its input contract.
func TestLoaderRejectsMissingReads(t *testing.T) {
	doc := strings.Replace(planDrainTestModeJSON(string(TargetPlanManagerPlan), nil), `"reads": [],`, ``, 1)
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
	def, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), defaultPlanDrainReads())
	if err != nil {
		t.Fatalf("load plan-target mode: %v", err)
	}
	withSyntheticModeRegistry(t, def)
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{prompts: &fakePrompts{render: echoRender}})

	rc := RunContext{
		Def:      def,
		PhaseDef: def.PhaseGraph.Phases["execute"],
		Target:   TargetInstance{Kind: TargetPlanManagerPlan, ID: "exec-123", Plan: &PlanExecutionContext{ExecutionID: "exec-123", PlanID: "plan-9"}},
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
	planKey, err := OwnershipKeyFor(TargetPlanManagerPlan, "exec-123")
	if err != nil || planKey != "plan--exec-123" {
		t.Fatalf("plan ownership key = %q (%v), want plan--exec-123", planKey, err)
	}
	refKey, err := OwnershipKeyFor(TargetPlanRef, "docs/plans/foo.md")
	if err != nil || refKey != "plan-ref--docs-plans-foo.md" {
		t.Fatalf("plan-ref ownership key = %q (%v), want sanitized path key", refKey, err)
	}
	if _, err := OwnershipKeyFor(TargetKind("galaxy"), "x"); err == nil {
		t.Fatalf("OwnershipKeyFor accepted unknown target kind")
	}

	round := RoundEnvelope{ScopeKind: string(TargetPlanManagerPlan), ScopeID: "exec-123"}
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
	def, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), []string{ReadOperatingMode, ReadPlanID, ReadPriorRoundsJSON, ReadPlanContextJSON})
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
