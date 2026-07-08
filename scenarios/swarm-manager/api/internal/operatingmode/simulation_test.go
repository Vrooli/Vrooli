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
			got, err := svc.SimulateMode(context.Background(), tt.mode)
			if err != nil {
				t.Fatalf("SimulateMode returned error: %v", err)
			}
			if got.Mode != string(tt.mode) {
				t.Fatalf("mode = %q, want %q", got.Mode, tt.mode)
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
		})
	}
}

func TestSimulateModeUsesRealTransitionGuards(t *testing.T) {
	root := t.TempDir()
	svc := newTestServiceWithOptions(t, root, serviceOptions{})

	loop, err := svc.SimulateMode(context.Background(), ModeHolisticLoop)
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

	drain, err := svc.SimulateMode(context.Background(), ModePhasedPlanDrain)
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

func TestSimulateModeIsIsolatedFromPersistentRoundState(t *testing.T) {
	root := t.TempDir()
	agent := &fakeAgent{}
	prompts := &fakePrompts{}
	svc := newTestService(t, root, agent, prompts)

	if _, err := svc.SimulateMode(context.Background(), ModeHolisticLoop); err != nil {
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

	_, err := svc.SimulateMode(context.Background(), ModeItemLevel)
	if err == nil {
		t.Fatal("SimulateMode item-level error = nil, want rejection")
	}
}
