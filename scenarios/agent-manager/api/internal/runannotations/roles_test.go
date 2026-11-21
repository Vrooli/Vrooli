package runannotations

import (
	"context"
	"strings"
	"testing"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

const roleTTL = 10 * time.Minute

const roleRuntime = `{"mode":"passive","default_role":"decide.fast","escalate_role":"decide.accurate",
  "escalation":{"enabled":true,"min_confidence":0.55,"same_model_ttl":"10m"},
  "batch":{"max_states":8,"max_questions":4}}`

func scoreLabel(id, extra string) string {
	return `{"id":"` + id + `","group":"agent","scope":"run","method":"decision",` + extra + `
	  "question":{"type":"score","instructions":"how much?","levels":["0","1","2","3","4"]},
	  "evidence":[{"source":"tool_summary","field":"calls"}],
	  "gate":{"precision":0.7,"recall":0.5,"min_gold":10},
	  "search":{"facet":"label:` + id + `","min_probability":0.6},
	  "compatibility":{"change":"breaking"}}`
}

// lowConfidence answers every question with a flat, unconfident distribution so
// the engine considers escalating it.
func lowConfidence(_ *inferencev1.DecideBatchRequest, _ int, question *inferencev1.DecisionQuestion) *inferencev1.DecisionAnswer {
	confidence, score := 0.1, 2.0
	return &inferencev1.DecisionAnswer{
		QuestionId: question.GetId(), Type: question.GetType(), Score: &score, Confidence: &confidence,
		Probabilities: []*inferencev1.DecisionProbability{
			{Option: "0", Probability: 0.2}, {Option: "1", Probability: 0.2}, {Option: "2", Probability: 0.2},
			{Option: "3", Probability: 0.2}, {Option: "4", Probability: 0.2},
		},
	}
}

func callRoles(decision *fakeDecision) []string {
	roles := make([]string, len(decision.calls))
	for index, call := range decision.calls {
		roles[index] = call.GetRole()
	}
	return roles
}

func TestLabelsAreBatchedByTheRoleTheyAsk(t *testing.T) {
	labels := `[` + scoreLabel("a.fast", "") + `,` + scoreLabel("b.accurate", `"role":"decide.accurate",`) + `,` + scoreLabel("c.fast", "") + `]`
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 4
	decision := &fakeDecision{}
	service, _ := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, labels, []RunFacts{fact}, decision, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 2 {
		t.Fatalf("calls = %v, want one per role", callRoles(decision))
	}
	byRole := map[string][]string{}
	for _, call := range decision.calls {
		for _, question := range call.GetQuestions() {
			byRole[call.GetRole()] = append(byRole[call.GetRole()], question.GetId())
		}
	}
	if strings.Join(byRole["decide.fast"], ",") != "a.fast,c.fast" || strings.Join(byRole["decide.accurate"], ",") != "b.accurate" {
		t.Fatalf("questions by role = %v", byRole)
	}
	if got := service.Affinity(); got != "decide.accurate+decide.fast" {
		t.Fatalf("affinity = %q, want it derived from the roles labels resolve to", got)
	}
}

func TestAffinityFollowsTheRolesNotAPackWideSetting(t *testing.T) {
	service, _ := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, `[`+scoreLabel("a", "")+`]`, nil, nil, Options{})
	if got := service.Affinity(); got != "decide.fast" {
		t.Fatalf("affinity = %q", got)
	}
	rerouted, _ := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, `[`+scoreLabel("a", `"role":"decide.jev",`)+`]`, nil, nil, Options{})
	if got := rerouted.Affinity(); got != "decide.jev" {
		t.Fatalf("affinity after re-routing = %q", got)
	}
}

func TestEscalationNeverReAsksTheSameModelAndRemembersWithATTL(t *testing.T) {
	fact1, fact2, fact3 := runFact("run-1"), runFact("run-2"), runFact("run-3")
	for _, fact := range []*RunFacts{&fact1, &fact2, &fact3} {
		fact.ToolSummary.ToolCalls = 4
	}
	// Both roles resolve to the same model: escalating cannot say anything new.
	decision := &fakeDecision{answer: lowConfidence, resolve: func(*inferencev1.DecideBatchRequest) (string, string) { return "laya:en", "decide.local" }}
	clock := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	service, _ := testServiceRuntime(t, roleRuntime, `[`+scoreLabel("a", "")+`]`, []RunFacts{fact1, fact2, fact3}, decision,
		Options{Clock: func() time.Time { return clock }})
	ctx := context.Background()

	first, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"run-1"}, TimeBudget: -1})
	if err != nil {
		t.Fatal(err)
	}
	if got := callRoles(decision); len(got) != 2 || got[0] != "decide.fast" || got[1] != "decide.accurate" {
		t.Fatalf("first pass calls = %v, want one escalation that discovers the model is the same", got)
	}
	if first.Escalation == nil || first.Escalation.Calls != 1 || first.Escalation.SameModelAnswers != 1 || first.Escalation.Adopted != 0 {
		t.Fatalf("first pass escalation = %+v", first.Escalation)
	}

	second, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"run-2", "run-3"}, TimeBudget: -1})
	if err != nil {
		t.Fatal(err)
	}
	if got := callRoles(decision); len(got) != 3 {
		t.Fatalf("calls = %v, want the second pass to ask only the first role", got)
	}
	if second.Escalation == nil || second.Escalation.Calls != 0 || second.Escalation.SkippedStates != 2 || second.Escalation.SkippedCalls != 1 {
		t.Fatalf("second pass escalation = %+v, want the same-model escalation skipped and counted", second.Escalation)
	}

	// After the TTL the engine asks again, so a model change is noticed.
	clock = clock.Add(11 * time.Minute)
	fact4 := runFact("run-4")
	fact4.ToolSummary.ToolCalls = 4
	service.features.(*FixtureSource).Set(fact4)
	if _, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"run-4"}, TimeBudget: -1}); err != nil {
		t.Fatal(err)
	}
	if got := callRoles(decision); len(got) != 5 {
		t.Fatalf("calls = %v, want the escalation retried once the ttl expired", got)
	}
}

func TestEscalationToADifferentModelIsAdoptedWhenItIsMoreConfident(t *testing.T) {
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 4
	decision := &fakeDecision{
		resolve: func(request *inferencev1.DecideBatchRequest) (string, string) {
			if request.GetRole() == "decide.accurate" {
				return "winnow:e4b", "decide.local.accurate"
			}
			return "laya:en", "decide.local"
		},
		answer: func(request *inferencev1.DecideBatchRequest, item int, question *inferencev1.DecisionQuestion) *inferencev1.DecisionAnswer {
			if request.GetRole() == "decide.accurate" {
				return defaultAnswer(question) // confidence 0.8
			}
			return lowConfidence(request, item, question)
		},
	}
	service, store := testServiceRuntime(t, roleRuntime, `[`+scoreLabel("a", "")+`]`, []RunFacts{fact}, decision, Options{})
	result, err := service.Evaluate(context.Background(), EvaluateOptions{RunIDs: []string{"run-1"}, TimeBudget: -1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Escalation == nil || result.Escalation.Adopted != 1 || result.Escalation.SameModelAnswers != 0 {
		t.Fatalf("escalation = %+v", result.Escalation)
	}
	row, err := latestRow(store, "run-1", "a")
	if err != nil || row == nil || row.Model != "winnow:e4b" || row.ModelVersion != "decide.local.accurate" || row.RequestedRole != "decide.accurate" {
		t.Fatalf("stored row = %+v (%v), want the escalated model's answer keyed by its own model", row, err)
	}
}

func TestEscalationToTheSameRoleIsNeverCalled(t *testing.T) {
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 4
	decision := &fakeDecision{answer: lowConfidence}
	runtime := `{"mode":"passive","default_role":"decide.fast","escalate_role":"decide.fast","escalation":{"enabled":true,"min_confidence":0.55}}`
	service, _ := testServiceRuntime(t, runtime, `[`+scoreLabel("a", "")+`]`, []RunFacts{fact}, decision, Options{})
	result, err := service.Evaluate(context.Background(), EvaluateOptions{RunIDs: []string{"run-1"}, TimeBudget: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 1 || result.Escalation == nil || result.Escalation.Calls != 0 || result.Escalation.SkippedStates != 1 {
		t.Fatalf("calls = %v escalation = %+v, want the escalation skipped without a call", callRoles(decision), result.Escalation)
	}
}

func TestAnAnswerAskedOfAnotherRoleIsAskedAgainAfterARoleChange(t *testing.T) {
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 4
	decision := &fakeDecision{}
	service, store := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, `[`+scoreLabel("a", "")+`]`, []RunFacts{fact}, decision, Options{})
	ctx := context.Background()
	if _, err := service.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil || len(decision.calls) != 1 {
		t.Fatalf("a settled answer was re-asked: %d calls (%v)", len(decision.calls), err)
	}
	// The label is re-routed to another role: the old model's answer is not the
	// answer the pack now asks for.
	rerouted, _ := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, `[`+scoreLabel("a", `"role":"decide.accurate",`)+`]`, []RunFacts{fact}, decision, Options{})
	rerouted.store = store
	if _, err := rerouted.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := callRoles(decision); len(got) != 2 || got[1] != "decide.accurate" {
		t.Fatalf("calls = %v, want the re-routed label asked of its new role", got)
	}
}

func TestAnAnswerWrittenBeforeRolesWereRecordedCountsAsTheDefaultRole(t *testing.T) {
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 4
	decision := &fakeDecision{}
	service, store := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, `[`+scoreLabel("a", "")+`]`, []RunFacts{fact}, decision, Options{})
	ctx := context.Background()
	if _, err := service.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE run_annotations SET requested_role = ''`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil || len(decision.calls) != 1 {
		t.Fatalf("a legacy answer for the default role was re-asked: %d calls (%v)", len(decision.calls), err)
	}
}

func TestModelLimitsSizeTheDigestAndTheQuestionBatch(t *testing.T) {
	labels := `[` + scoreLabel("a", "") + `,` + scoreLabel("b", "") + `,` + scoreLabel("c", "") + `]`
	warm, sized := longRunFact("run-warm"), longRunFact("run-1")
	warm.ToolSummary.ToolCalls, sized.ToolSummary.ToolCalls = 4, 4
	decision := &fakeDecision{}
	limits := limitsFunc(func(provider, resourceRole string) (ModelLimits, bool) {
		if resourceRole != "decide.local" {
			t.Errorf("limits asked for %q, want the resource role that answered", resourceRole)
		}
		return ModelLimits{ContextTokens: 400, MaxQuestions: 1}, true
	})
	service, _ := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast","digest_chars":1800,"batch":{"max_states":8,"max_questions":4}}`,
		labels, []RunFacts{warm, sized}, decision, Options{Limits: limits, MaxDigestChars: 1800})
	ctx := context.Background()
	// The first call to a role has not been answered yet, so nothing says which
	// model will read it: the pack's sizes stand.
	if _, err := service.Annotate(ctx, "run-warm", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 1 || len(decision.calls[0].GetQuestions()) != 3 {
		t.Fatalf("first calls = %d, want the pack's batch before the model is known", len(decision.calls))
	}
	decision.calls = nil
	if _, err := service.Annotate(ctx, "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 3 {
		t.Fatalf("calls = %d, want one question per call under the model's own limit", len(decision.calls))
	}
	for _, call := range decision.calls {
		if got := len(call.GetItems()[0].GetStateJson()); got > 400*charsPerToken {
			t.Fatalf("state is %d chars, want it sized below the %d the model's window allows", got, 400*charsPerToken)
		}
	}
}

func TestARestartedEngineSizesItsFirstCallFromStoredAnswers(t *testing.T) {
	labels := `[` + scoreLabel("a", "") + `,` + scoreLabel("b", "") + `]`
	first, second := runFact("run-1"), runFact("run-2")
	first.ToolSummary.ToolCalls, second.ToolSummary.ToolCalls = 4, 4
	limits := limitsFunc(func(string, string) (ModelLimits, bool) { return ModelLimits{MaxQuestions: 1}, true })
	runtime := `{"mode":"passive","default_role":"decide.fast","batch":{"max_states":8,"max_questions":4}}`
	before := &fakeDecision{}
	service, store := testServiceRuntime(t, runtime, labels, []RunFacts{first, second}, before, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	// A new process over the same store: nothing learned in memory.
	after := &fakeDecision{}
	restarted, _ := testServiceRuntime(t, runtime, labels, []RunFacts{first, second}, after, Options{Limits: limits})
	restarted.store = store
	if _, err := restarted.Annotate(context.Background(), "run-2", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(after.calls) != 2 {
		t.Fatalf("calls = %d, want the model's one-question limit applied from the first call", len(after.calls))
	}
}

func TestAQuestionBeyondTheModelsOptionLimitIsUnavailableNotAsked(t *testing.T) {
	warm, fact := runFact("run-warm"), runFact("run-1")
	warm.ToolSummary.ToolCalls, fact.ToolSummary.ToolCalls = 4, 4
	decision := &fakeDecision{}
	limits := limitsFunc(func(string, string) (ModelLimits, bool) { return ModelLimits{MaxOptions: 3}, true })
	service, _ := testServiceRuntime(t, `{"mode":"passive","default_role":"decide.fast"}`, `[`+scoreLabel("a", "")+`]`, []RunFacts{warm, fact}, decision, Options{Limits: limits})
	ctx := context.Background()
	if _, err := service.Annotate(ctx, "run-warm", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	decision.calls = nil
	view, err := service.Annotate(ctx, "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 0 || view.Labels[0].State != StateUnavailable || !strings.Contains(view.Labels[0].Reason, "question_exceeds_role_limits") {
		t.Fatalf("calls = %d view = %+v, want the five-level question refused for a three-option model", len(decision.calls), view.Labels[0])
	}
}

type limitsFunc func(provider, resourceRole string) (ModelLimits, bool)

func (f limitsFunc) ResourceLimits(_ context.Context, provider, resourceRole string) (ModelLimits, bool) {
	return f(provider, resourceRole)
}
