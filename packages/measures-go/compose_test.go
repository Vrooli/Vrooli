package measures

import (
	"strings"
	"testing"

	aisearch "github.com/vrooli/aisearch-go"
)

func TestMeasureComposer_EmbedsQuestionsThenIntent(t *testing.T) {
	chunk := aisearch.Chunk{
		Body: `{"name":"backlog.completed"}`, // payload — must NOT be the embed text
		Meta: map[string]any{
			MetaQuestions: []string{
				"how many backlog items did we complete this week",
				"backlog items closed last month",
			},
			MetaIntent: "How many backlog items were completed in a time window.",
		},
	}
	got := MeasureComposer{}.Compose(chunk)
	want := "how many backlog items did we complete this week\n" +
		"backlog items closed last month\n" +
		"How many backlog items were completed in a time window."
	if got != want {
		t.Fatalf("compose =\n%q\nwant\n%q", got, want)
	}
	if strings.Contains(got, "backlog.completed") {
		t.Fatal("payload body leaked into embedded text")
	}
}

func TestMeasureComposer_AcceptsAnySliceAndJoinedString(t *testing.T) {
	// []any (JSON-decoded) and a newline-joined string are both accepted.
	fromAny := MeasureComposer{}.Compose(aisearch.Chunk{
		Meta: map[string]any{MetaQuestions: []any{"q1", "q2"}, MetaIntent: "i"},
	})
	if fromAny != "q1\nq2\ni" {
		t.Fatalf("[]any compose = %q", fromAny)
	}
	fromStr := MeasureComposer{}.Compose(aisearch.Chunk{
		Meta: map[string]any{MetaQuestions: "q1\nq2", MetaIntent: "i"},
	})
	if fromStr != "q1\nq2\ni" {
		t.Fatalf("string compose = %q", fromStr)
	}
}

func TestMeasureComposer_FallsBackToBody(t *testing.T) {
	got := MeasureComposer{}.Compose(aisearch.Chunk{Body: "fallback body"})
	if got != "fallback body" {
		t.Fatalf("expected body fallback, got %q", got)
	}
}

func TestMeasureComposer_SatisfiesInterface(t *testing.T) {
	var _ aisearch.EmbeddingTextComposer = NewMeasureComposer()
}
