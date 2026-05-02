package operatingmode

import (
	"testing"
)

func findPhase(t *testing.T, entry ModeCatalogEntry, name string) ModeCatalogPhase {
	t.Helper()
	for _, phase := range entry.Phases {
		if phase.Phase == name {
			return phase
		}
	}
	t.Fatalf("phase %q not present in entry; got %v", name, phaseNames(entry.Phases))
	return ModeCatalogPhase{}
}

func phaseNames(phases []ModeCatalogPhase) []string {
	out := make([]string, len(phases))
	for i, phase := range phases {
		out[i] = phase.Phase
	}
	return out
}

func transitionsBetween(graph *ModeCatalogPhaseGraph, from, to string) []ModeCatalogTransition {
	if graph == nil {
		return nil
	}
	out := make([]ModeCatalogTransition, 0)
	for _, edge := range graph.Transitions {
		if edge.From == from && edge.To == to {
			out = append(out, edge)
		}
	}
	return out
}

func TestBuildCatalogEntry_HolisticLoop(t *testing.T) {
	entry := buildCatalogEntry(holisticLoopDefinition(), 0)
	if entry.Mode != string(ModeHolisticLoop) {
		t.Fatalf("entry.Mode = %q, want %q", entry.Mode, ModeHolisticLoop)
	}
	if !entry.SupportsPhases {
		t.Fatalf("expected supports_phases=true for holistic-loop")
	}
	if got, want := len(entry.Phases), 4; got != want {
		t.Fatalf("phase count = %d, want %d (%v)", got, want, phaseNames(entry.Phases))
	}

	investigate := findPhase(t, entry, "investigate")
	if !investigate.IsStart {
		t.Fatalf("investigate.IsStart = false, want true")
	}
	if investigate.IsTerminal {
		t.Fatalf("investigate.IsTerminal = true, want false")
	}
	if investigate.Title == "" || investigate.Purpose == "" || investigate.Trigger == "" {
		t.Fatalf("investigate prompt-catalog metadata missing: %#v", investigate)
	}
	if got, want := investigate.Label, "Investigate"; got != want {
		t.Fatalf("investigate.Label = %q, want %q (must be the humanized phase ID, free of mode prefix)", got, want)
	}
	if got, want := investigate.Title, "Holistic Loop Investigate"; got != want {
		t.Fatalf("investigate.Title = %q, want %q (PromptCatalog title must remain mode-prefixed for catalog disambiguation)", got, want)
	}
	if investigate.CatalogID == "" || investigate.SkillID == "" {
		t.Fatalf("investigate catalog/skill IDs missing: %#v", investigate)
	}
	if len(investigate.OutputArtifacts) == 0 {
		t.Fatalf("investigate output_artifacts empty")
	}
	if got, want := investigate.OutputArtifacts[0].Path, "modes/holistic-loop/findings.md"; got != want {
		t.Fatalf("investigate artifact path = %q, want %q", got, want)
	}
	if !investigate.OutputArtifacts[0].Required {
		t.Fatalf("investigate artifact must be required")
	}
	if !investigate.OutputContract.RequiresStructuredResult {
		t.Fatalf("investigate output_contract.requires_structured_result must be true")
	}
	if investigate.OutputContract.RequiredArtifactCount == 0 {
		t.Fatalf("investigate output_contract.required_artifact_count must be > 0")
	}

	execute := findPhase(t, entry, "execute")
	if !execute.WritesRepo {
		t.Fatalf("execute.WritesRepo = false, want true")
	}
	if !execute.SamplesReplanRate {
		t.Fatalf("execute.SamplesReplanRate = false, want true")
	}
	if execute.SamplesAcceptanceRate {
		t.Fatalf("execute.SamplesAcceptanceRate = true, want false")
	}

	review := findPhase(t, entry, "review")
	if !review.IsTerminal {
		t.Fatalf("review.IsTerminal = false, want true")
	}
	if !review.RequiresCriteria {
		t.Fatalf("review.RequiresCriteria = false, want true")
	}
	if !review.OutputContract.RequiresVerdict {
		t.Fatalf("review.OutputContract.RequiresVerdict = false, want true")
	}
	if !review.SamplesAcceptanceRate {
		t.Fatalf("review.SamplesAcceptanceRate = false, want true")
	}

	plan := findPhase(t, entry, "plan")
	if plan.WritesRepo {
		t.Fatalf("plan.WritesRepo = true, want false")
	}
}

func TestBuildCatalogEntry_HolisticLoop_PhaseGraph(t *testing.T) {
	entry := buildCatalogEntry(holisticLoopDefinition(), 0)
	graph := entry.PhaseGraph
	if graph == nil {
		t.Fatalf("PhaseGraph is nil for holistic-loop")
	}
	if graph.StartPhase != "investigate" {
		t.Fatalf("start_phase = %q, want investigate", graph.StartPhase)
	}
	if got, want := graph.Terminal, []string{"review"}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("terminal = %v, want %v", got, want)
	}

	executeReplan := transitionsBetween(graph, "execute", "investigate")
	if len(executeReplan) != 1 {
		t.Fatalf("execute->investigate transitions = %v, want 1", executeReplan)
	}
	if executeReplan[0].ConditionKind != string(TransitionConditionPayloadBool) {
		t.Fatalf("execute->investigate kind = %q, want payload_bool", executeReplan[0].ConditionKind)
	}
	if executeReplan[0].Label != "on payload.replan_needed=true" {
		t.Fatalf("execute->investigate label = %q, want %q", executeReplan[0].Label, "on payload.replan_needed=true")
	}

	executeReview := transitionsBetween(graph, "execute", "review")
	if len(executeReview) != 1 {
		t.Fatalf("execute->review transitions = %v, want 1", executeReview)
	}
	if executeReview[0].ConditionKind != string(TransitionConditionAlways) {
		t.Fatalf("execute->review kind = %q, want always", executeReview[0].ConditionKind)
	}
	if executeReview[0].Label != "always" {
		t.Fatalf("execute->review label = %q, want always", executeReview[0].Label)
	}

	// Edge ordering should be deterministic ((from, to, label) ascending).
	for i := 1; i < len(graph.Transitions); i++ {
		prev, cur := graph.Transitions[i-1], graph.Transitions[i]
		if prev.From > cur.From {
			t.Fatalf("transitions not sorted by from: %v", graph.Transitions)
		}
		if prev.From == cur.From && prev.To > cur.To {
			t.Fatalf("transitions not sorted by to within from: %v", graph.Transitions)
		}
	}
}

func TestBuildCatalogEntry_PhasedPlanDrain_ProgressTransitions(t *testing.T) {
	entry := buildCatalogEntry(phasedPlanDrainDefinition(), 0)
	graph := entry.PhaseGraph
	if graph == nil {
		t.Fatalf("PhaseGraph is nil for phased-plan-drain")
	}

	expectations := map[string]string{
		"execute_next": "on continue",
		"prepare_plan": "on replan",
		"review":       "on complete",
	}
	for to, label := range expectations {
		edges := transitionsBetween(graph, "classify_progress", to)
		if len(edges) != 1 {
			t.Fatalf("classify_progress->%s edges = %v, want 1", to, edges)
		}
		if edges[0].ConditionKind != string(TransitionConditionProgressDecision) {
			t.Fatalf("classify_progress->%s kind = %q, want progress_decision", to, edges[0].ConditionKind)
		}
		if edges[0].Label != label {
			t.Fatalf("classify_progress->%s label = %q, want %q", to, edges[0].Label, label)
		}
		if edges[0].ProgressDecision == "" {
			t.Fatalf("classify_progress->%s missing progress_decision in wire shape", to)
		}
	}

	executeNext := findPhase(t, entry, "execute_next")
	if !executeNext.OutputContract.RequiresHandoff {
		t.Fatalf("execute_next.RequiresHandoff = false, want true")
	}

	classify := findPhase(t, entry, "classify_progress")
	if !classify.OutputContract.RequiresProgress {
		t.Fatalf("classify_progress.RequiresProgress = false, want true")
	}
	if got, want := classify.Label, "Classify Progress"; got != want {
		t.Fatalf("classify_progress.Label = %q, want %q (humanized phase ID; no mode prefix)", got, want)
	}
	if got, want := classify.Title, "Phased Plan Classify Progress"; got != want {
		t.Fatalf("classify_progress.Title = %q, want %q (PromptCatalog title stays mode-prefixed)", got, want)
	}
	if len(classify.ResultBindings) == 0 {
		t.Fatalf("classify_progress.ResultBindings empty")
	}
	if classify.ResultBindings[0].Kind != ResultBindingProgressArtifact {
		t.Fatalf("classify_progress binding kind = %q, want progress_artifact", classify.ResultBindings[0].Kind)
	}

	preparePlan := findPhase(t, entry, "prepare_plan")
	if got, want := preparePlan.Label, "Prepare Plan"; got != want {
		t.Fatalf("prepare_plan.Label = %q, want %q", got, want)
	}
	executeNextPhase := findPhase(t, entry, "execute_next")
	if got, want := executeNextPhase.Label, "Execute Next"; got != want {
		t.Fatalf("execute_next.Label = %q, want %q", got, want)
	}
}

func TestBuildCatalogEntry_ItemLevel_NoPhaseGraph(t *testing.T) {
	def, err := DefinitionFor(ModeItemLevel)
	if err != nil {
		t.Fatalf("DefinitionFor item-level: %v", err)
	}
	entry := buildCatalogEntry(def, 0)
	if entry.SupportsPhases {
		t.Fatalf("item-level should not advertise supports_phases")
	}
	if len(entry.Phases) != 0 {
		t.Fatalf("item-level phases = %v, want empty", phaseNames(entry.Phases))
	}
	if entry.PhaseGraph != nil {
		t.Fatalf("item-level PhaseGraph = %#v, want nil", entry.PhaseGraph)
	}
}
