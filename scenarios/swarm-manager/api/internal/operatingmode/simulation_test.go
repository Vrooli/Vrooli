package operatingmode

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSimulateModeReturnsDeterministicTraceForPhaseModes(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	tests := []struct {
		mode       Mode
		wantPhases []string
	}{
		{
			mode:       ModeHolisticLoop,
			wantPhases: []string{"investigate", "plan", "execute", "execute", "review", "reconcile"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			got, err := svc.SimulateMode(context.Background(), tt.mode, "")
			if err != nil {
				t.Fatalf("SimulateMode returned error: %v", err)
			}
			if got.Mode != string(tt.mode) {
				t.Fatalf("mode = %q, want %q", got.Mode, tt.mode)
			}
			if got.ActivePreset != "happy-path" {
				t.Fatalf("active preset = %q, want happy-path", got.ActivePreset)
			}
			if len(got.Presets) == 0 || got.Presets[0].ID != "happy-path" {
				t.Fatalf("presets = %+v, want happy-path first", got.Presets)
			}
			if got.Initiative.Name != simulationInitiativeName || got.Initiative.Mode != string(tt.mode) {
				t.Fatalf("initiative = %+v, want simulation initiative for %s", got.Initiative, tt.mode)
			}
			if len(got.Trace) != len(tt.wantPhases) {
				t.Fatalf("trace len = %d, want %d: %+v", len(got.Trace), len(tt.wantPhases), got.Trace)
			}
			for i, phase := range tt.wantPhases {
				step := got.Trace[i]
				if step.Index != i || step.Phase != phase {
					t.Fatalf("step %d = index %d phase %q, want %d/%q", i, step.Index, step.Phase, i, phase)
				}
				if step.Round.Status != RoundStatusCompleted {
					t.Fatalf("step %d status = %q, want completed", i, step.Round.Status)
				}
				if len(step.Inputs.Items) == 0 || len(step.Inputs.AcceptanceCriteria) == 0 {
					t.Fatalf("step %d inputs missing mock items/criteria: %+v", i, step.Inputs)
				}
				// Seeded items carry realistic titles/statuses, not bare refs.
				if step.Inputs.Items[0].Title == "" || step.Inputs.Items[0].Status == "" {
					t.Fatalf("step %d item missing title/status: %+v", i, step.Inputs.Items[0])
				}
				if i < len(tt.wantPhases)-1 {
					if step.Transition == nil || step.Transition.To != tt.wantPhases[i+1] {
						t.Fatalf("step %d transition = %+v, want to %q", i, step.Transition, tt.wantPhases[i+1])
					}
					if step.Terminal {
						t.Fatalf("step %d marked terminal before final phase", i)
					}
				} else if !step.Terminal || step.Transition != nil {
					t.Fatalf("final step terminal/transition = %v/%+v, want terminal with no transition", step.Terminal, step.Transition)
				}
			}
			// The terminal reconcile step proposes a realistic backlog sync.
			last := got.Trace[len(got.Trace)-1]
			if last.Output.BacklogSync == nil || len(last.Output.BacklogSync.CompletedItems) == 0 {
				t.Fatalf("terminal reconcile backlog_sync = %+v, want completed items", last.Output.BacklogSync)
			}
		})
	}
}

func TestSimulateModeUsesRealTransitionGuards(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	loop, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "")
	if err != nil {
		t.Fatalf("SimulateMode holistic-loop: %v", err)
	}
	// The delegated execute phase walks the sub-mode's classified edge: the
	// first slice continues (inline self-edge on the parent phase), the second
	// completes and the PARENT guard routes to review.
	firstExecute := loop.Trace[2]
	if firstExecute.Phase != "execute" {
		t.Fatalf("trace[2].phase = %q, want execute", firstExecute.Phase)
	}
	if firstExecute.Transition == nil || firstExecute.Transition.To != "execute" ||
		firstExecute.Transition.Field != "progress" || firstExecute.Transition.Value != "continue" {
		t.Fatalf("first execute transition = %+v, want progress=continue -> execute (inline delegation loop)", firstExecute.Transition)
	}
	secondExecute := loop.Trace[3]
	if secondExecute.Phase != "execute" {
		t.Fatalf("trace[3].phase = %q, want execute", secondExecute.Phase)
	}
	if secondExecute.Transition == nil || secondExecute.Transition.To != "review" ||
		secondExecute.Transition.Field != "progress" || secondExecute.Transition.Value != "complete" {
		t.Fatalf("second execute transition = %+v, want progress=complete -> review (parent guard)", secondExecute.Transition)
	}
	if got := secondExecute.SkillID; got != "swarm-manager-phased-plan-execute-next" {
		t.Fatalf("delegated execute skill = %q, want the sub-mode's execute-next skill", got)
	}

	drain, err := svc.SimulateMode(context.Background(), ModePhasedPlanDrain, "")
	if err != nil {
		t.Fatalf("SimulateMode phased-plan-drain: %v", err)
	}
	if len(drain.Trace) != 2 {
		t.Fatalf("drain trace = %v, want [execute execute]", phaseList(drain.Trace))
	}
	first := drain.Trace[0]
	if first.Phase != "execute" || first.Terminal {
		t.Fatalf("trace[0] = %q terminal=%v, want non-terminal execute", first.Phase, first.Terminal)
	}
	// The classified edge derived progress=continue from the handoff and the
	// expanded eq-guard looped back to execute.
	if first.Transition == nil ||
		first.Transition.To != "execute" ||
		first.Transition.ConditionKind != GuardOpEq ||
		first.Transition.Field != "progress" ||
		first.Transition.Value != "continue" {
		t.Fatalf("first execute transition = %+v, want progress=continue -> execute", first.Transition)
	}
	// The second slice classifies complete: a matched guard with no target — a
	// guarded stop, surfaced so the UI can explain why the loop ended.
	last := drain.Trace[1]
	if !last.Terminal {
		t.Fatalf("trace[1] terminal = %v, want guarded stop", last.Terminal)
	}
	if last.Transition == nil || last.Transition.To != "" ||
		last.Transition.Field != "progress" || last.Transition.Value != "complete" {
		t.Fatalf("last execute transition = %+v, want progress=complete guarded stop", last.Transition)
	}
}

func TestSimulateModeHolisticBlockedDrainPresetLoopsBackThenCompletes(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	got, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "drain-blocked-replan")
	if err != nil {
		t.Fatalf("SimulateMode blocked-drain preset: %v", err)
	}
	if got.ActivePreset != "drain-blocked-replan" {
		t.Fatalf("active preset = %q, want drain-blocked-replan", got.ActivePreset)
	}
	// The first delegated execute round classifies blocked: the sub-mode stops
	// and the PARENT's blocked guard routes back to investigate (the composed
	// replan loop).
	firstExecute := got.Trace[2]
	if firstExecute.Phase != "execute" {
		t.Fatalf("trace[2].phase = %q, want execute", firstExecute.Phase)
	}
	if progress, _ := firstExecute.Round.Payload["progress"].(string); progress != "blocked" {
		t.Fatalf("first execute derived progress = %q, want blocked", progress)
	}
	if firstExecute.Transition == nil ||
		firstExecute.Transition.To != "investigate" ||
		firstExecute.Transition.ConditionKind != GuardOpEq ||
		firstExecute.Transition.Field != "progress" ||
		firstExecute.Transition.Value != "blocked" {
		t.Fatalf("first execute transition = %+v, want progress=blocked guard -> investigate", firstExecute.Transition)
	}
	// The trace must still terminate at reconcile after the second pass.
	last := got.Trace[len(got.Trace)-1]
	if last.Phase != "reconcile" || !last.Terminal {
		t.Fatalf("final step = %q terminal=%v, want reconcile terminal", last.Phase, last.Terminal)
	}
	// The second delegated pass completes cleanly.
	secondExecuteFound := false
	for _, step := range got.Trace[3:] {
		if step.Phase == "execute" {
			secondExecuteFound = true
			if progress, _ := step.Round.Payload["progress"].(string); progress != "complete" {
				t.Fatalf("second execute derived progress = %q, want complete", progress)
			}
		}
	}
	if !secondExecuteFound {
		t.Fatalf("expected a second execute pass after the blocked drain, trace: %+v", phaseList(got.Trace))
	}
}

func TestSimulateModePhasedPlanBranchPresets(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	cases := []struct {
		preset    string
		wantPath  []string
		wantValue string // classified value on the final (guarded-stop) step
	}{
		{preset: "happy-path", wantPath: []string{"execute", "execute"}, wantValue: "continue"},
		{preset: "complete-first-slice", wantPath: []string{"execute"}, wantValue: "complete"},
		{preset: "blocked", wantPath: []string{"execute"}, wantValue: "blocked"},
	}

	for _, tc := range cases {
		t.Run(tc.preset, func(t *testing.T) {
			got, err := svc.SimulateMode(context.Background(), ModePhasedPlanDrain, tc.preset)
			if err != nil {
				t.Fatalf("SimulateMode %s: %v", tc.preset, err)
			}
			if len(got.Trace) != len(tc.wantPath) {
				t.Fatalf("trace = %v, want %v", phaseList(got.Trace), tc.wantPath)
			}
			for i, phase := range tc.wantPath {
				if got.Trace[i].Phase != phase {
					t.Fatalf("trace = %v, want %v", phaseList(got.Trace), tc.wantPath)
				}
			}
			first := got.Trace[0]
			// Every branch routes through the single classified edge: the value
			// is derived from the handoff and surfaced on the transition.
			if first.Transition == nil || first.Transition.ConditionKind != GuardOpEq ||
				first.Transition.Field != "progress" || first.Transition.Value != tc.wantValue {
				t.Fatalf("first transition = %+v, want progress=%s", first.Transition, tc.wantValue)
			}
			// The walk always ends on a guarded stop (complete or blocked): the
			// matched guard has no target, yet is surfaced so the UI can explain
			// why the loop ended here.
			last := got.Trace[len(got.Trace)-1]
			if !last.Terminal {
				t.Fatalf("final step terminal = %v, want guarded stop", last.Terminal)
			}
			if last.Transition == nil || last.Transition.To != "" || last.Transition.Field != "progress" {
				t.Fatalf("final transition = %+v, want guarded stop on progress", last.Transition)
			}
		})
	}
}

func TestSimulateModeReviewNotAcceptedPresetRecordsVerdict(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	got, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "review-not-accepted")
	if err != nil {
		t.Fatalf("SimulateMode review-not-accepted: %v", err)
	}
	var review *SimulationStep
	for i := range got.Trace {
		if got.Trace[i].Phase == "review" {
			review = &got.Trace[i]
		}
	}
	if review == nil {
		t.Fatalf("no review step in trace: %+v", phaseList(got.Trace))
	}
	// A plain non-accepting verdict (not changes_requested) does not reloop.
	if review.Output.Verdict != "rejected" {
		t.Fatalf("review verdict = %q, want rejected", review.Output.Verdict)
	}
	// review appears exactly once — no reloop back to execute.
	if reviewCount := countPhase(got.Trace, "review"); reviewCount != 1 {
		t.Fatalf("review phase visited %d times, want 1 (no reloop): %v", reviewCount, phaseList(got.Trace))
	}
	// Routing terminates at reconcile: the gap is recorded, not re-executed.
	last := got.Trace[len(got.Trace)-1]
	if last.Phase != "reconcile" || !last.Terminal {
		t.Fatalf("final step = %q terminal=%v, want reconcile terminal", last.Phase, last.Terminal)
	}
}

// TestSimulateModeReviewChangesRequestedPresetLoopsBackToExecute proves the
// review-reloop branch: a changes_requested verdict routes back to execute for
// a second pass rather than terminating, and the preset walks the real guards
// to prove it.
func TestSimulateModeReviewChangesRequestedPresetLoopsBackToExecute(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	got, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "review-changes-requested")
	if err != nil {
		t.Fatalf("SimulateMode review-changes-requested: %v", err)
	}
	// execute and review are each visited twice: the first review returns
	// changes_requested and loops back to execute before the second accepts.
	if n := countPhase(got.Trace, "execute"); n != 2 {
		t.Fatalf("execute visited %d times, want 2 (reloop): %v", n, phaseList(got.Trace))
	}
	if n := countPhase(got.Trace, "review"); n != 2 {
		t.Fatalf("review visited %d times, want 2 (reloop): %v", n, phaseList(got.Trace))
	}
	// The first review is the changes_requested one; the walk still terminates
	// at reconcile once accepted.
	last := got.Trace[len(got.Trace)-1]
	if last.Phase != "reconcile" || !last.Terminal {
		t.Fatalf("final step = %q terminal=%v, want reconcile terminal", last.Phase, last.Terminal)
	}
}

func TestSimulateModeUnknownPresetFallsBackToHappyPath(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	got, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "does-not-exist")
	if err != nil {
		t.Fatalf("SimulateMode unknown preset: %v", err)
	}
	if got.ActivePreset != "happy-path" {
		t.Fatalf("active preset = %q, want happy-path fallback", got.ActivePreset)
	}
}

func TestSimulateModeIsIsolatedFromPersistentRoundState(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	prompts := &fakePrompts{}
	svc := newTestService(t, root, agent, prompts)

	if _, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "drain-blocked-replan"); err != nil {
		t.Fatalf("SimulateMode returned error: %v", err)
	}
	if len(agent.spawned) != 0 {
		t.Fatalf("agent spawned %d runs during simulation, want 0", len(agent.spawned))
	}
	if len(prompts.calls) != 0 {
		t.Fatalf("prompt calls = %#v, want none", prompts.calls)
	}
	if _, err := os.Stat(filepath.Join(root, "initiatives", simulationInitiativeName)); !os.IsNotExist(err) {
		t.Fatalf("simulation created persistent initiative state, stat err=%v", err)
	}
	holder, err := svc.lock.Inspect(simulationInitiativeName)
	if err != nil {
		t.Fatalf("inspect lock: %v", err)
	}
	if holder != nil {
		t.Fatalf("simulation acquired lock holder %+v", holder)
	}
}

func TestSimulateModeRejectsItemLevelMode(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	_, err := svc.SimulateMode(context.Background(), ModeItemLevel, "")
	if err == nil {
		t.Fatal("SimulateMode item-level error = nil, want rejection")
	}
}

// TestSimulationPresetsAreModeOwnedData proves the Phase 6 acceptance: presets
// are sourced from modes/<id>/example-runs/*.json (not hardcoded Go), carry the
// data-authored operator narrative, and are ordered happy-path first. This is
// what makes "add a simulation case" a data edit.
func TestSimulationPresetsAreModeOwnedData(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	def, err := DefinitionFor(ModeHolisticLoop)
	if err != nil {
		t.Fatalf("DefinitionFor: %v", err)
	}
	if len(def.ExampleRuns) == 0 {
		t.Fatal("holistic-loop Definition carries no example-runs; presets are not data-sourced")
	}
	if def.ExampleRuns[0].ID != "happy-path" {
		t.Fatalf("first example-run id = %q, want happy-path (happy-path-first ordering)", def.ExampleRuns[0].ID)
	}

	got, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "drain-blocked-replan")
	if err != nil {
		t.Fatalf("SimulateMode: %v", err)
	}
	var replan *SimulationPreset
	for i := range got.Presets {
		if got.Presets[i].ID == "drain-blocked-replan" {
			replan = &got.Presets[i]
		}
	}
	if replan == nil {
		t.Fatalf("drain-blocked-replan preset missing from %+v", got.Presets)
	}
	// The operator narrative comes straight from the example-run JSON.
	if replan.Branch == "" || replan.Scenario == "" || replan.Label == "" {
		t.Fatalf("preset metadata not data-sourced: %+v", replan)
	}
	if got.Initiative.Title != "Unify the audio-session lifecycle" {
		t.Fatalf("initiative title = %q, want the example-run's seeded title", got.Initiative.Title)
	}
}

// TestSimulateModeDetectsExamplePathDrift proves the runtime path assertion
// guards a fixture whose declared expected_path no longer matches the guards it
// walks: SimulateMode fails loudly rather than silently returning a wrong trace.
func TestSimulateModeDetectsExamplePathDrift(t *testing.T) {
	// A preset whose declared expected_path no longer matches the phases it walks
	// must be rejected, so a drifted fixture fails loudly instead of returning a
	// wrong trace.
	preset := simPreset{
		meta:         SimulationPreset{ID: "drifted"},
		expectedPath: []string{"investigate", "plan", "execute", "review", "WRONG"},
	}
	trace := []SimulationStep{
		{Phase: "investigate"},
		{Phase: "plan"},
		{Phase: "execute"},
		{Phase: "review"},
		{Phase: "reconcile"},
	}
	if err := assertSimulatedPath(preset, trace, 0); err == nil {
		t.Fatal("assertSimulatedPath accepted a drifted expected_path, want error")
	}
	// The matching path is accepted.
	preset.expectedPath[4] = "reconcile"
	if err := assertSimulatedPath(preset, trace, 0); err != nil {
		t.Fatalf("assertSimulatedPath rejected a matching path: %v", err)
	}
}

func phaseList(trace []SimulationStep) []string {
	out := make([]string, 0, len(trace))
	for _, step := range trace {
		out = append(out, step.Phase)
	}
	return out
}

func countPhase(trace []SimulationStep, phase string) int {
	n := 0
	for _, step := range trace {
		if step.Phase == phase {
			n++
		}
	}
	return n
}
