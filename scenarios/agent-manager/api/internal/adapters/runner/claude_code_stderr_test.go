package runner

import (
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// TestEmitStderrAsWarnOnSuccess_NonEmpty verifies that a successful run
// with non-empty captured stderr surfaces it as a warn-level log event.
// Before this fix, launch-time diagnostics (like the bwrap chdir error
// that broke swarm-manager initiative-feedback runs) were silently
// dropped on the success path.
func TestEmitStderrAsWarnOnSuccess_NonEmpty(t *testing.T) {
	sink := &captureSink{}
	runID := uuid.New()

	emitStderrAsWarnOnSuccess(runID, sink, "bwrap: warning: something\n")

	events := sink.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.EventType != domain.EventTypeLog {
		t.Errorf("EventType = %v; want log", ev.EventType)
	}
	logData, ok := ev.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("Data = %T; want *domain.LogEventData", ev.Data)
	}
	if logData.Level != "warn" {
		t.Errorf("level = %q; want warn", logData.Level)
	}
	if !strings.Contains(logData.Message, "bwrap: warning") {
		t.Errorf("message missing stderr content: %q", logData.Message)
	}
}

// TestEmitStderrAsWarnOnSuccess_Empty ensures we don't spam an empty log
// event when stderr was clean.
func TestEmitStderrAsWarnOnSuccess_Empty(t *testing.T) {
	sink := &captureSink{}
	emitStderrAsWarnOnSuccess(uuid.New(), sink, "")
	emitStderrAsWarnOnSuccess(uuid.New(), sink, "   \n  ")
	if got := len(sink.snapshot()); got != 0 {
		t.Errorf("expected 0 events for empty stderr, got %d", got)
	}
}

// TestEmitStderrAsWarnOnSuccess_NilSink is a smoke check for the no-sink
// path; it must not panic.
func TestEmitStderrAsWarnOnSuccess_NilSink(t *testing.T) {
	emitStderrAsWarnOnSuccess(uuid.New(), nil, "anything")
}

// TestTruncateForLog_BoundsPayload pins the truncation contract that
// keeps run-event payloads from ballooning when stderr is enormous.
func TestTruncateForLog_BoundsPayload(t *testing.T) {
	long := strings.Repeat("x", 5000)
	got := truncateForLog(long, 4096)
	if len(got) <= 4096 {
		t.Errorf("got len %d; want >4096 (suffix marker)", len(got))
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Errorf("missing truncation marker: %q", got[len(got)-30:])
	}

	short := "hello"
	if got := truncateForLog(short, 4096); got != short {
		t.Errorf("short string was modified: %q", got)
	}
}
