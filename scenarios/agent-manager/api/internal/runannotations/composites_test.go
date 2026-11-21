package runannotations

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// A mood pack: two decision labels answered per human message, a message-scope
// composite over them (valence = pleased - frustrated), a run composite over the
// message labels, and a composite over facts only.
func moodLabels(t *testing.T, proven ...string) string {
	t.Helper()
	build := func(hashes map[string]string) string {
		gate := func(id string) string {
			base := `{"precision":0.7,"recall":0.5,"min_gold":2`
			hash, ok := hashes[id]
			if !ok {
				return base + `}`
			}
			model, role := "winnow", "decide.local"
			if id == "run.long" {
				model, role = RuleModel, ""
			}
			return base + fmt.Sprintf(`,"measurements":[{"definition_hash":%q,"model":%q,"resource_role":%q,`+
				`"precision":0.9,"recall":0.9,"gold":10,"positives":4,"threshold":0.5,"temperature":1,"ece":0.05,"brier":0.1,`+
				`"split":"holdout","measured_at":"2026-11-20T10:00:00Z"}]}`, hash, model, role)
		}
		decision := func(id, statement string) string {
			return fmt.Sprintf(`{"id":%q,"group":"operator","scope":"message","method":"decision","population":"human_operator",
  "question":{"type":"noul","instructions":%q,"true_description":"yes","false_description":"no"},
  "evidence":[{"source":"operator_messages","field":"text"}],"gate":%s,
  "search":{"facet":"label:%s","min_probability":0.6},"compatibility":{"change":"breaking"}}`, id, statement, gate(id), id)
		}
		rule := func(id string, scope string, method, expr, extra string) string {
			return fmt.Sprintf(`{"id":%q,"group":"operator","scope":%q,"method":%q,"rule":%q,%s
  "evidence":[{"source":"workload","field":"kind"}],"gate":%s,
  "search":{"facet":"label:%s","min_probability":0.6},"compatibility":{"change":"breaking"}}`, id, scope, method, expr, extra, gate(id), id)
		}
		return `[` + strings.Join([]string{
			decision("operator.pleased", "is the operator pleased?"),
			decision("operator.frustrated", "is the operator frustrated?"),
			rule("operator.valence", "message", "composite", "labels['operator.pleased'] - labels['operator.frustrated']", `"population":"human_operator",`),
			rule("operator.frustrated_run", "run", "composite", "message_labels['operator.frustrated'].max", `"population":"human_operator","require":"human_message_count > 0",`),
			rule("run.long", "run", "composite", "duration_min > 20.0 ? 1.0 : 0.0", ``),
		}, ",") + `]`
	}
	pack, err := ParsePack([]byte(packWithLabels(build(nil))))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := pack.Compile()
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, id := range proven {
		hashes[id] = compiled.LabelHashes[id]
	}
	return build(hashes)
}

// moodAnswers answers pleased and frustrated from the message text, so a test
// controls each message's observation.
type moodAnswers struct {
	mu         sync.Mutex
	frustrated map[string]float64 // substring of the message -> probability
}

func (m *moodAnswers) answer(request *inferencev1.DecideBatchRequest, item int, question *inferencev1.DecisionQuestion) *inferencev1.DecisionAnswer {
	state := request.Items[item].GetStateJson()
	probability := 0.1
	m.mu.Lock()
	defer m.mu.Unlock()
	switch question.GetId() {
	case "operator.pleased":
		if strings.Contains(state, "thanks") {
			probability = 0.9
		}
	case "operator.frustrated":
		// Map order is random, so a message matching several fragments takes
		// the strongest, never whichever the iteration reached last.
		for text, value := range m.frustrated {
			if strings.Contains(state, text) && value > probability {
				probability = value
			}
		}
	}
	return &inferencev1.DecisionAnswer{QuestionId: question.GetId(), Type: question.GetType(), Noul: &probability}
}

func newMoodAnswers() *moodAnswers {
	return &moodAnswers{frustrated: map[string]float64{"failing": 0.9, "wrong": 0.9}}
}

// tickingClock advances a second per reading, so a rewritten row is visible.
func tickingClock() func() time.Time {
	var mu sync.Mutex
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(time.Second)
		return now
	}
}

func moodConversation() RunFacts {
	facts := conversation("chat", "interactive", "this keeps failing", "no, that is wrong", "ok that works, thanks")
	facts.Timings.DurationMinutes = 45
	return facts
}

func findLabel(t *testing.T, labels []AnnotationView, id string) AnnotationView {
	t.Helper()
	for _, label := range labels {
		if label.ID == id {
			return label
		}
	}
	t.Fatalf("label %s not in %+v", id, labels)
	return AnnotationView{}
}

func numberOf(t *testing.T, view AnnotationView) float64 {
	t.Helper()
	if view.State != StateObserved || view.Value.Number == nil {
		t.Fatalf("%s = %+v, want an observed number", view.ID, view)
	}
	return *view.Value.Number
}

func TestAMessageCompositeIsComputedFromThatMessagesOwnLabels(t *testing.T) {
	answers := newMoodAnswers()
	service, _ := testService(t, moodLabels(t), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
	ctx := context.Background()
	view, err := service.Annotate(ctx, "chat", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Messages) != 3 {
		t.Fatalf("messages = %d, want 3", len(view.Messages))
	}
	want := []float64{-0.8, -0.8, 0.8}
	for index, message := range view.Messages {
		got := numberOf(t, findLabel(t, message.Labels, "operator.valence"))
		if math.Abs(got-want[index]) > 1e-9 {
			t.Fatalf("message %d valence = %v, want %v (pleased - frustrated of that message)", index, got, want[index])
		}
	}
	// The composite is stored per message, so it is a series like any message label.
	series, err := service.Series(ctx, "operator.valence", "7d", "day")
	if err != nil {
		t.Fatal(err)
	}
	if series.Messages != 3 || len(series.Points) != 1 || series.Points[0].Mean == nil || math.Abs(*series.Points[0].Mean-(-0.8-0.8+0.8)/3) > 1e-9 {
		t.Fatalf("series = %+v, want the mean valence of the three messages", series)
	}
	if series.Points[0].FrictionPositives != 0 {
		t.Fatalf("a valence composite has no friction direction: %+v", series.Points[0])
	}
}

func TestARunCompositeReadsItsMessagesLabels(t *testing.T) {
	answers := newMoodAnswers()
	service, _ := testService(t, moodLabels(t), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
	view, err := service.Annotate(context.Background(), "chat", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := numberOf(t, findLabel(t, view.Labels, "operator.frustrated_run")); math.Abs(got-0.9) > 1e-9 {
		t.Fatalf("run composite = %v, want the max frustration of its messages, 0.9", got)
	}
}

func TestACompositeIsRewrittenOnlyWhenItsInputsChange(t *testing.T) {
	answers := newMoodAnswers()
	service, store := testService(t, moodLabels(t), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
	ctx := context.Background()
	runComposite := func() (float64, time.Time) {
		compiled, _ := service.pack.Label("operator.frustrated_run")
		row, err := store.Current(ctx, ScopeRun, "chat", "operator.frustrated_run", compiled.hash)
		if err != nil || row == nil {
			t.Fatalf("run composite row = %v (%v)", row, err)
		}
		return *row.Value.Number, row.ObservedAt
	}
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	value, wroteAt := runComposite()

	// Nothing changed: the stored composite stands.
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, again := runComposite(); !again.Equal(wroteAt) {
		t.Fatalf("an unchanged composite was rewritten: %v -> %v", wroteAt, again)
	}

	// A new message is answered: the run composite reads it, so it is recomputed.
	facts := moodConversation()
	extra := MessageFact{EventID: "chat-m4", At: messageBase.Add(3 * time.Hour), Text: "this is awful and wrong"}
	facts.OperatorHistory = append(facts.OperatorHistory, extra)
	facts.OperatorMessages = facts.OperatorHistory
	service.features.(*FixtureSource).Set(facts)
	answers.frustrated["awful"] = 0.99
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	grown, grownAt := runComposite()
	if math.Abs(grown-0.99) > 1e-9 || !grownAt.After(wroteAt) {
		t.Fatalf("after a new message the run composite = %v written %v, want 0.99 recomputed (was %v at %v)", grown, grownAt, value, wroteAt)
	}

	// A message's answer changes: the composite follows it.
	answers.frustrated["awful"] = 0.5
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if lowered, _ := runComposite(); math.Abs(lowered-0.9) > 1e-9 {
		t.Fatalf("after a message answer changed the run composite = %v, want 0.9", lowered)
	}
}

func TestAskingForACompositeAsksForItsInputs(t *testing.T) {
	answers := newMoodAnswers()
	service, _ := testService(t, moodLabels(t), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
	view, err := service.Annotate(context.Background(), "chat", AnnotateOptions{Labels: []string{"operator.valence"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Messages) != 3 {
		t.Fatalf("messages = %d, want the messages answered for the valence composite", len(view.Messages))
	}
	for _, message := range view.Messages {
		findLabel(t, message.Labels, "operator.pleased")
		findLabel(t, message.Labels, "operator.frustrated")
		numberOf(t, findLabel(t, message.Labels, "operator.valence"))
	}
	for _, label := range view.Labels {
		if label.ID == "run.long" || label.ID == "operator.frustrated_run" {
			t.Fatalf("a composite that was not asked for ran: %+v", label)
		}
	}
}

func TestACompositeOverLabelsIsProvenOnlyWhenEveryInputIsProven(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name              string
		proven            []string
		valence, run      string
		reasonMentions    string
		provenValenceRows int
	}{
		{"both inputs proven", []string{"operator.pleased", "operator.frustrated"}, GateProven, GateProven, "", 3},
		{"one input unproven", []string{"operator.pleased"}, GateUnproven, GateUnproven, "operator.frustrated", 0},
		{"no input proven", nil, GateUnproven, GateUnproven, "operator.", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answers := newMoodAnswers()
			service, _ := testService(t, moodLabels(t, tc.proven...), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
			view, err := service.Annotate(ctx, "chat", AnnotateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			valence := findLabel(t, view.Messages[0].Labels, "operator.valence")
			if valence.GateState != tc.valence || valence.Proven != (tc.valence == GateProven) {
				t.Fatalf("valence gate = %s proven=%v (%s), want %s", valence.GateState, valence.Proven, valence.Gate.Reason, tc.valence)
			}
			if len(valence.Gate.Inputs) != 2 {
				t.Fatalf("valence carries inputs %+v, want its two input proof keys", valence.Gate.Inputs)
			}
			for _, input := range valence.Gate.Inputs {
				if input.Model != "winnow" || input.ResourceRole != "decide.local" || input.DefinitionHash == "" {
					t.Fatalf("input key = %+v, want the model, resource role and definition of the observation read", input)
				}
			}
			if tc.reasonMentions != "" && !strings.Contains(valence.Gate.Reason, tc.reasonMentions) {
				t.Fatalf("reason = %q, want it to name %q", valence.Gate.Reason, tc.reasonMentions)
			}
			run := findLabel(t, view.Labels, "operator.frustrated_run")
			runWant := GateProven
			if !contains(tc.proven, "operator.frustrated") {
				runWant = GateUnproven
			}
			if run.GateState != runWant {
				t.Fatalf("run composite gate = %s (%s), want %s: it reads only frustrated", run.GateState, run.Gate.Reason, runWant)
			}
			coverage, err := service.Coverage(ctx, "7d")
			if err != nil {
				t.Fatal(err)
			}
			for _, label := range coverage.Labels {
				if label.ID == "operator.valence" {
					if label.ProvenObserved != tc.provenValenceRows || label.Gate.State != GateDerived {
						t.Fatalf("valence coverage proven=%d gate=%s, want %d / derived", label.ProvenObserved, label.Gate.State, tc.provenValenceRows)
					}
				}
			}
			labels, err := service.Labels(ctx, "7d")
			if err != nil {
				t.Fatal(err)
			}
			for _, label := range labels {
				if label.ID == "operator.valence" && label.GateState != GateDerived {
					t.Fatalf("valence label state = %s, want derived", label.GateState)
				}
			}
		})
	}
}

func TestACompositeOverFactsOnlyKeepsTheRuleHoldoutPath(t *testing.T) {
	answers := newMoodAnswers()
	service, _ := testService(t, moodLabels(t), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
	view, err := service.Annotate(context.Background(), "chat", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	long := findLabel(t, view.Labels, "run.long")
	if long.GateState != GateUnmeasured || len(long.Gate.Inputs) != 0 {
		t.Fatalf("a composite over facts = %s inputs %v, want the rule path: unmeasured without a hold-out", long.GateState, long.Gate.Inputs)
	}

	// With a hold-out for the rule model it is proven like any rule.
	service, _ = testService(t, moodLabels(t, "run.long"), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
	view, err = service.Annotate(context.Background(), "chat", AnnotateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if long := findLabel(t, view.Labels, "run.long"); long.GateState != GateProven || !long.Proven {
		t.Fatalf("a rule composite with a rule hold-out = %s (%s), want proven", long.GateState, long.Gate.Reason)
	}
}

func TestCompositePacksAreCheckedBeforeTheyRun(t *testing.T) {
	label := func(id, scope, method, rule string) string {
		return fmt.Sprintf(`{"id":%q,"group":"g","scope":%q,"method":%q,"rule":%q,
  "evidence":[{"source":"workload","field":"kind"}],"gate":{"precision":0.7,"recall":0.5,"min_gold":2},
  "search":{"facet":"label:%s","min_probability":0.6},"compatibility":{"change":"breaking"}}`, id, scope, method, rule, id)
	}
	base := label("a.run", "run", "deterministic", "true")
	cases := []struct {
		name, labels, want string
	}{
		{"an unknown input", label("c", "run", "composite", "labels['nope']"), "unknown label"},
		{"a computed key", base + "," + label("c", "run", "composite", "labels['a.run'] + labels[run_id]"), "literal key"},
		{"a map read whole", base + "," + label("c", "run", "composite", "size(labels) > 0 ? 1.0 : 0.0"), "literal key"},
		{"itself", label("c", "run", "composite", "labels['c']"), "cannot read itself"},
		{"another scope through labels", label("m", "message", "deterministic", "true") + "," + label("c", "run", "composite", "labels['m']"), "a message label"},
		{"message_labels on a message composite", label("m", "message", "deterministic", "true") + "," + label("c", "message", "composite", "message_labels['m'].max"), "run composite"},
		{"message_labels of a run label", base + "," + label("c", "run", "composite", "message_labels['a.run'].max"), "message-scope label"},
		{"a cycle", label("x", "run", "composite", "labels['y']") + "," + label("y", "run", "composite", "labels['x']"), "cycle"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pack, err := ParsePack([]byte(packWithLabels("[" + tc.labels + "]")))
			if err == nil {
				_, err = pack.Compile()
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
	// A derived composite takes its proof from its inputs, so declared gold is refused.
	withGold := strings.Replace(label("c", "run", "composite", "labels['a.run']"), `"min_gold":2`, `"min_gold":2,"measurements":[{"definition_hash":"`+strings.Repeat("a", 64)+`","model":"rule","precision":1,"recall":1,"gold":9,"positives":3,"threshold":0.5,"temperature":1,"split":"holdout"}]`, 1)
	pack, err := ParsePack([]byte(packWithLabels("[" + base + "," + withGold + "]")))
	if err == nil {
		_, err = pack.Compile()
	}
	if err == nil || !strings.Contains(err.Error(), "cannot declare gate measurements") {
		t.Fatalf("a derived composite with its own measurements: %v", err)
	}
}

func TestAMessageLabelAcceptsAComposite(t *testing.T) {
	pack, err := ParsePack([]byte(packWithLabels(moodLabels(t))))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := pack.Compile()
	if err != nil {
		t.Fatal(err)
	}
	valence, _ := compiled.Label("operator.valence")
	if !valence.derived() || strings.Join(valence.refs.Labels, ",") != "operator.frustrated,operator.pleased" {
		t.Fatalf("valence refs = %+v", valence.refs)
	}
	run, _ := compiled.Label("operator.frustrated_run")
	if strings.Join(run.refs.Messages, ",") != "operator.frustrated" || len(run.refs.Labels) != 0 {
		t.Fatalf("run composite refs = %+v", run.refs)
	}
	order := compiled.compositeOrder(ScopeMessage)
	if len(order) != 1 || order[0] != "operator.valence" {
		t.Fatalf("message composite order = %v", order)
	}
}
