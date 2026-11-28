package runannotations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"

	"agent-manager/internal/toolfacts"
)

type fakeDecision struct {
	calls  []*inferencev1.DecideBatchRequest
	answer func(request *inferencev1.DecideBatchRequest, item int, question *inferencev1.DecisionQuestion) *inferencev1.DecisionAnswer
	method inferencev1.DecisionMethod
	err    error
	// overflowAbove rejects any item whose state is longer than this many
	// bytes, the way a small-context model reports context_overflow.
	overflowAbove int
	// resolve names the model and resource role that answer a request. Without
	// it every request is answered by winnow on decide.local.
	resolve func(request *inferencev1.DecideBatchRequest) (model, resourceRole string)
}

func (f *fakeDecision) DecideBatch(_ context.Context, request *inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error) {
	f.calls = append(f.calls, request)
	if f.err != nil {
		return nil, f.err
	}
	method := f.method
	if method == inferencev1.DecisionMethod_DECISION_METHOD_UNSPECIFIED {
		method = inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL
	}
	response := &inferencev1.DecideBatchResponse{}
	for i := range request.Items {
		if f.overflowAbove > 0 && len(request.Items[i].GetStateJson()) > f.overflowAbove {
			response.Results = append(response.Results, &inferencev1.DecideResponse{
				Error: &inferencev1.InferenceError{
					Code:    inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW,
					Message: "no decision candidate answered for role decide.fast",
				},
			})
			continue
		}
		model, resourceRole := "winnow", "decide.local"
		if f.resolve != nil {
			model, resourceRole = f.resolve(request)
		}
		result := &inferencev1.DecideResponse{
			Method: method, Provider: "ollaya", Model: model, ResourceRole: resourceRole,
		}
		for _, question := range request.Questions {
			if f.answer != nil {
				result.Answers = append(result.Answers, f.answer(request, i, question))
				continue
			}
			result.Answers = append(result.Answers, defaultAnswer(question))
		}
		response.Results = append(response.Results, result)
	}
	return response, nil
}

func defaultAnswer(question *inferencev1.DecisionQuestion) *inferencev1.DecisionAnswer {
	confidence := 0.8
	switch question.GetType() {
	case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE:
		choice := question.GetOptions()[0].GetLabel()
		return &inferencev1.DecisionAnswer{
			QuestionId: question.GetId(), Type: question.GetType(), Choice: choice, Confidence: &confidence,
			Probabilities: []*inferencev1.DecisionProbability{{Option: choice, Probability: 0.8}, {Option: question.GetOptions()[1].GetLabel(), Probability: 0.2}},
		}
	case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE:
		score := 2.0
		return &inferencev1.DecisionAnswer{
			QuestionId: question.GetId(), Type: question.GetType(), Score: &score, Confidence: &confidence,
			Probabilities: []*inferencev1.DecisionProbability{{Option: "0", Probability: 0.1}, {Option: "1", Probability: 0.1}, {Option: "2", Probability: 0.6}, {Option: "3", Probability: 0.1}, {Option: "4", Probability: 0.1}},
		}
	default:
		noul := 0.7
		return &inferencev1.DecisionAnswer{QuestionId: question.GetId(), Type: question.GetType(), Noul: &noul}
	}
}

func testService(t *testing.T, labels string, facts []RunFacts, decision DecisionClient, opts Options) (*Service, *Store) {
	t.Helper()
	return testServicePack(t, packWithLabels(labels), facts, decision, opts)
}

// testServiceRuntime is testService with an explicit runtime block.
func testServiceRuntime(t *testing.T, runtime, labels string, facts []RunFacts, decision DecisionClient, opts Options) (*Service, *Store) {
	t.Helper()
	return testServicePack(t, `{"pack":{"name":"test","version":"1","runtime":`+runtime+`,"labels":`+labels+`}}`, facts, decision, opts)
}

func testServicePack(t *testing.T, packJSON string, facts []RunFacts, decision DecisionClient, opts Options) (*Service, *Store) {
	t.Helper()
	pack, err := LoadCompiled(writePack(t, packJSON))
	if err != nil {
		t.Fatal(err)
	}
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	for _, fact := range facts {
		fixture.Set(fact)
	}
	opts.Store = store
	opts.Features = fixture
	opts.Decisions = decision
	if opts.Clock == nil {
		opts.Clock = func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }
	}
	service, err := NewService(pack, opts)
	if err != nil {
		t.Fatal(err)
	}
	return service, store
}

func runFact(id string) RunFacts {
	return RunFacts{
		RunID:    id,
		Workload: WorkloadFacts{Kind: "adhoc"},
		OperatorMessages: []MessageFact{
			{EventID: id + "-m1", Text: "please fix this"},
			{EventID: id + "-m2", Text: "still broken"},
		},
		AgentText: []MessageFact{{EventID: id + "-a1", Text: "I changed the file"}},
		Timings:   TimingFacts{DurationMinutes: 30},
		Terminal:  TerminalFacts{Status: "completed"},
	}
}

const decisionLabels = `[
  {
    "id":"operator.valence","group":"operator","scope":"run","method":"decision",
    "question":{"type":"score","instructions":"score affect","levels":["-2","-1","0","+1","+2"],"offset":-2},
    "evidence":[{"source":"operator_messages","field":"text"}],
    "gate":{"precision":0.7,"recall":0.6,"min_gold":1},
    "search":{"facet":"label:operator.valence","min_probability":0.6},
    "compatibility":{"change":"breaking"}
  },
  {
    "id":"agent.objective_lost","group":"agent","scope":"run","method":"decision",
    "question":{"type":"noul","instructions":"lost objective?","true_description":"yes","false_description":"no"},
    "evidence":[{"source":"operator_messages","field":"text"}],
    "gate":{"precision":0.7,"recall":0.5,"min_gold":1},
    "search":{"facet":"label:agent.objective_lost","min_probability":0.6},
    "compatibility":{"change":"breaking"}
  }
]`

func TestDecideBatchAnswersManyLabelsForManyStatesInOneCall(t *testing.T) {
	decision := &fakeDecision{}
	service, store := testService(t, decisionLabels, []RunFacts{runFact("run-1"), runFact("run-2"), runFact("run-3")}, decision, Options{})
	result, err := service.Evaluate(context.Background(), EvaluateOptions{RunIDs: []string{"run-1", "run-2", "run-3"}, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 || len(result.Runs) != 3 {
		t.Fatalf("result = %+v", result)
	}
	if len(decision.calls) != 1 {
		t.Fatalf("decide calls = %d, want 1", len(decision.calls))
	}
	call := decision.calls[0]
	if len(call.Items) != 3 || len(call.Questions) != 2 {
		t.Fatalf("items=%d questions=%d, want 3/2", len(call.Items), len(call.Questions))
	}
	if call.Profile.String() != "PROFILE_LOCAL_ONLY" || call.PrivacyClass.String() != "PRIVACY_CLASS_CONFIDENTIAL" {
		t.Fatalf("privacy = %s %s", call.Profile, call.PrivacyClass)
	}
	if call.Role != "decide.accurate" {
		t.Fatalf("role = %q", call.Role)
	}
	for _, runID := range []string{"run-1", "run-2", "run-3"} {
		stored, err := store.Current(context.Background(), ScopeRun, runID, "operator.valence", service.Pack().LabelHashes["operator.valence"])
		if err != nil || stored == nil {
			t.Fatalf("%s valence: %v %v", runID, stored, err)
		}
		if stored.Probability == nil || *stored.Probability != 0.6 {
			t.Fatalf("%s probability = %v, want 0.6", runID, stored.Probability)
		}
		if stored.Value.Score == nil || *stored.Value.Score != 0 {
			t.Fatalf("%s score = %v, want level 2 + offset -2 = 0", runID, stored.Value.Score)
		}
		if len(stored.Value.Probabilities) != 5 {
			t.Fatalf("%s probabilities = %v", runID, stored.Value.Probabilities)
		}
	}
}

func TestLLMEnumIsStoredDistinctlyWithoutProbabilities(t *testing.T) {
	decision := &fakeDecision{method: inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM}
	service, store := testService(t, decisionLabels, []RunFacts{runFact("run-1")}, decision, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Current(context.Background(), ScopeRun, "run-1", "agent.objective_lost", service.Pack().LabelHashes["agent.objective_lost"])
	if err != nil || stored == nil {
		t.Fatalf("stored = %v %v", stored, err)
	}
	if stored.Method != "LLM_ENUM" {
		t.Fatalf("method = %q, want LLM_ENUM", stored.Method)
	}
	if stored.Probability != nil || stored.Confidence != nil {
		t.Fatalf("LLM_ENUM invented calibration: p=%v c=%v", stored.Probability, stored.Confidence)
	}
	if stored.Value.Noul == nil || *stored.Value.Noul != 0.7 {
		t.Fatalf("enum value = %+v", stored.Value)
	}
	if stored.Reason != "llm_enum_no_probabilities" {
		t.Fatalf("reason = %q", stored.Reason)
	}
}

func TestDecisionClientFailureIsUnavailableNotZero(t *testing.T) {
	decision := &fakeDecision{err: errors.New("gateway down")}
	service, store := testService(t, decisionLabels, []RunFacts{runFact("run-1")}, decision, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Current(context.Background(), ScopeRun, "run-1", "operator.valence", service.Pack().LabelHashes["operator.valence"])
	if err == nil && stored != nil {
		t.Fatal("unavailable observation was stored as an observed row")
	}
	latest, err := store.LatestForScope(context.Background(), ScopeRun, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 2 {
		t.Fatalf("rows = %d, want 2 unavailable rows", len(latest))
	}
	for _, row := range latest {
		if row.State != StateUnavailable || row.Probability != nil {
			t.Fatalf("row = %+v", row)
		}
		if !strings.HasPrefix(row.Reason, "decision_unavailable") {
			t.Fatalf("reason = %q", row.Reason)
		}
	}
}

func TestDeterministicFallbackAsksTheModelOnlyWhenUndecided(t *testing.T) {
	labels := `[{
	  "id":"run.type","group":"run","scope":"run","method":"deterministic",
	  "rule":"workload_kind == 'workflow_node' ? 'workflow' : ''",
	  "fallback":"decision",
	  "question":{"type":"choice","instructions":"type?","options":[{"label":"workflow"},{"label":"adhoc"}]},
	  "evidence":[{"source":"workload","field":"kind"}],
	  "gate":{"precision":0.9,"recall":0.9,"min_gold":1},
	  "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	}]`
	facts := []RunFacts{
		{RunID: "workflow-run", Workload: WorkloadFacts{Kind: "workflow_node"}},
		{RunID: "adhoc-run", Workload: WorkloadFacts{Kind: "adhoc"}},
	}
	decision := &fakeDecision{}
	service, _ := testService(t, labels, facts, decision, Options{})
	if _, err := service.Annotate(context.Background(), "workflow-run", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 0 {
		t.Fatalf("deterministic workflow run called the model %d times", len(decision.calls))
	}
	if _, err := service.Annotate(context.Background(), "adhoc-run", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 1 || len(decision.calls[0].Items) != 1 {
		t.Fatalf("fallback calls = %d", len(decision.calls))
	}
}

func TestCompatiblePredecessorIsReusedAndBreakingIsNot(t *testing.T) {
	base := `[{
	  "id":"operator.valence","group":"operator","scope":"run","method":"decision",
	  "question":{"type":"score","instructions":"%s","levels":["0","1","2"]},
	  "evidence":[{"source":"operator_messages","field":"text"}],
	  "gate":{"precision":0.7,"recall":0.6,"min_gold":1},
	  "search":{"facet":"label:operator.valence","min_probability":0.6},
	  "compatibility":{"change":"%s","compatible_with":[%s]}
	}]`
	v1 := fmt.Sprintf(base, "score affect v1", "breaking", "")
	pack1, err := LoadCompiled(packPathForTest(t, v1))
	if err != nil {
		t.Fatal(err)
	}
	hash1 := pack1.LabelHashes["operator.valence"]
	decision1 := &fakeDecision{answer: func(_ *inferencev1.DecideBatchRequest, _ int, question *inferencev1.DecisionQuestion) *inferencev1.DecisionAnswer {
		score, confidence := 1.0, 0.9
		return &inferencev1.DecisionAnswer{
			QuestionId: question.GetId(), Type: question.GetType(), Score: &score, Confidence: &confidence,
			Probabilities: []*inferencev1.DecisionProbability{{Option: "0", Probability: 0.05}, {Option: "1", Probability: 0.9}, {Option: "2", Probability: 0.05}},
		}
	}}
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	fixture.Set(runFact("run-1"))
	service1, err := NewService(pack1, Options{Store: store, Features: fixture, Decisions: decision1, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service1.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}

	// A reworded question that declares the old hash compatible reuses the
	// observation and makes no model call.
	v2 := fmt.Sprintf(base, "score affect, reworded", "compatible", `"`+hash1+`"`)
	pack2, err := LoadCompiled(packPathForTest(t, v2))
	if err != nil {
		t.Fatal(err)
	}
	decision2 := &fakeDecision{err: errors.New("must not be called for a compatible predecessor")}
	service2, err := NewService(pack2, Options{Store: store, Features: fixture, Decisions: decision2, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	view, err := service2.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision2.calls) != 0 {
		t.Fatalf("compatible reuse called the model %d times", len(decision2.calls))
	}
	if len(view.Labels) != 1 || view.Labels[0].State != StateCompatible {
		t.Fatalf("view = %+v", view.Labels)
	}
	if view.Labels[0].Probability == nil || *view.Labels[0].Probability != 0.9 {
		t.Fatalf("reused probability = %v", view.Labels[0].Probability)
	}

	// A breaking change with no declared predecessor evaluates fresh.
	v3 := fmt.Sprintf(base, "score affect, changed levels", "breaking", "")
	pack3, err := LoadCompiled(packPathForTest(t, v3))
	if err != nil {
		t.Fatal(err)
	}
	decision3 := &fakeDecision{}
	service3, err := NewService(pack3, Options{Store: store, Features: fixture, Decisions: decision3, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service3.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision3.calls) != 1 {
		t.Fatalf("breaking change did not evaluate: %d calls", len(decision3.calls))
	}
	if view.Labels[0].State != StateObserved {
		t.Fatalf("state = %s", view.Labels[0].State)
	}
}

func TestPriorRuleAvoidsTheModelAndOptionalIsSkipped(t *testing.T) {
	labels := `[
	  {
	    "id":"agent.objective_lost","group":"agent","scope":"run","method":"decision",
	    "prior_rule":"'objective-lost' in stepback_triggers",
	    "question":{"type":"noul","instructions":"lost?","true_description":"yes","false_description":"no"},
	    "evidence":[{"source":"deterministic_flags","field":"stepback_triggers"}],
	    "gate":{"precision":0.7,"recall":0.5,"min_gold":1},
	    "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	  },
	  {
	    "id":"operator.emotion","group":"operator","scope":"run","method":"decision","optional":true,
	    "question":{"type":"choice","instructions":"emotion?","options":[{"label":"joy"},{"label":"anger"}]},
	    "evidence":[{"source":"operator_messages","field":"text"}],
	    "gate":{"precision":0.6,"recall":0.5,"min_gold":1},
	    "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	  }
	]`
	fact := runFact("run-1")
	fact.DeterministicFlags.StepbackTriggers = []string{"objective-lost"}
	decision := &fakeDecision{}
	service, store := testService(t, labels, []RunFacts{fact}, decision, Options{})
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 0 {
		t.Fatalf("prior called the model %d times", len(decision.calls))
	}
	if len(view.Labels) != 1 || view.Labels[0].Probability == nil || *view.Labels[0].Probability != 1 {
		t.Fatalf("prior annotation = %+v", view.Labels)
	}
	if len(view.Skipped) != 1 || view.Skipped[0].Reason != "optional_not_requested" {
		t.Fatalf("skipped = %+v", view.Skipped)
	}
	stored, err := store.Current(context.Background(), ScopeRun, "run-1", "agent.objective_lost", service.Pack().LabelHashes["agent.objective_lost"])
	if err != nil || stored == nil || stored.Reason != "deterministic_prior" {
		t.Fatalf("stored prior = %+v %v", stored, err)
	}
}

func TestSignalLabelDistinguishesAbsentFromObserved(t *testing.T) {
	labels := `[{
	  "id":"operator.frustration","group":"operator","scope":"run","method":"signal","source":"frustration",
	  "question":{"type":"noul","instructions":"frustrated?","true_description":"yes","false_description":"no"},
	  "evidence":[{"source":"existing_signals","field":"probability","label":"frustration"}],
	  "gate":{"precision":0.7,"recall":0.5,"min_gold":1},
	  "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	}]`
	present := runFact("run-1")
	p := 0.81
	present.ExistingSignals = map[string]SignalFact{"frustration": {Label: "frustration", Probability: &p, LatestEventID: "signal-1", Version: "v1"}}
	absent := runFact("run-2")
	service, store := testService(t, labels, []RunFacts{present, absent}, nil, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Annotate(context.Background(), "run-2", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	hash := service.Pack().LabelHashes["operator.frustration"]
	stored, _ := store.Current(context.Background(), ScopeRun, "run-1", "operator.frustration", hash)
	if stored == nil || stored.Probability == nil || *stored.Probability != 0.81 || len(stored.EvidenceEventIDs) != 1 {
		t.Fatalf("present = %+v", stored)
	}
	missing, _ := store.Current(context.Background(), ScopeRun, "run-2", "operator.frustration", hash)
	if missing != nil {
		t.Fatal("absent signal stored as an observed row")
	}
	latest, _ := store.LatestForScope(context.Background(), ScopeRun, "run-2")
	if len(latest) != 1 || latest[0].State != StateUnavailable || latest[0].Reason != "signal_not_present" {
		t.Fatalf("absent = %+v", latest)
	}
}

func TestExtractIsDeclaredUnavailableAndCoverageNeverInventsData(t *testing.T) {
	labels := `[{
	  "id":"run.summary","group":"run","scope":"run","method":"extract",
	  "evidence":[{"source":"operator_messages","field":"text"}],
	  "gate":{"precision":0.7,"recall":0.5,"min_gold":1},
	  "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	}]`
	service, store := testService(t, labels, []RunFacts{runFact("run-1")}, nil, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	latest, _ := store.LatestForScope(context.Background(), ScopeRun, "run-1")
	if len(latest) != 1 || latest[0].State != StateUnavailable || latest[0].Reason != "extract_not_implemented_in_slice" {
		t.Fatalf("extract = %+v", latest)
	}
	coverage, err := service.Coverage(context.Background(), "7d")
	if err != nil {
		t.Fatal(err)
	}
	if len(coverage.Labels) != 1 || coverage.Summary.Unavailable != 1 || coverage.Summary.Observed != 0 {
		t.Fatalf("coverage = %+v", coverage)
	}
	if coverage.Labels[0].Gate.State != "unmeasured" || coverage.Labels[0].Rate != nil {
		t.Fatal("an unmeasured gate reported proven")
	}
	empty, emptyStore := testService(t, labels, nil, nil, Options{})
	_ = emptyStore
	response, err := empty.Coverage(context.Background(), "7d")
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary.Total() != 0 {
		t.Fatalf("empty summary = %+v", response.Summary)
	}
	if len(response.Notes) == 0 || !strings.Contains(response.Notes[0], "unknown, not zero") {
		t.Fatalf("notes = %v", response.Notes)
	}
}

func TestAddingALabelIsDataNotCode(t *testing.T) {
	labels := `[
	  {
	    "id":"work.changed_files","group":"work","scope":"run","method":"deterministic",
	    "rule":"tool_calls >= 1 ? 'touched' : 'read_only'",
	    "evidence":[{"source":"tool_summary","field":"calls"}],
	    "gate":{"precision":0.9,"recall":0.9,"min_gold":1},
	    "search":{"facet":"label:work.changed_files","min_probability":0.6},
	    "compatibility":{"change":"breaking"}
	  }
	]`
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 3
	service, _ := testService(t, labels, []RunFacts{fact}, nil, Options{})
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Labels) != 1 || view.Labels[0].Value.Text != "touched" {
		t.Fatalf("view = %+v", view.Labels)
	}
	if view.Labels[0].Method != "deterministic" {
		t.Fatalf("method = %q", view.Labels[0].Method)
	}
}

func TestCompositeCombinesOtherLabels(t *testing.T) {
	labels := `[
	  {
	    "id":"operator.frustration","group":"operator","scope":"run","method":"decision",
	    "question":{"type":"noul","instructions":"frustrated?","true_description":"yes","false_description":"no"},
	    "evidence":[{"source":"operator_messages","field":"text"}],
	    "gate":{"precision":0.7,"recall":0.5,"min_gold":1},
	    "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	  },
	  {
	    "id":"operator.distress","group":"operator","scope":"run","method":"composite",
	    "rule":"labels['operator.frustration'] * 0.5",
	    "question":{"type":"noul","instructions":"distress","true_description":"yes","false_description":"no"},
	    "evidence":[{"source":"existing_signals","field":"probability","label":"frustration"}],
	    "gate":{"precision":0.7,"recall":0.5,"min_gold":1},
	    "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	  }
	]`
	decision := &fakeDecision{answer: func(_ *inferencev1.DecideBatchRequest, _ int, question *inferencev1.DecisionQuestion) *inferencev1.DecisionAnswer {
		noul := 0.8
		return &inferencev1.DecisionAnswer{QuestionId: question.GetId(), Type: question.GetType(), Noul: &noul}
	}}
	service, _ := testService(t, labels, []RunFacts{runFact("run-1")}, decision, Options{})
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var composite *AnnotationView
	for i := range view.Labels {
		if view.Labels[i].ID == "operator.distress" {
			composite = &view.Labels[i]
		}
	}
	if composite == nil || composite.Method != "composite" || composite.Probability == nil || *composite.Probability != 0.4 {
		t.Fatalf("composite = %+v", composite)
	}
}

func TestCoverageContractShapeIsExplicitOnEmptyStore(t *testing.T) {
	labels := `[{
	  "id":"agent.stalled","group":"agent","scope":"run","method":"deterministic",
	  "rule":"tool_calls >= 1 ? 'yes' : 'no'",
	  "evidence":[{"source":"tool_summary","field":"calls"}],
	  "gate":{"precision":0.7,"recall":0.5,"min_gold":10},
	  "search":{"facet":"label:agent.stalled","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	service, _ := testService(t, labels, nil, nil, Options{})
	response, err := service.Coverage(context.Background(), "7d")
	if err != nil {
		t.Fatal(err)
	}
	if response.Window.Until == "" || response.Pack.Name != "test" {
		t.Fatalf("window/pack = %+v %+v", response.Window, response.Pack)
	}
	if response.EligibleRuns != 0 || response.ObservedRuns != 0 || response.Coverage != 0 {
		t.Fatalf("coverage = %+v", response)
	}
	if len(response.Labels) != 0 {
		t.Fatalf("empty store returned label rows: %+v", response.Labels)
	}
	if len(response.Notes) == 0 || !strings.Contains(response.Notes[0], "unknown, not zero") {
		t.Fatalf("notes = %v", response.Notes)
	}
}

func TestCoverageReportsRateOnlyForAProvenGate(t *testing.T) {
	unproven := `[{
	  "id":"agent.stalled","group":"agent","scope":"run","method":"deterministic",
	  "rule":"tool_calls >= 1 ? 'yes' : 'no'",
	  "evidence":[{"source":"tool_summary","field":"calls"}],
	  "gate":{"precision":0.7,"recall":0.5,"min_gold":10},
	  "search":{"facet":"label:agent.stalled","min_probability":0.6},
	  "friction":{"enabled":true,"min_probability":0.6,"cost_unit":"run","importance":"high"},
	  "compatibility":{"change":"breaking"}
	}]`
	fact := runFact("run-1")
	fact.ToolSummary.ToolCalls = 2
	service, _ := testService(t, unproven, []RunFacts{fact}, nil, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	response, err := service.Coverage(context.Background(), "7d")
	if err != nil {
		t.Fatal(err)
	}
	label := response.Labels[0]
	if label.State != "unproven" || label.Rate != nil {
		t.Fatalf("unproven label = %+v", label)
	}
	if label.Observed != 1 || label.Eligible != 1 {
		t.Fatalf("observed/eligible = %d/%d", label.Observed, label.Eligible)
	}
	if response.EligibleRuns != 1 || response.ObservedRuns != 1 || response.Coverage != 1 {
		t.Fatalf("run coverage = %+v", response)
	}

	unprovenPack, err := ParsePack([]byte(packWithLabels(unproven)))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := LabelDefinitionHash(unprovenPack.Labels[0])
	if err != nil {
		t.Fatal(err)
	}
	proven := strings.Replace(unproven, `"min_gold":10`,
		fmt.Sprintf(`"min_gold":10,"measurements":[{"definition_hash":%q,"model":"rule","resource_role":"","precision":0.9,"recall":0.8,"gold":10,"positives":4,"threshold":0.5,"temperature":1,"split":"holdout"}]`, hash), 1)
	provenService, _ := testService(t, proven, []RunFacts{fact}, nil, Options{})
	if _, err := provenService.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	response, err = provenService.Coverage(context.Background(), "7d")
	if err != nil {
		t.Fatal(err)
	}
	label = response.Labels[0]
	if label.State != "proven" || label.Rate == nil || *label.Rate != 1 {
		t.Fatalf("proven label = %+v", label)
	}
	if label.Gate.Precision != 0.9 || label.Gate.Gold != 10 || label.Gate.MinGold != 10 {
		t.Fatalf("gate = %+v", label.Gate)
	}
}

func TestRunViewLeadsWithCoverageAndUsesContractLabelIDs(t *testing.T) {
	labels := `[{
	  "id":"agent.stalled","group":"agent","scope":"run","method":"deterministic",
	  "rule":"'yes'","evidence":[{"source":"tool_summary","field":"calls"}],
	  "gate":{"precision":0.7,"recall":0.5,"min_gold":10},
	  "search":{"facet":"label:agent.stalled","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	service, _ := testService(t, labels, []RunFacts{runFact("run-1")}, nil, Options{})
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if view.Coverage.Observed != 1 || view.Coverage.Labels != 1 {
		t.Fatalf("coverage = %+v", view.Coverage)
	}
	if len(view.Labels) != 1 || view.Labels[0].ID != "agent.stalled" {
		t.Fatalf("labels = %+v", view.Labels)
	}
	if view.Labels[0].Gate.State != "unmeasured" {
		t.Fatalf("gate = %+v", view.Labels[0].Gate)
	}
}

func TestEvaluateContractShapeAndDryRunDoesNotPersist(t *testing.T) {
	labels := `[{
	  "id":"run.type","group":"run","scope":"run","method":"deterministic",
	  "rule":"workload_kind","evidence":[{"source":"workload","field":"kind"}],
	  "gate":{"precision":0.9,"recall":0.9,"min_gold":10},
	  "search":{"facet":"label:run.type","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	service, store := testService(t, labels, []RunFacts{runFact("run-1"), runFact("run-2")}, nil, Options{})
	result, err := service.Evaluate(context.Background(), EvaluateOptions{
		RunIDs: []string{"run-1", "run-2"}, Limit: 2, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Submitted != 2 || result.Observed != 2 || result.Coverage != 1 {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Labels) != 1 || result.Labels[0].ID != "run.type" || result.Labels[0].State != "unproven" {
		t.Fatalf("labels = %+v", result.Labels)
	}
	if result.Window.Until == "" {
		t.Fatalf("window = %+v", result.Window)
	}
	count, err := store.Count(context.Background())
	if err != nil || count != 0 {
		t.Fatalf("dry run persisted %d rows: %v", count, err)
	}
}

type fixedCensus int

func (c fixedCensus) EligibleRuns(context.Context, time.Time, time.Time) (int, error) {
	return int(c), nil
}

func TestCoverageUsesAnOptionalCandidateCensus(t *testing.T) {
	labels := `[{
	  "id":"run.type","group":"run","scope":"run","method":"deterministic",
	  "rule":"workload_kind","evidence":[{"source":"workload","field":"kind"}],
	  "gate":{"precision":0.9,"recall":0.9,"min_gold":10},
	  "search":{"facet":"label:run.type","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	service, _ := testService(t, labels, []RunFacts{runFact("run-1")}, nil, Options{Census: fixedCensus(100)})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	response, err := service.Coverage(context.Background(), "7d")
	if err != nil {
		t.Fatal(err)
	}
	if response.CoverageBasis != "candidate_census" || response.EligibleRuns != 100 || response.ObservedRuns != 1 {
		t.Fatalf("coverage = %+v", response)
	}
	if response.Coverage != 0.01 {
		t.Fatalf("coverage ratio = %v", response.Coverage)
	}
}

func TestAnnotateCacheAvoidsASecondModelCall(t *testing.T) {
	decision := &fakeDecision{}
	service, _ := testService(t, decisionLabels, []RunFacts{runFact("run-1")}, decision, Options{})
	for range 3 {
		if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	if len(decision.calls) != 1 {
		t.Fatalf("cached annotation called the model %d times", len(decision.calls))
	}
}

func TestEvaluateBoundsRunsByThePackBudget(t *testing.T) {
	labels := `[{
	  "id":"run.type","group":"run","scope":"run","method":"deterministic",
	  "rule":"workload_kind","evidence":[{"source":"workload","field":"kind"}],
	  "gate":{"precision":0.9,"recall":0.9,"min_gold":1},
	  "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	}]`
	pack, err := LoadCompiled(packPathForTest(t, labels))
	if err != nil {
		t.Fatal(err)
	}
	pack.Pack.Runtime.Budgets.MaxRunsPerPass = 2
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	for i := 0; i < 5; i++ {
		fixture.Set(runFact(fmt.Sprintf("run-%d", i)))
	}
	service, err := NewService(pack, Options{Store: store, Features: fixture, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Evaluate(context.Background(), EvaluateOptions{RunIDs: []string{"run-0", "run-1", "run-2", "run-3", "run-4"}, Limit: 99})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Runs) != 2 {
		t.Fatalf("runs = %d, want the budget of 2", len(result.Runs))
	}
}

func fixedClock() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }

func longRunFact(id string) RunFacts {
	fact := runFact(id)
	fact.OperatorMessages = nil
	for i := 0; i < 12; i++ {
		fact.OperatorMessages = append(fact.OperatorMessages, MessageFact{
			EventID: id + "-m" + string(rune('a'+i)), Text: strings.Repeat("operator detail ", 30),
		})
	}
	fact.AgentText = nil
	for i := 0; i < 8; i++ {
		fact.AgentText = append(fact.AgentText, MessageFact{
			EventID: id + "-a" + string(rune('a'+i)), Text: strings.Repeat("agent detail ", 30),
		})
	}
	return fact
}

func TestDecisionOverflowRetriesWithASmallerDigest(t *testing.T) {
	decision := &fakeDecision{overflowAbove: 1400}
	service, store := testService(t, decisionLabels, []RunFacts{longRunFact("run-1")}, decision, Options{})
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) < 2 {
		t.Fatalf("decide calls = %d, want a shrink retry", len(decision.calls))
	}
	first, retried := decision.calls[0].Items[0].GetStateJson(), ""
	for _, call := range decision.calls[1:] {
		for _, item := range call.Items {
			if len(item.GetStateJson()) < len(first) {
				retried = item.GetStateJson()
			}
		}
	}
	if retried == "" {
		t.Fatalf("no retry used a smaller state; first was %d bytes", len(first))
	}
	for _, label := range view.Labels {
		if label.State != StateObserved {
			t.Fatalf("%s = %s (%s)", label.ID, label.State, label.Reason)
		}
	}
	stored, err := store.Current(context.Background(), ScopeRun, "run-1", "operator.valence", service.Pack().LabelHashes["operator.valence"])
	if err != nil || stored == nil || stored.State != StateObserved {
		t.Fatalf("stored valence = %+v err=%v", stored, err)
	}
}

func TestDecisionOverflowIsReportedAfterOneShrinkAttempt(t *testing.T) {
	decision := &fakeDecision{overflowAbove: 10}
	service, _ := testService(t, decisionLabels, []RunFacts{longRunFact("run-1")}, decision, Options{})
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// The retry ladder splits a multi-question chunk, then shrinks the state
	// once. It must terminate rather than recurse on an unreadable state.
	if len(decision.calls) > 8 {
		t.Fatalf("decide calls = %d, retry did not terminate", len(decision.calls))
	}
	for _, label := range view.Labels {
		if label.State != StateUnavailable {
			t.Fatalf("a state no model can read must stay unknown: %s = %s (%s)", label.ID, label.State, label.Reason)
		}
		if !strings.Contains(label.Reason, "context_overflow") {
			t.Fatalf("%s reason = %q, want the real overflow reason", label.ID, label.Reason)
		}
	}
}

func TestDecisionBudgetBoundsStatesNotLabels(t *testing.T) {
	body := `{"pack":{"name":"test","version":"1","runtime":{"budgets":{"max_runs_per_pass":10,"max_decision_states_per_pass":2}},"labels":` + decisionLabels + `}}`
	path := writePack(t, body)
	pack, err := LoadCompiled(path)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	facts := []RunFacts{runFact("run-1"), runFact("run-2"), runFact("run-3"), runFact("run-4")}
	for _, fact := range facts {
		fixture.Set(fact)
	}
	decision := &fakeDecision{}
	service, err := NewService(pack, Options{
		Store: store, Features: fixture, Decisions: decision,
		Clock: func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Evaluate(context.Background(), EvaluateOptions{RunIDs: []string{"run-1", "run-2", "run-3", "run-4"}, Limit: 4})
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, call := range decision.calls {
		total += len(call.Items)
	}
	if total > 2 {
		t.Fatalf("decision states = %d, budget = 2", total)
	}
	unavailable := 0
	for _, view := range result.Runs {
		for _, label := range view.Labels {
			if label.State == StateUnavailable && strings.Contains(label.Reason, "decision_budget_exhausted") {
				unavailable++
			}
		}
	}
	if unavailable == 0 {
		t.Fatalf("budget overflow was not reported: %+v", result.Runs)
	}
}

const signalFallbackLabel = `[
  {
    "id":"operator.frustration","group":"operator","scope":"run","method":"signal","source":"frustration",
    "fallback":"decision",
    "question":{"type":"noul","instructions":"frustrated?","true_description":"yes","false_description":"no"},
    "evidence":[{"source":"existing_signals","field":"probability","label":"frustration"}],
    "gate":{"precision":0.8,"recall":0.7,"min_gold":1},
    "search":{"facet":"label:operator.frustration","min_probability":0.6},
    "friction":{"enabled":false,"cost_unit":"run","importance":"high"},
    "compatibility":{"change":"breaking"}
  }
]`

func TestSignalFallsBackToDecisionOnlyWhenTheSignalIsAbsent(t *testing.T) {
	decision := &fakeDecision{}
	service, _ := testService(t, signalFallbackLabel, []RunFacts{runFact("run-1")}, decision, Options{})
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 1 {
		t.Fatalf("decision calls = %d, want the declared fallback", len(decision.calls))
	}
	if len(view.Labels) != 1 || view.Labels[0].State != StateObserved || view.Labels[0].Method != "decision" {
		t.Fatalf("fallback observation = %+v", view.Labels)
	}

	confidence := 0.9
	present := runFact("run-2")
	present.ExistingSignals = map[string]SignalFact{
		"frustration": {Label: "frustration", Probability: &confidence, Version: "v1", LatestEventID: "signal-1"},
	}
	signalDecision := &fakeDecision{}
	signalService, _ := testService(t, signalFallbackLabel, []RunFacts{present}, signalDecision, Options{})
	view, err = signalService.Annotate(context.Background(), "run-2", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(signalDecision.calls) != 0 {
		t.Fatalf("decision calls = %d, want the stored signal to win", len(signalDecision.calls))
	}
	if len(view.Labels) != 1 || view.Labels[0].State != StateObserved || view.Labels[0].Method != "signal" {
		t.Fatalf("signal observation = %+v", view.Labels)
	}
}

func TestUnavailableObservationIsRetriedWithoutForce(t *testing.T) {
	decision := &fakeDecision{}
	service, store := testService(t, decisionLabels, []RunFacts{runFact("run-1")}, decision, Options{})
	hash := service.Pack().LabelHashes["operator.valence"]
	stale := annotation("run-1", "operator.valence", hash, time.Date(2026, 11, 18, 11, 0, 0, 0, time.UTC), nil)
	stale.State = StateUnavailable
	stale.Reason = "decision_unavailable: ai-gateway was restarting"
	if err := store.Save(context.Background(), []Annotation{stale}); err != nil {
		t.Fatal(err)
	}
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) == 0 {
		t.Fatal("a transiently unavailable label was reused instead of retried")
	}
	for _, label := range view.Labels {
		if label.ID == "operator.valence" && label.State != StateObserved {
			t.Fatalf("valence = %s (%s), want the retried answer", label.State, label.Reason)
		}
	}
}

type flakyDecision struct {
	calls     int
	failFirst int
}

func (f *flakyDecision) DecideBatch(_ context.Context, request *inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error) {
	f.calls++
	if f.calls <= f.failFirst {
		return nil, errors.New("decision timeout")
	}
	response := &inferencev1.DecideBatchResponse{}
	for range request.Items {
		result := &inferencev1.DecideResponse{Method: inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL}
		for _, question := range request.Questions {
			result.Answers = append(result.Answers, defaultAnswer(question))
		}
		response.Results = append(response.Results, result)
	}
	return response, nil
}

func TestTransportFailureIsRetriedOnceWithTheSameQuestions(t *testing.T) {
	decision := &flakyDecision{failFirst: 1}
	service, _ := testService(t, decisionLabels, []RunFacts{runFact("run-1"), runFact("run-2")}, decision, Options{})
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	// One failed batch plus one retry of the affected states.
	if decision.calls != 2 {
		t.Fatalf("decide calls = %d, want one retry", decision.calls)
	}
	view, err := service.Run(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range view.Labels {
		if label.State != StateObserved {
			t.Fatalf("%s = %s (%s), want the retried answer", label.ID, label.State, label.Reason)
		}
	}
}

func TestGoldDraftDefaultsToTheLabelGateSize(t *testing.T) {
	labels := strings.ReplaceAll(decisionLabels, `"min_gold":1`, `"min_gold":25`)
	facts := []RunFacts{runFact("run-1"), runFact("run-2"), runFact("run-3")}
	service, _ := testService(t, labels, facts, &fakeDecision{}, Options{})
	for _, fact := range facts {
		if _, err := service.Annotate(context.Background(), fact.RunID, AnnotateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	draft, err := service.GoldDraft(context.Background(), "operator.valence", "7d", 0)
	if err != nil {
		t.Fatal(err)
	}
	if draft.MinGold != 25 || draft.Limit != 25 {
		t.Fatalf("draft gate=%d limit=%d, want the gate to raise the default limit", draft.MinGold, draft.Limit)
	}
	if len(draft.Runs) > 3 {
		t.Fatalf("runs = %d, want the three observed runs", len(draft.Runs))
	}
	for _, run := range draft.Runs {
		if strings.TrimSpace(run.Digest) == "" {
			t.Fatalf("run %s has no digest", run.RunID)
		}
	}
}

func TestEvaluatePersistsFinishedGroupsBeforeThePassBudgetExpires(t *testing.T) {
	labels := `[{
	  "id":"run.type","group":"run","scope":"run","method":"deterministic",
	  "rule":"workload_kind","evidence":[{"source":"workload","field":"kind"}],
	  "gate":{"precision":0.9,"recall":0.9,"min_gold":1},
	  "search":{"facet":"f"},"compatibility":{"change":"breaking"}
	}]`
	pack, err := LoadCompiled(packPathForTest(t, labels))
	if err != nil {
		t.Fatal(err)
	}
	pack.Pack.Runtime.Batch.MaxStates = 2
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	runIDs := make([]string, 0, 5)
	for index := 0; index < 5; index++ {
		id := fmt.Sprintf("run-%d", index)
		runIDs = append(runIDs, id)
		fixture.Set(runFact(id))
	}
	service, err := NewService(pack, Options{Store: store, Features: fixture, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Evaluate(context.Background(), EvaluateOptions{
		RunIDs: runIDs, Limit: 99, TimeBudget: time.Nanosecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || result.Remaining != 3 || result.Submitted != 2 {
		t.Fatalf("bounded pass = submitted %d remaining %d truncated %v, want 2/3/true",
			result.Submitted, result.Remaining, result.Truncated)
	}
	if count, err := store.Count(context.Background()); err != nil || count != 2 {
		t.Fatalf("persisted rows after a bounded pass = %d (%v), want the first group", count, err)
	}
	// The next pass continues where the first stopped; a negative budget means
	// no wall-clock bound, which is what the single-run path and tests use.
	continued, err := service.Evaluate(context.Background(), EvaluateOptions{
		RunIDs: runIDs, Limit: 99, TimeBudget: -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if continued.Truncated || continued.Remaining != 0 || continued.Submitted != 5 {
		t.Fatalf("continuation = %+v, want the whole selection", continued)
	}
	if count, err := store.Count(context.Background()); err != nil || count != 5 {
		t.Fatalf("persisted rows after the continuation = %d (%v), want 5", count, err)
	}
}

func TestRequireSkipsALabelWithoutEvidenceInsteadOfGuessing(t *testing.T) {
	labels := `[{
	  "id":"agent.stalled","group":"agent","scope":"run","method":"decision",
	  "require":"agent_text_count > 0 || tool_calls > 0",
	  "question":{"type":"noul","instructions":"did the agent spin?",
	    "true_description":"yes","false_description":"no"},
	  "evidence":[{"source":"agent_text","field":"last_text"}],
	  "gate":{"precision":0.8,"recall":0.7,"min_gold":10},
	  "search":{"facet":"f","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	decision := &fakeDecision{}
	empty := runFact("run-empty")
	empty.OperatorMessages, empty.AgentText, empty.Timings = nil, nil, TimingFacts{}
	service, _ := testService(t, labels, []RunFacts{empty, runFact("run-full")}, decision, Options{})
	if _, err := service.Annotate(context.Background(), "run-empty", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != 0 {
		t.Fatalf("an evidence-free run asked the model %d time(s)", len(decision.calls))
	}
	view, err := service.Annotate(context.Background(), "run-empty", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Labels) != 0 {
		t.Fatalf("labels = %d, want an evidence-free run to carry no observation", len(view.Labels))
	}
	skipped := map[string]string{}
	for _, skip := range view.Skipped {
		skipped[skip.LabelID] = skip.Reason
	}
	if skipped["agent.stalled"] != "insufficient_evidence" {
		t.Fatalf("skip reason = %q, want insufficient_evidence", skipped["agent.stalled"])
	}
	if _, err := service.Annotate(context.Background(), "run-full", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) == 0 {
		t.Fatal("a run with agent text was never asked")
	}
}

func TestChangedEvidenceInvalidatesTheCachedAnswer(t *testing.T) {
	pack, err := LoadCompiled(packPathForTest(t, decisionLabels))
	if err != nil {
		t.Fatal(err)
	}
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	fact := runFact("run-1")
	fixture.Set(fact)
	decision := &fakeDecision{}
	service, err := NewService(pack, Options{Store: store, Features: fixture, Decisions: decision, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	first := len(decision.calls)
	if first == 0 {
		t.Fatal("the first annotation asked no model")
	}
	// The same evidence is a cache hit.
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) != first {
		t.Fatalf("unchanged evidence re-asked the model (%d -> %d calls)", first, len(decision.calls))
	}
	// An appended operator message is new evidence for the same definition; a
	// stale cached answer must not survive it.
	fact.OperatorMessages = append(fact.OperatorMessages, MessageFact{EventID: "run-1-m3", Text: "this is still broken"})
	fixture.Set(fact)
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(decision.calls) == first {
		t.Fatal("new evidence reused the cached answer")
	}
}

func TestLostEvidenceSupersedesAnEarlierAnswer(t *testing.T) {
	labels := `[{
	  "id":"agent.stalled","group":"agent","scope":"run","method":"decision",
	  "require":"agent_text_count > 0 || tool_calls > 0",
	  "question":{"type":"noul","instructions":"did the agent spin?",
	    "true_description":"yes","false_description":"no"},
	  "evidence":[{"source":"agent_text","field":"last_text"}],
	  "gate":{"precision":0.8,"recall":0.7,"min_gold":10},
	  "search":{"facet":"f","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	pack, err := LoadCompiled(packPathForTest(t, labels))
	if err != nil {
		t.Fatal(err)
	}
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	fact := runFact("run-1")
	fixture.Set(fact)
	decision := &fakeDecision{}
	service, err := NewService(pack, Options{Store: store, Features: fixture, Decisions: decision, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	before, err := latestRow(store, "run-1", "agent.stalled")
	if err != nil || before == nil || before.State != StateObserved {
		t.Fatalf("first annotation = %+v (%v), want observed", before, err)
	}
	// The evidence disappears (for example a classifier stopped treating a
	// harness block as operator text).
	fact.AgentText = nil
	fact.OperatorMessages = nil
	fact.ToolSummary = toolfacts.Facts{}
	fixture.Set(fact)
	view, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Skipped) != 1 || view.Skipped[0].Reason != "insufficient_evidence" {
		t.Fatalf("skipped = %+v, want the evidence guard to fire", view.Skipped)
	}
	after, err := latestRow(store, "run-1", "agent.stalled")
	if err != nil || after == nil {
		t.Fatalf("latest after losing evidence = %+v (%v)", after, err)
	}
	if after.State != StateNotObserved || after.Reason != "insufficient_evidence" {
		t.Fatalf("latest after losing evidence = %s/%s, want not_observed/insufficient_evidence", after.State, after.Reason)
	}
}

func TestLostCandidacySupersedesAnEarlierAnswer(t *testing.T) {
	labels := `[{
	  "id":"operator.frustration","group":"operator","scope":"run","method":"decision",
	  "candidates":"operator_message_count >= 1",
	  "question":{"type":"noul","instructions":"is the operator frustrated?",
	    "true_description":"yes","false_description":"no"},
	  "evidence":[{"source":"operator_messages","field":"last_text"}],
	  "gate":{"precision":0.8,"recall":0.7,"min_gold":10},
	  "search":{"facet":"f","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	pack, err := LoadCompiled(packPathForTest(t, labels))
	if err != nil {
		t.Fatal(err)
	}
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	fact := runFact("run-1")
	fixture.Set(fact)
	service, err := NewService(pack, Options{Store: store, Features: fixture, Decisions: &fakeDecision{}, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	// The messages vanish from the readable facts (a reclassified harness
	// block, a retention purge): the label is no longer a candidate.
	fact.OperatorMessages = nil
	fixture.Set(fact)
	if _, err := service.Annotate(context.Background(), "run-1", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	latest, err := latestRow(store, "run-1", "operator.frustration")
	if err != nil || latest == nil {
		t.Fatalf("latest = %+v (%v)", latest, err)
	}
	if latest.State != StateNotObserved || latest.Reason != "not_a_candidate" {
		t.Fatalf("latest = %s/%s, want not_observed/not_a_candidate", latest.State, latest.Reason)
	}
}

type deadlineRecorder struct {
	deadlines []time.Time
	calls     int
}

func (d *deadlineRecorder) DecideBatch(ctx context.Context, _ *inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error) {
	d.calls++
	if deadline, ok := ctx.Deadline(); ok {
		d.deadlines = append(d.deadlines, deadline)
	}
	return nil, errors.New("simulated gateway hang")
}

func TestPassBudgetBoundsTheDecisionCall(t *testing.T) {
	labels := `[{
	  "id":"operator.valence","group":"operator","scope":"run","method":"decision",
	  "candidates":"operator_message_count >= 1",
	  "question":{"type":"noul","instructions":"is the operator positive?",
	    "true_description":"yes","false_description":"no"},
	  "evidence":[{"source":"operator_messages","field":"last_text"}],
	  "gate":{"precision":0.8,"recall":0.7,"min_gold":10},
	  "search":{"facet":"f","min_probability":0.6},
	  "compatibility":{"change":"breaking"}
	}]`
	pack, err := LoadCompiled(packPathForTest(t, labels))
	if err != nil {
		t.Fatal(err)
	}
	store, _ := testStore(t)
	fixture := NewFixtureSource()
	fixture.Set(runFact("run-1"))
	recorder := &deadlineRecorder{}
	service, err := NewService(pack, Options{Store: store, Features: fixture, Decisions: recorder, Clock: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if _, err := service.Evaluate(context.Background(), EvaluateOptions{
		RunIDs: []string{"run-1"}, Limit: 1, TimeBudget: 30 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	if len(recorder.deadlines) == 0 {
		t.Fatal("the decision call had no deadline")
	}
	budget := recorder.deadlines[0].Sub(started)
	if budget <= 0 || budget > 31*time.Second {
		t.Fatalf("decision deadline = %s after start, want within the 30s pass budget", budget)
	}
}
