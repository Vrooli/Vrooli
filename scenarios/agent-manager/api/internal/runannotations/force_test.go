package runannotations

import (
	"context"
	"errors"
	"testing"
)

// askedBy counts the questions the model has been asked, by label.
func askedBy(decision *fakeDecision) map[string]int {
	asked := map[string]int{}
	for _, call := range decision.calls {
		for range call.GetItems() {
			for _, question := range call.GetQuestions() {
				asked[question.GetId()]++
			}
		}
	}
	return asked
}

func TestForcingACompositeNeverAsksItsMessagesAgain(t *testing.T) {
	decision := &fakeDecision{answer: newMoodAnswers().answer}
	service, _ := testService(t, moodLabels(t), []RunFacts{moodConversation()}, decision, Options{Clock: tickingClock()})
	ctx := context.Background()
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	before := askedBy(decision)
	if before["operator.dissatisfied"] == 0 || before["operator.pleased"] == 0 {
		t.Fatalf("setup asked %v, want the message labels answered once", before)
	}
	view, err := service.Annotate(ctx, "chat", AnnotateOptions{
		Labels: []string{"operator.dissatisfied_run"}, ForceLabels: []string{"operator.dissatisfied_run"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if after := askedBy(decision); after["operator.dissatisfied"] != before["operator.dissatisfied"] || after["operator.pleased"] != before["operator.pleased"] {
		t.Fatalf("recomputing a composite asked the model again: %v -> %v", before, after)
	}
	findLabel(t, view.Labels, "operator.dissatisfied_run")
}

func TestForcingALabelAsksOnlyThatLabelAgain(t *testing.T) {
	decision := &fakeDecision{answer: newMoodAnswers().answer}
	service, _ := testService(t, moodLabels(t), []RunFacts{moodConversation()}, decision, Options{Clock: tickingClock()})
	ctx := context.Background()
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{}); err != nil {
		t.Fatal(err)
	}
	before := askedBy(decision)
	if _, err := service.Annotate(ctx, "chat", AnnotateOptions{ForceLabels: []string{"operator.dissatisfied"}}); err != nil {
		t.Fatal(err)
	}
	after := askedBy(decision)
	if after["operator.dissatisfied"] <= before["operator.dissatisfied"] {
		t.Fatalf("a named label was not asked again: %v -> %v", before, after)
	}
	if after["operator.pleased"] != before["operator.pleased"] {
		t.Fatalf("a label that was not named was asked again: %v -> %v", before, after)
	}
}

func TestAForceThatNamesNoEvaluatedLabelIsRefused(t *testing.T) {
	service, _ := testService(t, moodLabels(t), []RunFacts{moodConversation()}, &fakeDecision{answer: newMoodAnswers().answer}, Options{Clock: tickingClock()})
	ctx := context.Background()
	for name, options := range map[string]AnnotateOptions{
		"an unknown label":                 {ForceLabels: []string{"operator.nope"}},
		"a label outside the labels asked": {Labels: []string{"operator.pleased"}, ForceLabels: []string{"operator.dissatisfied"}},
	} {
		if _, err := service.Annotate(ctx, "chat", options); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
		if _, err := service.Evaluate(ctx, EvaluateOptions{RunIDs: []string{"chat"}, Labels: options.Labels, ForceLabels: options.ForceLabels}); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("%s (evaluate): err = %v, want ErrInvalidRequest", name, err)
		}
	}
}
