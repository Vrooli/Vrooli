package runsignal

import (
	"encoding/json"
	"os"
	"strings"
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

// Lowercasing changes the byte length of some text (İ, the Kelvin sign) and
// cannot represent invalid UTF-8 faithfully; offsets must still index the
// original message and slice exactly the matched phrase.
func TestSelfReportOffsetsIndexTheOriginalText(t *testing.T) {
	for _, prefix := range []string{"İİİ ", "KK ", "\xff\xfe ", ""} {
		content := prefix + "I CANNOT FIND the file."
		events := []*domain.RunEvent{{ID: uuid.New(), Data: &domain.MessageEventData{Role: "assistant", Content: content}}}
		spans := DeriveSelfReportSpans(events)
		if len(spans) == 0 {
			t.Fatalf("prefix %q: no span", prefix)
		}
		for _, span := range spans {
			if got := content[span.StartOffset:span.EndOffset]; !strings.EqualFold(got, "CANNOT FIND") || span.StartOffset != len(prefix)+2 {
				t.Fatalf("prefix %q: span %+v slices %q", prefix, span, got)
			}
		}
	}
}

// A detector reports at most selfReportSpanLimit spans and marks the last one
// capped whenever anything is left unreported: more matches of the same
// phrase, or a match of a later phrase when the cap lands on a phrase
// boundary. Exactly the limit with nothing left is not capped.
func TestSelfReportSpanCappedWhenAnyMatchIsLeft(t *testing.T) {
	var detector SelfReportDetector
	for _, d := range SelfReportDetectors() {
		if d.Identifier() == "cannot-find" {
			detector = d
		}
	}
	detect := func(content string) []SelfReportSpan {
		return detector.Detect(&domain.RunEvent{ID: uuid.New(), Data: &domain.MessageEventData{Role: "assistant", Content: content}})
	}
	eight := strings.Repeat("cannot find. ", selfReportSpanLimit)
	for content, capped := range map[string]bool{
		eight:                          false,
		eight + "cannot find.":         true,
		eight + "could not locate it.": true, // the cap lands after the first phrase; a later phrase still matches
	} {
		spans := detect(content)
		if len(spans) != selfReportSpanLimit || spans[len(spans)-1].SpanCapped != capped {
			t.Errorf("%q: %d spans, last capped %v; want %d, capped %v", content, len(spans), spans[len(spans)-1].SpanCapped, selfReportSpanLimit, capped)
		}
	}
}

// The rule pack is resolved once per call, not per event: events that match
// no rule cost no allocations however many there are.
func TestDeriveSelfReportSpansResolvesDetectorsOnce(t *testing.T) {
	event := domain.NewProviderMessageEvent(uuid.New(), "user", "plain operator text", domain.MessageEventData{})
	allocs := func(n int) float64 {
		events := make([]*domain.RunEvent, n)
		for i := range events {
			events[i] = event
		}
		return testing.AllocsPerRun(20, func() { DeriveSelfReportSpans(events) })
	}
	if one, many := allocs(1), allocs(1000); many > one {
		t.Fatalf("allocations grew with events: %v for 1, %v for 1000", one, many)
	}
}
