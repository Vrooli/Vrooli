package runner_test

import (
	"strings"
	"sync"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// captureSink records emitted events thread-safely; used by output_helpers
// + sanitize tests in this package.
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

// TestEmitStderrAsWarnOnSuccess_NonEmpty verifies that successful runs
// with non-empty captured stderr surface it as a warn-level log event.
// Regression for the dropped bwrap-warning that hid the swarm-manager
// initiative-feedback chdir failure.
func TestEmitStderrAsWarnOnSuccess_NonEmpty(t *testing.T) {
	sink := &captureSink{}
	runner.EmitStderrAsWarnOnSuccess(uuid.New(), sink, "bwrap: warning: something\n")

	events := sink.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.EventType != domain.EventTypeLog {
		t.Errorf("EventType=%v want log", ev.EventType)
	}
	logData, ok := ev.Data.(*domain.LogEventData)
	if !ok {
		t.Fatalf("Data type=%T want *LogEventData", ev.Data)
	}
	if logData.Level != "warn" {
		t.Errorf("level=%q want warn", logData.Level)
	}
	if !strings.Contains(logData.Message, "bwrap: warning") {
		t.Errorf("message missing stderr content: %q", logData.Message)
	}
}

func TestEmitStderrAsWarnOnSuccess_Empty(t *testing.T) {
	sink := &captureSink{}
	runner.EmitStderrAsWarnOnSuccess(uuid.New(), sink, "")
	runner.EmitStderrAsWarnOnSuccess(uuid.New(), sink, "   \n  ")
	if got := len(sink.snapshot()); got != 0 {
		t.Errorf("expected 0 events, got %d", got)
	}
}

func TestEmitStderrAsWarnOnSuccess_NilSink(t *testing.T) {
	runner.EmitStderrAsWarnOnSuccess(uuid.New(), nil, "anything") // must not panic
}

func TestTruncateForLog_BoundsPayload(t *testing.T) {
	long := strings.Repeat("x", 5000)
	got := runner.TruncateForLog(long, 4096)
	if len(got) <= 4096 {
		t.Errorf("got len %d, want >4096 (suffix marker)", len(got))
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Errorf("missing truncation marker: %q", got[len(got)-30:])
	}
	short := "hello"
	if got := runner.TruncateForLog(short, 4096); got != short {
		t.Errorf("short string was modified: %q", got)
	}
}
