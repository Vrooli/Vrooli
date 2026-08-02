package runsignal

import (
	"encoding/json"
	"os"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestSelfReportFrozenCorpus(t *testing.T) {
	body, err := os.ReadFile("../runreport/testdata/self-report-corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus []struct{ Role, Content string }
	if err := json.Unmarshal(body, &corpus); err != nil {
		t.Fatal(err)
	}
	events := make([]*domain.RunEvent, 0, len(corpus))
	for _, item := range corpus {
		events = append(events, &domain.RunEvent{ID: uuid.New(), Data: &domain.MessageEventData{Role: item.Role, Content: item.Content}})
	}
	spans := DeriveSelfReportSpans(events)
	if len(spans) != 5 {
		t.Fatalf("span count = %d, want 5: %+v", len(spans), spans)
	}
	for _, span := range spans {
		if span.ClassifierVersion != SelfReportClassifierVersion || span.Text == "" {
			t.Fatalf("invalid span: %+v", span)
		}
	}
}

func TestDeriveSelfReportSpansAssistantOnly(t *testing.T) {
	events := []*domain.RunEvent{{ID: uuid.New(), Data: &domain.MessageEventData{Role: "assistant", Content: "I am blocked and need permission; instead I will retry."}}, {ID: uuid.New(), Data: &domain.MessageEventData{Role: "tool", Content: "I am blocked"}}, {ID: uuid.New(), Data: &domain.MessageEventData{Role: "assistant", Content: "I can find it"}}}
	got := DeriveSelfReportSpans(events)
	if len(got) != 3 {
		t.Fatalf("span count=%d, want 3: %+v", len(got), got)
	}
	for _, span := range got {
		if span.ClassifierVersion != SelfReportClassifierVersion || span.RuleID == "" || span.EndOffset <= span.StartOffset {
			t.Fatalf("invalid span: %+v", span)
		}
	}
}
