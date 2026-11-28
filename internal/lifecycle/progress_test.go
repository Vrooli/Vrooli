package lifecycle

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// captureSink records every published event for assertion.
type captureSink struct{ events []ProgressEvent }

func (s *captureSink) Publish(ev ProgressEvent) { s.events = append(s.events, ev) }

func TestProgressRendererEmitsAtQuiet(t *testing.T) {
	var out bytes.Buffer
	runner := &Runner{Out: &out, Verbosity: VerbosityQuiet}
	runner.publish(ProgressEvent{Kind: EventOperationStarted, Scenario: "alpha"})
	if got := out.String(); got != "starting alpha...\n" {
		t.Fatalf("quiet render = %q", got)
	}
}

func TestProgressRendererEmitsAtNormal(t *testing.T) {
	// Normal mode is the default on TTYs (where slog info is suppressed by
	// the top-level vrooli binary), so progress lines must reach the user
	// here or they see a silent 10+ second gap.
	var out bytes.Buffer
	runner := &Runner{Out: &out, Verbosity: VerbosityNormal}
	runner.publish(ProgressEvent{Kind: EventPhaseStarted, Scenario: "alpha", Phase: "setup"})
	if !strings.Contains(out.String(), "running setup phase for alpha...") {
		t.Fatalf("normal render dropped: %q", out.String())
	}
}

func TestProgressRendererSuppressedAtVerbose(t *testing.T) {
	// Verbose replays the full slog/debug stream plus child tool stdout,
	// so duplicating progress lines here adds noise.
	var out bytes.Buffer
	runner := &Runner{Out: &out, Verbosity: VerbosityVerbose}
	runner.publish(ProgressEvent{Kind: EventOperationStarted, Scenario: "alpha"})
	if got := out.String(); got != "" {
		t.Fatalf("verbose render should be silent, got %q", got)
	}
}

func TestProgressRendererNoOpOnNilWriter(t *testing.T) {
	runner := &Runner{Out: nil, Verbosity: VerbosityNormal}
	// Must not panic.
	runner.publish(ProgressEvent{Kind: EventOperationStarted, Scenario: "world"})
}

// TestRenderProgressLineTable pins every event → line mapping, including the
// silent kinds, so a renderer change is always a visible diff here.
func TestRenderProgressLineTable(t *testing.T) {
	cases := []struct {
		name string
		ev   ProgressEvent
		want string
	}{
		{"operation started", ProgressEvent{Kind: EventOperationStarted, Scenario: "a"}, "starting a...\n"},
		{"already running", ProgressEvent{Kind: EventOperationCompleted, Scenario: "a", AlreadyRunning: true}, "a is already running\n"},
		{"completed fresh start is silent", ProgressEvent{Kind: EventOperationCompleted, Scenario: "a", Verdict: "healthy"}, ""},
		{"failed is silent", ProgressEvent{Kind: EventOperationFailed, Scenario: "a", Reason: "boom"}, ""},
		{"stop", ProgressEvent{Kind: EventStopStarted, Scenario: "a"}, "stopping a...\n"},
		{"setup phase", ProgressEvent{Kind: EventPhaseStarted, Scenario: "a", Phase: "setup"}, "running setup phase for a...\n"},
		{"develop phase", ProgressEvent{Kind: EventPhaseStarted, Scenario: "a", Phase: "develop"}, "running develop phase for a...\n"},
		{"phase completed is silent", ProgressEvent{Kind: EventPhaseCompleted, Scenario: "a", Phase: "setup"}, ""},
		{"health wait", ProgressEvent{Kind: EventHealthWaiting, Scenario: "a"}, "waiting for a to become healthy...\n"},
		{"dependency starting", ProgressEvent{Kind: EventDependencyStarting, Scenario: "a", Dependency: "b", Reason: "not running"}, "a: starting dependency b (not running)\n"},
		{"dependency reused", ProgressEvent{Kind: EventDependencyReused, Scenario: "a", Dependency: "b"}, "a: dependency b already running; reusing existing process\n"},
		{"dependency reused after lock wait", ProgressEvent{Kind: EventDependencyReused, Scenario: "a", Dependency: "b", AfterLockWait: true}, "a: dependency b became ready while another invocation held its lifecycle lock; reusing existing process\n"},
		{"stale reuse_running", ProgressEvent{Kind: EventDependencyStalePolicy, Dependency: "b", Policy: "reuse_running", Reason: "binary changed"}, "b: stale but reused per freshness_policy=reuse_running (binary changed)\n"},
		{"stale rebuild_only", ProgressEvent{Kind: EventDependencyStalePolicy, Dependency: "b", Policy: "rebuild_only", Reason: "binary changed"}, "b: rebuilding stale dependency without restart per freshness_policy=rebuild_only (binary changed)\n"},
		{"resource starting", ProgressEvent{Kind: EventResourceStarting, Scenario: "a", Dependency: "postgres", Reason: "not running"}, "a: starting resource dependency postgres (not running)\n"},
		{"resource reused", ProgressEvent{Kind: EventResourceReused, Scenario: "a", Dependency: "postgres"}, "a: resource dependency postgres already running; reusing existing service\n"},
		{"resource ensure", ProgressEvent{Kind: EventResourceEnsureConfig, Scenario: "a", Dependency: "postgres"}, "a: ensuring postgres dependency config\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderProgressLine(tc.ev); got != tc.want {
				t.Fatalf("renderProgressLine = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWithProgressSinkFansOutAndStampsClock(t *testing.T) {
	at := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	var out bytes.Buffer
	runner := &Runner{Out: &out, Verbosity: VerbosityQuiet, deps: lifecycleDeps{
		now:   func() time.Time { return at },
		sleep: func(time.Duration) {},
	}}
	capture := &captureSink{}
	runner.WithProgressSink(capture)

	runner.publish(ProgressEvent{Kind: EventOperationStarted, Scenario: "alpha"})
	if out.String() != "starting alpha...\n" {
		t.Fatalf("renderer output = %q", out.String())
	}
	if len(capture.events) != 1 {
		t.Fatalf("capture events = %d, want 1", len(capture.events))
	}
	if !capture.events[0].At.Equal(at) {
		t.Fatalf("event At = %v, want injected clock %v", capture.events[0].At, at)
	}
}

type concurrentProgressSink struct{ count atomic.Int64 }

func (s *concurrentProgressSink) Publish(ProgressEvent) { s.count.Add(1) }

func TestProgressPublishAndSinkChurnIsRaceSafe(t *testing.T) {
	runner := &Runner{Verbosity: VerbosityVerbose, deps: lifecycleDeps{
		now:   time.Now,
		sleep: func(time.Duration) {},
	}}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				runner.publish(ProgressEvent{Kind: EventOperationStarted, Scenario: "alpha"})
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 250; j++ {
				sink := &concurrentProgressSink{}
				detach := runner.attachSink(sink)
				detach()
			}
		}()
	}
	wg.Wait()
}
