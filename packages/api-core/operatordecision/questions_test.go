package operatordecision

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func groupsDecision() Facts {
	return Facts{
		Question: "Back up which groups?",
		Options:  []Option{{Key: "chosen", Label: "Back up the groups I switch on"}, {Key: "none", Label: "Back up none"}},
		Questions: []Question{{
			ID: "groups", Type: QuestionMultiChoice, Prompt: "Which groups?",
			Choices:     []Choice{{Key: "ai-tools", Label: "AI tools"}, {Key: "keys", Label: "Keys and sign-ins"}},
			Recommended: []string{"ai-tools", "keys"}, RecommendationReason: "Neither exists anywhere else.",
		}},
	}
}

func jsonValue(t *testing.T, text string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

// A decision event carries its form in the shape the Hub's intake reads, so a
// page's Groups switches fill a real multi_choice question.
func TestADecisionCarriesItsFormInTheHubsShape(t *testing.T) {
	facts := groupsDecision()
	if err := facts.Validate(); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(facts.Payload()["questions"])
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"id":"groups","type":"multi_choice","prompt":"Which groups?",
		"choices":[{"key":"ai-tools","label":"AI tools"},{"key":"keys","label":"Keys and sign-ins"}],
		"recommended":{"question_id":"groups","selected":["ai-tools","keys"]},
		"recommendation_reason":"Neither exists anywhere else."}]`
	if got := jsonValue(t, string(encoded)); !reflect.DeepEqual(got, jsonValue(t, want)) {
		t.Fatalf("questions payload = %s", encoded)
	}
	if _, has := (Facts{Question: "Q?", Options: facts.Options}).Payload()["questions"]; has {
		t.Fatal("a decision with no form must emit the bytes it always did")
	}
}

func TestAFormTheHubWouldDropIsRefusedBeforeItIsSent(t *testing.T) {
	for name, change := range map[string]func(*Question){
		"unknown type":          func(q *Question) { q.Type = "variant" },
		"reserved id":           func(q *Question) { q.ID = "answer" },
		"one choice":            func(q *Question) { q.Choices = q.Choices[:1] },
		"repeated choice":       func(q *Question) { q.Choices[1].Key = q.Choices[0].Key },
		"unknown recommended":   func(q *Question) { q.Recommended = []string{"vault"} },
		"single with two picks": func(q *Question) { q.Type = QuestionSingleChoice },
		"no prompt":             func(q *Question) { q.Prompt = " " },
	} {
		facts := groupsDecision()
		change(&facts.Questions[0])
		if err := facts.Validate(); err == nil {
			t.Errorf("%s: Validate accepted a form the Hub would drop", name)
		}
	}
}

func TestTheResolutionCarriesTheOperatorsFormAnswers(t *testing.T) {
	var resolution DecisionResolution
	if err := json.Unmarshal([]byte(`{"ask_id":"a1","state":"answered","answer":"chosen",
		"responses":[{"question_id":"groups","selected":["keys"],"text":"","sections":[]}],
		"answer_summary":"Answer: Back up the groups I switch on (chosen)"}`), &resolution); err != nil {
		t.Fatal(err)
	}
	response, ok := resolution.Response("groups")
	if !ok || !reflect.DeepEqual(response.Selected, []string{"keys"}) {
		t.Fatalf("groups response = %+v, %v", response, ok)
	}
	if _, ok := resolution.Response("other"); ok {
		t.Fatal("an unanswered question has no response")
	}
	if !strings.Contains(resolution.AnswerSummary, "chosen") {
		t.Fatal("the answer summary is carried")
	}
}

// The event bus carries the payload as a protobuf Struct. A form with a
// recommendation must survive that conversion, which takes only []any lists.
func TestAFormSurvivesTheEventBusPayloadConversion(t *testing.T) {
	facts := groupsDecision()
	event := BuildEventTyped("data-backup-manager", "data-backup-manager.sensitive.decision_requested.v1", time.Now(), facts)
	payload, err := structpb.NewStruct(event.Payload)
	if err != nil {
		t.Fatalf("the event bus refuses the payload: %v", err)
	}
	encoded, err := protojson.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Questions []struct {
			ID          string `json:"id"`
			Recommended struct {
				Selected []string `json:"selected"`
			} `json:"recommended"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Questions) != 1 || wire.Questions[0].ID != "groups" || !reflect.DeepEqual(wire.Questions[0].Recommended.Selected, []string{"ai-tools", "keys"}) {
		t.Fatalf("questions on the wire = %s", encoded)
	}
}
