package nextaction

import "testing"

func TestBlockerVocabularyIsExplicitAndFailClosed(t *testing.T) {
	for _, code := range []string{string(PlanChanged), string(PlanNotAccepted), string(PlanInvalid), string(UnmetDependencies), string(QueueCap), string(CostCap), string(CircuitOpen)} {
		if err := ValidateBlockerCode(code); err != nil {
			t.Fatalf("ValidateBlockerCode(%q) error = %v", code, err)
		}
	}
	if err := ValidateBlockerCode("unmapped"); err == nil {
		t.Fatal("ValidateBlockerCode accepted an unmapped code")
	}
}

func TestActionForBlockerUsesCodesNotMessages(t *testing.T) {
	if got := ActionForBlocker(string(PlanInvalid)); got != RepairPlan {
		t.Fatalf("ActionForBlocker(plan_invalid) = %q, want %q", got, RepairPlan)
	}
	if got := ActionForBlocker(string(QueueCap)); got != Run {
		t.Fatalf("ActionForBlocker(queue_cap) = %q, want %q", got, Run)
	}
}

// Every action id must be classified deliberately. The default arm returns
// EffectStateChange, which is a safe fallback but hides an unclassified
// action — this walks the declared vocabulary so a new id shows up here
// rather than silently inheriting the default.
func TestEffectForCoversEveryDeclaredAction(t *testing.T) {
	classified := map[ID]Effect{
		None:                EffectNone,
		Decide:              EffectNone,
		ResolveDependencies: EffectNone,
		ViewExecution:       EffectNone,
		Chain:               EffectNone,
		Run:                 EffectAgentRun,
		Retry:               EffectAgentRun,
		Review:              EffectAgentRun,
		AuthorPlan:          EffectAgentRun,
		RepairPlan:          EffectAgentRun,
		PlanGoal:            EffectAgentRun,
		AuthorFollowup:      EffectAgentRun,
		AcceptSuggestion:    EffectStateChange,
		AcceptPlan:          EffectStateChange,
		Archive:             EffectStateChange,
		DispatchFollowup:    EffectStateChange,
		DefineCriteria:      EffectStateChange,
		CloseOut:            EffectStateChange,
	}
	for id, want := range classified {
		if got := EffectFor(id); got != want {
			t.Errorf("EffectFor(%q) = %q, want %q", id, got, want)
		}
	}
}

// An action whose TransitionKey resolves to a declared workflow must not be
// classified as a cheap state change: that combination would render a button
// promising an immediate save while dispatching an agent.
func TestWorkflowBackedActionsDeclareAgentEffect(t *testing.T) {
	workflowKeys := map[string]bool{
		"plan.author": true, "plan.repair": true, "goal.plan": true,
		"follow_up.dispatch": false, "goal.close_out": false,
	}
	for _, id := range []ID{AuthorPlan, RepairPlan, PlanGoal, DispatchFollowup, CloseOut} {
		key := TransitionKey(id)
		if key == "" {
			t.Fatalf("%q unexpectedly has no transition key", id)
		}
		isWorkflow, known := workflowKeys[key]
		if !known {
			t.Fatalf("transition %q is not covered by this test", key)
		}
		effect := EffectFor(id)
		if isWorkflow && effect != EffectAgentRun {
			t.Errorf("%q starts workflow %q but declares effect %q", id, key, effect)
		}
		if !isWorkflow && effect == EffectAgentRun {
			t.Errorf("%q starts deterministic %q but declares an agent run", id, key)
		}
	}
}

func TestOnlyArchiveIsDestructive(t *testing.T) {
	if !IsDestructive(Archive) {
		t.Error("archive must be destructive")
	}
	for _, id := range []ID{Run, Review, AcceptPlan, CloseOut, None} {
		if IsDestructive(id) {
			t.Errorf("%q must not be destructive", id)
		}
	}
}
