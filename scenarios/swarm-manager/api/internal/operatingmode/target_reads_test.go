package operatingmode

import (
	"context"
	"sort"
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

// TestBaseAndAdapterReadsComposeByUnion pins the D1b composition rule: for
// every target kind the available read set is exactly base ∪ adapter (no
// conditional emptiness, no scope switch), and each adapter resolves exactly
// the reads it declares.
func TestBaseAndAdapterReadsComposeByUnion(t *testing.T) {
	for _, kind := range []TargetKind{TargetPlanManagerPlan, TargetPlanRef, TargetInitiative} {
		adapter, err := AdapterFor(kind)
		if err != nil {
			t.Fatalf("AdapterFor(%s): %v", kind, err)
		}
		want := append(BaseReadNames(), adapter.Provides()...)
		sort.Strings(want)
		got, err := AvailableReadNames(kind)
		if err != nil {
			t.Fatalf("AvailableReadNames(%s): %v", kind, err)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("AvailableReadNames(%s) = %v, want base ∪ adapter = %v", kind, got, want)
		}
		// The adapter resolves exactly its declared reads.
		resolved := adapter.Reads(TargetInstance{Kind: kind, ID: "x"})
		if len(resolved) != len(adapter.Provides()) {
			t.Fatalf("adapter %s resolves %d reads, declares %d", kind, len(resolved), len(adapter.Provides()))
		}
		for _, name := range adapter.Provides() {
			if _, ok := resolved[name]; !ok {
				t.Fatalf("adapter %s declares %q but does not resolve it", kind, name)
			}
		}
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

// TestDeclaredReadsRejectsForeignRead pins the runtime twin of the loader's
// read-side validation: a phase whose declared contract references a read the
// target does not provide fails typed instead of rendering empty.
func TestDeclaredReadsRejectsForeignRead(t *testing.T) {
	def, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), defaultPlanDrainReads())
	if err != nil {
		t.Fatalf("load plan-target mode: %v", err)
	}
	phaseDef := def.PhaseGraph.Phases["execute"]
	phaseDef.Reads = append(phaseDef.Reads, ReadMemberItemsJSON)
	rc := RunContext{Def: def, PhaseDef: phaseDef, Target: TargetInstance{Kind: TargetPlanManagerPlan, ID: "exec-123"}}
	if _, err := rc.DeclaredReads(RoundEnvelope{Round: 1}, ""); err == nil || !strings.Contains(err.Error(), "does not provide") {
		t.Fatalf("DeclaredReads = %v, want does-not-provide error", err)
	}
}

// TestLoaderRejectsForeignAdapterRead: a plan-target mode declaring an
// initiative-adapter read fails load-time read-side validation with an error
// naming the providing target.
func TestLoaderRejectsForeignAdapterRead(t *testing.T) {
	_, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), append(defaultPlanDrainReads(), ReadMemberItemsJSON))
	if err == nil || !strings.Contains(err.Error(), "does not provide") {
		t.Fatalf("err = %v, want target-does-not-provide read validation error", err)
	}
	if !strings.Contains(err.Error(), string(TargetInitiative)) {
		t.Fatalf("err = %v, want error naming the providing target %q", err, TargetInitiative)
	}
}

// TestLoaderRejectsUnsatisfiableReadSlot: a declared read no provider supplies
// is an unsatisfiable template slot and fails the load.
func TestLoaderRejectsUnsatisfiableReadSlot(t *testing.T) {
	_, err := loadPlanDrainTestMode(t, string(TargetPlanManagerPlan), append(defaultPlanDrainReads(), "NOT_A_REAL_READ"))
	if err == nil || !strings.Contains(err.Error(), "unsatisfiable") {
		t.Fatalf("err = %v, want unsatisfiable-slot read validation error", err)
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
	summary := summarizePhaseReads([]string{ReadOperatingMode, ReadPlanID, ReadPriorRoundsJSON, ReadPlanContextJSON})
	if strings.Join(summary.Base, ",") != ReadOperatingMode+","+ReadPriorRoundsJSON {
		t.Fatalf("base group = %v", summary.Base)
	}
	if strings.Join(summary.Target, ",") != ReadPlanID+","+ReadPlanContextJSON {
		t.Fatalf("target group = %v", summary.Target)
	}
}
