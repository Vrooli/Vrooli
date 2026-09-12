package session

import (
	"testing"
	"time"

	"web-console/internal/backend"
)

// fakeActivityClock runs timers only when the test advances time.
type fakeActivityClock struct {
	now    time.Time
	timers []*fakeActivityTimer
}

type fakeActivityTimer struct {
	at      time.Time
	f       func()
	stopped bool
}

func newFakeActivityClock() *fakeActivityClock {
	return &fakeActivityClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeActivityClock) Now() time.Time { return c.now }

func (c *fakeActivityClock) AfterFunc(d time.Duration, f func()) func() bool {
	t := &fakeActivityTimer{at: c.now.Add(d), f: f}
	c.timers = append(c.timers, t)
	return func() bool {
		was := !t.stopped
		t.stopped = true
		return was
	}
}

func (c *fakeActivityClock) Advance(d time.Duration) {
	target := c.now.Add(d)
	for {
		var next *fakeActivityTimer
		for _, t := range c.timers {
			if !t.stopped && !t.at.After(target) && (next == nil || t.at.Before(next.at)) {
				next = t
			}
		}
		if next == nil {
			break
		}
		c.now = next.at
		next.stopped = true
		next.f()
	}
	c.now = target
}

// scriptedDetector returns whatever analysis the test sets.
type scriptedDetector struct{ analysis backend.PromptAnalysis }

func (s *scriptedDetector) Analyze(backend.ScreenView) backend.PromptAnalysis { return s.analysis }

type fakeScreen struct{}

func (fakeScreen) Cols() int         { return 80 }
func (fakeScreen) Rows() int         { return 24 }
func (fakeScreen) CursorRow() int    { return 0 }
func (fakeScreen) CursorCol() int    { return 0 }
func (fakeScreen) PlainText() string { return "" }

type activityHarness struct {
	clock     *fakeActivityClock
	detector  *scriptedDetector
	d         *ActivityDetector
	published []Activity
	reads     int
}

func newActivityHarness(t *testing.T) *activityHarness {
	t.Helper()
	h := &activityHarness{clock: newFakeActivityClock(), detector: &scriptedDetector{}}
	h.d = NewActivityDetector(ActivityConfig{
		SessionID: "s1",
		Screen:    func() backend.ScreenView { h.reads++; return fakeScreen{} },
		Harness:   func() (string, backend.PromptDetector) { return "claude", h.detector },
		Publish:   func(a Activity) { h.published = append(h.published, a) },
		Clock:     h.clock,
	})
	return h
}

func (h *activityHarness) states() []ActivityState {
	out := make([]ActivityState, 0, len(h.published))
	for _, a := range h.published {
		out = append(out, a.State)
	}
	return out
}

func assertStates(t *testing.T, h *activityHarness, want ...ActivityState) {
	t.Helper()
	got := h.states()
	if len(got) != len(want) {
		t.Fatalf("published states = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("published states = %v, want %v", got, want)
		}
	}
}

func TestActivityStartsUnknownAndOutputMeansWorking(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	if got := h.d.Current().State; got != ActivityUnknown {
		t.Fatalf("initial state = %s, want unknown", got)
	}
	h.d.OnFrame()
	assertStates(t, h, ActivityWorking)
	if h.published[0].LastOutputAt.IsZero() || h.published[0].Since.IsZero() {
		t.Fatalf("working activity must carry since and last output: %+v", h.published[0])
	}
}

func TestActivityQuietAtPromptIsIdle(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.d.OnFrame()
	h.detector.analysis = backend.PromptAnalysis{AwaitingInput: true, Confidence: 0.75}
	h.clock.Advance(1400 * time.Millisecond)
	if got := h.d.Current().State; got != ActivityWorking {
		t.Fatalf("before the quiet window state = %s, want working", got)
	}
	h.clock.Advance(200 * time.Millisecond)
	assertStates(t, h, ActivityWorking, ActivityIdle)
	if h.published[1].Source != SourceScreen {
		t.Fatalf("idle source = %s, want screen", h.published[1].Source)
	}
}

func TestActivityPromptBoxMeansWaitingWithContent(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.d.OnFrame()
	prompt := &backend.PendingPrompt{Kind: "permission", Text: "Do you want to proceed?", Options: []backend.PromptOption{{Key: "1", Label: "Yes", Selected: true}, {Key: "2", Label: "No"}}}
	h.detector.analysis = backend.PromptAnalysis{Prompt: prompt, Confidence: 0.85}
	h.clock.Advance(activityDebounce)
	assertStates(t, h, ActivityWorking, ActivityWaiting)
	if got := h.published[1].Prompt; got == nil || got.Text != "Do you want to proceed?" || len(got.Options) != 2 {
		t.Fatalf("waiting prompt = %+v", got)
	}
	// Output that does not change the prompt does not republish.
	h.d.OnFrame()
	h.clock.Advance(2 * time.Second)
	assertStates(t, h, ActivityWorking, ActivityWaiting)
}

func TestActivityQuietWithoutEvidenceIsUnknown(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.d.OnFrame()
	h.clock.Advance(2 * time.Second)
	assertStates(t, h, ActivityWorking, ActivityUnknown)
}

func TestActivityBelowConfidenceFloorReadsUnknown(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.d.OnFrame()
	h.detector.analysis = backend.PromptAnalysis{AwaitingInput: true, Confidence: 0.5}
	h.clock.Advance(2 * time.Second)
	if got := h.d.Current().State; got != ActivityUnknown {
		t.Fatalf("low-confidence idle reads %s, want unknown", got)
	}
}

func TestActivityHookWaitingHoldsUntilOutputResumes(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.d.OnFrame()
	h.clock.Advance(2 * time.Second) // unknown
	h.d.OnHook(HookWaiting)
	if got := h.d.Current(); got.State != ActivityWaiting || got.Source != SourceHook || got.Confidence < 0.9 {
		t.Fatalf("after the Notification hook = %+v, want waiting from hook", got)
	}
	// No output: the hook keeps holding even though the screen shows nothing.
	h.clock.Advance(5 * time.Second)
	if got := h.d.Current().State; got != ActivityWaiting {
		t.Fatalf("without output the hook's waiting must hold, got %s", got)
	}
	// The user answers; the agent works again.
	h.d.OnFrame()
	h.clock.Advance(activityDebounce)
	if got := h.d.Current().State; got != ActivityWorking {
		t.Fatalf("after output resumes = %s, want working", got)
	}
}

func TestActivityHarnessPromptRanksAboveScreen(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.d.OnHarnessPrompt(backend.PendingPrompt{Kind: "permission", Text: "Run tests?"})
	h.d.OnFrame()
	h.detector.analysis = backend.PromptAnalysis{Working: true, Confidence: 0.85}
	h.clock.Advance(2 * time.Second)
	if got := h.d.Current(); got.State != ActivityWaiting || got.Source != SourceHarnessEvent || got.Confidence != confidenceHarnessEvent {
		t.Fatalf("harness prompt must hold over the screen: %+v", got)
	}
	h.d.OnHarnessPromptResolved()
	if got := h.d.Current().State; got != ActivityWorking {
		t.Fatalf("after the reply = %s, want working", got)
	}
}

func TestActivityEchoOfUserInputIsNotWork(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.d.OnFrame()
	h.detector.analysis = backend.PromptAnalysis{AwaitingInput: true, Confidence: 0.75}
	h.clock.Advance(2 * time.Second) // idle
	h.d.OnInput()
	h.clock.Advance(20 * time.Millisecond)
	h.d.OnFrame() // the echo of the typed character
	h.clock.Advance(activityDebounce)
	assertStates(t, h, ActivityWorking, ActivityIdle)
}

func TestActivityHeartbeatWhileWorking(t *testing.T) {
	// [REQ:P0-017e]
	h := newActivityHarness(t)
	h.detector.analysis = backend.PromptAnalysis{Working: true, Confidence: 0.85}
	for i := 0; i < 16; i++ {
		h.d.OnFrame()
		h.clock.Advance(time.Second)
	}
	if len(h.published) != 2 {
		t.Fatalf("published %d envelopes over 16 s of work, want 2 (first + one 15 s heartbeat): %v", len(h.published), h.states())
	}
}

func TestActivityScreenReadsAreDebounced(t *testing.T) {
	// [REQ:P0-017e] 1,000 frames over one second read the screen a bounded number of times.
	h := newActivityHarness(t)
	for i := 0; i < 1000; i++ {
		h.d.OnFrame()
		h.clock.Advance(time.Millisecond)
	}
	h.clock.Advance(2 * time.Second)
	if h.reads > 10 {
		t.Fatalf("screen read %d times for a 1,000-frame burst, want at most 10", h.reads)
	}
}

func TestActivityScreenIsReadDuringContinuousOutput(t *testing.T) {
	// [REQ:P0-017e] A working agent redraws its spinner faster than the
	// debounce window. The screen must still be read while output continues,
	// so the in-progress marker (not the output clock) names the state. Failing
	// value before the fix, observed live on 2026-09-11: a codex pane with
	// "esc to interrupt" on screen stayed source=output_clock with no harness,
	// because every frame reset the trailing debounce.
	h := newActivityHarness(t)
	h.detector.analysis = backend.PromptAnalysis{Working: true, Confidence: 0.85}
	for i := 0; i < 50; i++ {
		h.d.OnFrame()
		h.clock.Advance(20 * time.Millisecond)
	}
	if h.reads == 0 {
		t.Fatal("screen never read during one second of continuous output")
	}
	current := h.d.Current()
	if current.Source != SourceScreen || current.Harness != "claude" {
		t.Fatalf("activity during continuous output = %+v, want the screen's working state for the harness", current)
	}
}

func TestActivityClosedDetectorIgnoresEvents(t *testing.T) {
	h := newActivityHarness(t)
	h.d.Close()
	h.d.OnFrame()
	h.d.OnHook(HookWaiting)
	h.clock.Advance(time.Minute)
	if len(h.published) != 0 {
		t.Fatalf("closed detector published %v", h.states())
	}
}
