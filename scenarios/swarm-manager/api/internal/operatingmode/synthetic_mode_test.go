package operatingmode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
)

const modeSyntheticHarness Mode = "synthetic-harness"

func TestSyntheticModeHarnessExercisesAuthoringPolicies(t *testing.T) {
	if ValidateMode(string(modeSyntheticHarness)) {
		t.Fatalf("%q leaked into the production registry before the test overlay", modeSyntheticHarness)
	}

	def := syntheticHarnessDefinition()
	withSyntheticModeRegistry(t, def)

	if err := ValidateRegistry(); err != nil {
		t.Fatalf("ValidateRegistry with synthetic mode returned error: %v", err)
	}
	if err := ValidatePromptCatalog(ExpectedPromptCatalogEntry); err != nil {
		t.Fatalf("ValidatePromptCatalog with synthetic mode returned error: %v", err)
	}
	if !def.Metrics.CountsReplanSample("decide") ||
		!def.Metrics.CountsAcceptanceSample("review") ||
		!def.Metrics.IsAcceptedVerdict("accepted") {
		t.Fatalf("synthetic metrics policy = %+v, want decide replan and review acceptance semantics", def.Metrics)
	}

	root := t.TempDir()
	agent := &fakeAgent{states: map[string]agentmanager.RunState{}}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		agent: agent,
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-synthetic": {
				Name:               "init-synthetic",
				Title:              "Synthetic Initiative",
				Description:        "Exercises test-only operating-mode behavior.",
				Mode:               string(modeSyntheticHarness),
				Items:              []string{"execute/synthetic"},
				AcceptanceCriteria: []string{"review synthetic output"},
			},
		}},
	})

	catalog, err := svc.Catalog()
	if err != nil {
		t.Fatalf("Catalog returned error: %v", err)
	}
	syntheticEntry, ok := catalogModeByID(catalog, modeSyntheticHarness)
	if !ok {
		t.Fatalf("catalog did not include test overlay mode %q", modeSyntheticHarness)
	}
	if syntheticEntry.Default || !syntheticEntry.Switchable {
		t.Fatalf("synthetic catalog entry default/switchable = %v/%v, want false/true", syntheticEntry.Default, syntheticEntry.Switchable)
	}
	if !syntheticEntry.Capabilities.SupportsPhases ||
		!syntheticEntry.Capabilities.CanStartPhases ||
		!syntheticEntry.Capabilities.SupportsArtifacts ||
		!syntheticEntry.Capabilities.RequiresAcceptanceCriteria {
		t.Fatalf("synthetic capabilities = %+v, want phase/artifact/criteria support", syntheticEntry.Capabilities)
	}

	assess := startSyntheticPhase(t, svc, "init-synthetic", "assess")
	completeSyntheticRun(t, agent, assess.RunID, `{
		"operating_mode_result": {
			"handoff": {"summary": "assessment complete"}
		}
	}`)
	refreshSyntheticRound(t, svc, "init-synthetic", assess.Round)

	decide := startSyntheticPhase(t, svc, "init-synthetic", "decide")
	completeSyntheticRun(t, agent, decide.RunID, `{
		"operating_mode_result": {
			"progress": {
				"decision": "replan",
				"current_phase": "assess",
				"rationale": "more assessment is needed"
			}
		}
	}`)
	replanned := refreshSyntheticRound(t, svc, "init-synthetic", decide.Round)
	if progress, ok := RoundPayload(replanned.Payload).Progress(); !ok || progress.Decision != ProgressReplan {
		t.Fatalf("replanned payload progress = %+v/%v, want replan decision", progress, ok)
	}

	progressPath := filepath.Join(root, "initiatives", "init-synthetic", "modes", "synthetic-harness", "progress.json")
	progressBytes, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("read derived progress artifact: %v", err)
	}
	var progress ProgressState
	if err := json.Unmarshal(progressBytes, &progress); err != nil {
		t.Fatalf("decode derived progress artifact: %v", err)
	}
	if progress.Decision != ProgressReplan || strings.TrimSpace(progress.UpdatedAt) == "" {
		t.Fatalf("derived progress artifact = %+v, want replan with updated_at", progress)
	}

	replanWorkspace, err := svc.Workspace(context.Background(), "init-synthetic")
	if err != nil {
		t.Fatalf("Workspace after replan returned error: %v", err)
	}
	replanActions := workspacePhaseActions(replanWorkspace)
	if !replanActions["assess"].Startable {
		t.Fatalf("assess action after replan = %+v, want startable", replanActions["assess"])
	}
	if replanActions["review"].Startable {
		t.Fatalf("review action after replan = %+v, want blocked", replanActions["review"])
	}

	assessAgain := startSyntheticPhase(t, svc, "init-synthetic", "assess")
	completeSyntheticRun(t, agent, assessAgain.RunID, `{
		"operating_mode_result": {
			"handoff": {"summary": "assessment confirmed"}
		}
	}`)
	refreshSyntheticRound(t, svc, "init-synthetic", assessAgain.Round)

	decideAgain := startSyntheticPhase(t, svc, "init-synthetic", "decide")
	completeSyntheticRun(t, agent, decideAgain.RunID, `{
		"operating_mode_result": {
			"progress": {
				"decision": "complete",
				"current_phase": "review",
				"rationale": "ready for review"
			}
		}
	}`)
	completed := refreshSyntheticRound(t, svc, "init-synthetic", decideAgain.Round)
	if progress, ok := RoundPayload(completed.Payload).Progress(); !ok || progress.Decision != ProgressComplete {
		t.Fatalf("completed payload progress = %+v/%v, want complete decision", progress, ok)
	}

	reviewWorkspace, err := svc.Workspace(context.Background(), "init-synthetic")
	if err != nil {
		t.Fatalf("Workspace after complete returned error: %v", err)
	}
	reviewActions := workspacePhaseActions(reviewWorkspace)
	if !reviewActions["review"].Startable {
		t.Fatalf("review action after complete = %+v, want startable", reviewActions["review"])
	}
}

func TestSyntheticModeHarnessDoesNotLeakIntoProductionRegistry(t *testing.T) {
	if ValidateMode(string(modeSyntheticHarness)) {
		t.Fatalf("%q leaked into production registry", modeSyntheticHarness)
	}
	for _, entry := range PromptCatalogEntries() {
		if entry.Mode == string(modeSyntheticHarness) {
			t.Fatalf("%q leaked into production prompt catalog entries", modeSyntheticHarness)
		}
	}
	for _, mode := range Modes() {
		if mode == modeSyntheticHarness {
			t.Fatalf("%q leaked into production mode list", modeSyntheticHarness)
		}
	}
}

func syntheticHarnessDefinition() Definition {
	return buildInitiativeMode(initiativeModeSpec{
		Mode:                   modeSyntheticHarness,
		Label:                  "Synthetic Harness",
		Description:            "Synthetic harness used by registry tests; not a production mode.",
		BestFor:                []string{"Exercising registry validators"},
		NotFor:                 []string{"Anything resembling production work"},
		Tradeoffs:              []string{"Test-only — never registered outside test scope"},
		WhenInDoubtPickInstead: ModeItemLevel,
		RunStrategy:            RunStrategyOperatorGatedLoop,
		ArtifactRoot:           "modes/synthetic-harness",
		PromptCatalogPrefix:    "swarm-manager-synthetic-harness",
		DefaultProfileKey:      ProfileDeepWork,
		StartPhase:             "assess",
		Terminal:               []Phase{"review"},
		Transitions: map[Phase][]Phase{
			"assess": {"decide"},
			"decide": {"assess", "review"},
			"review": {"assess"},
		},
		TransitionRules: map[Phase][]TransitionRule{
			"decide": {
				{
					When: TransitionCondition{
						Kind:             TransitionConditionProgressDecision,
						ProgressDecision: ProgressReplan,
					},
					Next: []Phase{"assess"},
				},
				{
					When: TransitionCondition{
						Kind:             TransitionConditionProgressDecision,
						ProgressDecision: ProgressComplete,
					},
					Next: []Phase{"review"},
				},
			},
		},
		Phases: []initiativePhaseSpec{
			{
				Phase:         "assess",
				Purpose:       "synthetic_harness_assess",
				PromptPurpose: "Assess the synthetic methodology state.",
				ProfileKey:    ProfileDeepWork,
			},
			{
				Phase:            "decide",
				Purpose:          "synthetic_harness_decide",
				PromptPurpose:    "Classify whether the synthetic methodology should replan or review.",
				ProfileKey:       ProfileAnalysis,
				ResultBindings:   []ResultBinding{progressResultArtifact("modes/synthetic-harness/progress.json")},
				Metrics:          PhaseMetricsSpec{CountsReplanSample: true},
				RequiresProgress: true,
			},
			{
				Phase:            "review",
				Purpose:          "synthetic_harness_review",
				PromptPurpose:    "Review the synthetic methodology output.",
				ProfileKey:       ProfileAnalysis,
				RequiresVerdict:  true,
				RequiresCriteria: true,
				Metrics:          PhaseMetricsSpec{CountsAcceptanceSample: true},
			},
		},
	})
}

func withSyntheticModeRegistry(t *testing.T, def Definition) {
	t.Helper()
	previous := registry
	next := cloneRegistryForTest()
	next[def.Mode] = def
	registry = next
	t.Cleanup(func() {
		registry = previous
	})
}

func catalogModeByID(catalog ModeCatalog, mode Mode) (ModeCatalogEntry, bool) {
	for _, entry := range catalog.Modes {
		if entry.Mode == string(mode) {
			return entry, true
		}
	}
	return ModeCatalogEntry{}, false
}

func startSyntheticPhase(t *testing.T, svc *Service, initiativeName string, phase Phase) RoundEnvelope {
	t.Helper()
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: initiativeName,
		Phase:          string(phase),
		RequestedBy:    "synthetic-test",
	})
	if err != nil {
		t.Fatalf("StartPhase(%s) returned error: %v", phase, err)
	}
	return round
}

func completeSyntheticRun(t *testing.T, agent *fakeAgent, runID string, summary string) {
	t.Helper()
	agent.states[runID] = agentmanager.RunState{
		RunID:     runID,
		Status:    "complete",
		Summary:   summary,
		StartedAt: "2026-04-30T12:00:00Z",
	}
}

func refreshSyntheticRound(t *testing.T, svc *Service, initiativeName string, number int) RoundEnvelope {
	t.Helper()
	round, err := svc.RefreshRound(context.Background(), initiativeName, modeSyntheticHarness, number)
	if err != nil {
		t.Fatalf("RefreshRound(%03d) returned error: %v", number, err)
	}
	if round.Status != RoundStatusCompleted {
		t.Fatalf("round %03d status = %q error=%q, want completed", number, round.Status, round.Error)
	}
	return round
}

func workspacePhaseActions(workspace Workspace) map[Phase]WorkspacePhase {
	out := make(map[Phase]WorkspacePhase, len(workspace.Definition.Phases))
	for _, phase := range workspace.Definition.Phases {
		out[Phase(phase.Phase)] = phase
	}
	return out
}
