package runner

import (
	"agent-manager/internal/domain"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Diagnostic stream samples — added for execution_error enrichment (2026-04-20)
// =============================================================================

// diagnosticSamples hold result events that the prior parser swallowed into
// empty error events. Each fixture pins a real-world shape we need to
// classify from telemetry alone.
var diagnosticSamples = map[string]string{
	"result_error_max_turns": `{"type":"result","subtype":"error_max_turns","is_error":true,"duration_ms":450000,"num_turns":60,"session_id":"s1","result":"","total_cost_usd":0}`,
	"result_error_empty":     `{"type":"result","subtype":"","is_error":true,"duration_ms":100,"num_turns":0,"session_id":"s2","result":"","total_cost_usd":0}`,
	"result_error_during":    `{"type":"result","subtype":"error_during_execution","is_error":true,"duration_ms":5000,"num_turns":12,"session_id":"s3","result":"internal API error","total_cost_usd":0}`,
	"system_auto_compact":    `{"type":"system","subtype":"auto-compacting","result":"Auto-compacting conversation to free tokens","session_id":"s4"}`,
}

// =============================================================================
// parseResultEvent — enriched error details
// =============================================================================

func TestClaudeCodeRunner_ParseResultEvent_ErrorMaxTurns(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	events, err := runner.parseStreamEvents(runID, diagnosticSamples["result_error_max_turns"])
	if err != nil {
		t.Fatalf("parseStreamEvents returned error: %v", err)
	}
	errEvent := findErrorEvent(events)
	if errEvent == nil {
		t.Fatal("expected an error event")
	}
	errData, ok := errEvent.Data.(*domain.ErrorEventData)
	if !ok {
		t.Fatalf("expected ErrorEventData, got %T", errEvent.Data)
	}
	if errData.Code != "execution_error" {
		t.Errorf("expected code=execution_error, got %s", errData.Code)
	}
	if !strings.HasPrefix(errData.Message, "claude-code terminated with is_error=true (subtype=error_max_turns, turns=60, duration_ms=450000)") {
		t.Errorf("message missing expected prefix, got %q", errData.Message)
	}
	if errData.Details == nil {
		t.Fatal("expected Details to be populated")
	}
	if got := errData.Details["subtype"]; got != "error_max_turns" {
		t.Errorf("subtype: got %v, want error_max_turns", got)
	}
	if got := errData.Details["num_turns"]; got != 60 {
		t.Errorf("num_turns: got %v, want 60", got)
	}
	if got := errData.Details["duration_ms"]; got != 450000 {
		t.Errorf("duration_ms: got %v, want 450000", got)
	}
	if got := errData.Details["session_id"]; got != "s1" {
		t.Errorf("session_id: got %v, want s1", got)
	}
	if got := errData.Details["result_text"]; got != "" {
		t.Errorf("result_text: got %v, want empty", got)
	}
}

func TestClaudeCodeRunner_ParseResultEvent_ErrorEmpty(t *testing.T) {
	// Regression for run 66cbfd0a: an is_error=true with every interesting
	// field empty must still produce a non-empty, classifiable message.
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	events, err := runner.parseStreamEvents(runID, diagnosticSamples["result_error_empty"])
	if err != nil {
		t.Fatalf("parseStreamEvents returned error: %v", err)
	}
	errEvent := findErrorEvent(events)
	if errEvent == nil {
		t.Fatal("expected an error event")
	}
	errData := errEvent.Data.(*domain.ErrorEventData)
	if strings.TrimSpace(errData.Message) == "" {
		t.Fatal("message must not be empty even when result and subtype are empty")
	}
	if !strings.Contains(errData.Message, "turns=0") || !strings.Contains(errData.Message, "duration_ms=100") {
		t.Errorf("expected counters in message, got %q", errData.Message)
	}
	if errData.Details == nil || errData.Details["result_text"] != "" {
		t.Errorf("expected empty result_text in details, got %+v", errData.Details)
	}
}

func TestClaudeCodeRunner_ParseResultEvent_ErrorDuringExecution_CarriesResultText(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	events, err := runner.parseStreamEvents(runID, diagnosticSamples["result_error_during"])
	if err != nil {
		t.Fatalf("parseStreamEvents returned error: %v", err)
	}
	errEvent := findErrorEvent(events)
	if errEvent == nil {
		t.Fatal("expected an error event")
	}
	errData := errEvent.Data.(*domain.ErrorEventData)
	if !strings.Contains(errData.Message, "internal API error") {
		t.Errorf("expected message to include result_text, got %q", errData.Message)
	}
	if errData.Details["result_text"] != "internal API error" {
		t.Errorf("expected result_text carried in details, got %v", errData.Details["result_text"])
	}
	// Success message fallback must NOT be produced when there's a result_text
	// and a non-empty subtype; we still want the subtype visible.
	if !strings.Contains(errData.Message, "subtype=error_during_execution") {
		t.Errorf("expected subtype in message, got %q", errData.Message)
	}
}

// The rate-limit branch in parseResultEvent must short-circuit before the
// enrichment path — preserving the existing RateLimitEventData contract.
func TestClaudeCodeRunner_ParseResultEvent_RateLimitPrecedence(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	// The existing "result_error" sample in claude_code_stream_test.go is a
	// rate-limit-shaped payload. We reuse it here for isolation.
	events, err := runner.parseStreamEvents(runID, claudeCodeSamples["result_error"])
	if err != nil {
		t.Fatalf("parseStreamEvents returned error: %v", err)
	}
	for _, e := range events {
		if e == nil {
			continue
		}
		if _, ok := e.Data.(*domain.ErrorEventData); ok {
			t.Errorf("rate-limit payload must not produce an ErrorEventData, got %+v", e.Data)
		}
	}
	// And the rate-limit event must be present.
	hasRL := false
	for _, e := range events {
		if _, ok := e.Data.(*domain.RateLimitEventData); ok {
			hasRL = true
			break
		}
	}
	if !hasRL {
		t.Error("expected a RateLimitEventData in the output")
	}
}

// =============================================================================
// Diagnostic helpers — small, independently testable
// =============================================================================

func TestTailBytesUTF8Safe_TruncatesAtRuneBoundary(t *testing.T) {
	// Build a string whose last slice boundary falls mid-rune. 'é' is 2 bytes
	// in UTF-8. With max=3 on "aaé", the naive slice would cut mid-rune.
	input := "aaaaaaé"
	got := tailBytesUTF8Safe(input, 3)
	if !isValidUTF8(got) {
		t.Fatalf("result not valid UTF-8: %q", got)
	}
	if len(got) > 3 {
		// Rewinding to a rune boundary may shrink the window; must never exceed.
		t.Fatalf("got longer than max: %d > 3", len(got))
	}
}

func TestTailBytesUTF8Safe_ShortInputUnchanged(t *testing.T) {
	if got := tailBytesUTF8Safe("abc", 100); got != "abc" {
		t.Errorf("got %q, want abc", got)
	}
}

func TestRedactSecrets_RedactsBearerAndApiKey(t *testing.T) {
	in := "Authorization: Bearer abcdef123456 and api_key=supersecretvalue and sk-abcdefgh"
	out := redactSecrets(in)
	if strings.Contains(out, "abcdef123456") {
		t.Errorf("Bearer token was not redacted: %q", out)
	}
	if strings.Contains(out, "supersecretvalue") {
		t.Errorf("api_key value was not redacted: %q", out)
	}
	if strings.Contains(out, "sk-abcdefgh") {
		t.Errorf("sk- key was not redacted: %q", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Error("expected at least one <redacted> replacement")
	}
}

func TestIsAutoCompactMarker_TableDriven(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"regular log line", false},
		{"Auto-compacting conversation to free tokens", true},
		{"  CONVERSATION HISTORY HAS BEEN COMPACTED  ", true},
		{"context has been compacted", true},
		{"Automatic compaction begins now", true},
		{"auto-compact", false}, // not a full marker
	}
	for _, c := range cases {
		if got := isAutoCompactMarker(c.in); got != c.want {
			t.Errorf("isAutoCompactMarker(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseStreamEvents_AutoCompactSystemEmitsCompactionEvent(t *testing.T) {
	runner := &ClaudeCodeRunner{}
	runID := uuid.New()

	events, err := runner.parseStreamEvents(runID, diagnosticSamples["system_auto_compact"])
	if err != nil {
		t.Fatalf("parseStreamEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	got, ok := events[0].Data.(*domain.CompactionEventData)
	if !ok {
		t.Fatalf("expected CompactionEventData, got %T", events[0].Data)
	}
	if got.Trigger != "auto" {
		t.Errorf("expected trigger=auto, got %q", got.Trigger)
	}
}

// =============================================================================
// Heartbeat — idle-stream detection
// =============================================================================

// captureSink is a minimal EventSink that records every emitted event in a
// thread-safe slice for assertions.
type captureSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (c *captureSink) Emit(e *domain.RunEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
	return nil
}
func (c *captureSink) Close() error { return nil }

func (c *captureSink) snapshot() []*domain.RunEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*domain.RunEvent, len(c.events))
	copy(out, c.events)
	return out
}

func TestHeartbeat_EmitsIdleLogAfterThreshold(t *testing.T) {
	// Swap thresholds low for a fast, deterministic test.
	defer func(prev1, prev2 int64) {
		streamIdleHeartbeatMillis = prev1
		streamIdleHeartbeatTickMillis = prev2
	}(streamIdleHeartbeatMillis, streamIdleHeartbeatTickMillis)
	streamIdleHeartbeatMillis = 50
	streamIdleHeartbeatTickMillis = 10

	sink := &captureSink{}
	hb := newHeartbeat(uuid.New(), sink)
	hb.start()
	defer hb.stop()

	// Wait enough for at least one idle tick to fire.
	time.Sleep(200 * time.Millisecond)

	found := false
	for _, e := range sink.snapshot() {
		if ld, ok := e.Data.(*domain.LogEventData); ok {
			if strings.HasPrefix(ld.Message, "stream idle for") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected at least one 'stream idle for' log event")
	}
}

func TestHeartbeat_ResetsOnRecordEvent(t *testing.T) {
	defer func(prev1, prev2 int64) {
		streamIdleHeartbeatMillis = prev1
		streamIdleHeartbeatTickMillis = prev2
	}(streamIdleHeartbeatMillis, streamIdleHeartbeatTickMillis)
	streamIdleHeartbeatMillis = 80
	streamIdleHeartbeatTickMillis = 10

	sink := &captureSink{}
	hb := newHeartbeat(uuid.New(), sink)
	hb.start()
	defer hb.stop()

	// Record activity every 20ms for 150ms — well above threshold if summed,
	// but each recordEvent should reset the idle window so no log fires.
	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		hb.recordEvent("tool_call:Bash")
		time.Sleep(20 * time.Millisecond)
	}

	for _, e := range sink.snapshot() {
		if ld, ok := e.Data.(*domain.LogEventData); ok {
			if strings.HasPrefix(ld.Message, "stream idle for") {
				t.Fatalf("unexpected idle log while events were being recorded: %q", ld.Message)
			}
		}
	}
}

// =============================================================================
// Helpers
// =============================================================================

func findErrorEvent(events []*domain.RunEvent) *domain.RunEvent {
	for _, e := range events {
		if e == nil {
			continue
		}
		if _, ok := e.Data.(*domain.ErrorEventData); ok {
			return e
		}
	}
	return nil
}

func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		r, size := decodeRune(s[i:])
		if r == '\uFFFD' && size == 1 {
			return false
		}
		i += size
	}
	return true
}

// Tiny wrapper so we don't import unicode/utf8 for a single use in tests.
func decodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return '\uFFFD', 0
	}
	return []rune(s)[0], len(string([]rune(s)[0]))
}
