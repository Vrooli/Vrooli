// Package piimask masks personally-identifiable information in conversation
// content before it is persisted as searchable run_events. It is the W5 engine
// behind the conversationpolicy `redact` action: a run matching a redact rule
// at the import seam has every indexed content field scrubbed of PII here,
// BEFORE it reaches the event store that feeds the lexical and semantic search
// indexes.
//
// Detection is delegated to the shared pii-detector resource through the
// governed ai-gateway DETECT_PII route (see GatewayMasker); this package owns
// only the policy decision of "what to mask" and the fail-closed contract of
// "never index unmasked content the operator asked to redact".
//
// Fail-closed is the load-bearing security property. If the detector or the
// gateway is unavailable or errors, the content is NOT indexed in the clear:
// the maskable fields are replaced with a loud sentinel and a counter is
// incremented, so the PII the operator asked to redact can never silently leak
// into search.
package piimask

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"agent-manager/internal/domain"
)

// UnavailableSentinel replaces content whose PII could not be masked because
// the detector/gateway was unavailable or errored. It is deliberately visible
// (not empty) so an operator auditing a redact-matched conversation sees that
// the content existed and was withheld, rather than silently vanishing.
const UnavailableSentinel = "[REDACTED: PII masking unavailable]"

// Span is one detected PII region within a scanned text, in byte offsets. It is
// the subset of the pii-detector detect contract this package consumes; the
// gateway returns additional fields (confidence, source) that masking ignores.
type Span struct {
	Category string `json:"category"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Text     string `json:"text"`
}

// DetectResult is the pii-detector `gateway detect --json` wire contract as
// returned verbatim in the ai-gateway ExecuteRoute OutputText. ModelAvailable
// reports whether the NER pass ran; the deterministic regex/checksum pass
// always runs, so spans can be non-empty even when the model is unavailable.
type DetectResult struct {
	Spans          []Span `json:"spans"`
	ModelAvailable bool   `json:"model_available"`
	ModelID        string `json:"model_id,omitempty"`
}

// Detector returns PII spans for one text. A nil error with no spans means the
// text is clean. A non-nil error means detection could not be completed and the
// caller MUST fail closed (never index the text unmasked).
type Detector interface {
	Detect(ctx context.Context, text string) (DetectResult, error)
}

// Counters is a point-in-time observability snapshot of the masking engine.
type Counters struct {
	// EventsScanned is how many content-bearing events were routed through
	// detection for a redact-matched import.
	EventsScanned int64 `json:"eventsScanned"`
	// SpansMasked is the total number of PII spans replaced across all events.
	SpansMasked int64 `json:"spansMasked"`
	// FailClosed is how many events were withheld (content replaced with the
	// unavailable sentinel) because detection could not be completed. A non-zero
	// value is a loud signal that the detector/gateway was unhealthy during an
	// import the operator asked to redact.
	FailClosed int64 `json:"failClosed"`
}

// Masker applies a Detector to conversation event content under the redact
// policy. It is safe for concurrent use (the counters are atomic); the Detector
// it wraps must be concurrency-safe if the caller masks events in parallel.
type Masker struct {
	detector Detector

	eventsScanned atomic.Int64
	spansMasked   atomic.Int64
	failClosed    atomic.Int64
}

// New wraps a Detector. A nil detector yields a Masker that fails closed on
// every event, so a misconfigured redact rule withholds content rather than
// leaking it.
func New(detector Detector) *Masker {
	return &Masker{detector: detector}
}

// Placeholder returns the stable masking token for a PII category. Masking
// preserves the category so a reader of a redacted conversation retains the
// shape of what was removed (an email vs a phone number) without the value.
func Placeholder(category string) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return "[REDACTED]"
	}
	return "[REDACTED:" + strings.ToUpper(category) + "]"
}

// MaskText detects PII in one text and returns it with every span replaced by
// its category placeholder. The returned bool reports whether a fail-closed
// replacement occurred (detection failed). On a detection failure the text is
// replaced wholesale with UnavailableSentinel — never returned in the clear.
func (m *Masker) MaskText(ctx context.Context, text string) (string, bool) {
	if strings.TrimSpace(text) == "" {
		return text, false
	}
	if m == nil || m.detector == nil {
		m.bumpFailClosed()
		return UnavailableSentinel, true
	}
	m.eventsScanned.Add(1)
	result, err := m.detector.Detect(ctx, text)
	if err != nil {
		m.bumpFailClosed()
		return UnavailableSentinel, true
	}
	masked, count := applySpans(text, result.Spans)
	if count > 0 {
		m.spansMasked.Add(int64(count))
	}
	return masked, false
}

func (m *Masker) bumpFailClosed() {
	if m == nil {
		return
	}
	m.failClosed.Add(1)
}

// Snapshot returns the current masking counters. Safe for concurrent use.
func (m *Masker) Snapshot() Counters {
	if m == nil {
		return Counters{}
	}
	return Counters{
		EventsScanned: m.eventsScanned.Load(),
		SpansMasked:   m.spansMasked.Load(),
		FailClosed:    m.failClosed.Load(),
	}
}

// MaskEvent scrubs PII from the indexed content fields of one run event in
// place-safe fashion (it returns a masked copy; the input event is never
// mutated). Only message, tool_call, and tool_result payloads carry searchable
// content, so other event types pass through untouched. The returned bool is
// true when any field on this event failed closed.
//
// The content surface here mirrors conversationsearch's sourceMessage: message
// Content, tool_call ToolName+Input, and tool_result Output/Error (+ the stored
// output blocks and raw JSON). Masking all of them keeps the search index — and
// any later re-projection of the stored event — free of the redacted PII.
func (m *Masker) MaskEvent(ctx context.Context, event *domain.RunEvent) (*domain.RunEvent, bool) {
	if event == nil || event.Data == nil {
		return event, false
	}
	switch data := event.Data.(type) {
	case *domain.MessageEventData:
		masked := *data
		var failed bool
		masked.Content, failed = m.MaskText(ctx, data.Content)
		return cloneEvent(event, &masked), failed
	case *domain.ToolCallEventData:
		masked := *data
		maskedInput, failed := m.maskInputMap(ctx, data.Input)
		masked.Input = maskedInput
		return cloneEvent(event, &masked), failed
	case *domain.ToolResultEventData:
		masked := *data
		failed := false
		mask := func(field string) string {
			out, f := m.MaskText(ctx, field)
			failed = failed || f
			return out
		}
		masked.Output = mask(data.Output)
		masked.Error = mask(data.Error)
		masked.RawOutputJSON = mask(data.RawOutputJSON)
		if len(data.OutputBlocks) > 0 {
			blocks := make([]domain.ToolOutputBlock, len(data.OutputBlocks))
			for i, block := range data.OutputBlocks {
				blocks[i] = block
				blocks[i].Text = mask(block.Text)
				blocks[i].RawJSON = mask(block.RawJSON)
			}
			masked.OutputBlocks = blocks
		}
		return cloneEvent(event, &masked), failed
	default:
		return event, false
	}
}

// maskInputMap scrubs PII from the string leaves of a tool-call input map.
// Working leaf-by-leaf keeps the JSON structure intact (so the tool-call shape
// is preserved for display) while still masking every value that reaches the
// index via the JSON-serialized content in conversationsearch.
func (m *Masker) maskInputMap(ctx context.Context, input map[string]any) (map[string]any, bool) {
	if len(input) == 0 {
		return input, false
	}
	out := make(map[string]any, len(input))
	failed := false
	for key, value := range input {
		masked, f := m.maskValue(ctx, value)
		if f {
			failed = true
		}
		out[key] = masked
	}
	return out, failed
}

func (m *Masker) maskValue(ctx context.Context, value any) (any, bool) {
	switch typed := value.(type) {
	case string:
		return m.MaskText(ctx, typed)
	case []any:
		out := make([]any, len(typed))
		failed := false
		for i, child := range typed {
			masked, f := m.maskValue(ctx, child)
			if f {
				failed = true
			}
			out[i] = masked
		}
		return out, failed
	case map[string]any:
		return m.maskInputMap(ctx, typed)
	default:
		return value, false
	}
}

// cloneEvent returns a shallow copy of event carrying the masked payload. The
// identity, type, and timestamp are preserved; only the content payload is
// swapped so the caller never mutates the parser-owned event.
func cloneEvent(event *domain.RunEvent, data domain.EventPayload) *domain.RunEvent {
	clone := *event
	clone.Data = data
	return &clone
}

// applySpans replaces each detected span in text with its category placeholder,
// returning the masked text and the number of spans applied. Overlapping or
// out-of-range spans are resolved deterministically: spans are applied
// right-to-left so earlier byte offsets stay valid, and a span fully covered by
// a later-applied one is skipped. Offsets are treated as byte offsets to match
// the pii-detector contract (Go's []byte(text) indexing).
func applySpans(text string, spans []Span) (string, int) {
	if len(spans) == 0 {
		return text, 0
	}
	bytes := []byte(text)
	n := len(bytes)
	valid := make([]Span, 0, len(spans))
	for _, span := range spans {
		if span.Start < 0 || span.End > n || span.Start >= span.End {
			continue
		}
		valid = append(valid, span)
	}
	if len(valid) == 0 {
		return text, 0
	}
	// Apply from the highest start offset downward so replacements do not shift
	// the offsets of spans not yet applied. Ties break on the wider span first.
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Start == valid[j].Start {
			return valid[i].End > valid[j].End
		}
		return valid[i].Start > valid[j].Start
	})
	masked := bytes
	applied := 0
	coveredFrom := n + 1 // lowest start offset already masked
	for _, span := range valid {
		if span.End > coveredFrom {
			// This span overlaps a region already replaced to its right; skip it
			// so placeholders never nest.
			continue
		}
		placeholder := []byte(Placeholder(span.Category))
		next := make([]byte, 0, len(masked)-(span.End-span.Start)+len(placeholder))
		next = append(next, masked[:span.Start]...)
		next = append(next, placeholder...)
		next = append(next, masked[span.End:]...)
		masked = next
		applied++
		coveredFrom = span.Start
	}
	return string(masked), applied
}

// DecodeDetectResult parses the pii-detector detect JSON returned as the
// ai-gateway ExecuteRoute OutputText. An empty or whitespace payload is a clean
// result, not an error, so a detector that found nothing is distinguishable
// from a transport failure (which the GatewayMasker reports as an error).
func DecodeDetectResult(outputText string) (DetectResult, error) {
	trimmed := strings.TrimSpace(outputText)
	if trimmed == "" {
		return DetectResult{Spans: []Span{}}, nil
	}
	var result DetectResult
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return DetectResult{}, fmt.Errorf("decode pii-detector result: %w", err)
	}
	return result, nil
}
