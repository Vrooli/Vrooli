package smoketest

import (
	"strings"
	"testing"
	"time"
)

func fixtureTrace(kind LaunchRunKind) LaunchTrace {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	names := requiredEvents(kind)
	events := make([]LaunchEvent, 0, len(names))
	for i, name := range names {
		events = append(events, LaunchEvent{Name: name, Component: "fixture", Role: "main", MonotonicNs: int64(i) * 10_000_000, WallTime: start.Add(time.Duration(i) * 10 * time.Millisecond)})
	}
	return LaunchTrace{SchemaVersion: LaunchTraceSchemaVersion, RunID: "run-1-" + string(kind), RunKind: kind, StartedAt: start, CompletedAt: start.Add(time.Second), Events: events}
}

func TestLaunchTraceValidatesProtocolAndDemoSeparately(t *testing.T) {
	protocol := fixtureTrace(LaunchRunProtocol)
	demo := fixtureTrace(LaunchRunDemo)
	if err := protocol.Validate(); err != nil {
		t.Fatalf("protocol trace invalid: %v", err)
	}
	if err := demo.Validate(); err != nil {
		t.Fatalf("demo trace invalid: %v", err)
	}
	if protocol.RunID == demo.RunID {
		t.Fatal("protocol and demo traces must have distinct IDs")
	}
}

func TestLaunchTraceRejectsReorderedAndCredentialData(t *testing.T) {
	trace := fixtureTrace(LaunchRunDemo)
	trace.Events[2].MonotonicNs = 1
	if err := trace.Validate(); err == nil || !strings.Contains(err.Error(), "monotonic") {
		t.Fatalf("expected ordering error, got %v", err)
	}
	trace = fixtureTrace(LaunchRunDemo)
	trace.Events[0].Details = map[string]string{"authorization": "Bearer abc"}
	if err := trace.Validate(); err == nil || !strings.Contains(err.Error(), "credential") {
		t.Fatalf("expected redaction error, got %v", err)
	}
}

func TestLaunchTraceSegmentUsesMonotonicTime(t *testing.T) {
	trace := fixtureTrace(LaunchRunDemo)
	duration, err := trace.Segment(EventSplashCreated, EventSplashFirstPaint)
	if err != nil {
		t.Fatal(err)
	}
	if duration != 20*time.Millisecond {
		t.Fatalf("duration = %v, want 20ms", duration)
	}
}
