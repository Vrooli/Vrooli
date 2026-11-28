package piimask

import (
	"context"
	"errors"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// fakeDetector is a deterministic in-memory detector. It returns the spans
// registered for an exact input text, errors for texts registered to fail, and
// a clean (empty) result otherwise — so a test can assert masking, cleanliness,
// and fail-closed without a live gateway or detector resource.
type fakeDetector struct {
	spans    map[string][]Span
	failOn   map[string]struct{}
	failAll  bool
	callLog  []string
	modelOK  bool
	modelErr error
}

func newFakeDetector() *fakeDetector {
	return &fakeDetector{spans: map[string][]Span{}, failOn: map[string]struct{}{}, modelOK: true}
}

func (f *fakeDetector) Detect(_ context.Context, text string) (DetectResult, error) {
	f.callLog = append(f.callLog, text)
	if f.modelErr != nil {
		return DetectResult{}, f.modelErr
	}
	if f.failAll {
		return DetectResult{}, errors.New("detector unavailable")
	}
	if _, fail := f.failOn[text]; fail {
		return DetectResult{}, errors.New("detector failed for this text")
	}
	return DetectResult{Spans: f.spans[text], ModelAvailable: f.modelOK}, nil
}

// emailSpan builds a span over the first occurrence of email in text (byte
// offsets), matching the pii-detector regex contract.
func emailSpan(text, email string) Span {
	start := strings.Index(text, email)
	return Span{Category: "EMAIL_ADDRESS", Start: start, End: start + len(email), Text: email}
}

func TestMaskTextReplacesSpansWithCategoryPlaceholder(t *testing.T) {
	const text = "reach me at bob@example.com please"
	det := newFakeDetector()
	det.spans[text] = []Span{emailSpan(text, "bob@example.com")}
	masker := New(det)

	masked, failed := masker.MaskText(context.Background(), text)
	if failed {
		t.Fatalf("clean detection must not fail closed")
	}
	if strings.Contains(masked, "bob@example.com") {
		t.Fatalf("masked text still contains the PII: %q", masked)
	}
	if !strings.Contains(masked, Placeholder("EMAIL_ADDRESS")) {
		t.Fatalf("masked text missing the category placeholder: %q", masked)
	}
	if want := "reach me at " + Placeholder("EMAIL_ADDRESS") + " please"; masked != want {
		t.Fatalf("unexpected mask: got %q want %q", masked, want)
	}
	if got := masker.Snapshot(); got.SpansMasked != 1 || got.EventsScanned != 1 || got.FailClosed != 0 {
		t.Fatalf("unexpected counters: %+v", got)
	}
}

func TestMaskTextLeavesCleanTextUntouchedWithNoCall(t *testing.T) {
	det := newFakeDetector()
	masker := New(det)

	// Whitespace/empty never calls the detector at all (perf: no wasted calls).
	if masked, failed := masker.MaskText(context.Background(), "   "); masked != "   " || failed {
		t.Fatalf("whitespace must pass through untouched, got %q failed=%v", masked, failed)
	}
	if len(det.callLog) != 0 {
		t.Fatalf("empty text must not call the detector, calls=%v", det.callLog)
	}

	// Non-empty but clean text calls the detector and passes through unchanged.
	const clean = "the build is green and ready to ship"
	masked, failed := masker.MaskText(context.Background(), clean)
	if masked != clean || failed {
		t.Fatalf("clean text must be unchanged, got %q failed=%v", masked, failed)
	}
	if got := masker.Snapshot(); got.SpansMasked != 0 || got.FailClosed != 0 || got.EventsScanned != 1 {
		t.Fatalf("unexpected counters: %+v", got)
	}
}

func TestMaskTextFailsClosedOnDetectorError(t *testing.T) {
	const text = "my ssn is 123-45-6789"
	det := newFakeDetector()
	det.failOn[text] = struct{}{}
	masker := New(det)

	masked, failed := masker.MaskText(context.Background(), text)
	if !failed {
		t.Fatalf("a detector error must fail closed")
	}
	if masked != UnavailableSentinel {
		t.Fatalf("fail-closed must replace content with the sentinel, got %q", masked)
	}
	if strings.Contains(masked, "123-45-6789") {
		t.Fatalf("fail-closed must never leak the unscanned PII: %q", masked)
	}
	if got := masker.Snapshot(); got.FailClosed != 1 {
		t.Fatalf("fail-closed must be counted, got %+v", got)
	}
}

func TestNilDetectorFailsClosed(t *testing.T) {
	masker := New(nil)
	masked, failed := masker.MaskText(context.Background(), "contains pii")
	if !failed || masked != UnavailableSentinel {
		t.Fatalf("a nil detector must fail closed, got %q failed=%v", masked, failed)
	}
	if got := masker.Snapshot().FailClosed; got != 1 {
		t.Fatalf("fail-closed must be counted, got %d", got)
	}
}

func TestMaskEventMessage(t *testing.T) {
	const content = "ping me at jane@corp.io"
	det := newFakeDetector()
	det.spans[content] = []Span{emailSpan(content, "jane@corp.io")}
	masker := New(det)

	event := domain.NewMessageEvent(uuid.New(), "assistant", content)
	masked, failed := masker.MaskEvent(context.Background(), event)
	if failed {
		t.Fatalf("clean masking must not fail closed")
	}
	// The input event must never be mutated (parser owns it).
	if orig := event.Data.(*domain.MessageEventData).Content; orig != content {
		t.Fatalf("source event was mutated: %q", orig)
	}
	got := masked.Data.(*domain.MessageEventData).Content
	if strings.Contains(got, "jane@corp.io") {
		t.Fatalf("message content still contains PII: %q", got)
	}
}

func TestMaskEventToolCallMasksStringLeaves(t *testing.T) {
	const arg = "email bob@x.com now"
	det := newFakeDetector()
	det.spans[arg] = []Span{emailSpan(arg, "bob@x.com")}
	masker := New(det)

	event := domain.NewToolCallEvent(uuid.New(), "bash", "call-1", map[string]any{
		"command": arg,
		"count":   float64(3), // non-string leaves pass through untouched
		"nested":  map[string]any{"note": arg},
	})
	masked, failed := masker.MaskEvent(context.Background(), event)
	if failed {
		t.Fatalf("clean masking must not fail closed")
	}
	data := masked.Data.(*domain.ToolCallEventData)
	if got := data.Input["command"].(string); strings.Contains(got, "bob@x.com") {
		t.Fatalf("tool-call command still contains PII: %q", got)
	}
	if got := data.Input["nested"].(map[string]any)["note"].(string); strings.Contains(got, "bob@x.com") {
		t.Fatalf("nested tool-call string still contains PII: %q", got)
	}
	if data.Input["count"].(float64) != 3 {
		t.Fatalf("non-string leaves must be untouched, got %v", data.Input["count"])
	}
}

func TestMaskEventToolResultFailsClosed(t *testing.T) {
	const output = "secret token is in here"
	det := newFakeDetector()
	det.failOn[output] = struct{}{}
	masker := New(det)

	event := domain.NewToolResultEvent(uuid.New(), "bash", "call-1", output, nil)
	masked, failed := masker.MaskEvent(context.Background(), event)
	if !failed {
		t.Fatalf("a detector error must fail closed")
	}
	if got := masked.Data.(*domain.ToolResultEventData).Output; got != UnavailableSentinel {
		t.Fatalf("fail-closed tool output must be the sentinel, got %q", got)
	}
}

func TestApplySpansRightToLeftAndSkipsOutOfRange(t *testing.T) {
	const text = "a bob@x.com b jane@y.io c"
	spans := []Span{
		emailSpan(text, "bob@x.com"),
		emailSpan(text, "jane@y.io"),
		{Category: "EMAIL", Start: -1, End: 5},            // invalid: negative start
		{Category: "EMAIL", Start: 2, End: len(text) + 9}, // invalid: end past len
	}
	masked, count := applySpans(text, spans)
	if count != 2 {
		t.Fatalf("expected exactly 2 valid spans applied, got %d (%q)", count, masked)
	}
	if strings.Contains(masked, "bob@x.com") || strings.Contains(masked, "jane@y.io") {
		t.Fatalf("masked text still contains PII: %q", masked)
	}
}

func TestDecodeDetectResult(t *testing.T) {
	result, err := DecodeDetectResult(`{"spans":[{"category":"EMAIL_ADDRESS","start":0,"end":5,"text":"a@b.c","confidence":0.9,"source":"regex"}],"model_available":false}`)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Spans) != 1 || result.Spans[0].Category != "EMAIL_ADDRESS" {
		t.Fatalf("unexpected decode: %+v", result)
	}
	// An empty payload is a clean result, not an error.
	if clean, err := DecodeDetectResult("  "); err != nil || len(clean.Spans) != 0 {
		t.Fatalf("empty payload must decode clean, got %+v err=%v", clean, err)
	}
}
