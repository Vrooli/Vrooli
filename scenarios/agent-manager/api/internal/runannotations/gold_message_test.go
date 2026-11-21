package runannotations

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func evaluatedMoodService(t *testing.T) (*Service, *Store) {
	t.Helper()
	answers := newMoodAnswers()
	service, store := testService(t, moodLabels(t), []RunFacts{moodConversation()}, &fakeDecision{answer: answers.answer}, Options{Clock: tickingClock()})
	if _, err := service.Annotate(context.Background(), "chat", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	return service, store
}

func TestMessageGoldIsScoredBySubjectAndUnobservedRowsAreReportedNotDropped(t *testing.T) {
	service, _ := evaluatedMoodService(t)
	ctx := context.Background()
	rows := []GoldRow{
		{RunID: "chat", ScopeID: "chat-m1", Positive: true},
		{RunID: "chat", ScopeID: "chat-m2", Positive: true},
		{RunID: "chat", ScopeID: "chat-m3", Positive: false},
		{RunID: "chat", ScopeID: "chat-m9", Positive: true}, // never observed under this definition
	}
	result, err := service.Benchmark(ctx, BenchmarkRequest{Label: "operator.frustrated", Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	if result.Labelled != 4 || result.Gold != 3 || result.Unobserved != 1 || result.Unavailable != 0 {
		t.Fatalf("labelled %d gold %d unobserved %d unavailable %d, want 4/3/1/0", result.Labelled, result.Gold, result.Unobserved, result.Unavailable)
	}
	if len(result.UnobservedSubjects) != 1 || result.UnobservedSubjects[0] != "chat-m9" {
		t.Fatalf("unobserved subjects = %v, want the message that has no observation", result.UnobservedSubjects)
	}
	if result.Measurement == nil || result.Measurement.Model != "winnow" || result.Measurement.Gold != 3 || !strings.Contains(result.Note, "no stored answer") {
		t.Fatalf("measurement = %+v note = %q", result.Measurement, result.Note)
	}
	// Two messages of one run are two subjects, not one.
	if result.TP+result.FN != 2 || result.TN+result.FP != 1 {
		t.Fatalf("confusion = %+v, want 2 positives and 1 negative scored", result)
	}

	allUnobserved, err := service.Benchmark(ctx, BenchmarkRequest{Label: "operator.frustrated", Rows: []GoldRow{{RunID: "chat", ScopeID: "chat-m8", Positive: true}}})
	if err != nil || allUnobserved.Gold != 0 || allUnobserved.Unobserved != 1 || allUnobserved.Measurement != nil || !strings.Contains(allUnobserved.Note, "annotate them") {
		t.Fatalf("a gold set with nothing observed = %+v (%v), want it accepted and explained", allUnobserved, err)
	}
}

func TestMessageGoldRowsMustNameTheMessageAndStayDisjoint(t *testing.T) {
	service, _ := evaluatedMoodService(t)
	ctx := context.Background()
	if _, err := service.Benchmark(ctx, BenchmarkRequest{Label: "operator.frustrated", Rows: []GoldRow{{RunID: "chat", Positive: true}}}); !errors.Is(err, ErrInvalidRequest) ||
		!strings.Contains(err.Error(), "scope_id") {
		t.Fatalf("a message gold row with only a run: %v", err)
	}
	tuning := []GoldRow{{RunID: "chat", ScopeID: "chat-m1", Positive: true}, {RunID: "chat", ScopeID: "chat-m3", Positive: false}}
	holdout := []GoldRow{{RunID: "chat", ScopeID: "chat-m2", Positive: true}, {RunID: "chat", ScopeID: "chat-m3", Positive: false}}
	_, err := service.Benchmark(ctx, BenchmarkRequest{Label: "operator.frustrated", Split: SplitHoldout, Rows: holdout, FitRows: tuning})
	if !errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), "chat-m3") {
		t.Fatalf("a hold-out sharing a message with the tuning rows: %v", err)
	}
	// Messages of the same run are different subjects: a disjoint split is accepted.
	disjointHoldout := []GoldRow{{RunID: "chat", ScopeID: "chat-m2", Positive: true}}
	if _, err := service.Benchmark(ctx, BenchmarkRequest{Label: "operator.frustrated", Split: SplitHoldout, Rows: disjointHoldout, FitRows: tuning}); err != nil {
		t.Fatalf("a split by message of one run: %v", err)
	}
	if _, err := service.Benchmark(ctx, BenchmarkRequest{Label: "operator.valence", Rows: tuning}); !errors.Is(err, ErrInvalidRequest) ||
		!strings.Contains(err.Error(), "derived") {
		t.Fatalf("a derived composite has no gold of its own: %v", err)
	}
}

func TestMessageGoldDraftIsBlindAndNamesTheMessage(t *testing.T) {
	service, _ := evaluatedMoodService(t)
	draft, err := service.GoldDraft(context.Background(), "operator.frustrated", "30d", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Runs) != 3 {
		t.Fatalf("draft = %+v, want one packet per observed message", draft.Runs)
	}
	texts := map[string]string{"chat-m1": "this keeps failing", "chat-m2": "that is wrong", "chat-m3": "that works, thanks"}
	for _, packet := range draft.Runs {
		want, ok := texts[packet.ScopeID]
		if !ok || packet.RunID != "chat" || !strings.Contains(packet.Digest, want) {
			t.Fatalf("packet = %+v, want the message %q", packet, want)
		}
		if strings.Contains(packet.Digest, "probab") || strings.Contains(packet.Digest, "0.9") {
			t.Fatalf("a packet leaks the stored answer: %s", packet.Digest)
		}
		if strings.Contains(packet.Digest, "that works, thanks") && strings.Contains(packet.Digest, "this keeps failing") {
			t.Fatalf("a message packet carried two messages: %s", packet.Digest)
		}
	}
}

func TestPinnedEvaluationAnswersOnlyTheNamedMessages(t *testing.T) {
	answers := newMoodAnswers()
	decision := &fakeDecision{answer: answers.answer, resolve: candidateResolver}
	service, store := testService(t, moodLabels(t), []RunFacts{moodConversation()}, decision, Options{Clock: tickingClock()})
	ctx := context.Background()
	if _, err := service.Evaluate(ctx, EvaluateOptions{
		RunIDs: []string{"chat"}, Labels: []string{"operator.frustrated"}, MessageIDs: []string{"chat-m2"},
		PinResourceRole: "decide.local.candidate", TimeBudget: -1,
	}); err != nil {
		t.Fatal(err)
	}
	hash := service.Pack().LabelHashes["operator.frustrated"]
	ids := []string{"chat-m1", "chat-m2", "chat-m3"}
	trial, err := store.Trials().LatestObservations(ctx, ScopeMessage, ids, "operator.frustrated", hash, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(trial) != 1 || trial["chat-m2"].Model != "candidate-model" || trial["chat-m2"].ModelVersion != "decide.local.candidate" {
		t.Fatalf("trial rows = %+v, want only the named message, answered by the pinned candidate", trial)
	}
	if production, err := store.LatestObservations(ctx, ScopeMessage, ids, "operator.frustrated", hash, "", ""); err != nil || len(production) != 0 {
		t.Fatalf("production rows = %+v (%v): a pinned pass wrote production", production, err)
	}
	// The named message is judged by the same benchmark, from the trials.
	result, err := service.Benchmark(ctx, BenchmarkRequest{
		Label: "operator.frustrated", Source: SourcePinned, ResourceRole: "decide.local.candidate",
		Rows: []GoldRow{{RunID: "chat", ScopeID: "chat-m2", Positive: true}, {RunID: "chat", ScopeID: "chat-m1", Positive: true}},
	})
	if err != nil || result.Gold != 1 || result.Unobserved != 1 || result.Model != "candidate-model" {
		t.Fatalf("pinned benchmark = %+v (%v)", result, err)
	}
}
