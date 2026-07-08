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
			wantPhases: []string{"investigate", "plan", "execute", "review", "reconcile"},
		},
		{
			mode:       ModePhasedPlanDrain,
			wantPhases: []string{"prepare_plan", "execute_next", "classify_progress", "review", "reconcile"},
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
	execute := loop.Trace[2]
	if execute.Phase != "execute" {
		t.Fatalf("trace[2].phase = %q, want execute", execute.Phase)
	}
	if execute.Transition == nil || execute.Transition.To != "review" || execute.Transition.Label != "always" {
		t.Fatalf("execute transition = %+v, want always -> review", execute.Transition)
	}

	drain, err := svc.SimulateMode(context.Background(), ModePhasedPlanDrain, "")
	if err != nil {
		t.Fatalf("SimulateMode phased-plan-drain: %v", err)
	}
	classify := drain.Trace[2]
	if classify.Phase != "classify_progress" {
		t.Fatalf("trace[2].phase = %q, want classify_progress", classify.Phase)
	}
	if classify.Output.Progress == nil || classify.Output.Progress.Decision != ProgressComplete {
		t.Fatalf("classify output progress = %+v, want complete", classify.Output.Progress)
	}
	if classify.Transition == nil ||
		classify.Transition.To != "review" ||
		classify.Transition.ConditionKind != string(TransitionConditionProgressDecision) ||
		classify.Transition.ProgressDecision != string(ProgressComplete) {
		t.Fatalf("classify transition = %+v, want complete -> review", classify.Transition)
	}
}

func TestSimulateModeHolisticReplanPresetLoopsBackThenCompletes(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	got, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "replan-after-execute")
	if err != nil {
		t.Fatalf("SimulateMode replan preset: %v", err)
	}
	if got.ActivePreset != "replan-after-execute" {
		t.Fatalf("active preset = %q, want replan-after-execute", got.ActivePreset)
	}
	// First execute must route back to investigate via the real payload-bool guard.
	firstExecute := got.Trace[2]
	if firstExecute.Phase != "execute" {
		t.Fatalf("trace[2].phase = %q, want execute", firstExecute.Phase)
	}
	if !firstExecute.Output.ReplanNeeded {
		t.Fatalf("first execute replan_needed = false, want true")
	}
	if firstExecute.Transition == nil ||
		firstExecute.Transition.To != "investigate" ||
		firstExecute.Transition.ConditionKind != string(TransitionConditionPayloadBool) {
		t.Fatalf("first execute transition = %+v, want payload_bool -> investigate", firstExecute.Transition)
	}
	// The trace must still terminate at reconcile after the second pass.
	last := got.Trace[len(got.Trace)-1]
	if last.Phase != "reconcile" || !last.Terminal {
		t.Fatalf("final step = %q terminal=%v, want reconcile terminal", last.Phase, last.Terminal)
	}
	// The second execute pass must not request replan again.
	secondExecuteFound := false
	for _, step := range got.Trace[3:] {
		if step.Phase == "execute" {
			secondExecuteFound = true
			if step.Output.ReplanNeeded {
				t.Fatalf("second execute still requests replan: %+v", step.Output)
			}
		}
	}
	if !secondExecuteFound {
		t.Fatalf("expected a second execute pass after replan, trace: %+v", phaseList(got.Trace))
	}
}

func TestSimulateModePhasedPlanBranchPresets(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	cases := []struct {
		preset       string
		wantDecision ProgressDecision
		wantTerminal string
		wantTo       string // classify_progress first-visit routing target ("" for terminal)
	}{
		{preset: "continue-next-slice", wantDecision: ProgressContinue, wantTerminal: "reconcile", wantTo: "execute_next"},
		{preset: "replan-plan", wantDecision: ProgressReplan, wantTerminal: "reconcile", wantTo: "prepare_plan"},
		{preset: "blocked", wantDecision: ProgressBlocked, wantTerminal: "classify_progress", wantTo: ""},
	}

	for _, tc := range cases {
		t.Run(tc.preset, func(t *testing.T) {
			got, err := svc.SimulateMode(context.Background(), ModePhasedPlanDrain, tc.preset)
			if err != nil {
				t.Fatalf("SimulateMode %s: %v", tc.preset, err)
			}
			classify := got.Trace[2]
			if classify.Phase != "classify_progress" {
				t.Fatalf("trace[2].phase = %q, want classify_progress", classify.Phase)
			}
			if classify.Output.Progress == nil || classify.Output.Progress.Decision != tc.wantDecision {
				t.Fatalf("classify decision = %+v, want %q", classify.Output.Progress, tc.wantDecision)
			}
			if tc.wantTo == "" {
				// The blocked guard matches but has no downstream target: the
				// step is terminal, yet the transition is still surfaced so the
				// UI can explain why the cycle stopped here.
				if !classify.Terminal {
					t.Fatalf("blocked classify terminal = %v, want terminal", classify.Terminal)
				}
				if classify.Transition == nil || classify.Transition.To != "" ||
					classify.Transition.ProgressDecision != string(ProgressBlocked) {
					t.Fatalf("blocked transition = %+v, want blocked guard with empty target", classify.Transition)
				}
			} else if classify.Transition == nil || classify.Transition.To != tc.wantTo {
				t.Fatalf("classify transition = %+v, want to %q", classify.Transition, tc.wantTo)
			}
			last := got.Trace[len(got.Trace)-1]
			if last.Phase != tc.wantTerminal || !last.Terminal {
				t.Fatalf("final step = %q terminal=%v, want %q terminal", last.Phase, last.Terminal, tc.wantTerminal)
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
	if review.Output.Verdict != "changes_requested" {
		t.Fatalf("review verdict = %q, want changes_requested", review.Output.Verdict)
	}
	// Routing is unchanged: review still advances to reconcile.
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

	if _, err := svc.SimulateMode(context.Background(), ModeHolisticLoop, "replan-after-execute"); err != nil {
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

func phaseList(trace []SimulationStep) []string {
	out := make([]string, 0, len(trace))
	for _, step := range trace {
		out = append(out, step.Phase)
	}
	return out
}
