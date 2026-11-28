package runannotations

import (
	"context"
	"strings"
	"testing"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

const pinnedLabels = `[
  {"id":"run.kind","group":"run","scope":"run","method":"deterministic","rule":"workload_kind",
   "evidence":[{"source":"workload","field":"kind"}],"gate":{"precision":0.7,"recall":0.5,"min_gold":2},
   "search":{"facet":"label:run.kind","min_probability":0.6},"compatibility":{"change":"breaking"}},
  {"id":"agent.stalled","group":"agent","scope":"run","method":"decision",
   "candidates":"false",
   "question":{"type":"noul","instructions":"spinning?","true_description":"yes","false_description":"no"},
   "evidence":[{"source":"tool_summary","field":"calls"}],"gate":{"precision":0.7,"recall":0.5,"min_gold":2},
   "search":{"facet":"label:agent.stalled","min_probability":0.6},
   "friction":{"enabled":true,"min_probability":0.6,"cost_unit":"run","importance":"high"},
   "compatibility":{"change":"breaking"}}
]`

func pinnedService(t *testing.T, decision *fakeDecision) (*Service, *Store) {
	t.Helper()
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 5
	return testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, pinnedLabels, []RunFacts{fact}, decision, Options{})
}

func candidateResolver(request *inferencev1.DecideBatchRequest) (string, string) {
	if pin := request.GetPinResourceRole(); pin != "" {
		return "candidate-model", pin
	}
	return "winnow", "decide.local"
}

func TestPinnedEvaluationAsksOneCandidateAndStoresApartFromProduction(t *testing.T) {
	decision := &fakeDecision{resolve: candidateResolver}
	service, store := pinnedService(t, decision)
	ctx := context.Background()
	// Production first. agent.stalled is not a candidate for this run, so it has
	// no production answer at all.
	if _, err := service.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	productionRows, _ := store.Count(ctx)
	callsBefore := len(decision.calls)

	result, err := service.Evaluate(ctx, EvaluateOptions{
		RunIDs: []string{"run-1"}, PinResourceRole: "decide.local.candidate", Role: "decide.candidates", TimeBudget: -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Pinned != "decide.local.candidate" || len(decision.calls) != callsBefore+1 {
		t.Fatalf("pinned = %q calls = %d", result.Pinned, len(decision.calls)-callsBefore)
	}
	request := decision.calls[len(decision.calls)-1]
	if request.GetPinResourceRole() != "decide.local.candidate" || request.GetRole() != "decide.candidates" {
		t.Fatalf("request role %q pin %q, want the walk role and the pinned candidate", request.GetRole(), request.GetPinResourceRole())
	}
	if after, _ := store.Count(ctx); after != productionRows {
		t.Fatalf("production rows %d -> %d: a pinned pass wrote to production", productionRows, after)
	}
	trial, err := store.Trials().CurrentFor(ctx, ScopeRun, "run-1", "agent.stalled", service.Pack().LabelHashes["agent.stalled"], "decide.local.candidate")
	if err != nil || trial == nil || trial.Model != "candidate-model" || trial.RequestedRole != "decide.candidates" {
		t.Fatalf("trial row = %+v (%v), want it keyed by the candidate and answering the gold run even though it is no production candidate", trial, err)
	}
	for _, skipped := range result.Runs[0].Skipped {
		if skipped.LabelID == "run.kind" && skipped.Reason != "pinned_not_decision" {
			t.Fatalf("rule label skipped for %q, want it left to production", skipped.Reason)
		}
	}

	// The same pinned evaluation again reuses the stored trial: no inference.
	again, err := service.Evaluate(ctx, EvaluateOptions{
		RunIDs: []string{"run-1"}, PinResourceRole: "decide.local.candidate", Role: "decide.candidates", TimeBudget: -1,
	})
	if err != nil || len(decision.calls) != callsBefore+1 {
		t.Fatalf("a repeated pinned evaluation spent inference: %d calls (%v)", len(decision.calls)-callsBefore, err)
	}
	if again.Runs[0].Labels[0].Model != "candidate-model" {
		t.Fatalf("cached view = %+v", again.Runs[0].Labels[0])
	}

	view, err := service.RunTrial(ctx, "run-1")
	if err != nil || len(view.Labels) != 1 || view.Labels[0].ResourceRole != "decide.local.candidate" {
		t.Fatalf("trial view = %+v (%v)", view.Labels, err)
	}
	production, err := service.Run(ctx, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range production.Labels {
		if label.ID == "agent.stalled" {
			t.Fatalf("production view carries the trial answer: %+v", label)
		}
	}
}

func TestPinnedAnswerFromAnotherCandidateIsNotStoredAsTheCandidates(t *testing.T) {
	decision := &fakeDecision{resolve: func(*inferencev1.DecideBatchRequest) (string, string) { return "intruder", "decide.local" }}
	service, store := pinnedService(t, decision)
	ctx := context.Background()
	result, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"run-1"}, PinResourceRole: "decide.local.candidate", TimeBudget: -1})
	if err != nil {
		t.Fatal(err)
	}
	label := result.Runs[0].Labels[0]
	if label.State != StateUnavailable || !strings.Contains(label.Reason, "pin_mismatch") {
		t.Fatalf("label = %+v, want the mismatched answer refused", label)
	}
	if got, _ := store.Trials().CurrentFor(ctx, ScopeRun, "run-1", "agent.stalled", service.Pack().LabelHashes["agent.stalled"], "decide.local"); got != nil {
		t.Fatalf("an answer from another candidate was cached as a trial: %+v", got)
	}
}

func TestPinnedPassNeverEscalatesOrLearnsRoleResolutions(t *testing.T) {
	decision := &fakeDecision{answer: lowConfidence, resolve: candidateResolver}
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 5
	labels := `[` + scoreLabel("a", "") + `]`
	service, _ := testServiceRuntime(t, roleRuntime, labels, []RunFacts{fact}, decision, Options{})
	result, err := service.Evaluate(context.Background(), EvaluateOptions{RunIDs: []string{"run-1"}, PinResourceRole: "decide.local.candidate", TimeBudget: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 1 || result.Escalation != nil {
		t.Fatalf("calls = %v escalation = %+v, want a bake-off to measure one model, not escalate", callRoles(decision), result.Escalation)
	}
	if service.roles.sameModel("decide.fast", "decide.accurate", service.now(), roleTTL) {
		t.Fatal("a pinned answer taught the engine what a role resolves to")
	}
}
